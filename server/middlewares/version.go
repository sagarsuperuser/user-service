package middlewares

import (
	"context"
	"fmt"
	"net/http"
	"runtime"

	"github.com/sagarsuperuser/userprofile/internal/httputil"
	"github.com/sagarsuperuser/userprofile/internal/versions"
)

// VersionMiddleware validates API versions and decorates responses with version headers.
type VersionMiddleware struct {
	serverVersion     string
	defaultAPIVersion string
	minAPIVersion     string
}

func NewVersionMiddleware(serverVersion, defaultAPIVersion, minAPIVersion string) (*VersionMiddleware, error) {
	if defaultAPIVersion == "" || minAPIVersion == "" {
		return nil, fmt.Errorf("default and minimum API versions must be set")
	}
	if versions.LessThan(defaultAPIVersion, minAPIVersion) {
		return nil, fmt.Errorf("default API version (%s) must be >= min API version (%s)", defaultAPIVersion, minAPIVersion)
	}
	return &VersionMiddleware{
		serverVersion:     serverVersion,
		defaultAPIVersion: defaultAPIVersion,
		minAPIVersion:     minAPIVersion,
	}, nil
}

type versionUnsupportedError struct {
	version, minVersion, maxVersion string
}

func (e versionUnsupportedError) Error() string {
	if e.minVersion != "" {
		return fmt.Sprintf("client version %s is too old. Minimum supported API version is %s, please upgrade your client", e.version, e.minVersion)
	}
	return fmt.Sprintf("client version %s is too new. Maximum supported API version is %s", e.version, e.maxVersion)
}

func (e versionUnsupportedError) InvalidParameter() {}

// WrapHandler implements the Middleware interface to enforce version checks.
func (v VersionMiddleware) WrapHandler(next httputil.APIFunc) httputil.APIFunc {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
		w.Header().Set("Server", fmt.Sprintf("user-service/%s (%s)", v.serverVersion, runtime.GOOS))
		w.Header().Set("Api-Version", v.defaultAPIVersion)
		w.Header().Set("Ostype", runtime.GOOS)

		apiVersion := vars["version"]
		if apiVersion == "" {
			apiVersion = v.defaultAPIVersion
		}
		if versions.LessThan(apiVersion, v.minAPIVersion) {
			return versionUnsupportedError{version: apiVersion, minVersion: v.minAPIVersion}
		}
		if versions.GreaterThan(apiVersion, v.defaultAPIVersion) {
			return versionUnsupportedError{version: apiVersion, maxVersion: v.defaultAPIVersion}
		}
		ctx = context.WithValue(ctx, httputil.APIVersionKey{}, apiVersion)
		return next(ctx, w, r, vars)
	}
}
