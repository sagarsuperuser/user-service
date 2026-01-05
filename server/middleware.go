package server

import (
	"github.com/sagarsuperuser/userprofile/internal/httputil"
)

// handlerWithGlobalMiddlewares wraps the handler function for a request with
// the server's global middlewares. The order of the middlewares is backwards,
// meaning that the first in the list will be evaluated last.
func (s *Server) handlerWithGlobalMiddlewares(handler httputil.APIFunc) httputil.APIFunc {
	next := handler

	for i := len(s.middlewares) - 1; i >= 0; i-- {
		next = s.middlewares[i].WrapHandler(next)
	}
	// TODO - add RequestMiddleware for debugging requests/response
	return next
}
