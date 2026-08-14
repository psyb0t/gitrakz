// Named constants for every endpoint path, block-type string, and default
// used across the SPA — per the "no magic literals" rule, nothing here gets
// hand-typed at a call site.

// --- API endpoint paths (exact paths from api/api.yml) ---
export const API_OWNERS = "/api/owners";
export const API_REPOS = "/api/repos";
export const API_TIMELINE = "/api/timeline";
export const API_SESSIONS = "/api/sessions";
export const API_SYNC = "/api/sync";
export const API_SYNC_STATUS = "/api/sync/status";
export const API_TEMPLATES = "/api/templates";
export const API_TEMPLATE_GENERATE = "/api/templates/generate";
export const API_RUN = "/api/run";
export const API_EXPORT = "/api/export";

export function apiTemplateByID(id: string): string {
  return `${API_TEMPLATES}/${encodeURIComponent(id)}`;
}

// --- Query param names (api/api.yml uses snake_case for page/per_page,
// everything else is a bare lowercase word — NOT camelCase; only response
// BODIES are tagliatelle camelCase) ---
export const QUERY_PARAM_OWNER = "owner";
export const QUERY_PARAM_REPO = "repo";
export const QUERY_PARAM_TYPE = "type";
export const QUERY_PARAM_FROM = "from";
export const QUERY_PARAM_TO = "to";
export const QUERY_PARAM_PAGE = "page";
export const QUERY_PARAM_PER_PAGE = "per_page";
export const QUERY_PARAM_GAP = "gap";

// --- Pagination defaults (mirrors PerPageQueryParam in api/api.yml) ---
export const DEFAULT_PAGE = 1;
export const DEFAULT_PER_PAGE = 50;
export const MAX_PER_PAGE = 200;

// --- Display block type strings (Block.type — matches
// internal/pkg/common/blocks.BlockType exactly) ---
export const BLOCK_TYPE_HEADING = "heading";
export const BLOCK_TYPE_TEXT = "text";
export const BLOCK_TYPE_LIST = "list";
export const BLOCK_TYPE_TABLE = "table";
export const BLOCK_TYPE_KEYVALUE = "keyvalue";
export const BLOCK_TYPE_METRIC = "metric";
export const BLOCK_TYPE_CODE = "code";
export const BLOCK_TYPE_CHART = "chart";

export const BLOCK_TYPES = [
  BLOCK_TYPE_HEADING,
  BLOCK_TYPE_TEXT,
  BLOCK_TYPE_LIST,
  BLOCK_TYPE_TABLE,
  BLOCK_TYPE_KEYVALUE,
  BLOCK_TYPE_METRIC,
  BLOCK_TYPE_CODE,
  BLOCK_TYPE_CHART,
] as const;

// --- Transform primitive names (internal/pkg/engine's fixed library, per
// TransformStep.primitive's doc comment in api/api.yml) ---
export const TRANSFORM_PRIMITIVES = [
  "sessionize",
  "exclude-off-time",
  "split-by-active-days",
  "aggregate",
  "group-by",
  "rate",
  "describe-work",
  "passthrough",
] as const;

// --- Form field kinds a template's `form` can declare ---
export const FORM_FIELD_TYPES = [
  "string",
  "number",
  "boolean",
  "date",
  "select",
] as const;

// --- HTTP / correlation ---
export const REQUEST_ID_HEADER = "X-Request-Id";
export const CONTENT_TYPE_JSON = "application/json";

// --- Sessionization default gap, matches GITRAKZ_SESSION_GAP's default
// mentioned in .git-trakz.md (30 minutes, in seconds) — only used as the
// pre-filled value for the sessions view's gap input, the server owns the
// real default when this is left unset. ---
export const DEFAULT_SESSION_GAP_SECONDS = 30 * 60;

// --- Misc UI ---
export const TOAST_DEFAULT_DURATION_MS = 6000;
