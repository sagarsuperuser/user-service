package router

import (
	"context"
	"net/http"

	"github.com/sagarsuperuser/userprofile/errdefs"
	sessionUtils "github.com/sagarsuperuser/userprofile/internal/session"
	"github.com/sagarsuperuser/userprofile/store"
)

// sessionContextKey is unexported to avoid collisions.
type sessionContextKey struct{}

// SessionInfoFromContext retrieves session info from context.
func SessionInfoFromContext(ctx context.Context) (*store.SessionInfo, bool) {
	s, ok := ctx.Value(sessionContextKey{}).(*store.SessionInfo)
	return s, ok
}

func readCookie(r *http.Request, name string) (string, bool) {
	c, err := r.Cookie(name)
	if err != nil {
		return "", false
	}
	return c.Value, true
}

// AuthSession wraps a route to enforce session authentication.
func AuthSession(store *store.Store) RouteWrapper {
	return func(route Route) Route {
		return localRoute{
			method: route.Method(),
			path:   route.Path(),
			handler: func(ctx context.Context, rw http.ResponseWriter, req *http.Request, vars map[string]string) error {
				token, ok := readCookie(req, sessionUtils.SessionCookieName)
				if !ok || token == "" {
					return errdefs.Unauthorized(sessionUtils.ErrSessionCookieNotFound)
				}
				sess, err := store.GetActiveSessionByToken(ctx, token)
				if err != nil {
					return errdefs.Unauthorized(sessionUtils.ErrSesssionExpired)
				}

				ctxWithSession := context.WithValue(ctx, sessionContextKey{}, sess)
				return route.Handler()(ctxWithSession, rw, req.WithContext(ctxWithSession), vars)
			},
		}
	}
}
