package mediaretention

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/aldinokemal/go-whatsapp-web-multidevice/config"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	sweepNow   = time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC)
	oldTime    = sweepNow.AddDate(0, 0, -40)
	freshTime  = sweepNow.AddDate(0, 0, -1)
	sweepAfter = Cutoff(sweepNow, 30)
)

// extractMediaName mirrors the file name utils.ExtractMedia generates.
func extractMediaName(ext string) string {
	return fmt.Sprintf("%d-%s%s", oldTime.Unix(), uuid.NewString(), ext)
}

func writeFile(t *testing.T, path string, modTime time.Time) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte("data"), 0o600))
	require.NoError(t, os.Chtimes(path, modTime, modTime))
}

func setDirTime(t *testing.T, path string, modTime time.Time) {
	t.Helper()
	require.NoError(t, os.Chtimes(path, modTime, modTime))
}

func useTempPaths(t *testing.T) (media, sendItems, storages string) {
	t.Helper()
	base := t.TempDir()
	media = filepath.Join(base, "statics", "media")
	sendItems = filepath.Join(base, "statics", "senditems")
	storages = filepath.Join(base, "storages")
	for _, dir := range []string{media, sendItems, storages} {
		require.NoError(t, os.MkdirAll(dir, 0o755))
	}

	oldMedia, oldSendItems, oldStorages := config.PathMedia, config.PathSendItems, config.PathStorages
	config.PathMedia, config.PathSendItems, config.PathStorages = media, sendItems, storages
	t.Cleanup(func() {
		config.PathMedia, config.PathSendItems, config.PathStorages = oldMedia, oldSendItems, oldStorages
	})
	return media, sendItems, storages
}

func TestSweepDeletesOnlyExpiredMedia(t *testing.T) {
	media, sendItems, storages := useTempPaths(t)

	oldInbound := filepath.Join(media, extractMediaName(".jpg"))
	freshInbound := filepath.Join(media, extractMediaName(".jpg"))
	oldDownloaded := filepath.Join(media, "6289605618749", "2026-08-01", extractMediaName(".pdf"))
	freshDownloaded := filepath.Join(media, "6289605618750", "2026-10-13", extractMediaName(".pdf"))
	oldSent := filepath.Join(sendItems, "new-photo.jpg")
	freshSent := filepath.Join(sendItems, "temp_audio_abc.ogg")
	oldLegacyCopy := filepath.Join(storages, extractMediaName(".jpeg"))
	oldLegacyNoExt := filepath.Join(storages, extractMediaName(""))
	freshLegacyCopy := filepath.Join(storages, extractMediaName(".png"))

	writeFile(t, oldInbound, oldTime)
	writeFile(t, freshInbound, freshTime)
	writeFile(t, oldDownloaded, oldTime)
	writeFile(t, freshDownloaded, freshTime)
	writeFile(t, oldSent, oldTime)
	writeFile(t, freshSent, freshTime)
	writeFile(t, oldLegacyCopy, oldTime)
	writeFile(t, oldLegacyNoExt, oldTime)
	writeFile(t, freshLegacyCopy, freshTime)

	// Non-media files that must survive no matter how old they are.
	keep := []string{
		filepath.Join(media, ".gitignore"),
		filepath.Join(sendItems, ".gitignore"),
		filepath.Join(storages, "whatsapp.db"),
		filepath.Join(storages, "whatsapp.db-wal"),
		filepath.Join(storages, "whatsapp.db-shm"),
		filepath.Join(storages, "chatstorage.db"),
		filepath.Join(storages, "chatstorage.db-journal"),
		filepath.Join(storages, "whatsapp-keys.db"),
		filepath.Join(storages, "history-1760000000-6289605618749:11@s.whatsapp.net-1-RECENT.json"),
		filepath.Join(storages, "notes.txt"),
		// storages/ is swept non-recursively, even for media-shaped names.
		filepath.Join(storages, "backup", extractMediaName(".jpg")),
	}
	for _, path := range keep {
		writeFile(t, path, oldTime)
	}

	// Per-chat directories are created when media is downloaded, so they are
	// as old as their oldest file.
	setDirTime(t, filepath.Join(media, "6289605618749", "2026-08-01"), oldTime)
	setDirTime(t, filepath.Join(media, "6289605618749"), oldTime)
	setDirTime(t, filepath.Join(media, "6289605618750", "2026-10-13"), freshTime)
	setDirTime(t, filepath.Join(media, "6289605618750"), freshTime)

	result := Sweep(DefaultTargets(), sweepAfter)

	for _, path := range []string{oldInbound, oldDownloaded, oldSent, oldLegacyCopy, oldLegacyNoExt} {
		assert.NoFileExists(t, path)
	}
	for _, path := range append(keep, freshInbound, freshDownloaded, freshSent, freshLegacyCopy) {
		assert.FileExists(t, path)
	}

	// The emptied, idle per-chat tree is pruned; the active one and the roots stay.
	assert.NoDirExists(t, filepath.Join(media, "6289605618749"))
	assert.DirExists(t, filepath.Join(media, "6289605618750", "2026-10-13"))
	for _, root := range []string{media, sendItems, storages} {
		assert.DirExists(t, root)
	}

	assert.Equal(t, 5, result.Files)
	assert.Equal(t, int64(5*len("data")), result.Bytes)
	assert.Equal(t, 2, result.Dirs)
	assert.Zero(t, result.Errors)
}

