package com.valksor.mehrhof.services

import com.google.gson.Gson
import com.google.gson.JsonObject
import com.intellij.notification.NotificationGroupManager
import com.intellij.notification.NotificationType
import com.intellij.openapi.Disposable
import com.intellij.openapi.components.Service
import com.intellij.openapi.diagnostic.Logger
import com.intellij.openapi.project.Project
import com.valksor.mehrhof.api.EventStreamClient
import com.valksor.mehrhof.api.EventType
import com.valksor.mehrhof.api.MehrhofApiClient
import com.valksor.mehrhof.api.models.GuideResponse
import com.valksor.mehrhof.api.models.TaskInfo
import com.valksor.mehrhof.api.models.TaskResponse
import com.valksor.mehrhof.api.models.TaskWork
import com.valksor.mehrhof.settings.MehrhofSettings
import com.intellij.util.EnvironmentUtil
import kotlinx.coroutines.*
import java.io.File
import java.util.concurrent.CopyOnWriteArrayList
import java.util.concurrent.atomic.AtomicBoolean
import java.util.concurrent.atomic.AtomicInteger

/**
 * Per-project service managing Mehrhof server connection and state.
 *
 * Provides:
 * - Connection management (connect/disconnect/reconnect)
 * - Current task and workflow state tracking
 * - Event stream subscription
 * - State change listeners
 */
@Service(Service.Level.PROJECT)
class MehrhofProjectService(private val project: Project) : Disposable {

    private val log = Logger.getInstance(MehrhofProjectService::class.java)
    private val settings = MehrhofSettings.getInstance()
    private val gson = Gson()

    // Connection state
    private val connected = AtomicBoolean(false)
    private val connecting = AtomicBoolean(false)
    private val reconnectAttempts = AtomicInteger(0)

    // Server process management
    private var serverProcess: Process? = null
    private var serverPort: Int? = null
    private var serverOutputJob: Job? = null

    // API clients
    private var apiClient: MehrhofApiClient? = null
    private var eventStreamClient: EventStreamClient? = null

    // Current state
    @Volatile
    var currentTask: TaskInfo? = null
        private set

    @Volatile
    var currentTaskWork: TaskWork? = null
        private set

    @Volatile
    var workflowState: String = "idle"
        private set

    @Volatile
    var pendingQuestion: String? = null
        private set

    @Volatile
    var pendingQuestionOptions: List<String>? = null
        private set

    // Coroutine scope for background tasks
    private val scope = CoroutineScope(Dispatchers.IO + SupervisorJob())

    // State change listeners
    private val stateListeners = CopyOnWriteArrayList<StateListener>()

    /**
     * Listener interface for state changes.
     */
    interface StateListener {
        fun onConnectionChanged(connected: Boolean) {}
        fun onWorkflowStateChanged(state: String, previousState: String?) {}
        fun onTaskChanged(task: TaskInfo?, work: TaskWork?) {}
        fun onQuestionReceived(question: String, options: List<String>?) {}
        fun onAgentMessage(content: String, type: String?) {}
        fun onError(message: String) {}
    }

    /**
     * Add a state change listener.
     */
    fun addStateListener(listener: StateListener) {
        stateListeners.add(listener)
    }

    /**
     * Remove a state change listener.
     */
    fun removeStateListener(listener: StateListener) {
        stateListeners.remove(listener)
    }

    /**
     * Check if connected to the server.
     */
    fun isConnected(): Boolean = connected.get()

    /**
     * Check if the server process is running.
     */
    fun isServerRunning(): Boolean = serverProcess?.isAlive == true

    /**
     * Get the server port, or null if not running.
     */
    fun getServerPort(): Int? = serverPort

    /**
     * Get the API client, or null if not connected.
     */
    fun getApiClient(): MehrhofApiClient? = apiClient

    // ========================================================================
    // Server Process Management
    // ========================================================================

