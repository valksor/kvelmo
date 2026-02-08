package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/valksor/go-mehrhof/internal/server/static"
	"github.com/valksor/go-toolkit/eventbus"
)

// setupRouter creates and configures the HTTP router.
func (s *Server) setupRouter() http.Handler {
	mux := http.NewServeMux()

	// Serve static assets - skip in API-only mode
	if !s.config.APIOnly {
		// Fonts and licenses
		staticFS := http.FileServer(http.FS(static.Public()))
		mux.Handle("GET /fonts/", staticFS)
		mux.Handle("GET /licenses.json", staticFS)

		// React SPA assets - files are at assets/ inside the embed
		reactFS := http.FileServer(http.FS(static.ReactApp()))
		mux.Handle("GET /assets/", reactFS)
	}

	// Health check (public)
	mux.HandleFunc("GET /health", s.handleHealth)

	mux.HandleFunc("GET /api/v1/auth/csrf", s.handleCSRFToken)

	// API routes
	mux.HandleFunc("GET /api/v1/status", s.handleViaRouter(CommandRoute{
		Command:           "server-status",
		InjectFn:          s.injectServerStatus(),
		UnwrapData:        true,
		AllowNilConductor: true,
	}))
	mux.HandleFunc("GET /api/v1/context", s.handleViaRouter(CommandRoute{
		Command:           "server-context",
		InjectFn:          s.injectServerContext(),
		UnwrapData:        true,
		AllowNilConductor: true,
	}))
	mux.HandleFunc("GET /api/v1/docs-url", s.handleViaRouter(CommandRoute{
		Command:           "docs-url",
		UnwrapData:        true,
		AllowNilConductor: true,
	}))

	// License routes
	mux.HandleFunc("GET /api/v1/license", s.handleViaRouter(CommandRoute{
		Command:           "license",
		UnwrapData:        true,
		AllowNilConductor: true,
	}))
	mux.HandleFunc("GET /api/v1/license/info", s.handleViaRouter(CommandRoute{
		Command:           "license-info",
		UnwrapData:        true,
		AllowNilConductor: true,
	}))

	// Project mode routes
	if s.config.Mode == ModeProject {
		// Task endpoints
		mux.HandleFunc("GET /api/v1/task", s.handleViaRouter(CommandRoute{
			Command: "task",
		}))
		mux.HandleFunc("GET /api/v1/tasks", s.handleViaRouter(CommandRoute{
			Command: "tasks",
		}))
		mux.HandleFunc("GET /api/v1/tasks/{id}/specs", s.handleViaRouter(CommandRoute{
			Command:    "specifications",
			ParseFn:    parseTaskIDInvocation,
			UnwrapData: true,
		}))
		mux.HandleFunc("GET /api/v1/tasks/{id}/specs/{number}/diff", s.handleViaRouter(CommandRoute{
			Command:    "specification-diff",
			ParseFn:    parseSpecificationDiffInvocation,
			UnwrapData: true,
		}))
		mux.HandleFunc("GET /api/v1/tasks/{id}/sessions", s.handleViaRouter(CommandRoute{
			Command:    "sessions",
			ParseFn:    parseTaskIDInvocation,
			UnwrapData: true,
		}))

		// Work data by ID (active or completed tasks)
		mux.HandleFunc("GET /api/v1/work/{id}", s.handleViaRouter(CommandRoute{
			Command:    "work",
			ParseFn:    parseWorkByIDInvocation,
			UnwrapData: true,
		}))

		// Workflow action endpoints
		mux.HandleFunc("POST /api/v1/workflow/start", s.handleViaRouter(CommandRoute{
			Command: "start",
			ParseFn: s.parseStartInvocation,
		}))
		mux.HandleFunc("POST /api/v1/workflow/plan", s.handleViaRouter(CommandRoute{
			Command: "plan",
		}))
		mux.HandleFunc("POST /api/v1/workflow/implement", s.handleViaRouter(CommandRoute{
			Command: "implement",
			ParseFn: parseImplementInvocation,
		}))
		mux.HandleFunc("POST /api/v1/workflow/implement/review/{n}", s.handleViaRouter(CommandRoute{
			Command: "implement",
			ParseFn: parseImplementReviewInvocation,
		}))
		mux.HandleFunc("POST /api/v1/workflow/review", s.handleViaRouter(CommandRoute{
			Command: "review",
		}))
		mux.HandleFunc("POST /api/v1/workflow/finish", s.handleViaRouter(CommandRoute{
			Command: "finish",
			ParseFn: parseFinishInvocation,
		}))
		mux.HandleFunc("POST /api/v1/workflow/undo", s.handleViaRouter(CommandRoute{
			Command: "undo",
		}))
		mux.HandleFunc("POST /api/v1/workflow/redo", s.handleViaRouter(CommandRoute{
			Command: "redo",
		}))
		mux.HandleFunc("POST /api/v1/workflow/answer", s.handleViaRouter(CommandRoute{
			Command: "answer",
			ParseFn: parseAnswerInvocation,
		}))
		mux.HandleFunc("POST /api/v1/workflow/resume", s.handleViaRouter(CommandRoute{
			Command: "continue",
		}))
		mux.HandleFunc("POST /api/v1/workflow/abandon", s.handleViaRouter(CommandRoute{
			Command: "abandon",
		}))
		mux.HandleFunc("POST /api/v1/workflow/reset", s.handleViaRouter(CommandRoute{
			Command: "reset",
		}))
		mux.HandleFunc("POST /api/v1/workflow/continue", s.handleViaRouter(CommandRoute{
			Command: "continue",
			ParseFn: parseContinueInvocation,
		}))
		mux.HandleFunc("POST /api/v1/workflow/auto", s.handleViaRouter(CommandRoute{
			Command: "auto",
			ParseFn: parseAutoInvocation,
		}))
		mux.HandleFunc("POST /api/v1/workflow/question", s.handleWorkflowQuestion)

		// Notes endpoints
		mux.HandleFunc("POST /api/v1/tasks/{id}/notes", s.handleViaRouter(CommandRoute{
			Command: "note",
			ParseFn: parseNoteInvocation,
		}))
		mux.HandleFunc("GET /api/v1/tasks/{id}/notes", s.handleViaRouter(CommandRoute{
			Command: "notes",
			ParseFn: parseNotesInvocation,
		}))

		// Labels endpoints
		mux.HandleFunc("GET /api/v1/task/labels", s.handleViaRouter(CommandRoute{
			Command: "label",
			ParseFn: parseLabelsGetInvocation,
		}))
		mux.HandleFunc("POST /api/v1/task/labels", s.handleViaRouter(CommandRoute{
			Command: "label",
			ParseFn: parseLabelsPostInvocation,
		}))
		mux.HandleFunc("GET /api/v1/labels", s.handleViaRouter(CommandRoute{
			Command: "labels",
		}))

		// Hierarchy endpoint
		mux.HandleFunc("GET /api/v1/task/hierarchy", s.handleViaRouter(CommandRoute{
			Command:    "hierarchy",
			UnwrapData: true,
		}))

		// Cost tracking endpoints
		mux.HandleFunc("GET /api/v1/tasks/{id}/costs", s.handleViaRouter(CommandRoute{
			Command: "costs",
			ParseFn: parseTaskIDInvocation,
		}))
		mux.HandleFunc("GET /api/v1/costs", s.handleViaRouter(CommandRoute{
			Command: "costs",
			ParseFn: parseAggregateCostsInvocation,
		}))

		// Guide endpoint
		mux.HandleFunc("GET /api/v1/guide", s.handleViaRouter(CommandRoute{
			Command:    "guide",
			UnwrapData: true,
		}))

		// Info endpoints
		mux.HandleFunc("GET /api/v1/agents", s.handleViaRouter(CommandRoute{
			Command:    "agents",
			UnwrapData: true,
		}))
		mux.HandleFunc("GET /api/v1/providers", s.handleViaRouter(CommandRoute{
			Command:           "providers",
			UnwrapData:        true,
			AllowNilConductor: true,
		}))
		mux.HandleFunc("GET /api/v1/command/status", s.handleViaRouter(CommandRoute{
			Command: "status",
		}))
		mux.HandleFunc("GET /api/v1/command/cost", s.handleViaRouter(CommandRoute{
			Command: "cost",
		}))
		mux.HandleFunc("GET /api/v1/command/budget", s.handleViaRouter(CommandRoute{
			Command: "budget",
		}))
		mux.HandleFunc("GET /api/v1/command/list", s.handleViaRouter(CommandRoute{
			Command: "list",
		}))
		mux.HandleFunc("GET /api/v1/command/specification", s.handleViaRouter(CommandRoute{
			Command: "specification",
			ParseFn: parseSpecificationInvocation,
		}))

		// Agent Alias endpoints
		mux.HandleFunc("GET /api/v1/agents/aliases", s.handleViaRouter(CommandRoute{
			Command:    "agent-alias",
			ParseFn:    parseAgentAliasListInvocation,
			UnwrapData: true,
		}))
		mux.HandleFunc("POST /api/v1/agents/aliases", s.handleViaRouter(CommandRoute{
			Command: "agent-alias",
			ParseFn: parseAgentAliasAddInvocation,
		}))
		mux.HandleFunc("DELETE /api/v1/agents/aliases/", s.handleViaRouter(CommandRoute{
			Command: "agent-alias",
			ParseFn: parseAgentAliasDeleteInvocation,
		}))

		// Browser automation endpoints
		mux.HandleFunc("GET /api/v1/browser/status", s.handleViaRouter(CommandRoute{
			Command:    "browser",
			ParseFn:    parseBrowserGetSubcommand("status"),
			UnwrapData: true,
		}))
		mux.HandleFunc("GET /api/v1/browser/tabs", s.handleViaRouter(CommandRoute{
			Command:    "browser",
			ParseFn:    parseBrowserGetSubcommand("tabs"),
			UnwrapData: true,
		}))
		mux.HandleFunc("POST /api/v1/browser/goto", s.handleViaRouter(CommandRoute{
			Command:    "browser",
			ParseFn:    parseBrowserSubcommand("goto"),
			UnwrapData: true,
		}))
		mux.HandleFunc("POST /api/v1/browser/navigate", s.handleViaRouter(CommandRoute{
			Command:    "browser",
			ParseFn:    parseBrowserSubcommand("navigate"),
			UnwrapData: true,
		}))
		mux.HandleFunc("POST /api/v1/browser/switch", s.handleViaRouter(CommandRoute{
			Command:    "browser",
			ParseFn:    parseBrowserSubcommand("switch"),
			UnwrapData: true,
		}))
		mux.HandleFunc("POST /api/v1/browser/screenshot", s.handleViaRouter(CommandRoute{
			Command:    "browser",
			ParseFn:    parseBrowserSubcommand("screenshot"),
			UnwrapData: true,
		}))
		mux.HandleFunc("POST /api/v1/browser/click", s.handleViaRouter(CommandRoute{
			Command:    "browser",
			ParseFn:    parseBrowserSubcommand("click"),
			UnwrapData: true,
		}))
		mux.HandleFunc("POST /api/v1/browser/type", s.handleViaRouter(CommandRoute{
			Command:    "browser",
			ParseFn:    parseBrowserSubcommand("type"),
			UnwrapData: true,
		}))
		mux.HandleFunc("POST /api/v1/browser/eval", s.handleViaRouter(CommandRoute{
			Command:    "browser",
			ParseFn:    parseBrowserSubcommand("eval"),
			UnwrapData: true,
		}))
		mux.HandleFunc("POST /api/v1/browser/dom", s.handleViaRouter(CommandRoute{
			Command:    "browser",
			ParseFn:    parseBrowserSubcommand("dom"),
			UnwrapData: true,
		}))
		mux.HandleFunc("POST /api/v1/browser/reload", s.handleViaRouter(CommandRoute{
			Command:    "browser",
			ParseFn:    parseBrowserSubcommand("reload"),
			UnwrapData: true,
		}))
		mux.HandleFunc("POST /api/v1/browser/close", s.handleViaRouter(CommandRoute{
			Command:    "browser",
			ParseFn:    parseBrowserSubcommand("close"),
			UnwrapData: true,
		}))
		mux.HandleFunc("GET /api/v1/browser/cookies", s.handleViaRouter(CommandRoute{
			Command:    "browser",
			ParseFn:    parseBrowserCookiesGetInvocation,
			UnwrapData: true,
		}))
		mux.HandleFunc("POST /api/v1/browser/cookies", s.handleViaRouter(CommandRoute{
			Command:    "browser",
			ParseFn:    parseBrowserSubcommand("cookies-set"),
			UnwrapData: true,
		}))

		// Browser DevTools endpoints
		mux.HandleFunc("POST /api/v1/browser/network", s.handleViaRouter(CommandRoute{
			Command:    "browser",
			ParseFn:    parseBrowserSubcommand("network"),
			UnwrapData: true,
		}))
		mux.HandleFunc("POST /api/v1/browser/console", s.handleViaRouter(CommandRoute{
			Command:    "browser",
			ParseFn:    parseBrowserSubcommand("console"),
			UnwrapData: true,
		}))
		mux.HandleFunc("POST /api/v1/browser/websocket", s.handleViaRouter(CommandRoute{
			Command:    "browser",
			ParseFn:    parseBrowserSubcommand("websocket"),
			UnwrapData: true,
		}))
		mux.HandleFunc("POST /api/v1/browser/source", s.handleViaRouter(CommandRoute{
			Command:    "browser",
			ParseFn:    parseBrowserSubcommand("source"),
			UnwrapData: true,
		}))
		mux.HandleFunc("POST /api/v1/browser/scripts", s.handleViaRouter(CommandRoute{
			Command:    "browser",
			ParseFn:    parseBrowserSubcommand("scripts"),
			UnwrapData: true,
		}))
		mux.HandleFunc("POST /api/v1/browser/styles", s.handleViaRouter(CommandRoute{
			Command:    "browser",
			ParseFn:    parseBrowserSubcommand("styles"),
			UnwrapData: true,
		}))
		mux.HandleFunc("POST /api/v1/browser/coverage", s.handleViaRouter(CommandRoute{
			Command:    "browser",
			ParseFn:    parseBrowserSubcommand("coverage"),
			UnwrapData: true,
		}))

		// Security scan endpoint
		mux.HandleFunc("POST /api/v1/scan", s.handleViaRouter(CommandRoute{
			Command:    "security-scan",
			ParseFn:    parseSecurityScanInvocation,
			UnwrapData: true,
		}))

		// Memory endpoints
		mux.HandleFunc("GET /api/v1/memory/search", s.handleViaRouter(CommandRoute{
			Command:    "memory",
			ParseFn:    parseMemorySearchInvocation,
			UnwrapData: true,
		}))
		mux.HandleFunc("POST /api/v1/memory/index", s.handleViaRouter(CommandRoute{
			Command:    "memory",
			ParseFn:    parseMemoryIndexInvocation,
			UnwrapData: true,
		}))
		mux.HandleFunc("GET /api/v1/memory/stats", s.handleViaRouter(CommandRoute{
			Command:    "memory",
			ParseFn:    parseMemoryStatsInvocation,
			UnwrapData: true,
		}))

		// Library endpoints
		mux.HandleFunc("GET /api/v1/library", s.handleViaRouter(CommandRoute{
			Command:    "library",
			ParseFn:    parseLibraryListInvocation,
			UnwrapData: true,
		}))
		mux.HandleFunc("GET /api/v1/library/stats", s.handleViaRouter(CommandRoute{
			Command:    "library",
			ParseFn:    parseLibraryStatsInvocation,
			UnwrapData: true,
		}))
		mux.HandleFunc("POST /api/v1/library/pull", s.handleViaRouter(CommandRoute{
			Command:    "library",
			ParseFn:    parseLibraryPullInvocation,
			UnwrapData: true,
		}))
		mux.HandleFunc("POST /api/v1/library/pull/preview", s.handleViaRouter(CommandRoute{
			Command:    "library",
			ParseFn:    parseLibraryPullPreviewInvocation,
			UnwrapData: true,
		}))
		mux.HandleFunc("GET /api/v1/library/{id}/items", s.handleViaRouter(CommandRoute{
			Command:    "library",
			ParseFn:    parseLibraryItemsInvocation,
			UnwrapData: true,
		}))
		mux.HandleFunc("GET /api/v1/library/", s.handleViaRouter(CommandRoute{
			Command:    "library",
			ParseFn:    parseLibraryShowInvocation,
			UnwrapData: true,
		}))
		mux.HandleFunc("DELETE /api/v1/library/", s.handleViaRouter(CommandRoute{
			Command:    "library",
			ParseFn:    parseLibraryRemoveInvocation,
			UnwrapData: true,
		}))

		// Links endpoints
		mux.HandleFunc("GET /api/v1/links", s.handleViaRouter(CommandRoute{
			Command:    "links",
			ParseFn:    parseLinksListInvocation,
			UnwrapData: true,
		}))
		mux.HandleFunc("GET /api/v1/links/search", s.handleViaRouter(CommandRoute{
			Command:    "links",
			ParseFn:    parseLinksSearchInvocation,
			UnwrapData: true,
		}))
		mux.HandleFunc("GET /api/v1/links/stats", s.handleViaRouter(CommandRoute{
			Command:    "links",
			ParseFn:    parseLinksStatsInvocation,
			UnwrapData: true,
		}))
		mux.HandleFunc("GET /api/v1/links/", s.handleViaRouter(CommandRoute{
			Command:    "links",
			ParseFn:    parseLinksEntityInvocation,
			UnwrapData: true,
		}))
		mux.HandleFunc("POST /api/v1/links/rebuild", s.handleViaRouter(CommandRoute{
			Command:    "links",
			ParseFn:    parseLinksRebuildInvocation,
			UnwrapData: true,
		}))

		// Find search endpoints (JSON mode via router, SSE streaming via thin handler)
		mux.HandleFunc("GET /api/v1/find", s.handleFindSearch)
		mux.HandleFunc("POST /api/v1/find", s.handleFindSearch)

		// Budget endpoints
		mux.HandleFunc("GET /api/v1/budget/monthly/status", s.handleBudgetMonthlyStatus)
		mux.HandleFunc("POST /api/v1/budget/monthly/reset", s.handleViaRouter(CommandRoute{
			Command: "budget-reset",
		}))

		// Sync and simplify endpoints
		mux.HandleFunc("POST /api/v1/workflow/sync", s.handleViaRouter(CommandRoute{
			Command: "sync",
			ParseFn: parseSyncInvocation,
		}))
		mux.HandleFunc("POST /api/v1/workflow/simplify", s.handleViaRouter(CommandRoute{
			Command: "simplify",
			ParseFn: parseSimplifyInvocation,
		}))

		// Standalone review/simplify endpoints (SSE+JSON split: SSE stays inline, JSON goes through router)
		mux.HandleFunc("POST /api/v1/workflow/review/standalone", s.handleStandaloneReviewDispatch)
		mux.HandleFunc("POST /api/v1/workflow/simplify/standalone", s.handleStandaloneSimplifyDispatch)

		// Templates endpoints
		mux.HandleFunc("GET /api/v1/templates", s.handleViaRouter(CommandRoute{
			Command:           "template",
			ParseFn:           parseTemplateListInvocation,
			UnwrapData:        true,
			AllowNilConductor: true,
		}))
		mux.HandleFunc("GET /api/v1/templates/{name}", s.handleViaRouter(CommandRoute{
			Command:           "template",
			ParseFn:           parseTemplateGetInvocation,
			UnwrapData:        true,
			AllowNilConductor: true,
		}))
		mux.HandleFunc("POST /api/v1/templates/apply", s.handleViaRouter(CommandRoute{
			Command:    "template",
			ParseFn:    parseTemplateApplyInvocation,
			UnwrapData: true,
		}))

		// Settings endpoints
		mux.HandleFunc("GET /api/v1/settings", s.handleViaRouter(CommandRoute{
			Command:    "settings-get",
			ParseFn:    parseSettingsGetInvocation,
			InjectFn:   s.injectSettingsMode(),
			UnwrapData: true,
		}))
		mux.HandleFunc("POST /api/v1/settings", s.handleSaveSettings)
		mux.HandleFunc("GET /api/v1/settings/explain", s.handleViaRouter(CommandRoute{
			Command:    "config-explain",
			ParseFn:    parseConfigExplainInvocation,
			UnwrapData: true,
		}))
		mux.HandleFunc("GET /api/v1/settings/provider-health", s.handleViaRouter(CommandRoute{
			Command:    "provider-health",
			UnwrapData: true,
		}))

		// Sandbox endpoints
		mux.HandleFunc("GET /api/v1/sandbox/status", s.handleViaRouter(CommandRoute{
			Command:           "sandbox-status",
			UnwrapData:        true,
			AllowNilConductor: true,
		}))
		mux.HandleFunc("POST /api/v1/sandbox/enable", s.handleViaRouter(CommandRoute{
			Command:    "sandbox-enable",
			UnwrapData: true,
		}))
		mux.HandleFunc("POST /api/v1/sandbox/disable", s.handleViaRouter(CommandRoute{
			Command:    "sandbox-disable",
			UnwrapData: true,
		}))

		// Interactive chat API (always available in API mode for IDE plugins)
		mux.HandleFunc("POST /api/v1/interactive/chat", s.handleInteractiveChat)
		mux.HandleFunc("POST /api/v1/interactive/command", s.handleInteractiveCommand)
		mux.HandleFunc("POST /api/v1/interactive/answer", s.handleViaRouter(CommandRoute{
			Command: "interactive-answer",
			ParseFn: parseInteractiveAnswerInvocation,
		}))
		mux.HandleFunc("GET /api/v1/interactive/state", s.handleViaRouter(CommandRoute{
			Command:           "interactive-state",
			UnwrapData:        true,
			AllowNilConductor: true,
		}))
		mux.HandleFunc("POST /api/v1/interactive/stop", s.handleInteractiveStop)
		mux.HandleFunc("GET /api/v1/interactive/commands", s.handleViaRouter(CommandRoute{
			Command:           "interactive-commands",
			UnwrapData:        true,
			AllowNilConductor: true,
		}))

		// Commit API endpoints (always available)
		mux.HandleFunc("GET /api/v1/commit/changes", s.handleViaRouter(CommandRoute{
			Command:    "commit",
			ParseFn:    parseCommitChangesInvocation,
			UnwrapData: true,
		}))
		mux.HandleFunc("GET /api/v1/commit/plan", s.handleViaRouter(CommandRoute{
			Command:    "commit",
			ParseFn:    parseCommitPlanInvocation,
			UnwrapData: true,
		}))
		mux.HandleFunc("POST /api/v1/commit/execute", s.handleViaRouter(CommandRoute{
			Command:    "commit",
			ParseFn:    parseCommitExecuteInvocation,
			UnwrapData: true,
		}))

		// Stack management API (always available)
		mux.HandleFunc("GET /api/v1/stack", s.handleViaRouter(CommandRoute{
			Command:    "stack",
			ParseFn:    parseStackSubcommand("list"),
			UnwrapData: true,
		}))
		mux.HandleFunc("POST /api/v1/stack/sync", s.handleViaRouter(CommandRoute{
			Command:    "stack",
			ParseFn:    parseStackSubcommand("sync"),
			UnwrapData: true,
		}))
		mux.HandleFunc("GET /api/v1/stack/rebase-preview", s.handleViaRouter(CommandRoute{
			Command:    "stack",
			ParseFn:    parseStackRebasePreviewInvocation,
			UnwrapData: true,
		}))
		mux.HandleFunc("POST /api/v1/stack/rebase", s.handleViaRouter(CommandRoute{
			Command:    "stack",
			ParseFn:    parseStackRebaseInvocation,
			UnwrapData: true,
		}))

		// Project workflow endpoints
		mux.HandleFunc("POST /api/v1/project/upload", s.handleViaRouter(CommandRoute{
			Command:    "project",
			ParseFn:    s.parseProjectUploadInvocation(),
			UnwrapData: true,
		}))
		mux.HandleFunc("POST /api/v1/project/source", s.handleViaRouter(CommandRoute{
			Command:    "project",
			ParseFn:    s.parseProjectSourceInvocation(),
			UnwrapData: true,
		}))
		mux.HandleFunc("POST /api/v1/project/plan", s.handleViaRouter(CommandRoute{
			Command:    "project",
			ParseFn:    parseProjectPlanInvocation,
			UnwrapData: true,
		}))
		mux.HandleFunc("GET /api/v1/project/queues", s.handleViaRouter(CommandRoute{
			Command:    "project",
			ParseFn:    parseProjectSubcommand("queues"),
			UnwrapData: true,
		}))
		mux.HandleFunc("GET /api/v1/project/queue/", s.handleViaRouter(CommandRoute{
			Command:    "project",
			ParseFn:    parseProjectQueueInvocation,
			UnwrapData: true,
		}))
		mux.HandleFunc("DELETE /api/v1/project/queue/", s.handleViaRouter(CommandRoute{
			Command:    "project",
			ParseFn:    parseProjectQueueDeleteInvocation,
			UnwrapData: true,
		}))
		mux.HandleFunc("GET /api/v1/project/tasks", s.handleViaRouter(CommandRoute{
			Command:    "project",
			ParseFn:    parseProjectTasksInvocation,
			UnwrapData: true,
		}))
		mux.HandleFunc("PUT /api/v1/project/tasks/", s.handleViaRouter(CommandRoute{
			Command:    "project",
			ParseFn:    parseProjectTaskEditInvocation,
			UnwrapData: true,
		}))
		mux.HandleFunc("POST /api/v1/project/reorder", s.handleViaRouter(CommandRoute{
			Command:    "project",
			ParseFn:    parseProjectReorderInvocation,
			UnwrapData: true,
		}))
		mux.HandleFunc("POST /api/v1/project/submit", s.handleViaRouter(CommandRoute{
			Command:    "project",
			ParseFn:    parseProjectSubmitInvocation,
			UnwrapData: true,
		}))
		mux.HandleFunc("POST /api/v1/project/start", s.handleViaRouter(CommandRoute{
			Command:    "project",
			ParseFn:    parseProjectStartInvocation,
			UnwrapData: true,
		}))
		mux.HandleFunc("POST /api/v1/project/sync", s.handleProjectSync)

		// Quick tasks endpoints (API always available, UI only in full mode)
		mux.HandleFunc("GET /api/v1/quick", s.handleViaRouter(CommandRoute{
			Command:    "quick-list",
			UnwrapData: true,
		}))
		mux.HandleFunc("POST /api/v1/quick", s.handleViaRouter(CommandRoute{
			Command:    "quick",
			ParseFn:    parseQuickCreateInvocation,
			UnwrapData: true,
		}))
		mux.HandleFunc("POST /api/v1/quick/submit-source", s.handleViaRouter(CommandRoute{
			Command:    "submit-source",
			ParseFn:    parseSubmitSourceInvocation,
			UnwrapData: true,
		}))
		// Quick task item endpoints using Go 1.22+ wildcard patterns
		mux.HandleFunc("GET /api/v1/quick/{taskId}", s.handleViaRouter(CommandRoute{
			Command:    "quick-get",
			ParseFn:    parseQuickTaskIDInvocation,
			UnwrapData: true,
		}))
		mux.HandleFunc("POST /api/v1/quick/{taskId}/note", s.handleViaRouter(CommandRoute{
			Command: "quick-note",
			ParseFn: parseQuickNoteInvocation,
		}))
		mux.HandleFunc("POST /api/v1/quick/{taskId}/optimize", s.handleViaRouter(CommandRoute{
			Command:    "optimize",
			ParseFn:    parseQuickOptimizeInvocation,
			UnwrapData: true,
		}))
		mux.HandleFunc("POST /api/v1/quick/{taskId}/export", s.handleViaRouter(CommandRoute{
			Command:    "export",
			ParseFn:    parseQuickExportInvocation,
			UnwrapData: true,
		}))
		mux.HandleFunc("POST /api/v1/quick/{taskId}/submit", s.handleViaRouter(CommandRoute{
			Command:    "submit",
			ParseFn:    parseQuickSubmitInvocation,
			UnwrapData: true,
		}))
		mux.HandleFunc("POST /api/v1/quick/{taskId}/start", s.handleViaRouter(CommandRoute{
			Command: "start",
			ParseFn: parseQuickStartInvocation,
		}))
		mux.HandleFunc("DELETE /api/v1/quick/{taskId}", s.handleViaRouter(CommandRoute{
			Command: "delete",
			ParseFn: parseQuickDeleteInvocation,
		}))

		// Running parallel tasks endpoints
		mux.HandleFunc("GET /api/v1/running", s.handleViaRouter(CommandRoute{
			Command:           "running-list",
			InjectFn:          s.injectTaskRegistry(),
			UnwrapData:        true,
			AllowNilConductor: true,
		}))
		mux.HandleFunc("POST /api/v1/running/{id}/cancel", s.handleViaRouter(CommandRoute{
			Command:           "running-cancel",
			ParseFn:           parseRunningCancelInvocation,
			InjectFn:          s.injectTaskRegistry(),
			UnwrapData:        true,
			AllowNilConductor: true,
		}))
		mux.HandleFunc("GET /api/v1/running/{id}/stream", s.handleRunningTaskStream)
		mux.HandleFunc("POST /api/v1/parallel", s.handleParallelStart)
	}

	// Global mode routes
	if s.config.Mode == ModeGlobal {
		mux.HandleFunc("GET /api/v1/projects", s.handleViaRouter(CommandRoute{
			Command:           "projects-list",
			UnwrapData:        true,
			AllowNilConductor: true,
		}))
		mux.HandleFunc("POST /api/v1/projects/select", s.handleSelectProject)

		// Settings endpoints
		mux.HandleFunc("GET /api/v1/settings", s.handleViaRouter(CommandRoute{
			Command:           "settings-get",
			ParseFn:           parseSettingsGetInvocation,
			InjectFn:          s.injectSettingsMode(),
			UnwrapData:        true,
			AllowNilConductor: true,
		}))
		mux.HandleFunc("POST /api/v1/settings", s.handleSaveSettings)
		mux.HandleFunc("GET /api/v1/settings/explain", s.handleViaRouter(CommandRoute{
			Command:    "config-explain",
			ParseFn:    parseConfigExplainInvocation,
			UnwrapData: true,
		}))
		mux.HandleFunc("GET /api/v1/settings/provider-health", s.handleViaRouter(CommandRoute{
			Command:    "provider-health",
			UnwrapData: true,
		}))

		// Budget status endpoint (returns placeholder when no workspace)
		mux.HandleFunc("GET /api/v1/budget/monthly/status", s.handleBudgetMonthlyStatus)

		// Sandbox endpoints (also available in global mode)
		mux.HandleFunc("GET /api/v1/sandbox/status", s.handleViaRouter(CommandRoute{
			Command:           "sandbox-status",
			UnwrapData:        true,
			AllowNilConductor: true,
		}))
		mux.HandleFunc("POST /api/v1/sandbox/enable", s.handleViaRouter(CommandRoute{
			Command:    "sandbox-enable",
			UnwrapData: true,
		}))
		mux.HandleFunc("POST /api/v1/sandbox/disable", s.handleViaRouter(CommandRoute{
			Command:    "sandbox-disable",
			UnwrapData: true,
		}))

		// Library endpoints (shared collections available without project)
		globalLibInject := injectLibraryGlobalMode(s.config.Mode)
		mux.HandleFunc("GET /api/v1/library", s.handleViaRouter(CommandRoute{
			Command:    "library",
			ParseFn:    parseLibraryListInvocation,
			InjectFn:   globalLibInject,
			UnwrapData: true,
		}))
		mux.HandleFunc("GET /api/v1/library/stats", s.handleViaRouter(CommandRoute{
			Command:    "library",
			ParseFn:    parseLibraryStatsInvocation,
			InjectFn:   globalLibInject,
			UnwrapData: true,
		}))
		mux.HandleFunc("POST /api/v1/library/pull", s.handleViaRouter(CommandRoute{
			Command:    "library",
			ParseFn:    parseLibraryPullInvocation,
			UnwrapData: true,
		}))
		mux.HandleFunc("POST /api/v1/library/pull/preview", s.handleViaRouter(CommandRoute{
			Command:    "library",
			ParseFn:    parseLibraryPullPreviewInvocation,
			UnwrapData: true,
		}))
		mux.HandleFunc("GET /api/v1/library/{id}/items", s.handleViaRouter(CommandRoute{
			Command:    "library",
			ParseFn:    parseLibraryItemsInvocation,
			UnwrapData: true,
		}))
		mux.HandleFunc("GET /api/v1/library/", s.handleViaRouter(CommandRoute{
			Command:    "library",
			ParseFn:    parseLibraryShowInvocation,
			UnwrapData: true,
		}))
		mux.HandleFunc("DELETE /api/v1/library/", s.handleViaRouter(CommandRoute{
			Command:    "library",
			ParseFn:    parseLibraryRemoveInvocation,
			UnwrapData: true,
		}))
	}

	// Switch project route (available when started in global mode)
	if s.startedInGlobalMode {
		mux.HandleFunc("POST /api/v1/projects/switch", s.handleSwitchProject)
	}

	// SSE events endpoint
	mux.HandleFunc("GET /api/v1/events", s.handleEvents)

	// Agent logs streaming endpoints
	mux.HandleFunc("GET /api/v1/agent/logs/stream", s.handleAgentLogs)
	mux.HandleFunc("GET /api/v1/agent/logs/history", s.handleViaRouter(CommandRoute{
		Command:           "agent-logs-history",
		ParseFn:           parseAgentLogsHistoryInvocation,
		UnwrapData:        true,
		AllowNilConductor: true,
	}))

	// React SPA catch-all - skip in API-only mode
	// Note: /{path...} matches all paths including root "/" in Go 1.22+
	if !s.config.APIOnly {
		mux.HandleFunc("GET /{path...}", s.handleReactApp)
	}

	// Wrap with middleware chain (innermost to outermost):
	// 1. Logging (innermost)
	// 2. CSRF validation (outermost)
	handler := s.withMiddleware(mux)
	handler = s.csrfMiddleware(handler)

	return handler
}

