package com.valksor.mehrhof.api

import com.valksor.mehrhof.api.models.*

// ========================================================================
// Queue Task Extensions (via Interactive API)
// ========================================================================

/**
 * Create a quick task.
 * POST /api/v1/interactive/command
 */
fun MehrhofApiClient.createQuickTask(description: String): Result<InteractiveCommandResponse> =
    post("/api/v1/interactive/command", InteractiveCommandRequest("quick", listOf(description)))

/**
 * Delete a queue task.
 * POST /api/v1/interactive/command
 */
fun MehrhofApiClient.deleteQueueTask(
    queueId: String,
    taskId: String
): Result<InteractiveCommandResponse> =
    post("/api/v1/interactive/command", InteractiveCommandRequest("delete", listOf("$queueId/$taskId")))

/**
 * Export a queue task to markdown.
 * POST /api/v1/interactive/command
 */
fun MehrhofApiClient.exportQueueTask(
    queueId: String,
    taskId: String
): Result<InteractiveCommandResponse> =
    post("/api/v1/interactive/command", InteractiveCommandRequest("export", listOf("$queueId/$taskId")))

/**
 * AI optimize a queue task.
 * POST /api/v1/interactive/command
 */
fun MehrhofApiClient.optimizeQueueTask(
    queueId: String,
    taskId: String
): Result<InteractiveCommandResponse> =
    post("/api/v1/interactive/command", InteractiveCommandRequest("optimize", listOf("$queueId/$taskId")))

/**
 * Submit a queue task to a provider.
 * POST /api/v1/interactive/command
 */
fun MehrhofApiClient.submitQueueTask(
    queueId: String,
    taskId: String,
    provider: String
): Result<InteractiveCommandResponse> =
    post("/api/v1/interactive/command", InteractiveCommandRequest("submit", listOf("$queueId/$taskId", provider)))

/**
 * Sync task from provider.
 * POST /api/v1/interactive/command
 */
fun MehrhofApiClient.syncTask(): Result<InteractiveCommandResponse> =
    post("/api/v1/interactive/command", InteractiveCommandRequest("sync", emptyList()))

// ========================================================================
// Find & Search Extensions
// ========================================================================

/**
 * Search the codebase.
 * GET /api/v1/find?q={query}
 */
fun MehrhofApiClient.find(query: String): Result<FindSearchResponse> =
    get("/api/v1/find?q=${java.net.URLEncoder.encode(query, "UTF-8")}")

// ========================================================================
// Memory Extensions
// ========================================================================

/**
 * Search memory for similar tasks.
 * GET /api/v1/memory/search?q={query}
 */
fun MehrhofApiClient.memorySearch(query: String): Result<MemorySearchResponse> =
    get("/api/v1/memory/search?q=${java.net.URLEncoder.encode(query, "UTF-8")}&limit=10")

/**
 * Index a task to memory.
 * POST /api/v1/memory/index
 */
fun MehrhofApiClient.memoryIndex(taskId: String): Result<MemoryIndexResponse> =
    post("/api/v1/memory/index", MemoryIndexRequest(taskId))

/**
 * Get memory statistics.
 * GET /api/v1/memory/stats
 */
fun MehrhofApiClient.memoryStats(): Result<MemoryStatsResponse> = get("/api/v1/memory/stats")

// ========================================================================
// Library Extensions
// ========================================================================

/**
 * List documentation collections.
 * GET /api/v1/library
 */
fun MehrhofApiClient.libraryList(): Result<LibraryListResponse> = get("/api/v1/library")

/**
 * Show a specific collection.
 * GET /api/v1/library/{nameOrId}
 */
fun MehrhofApiClient.libraryShow(nameOrId: String): Result<LibraryShowResponse> =
    get("/api/v1/library/${java.net.URLEncoder.encode(nameOrId, "UTF-8")}")

/**
 * Get library statistics.
 * GET /api/v1/library/stats
 */
fun MehrhofApiClient.libraryStats(): Result<LibraryStatsResponse> = get("/api/v1/library/stats")

