// Package mediaretention deletes media files GoWA wrote to disk once they are
// older than a configured number of days (WHATSAPP_MEDIA_RETENTION_DAYS).
//
// Only the media directories are swept: statics/media (including the
// per-chat/per-date subdirectories created by the download-media API),
// statics/senditems, and the legacy inbound-image copies older releases wrote
// directly into storages/. The databases, history dumps and every other file
// in storages/ are never touched.
package mediaretention

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/aldinokemal/go-whatsapp-web-multidevice/config"
	"github.com/sirupsen/logrus"
)

// SweepInterval is how often the scheduler re-runs the sweep after the
// startup pass.
const SweepInterval = time.Hour

// maxRetentionDays caps the configured value so the cutoff date cannot
// overflow into the future (which would make every file look expired).
const maxRetentionDays = 100 * 365

// legacyStoragesMediaPattern matches the "<unix>-<uuid><ext>" names
// utils.ExtractMedia generates. storages/ also holds the WhatsApp and chat
// storage databases (plus -wal/-shm sidecars) and history-*.json dumps, so
// only files with this exact shape are eligible there.
var legacyStoragesMediaPattern = regexp.MustCompile(
	`^[0-9]+-[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}(\.[^/\\]*)?$`,
)

// Target is one directory the sweep cleans.
type Target struct {
	Dir string
	// Recursive descends into subdirectories and removes subdirectories that
	// were already idle before the cutoff and are empty after the sweep. The
	// root directory itself is never removed.
	Recursive bool
	// Match restricts which file names are eligible; nil means any file.
	// Dotfiles (e.g. the tracked .gitignore placeholders) are always kept.
	Match func(name string) bool
}

// Result summarizes one sweep.
type Result struct {
	Files  int
	Dirs   int
	Bytes  int64
	Errors int
}

// IsLegacyStoragesMedia reports whether name is a media copy written into
// storages/ by ExtractMedia (as opposed to a database or history dump).
func IsLegacyStoragesMedia(name string) bool {
	return legacyStoragesMediaPattern.MatchString(name)
}

// DefaultTargets returns the media directories GoWA writes to.
func DefaultTargets() []Target {
	return []Target{
		{Dir: config.PathMedia, Recursive: true},
		{Dir: config.PathSendItems, Recursive: true},
		{Dir: config.PathStorages, Recursive: false, Match: IsLegacyStoragesMedia},
	}
}

// Cutoff returns the modification time before which media is deleted.
func Cutoff(now time.Time, retentionDays int) time.Time {
	retentionDays = min(retentionDays, maxRetentionDays)
	return now.AddDate(0, 0, -retentionDays)
}

// Sweep deletes regular files under targets last modified before cutoff.
// Missing directories are skipped; per-file errors are logged and counted
// without aborting the sweep.
func Sweep(targets []Target, cutoff time.Time) Result {
	var result Result
	for _, target := range targets {
		sweepTarget(target, cutoff, &result)
	}
	return result
}

func sweepTarget(target Target, cutoff time.Time, result *Result) {
	if target.Dir == "" {
		return
	}
	root := filepath.Clean(target.Dir)
	// The directory itself may be a symlink (e.g. statics/media moved to a
	// bigger disk); resolve it so WalkDir walks the target. Symlinks below
	// the root are still never followed.
	if resolved, err := filepath.EvalSymlinks(root); err == nil {
		root = resolved
	} else if !errors.Is(err, fs.ErrNotExist) {
		logrus.Warnf("[MEDIA_RETENTION] cannot resolve %s: %v", root, err)
		result.Errors++
		return
	}
	// Subdirectories that were idle before the cutoff, in walk (pre-)order,
	// captured before their files are deleted (deleting bumps their mtime).
	var idleDirs []string

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return nil
			}
			logrus.Warnf("[MEDIA_RETENTION] cannot read %s: %v", path, err)
			result.Errors++
			if d != nil && d.IsDir() && path != root {
				return fs.SkipDir
			}
			return nil
		}

		if d.IsDir() {
			if path == root {
				return nil
			}
			if !target.Recursive {
				return fs.SkipDir
			}
			if info, infoErr := d.Info(); infoErr == nil && info.ModTime().Before(cutoff) {
				idleDirs = append(idleDirs, path)
			}
			return nil
		}

		// WalkDir does not follow symlinks; only plain files are eligible.
		if !d.Type().IsRegular() {
			return nil
		}
		name := d.Name()
		if strings.HasPrefix(name, ".") {
			return nil
		}
		if target.Match != nil && !target.Match(name) {
			return nil
		}

		info, infoErr := d.Info()
		if infoErr != nil {
			if !errors.Is(infoErr, fs.ErrNotExist) {
				logrus.Warnf("[MEDIA_RETENTION] cannot stat %s: %v", path, infoErr)
				result.Errors++
			}
			return nil
		}
		if !info.ModTime().Before(cutoff) {
			return nil
		}

		if removeErr := os.Remove(path); removeErr != nil {
			if !errors.Is(removeErr, fs.ErrNotExist) {
				logrus.Warnf("[MEDIA_RETENTION] cannot delete %s: %v", path, removeErr)
				result.Errors++
			}
			return nil
		}
		result.Files++
		result.Bytes += info.Size()
		return nil
	})
	if err != nil {
		logrus.Warnf("[MEDIA_RETENTION] sweep of %s stopped: %v", root, err)
		result.Errors++
	}

	// Deepest first, so a parent emptied by removing its children goes too.
	// os.Remove refuses non-empty directories, so this never deletes content.
	for i := len(idleDirs) - 1; i >= 0; i-- {
		if os.Remove(idleDirs[i]) == nil {
			result.Dirs++
		}
	}
}

// Start runs a sweep immediately and then every SweepInterval until ctx is
// cancelled. retentionDays <= 0 disables it.
func Start(ctx context.Context, retentionDays int, targets []Target) {
	if retentionDays <= 0 {
		return
	}
	go run(ctx, retentionDays, targets, SweepInterval)
}

func run(ctx context.Context, retentionDays int, targets []Target, interval time.Duration) {
	sweepAndLog(retentionDays, targets)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			sweepAndLog(retentionDays, targets)
		case <-ctx.Done():
			return
		}
	}
}

func sweepAndLog(retentionDays int, targets []Target) {
	result := Sweep(targets, Cutoff(time.Now(), retentionDays))
	if result.Files == 0 && result.Dirs == 0 && result.Errors == 0 {
		logrus.Debugf("[MEDIA_RETENTION] nothing older than %d days", retentionDays)
		return
	}
	logrus.Infof("[MEDIA_RETENTION] deleted %d files (%d bytes) and %d empty directories older than %d days; %d errors",
		result.Files, result.Bytes, result.Dirs, retentionDays, result.Errors)
}
