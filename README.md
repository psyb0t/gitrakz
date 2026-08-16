# gitrakz

[![CI](https://github.com/psyb0t/gitrakz/actions/workflows/pipeline.yml/badge.svg?branch=main)](https://github.com/psyb0t/gitrakz/actions/workflows/pipeline.yml)
[![coverage](https://raw.githubusercontent.com/psyb0t/gitrakz/badges/coverage.svg)](https://github.com/psyb0t/gitrakz/actions/workflows/pipeline.yml)
[![version](https://raw.githubusercontent.com/psyb0t/gitrakz/badges/version.svg)](https://github.com/psyb0t/gitrakz/releases)
[![license](https://raw.githubusercontent.com/psyb0t/gitrakz/badges/license.svg)](LICENSE)
[![Docker Pulls](https://img.shields.io/docker/pulls/psyb0t/gitrakz?style=flat-square)](https://hub.docker.com/r/psyb0t/gitrakz)

Self-hosted GitHub activity tracker: it syncs your `gh` activity into a local SQLite database, renders a filterable timeline and derived work sessions, and runs **programmatic templates** — deterministic transform pipelines composed onto typed display blocks — that export to CSV, PDF, or JSON. A single Go binary with the Svelte UI embedded in it; no external services required.

![Timeline](docs/screenshots/timeline.png)

A template is not a saved prompt — it is a saved **composition**: a form, a transform pipeline over the timeline, and a display layout. The pipeline is deterministic; the LLM (optional) only *composes* a template from the building blocks or narrates a single prose block. Everything you see is one of a fixed set of renderers.

![A template run](docs/screenshots/template-run.png)

## Contents

- [Features](#features)
- [How it works](#how-it-works)
- [Quickstart](#quickstart)
- [Configuration](#configuration)
- [Building blocks and templates](#building-blocks-and-templates)
- [HTTP API](#http-api)
- [MCP](#mcp)
- [Agent integrations](#agent-integrations)
- [Development](#development)
- [Security notes](#security-notes)
- [License](#license)

## Features

- **Timeline** — every commit / PR / review / issue / release for a GitHub user, grouped by day, with owner/repo, title, `+additions/-deletions`, time and link. Filter by owner, repo, type and date range; paginated with `hasMore` (never a `total` count).
- **Sessions** — derived, not stored: events are grouped per owner and split into work sessions whenever the idle gap exceeds `GITRAKZ_SESSION_GAP`, with a configurable lead-in so pre-first-commit work is counted. The grounding for timesheet-style reports.
- **Templates** — built-in and custom. A template = `{ form, transform, layout, exports }`. Run one over a filtered range and get a typed block document you can export. Built-ins ship ready to run; custom ones are created, edited (built-ins clone-on-edit) or **LLM-composed** from a plain-language description.
- **Deterministic transforms** — a fixed library of primitives (`sessionize`, `exclude-off-time`, `split-by-active-days`, `group-by`, `aggregate`, `rate`, `passthrough`) plus one LLM-backed step (`describe-work`) whose output is cached in the DB by prompt/config version so a re-run costs nothing.
- **Typed display blocks** — `heading`, `text`, `table`, `metric`, `keyvalue`, `list`, `code`, `chart`. One renderer per block, in the SPA and in every exporter.
- **Exports** — CSV, JSON and PDF of any document or template run.
- **Embedded Svelte SPA** — the UI is compiled into the binary. Shareable deep-link runs: `/?tpl=<id>&run=1`.
- **Single static binary or a small Docker image** — pure-Go SQLite (no cgo), migrations embedded, sync on a background ticker.
- **LLM-optional** — `describe-work`, `text` prose blocks and "generate a template with AI" go through [`elelem`](https://github.com/psyb0t/elelem), selecting an `openai` (or OpenAI-compatible) or `anthropic` provider by env. Leave the provider config empty and everything deterministic still works.

## How it works

```
gh CLI ──sync──▶ SQLite (events) ──▶ timeline / sessions
                                 └──▶ template = form + transform pipeline + layout
                                              └──▶ typed block document ──▶ SPA / CSV / PDF / JSON
```

`gitrakz` shells out to the GitHub `gh` CLI to discover a user's repos and pull their activity incrementally — per-repo and fail-soft, so one rate-limited repo never aborts the rest — persisting events into SQLite. A run resolves a template, queries the filtered timeline, runs the transform pipeline deterministically, maps the result onto the display layout, and returns the block document.

## Quickstart

### Install (recommended)

Per-user (no root) — installs into your home, just for you:

```bash
curl -fsSL https://raw.githubusercontent.com/psyb0t/gitrakz/main/install.sh | bash
```

That drops the `gitrakz` command into `~/.local/bin` and the config into
`~/.config/gitrakz`. If `~/.local/bin` is not on your `PATH`, the installer prints the
exact one-liner to add it (for both bash and zsh).

System-wide — one shared stack any user in the `docker` group can drive:

```bash
curl -fsSL https://raw.githubusercontent.com/psyb0t/gitrakz/main/install.sh | sudo bash
```

That puts the command in `/usr/local/bin` and the config in `/etc/gitrakz`
(root-owned, readable by the `docker` group). Running as root also installs the
GitHub CLI for you if it is missing; the per-user install expects `gh` to be
present already.

Either way it pins the local stack to the latest release tag (never `:latest` on
your machine) and drops `docker-compose.yml`, a visible `.env.example`, and an
owner-only `.env` copied from that example on first install. Later installs and
upgrades refresh `.env.example` but leave your `.env` alone, apart from its image
pin. The mode is chosen from who runs it — root gives system-wide, otherwise
per-user — and you can force it with `--user` or `--system`. No source checkout
required.

Authenticate GitHub once — the wrapper reads a token from it — then start it. It
tracks *your own* activity by default:

```bash
gh auth login
gitrakz start
```

Open <http://127.0.0.1:8080>. The rest is deliberately boring:

```bash
gitrakz setup            # refresh compose + .env.example; creates .env only if missing
gitrakz status
gitrakz logs -f
gitrakz stop
gitrakz upgrade          # back up data, re-pin to the latest release, then pull it
gitrakz upgrade --rolling # test the moving :latest built from main, just this once
gitrakz restore ~/.config/gitrakz/backups/<timestamp>.tar.gz
gitrakz uninstall        # remove the command; asks before deleting your data
```

Configuration lives in `~/.config/gitrakz/.env` (or `/etc/gitrakz/.env` for a
system-wide install); its current reference is always beside it as
`.env.example`. SQLite state stays in Docker's named `gitrakz-data`
volume. Every `gitrakz upgrade` writes a snapshot named
`YYYYMMDDHHMMSS.tar.gz` under the install directory's `backups/` folder and
keeps the newest three. `gitrakz restore <backup.tar.gz>` validates the archive,
asks before replacing the volume, snapshots its current contents, and leaves the
stack stopped for an explicit `gitrakz start`. **Edit `.env` right after install** to
change the published port (`GITRAKZ_PUBLISH_PORT`), expose it beyond localhost
(`GITRAKZ_PUBLISH_ADDR`), set an API bearer token (`GITRAKZ_AUTH_TOKEN`), track a
different user (`GITRAKZ_GH_USER`), or enable the optional LLM features
(`GITRAKZ_ELELEM_*`). `gitrakz start` picks the changes up. The GitHub token is
never written to `.env`; the wrapper injects it at runtime from `gh auth token`.

### Run it with Docker directly

The wrapper is only a guardrail around Docker. To drive it yourself, pass the
token straight through — that is the only auth gitrakz needs (it runs `gh` inside
the container with it):

```bash
docker run --rm -p 8080:8080 \
  -e GH_TOKEN="$(gh auth token)" \
  -v gitrakz-data:/data \
  psyb0t/gitrakz:v0.7.2 run          # pin a release tag, not :latest
```

Add any `-e GITRAKZ_*` from the [Configuration](#configuration) table. Or run
the same pinned stack straight from the compose file the installer wrote:

```bash
export GH_TOKEN="$(gh auth token)"
docker compose --project-directory ~/.config/gitrakz \
  --env-file ~/.config/gitrakz/.env -f ~/.config/gitrakz/docker-compose.yml up -d
```

### From source

```bash
git clone https://github.com/psyb0t/gitrakz
cd gitrakz
make build                          # static binary in ./build (via Docker)
GH_TOKEN="$(gh auth token)" ./build/gitrakz run
```

Local dev without Docker:

```bash
cd web && pnpm install && pnpm build && cd ..   # build the embedded SPA
GH_TOKEN="$(gh auth token)" go run ./cmd run
```

`gh` must be installed and authenticated (`gh auth login`) wherever the binary
runs — gitrakz shells out to it, using `GH_TOKEN` for auth.

## Configuration

All configuration is environment variables, prefixed `GITRAKZ_`. Everything has a sane default — with authenticated `gh`, gitrakz runs with no config at all.

| Variable | Default | What it does |
|---|---|---|
| `GITRAKZ_GH_USER` | *(gh login)* | The GitHub user whose activity is tracked. Defaults to the `gh` CLI's authenticated login, so leave it unset to track yourself; set it to track another user. |
| `GITRAKZ_AUTH_TOKEN` | *(empty)* | When set, `/api` requires `Authorization: Bearer <token>`. Empty = open (single-user / trusted network). |
| `GITRAKZ_SYNC_SINCE` | `2025-01-01` | Earliest activity to pull on a first sync. |
| `GITRAKZ_SYNC_INTERVAL` | `30m` | Background incremental-sync cadence. |
| `GITRAKZ_SESSION_GAP` | `30m` | Idle gap that starts a new work session. |
| `GITRAKZ_SESSION_LEADIN` | `25m` | Padding added before a session's first event (pre-commit work). |
| `GITRAKZ_ELELEM_TYPE` | `openai` | LLM provider driver — `openai` (also any OpenAI-compatible endpoint) or `anthropic`. |
| `GITRAKZ_ELELEM_BASE_URL` | *(empty)* | API host for the provider, used by `describe-work`, prose blocks and template generation. |
| `GITRAKZ_ELELEM_API_KEY` | *(empty)* | API key for the LLM endpoint. |

The model, reasoning effort, and temperature are configured in the web UI's
Settings page (chosen from the provider's available models) and stored in the
database.

Logging follows the standard `LOG_LEVEL` / `LOG_FORMAT` / `LOG_ADD_SOURCE` env vars. Every request is logged with a `requestId` that the SPA also echoes to the browser console, so a full request reconstructs across both sides.

## Building blocks and templates

A template is programmatic and has three parts, composed from two fixed libraries:

```
template = { id, name, description,
             form,       // input fields a run collects (rate, lead-in, off-hours…)
             transform,  // ordered pipeline of transform primitives (deterministic)
             layout,     // display-block composition rendering the transform output
             exports }   // [csv, pdf, json]
```

**Transform primitives** (compute over the timeline): `sessionize`, `exclude-off-time`, `split-by-active-days`, `group-by`, `aggregate`, `rate`, `passthrough`, the LLM-backed `describe-work` (cached in the DB by prompt/config version), and `llm` — a user-authored LLM step that runs a caller-supplied instruction over the pipeline's current data and writes the response as a row, optionally constrained to a caller-supplied JSON schema for structured output.

**Display blocks** (render the result): `heading`, `text`, `list`, `table`, `keyvalue`, `metric`, `code`, `chart`.

Ships with built-in templates — *Activity summary*, *Commits per repo*, and a *Work sessions timesheet* — and "Generate with AI" turns a description into a draft template you review and save. No raw HTML or code is ever authored by hand.

## HTTP API

Everything the SPA does is a REST call under `/api/v1` (JSON, camelCase), behind `GITRAKZ_AUTH_TOKEN` when set. The `/api/v1` prefix lives in the spec's `servers:` block; the paths are version-less there.

```
GET  /                           # the embedded Svelte SPA
GET  /api/v1/owners              # distinct owners
GET  /api/v1/repos?owner=        # repos under an owner
GET  /api/v1/timeline?owner=&repo=&type=&from=&to=&page=&perPage=
GET  /api/v1/sessions?…          # sessionized view + heuristic hours
POST /api/v1/sync                # trigger an incremental sync
GET  /api/v1/sync/status         # last sync status
GET  /api/v1/templates           # list (built-in + custom)
POST /api/v1/templates           # create a custom template
PUT  /api/v1/templates/{id}      # edit (built-ins clone-on-edit)
DELETE /api/v1/templates/{id}    # delete a custom template
POST /api/v1/templates/generate  # LLM-compose a template draft from a description
POST /api/v1/run                 # run a template over a filter -> block document (JSON)
POST /api/v1/export              # export a document / run to csv|pdf|json
```

The OpenAPI spec at `api/api.yml` is the source of truth; the server interface and types are generated from it.

## MCP

Every capability above is also exposed as an MCP (Model Context Protocol) tool, so an MCP-speaking client (Claude Code, etc.) can drive gitrakz directly instead of making raw REST calls. Both transports serve the same tools against the same running instance's SQLite data:

- **Streamable HTTP**, mounted at `/mcp` on the running instance alongside `/api/v1` — Bearer-gated the same way (`Authorization: Bearer <token>`) when `GITRAKZ_AUTH_TOKEN` is set.
- **stdio**, via the binary's `gitrakz mcp` subcommand — opens the same SQLite database directly (no running HTTP server needed) and speaks MCP over stdin/stdout. Run it through Docker for a containerized install: `docker run --rm -i -e GH_TOKEN -v gitrakz-data:/data psyb0t/gitrakz:vX.Y.Z mcp`.

```
gitrakz_list_owners      # every owner with ingested activity
gitrakz_list_repos       # repos under owner=
gitrakz_list_templates   # every saved template (built-in + custom)
gitrakz_get_template     # one template by id=
gitrakz_run_template     # run templateId= over an optional filter -> block document
gitrakz_trigger_sync     # trigger one incremental gh sync
gitrakz_get_sync_status  # current sync status
gitrakz_list_sessions    # derived work sessions over an optional filter
gitrakz_query_timeline   # one page of the filtered, newest-first timeline
```

Every tool wraps the same service layer the REST handlers use — same request/response shapes, same sync/LLM cost caveats. See [.agents/skills/gitrakz/SKILL.md](.agents/skills/gitrakz/SKILL.md) for the full MCP client config examples (both transports).

## Agent integrations

This repo ships a documentation skill for agents that drive a gitrakz instance:
setup (installer or Docker), the `/api/v1` REST surface and the `/mcp` MCP
surface, and the `GH_TOKEN` auth model.

### Claude Code

```bash
claude plugin marketplace add psyb0t/agents
claude plugin install gitrakz@psyb0t
```

Claude Code asks for the gitrakz URL (and an optional bearer token) when the
plugin is enabled; the token is stored as sensitive user configuration.

### Codex

```bash
codex plugin marketplace add psyb0t/agents
codex plugin add gitrakz@psyb0t
```

Inside this repository, use `$gitrakz`. After marketplace installation, use
`$gitrakz:gitrakz`.

### OpenClaw

The same skill is published to ClawHub on tagged releases:

```bash
openclaw skills install @psyb0t/gitrakz
```

The detailed setup reference is
[.agents/skills/gitrakz/references/setup.md](.agents/skills/gitrakz/references/setup.md).

## Development

```bash
make lint            # golangci-lint (80+ linters) + go fix diff
make test            # go test -race ./...
make test-coverage   # 90% gate (excludes generated code, /cmd and services)
make generate        # regenerate the OpenAPI server + gorm repos (go generate ./...)
make build           # static binary via Docker
make run-dev         # run in the dev container
```

Stack: Go on [`servicepack`](https://github.com/psyb0t/servicepack) · HTTP via [`aichteeteapee`](https://github.com/psyb0t/aichteeteapee) `serbewr` + `oapi-codegen` strict server · SQLite via `gorm` + `gorm-gen` (schema owned by embedded SQL migrations, never AutoMigrate) · LLM via [`elelem`](https://github.com/psyb0t/elelem) · `gh` shell-out via [`commander`](https://github.com/psyb0t/commander) · Svelte 5 + Vite SPA embedded with `go:embed`.

Project layout follows [golang-standards/project-layout](https://github.com/golang-standards/project-layout): `cmd/` entrypoints, `internal/pkg/http/{api,server}` (generated API + handlers), `internal/pkg/services/http-server` (the servicepack service), `internal/pkg/db/{models,repositories,migrations}`, `internal/pkg/transform/*` and `internal/pkg/common/*`, `web/` for the SPA.

See [CHANGELOG.md](CHANGELOG.md) for release notes.

## Security notes

- `gitrakz` runs the `gh` CLI with whatever credentials it's given — mount a read-scoped token; it only reads activity.
- Set `GITRAKZ_AUTH_TOKEN` for anything beyond a trusted single-user network; without it the API is open.
- Point `GITRAKZ_ELELEM_BASE_URL` only at endpoints you control or trust — `describe-work` and template generation send commit titles / diffs there.
- The SPA renders markdown without raw-HTML injection; template output is typed blocks, never author-supplied HTML.

## License

MIT — see [LICENSE](LICENSE).

---

*Built with spite using [servicepack](https://github.com/psyb0t/servicepack).*
