import { svelte } from "@sveltejs/vite-plugin-svelte";
import { defineConfig } from "vite";

// The Go binary's spa.go embeds internal/pkg/services/http-server/web/dist
// via a go:embed directive relative to spa.go's own directory, which cannot
// climb out of that directory with "..". So the Svelte build output must
// land directly under that path, not under this project's own dist/ or
// build/. emptyOutDir wipes the placeholder index.html that ships there
// today.
const HTTP_SERVER_WEB_DIST =
  "../internal/pkg/services/http-server/web/dist";

// Local dev only (npm run dev) — proxies /api to a running gitrakz backend
// so the SPA can be developed against real data without a full build. Not
// used by the production build; the embedded binary serves API + SPA from
// the same origin, so no proxy exists there.
const DEV_API_PROXY_TARGET =
  process.env["VITE_API_PROXY_TARGET"] ?? "http://localhost:8080";

export default defineConfig({
  plugins: [svelte()],
  build: {
    outDir: HTTP_SERVER_WEB_DIST,
    emptyOutDir: true,
  },
  server: {
    proxy: {
      "/api": {
        target: DEV_API_PROXY_TARGET,
        changeOrigin: true,
      },
    },
  },
});