// withMiddleware wraps the handler with common middleware.
func (s *Server) withMiddleware(h http.Handler) http.Handler {
	// Logging middleware
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create response wrapper to capture status code
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		h.ServeHTTP(rw, r)

		// Log request
		slog.Debug("http request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rw.statusCode,
			"duration", time.Since(start).String(),
		)
	})
}

// responseWriter wraps http.ResponseWriter to capture status code.
type responseWriter struct {
	http.ResponseWriter

	statusCode     int
	headersWritten bool
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	if !rw.headersWritten {
		rw.ResponseWriter.WriteHeader(code)
		rw.headersWritten = true
	}
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	// Mark headers as written on first Write call
	if !rw.headersWritten {
		rw.headersWritten = true
	}

	return rw.ResponseWriter.Write(b)
}

// Flush implements http.Flusher for SSE (Server-Sent Events) support.
// It delegates to the underlying ResponseWriter's Flush method if available.
func (rw *responseWriter) Flush() {
	if flusher, ok := rw.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// Health check handler.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
		"mode":   s.modeString(),
	})
}

// Events handler provides SSE stream of events.
func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	// Disable write timeout for SSE (allows indefinite streaming during long agent operations)
	rc := http.NewResponseController(w)
	if err := rc.SetWriteDeadline(time.Time{}); err != nil {
		slog.Debug("could not disable write deadline for SSE", "error", err)
	}

	// Set CORS header first (before any error response)
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Check if response writer supports flushing BEFORE setting SSE headers
	flusher, ok := w.(http.Flusher)
	if !ok {
		s.writeError(w, http.StatusInternalServerError, "streaming not supported")

		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// If no event bus, just keep connection alive with heartbeats
	if s.config.EventBus == nil {
		s.writeSSEEvent(w, flusher, "connected", map[string]string{"status": "connected"})
		s.runSSEHeartbeatLoop(w, flusher, r.Context())

		return
	}

	// Use channel-based event delivery to prevent panic on client disconnect.
	// The callback only writes to a channel (never panics), while the main loop
	// checks ctx.Done() before writing to ResponseWriter.
	eventCh := make(chan eventbus.Event, 500) // Increased from 100 to prevent event drops during heavy agent streaming
	subID := s.config.EventBus.SubscribeAll(func(e eventbus.Event) {
		select {
		case eventCh <- e:
		default:
			// Channel full, drop event (non-blocking to prevent callback from hanging)
		}
	})
	defer s.config.EventBus.Unsubscribe(subID)
	defer close(eventCh)

	// Send initial connection event
	s.writeSSEEvent(w, flusher, "connected", map[string]string{"status": "connected"})

	// Run combined event + heartbeat loop (blocks until client disconnects)
	s.runSSEEventLoop(w, flusher, r.Context(), eventCh)
}

