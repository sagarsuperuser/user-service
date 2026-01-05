package middlewares

import (
	"github.com/sagarsuperuser/userprofile/internal/httputil"
)

// Middleware is an interface to allow the use of ordinary functions as API filters.
// Any struct that has the appropriate signature can be registered as a middleware.
type Middleware interface {
	WrapHandler(httputil.APIFunc) httputil.APIFunc
}
