package router

import (
	"context"
	"errors"
	"net/http"

	"github.com/sagarsuperuser/userprofile/errdefs"
	sessionUtils "github.com/sagarsuperuser/userprofile/internal/session"
	"github.com/sagarsuperuser/userprofile/store"
)

// sessionContextKey is unexported.
type sessionContextKey struct{}

// SessionInfoFromContext retrieves session info from context.
func SessionInfoFromContext(ctx context.Context) *store.SessionInfo {
	if ctx == nil {
		return nil
	}

	if val := ctx.Value(sessionContextKey{}); val != nil {
		return val.(*store.SessionInfo)
	}

	return nil
}

func readCookie(r *http.Request, name string) string {
	c, err := r.Cookie(name)
	if err != nil {
		return ""
	}
	return c.Value
}

// AuthSession wraps a route to enforce session authentication.
func AuthSession(store *store.Store) RouteWrapper {
	return func(route Route) Route {
		return localRoute{
			method: route.Method(),
			path:   route.Path(),
			handler: func(ctx context.Context, rw http.ResponseWriter, req *http.Request, vars map[string]string) error {
				token := readCookie(req, sessionUtils.SessionCookieName)
				if token == "" {
					return errdefs.Unauthorized(sessionUtils.ErrSessionCookieNotFound)
				}
				sess, err := store.GetActiveSessionByToken(ctx, token)
				if err != nil {
					if errors.Is(err, sessionUtils.ErrSesssionExpired) {
						return errdefs.Unauthorized(err)
					}
					return errdefs.System(err)
				}

				ctx = context.WithValue(ctx, sessionContextKey{}, sess)
				return route.Handler()(ctx, rw, req, vars)
			},
		}
	}
}
