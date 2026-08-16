package mcp

// serverName / serverVersion identify gitrakz to MCP clients in the
// initialize handshake.
const (
	serverName    = "gitrakz"
	serverVersion = "0.1.0"
)

// Tool names. Every tool is prefixed "gitrakz_" so a client juggling
// several MCP servers can tell gitrakz's tools apart at a glance.
const (
	toolNameListOwners    = "gitrakz_list_owners"
	toolNameListRepos     = "gitrakz_list_repos"
	toolNameListTemplates = "gitrakz_list_templates"
	toolNameGetTemplate   = "gitrakz_get_template"
	toolNameRunTemplate   = "gitrakz_run_template"
	toolNameTriggerSync   = "gitrakz_trigger_sync"
	toolNameGetSyncStatus = "gitrakz_get_sync_status"
	toolNameListSessions  = "gitrakz_list_sessions"
	toolNameQueryTimeline = "gitrakz_query_timeline"
)

// runQueryPerPage is the page size queryFullTimeline uses internally when a
// tool needs an entire filtered timeline instead of one paginated page
// (RunTemplate, ListSessions). Mirrors internal/pkg/http/server's
// identically-purposed constant.
const runQueryPerPage = 500
