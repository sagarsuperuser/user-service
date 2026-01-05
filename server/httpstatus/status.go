package httpstatus

import (
	"context"
	"fmt"
	"net/http"

	cerrdefs "github.com/containerd/errdefs"
	"github.com/rs/zerolog"
)

// FromError retrieves status code from error message.
func FromError(err error) int {
	if err == nil {
		zerolog.Ctx(context.TODO()).Err(err).Msg("unexpected HTTP error handling")
		return http.StatusInternalServerError
	}

	// Resolve the error to ensure status is chosen from the first outermost error
	rerr := cerrdefs.Resolve(err)

	// Note that the below functions are already checking the error causal chain for matches.
	// Only check errors from the errdefs package, no new error type checking may be added
	switch {
	case cerrdefs.IsNotFound(rerr):
		return http.StatusNotFound
	case cerrdefs.IsInvalidArgument(rerr):
		return http.StatusBadRequest
	case cerrdefs.IsConflict(rerr):
		return http.StatusConflict
	case cerrdefs.IsUnauthorized(rerr):
		return http.StatusUnauthorized
	case cerrdefs.IsUnavailable(rerr):
		return http.StatusServiceUnavailable
	case cerrdefs.IsPermissionDenied(rerr):
		return http.StatusForbidden
	case cerrdefs.IsNotModified(rerr):
		return http.StatusNotModified
	case cerrdefs.IsNotImplemented(rerr):
		return http.StatusNotImplemented
	case cerrdefs.IsInternal(rerr) || cerrdefs.IsDataLoss(rerr) || cerrdefs.IsDeadlineExceeded(rerr) || cerrdefs.IsCanceled(rerr):
		return http.StatusInternalServerError
	default:
		switch e := err.(type) {
		case interface{ Unwrap() error }:
			return FromError(e.Unwrap())
		case interface{ Unwrap() []error }:
			for _, ue := range e.Unwrap() {
				if statusCode := FromError(ue); statusCode != http.StatusInternalServerError {
					return statusCode
				}
			}
		}

		if !cerrdefs.IsUnknown(err) {
			zerolog.Ctx(context.TODO()).
				Debug().
				Str("module", "api").
				Err(err).
				Str("error_type", fmt.Sprintf("%T", err)).
				Msg("FIXME: Got an API for which error does not match any expected type!!!")
		}

		return http.StatusInternalServerError
	}
}
