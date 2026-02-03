package com.valksor.mehrhof.toolwindow

import com.intellij.openapi.actionSystem.ActionManager
import com.intellij.openapi.actionSystem.ActionPlaces
import com.intellij.openapi.actionSystem.DefaultActionGroup
import com.intellij.openapi.project.Project
import com.intellij.openapi.ui.SimpleToolWindowPanel
import com.intellij.ui.JBColor
import com.intellij.ui.components.JBLabel
import com.intellij.ui.components.JBScrollPane
import com.intellij.ui.components.JBTextField
import com.intellij.util.ui.JBUI
import com.valksor.mehrhof.api.models.TaskInfo
import com.valksor.mehrhof.api.models.TaskWork
import com.valksor.mehrhof.services.MehrhofProjectService
import com.valksor.mehrhof.util.CommandParser
import com.valksor.mehrhof.util.HtmlEscaper
import com.valksor.mehrhof.util.ParsedInput
import com.valksor.mehrhof.util.WorkflowUtils
import kotlinx.coroutines.*
import java.awt.*
import java.awt.event.KeyAdapter
import java.awt.event.KeyEvent
import javax.swing.*
import javax.swing.text.html.HTMLEditorKit
import javax.swing.text.html.StyleSheet

/**
 * Interactive terminal panel matching the web UI's /interactive page.
 * Provides chat interface, command input, and action buttons.
 */
