package httputil

import (
	"context"
	"encoding/json"
	"net/http"
)

// APIVersionKey is the client's requested API version.
type APIVersionKey struct{}

// APIFunc is an adapter to allow the use of ordinary functions as API endpoints.
// Any function that has the appropriate signature can be registered as an API endpoint.
type APIFunc func(ctx context.Context, w http.ResponseWriter, r *http.Request, vars map[string]string) error

// WriteRawJSON writes the value v to the http response stream as json with standard json encoding.
func WriteRawJSON(w http.ResponseWriter, status int, v any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}

// VersionFromContext returns an API version from the context using APIVersionKey.
// It panics if the context value does not have version.Version type.
func VersionFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}

	if val := ctx.Value(APIVersionKey{}); val != nil {
		return val.(string)
	}

	return ""
}
