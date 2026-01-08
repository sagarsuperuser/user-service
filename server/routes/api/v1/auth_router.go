package v1

import "github.com/sagarsuperuser/userprofile/internal/router"

type authRouter struct {
	backend *APIV1Service
	routes  []router.Route
}

// NewAuthRouter initializes a router for auth-related endpoints.
func NewAuthRouter(svc *APIV1Service) router.Router {
	r := &authRouter{
		backend: svc,
	}
	r.initRoutes()
	return r
}

func (ar *authRouter) Routes() []router.Route {
	return ar.routes
}

func (ar *authRouter) initRoutes() {
	// protect routes with session middleware as a RouteWrapper
	sessionMW := router.AuthSession(ar.backend.Store)
	ar.routes = []router.Route{
		router.NewPostRoute("/auth/signup", ar.backend.SignUp),
		router.NewPostRoute("/auth/login", ar.backend.LogIn),
		router.NewGetRoute("/oauth2/login", ar.backend.Oauth2Login),
		router.NewGetRoute("/oauth2/callback", ar.backend.Oauth2Callback),
		router.NewGetRoute("/auth/logout", ar.backend.LogOut, sessionMW),
	}
}
