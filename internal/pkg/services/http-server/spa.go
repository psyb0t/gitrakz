package httpserver

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"

	"github.com/psyb0t/aichteeteapee"
	"github.com/psyb0t/ctxerrors"
)

// webDistRoot is spa.go's own directory prefix on the embedded tree, since
// the go:embed pattern is relative to this file and cannot climb out of it
// with "..", so the Svelte build output lives right next to this package
// rather than at the repo root. Wave 5 replaces the placeholder under
// web/dist with the real build; this file and its embed directive don't
// change.
const (
	webDistRoot  = "web/dist"
	spaIndexFile = "index.html"

	// spaAllowedMethods is the Allow header value on a 405 from the SPA
	// route — see newSPAHandler's method check.
	spaAllowedMethods = "GET, HEAD"
)

//go:embed all:web/dist
var webDistFS embed.FS

// newSPAHandler serves the embedded Svelte build. Any request path that
// doesn't match a real file under web/dist falls back to index.html, so
// the client-side router resolves deep links like /templates/foo.
//
// The route it's mounted on (see router.go) carries no method restriction
// — net/http.ServeMux panics on registering a GET-only pattern alongside
// the any-method /api/{path...} pattern, since neither dominates the
// other's specificity — so this handler enforces GET/HEAD itself instead.
func newSPAHandler() (http.Handler, error) {
	sub, err := fs.Sub(webDistFS, webDistRoot)
	if err != nil {
		return nil, ctxerrors.Wrap(err, "sub embedded web/dist filesystem")
	}

	fileServer := http.FileServer(http.FS(sub))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set(aichteeteapee.HeaderNameAllow, spaAllowedMethods)
			w.WriteHeader(http.StatusMethodNotAllowed)

			return
		}

		name := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if name == "" {
			name = spaIndexFile
		}

		if _, statErr := fs.Stat(sub, name); statErr != nil {
			r = r.Clone(r.Context())
			r.URL.Path = "/" + spaIndexFile
		}

		fileServer.ServeHTTP(w, r)
	}), nil
}