/**
 * Pull a documentation collection.
 * POST /api/v1/interactive/command
 */
fun MehrhofApiClient.libraryPull(
    source: String,
    name: String? = null,
    shared: Boolean = false
): Result<InteractiveCommandResponse> {
    val args = mutableListOf("pull", source)
    if (name != null) {
        args.add("--name")
        args.add(name)
    }
    if (shared) {
        args.add("--shared")
    }
    return post("/api/v1/interactive/command", InteractiveCommandRequest("library", args))
}

/**
 * Remove a documentation collection.
 * POST /api/v1/interactive/command
 */
fun MehrhofApiClient.libraryRemove(nameOrId: String): Result<InteractiveCommandResponse> =
    post("/api/v1/interactive/command", InteractiveCommandRequest("library", listOf("remove", nameOrId)))

// ========================================================================
// Links Extensions
// ========================================================================

/**
 * List all links.
 * GET /api/v1/links
 */
fun MehrhofApiClient.linksList(): Result<LinksListResponse> = get("/api/v1/links")

/**
 * Get links for a specific entity.
 * GET /api/v1/links/{entityId}
 */
fun MehrhofApiClient.linksGet(entityId: String): Result<EntityLinksResponse> =
    get("/api/v1/links/${java.net.URLEncoder.encode(entityId, "UTF-8")}")

/**
 * Search for entities.
 * GET /api/v1/links/search?q={query}
 */
fun MehrhofApiClient.linksSearch(query: String): Result<LinksSearchResponse> =
    get("/api/v1/links/search?q=${java.net.URLEncoder.encode(query, "UTF-8")}")

/**
 * Get links statistics.
 * GET /api/v1/links/stats
 */
fun MehrhofApiClient.linksStats(): Result<LinksStatsResponse> = get("/api/v1/links/stats")

/**
 * Rebuild links index.
 * POST /api/v1/interactive/command
 */
fun MehrhofApiClient.linksRebuild(): Result<InteractiveCommandResponse> =
    post("/api/v1/interactive/command", InteractiveCommandRequest("links", listOf("rebuild")))

// ========================================================================
// Browser Extensions
// ========================================================================

/**
 * Get browser status.
 * GET /api/v1/browser/status
 */
fun MehrhofApiClient.browserStatus(): Result<BrowserStatusResponse> = get("/api/v1/browser/status")

/**
 * List browser tabs.
 * GET /api/v1/browser/tabs
 */
fun MehrhofApiClient.browserTabs(): Result<BrowserTabsResponse> = get("/api/v1/browser/tabs")

/**
 * Open a URL in a new browser tab.
 * POST /api/v1/browser/goto
 */
fun MehrhofApiClient.browserGoto(url: String): Result<BrowserGotoResponse> =
    post("/api/v1/browser/goto", BrowserGotoRequest(url))

/**
 * Navigate current tab to a URL.
 * POST /api/v1/browser/navigate
 */
fun MehrhofApiClient.browserNavigate(
    url: String,
    tabId: String? = null
): Result<BrowserNavigateResponse> = post("/api/v1/browser/navigate", BrowserNavigateRequest(tabId, url))

/**
 * Click an element by CSS selector.
 * POST /api/v1/browser/click
 */
fun MehrhofApiClient.browserClick(
    selector: String,
    tabId: String? = null
): Result<BrowserClickResponse> = post("/api/v1/browser/click", BrowserClickRequest(tabId, selector))

/**
 * Type text into an element.
 * POST /api/v1/browser/type
 */
fun MehrhofApiClient.browserType(
    selector: String,
    text: String,
    tabId: String? = null,
    clear: Boolean = false
): Result<BrowserTypeResponse> = post("/api/v1/browser/type", BrowserTypeRequest(tabId, selector, text, clear))

/**
 * Evaluate JavaScript in the browser.
 * POST /api/v1/browser/eval
 */
fun MehrhofApiClient.browserEval(
    expression: String,
    tabId: String? = null
): Result<BrowserEvalResponse> = post("/api/v1/browser/eval", BrowserEvalRequest(tabId, expression))

