package utils

import (
	"errors"
	"net/url"
	"strings"
)

// redactedUserinfo replaces the user and password of a URL in log output.
const redactedUserinfo = "***"

// RedactURLCredentials hides the user and password of a URL (e.g. a proxy URL)
// so it can be logged, keeping scheme, host and port for diagnosis:
// "socks5://user:pass@host:1080" becomes "socks5://***:***@host:1080".
// Values that do not parse as a URL never come back verbatim when they could
// carry credentials.
func RedactURLCredentials(raw string) string {
	if u, err := url.Parse(raw); err == nil && u.Opaque == "" {
		if u.User == nil {
			return raw
		}
		// Splice the mask in by hand: url.UserPassword would percent-encode "*".
		userinfo := redactedUserinfo
		if _, hasPassword := u.User.Password(); hasPassword {
			userinfo += ":" + redactedUserinfo
		}
		u.User = nil
		prefix := "//"
		if u.Scheme != "" {
			prefix = u.Scheme + "://"
		}
		if rest, ok := strings.CutPrefix(u.String(), prefix); ok {
			return prefix + userinfo + "@" + rest
		}
	}
	return redactUnparsedURL(raw)
}

// redactUnparsedURL is the fallback for values url.Parse rejects (or reads as
// opaque, like "user:pass@host:1080"): everything before the last "@" after
// the scheme is treated as userinfo and dropped.
func redactUnparsedURL(raw string) string {
	at := strings.LastIndex(raw, "@")
	if at < 0 {
		if strings.Contains(raw, ":") {
			// Could be "user:pass" with the host missing; nothing safe to keep.
			return "<redacted>"
		}
		return raw
	}
	prefix := ""
	if i := strings.Index(raw, "://"); i >= 0 && i < at {
		prefix = raw[:i+len("://")]
	}
	return prefix + redactedUserinfo + ":" + redactedUserinfo + raw[at:]
}

// RedactURLError rewrites the *url.Error that url.Parse returns, whose message
// embeds the raw input URL, so the credentials it carries are not logged.
// Other errors are returned unchanged.
func RedactURLError(err error) error {
	var urlErr *url.Error
	if !errors.As(err, &urlErr) {
		return err
	}
	return &url.Error{Op: urlErr.Op, URL: RedactURLCredentials(urlErr.URL), Err: urlErr.Err}
}