class InteractivePanel(
    @Suppress("unused") private val project: Project,
    private val service: MehrhofProjectService
) : SimpleToolWindowPanel(true, true),
    MehrhofProjectService.StateListener {
    private val scope = CoroutineScope(Dispatchers.Main + SupervisorJob())

    // Header components
    private val taskIdLabel = JBLabel("No active task")
    private val stateBadge = StateBadge()

    // Server control
    private val startStopButton = JButton("Start Server")
    private val connectionStatusLabel = JBLabel()

    // Messages area
    private val messagesPane =
        JEditorPane().apply {
            contentType = "text/html"
            isEditable = false
            val kit = HTMLEditorKit()
            val styleSheet = StyleSheet()
            styleSheet.addRule("body { font-family: sans-serif; font-size: 12px; margin: 8px; }")
            styleSheet.addRule(".user { color: #2196F3; margin-bottom: 8px; }")
            styleSheet.addRule(".assistant { color: #4CAF50; margin-bottom: 8px; }")
            styleSheet.addRule(".system { color: #9E9E9E; font-style: italic; margin-bottom: 8px; }")
            styleSheet.addRule(".error { color: #F44336; margin-bottom: 8px; }")
            styleSheet.addRule(".command { color: #FF9800; margin-bottom: 4px; }")
            kit.styleSheet = styleSheet
            editorKit = kit
        }
    private val messagesScrollPane = JBScrollPane(messagesPane)
    private val messagesContent = StringBuilder("<html><body>")

    // Input area
    private val inputField =
        JBTextField().apply {
            emptyText.text = "Type a command or message..."
        }
    private val sendButton = JButton("Send")
    private val stopButton =
        JButton("Stop").apply {
            isEnabled = false
            foreground = JBColor.RED
        }

    // Command history
    private val commandHistory = mutableListOf<String>()
    private var historyIndex = -1

    init {
        setupToolbar()
        setupContent()
        setupListeners()
        updateConnectionStatus()
        updateTaskInfo()
    }

    private fun setupToolbar() {
        val actionGroup =
            DefaultActionGroup().apply {
                add(ActionManager.getInstance().getAction("Mehrhof.Toolbar.Refresh"))
            }

        val toolbar =
            ActionManager
                .getInstance()
                .createActionToolbar(ActionPlaces.TOOLWINDOW_TITLE, actionGroup, true)
        toolbar.targetComponent = this
        setToolbar(toolbar.component)
    }

    private fun setupContent() {
        val mainPanel =
            JPanel(BorderLayout()).apply {
                border = JBUI.Borders.empty(8)
            }

        // Header: Server control + Task info
        val headerPanel =
            JPanel(BorderLayout()).apply {
                border = JBUI.Borders.emptyBottom(8)

                // Server control row
                val serverPanel =
                    JPanel(FlowLayout(FlowLayout.LEFT, 8, 0)).apply {
                        add(startStopButton)
                        add(connectionStatusLabel)
                    }
                add(serverPanel, BorderLayout.NORTH)

                // Task info row
                val taskPanel =
                    JPanel(FlowLayout(FlowLayout.LEFT, 8, 4)).apply {
                        add(JBLabel("Task:"))
                        add(taskIdLabel)
                        add(Box.createHorizontalStrut(16))
                        add(stateBadge)
                    }
                add(taskPanel, BorderLayout.SOUTH)
            }
        mainPanel.add(headerPanel, BorderLayout.NORTH)

        // Center: Messages + Actions split
        val centerPanel =
            JSplitPane(JSplitPane.HORIZONTAL_SPLIT).apply {
                dividerLocation = 500
                resizeWeight = 0.75

                // Left: Messages area
                leftComponent =
                    JPanel(BorderLayout()).apply {
                        add(messagesScrollPane, BorderLayout.CENTER)

                        // Input row
                        val inputPanel =
                            JPanel(BorderLayout()).apply {
                                border = JBUI.Borders.emptyTop(8)
                                add(inputField, BorderLayout.CENTER)

                                val buttonPanel =
                                    JPanel(FlowLayout(FlowLayout.RIGHT, 4, 0)).apply {
                                        add(stopButton)
                                        add(sendButton)
                                    }
                                add(buttonPanel, BorderLayout.EAST)
                            }
                        add(inputPanel, BorderLayout.SOUTH)
                    }

                // Right: Actions panel
                rightComponent =
                    ActionsPanel { command, args ->
                        executeCommand(command, args)
                    }
            }
        mainPanel.add(centerPanel, BorderLayout.CENTER)

        setContent(mainPanel)

        // Initial message
        appendSystemMessage(
            "Welcome to Mehrhof Interactive Mode. Start server and use commands or buttons to control the workflow."
        )
    }

    private fun setupListeners() {
        service.addStateListener(this)

        // Start/Stop server
        startStopButton.addActionListener {
            if (service.isServerRunning()) {
                service.stopServer()
            } else {
                service.startServer()
            }
            updateConnectionStatus()
        }

        // Send button
        sendButton.addActionListener { sendInput() }

        // Stop button
        stopButton.addActionListener {
            scope.launch {
                val client = service.getApiClient() ?: return@launch
                withContext(Dispatchers.IO) {
                    client.stopOperation()
                }
                appendSystemMessage("Operation cancelled.")
                stopButton.isEnabled = false
            }
        }

        // Enter key in input field
        inputField.addKeyListener(
            object : KeyAdapter() {
                override fun keyPressed(e: KeyEvent) {
                    when (e.keyCode) {
                        KeyEvent.VK_ENTER -> {
                            sendInput()
                            e.consume()
                        }
                        KeyEvent.VK_UP -> {
                            navigateHistory(-1)
                            e.consume()
                        }
                        KeyEvent.VK_DOWN -> {
                            navigateHistory(1)
                            e.consume()
                        }
                    }
                }
            }
        )
    }

    private fun sendInput() {
        val text = inputField.text.trim()
        if (text.isEmpty()) return

        inputField.text = ""
        commandHistory.add(text)
        historyIndex = commandHistory.size

        // Parse input using CommandParser
        when (val parsed = CommandParser.parse(text)) {
            is ParsedInput.Command -> executeCommand(parsed.name, parsed.args)
            is ParsedInput.Answer -> answerQuestion(parsed.response)
            is ParsedInput.Chat -> sendChat(parsed.message)
            is ParsedInput.Help -> showHelp()
            is ParsedInput.Clear -> clearMessages()
            is ParsedInput.Error -> appendErrorMessage(parsed.message)
        }
    }

    private fun executeCommand(
        command: String,
        args: List<String>
    ) {
        if (!service.isConnected()) {
            appendErrorMessage("Not connected to server. Start server first.")
            return
        }

        appendCommandMessage("$command ${args.joinToString(" ")}".trim())
        stopButton.isEnabled = true

        scope.launch {
            val client = service.getApiClient() ?: return@launch

            val result =
                withContext(Dispatchers.IO) {
                    client.executeCommand(command, args)
                }

            result
                .onSuccess { response ->
                    if (response.success) {
                        response.message?.let { appendSystemMessage(it) }
                        response.state?.let { stateBadge.setState(it) }
                    } else {
                        appendErrorMessage(response.error ?: response.message ?: "Command failed")
                    }
                }.onFailure { error ->
                    appendErrorMessage("Error: ${error.message}")
                }

            stopButton.isEnabled = false
            service.refreshState()
        }
    }

    private fun sendChat(message: String) {
        if (!service.isConnected()) {
            appendErrorMessage("Not connected to server. Start server first.")
            return
        }

        appendUserMessage(message)
        stopButton.isEnabled = true

        scope.launch {
            val client = service.getApiClient() ?: return@launch

            val result =
                withContext(Dispatchers.IO) {
                    client.chat(message)
                }

            result
                .onSuccess { response ->
                    if (response.success) {
                        response.messages?.forEach { msg ->
                            when (msg.role) {
                                "assistant" -> appendAssistantMessage(msg.content)
                                "user" -> { /* Already displayed */ }
                                else -> appendSystemMessage(msg.content)
                            }
                        }
                    } else {
                        appendErrorMessage(response.error ?: response.message ?: "Chat failed")
                    }
                }.onFailure { error ->
                    appendErrorMessage("Error: ${error.message}")
                }

            stopButton.isEnabled = false
        }
    }

    private fun answerQuestion(response: String) {
        if (!service.isConnected()) {
            appendErrorMessage("Not connected to server. Start server first.")
            return
        }

        appendUserMessage("Answer: $response")

        scope.launch {
            val client = service.getApiClient() ?: return@launch

            val result =
                withContext(Dispatchers.IO) {
                    client.answerInteractive(response)
                }

            result
                .onSuccess { res ->
                    if (res.success) {
                        res.message?.let { appendSystemMessage(it) }
                    } else {
                        appendErrorMessage(res.error ?: res.message ?: "Answer failed")
                    }
                }.onFailure { error ->
                    appendErrorMessage("Error: ${error.message}")
                }

            service.refreshState()
        }
    }

    private fun navigateHistory(direction: Int) {
        if (commandHistory.isEmpty()) return

        historyIndex = (historyIndex + direction).coerceIn(0, commandHistory.size)
        inputField.text =
            if (historyIndex < commandHistory.size) {
                commandHistory[historyIndex]
            } else {
                ""
            }
    }

    // ========================================================================
    // Message Display
    // ========================================================================

    private fun appendUserMessage(text: String) {
        appendMessage("user", "You: $text")
    }

    private fun appendAssistantMessage(text: String) {
        appendMessage("assistant", "Agent: $text")
    }

    private fun appendSystemMessage(text: String) {
        appendMessage("system", text)
    }

    private fun appendErrorMessage(text: String) {
        appendMessage("error", "Error: $text")
    }

    private fun appendCommandMessage(text: String) {
        appendMessage("command", "> $text")
    }

    private fun appendMessage(
        cssClass: String,
        text: String
    ) {
        messagesContent.append(HtmlEscaper.formatMessage(cssClass, text))
        updateMessagesPane()
    }

    private fun updateMessagesPane() {
        messagesPane.text = messagesContent.toString() + "</body></html>"
        SwingUtilities.invokeLater {
            messagesPane.caretPosition = messagesPane.document.length
        }
    }

    private fun clearMessages() {
        messagesContent.clear()
        messagesContent.append("<html><body>")
        updateMessagesPane()
    }

    private fun showHelp() {
        appendSystemMessage(
            """
            Commands:
            - start <ref> - Start a task (e.g., start github:123)
            - plan - Run planning phase
            - implement - Run implementation phase
            - implement review <n> - Fix issues from review
            - review - Run code review
            - review <n> - View review content
            - continue - Resume from waiting
            - finish - Complete the task
            - abandon - Discard the task
            - undo/redo - Navigate checkpoints
            - status - Show task status
            - cost - Show token usage
            - budget - Show budget status
            - chat <msg> - Chat with agent
            - question <msg> - Ask agent a question
            - answer <resp> - Answer agent question
            - note <msg> - Add a note
            - find <query> - AI-powered code search
            - memory <query> - Search similar tasks
            - library [cmd] - Manage documentation library
            - specification [n] - View specifications
            - quick <desc> - Create quick task
            - simplify - Simplify code
            - label [add|rm] <labels> - Manage labels
            - clear - Clear messages
            - help - Show this help
            """.trimIndent()
        )
    }

    // ========================================================================
    // StateListener Implementation
    // ========================================================================

    override fun onConnectionChanged(connected: Boolean) {
        SwingUtilities.invokeLater {
            updateConnectionStatus()
            if (connected) {
                appendSystemMessage("Connected to server.")
                refreshInteractiveState()
            } else {
                appendSystemMessage("Disconnected from server.")
            }
        }
    }

    override fun onWorkflowStateChanged(
        state: String,
        previousState: String?
    ) {
        SwingUtilities.invokeLater {
            stateBadge.setState(state)
            if (previousState != null && state != previousState) {
                appendSystemMessage("State changed: $previousState -> $state")
            }
        }
    }

    override fun onTaskChanged(
        task: TaskInfo?,
        work: TaskWork?
    ) {
        SwingUtilities.invokeLater {
            updateTaskInfo()
        }
    }

    override fun onQuestionReceived(
        question: String,
        options: List<String>?
    ) {
        SwingUtilities.invokeLater {
            appendSystemMessage("Agent asks: $question")
            if (!options.isNullOrEmpty()) {
                appendSystemMessage("Options: ${options.joinToString(", ")}")
            }
            appendSystemMessage("Use 'answer <response>' to reply.")
        }
    }

    override fun onAgentMessage(
        content: String,
        type: String?
    ) {
        SwingUtilities.invokeLater {
            appendAssistantMessage(content)
        }
    }

    override fun onError(message: String) {
        SwingUtilities.invokeLater {
            appendErrorMessage(message)
        }
    }

    // ========================================================================
    // UI Updates
    // ========================================================================

    private fun updateConnectionStatus() {
        if (service.isServerRunning()) {
            startStopButton.text = "Stop Server"
            val port = service.getServerPort()
            connectionStatusLabel.text =
                if (service.isConnected()) {
                    "Connected (port $port)"
                } else {
                    "Starting..."
                }
            connectionStatusLabel.foreground =
                if (service.isConnected()) {
                    JBColor.GREEN.darker()
                } else {
                    JBColor.ORANGE
                }
        } else {
            startStopButton.text = "Start Server"
            connectionStatusLabel.text = "Not running"
            connectionStatusLabel.foreground = JBColor.GRAY
        }

        sendButton.isEnabled = service.isConnected()
    }

    private fun updateTaskInfo() {
        val task = service.currentTask
        val work = service.currentTaskWork

        if (task != null) {
            val title = work?.title ?: task.ref
            val shortId = task.id.take(7)
            taskIdLabel.text = "$shortId - $title"
        } else {
            taskIdLabel.text = "No active task"
        }

        stateBadge.setState(service.workflowState)
    }

    private fun refreshInteractiveState() {
        scope.launch {
            val client = service.getApiClient() ?: return@launch

            val result =
                withContext(Dispatchers.IO) {
                    client.getInteractiveState()
                }

            result.onSuccess { state ->
                state.state?.let { stateBadge.setState(it) }
                if (state.taskId != null) {
                    val title = state.title ?: state.taskId
                    taskIdLabel.text = "${state.taskId.take(7)} - $title"
                }
            }
        }
    }

    fun dispose() {
        service.removeStateListener(this)
        scope.cancel()
    }
}