/**
 * Take a screenshot.
 * POST /api/v1/browser/screenshot
 */
fun MehrhofApiClient.browserScreenshot(
    tabId: String? = null,
    format: String? = null,
    fullPage: Boolean = false
): Result<BrowserScreenshotResponse> =
    post("/api/v1/browser/screenshot", BrowserScreenshotRequest(tabId, format, null, fullPage))

/**
 * Reload the page.
 * POST /api/v1/browser/reload
 */
fun MehrhofApiClient.browserReload(
    tabId: String? = null,
    hard: Boolean = false
): Result<BrowserReloadResponse> = post("/api/v1/browser/reload", BrowserReloadRequest(tabId, hard))

/**
 * Close a browser tab.
 * POST /api/v1/browser/close
 */
fun MehrhofApiClient.browserClose(tabId: String): Result<BrowserCloseResponse> =
    post("/api/v1/browser/close", BrowserCloseRequest(tabId))

/**
 * Get console messages.
 * POST /api/v1/browser/console
 */
fun MehrhofApiClient.browserConsole(
    tabId: String? = null,
    duration: Int? = null,
    level: String? = null
): Result<BrowserConsoleResponse> = post("/api/v1/browser/console", BrowserConsoleRequest(tabId, duration, level))

/**
 * Get network requests.
 * POST /api/v1/browser/network
 */
fun MehrhofApiClient.browserNetwork(
    tabId: String? = null,
    duration: Int? = null,
    captureBody: Boolean = false
): Result<BrowserNetworkResponse> =
    post("/api/v1/browser/network", BrowserNetworkRequest(tabId, duration, captureBody, null))

// ========================================================================
// Auto Workflow Extensions
// ========================================================================

/**
 * Run automated workflow.
 * POST /api/v1/workflow/auto
 */
fun MehrhofApiClient.auto(loops: Int = 0): Result<InteractiveCommandResponse> {
    val args = if (loops > 0) listOf("--loops", loops.toString()) else emptyList()
    return post("/api/v1/interactive/command", InteractiveCommandRequest("auto", args))
}

// ========================================================================
// Budget Extensions
// ========================================================================

/**
 * Get monthly budget status.
 * GET /api/v1/budget/monthly/status
 */
fun MehrhofApiClient.budgetStatus(): Result<BudgetStatusResponse> = get("/api/v1/budget/monthly/status")

/**
 * Reset monthly budget spending.
 * POST /api/v1/budget/monthly/reset
 */
fun MehrhofApiClient.budgetReset(): Result<InteractiveCommandResponse> =
    post("/api/v1/interactive/command", InteractiveCommandRequest("budget", listOf("reset")))

// ========================================================================
// Simplify Extensions
// ========================================================================

/**
 * Run code simplification.
 * POST /api/v1/workflow/simplify
 */
fun MehrhofApiClient.simplify(
    path: String? = null,
    instructions: String? = null
): Result<InteractiveCommandResponse> {
    val args = mutableListOf<String>()
    if (path != null) args.add(path)
    if (instructions != null) {
        args.add("--instructions")
        args.add(instructions)
    }
    return post("/api/v1/interactive/command", InteractiveCommandRequest("simplify", args))
}

// ========================================================================
// Label Extensions
// ========================================================================

/**
 * List labels for the current task.
 * GET /api/v1/labels
 */
fun MehrhofApiClient.labelsList(): Result<LabelsListResponse> = get("/api/v1/labels")

/**
 * Add a label to the current task.
 * POST /api/v1/labels
 */
fun MehrhofApiClient.labelsAdd(label: String): Result<InteractiveCommandResponse> =
    post("/api/v1/interactive/command", InteractiveCommandRequest("label", listOf("add", label)))

/**
 * Remove a label from the current task.
 * POST /api/v1/interactive/command
 */
fun MehrhofApiClient.labelsRemove(label: String): Result<InteractiveCommandResponse> =
    post("/api/v1/interactive/command", InteractiveCommandRequest("label", listOf("remove", label)))
