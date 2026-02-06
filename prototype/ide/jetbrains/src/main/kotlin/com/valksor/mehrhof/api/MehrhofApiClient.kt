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
    // HTTP Helpers
    // ========================================================================

    internal inline fun <reified T> get(path: String): Result<T> {
        val requestBuilder =
            Request
                .Builder()
                .url("$baseUrl$path")
                .get()
                .addHeader("Accept", "application/json")

        sessionCookie?.let { requestBuilder.addHeader("Cookie", it) }

        return executeRequest(requestBuilder.build())
    }

    internal inline fun <reified T> post(
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

    internal inline fun <reified T> executeRequest(request: Request): Result<T> =
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

    internal fun extractSessionCookie(response: okhttp3.Response) {
        val setCookie = response.header("Set-Cookie") ?: return
        val match = Regex("mehr_session=([^;]+)").find(setCookie) ?: return
        sessionCookie = "mehr_session=${match.groupValues[1]}"
    }

    internal inline fun <reified T> processResponse(response: okhttp3.Response): Result<T> {
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

    internal fun parseErrorMessage(
        body: String?,
        code: Int,
        message: String
    ): String =
        try {
            body?.let { gson.fromJson(it, ErrorResponse::class.java)?.error }
        } catch (_: Exception) {
            null
        } ?: "HTTP $code: $message"

    internal inline fun <reified T> parseResponseBody(body: String): Result<T> =
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
