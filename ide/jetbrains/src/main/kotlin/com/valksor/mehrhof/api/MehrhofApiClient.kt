package com.valksor.mehrhof.api

import com.google.gson.Gson
import com.google.gson.GsonBuilder
import com.valksor.mehrhof.api.models.*
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import java.io.IOException
import java.util.concurrent.TimeUnit

/**
 * HTTP client for communicating with the Mehrhof Web UI server.
 *
 * All methods are blocking and should be called from a background thread.
 */
@Suppress("TooManyFunctions") // API client: each method maps to one endpoint
class MehrhofApiClient(
    private val baseUrl: String
) {
    private val client =
        OkHttpClient
            .Builder()
            .connectTimeout(10, TimeUnit.SECONDS)
            .readTimeout(60, TimeUnit.SECONDS) // Longer for workflow operations
            .writeTimeout(30, TimeUnit.SECONDS)
            .build()

    private val gson: Gson =
        GsonBuilder()
            .setLenient()
            .create()

    private val jsonMediaType = "application/json; charset=utf-8".toMediaType()

    @Volatile
    private var sessionCookie: String? = null

    @Volatile
    private var csrfToken: String? = null

    fun setSessionCookie(cookie: String?) {
        this.sessionCookie = cookie
    }

    fun setCsrfToken(token: String?) {
        this.csrfToken = token
    }

    // ========================================================================
    // Status & Task Endpoints
    // ========================================================================

    /**
     * Get server status.
     * GET /api/v1/status
     */
    fun getStatus(): Result<StatusResponse> = get("/api/v1/status")

    /**
     * Get active task details.
     * GET /api/v1/task
     */
    fun getTask(): Result<TaskResponse> = get("/api/v1/task")

    /**
     * List all tasks in the workspace.
     * GET /api/v1/tasks
     */
    fun getTasks(): Result<TaskListResponse> = get("/api/v1/tasks")

    /**
     * Get specifications for a task.
     * GET /api/v1/tasks/{id}/specs
     */
    fun getSpecifications(taskId: String): Result<SpecificationsResponse> = get("/api/v1/tasks/$taskId/specs")

    /**
     * Get sessions for a task.
     * GET /api/v1/tasks/{id}/sessions
     */
    fun getSessions(taskId: String): Result<SessionsResponse> = get("/api/v1/tasks/$taskId/sessions")

    /**
     * Get guidance on next actions.
     * GET /api/v1/guide
     */
    fun getGuide(): Result<GuideResponse> = get("/api/v1/guide")

    // ========================================================================
    // Workflow Endpoints
    // ========================================================================

    /**
     * Start a new task.
     * POST /api/v1/workflow/start
     */
    fun startTask(
        ref: String? = null,
        content: String? = null
    ): Result<WorkflowResponse> = post("/api/v1/workflow/start", StartTaskRequest(ref = ref, content = content))

    /**
     * Run planning step.
     * POST /api/v1/workflow/plan
     */
    fun plan(agent: String? = null): Result<WorkflowResponse> =
        post("/api/v1/workflow/plan", WorkflowRequest(agent = agent))

    /**
     * Run implementation step.
     * POST /api/v1/workflow/implement
     */
    fun implement(agent: String? = null): Result<WorkflowResponse> =
        post("/api/v1/workflow/implement", WorkflowRequest(agent = agent))

    /**
     * Run review step.
     * POST /api/v1/workflow/review
     */
    fun review(agent: String? = null): Result<WorkflowResponse> =
        post("/api/v1/workflow/review", WorkflowRequest(agent = agent))

    /**
     * Finish the task.
     * POST /api/v1/workflow/finish
     */
    fun finish(options: FinishRequest = FinishRequest()): Result<WorkflowResponse> =
        post("/api/v1/workflow/finish", options)

    /**
     * Continue workflow with optional auto-execution.
     * POST /api/v1/workflow/continue
     */
    fun continueWorkflow(auto: Boolean = false): Result<ContinueResponse> =
        post("/api/v1/workflow/continue", mapOf("auto" to auto))

    /**
     * Undo to previous checkpoint.
     * POST /api/v1/workflow/undo
     */
    fun undo(): Result<WorkflowResponse> = post("/api/v1/workflow/undo", emptyMap<String, Any>())

    /**
     * Redo to next checkpoint.
     * POST /api/v1/workflow/redo
     */
    fun redo(): Result<WorkflowResponse> = post("/api/v1/workflow/redo", emptyMap<String, Any>())

    /**
     * Answer a pending agent question.
     * POST /api/v1/workflow/answer
     */
    fun answer(answer: String): Result<WorkflowResponse> =
        post("/api/v1/workflow/answer", AnswerRequest(answer = answer))

    /**
     * Resume a paused task.
     * POST /api/v1/workflow/resume
     */
    fun resume(): Result<WorkflowResponse> = post("/api/v1/workflow/resume", emptyMap<String, Any>())

    /**
     * Abandon the current task.
     * POST /api/v1/workflow/abandon
     */
    fun abandon(): Result<WorkflowResponse> = post("/api/v1/workflow/abandon", emptyMap<String, Any>())

    /**
     * Reset workflow state to idle.
     * POST /api/v1/workflow/reset
     */
    fun reset(): Result<WorkflowResponse> = post("/api/v1/workflow/reset", emptyMap<String, Any>())

    /**
     * Ask the agent a question during implementation.
     * POST /api/v1/workflow/question
     */
    fun question(message: String): Result<WorkflowResponse> =
        post("/api/v1/workflow/question", QuestionRequest(message))

    /**
     * Add a note to a task.
     * POST /api/v1/tasks/{id}/notes
     */
    fun addNote(
        taskId: String,
        message: String
    ): Result<AddNoteResponse> = post("/api/v1/tasks/$taskId/notes", AddNoteRequest(message))

    // ========================================================================
    // Interactive API Endpoints
    // ========================================================================

    /**
     * Execute an interactive command.
     * POST /api/v1/interactive/command
     */
    fun executeCommand(
        command: String,
        args: List<String> = emptyList()
    ): Result<InteractiveCommandResponse> =
        post("/api/v1/interactive/command", InteractiveCommandRequest(command, args))

    /**
     * Send a chat message to the agent.
     * POST /api/v1/interactive/chat
     */
    fun chat(message: String): Result<InteractiveChatResponse> =
        post("/api/v1/interactive/chat", InteractiveChatRequest(message))

    /**
     * Answer an agent's question.
     * POST /api/v1/interactive/answer
     */
    fun answerInteractive(response: String): Result<InteractiveCommandResponse> =
        post("/api/v1/interactive/answer", InteractiveAnswerRequest(response))

    /**
     * Get current interactive state.
     * GET /api/v1/interactive/state
     */
    fun getInteractiveState(): Result<InteractiveStateResponse> = get("/api/v1/interactive/state")

    /**
     * Stop the current running operation.
     * POST /api/v1/interactive/stop
     */
    fun stopOperation(): Result<InteractiveStopResponse> = post("/api/v1/interactive/stop", emptyMap<String, Any>())

    /**
     * Get available commands for discovery.
     * GET /api/v1/interactive/commands
     */
    fun getCommands(): Result<CommandsResponse> = get("/api/v1/interactive/commands")

    // ========================================================================
    // Cost Endpoints
    // ========================================================================

    /**
     * Get costs for a specific task.
     * GET /api/v1/tasks/{id}/costs
     */
    fun getTaskCosts(taskId: String): Result<TaskCostResponse> = get("/api/v1/tasks/$taskId/costs")

    /**
     * Get all costs.
     * GET /api/v1/costs
     */
    fun getAllCosts(): Result<AllCostsResponse> = get("/api/v1/costs")

    // ========================================================================
    // Agent & Provider Endpoints
    // ========================================================================

    /**
     * List available agents.
     * GET /api/v1/agents
     */
    fun getAgents(): Result<AgentsListResponse> = get("/api/v1/agents")

    /**
     * List available providers.
     * GET /api/v1/providers
     */
    fun getProviders(): Result<ProvidersListResponse> = get("/api/v1/providers")

    // ========================================================================
    // Queue Task Endpoints (via Interactive API)
    // ========================================================================

    /**
     * Create a quick task.
     * POST /api/v1/interactive/command
     */
    fun createQuickTask(description: String): Result<InteractiveCommandResponse> =
        post("/api/v1/interactive/command", InteractiveCommandRequest("quick", listOf(description)))

    /**
     * Delete a queue task.
     * POST /api/v1/interactive/command
     */
    fun deleteQueueTask(
        queueId: String,
        taskId: String
    ): Result<InteractiveCommandResponse> =
        post("/api/v1/interactive/command", InteractiveCommandRequest("delete", listOf("$queueId/$taskId")))

    /**
     * Export a queue task to markdown.
     * POST /api/v1/interactive/command
     */
    fun exportQueueTask(
        queueId: String,
        taskId: String
    ): Result<InteractiveCommandResponse> =
        post("/api/v1/interactive/command", InteractiveCommandRequest("export", listOf("$queueId/$taskId")))

    /**
     * AI optimize a queue task.
     * POST /api/v1/interactive/command
     */
    fun optimizeQueueTask(
        queueId: String,
        taskId: String
    ): Result<InteractiveCommandResponse> =
        post("/api/v1/interactive/command", InteractiveCommandRequest("optimize", listOf("$queueId/$taskId")))

    /**
     * Submit a queue task to a provider.
     * POST /api/v1/interactive/command
     */
    fun submitQueueTask(
        queueId: String,
        taskId: String,
        provider: String
    ): Result<InteractiveCommandResponse> =
        post("/api/v1/interactive/command", InteractiveCommandRequest("submit", listOf("$queueId/$taskId", provider)))

    /**
     * Sync task from provider.
     * POST /api/v1/interactive/command
     */
    fun syncTask(): Result<InteractiveCommandResponse> =
        post("/api/v1/interactive/command", InteractiveCommandRequest("sync", emptyList()))

    // ========================================================================
    // Find & Search Endpoints
    // ========================================================================

    /**
     * Search the codebase.
     * GET /api/v1/find?q={query}
     */
    fun find(query: String): Result<FindSearchResponse> =
        get("/api/v1/find?q=${java.net.URLEncoder.encode(query, "UTF-8")}")

    // ========================================================================
    // Memory Endpoints
    // ========================================================================

    /**
     * Search memory for similar tasks.
     * GET /api/v1/memory/search?q={query}
     */
    fun memorySearch(query: String): Result<MemorySearchResponse> =
        get("/api/v1/memory/search?q=${java.net.URLEncoder.encode(query, "UTF-8")}&limit=10")

    /**
     * Index a task to memory.
     * POST /api/v1/memory/index
     */
    fun memoryIndex(taskId: String): Result<MemoryIndexResponse> =
        post("/api/v1/memory/index", MemoryIndexRequest(taskId))

    /**
     * Get memory statistics.
     * GET /api/v1/memory/stats
     */
    fun memoryStats(): Result<MemoryStatsResponse> = get("/api/v1/memory/stats")

    // ========================================================================
    // Library Endpoints
    // ========================================================================

    /**
     * List documentation collections.
     * GET /api/v1/library
     */
    fun libraryList(): Result<LibraryListResponse> = get("/api/v1/library")

    /**
     * Show a specific collection.
     * GET /api/v1/library/{nameOrId}
     */
    fun libraryShow(nameOrId: String): Result<LibraryShowResponse> =
        get("/api/v1/library/${java.net.URLEncoder.encode(nameOrId, "UTF-8")}")

    /**
     * Get library statistics.
     * GET /api/v1/library/stats
     */
    fun libraryStats(): Result<LibraryStatsResponse> = get("/api/v1/library/stats")

    /**
     * Pull a documentation collection.
     * POST /api/v1/interactive/command
     */
    fun libraryPull(
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
    fun libraryRemove(nameOrId: String): Result<InteractiveCommandResponse> =
        post("/api/v1/interactive/command", InteractiveCommandRequest("library", listOf("remove", nameOrId)))

    // ========================================================================
    // Links Endpoints
    // ========================================================================

    /**
     * List all links.
     * GET /api/v1/links
     */
    fun linksList(): Result<LinksListResponse> = get("/api/v1/links")

    /**
     * Get links for a specific entity.
     * GET /api/v1/links/{entityId}
     */
    fun linksGet(entityId: String): Result<EntityLinksResponse> =
        get("/api/v1/links/${java.net.URLEncoder.encode(entityId, "UTF-8")}")

    /**
     * Search for entities.
     * GET /api/v1/links/search?q={query}
     */
    fun linksSearch(query: String): Result<LinksSearchResponse> =
        get("/api/v1/links/search?q=${java.net.URLEncoder.encode(query, "UTF-8")}")

    /**
     * Get links statistics.
     * GET /api/v1/links/stats
     */
    fun linksStats(): Result<LinksStatsResponse> = get("/api/v1/links/stats")

    /**
     * Rebuild links index.
     * POST /api/v1/interactive/command
     */
    fun linksRebuild(): Result<InteractiveCommandResponse> =
        post("/api/v1/interactive/command", InteractiveCommandRequest("links", listOf("rebuild")))

    // ========================================================================
    // Browser Endpoints
    // ========================================================================

    /**
     * Get browser status.
     * GET /api/v1/browser/status
     */
    fun browserStatus(): Result<BrowserStatusResponse> = get("/api/v1/browser/status")

    /**
     * List browser tabs.
     * GET /api/v1/browser/tabs
     */
    fun browserTabs(): Result<BrowserTabsResponse> = get("/api/v1/browser/tabs")

    /**
     * Open a URL in a new browser tab.
     * POST /api/v1/browser/goto
     */
    fun browserGoto(url: String): Result<BrowserGotoResponse> = post("/api/v1/browser/goto", BrowserGotoRequest(url))

    /**
     * Navigate current tab to a URL.
     * POST /api/v1/browser/navigate
     */
    fun browserNavigate(
        url: String,
        tabId: String? = null
    ): Result<BrowserNavigateResponse> = post("/api/v1/browser/navigate", BrowserNavigateRequest(tabId, url))

    /**
     * Click an element by CSS selector.
     * POST /api/v1/browser/click
     */
    fun browserClick(
        selector: String,
        tabId: String? = null
    ): Result<BrowserClickResponse> = post("/api/v1/browser/click", BrowserClickRequest(tabId, selector))

    /**
     * Type text into an element.
     * POST /api/v1/browser/type
     */
    fun browserType(
        selector: String,
        text: String,
        tabId: String? = null,
        clear: Boolean = false
    ): Result<BrowserTypeResponse> = post("/api/v1/browser/type", BrowserTypeRequest(tabId, selector, text, clear))

    /**
     * Evaluate JavaScript in the browser.
     * POST /api/v1/browser/eval
     */
    fun browserEval(
        expression: String,
        tabId: String? = null
    ): Result<BrowserEvalResponse> = post("/api/v1/browser/eval", BrowserEvalRequest(tabId, expression))

    /**
     * Take a screenshot.
     * POST /api/v1/browser/screenshot
     */
    fun browserScreenshot(
        tabId: String? = null,
        format: String? = null,
        fullPage: Boolean = false
    ): Result<BrowserScreenshotResponse> =
        post("/api/v1/browser/screenshot", BrowserScreenshotRequest(tabId, format, null, fullPage))

    /**
     * Reload the page.
     * POST /api/v1/browser/reload
     */
    fun browserReload(
        tabId: String? = null,
        hard: Boolean = false
    ): Result<BrowserReloadResponse> = post("/api/v1/browser/reload", BrowserReloadRequest(tabId, hard))

    /**
     * Close a browser tab.
     * POST /api/v1/browser/close
     */
    fun browserClose(tabId: String): Result<BrowserCloseResponse> =
        post("/api/v1/browser/close", BrowserCloseRequest(tabId))

    /**
     * Get console messages.
     * POST /api/v1/browser/console
     */
    fun browserConsole(
        tabId: String? = null,
        duration: Int? = null,
        level: String? = null
    ): Result<BrowserConsoleResponse> = post("/api/v1/browser/console", BrowserConsoleRequest(tabId, duration, level))

    /**
     * Get network requests.
     * POST /api/v1/browser/network
     */
    fun browserNetwork(
        tabId: String? = null,
        duration: Int? = null,
        captureBody: Boolean = false
    ): Result<BrowserNetworkResponse> =
        post("/api/v1/browser/network", BrowserNetworkRequest(tabId, duration, captureBody, null))

    // ========================================================================
    // HTTP Helpers
    // ========================================================================

    private inline fun <reified T> get(path: String): Result<T> {
        val requestBuilder =
            Request
                .Builder()
                .url("$baseUrl$path")
                .get()
                .addHeader("Accept", "application/json")

        sessionCookie?.let { requestBuilder.addHeader("Cookie", it) }

        return executeRequest(requestBuilder.build())
    }

    private inline fun <reified T> post(
        path: String,
        body: Any
    ): Result<T> {
        val jsonBody = gson.toJson(body)
        val requestBuilder =
            Request
                .Builder()
                .url("$baseUrl$path")
                .post(jsonBody.toRequestBody(jsonMediaType))
                .addHeader("Accept", "application/json")
                .addHeader("Content-Type", "application/json")

        sessionCookie?.let { requestBuilder.addHeader("Cookie", it) }
        csrfToken?.let { requestBuilder.addHeader("X-Csrf-Token", it) }

        return executeRequest(requestBuilder.build())
    }

    private inline fun <reified T> executeRequest(request: Request): Result<T> =
        try {
            client.newCall(request).execute().use { response ->
                extractSessionCookie(response)
                processResponse(response)
            }
        } catch (e: IOException) {
            Result.failure(ApiException(0, "Network error: ${e.message}"))
        } catch (e: Exception) {
            Result.failure(ApiException(0, "Unexpected error: ${e.message}"))
        }

    private fun extractSessionCookie(response: okhttp3.Response) {
        val setCookie = response.header("Set-Cookie") ?: return
        val match = Regex("mehr_session=([^;]+)").find(setCookie) ?: return
        sessionCookie = "mehr_session=${match.groupValues[1]}"
    }

    private inline fun <reified T> processResponse(response: okhttp3.Response): Result<T> {
        val body = response.body?.string()

        if (!response.isSuccessful) {
            val errorMsg = parseErrorMessage(body, response.code, response.message)
            return Result.failure(ApiException(response.code, errorMsg))
        }

        if (body.isNullOrBlank()) {
            return Result.failure(ApiException(0, "Empty response body"))
        }

        return parseResponseBody(body)
    }

    private fun parseErrorMessage(
        body: String?,
        code: Int,
        message: String
    ): String =
        try {
            body?.let { gson.fromJson(it, ErrorResponse::class.java)?.error }
        } catch (_: Exception) {
            null
        } ?: "HTTP $code: $message"

    private inline fun <reified T> parseResponseBody(body: String): Result<T> =
        try {
            val result = gson.fromJson(body, T::class.java)
            Result.success(result)
        } catch (e: Exception) {
            Result.failure(ApiException(0, "Failed to parse response: ${e.message}"))
        }

    /**
     * Check if the server is reachable.
     */
    fun isReachable(): Boolean =
        try {
            val requestBuilder =
                Request
                    .Builder()
                    .url("$baseUrl/health")
                    .get()

            sessionCookie?.let { requestBuilder.addHeader("Cookie", it) }

            client.newCall(requestBuilder.build()).execute().use { response ->
                response.isSuccessful
            }
        } catch (_: Exception) {
            false
        }
}

/**
 * Exception thrown when an API request fails.
 */
class ApiException(
    val statusCode: Int,
    override val message: String
) : Exception(message)
