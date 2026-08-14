package httpserver

import (
	"net/http"

	"github.com/psyb0t/aichteeteapee/serbewr"
	"github.com/psyb0t/aichteeteapee/serbewr/middleware"
)

// apiMountPattern / spaMountPattern are Go 1.22+ ServeMux wildcard
// patterns, both registered with NO method restriction (RouteConfig.Method
// left empty) — net/http.ServeMux panics at registration time on two
// patterns whose specificity disagrees across the method and path axes
// (e.g. an any-method "/api/{path...}" vs a GET-only "/{path...}"), so
// spaHandler itself rejects non-GET/HEAD (see newSPAHandler) rather than
// the route restricting the method. api.HandlerWithOptions already bakes
// "/api/..." into every route it registers (see
// internal/pkg/http/api/api.gen.go), so apiMountPattern forwards the raw
// request into that sub-handler unmodified — no BaseURL prefix, no path
// stripping. ServeMux prefers the more specific "/api/..." pattern over
// the catch-all SPA fallback, so anything under /api/ never reaches
// spaMountPattern.
const (
	apiMountPattern = "/api/{path...}"
	spaMountPattern = "/{path...}"
)

// newRouter builds the serbewr.Router gitrakz serves: the generated API
// under /api/ (Bearer-gated when authToken is set) and the embedded SPA
// everywhere else.
func newRouter(
	authToken string,
	apiHandler http.Handler,
	spaHandler http.Handler,
) *serbewr.Router {
	var apiMiddlewares []middleware.Middleware
	if authToken != "" {
		apiMiddlewares = append(apiMiddlewares, bearerAuth(authToken))
	}

	return &serbewr.Router{
		GlobalMiddlewares: []middleware.Middleware{
			middleware.RequestID(),
			middleware.Logger(),
			middleware.Recovery(),
			middleware.SecurityHeaders(),
			middleware.CORS(),
		},
		Groups: []serbewr.GroupConfig{
			{
				Middlewares: apiMiddlewares,
				Routes: []serbewr.RouteConfig{
					{Path: apiMountPattern, Handler: apiHandler.ServeHTTP},
				},
			},
			{
				Routes: []serbewr.RouteConfig{
					{Path: spaMountPattern, Handler: spaHandler.ServeHTTP},
				},
			},
		},
	}
}
