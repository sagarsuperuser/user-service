package httputil

import (
	"context"
	"net/http"
)

// APIVersionKey is the client's requested API version.
type APIVersionKey struct{}

// APIFunc is an adapter to allow the use of ordinary functions as API endpoints.
// Any function that has the appropriate signature can be registered as an API endpoint.
type APIFunc func(ctx context.Context, w http.ResponseWriter, r *http.Request, vars map[string]string) error
