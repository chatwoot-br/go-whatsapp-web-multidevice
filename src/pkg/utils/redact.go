package utils

import (
	"errors"
	"net/url"
	"regexp"
	"sort"
	"strings"
)

// redactedUserinfo replaces the user and password of a URL in log output.
const redactedUserinfo = "***"

// errURLDetailsWithheld replaces url.Parse's detail message, which quotes
// pieces of the input (e.g. `invalid port ":s3cret" after host`).
var errURLDetailsWithheld = errors.New("invalid URL (details withheld: they may contain credentials)")

var leadingURLScheme = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9+.-]*://`)

// RedactURLCredentials hides the user and password of a URL (e.g. a proxy URL)
// so it can be logged, keeping scheme, host and port for diagnosis:
// "socks5://user:pass@host:1080" becomes "socks5://***:***@host:1080".
//
// The userinfo is taken as everything before the last "@" rather than what
// url.Parse reports, because an unescaped '#', '/', '?' or '@' in a password
// makes url.Parse split the credentials into host, path or fragment.
func RedactURLCredentials(raw string) string {
	prefix, userinfo, rest, ok := splitUserinfo(raw)
	if !ok {
		if _, err := url.Parse(raw); err != nil && strings.Contains(raw, ":") {
			// Could be "scheme://user:pass" with the host missing.
			return "<redacted>"
		}
		return raw
	}
	mask := redactedUserinfo
	if strings.Contains(userinfo, ":") {
		mask += ":" + redactedUserinfo
	}
	return prefix + mask + rest
}

func RedactURLError(err error, raw string) error {
	if err == nil {
		return nil
	}
	var urlErr *url.Error
	if errors.As(err, &urlErr) && urlErr.URL == raw {
		return &url.Error{Op: urlErr.Op, URL: RedactURLCredentials(urlErr.URL), Err: errURLDetailsWithheld}
	}
	_, userinfo, _, ok := splitUserinfo(raw)
	if !ok {
		return err
	}
	u, parseErr := url.Parse(raw)
	if parseErr != nil || u.User == nil || u.User.String() != userinfo {
		return errURLDetailsWithheld
	}
	secrets := strings.Split(userinfo, ":")
	secrets = append(secrets, userinfo, u.User.Username())
	if password, hasPassword := u.User.Password(); hasPassword {
		secrets = append(secrets, password)
	}
	sort.Slice(secrets, func(i, j int) bool { return len(secrets[i]) > len(secrets[j]) })
	var replacements []string
	for _, secret := range secrets {
		if secret != "" {
			replacements = append(replacements, secret, redactedUserinfo)
		}
	}
	msg := err.Error()
	redacted := strings.NewReplacer(replacements...).Replace(msg)
	if redacted == msg {
		return err
	}
	return errors.New(redacted)
}

// splitUserinfo splits raw into "scheme://", the userinfo, and "@host..." at
// the last "@". ok is false when raw has no "@".
func splitUserinfo(raw string) (prefix, userinfo, rest string, ok bool) {
	at := strings.LastIndex(raw, "@")
	if at < 0 {
		return "", "", "", false
	}
	if scheme := leadingURLScheme.FindString(raw); len(scheme) > 0 && len(scheme) <= at {
		prefix = scheme
	} else if strings.HasPrefix(raw, "//") {
		prefix = "//"
	}
	return prefix, raw[len(prefix):at], raw[at:], true
}
