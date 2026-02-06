package com.valksor.mehrhof.services

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
import com.valksor.mehrhof.api.models.TaskInfo
import com.valksor.mehrhof.api.models.TaskWork
import com.valksor.mehrhof.settings.MehrhofSettings
import com.valksor.mehrhof.util.CommandParser
import com.valksor.mehrhof.util.EventParser
import com.valksor.mehrhof.util.ReconnectionPolicy
import kotlinx.coroutines.*
import java.util.concurrent.CopyOnWriteArrayList
import java.util.concurrent.atomic.AtomicBoolean

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
class MehrhofProjectService(
    private val project: Project
) : Disposable {
    private val log = Logger.getInstance(MehrhofProjectService::class.java)
    private val settings = MehrhofSettings.getInstance()

    // Coroutine scope for background tasks
    private val scope = CoroutineScope(Dispatchers.IO + SupervisorJob())

    // State change listeners
    private val stateListeners = CopyOnWriteArrayList<StateListener>()

    // Connection state
    private val connected = AtomicBoolean(false)
    private val connecting = AtomicBoolean(false)
    private val reconnectionPolicy =
        ReconnectionPolicy(
            maxAttempts = settings.maxReconnectAttempts,
            delaySeconds = settings.reconnectDelaySeconds
        )

    // Server process management (delegated)
    private val serverManager =
        MehrhofServerManager(
            scope = scope,
            onServerReady = { url -> connectToUrl(url) },
            onError = { message -> notifyError(message) },
            onInfo = { message -> notifyInfo(message) },
            onProcessExited = { stateListeners.forEach { it.onConnectionChanged(false) } }
        )

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

    /**
     * Listener interface for state changes.
     */
    interface StateListener {
        fun onConnectionChanged(connected: Boolean) {}

        fun onWorkflowStateChanged(
            state: String,
            previousState: String?
        ) {}

        fun onTaskChanged(
            task: TaskInfo?,
            work: TaskWork?
        ) {}

        fun onQuestionReceived(
            question: String,
            options: List<String>?
        ) {}

        fun onAgentMessage(
            content: String,
            type: String?
        ) {}

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
    fun isServerRunning(): Boolean = serverManager.isRunning()

    /**
     * Get the server port, or null if not running.
     */
    fun getServerPort(): Int? = serverManager.getServerPort()

    /**
     * Get the API client, or null if not connected.
     */
    fun getApiClient(): MehrhofApiClient? = apiClient

    /**
     * Start the Mehrhof server for this project.
     * Delegates to [MehrhofServerManager] which spawns `mehr serve --api`.
     */
    fun startServer() {
        serverManager.startServer(project, settings)
    }

    /**
     * Stop the Mehrhof server process.
     */
    fun stopServer() {
        serverManager.stopServer(preShutdown = { disconnect() })
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
        for (@Suppress("UnusedPrivateProperty") attempt in 1..10) {
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
        reconnectionPolicy.reset()

        // Notify listeners
        withContext(Dispatchers.Main) {
            stateListeners.forEach { it.onConnectionChanged(true) }
        }

        // Refresh initial state
        refreshState()

        // Fetch available commands for dynamic discovery
        fetchCommands()

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
        val port = serverManager.getServerPort()
        if (serverManager.isRunning() && port != null) {
            connectToUrl("http://localhost:$port")
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

        eventStreamClient =
            EventStreamClient(
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

    private fun handleEvent(
        eventType: EventType,
        data: JsonObject
    ) {
        scope.launch {
            when (val parsed = EventParser.parse(eventType, data)) {
                is EventParser.ParsedEvent.WorkflowStateChanged -> {
                    val previousState = workflowState
                    workflowState = parsed.newState

                    withContext(Dispatchers.Main) {
                        stateListeners.forEach {
                            it.onWorkflowStateChanged(parsed.newState, previousState)
                        }
                    }
                }

                is EventParser.ParsedEvent.TaskLifecycleEvent -> {
                    refreshState()
                }

                is EventParser.ParsedEvent.QuestionAsked -> {
                    pendingQuestion = parsed.question
                    pendingQuestionOptions = parsed.options

                    withContext(Dispatchers.Main) {
                        stateListeners.forEach {
                            it.onQuestionReceived(parsed.question, parsed.options)
                        }
                    }

                    if (settings.showNotifications) {
                        notifyQuestion(parsed.question)
                    }
                }

                is EventParser.ParsedEvent.AnswerProvided -> {
                    pendingQuestion = null
                    pendingQuestionOptions = null
                }

                is EventParser.ParsedEvent.AgentMessage -> {
                    withContext(Dispatchers.Main) {
                        stateListeners.forEach {
                            it.onAgentMessage(parsed.content, parsed.type)
                        }
                    }
                }

                is EventParser.ParsedEvent.Error -> {
                    withContext(Dispatchers.Main) {
                        stateListeners.forEach { it.onError(parsed.message) }
                    }

                    if (settings.showNotifications) {
                        notifyError(parsed.message)
                    }
                }

                is EventParser.ParsedEvent.Ignored -> {
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
            client
                .getTask()
                .onSuccess { response ->
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
     * Fetch available commands from the discovery API and update the command parser.
     * This allows the IDE to dynamically discover commands instead of using a hardcoded list.
     */
    private fun fetchCommands() {
        scope.launch {
            val client = apiClient ?: return@launch

            client
                .getCommands()
                .onSuccess { response ->
                    CommandParser.updateCommands(response.commands)
                    log.info("Updated command list with ${response.commands.size} commands from discovery API")
                }.onFailure { error ->
                    log.warn("Failed to fetch commands, using defaults: ${error.message}")
                    // Keep using default commands on failure
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
        val attempts = reconnectionPolicy.recordAttempt()
        if (!reconnectionPolicy.shouldReconnect()) {
            log.warn("Max reconnect attempts reached")
            notifyError("Failed to reconnect to Mehrhof server after $attempts attempts")
            return
        }

        log.info("Scheduling reconnect attempt $attempts in ${settings.reconnectDelaySeconds}s")

        scope.launch {
            delay(reconnectionPolicy.nextDelayMs())
            if (!connected.get()) {
                // Reconnect to the appropriate URL
                val managerPort = serverManager.getServerPort()
                val url =
                    when {
                        managerPort != null -> "http://localhost:$managerPort"
                        settings.serverUrl.isNotEmpty() -> settings.serverUrl
                        else -> return@launch // No URL to reconnect to
                    }
                doConnectToUrl(url)
            }
        }
    }

    private fun notifyInfo(message: String) {
        if (!settings.showNotifications) return

        NotificationGroupManager
            .getInstance()
            .getNotificationGroup("Mehrhof")
            .createNotification(message, NotificationType.INFORMATION)
            .notify(project)
    }

    private fun notifyError(message: String) {
        NotificationGroupManager
            .getInstance()
            .getNotificationGroup("Mehrhof")
            .createNotification(message, NotificationType.ERROR)
            .notify(project)
    }

    private fun notifyQuestion(question: String) {
        NotificationGroupManager
            .getInstance()
            .getNotificationGroup("Mehrhof")
            .createNotification("Agent question: $question", NotificationType.WARNING)
            .notify(project)
    }

    override fun dispose() {
        serverManager.dispose()
        disconnect()
        scope.cancel()
        stateListeners.clear()
    }

    companion object {
        fun getInstance(project: Project): MehrhofProjectService = project.getService(MehrhofProjectService::class.java)
    }
}
