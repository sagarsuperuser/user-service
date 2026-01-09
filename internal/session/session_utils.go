package sessionUtils

import (
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
)

const (
	SessionDuration   = 24 * time.Hour
	SessionCookieName = "sid"
)

var ErrSesssionExpired error = errors.New("session expired or not found")
var ErrSessionKeyNotFound error = errors.New("session key not found in context")
var ErrSessionCookieNotFound error = errors.New("session cookie not found in request")

// GenerateSessionID generates a unique session ID.
//
// Uses UUID v4 (random) for high entropy and uniqueness.
// Session IDs are stored in cookies and used to identify user sessions.
func GenerateSessionID() string {
	return uuid.NewString()
}

// SetSessionookie sets the token to the cookie.
func SetSessionCookie(rw http.ResponseWriter, req *http.Request, expiry time.Time, token string) {
	http.SetCookie(rw, &http.Cookie{
		Name:     SessionCookieName,
		Value:    token,
		Expires:  expiry,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

// ClearSessionCookie expires the session cookie on the client.
func ClearSessionCookie(rw http.ResponseWriter) {
	http.SetCookie(rw, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		MaxAge:   -1,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}
