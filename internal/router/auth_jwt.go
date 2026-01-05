package router

import (
	"context"
	"net/http"

	"github.com/sagarsuperuser/userprofile/errdefs"
	jwtUtils "github.com/sagarsuperuser/userprofile/internal/jwt"
)

type userIDContextKey struct{}

// UserIDFromContext retrieves user info from context.
func UserIDFromContext(ctx context.Context) (int64, bool) {
	s, ok := ctx.Value(userIDContextKey{}).(int64)
	return s, ok
}

// AuthJWT wraps a route to enforce JWT auth.
func AuthJWT(secret string) RouteWrapper {
	return func(route Route) Route {
		return localRoute{
			method: route.Method(),
			path:   route.Path(),
			handler: func(ctx context.Context, rw http.ResponseWriter, req *http.Request, vars map[string]string) error {
				token, err := jwtUtils.ValidateAccessToken(req, secret)
				if err != nil {
					return errdefs.Unauthorized(jwtUtils.ErrJWTValidate)
				}
				userID, err := jwtUtils.GetUserIDFromToken(token)
				if err != nil {
					return errdefs.Unauthorized(jwtUtils.ErrJWTUserIDNotFound)
				}

				ctxWithUser := context.WithValue(ctx, userIDContextKey{}, userID)
				return route.Handler()(ctxWithUser, rw, req.WithContext(ctxWithUser), vars)
			},
		}
	}
}