/**
 * State badge component showing current workflow state.
 */
private class StateBadge : JPanel(FlowLayout(FlowLayout.CENTER, 4, 2)) {
    private val label = JBLabel("idle")

    init {
        border = JBUI.Borders.empty(2, 8)
        isOpaque = true
        add(label)
        setState("idle")
    }

    fun setState(state: String) {
        label.text = WorkflowUtils.formatState(state)
        background = WorkflowUtils.getStateBackground(state)
        label.foreground = WorkflowUtils.getStateForeground(state)
    }
}

/**
 * Actions panel with workflow buttons.
 */
private class ActionsPanel(
    private val onCommand: (String, List<String>) -> Unit
) : JPanel(BorderLayout()) {
    init {
        border = JBUI.Borders.empty(0, 8)
        preferredSize = Dimension(200, 0)

        val content =
            JPanel().apply {
                layout = BoxLayout(this, BoxLayout.Y_AXIS)
                border = JBUI.Borders.empty(8)
            }

        // Actions section
        content.add(createSectionLabel("Workflow"))
        content.add(Box.createVerticalStrut(8))
        content.add(createButton("Start Task...", "start"))
        content.add(createButton("Plan", "plan"))
        content.add(createButton("Implement", "implement"))
        content.add(createButton("Review", "review"))
        content.add(createButton("Continue", "continue"))
        content.add(Box.createVerticalStrut(8))
        content.add(createButton("Finish", "finish", JBColor.GREEN.darker()))
        content.add(createButton("Abandon", "abandon", JBColor.RED))
        content.add(Box.createVerticalStrut(16))

        // Checkpoints section
        content.add(createSectionLabel("Checkpoints"))
        content.add(Box.createVerticalStrut(8))
        content.add(createButton("Undo", "undo"))
        content.add(createButton("Redo", "redo"))
        content.add(Box.createVerticalStrut(16))

        // Info section
        content.add(createSectionLabel("Info"))
        content.add(Box.createVerticalStrut(8))
        content.add(createButton("Status", "status"))
        content.add(createButton("Cost", "cost"))
        content.add(createButton("Budget", "budget"))
        content.add(createButton("List Tasks", "list"))
        content.add(createButton("Specifications", "specification"))
        content.add(Box.createVerticalStrut(16))

        // Tools section
        content.add(createSectionLabel("Tools"))
        content.add(Box.createVerticalStrut(8))
        content.add(createButton("Find Code...", "find"))
        content.add(createButton("Search Memory...", "memory"))
        content.add(createButton("Library", "library"))
        content.add(createButton("Quick Task...", "quick"))
        content.add(createButton("Simplify", "simplify"))
        content.add(createButton("Add Note...", "note"))

        content.add(Box.createVerticalGlue())

        add(JBScrollPane(content), BorderLayout.CENTER)
    }

    private fun createSectionLabel(text: String): JComponent =
        JBLabel(text).apply {
            font = font.deriveFont(Font.BOLD, 12f)
            alignmentX = Component.LEFT_ALIGNMENT
        }

    private fun createButton(
        text: String,
        command: String,
        color: Color? = null
    ): JButton =
        JButton(text).apply {
            alignmentX = Component.LEFT_ALIGNMENT
            maximumSize = Dimension(Int.MAX_VALUE, preferredSize.height)
            color?.let { foreground = it }
            addActionListener { handleCommand(command) }
        }

    private fun handleCommand(command: String) {
        when (command) {
            "start" -> promptAndExecute("Enter task reference (e.g., github:123):", "Start Task", command)
            "find" -> promptAndExecute("Enter search query:", "Find Code", command)
            "memory" -> promptAndExecute("Enter search query:", "Search Memory", command)
            "quick" -> promptAndExecute("Enter task description:", "Create Quick Task", command)
            "note" -> promptAndExecute("Enter note:", "Add Note", command)
            "finish" -> confirmAndExecute("Complete this task?", command)
            "abandon" -> confirmAndExecute("Discard this task? This will delete the branch!", command)
            else -> onCommand(command, emptyList())
        }
    }

    private fun promptAndExecute(
        prompt: String,
        title: String,
        command: String
    ) {
        val input = JOptionPane.showInputDialog(this, prompt, title, JOptionPane.PLAIN_MESSAGE)
        if (!input.isNullOrBlank()) {
            onCommand(command, listOf(input))
        }
    }

    private fun confirmAndExecute(
        message: String,
        command: String
    ) {
        val result = JOptionPane.showConfirmDialog(this, message, "Confirm", JOptionPane.YES_NO_OPTION)
        if (result == JOptionPane.YES_OPTION) {
            onCommand(command, emptyList())
        }
    }
}