    /**
     * Find the mehr binary, checking user config then default install locations.
     */
    private fun findMehrBinary(): String {
        // User configured path takes priority
        val configured = settings.mehrExecutable
        if (configured.isNotEmpty() && File(configured).canExecute()) {
            return configured
        }

        // Try default install locations
        val home = System.getProperty("user.home")
        val candidates = listOf(
            "$home/.local/bin/mehr",
            "$home/bin/mehr",
            "/usr/local/bin/mehr"
        )

        for (path in candidates) {
            if (File(path).canExecute()) {
                return path
            }
        }

        throw IllegalStateException(
            "mehr not found. Install with 'curl -fsSL https://valksor.com/install | bash' " +
                "or configure path in Settings → Tools → Mehrhof"
        )
    }

    /**
     * Start the Mehrhof server for this project.
     * Spawns `mehr serve` and captures the port from output.
     */
    fun startServer() {
        if (serverProcess?.isAlive == true) {
            log.info("Server already running")
            return
        }

        val projectPath = project.basePath
        if (projectPath == null) {
            notifyError("Cannot start server: no project path")
            return
        }

        val mehrBinary: String
        try {
            mehrBinary = findMehrBinary()
        } catch (e: IllegalStateException) {
            notifyError(e.message ?: "mehr not found")
            return
        }

        log.info("Starting Mehrhof server in $projectPath using $mehrBinary")

        try {
            // Use IntelliJ's EnvironmentUtil to get user's shell environment
            // This loads PATH and other variables from user's login shell (bash, zsh, fish, etc.)
            val processBuilder = ProcessBuilder(mehrBinary, "serve")
                .directory(File(projectPath))
                .redirectErrorStream(true)

            // Apply user's shell environment from EnvironmentUtil
            val env = processBuilder.environment()
            env.putAll(EnvironmentUtil.getEnvironmentMap())

            serverProcess = processBuilder.start()

            // Read stdout in background, parse for port
            serverOutputJob = scope.launch(Dispatchers.IO) {
                val output = StringBuilder()
                val process = serverProcess ?: return@launch

                try {
                    process.inputStream?.bufferedReader()?.useLines { lines ->
                        for (line in lines) {
                            output.appendLine(line)
                            log.info("Server: $line")

                            // Parse: "Server running at: http://localhost:XXXXX" or similar
                            val match = Regex("""Server running at: https?://[^:]+:(\d+)""").find(line)
                            if (match != null) {
                                val port = match.groupValues[1].toIntOrNull()
                                if (port != null) {
                                    serverPort = port
                                    val url = "http://localhost:$port"
                                    log.info("Server started on port $port")

                                    withContext(Dispatchers.Main) {
                                        notifyInfo("Server started on port $port")
                                        stateListeners.forEach { it.onConnectionChanged(false) }
                                    }

                                    // Connect to the server
                                    connectToUrl(url)
                                }
                            }
                        }
                    }
                } catch (e: Exception) {
                    if (e !is CancellationException) {
                        log.warn("Error reading server output: ${e.message}")
                    }
                }

                // Process ended - capture exit code
                val exitCode = try { process.waitFor() } catch (_: Exception) { -1 }
                val capturedPort = serverPort

                withContext(Dispatchers.Main) {
                    if (capturedPort == null) {
                        // Server failed to start - show error with output
                        val lastOutput = output.toString().takeLast(500)
                        notifyError("Server exited (code $exitCode):\n$lastOutput")
                    }
                    serverProcess = null
                    serverPort = null
                    stateListeners.forEach { it.onConnectionChanged(false) }
                }
            }
        } catch (e: Exception) {
            log.error("Failed to start server: ${e.message}")
            notifyError("Failed to start server: ${e.message}")
            serverProcess = null
        }
    }

    /**
     * Stop the Mehrhof server process.
     */
    fun stopServer() {
        log.info("Stopping Mehrhof server")

        // Disconnect first
        disconnect()

        // Cancel output reading job
        serverOutputJob?.cancel()
        serverOutputJob = null

        // Destroy the process
        serverProcess?.let { process ->
            process.destroy()
            // Wait a bit for graceful shutdown
            scope.launch(Dispatchers.IO) {
                try {
                    withTimeout(5000) {
                        while (process.isAlive) {
                            delay(100)
                        }
                    }
                } catch (_: TimeoutCancellationException) {
                    // Force kill if still alive
                    process.destroyForcibly()
                }
            }
        }

        serverProcess = null
        serverPort = null

        // Note: disconnect() already notifies listeners, no need to do it again
        notifyInfo("Server stopped")
    }