func TestSweepKeepsRecentlyActiveEmptyDirectory(t *testing.T) {
	media, _, _ := useTempPaths(t)
	dir := filepath.Join(media, "6289605618749", "2026-10-14")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	setDirTime(t, dir, freshTime)
	setDirTime(t, filepath.Dir(dir), freshTime)

	result := Sweep(DefaultTargets(), sweepAfter)

	// A fresh empty directory may be about to receive a download.
	assert.DirExists(t, dir)
	assert.Zero(t, result.Dirs)
}

func TestSweepDoesNotFollowSymlinks(t *testing.T) {
	media, _, _ := useTempPaths(t)
	outside := filepath.Join(t.TempDir(), "outside.jpg")
	writeFile(t, outside, oldTime)
	link := filepath.Join(media, extractMediaName(".jpg"))
	require.NoError(t, os.Symlink(outside, link))

	outsideDir := t.TempDir()
	outsideDirFile := filepath.Join(outsideDir, extractMediaName(".jpg"))
	writeFile(t, outsideDirFile, oldTime)
	require.NoError(t, os.Symlink(outsideDir, filepath.Join(media, "linked-dir")))

	Sweep(DefaultTargets(), sweepAfter)

	assert.FileExists(t, outside)
	assert.FileExists(t, outsideDirFile)
}

func TestSweepToleratesMissingDirectories(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "does-not-exist")

	result := Sweep([]Target{{Dir: missing, Recursive: true}, {Dir: ""}}, sweepAfter)

	assert.Equal(t, Result{}, result)
}

func TestStartDisabledDoesNothing(t *testing.T) {
	media, _, _ := useTempPaths(t)
	path := filepath.Join(media, extractMediaName(".jpg"))
	writeFile(t, path, oldTime)

	for _, days := range []int{0, -1} {
		Start(t.Context(), days, DefaultTargets())
	}

	// Start is asynchronous when enabled; give a (wrongly) started sweeper
	// time to run before asserting nothing was deleted.
	time.Sleep(50 * time.Millisecond)
	assert.FileExists(t, path)
}

func TestRunSweepsOnStartup(t *testing.T) {
	media, _, _ := useTempPaths(t)
	path := filepath.Join(media, extractMediaName(".jpg"))
	writeFile(t, path, time.Now().AddDate(0, 0, -40))

	Start(t.Context(), 30, DefaultTargets())

	assert.Eventually(t, func() bool {
		_, err := os.Stat(path)
		return os.IsNotExist(err)
	}, 2*time.Second, 10*time.Millisecond)
}

func TestIsLegacyStoragesMedia(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{extractMediaName(".jpg"), true},
		{extractMediaName(".jpeg"), true},
		{extractMediaName(""), true},
		{"whatsapp.db", false},
		{"whatsapp.db-wal", false},
		{"chatstorage.db", false},
		{"chatstorage.db-shm", false},
		{"history-1760000000-6289605618749:11@s.whatsapp.net-1-RECENT.json", false},
		{"1760000000-not-a-uuid.jpg", false},
		{"x1760000000-" + uuid.NewString() + ".jpg", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, IsLegacyStoragesMedia(tt.name))
		})
	}
}

func TestCutoff(t *testing.T) {
	assert.Equal(t, time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC), Cutoff(sweepNow, 30))
}

func TestSweepFollowsSymlinkedRoot(t *testing.T) {
	realDir := t.TempDir()
	old := filepath.Join(realDir, extractMediaName(".jpg"))
	fresh := filepath.Join(realDir, extractMediaName(".jpg"))
	writeFile(t, old, oldTime)
	writeFile(t, fresh, freshTime)
	link := filepath.Join(t.TempDir(), "media")
	require.NoError(t, os.Symlink(realDir, link))

	result := Sweep([]Target{{Dir: link + "/", Recursive: true}}, sweepAfter)

	assert.NoFileExists(t, old)
	assert.FileExists(t, fresh)
	assert.DirExists(t, realDir)
	assert.Equal(t, 1, result.Files)
}

func TestCutoffCapsHugeRetention(t *testing.T) {
	cutoff := Cutoff(sweepNow, math.MaxInt)

	assert.True(t, cutoff.Before(sweepNow))
	assert.Equal(t, Cutoff(sweepNow, maxRetentionDays), cutoff)
}
