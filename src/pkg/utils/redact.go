package utils

import (
	"errors"
	"net/url"
	"strings"
)

// redactedUserinfo replaces the user and password of a URL in log output.
const redactedUserinfo = "***"

// errURLDetailsWithheld replaces url.Parse's detail message, which quotes
// pieces of the input (e.g. `invalid port ":s3cret" after host`).
var errURLDetailsWithheld = errors.New("invalid URL (details withheld: they may contain credentials)")

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

// RedactURLError makes an error involving the URL raw safe to log. A
// *url.Error about raw itself (from url.Parse) has its URL redacted and its
// detail message withheld; any other error has the user and password of raw
// masked out of its message, e.g. whatsmeow's `unsupported proxy scheme "user"`
// for "user:pass@host", or a dial error naming a misparsed user as the host.
func RedactURLError(err error, raw string) error {
	var urlErr *url.Error
	if errors.As(err, &urlErr) && urlErr.URL == raw {
		return &url.Error{Op: urlErr.Op, URL: RedactURLCredentials(urlErr.URL), Err: errURLDetailsWithheld}
	}
	_, userinfo, _, ok := splitUserinfo(raw)
	if err == nil || !ok {
		return err
	}
	secrets := strings.Split(userinfo, ":")
	// A host url.Parse found inside our userinfo is a credential fragment, e.g.
	// "ss" for "user:pa@ss/word@host", which then shows up in dial errors.
	if u, parseErr := url.Parse(raw); parseErr == nil && u.Hostname() != "" && strings.Contains(userinfo, u.Hostname()) {
		secrets = append(secrets, u.Hostname())
	}
	msg := err.Error()
	redacted := msg
	for _, secret := range secrets {
		if secret != "" {
			redacted = strings.ReplaceAll(redacted, secret, redactedUserinfo)
		}
	}
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
	if i := strings.Index(raw, "://"); i >= 0 && i < at {
		prefix = raw[:i+len("://")]
	} else if strings.HasPrefix(raw, "//") {
		prefix = "//"
	}
	return prefix, raw[len(prefix):at], raw[at:], true
}