// writeJSON writes a JSON response.
func (s *Server) writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("failed to encode JSON response", "error", err)
	}
}

// writeError writes an error response.
func (s *Server) writeError(w http.ResponseWriter, status int, message string) {
	s.writeJSON(w, status, map[string]string{
		"error": message,
	})
}

// writeSSEEvent writes a Server-Sent Event.
// Includes panic recovery as defense-in-depth against invalid ResponseWriter access.
func (s *Server) writeSSEEvent(w http.ResponseWriter, flusher http.Flusher, eventType string, data any) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("panic in SSE write (client likely disconnected)", "panic", r, "event_type", eventType)
		}
	}()

	jsonData, err := json.Marshal(data)
	if err != nil {
		slog.Error("failed to marshal SSE data", "error", err)

		return
	}

	if _, err = w.Write([]byte("event: " + eventType + "\n")); err != nil {
		slog.Error("failed to write SSE event", "error", err)

		return
	}
	if _, err = w.Write([]byte("data: " + string(jsonData) + "\n\n")); err != nil {
		slog.Error("failed to write SSE data", "error", err)

		return
	}
	flusher.Flush()
}

// sendHeartbeat sends a heartbeat event with current workflow state.
// Returns true (retained for API consistency; disconnection is detected via context cancellation).
//
//nolint:unparam // Return value kept for API consistency; callers handle disconnection via context
func (s *Server) sendHeartbeat(w http.ResponseWriter, flusher http.Flusher, ctx context.Context, lastState *string) bool {
	if s.config.Conductor != nil {
		if status, err := s.config.Conductor.Status(ctx); err == nil {
			stateChanged := status.State != *lastState
			*lastState = status.State

			s.writeSSEEvent(w, flusher, "heartbeat", map[string]any{
				"state":         status.State,
				"state_changed": stateChanged,
				"task_id":       status.TaskID,
				"specs":         status.Specifications,
				"checkpoints":   status.Checkpoints,
				"agent":         status.Agent,
				"timestamp":     time.Now().Unix(),
			})

			return true
		}
	}
	// Fallback: send minimal heartbeat event (no conductor or status failed)
	s.writeSSEEvent(w, flusher, "heartbeat", map[string]any{
		"state":     "unknown",
		"timestamp": time.Now().Unix(),
	})

	return true
}