    /**
     * Connect to a specific URL.
     */
    private fun connectToUrl(url: String) {
        if (connected.get() || connecting.get()) {
            return
        }

        connecting.set(true)
        scope.launch {
            try {
                doConnectToUrl(url)
            } finally {
                connecting.set(false)
            }
        }
    }

    private suspend fun doConnectToUrl(serverUrl: String) {
        log.info("Connecting to Mehrhof server at $serverUrl")

        // Create API client
        val client = MehrhofApiClient(serverUrl)

        // Check if server is reachable (with retries for startup)
        var reachable = false
        for (attempt in 1..10) {
            if (client.isReachable()) {
                reachable = true
                break
            }
            delay(500)
        }

        if (!reachable) {
            log.warn("Mehrhof server not reachable at $serverUrl")
            notifyError("Cannot connect to Mehrhof server at $serverUrl")
            handleConnectionFailure()
            return
        }

        apiClient = client
        connected.set(true)
        reconnectAttempts.set(0)

        // Notify listeners
        withContext(Dispatchers.Main) {
            stateListeners.forEach { it.onConnectionChanged(true) }
        }

        // Refresh initial state
        refreshState()

        // Connect to event stream
        connectEventStream(serverUrl)

        log.info("Connected to Mehrhof server")
    }

    /**
     * Connect to the Mehrhof server.
     * If a URL is configured in settings, use it. Otherwise, requires startServer() first.
     */
    fun connect() {
        // If server is already running (started by plugin), we should already be connected
        if (serverProcess?.isAlive == true && serverPort != null) {
            connectToUrl("http://localhost:$serverPort")
            return
        }

        // Check if user configured a manual URL
        val serverUrl = settings.serverUrl
        if (serverUrl.isEmpty()) {
            log.info("No server URL configured - use startServer() first")
            notifyError("No server running. Click 'Start Server' to launch.")
            return
        }

        // Connect to manually configured URL
        connectToUrl(serverUrl)
    }

    private fun connectEventStream(serverUrl: String) {
        eventStreamClient?.disconnect()

        eventStreamClient = EventStreamClient(
            baseUrl = serverUrl,
            onEvent = { eventType, data -> handleEvent(eventType, data) },
            onError = { error ->
                log.warn("SSE error: $error")
                scope.launch {
                    withContext(Dispatchers.Main) {
                        stateListeners.forEach { it.onError(error) }
                    }
                }
            },
            onConnected = {
                log.info("SSE connected")
            },
            onDisconnected = {
                log.info("SSE disconnected")
                if (connected.get() && settings.autoReconnect) {
                    scheduleReconnect()
                }
            }
        )

        eventStreamClient?.connect()
    }

    private fun handleEvent(eventType: EventType, data: JsonObject) {
        scope.launch {
            when (eventType) {
                EventType.WORKFLOW_STATE_CHANGED -> {
                    val newState = data.get("state")?.asString ?: return@launch
                    val previousState = workflowState
                    workflowState = newState

                    withContext(Dispatchers.Main) {
                        stateListeners.forEach {
                            it.onWorkflowStateChanged(newState, previousState)
                        }
                    }
                }

                EventType.TASK_STARTED, EventType.TASK_COMPLETED, EventType.TASK_FAILED -> {
                    refreshState()
                }

                EventType.QUESTION_ASKED -> {
                    val question = data.get("question")?.asString ?: return@launch
                    val options = data.getAsJsonArray("options")?.map { it.asString }
                    pendingQuestion = question
                    pendingQuestionOptions = options

                    withContext(Dispatchers.Main) {
                        stateListeners.forEach { it.onQuestionReceived(question, options) }
                    }

                    if (settings.showNotifications) {
                        notifyQuestion(question)
                    }
                }

                EventType.ANSWER_PROVIDED -> {
                    pendingQuestion = null
                    pendingQuestionOptions = null
                }

                EventType.AGENT_MESSAGE -> {
                    val content = data.get("content")?.asString ?: return@launch
                    val type = data.get("type")?.asString

                    withContext(Dispatchers.Main) {
                        stateListeners.forEach { it.onAgentMessage(content, type) }
                    }
                }

                EventType.ERROR -> {
                    val error = data.get("error")?.asString
                        ?: data.get("message")?.asString
                        ?: "Unknown error"

                    withContext(Dispatchers.Main) {
                        stateListeners.forEach { it.onError(error) }
                    }

                    if (settings.showNotifications) {
                        notifyError(error)
                    }
                }

                else -> {
                    // Ignore other events
                }
            }
        }
    }

