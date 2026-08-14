# Changelog

All notable changes per release. Versions follow [semver](https://semver.org)
pre-1.0 conventions: minor bumps may include breaking REST changes (called out
explicitly), patch bumps are docs / build / fixes only.

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