// runSSEHeartbeatLoop sends periodic heartbeat events to keep SSE connections alive during long operations.
// It polls Conductor.Status() every 15 seconds and emits heartbeat events with workflow state info.
// Blocks until the context is cancelled (client disconnects).
func (s *Server) runSSEHeartbeatLoop(w http.ResponseWriter, flusher http.Flusher, ctx context.Context) {
	var lastState string
	if s.config.Conductor != nil {
		if status, err := s.config.Conductor.Status(ctx); err == nil {
			lastState = status.State
		}
	}

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !s.sendHeartbeat(w, flusher, ctx, &lastState) {
				return
			}
		}
	}
}

// runSSEEventLoop processes events from a channel while sending periodic heartbeats.
// This pattern prevents panics when clients disconnect by checking context before writes.
// Blocks until the context is cancelled (client disconnects).
func (s *Server) runSSEEventLoop(w http.ResponseWriter, flusher http.Flusher, ctx context.Context, eventCh <-chan eventbus.Event) {
	var lastState string
	if s.config.Conductor != nil {
		if status, err := s.config.Conductor.Status(ctx); err == nil {
			lastState = status.State
		}
	}

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case e, ok := <-eventCh:
			if !ok {
				return
			}
			s.writeSSEEvent(w, flusher, string(e.Type), e.Data)
		case <-ticker.C:
			if !s.sendHeartbeat(w, flusher, ctx, &lastState) {
				return
			}
		}
	}
}
