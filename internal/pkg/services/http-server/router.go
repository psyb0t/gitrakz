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
// (e.g. an any-method "/api/v1/{path...}" vs a GET-only "/{path...}"), so
// spaHandler itself rejects non-GET/HEAD (see newSPAHandler) rather than
// the route restricting the method. The OpenAPI paths are version-less;
// api.HandlerWithOptions is given BaseURL "/api/v1" (see http_server.go),
// so it registers "/api/v1/..." into every route (see
// internal/pkg/http/api/api.gen.go), and apiMountPattern forwards the raw
// request into that sub-handler unmodified — no path stripping. ServeMux
// prefers the more specific "/api/v1/..." pattern over the catch-all SPA
// fallback, so anything under /api/v1/ never reaches spaMountPattern.
const (
	apiBaseURL      = "/api/v1"
	apiMountPattern = apiBaseURL + "/{path...}"
	spaMountPattern = "/{path...}"
)

// newRouter builds the serbewr.Router gitrakz serves: the generated API
// under /api/v1/ (Bearer-gated when authToken is set) and the embedded SPA
// everywhere else.
func newRouter(
	authToken string,
	apiHandler http.Handler,
	spaHandler http.Handler,
) *serbewr.Router {
	var apiMiddlewares []middleware.Middleware
	if authToken != "" {
		apiMiddlewares = append(
			apiMiddlewares,
			middleware.BearerAuth(
				middleware.WithBearerAuthTokens(authToken),
			),
		)
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
