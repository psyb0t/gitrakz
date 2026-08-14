package httpserver

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/psyb0t/aichteeteapee"
	"github.com/psyb0t/aichteeteapee/serbewr/middleware"
	"github.com/psyb0t/ctxscope"
)

// bearerAuth rejects any request whose Authorization header doesn't carry
// "Bearer <token>" matching token exactly (constant-time compare — this
// gates every state-changing endpoint under /api/ once cfg.AuthToken is
// set, per the single-user "still gated" auth model).
func bearerAuth(token string) middleware.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get(aichteeteapee.HeaderNameAuthorization)
			provided := strings.TrimPrefix(
				header, aichteeteapee.AuthSchemeBearer,
			)

			hasScheme := len(provided) != len(header)
			matches := subtle.ConstantTimeCompare(
				[]byte(provided), []byte(token),
			) == 1

			if hasScheme && matches {
				next.ServeHTTP(w, r)

				return
			}

			ctxscope.GetLogger(r.Context()).Warn(
				"rejected unauthenticated api request",
				"path", r.URL.Path,
			)

			aichteeteapee.WriteJSON(
				w,
				http.StatusUnauthorized,
				aichteeteapee.ErrorResponseUnauthorized,
			)
		})
	}
}
