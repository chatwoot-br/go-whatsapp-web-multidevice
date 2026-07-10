package error

import "net/http"

type LoginError string

// Error for complying the error interface
func (e LoginError) Error() string {
	return string(e)
}

// ErrCode will return the error code based on the error data type
func (e LoginError) ErrCode() string {
	return "ALREADY_LOGGED_IN"
}

// StatusCode will return the HTTP status code based on the error data type
func (e LoginError) StatusCode() int {
	return http.StatusBadRequest
}

type AuthError string

func (err AuthError) Error() string {
	return string(err)
}

// ErrCode will return the error code based on the error data type
func (err AuthError) ErrCode() string {
	return "AUTHENTICATION_ERROR"
}

// StatusCode will return the HTTP status code based on the error data type
func (err AuthError) StatusCode() int {
	return http.StatusUnauthorized
}

type qrChannelError string

func (err qrChannelError) Error() string {
	return string(err)
}

// ErrCode will return the error code based on the error data type
func (err qrChannelError) ErrCode() string {
	return "QR_CHANNEL_ERROR"
}

// StatusCode will return the HTTP status code based on the error data type
func (err qrChannelError) StatusCode() int {
	return http.StatusInternalServerError
}

type sessionSavedError string

func (err sessionSavedError) Error() string {
	return string(err)
}

// ErrCode will return the error code based on the error data type
func (err sessionSavedError) ErrCode() string {
	return "SESSION_SAVED_ERROR"
}

// StatusCode will return the HTTP status code based on the error data type
func (err sessionSavedError) StatusCode() int {
	return http.StatusInternalServerError
}

type notFoundError string

func (err notFoundError) Error() string {
	return string(err)
}

// ErrCode will return the error code based on the error data type
func (err notFoundError) ErrCode() string {
	return "NOT_FOUND"
}

// StatusCode will return the HTTP status code based on the error data type
func (err notFoundError) StatusCode() int {
	return http.StatusNotFound
}

var (
	ErrAlreadyLoggedIn = LoginError("you are already logged in.")
	ErrNotConnected    = AuthError("you are not connect to services server, please reconnect")
	ErrNotLoggedIn     = AuthError("you are not logged in")
	ErrReconnect       = AuthError("reconnect error")
	// ErrSessionDeleted marks a device slot whose WhatsApp session is gone (remote or
	// explicit logout). Reconnect cannot recover it — only a new pairing can — so
	// callers should surface the re-login path (QR via /app/login or pair code)
	// instead of retrying the connection.
	ErrSessionDeleted = AuthError("device is logged out (session deleted); re-pair it via GET /app/login (QR) or GET /app/login-with-code")
	ErrQrChannel      = qrChannelError("QR channel error")
	ErrSessionSaved   = sessionSavedError("your session have been saved, please wait to connect 2 second and refresh again")
	ErrDeviceNotFound = notFoundError("device not found")
)
