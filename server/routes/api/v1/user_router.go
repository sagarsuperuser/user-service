package v1

import "github.com/sagarsuperuser/userprofile/internal/router"

type userRouter struct {
	backend *APIV1Service
	routes  []router.Route
}

// NewUserRouter initializes a router for user endpoints.
func NewUserRouter(svc *APIV1Service) router.Router {
	r := &userRouter{backend: svc}
	r.initRoutes()
	return r
}

func (ur *userRouter) Routes() []router.Route {
	return ur.routes
}

func (ur *userRouter) initRoutes() {
	// protect user routes with session middleware.
	sessionMW := router.AuthSession(ur.backend.Store)
	ur.routes = []router.Route{
		router.NewGetRoute("/user/me", ur.backend.GetCurrentUser, sessionMW),
		router.NewPatchRoute("/user", ur.backend.UpdateUser, sessionMW),
	}
}