    /**
     * Refresh current task and workflow state from the server.
     */
    fun refreshState() {
        scope.launch {
            val client = apiClient ?: return@launch

            // Get current task
            client.getTask().onSuccess { response ->
                val oldTask = currentTask
                currentTask = response.task
                currentTaskWork = response.work

                if (response.pendingQuestion != null) {
                    pendingQuestion = response.pendingQuestion.question
                    pendingQuestionOptions = response.pendingQuestion.options
                }

                // Get workflow state from guide
                client.getGuide().onSuccess { guide ->
                    val newState = guide.state ?: "idle"
                    val previousState = workflowState
                    workflowState = newState

                    withContext(Dispatchers.Main) {
                        stateListeners.forEach {
                            it.onTaskChanged(currentTask, currentTaskWork)
                            if (newState != previousState) {
                                it.onWorkflowStateChanged(newState, previousState)
                            }
                        }
                    }
                }
            }.onFailure { error ->
                log.warn("Failed to refresh state: ${error.message}")
            }
        }
    }

    /**
     * Disconnect from the server.
     */
    fun disconnect() {
        log.info("Disconnecting from Mehrhof server")

        eventStreamClient?.disconnect()
        eventStreamClient = null

        apiClient = null
        connected.set(false)

        currentTask = null
        currentTaskWork = null
        workflowState = "idle"
        pendingQuestion = null
        pendingQuestionOptions = null

        stateListeners.forEach { it.onConnectionChanged(false) }
    }

    private fun handleConnectionFailure() {
        connected.set(false)
        stateListeners.forEach { it.onConnectionChanged(false) }

        if (settings.autoReconnect) {
            scheduleReconnect()
        }
    }

    private fun scheduleReconnect() {
        val attempts = reconnectAttempts.incrementAndGet()
        if (attempts > settings.maxReconnectAttempts) {
            log.warn("Max reconnect attempts reached")
            notifyError("Failed to reconnect to Mehrhof server after $attempts attempts")
            return
        }

        log.info("Scheduling reconnect attempt $attempts in ${settings.reconnectDelaySeconds}s")

        scope.launch {
            delay(settings.reconnectDelaySeconds * 1000L)
            if (!connected.get()) {
                // Reconnect to the appropriate URL
                val url = when {
                    serverPort != null -> "http://localhost:$serverPort"
                    settings.serverUrl.isNotEmpty() -> settings.serverUrl
                    else -> return@launch // No URL to reconnect to
                }
                doConnectToUrl(url)
            }
        }
    }

    // ========================================================================
    // Notifications
    // ========================================================================

    private fun notifyInfo(message: String) {
        if (!settings.showNotifications) return

        NotificationGroupManager.getInstance()
            .getNotificationGroup("Mehrhof")
            .createNotification(message, NotificationType.INFORMATION)
            .notify(project)
    }

    private fun notifyError(message: String) {
        NotificationGroupManager.getInstance()
            .getNotificationGroup("Mehrhof")
            .createNotification(message, NotificationType.ERROR)
            .notify(project)
    }

    private fun notifyQuestion(question: String) {
        NotificationGroupManager.getInstance()
            .getNotificationGroup("Mehrhof")
            .createNotification("Agent question: $question", NotificationType.WARNING)
            .notify(project)
    }

    // ========================================================================
    // Disposable
    // ========================================================================

    override fun dispose() {
        // Stop server if running
        serverOutputJob?.cancel()
        serverProcess?.destroy()
        serverProcess = null
        serverPort = null

        disconnect()
        scope.cancel()
        stateListeners.clear()
    }

    companion object {
        fun getInstance(project: Project): MehrhofProjectService =
            project.getService(MehrhofProjectService::class.java)
    }
}
