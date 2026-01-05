package middlewares

import (
	"net/http"
	"runtime/debug"

	"github.com/rs/zerolog/hlog"
)

// Recovery is a mux middleware that recovers from panics and logs them.
func Recovery() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					hlog.FromRequest(r).
						Error().
						Interface("panic", rec).
						Bytes("stack", debug.Stack()).
						Msg("panic recovered")

					http.Error(w, "internal server error", http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
