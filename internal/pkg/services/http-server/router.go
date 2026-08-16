package httpserver

import (
	"net/http"

	"github.com/psyb0t/aichteeteapee/serbewr"
	"github.com/psyb0t/aichteeteapee/serbewr/middleware"
)

// apiMountPattern / mcpMountPattern / spaMountPattern are Go 1.22+ ServeMux
// patterns, all registered with no method restriction: ServeMux panics at
// registration if two patterns disagree in specificity across the method
// and path axes (e.g. any-method "/api/v1/{path...}" vs GET-only
// "/{path...}"), so spaHandler rejects non-GET/HEAD itself instead (see
// newSPAHandler). api.HandlerWithOptions gets BaseURL "/api/v1" (see
// http_server.go), so it registers "/api/v1/..." (see
// internal/pkg/http/api/api.gen.go) and apiMountPattern forwards requests
// into it unmodified. mcpMountPattern is an exact, single-endpoint match —
// the MCP streamable HTTP transport serves GET/POST/DELETE at one URL, not
// a subtree. ServeMux prefers the more specific patterns, so neither
// /api/v1/ nor /mcp ever falls through to spaMountPattern.
const (
	apiBaseURL      = "/api/v1"
	apiMountPattern = apiBaseURL + "/{path...}"
	mcpMountPattern = "/mcp"
	spaMountPattern = "/{path...}"
)

// newRouter builds the serbewr.Router gitrakz serves: the generated API
// under /api/v1/ and the MCP server at /mcp (both Bearer-gated when
// authToken is set, sharing the same gate as the rest of the API), and the
// embedded SPA everywhere else.
func newRouter(
	authToken string,
	apiHandler http.Handler,
	mcpHandler http.Handler,
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
					{Path: mcpMountPattern, Handler: mcpHandler.ServeHTTP},
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
