# Changelog

All notable changes per release. Versions follow [semver](https://semver.org)
pre-1.0 conventions: minor bumps may include breaking REST changes (called out
explicitly), patch bumps are docs / build / fixes only.

## v0.4.1 — 2026-08-14

Docs: sync `.env.example` with the config changes from v0.3.0 and v0.4.0.

- Drop the removed `GITRAKZ_HTTP_ADDR` entry (the container listens on a fixed
  `:8080`; publish the host port with `-p`), and mark `GITRAKZ_GH_USER` optional
  (defaults to the `gh` CLI's authenticated login).

## v0.4.0 — 2026-08-14

`GITRAKZ_GH_USER` is now optional — with an authenticated `gh` CLI, gitrakz
tracks you with zero configuration.

- When `GITRAKZ_GH_USER` is unset, the sync engine defaults to the login the
  `gh` CLI is authenticated as (`gh api user`), resolved on the first sync. Set
  the variable only to track a *different* user's public activity. Previously it
  was required and gitrakz refused to start without it.

## v0.3.0 — 2026-08-14

**Breaking.** The `GITRAKZ_HTTP_ADDR` env var is removed — the container always
listens on `:8080`. If you set it, drop it and publish the host port with
`docker run -p <host-port>:8080` instead. Also toolifies the repository
generator.

- Remove the `GITRAKZ_HTTP_ADDR` knob. gitrakz ships as a Docker image, so the
  address that matters is the host port you map with `-p`; the in-container bind
  is now a fixed `:8080`, documented in the Dockerfile with `EXPOSE 8080`.
  Bare-metal `go run ./cmd run` also binds `:8080`.
- The repository generator is now a Go tool: `cmd/repogen` is registered in
  `go.mod`'s `tool` block and invoked via `go tool repogen` from the
  repositories `gen.go`, instead of `go run ../../../../cmd/repogen`. It builds
  from the pinned, vendored toolchain like every other generator, with no
  fragile relative path.
- `make build` now targets `./cmd` instead of the framework default `./cmd/...`,
  which broke once `cmd/repogen` was added as a second package main (`go build
  -o` can't write multiple packages to one file). A `scripts/make/build.sh`
  override — picked ahead of the framework copy — matches the release Dockerfile.

## v0.2.2 — 2026-08-14

Build/CI. No code change.

- Add `make generate` (`go generate ./...`) plus path-scoped `make generate-api`
  and `make generate-repos`, and turn on the pipeline's codegen-drift gate
  (`has_codegen: true`). CI now fails if a committed `*.gen.go` — the
  oapi-codegen HTTP server or the gorm repositories — is stale versus its source
  (`api/api.yml` / the models). Both generators are idempotent, so a clean tree
  stays clean.

## v0.2.1 — 2026-08-14

- Replace the hand-rolled bearer-auth middleware with
  `aichteeteapee/serbewr/middleware.BearerAuth` (added upstream in aichteeteapee
  v1.12.0). Behavior is unchanged: with `GITRAKZ_AUTH_TOKEN` set, `/api/v1`
  requires `Authorization: Bearer <token>` (constant-time compared) and answers
  the JSON error envelope on failure.

## v0.2.0 — 2026-08-14

- **Breaking (API):** REST endpoints are now versioned under `/api/v1` (e.g.
  `GET /api/v1/owners`). The version prefix moved into the OpenAPI `servers:`
  block — the paths there are version-less, and the generated handler applies
  the prefix via `BaseURL`, so it lives in exactly one place. Point any client
  at the `/api/v1` base; the embedded SPA is already updated.
- **Removed** the unimplemented `text/event-stream` (SSE) response from
  `POST /api/v1/run`. A run always returns the whole typed-block document as a
  single JSON response; any LLM-backed prose block is computed server-side (and
  cached by prompt/config version), then included in that document.

## v0.1.0 — 2026-08-14

Initial release.

- **Sync** — pulls a GitHub user's activity (commits / PRs / reviews / issues /
  releases) via the `gh` CLI into a local SQLite database, incrementally and
  per-repo fail-soft, on a background ticker plus an on-demand `POST /api/sync`.
- **Timeline + sessions** — a filterable, paginated (`hasMore`) event timeline
  grouped by day, and a derived sessions view that splits activity into work
  sessions on the configured idle gap with a lead-in for pre-first-commit work.
- **Templates** — programmatic `{ form, transform, layout, exports }`
  compositions run over a filtered range into a typed block document. Ships
  three built-in templates (activity summary, commits per repo, work-sessions
  timesheet), seeded on boot; custom templates are CRUD-managed, with built-ins
  clone-on-edit, and can be LLM-composed from a description.
- **Transforms + display blocks** — eight deterministic transform primitives
  plus an LLM-backed `describe-work` step cached in the DB by prompt/config
  version; eight typed display blocks with one renderer each.
- **Exports** — CSV, PDF and JSON of any document or template run.
- **Embedded Svelte SPA** — timeline, filter sidebar, range-select + template
  runner, templates manager, sessions view, and shareable deep-link runs
  (`/?tpl=<id>&run=1`), compiled into the binary via `go:embed`.
- **HTTP API** — a spec-first (`api/api.yml`) REST surface under `/api`, mounted
  on `aichteeteapee`/`serbewr`, optionally behind a bearer `GITRAKZ_AUTH_TOKEN`.
- Single static Go binary (pure-Go SQLite, embedded migrations) or a multi-arch
  Docker image.
