package com.valksor.mehrhof.toolwindow

import com.intellij.icons.AllIcons
import com.intellij.openapi.actionSystem.ActionManager
import com.intellij.openapi.actionSystem.ActionPlaces
import com.intellij.openapi.actionSystem.DefaultActionGroup
import com.intellij.openapi.project.Project
import com.intellij.openapi.ui.SimpleToolWindowPanel
import com.intellij.ui.ColoredListCellRenderer
import com.intellij.ui.JBColor
import com.intellij.ui.SimpleTextAttributes
import com.intellij.ui.components.JBLabel
import com.intellij.ui.components.JBList
import com.intellij.ui.components.JBScrollPane
import com.intellij.util.ui.JBUI
import com.valksor.mehrhof.api.models.TaskInfo
import com.valksor.mehrhof.api.models.TaskSummary
import com.valksor.mehrhof.api.models.TaskWork
import com.valksor.mehrhof.services.MehrhofProjectService
import com.valksor.mehrhof.util.WorkflowButtonState
import com.valksor.mehrhof.util.WorkflowUtils
import kotlinx.coroutines.*
import java.awt.BorderLayout
import java.awt.FlowLayout
import javax.swing.*

/**
 * Main panel showing task list, current task details, and workflow controls.
 */
class TaskListPanel(
    @Suppress("unused") private val project: Project,
    private val service: MehrhofProjectService
) : SimpleToolWindowPanel(true, true),
    MehrhofProjectService.StateListener {
    private val scope = CoroutineScope(Dispatchers.Main + SupervisorJob())

    // UI components
    private val connectionStatusLabel = JBLabel()
    private val startStopButton = JButton("Start Server")
    private val currentTaskPanel = CurrentTaskPanel()
    private val taskListModel = DefaultListModel<TaskSummary>()
    private val taskList = JBList(taskListModel)
    private val workflowButtonsPanel = WorkflowButtonsPanel()

    init {
        setupToolbar()
        setupContent()
        setupListeners()

        // Initial UI update
        updateConnectionStatus()
        updateCurrentTask()
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

        // Connection status and server control at top
        val statusPanel =
            JPanel(FlowLayout(FlowLayout.LEFT)).apply {
                add(startStopButton)
                add(Box.createHorizontalStrut(16))
                add(JBLabel("Status: "))
                add(connectionStatusLabel)
            }
        mainPanel.add(statusPanel, BorderLayout.NORTH)

        // Center split: current task + task list
        val centerPanel =
            JPanel(BorderLayout()).apply {
                border = JBUI.Borders.emptyTop(8)
            }

        // Current task panel
        centerPanel.add(currentTaskPanel, BorderLayout.NORTH)

        // Task list
        taskList.cellRenderer = TaskListCellRenderer()
        taskList.selectionMode = ListSelectionModel.SINGLE_SELECTION
        // Note: Task selection is view-only; double-click could be added
        // to switch to a task in the future

        val listPanel =
            JPanel(BorderLayout()).apply {
                border = JBUI.Borders.emptyTop(8)
                add(JBLabel("Recent Tasks:"), BorderLayout.NORTH)
                add(JBScrollPane(taskList), BorderLayout.CENTER)
            }
        centerPanel.add(listPanel, BorderLayout.CENTER)

        mainPanel.add(centerPanel, BorderLayout.CENTER)

        // Workflow buttons at bottom
        mainPanel.add(workflowButtonsPanel, BorderLayout.SOUTH)

        setContent(mainPanel)
    }

    private fun setupListeners() {
        service.addStateListener(this)
        workflowButtonsPanel.setActionListener { action ->
            performWorkflowAction(action)
        }

        // Start/Stop server button
        startStopButton.addActionListener {
            if (service.isServerRunning()) {
                service.stopServer()
            } else {
                service.startServer()
            }
            updateConnectionStatus()
        }
    }

    // ========================================================================
    // StateListener Implementation
    // ========================================================================

    override fun onConnectionChanged(connected: Boolean) {
        SwingUtilities.invokeLater {
            updateConnectionStatus()
            if (connected) {
                refreshTaskList()
            }
        }
    }

    override fun onWorkflowStateChanged(
        state: String,
        previousState: String?
    ) {
        SwingUtilities.invokeLater {
            updateCurrentTask()
            workflowButtonsPanel.updateState(state)
        }
    }

    override fun onTaskChanged(
        task: TaskInfo?,
        work: TaskWork?
    ) {
        SwingUtilities.invokeLater {
            updateCurrentTask()
        }
    }

    override fun onQuestionReceived(
        question: String,
        options: List<String>?
    ) {
        SwingUtilities.invokeLater {
            showQuestionDialog(question, options)
        }
    }

    override fun onError(message: String) {
        SwingUtilities.invokeLater {
            connectionStatusLabel.text = "Error"
            connectionStatusLabel.foreground = JBColor.RED
        }
    }

    // ========================================================================
    // UI Updates
    // ========================================================================

    private fun updateConnectionStatus() {
        // Update start/stop button
        if (service.isServerRunning()) {
            startStopButton.text = "Stop Server"
            val port = service.getServerPort()
            if (port != null) {
                connectionStatusLabel.text = if (service.isConnected()) "Connected (port $port)" else "Starting..."
                connectionStatusLabel.foreground = if (service.isConnected()) JBColor.GREEN.darker() else JBColor.ORANGE
            }
        } else {
            startStopButton.text = "Start Server"
            connectionStatusLabel.text = "Not running"
            connectionStatusLabel.foreground = JBColor.GRAY
        }

        workflowButtonsPanel.setEnabled(service.isConnected())
    }

    private fun updateCurrentTask() {
        val task = service.currentTask
        val work = service.currentTaskWork
        val state = service.workflowState

        currentTaskPanel.update(task, work, state)
        workflowButtonsPanel.updateState(state)
    }

    private fun refreshTaskList() {
        scope.launch {
            val client = service.getApiClient() ?: return@launch

            withContext(Dispatchers.IO) {
                client.getTasks()
            }.onSuccess { response ->
                SwingUtilities.invokeLater {
                    taskListModel.clear()
                    response.tasks.forEach { taskListModel.addElement(it) }
                }
            }
        }
    }

    // ========================================================================
    // Workflow Actions
    // ========================================================================

    private fun performWorkflowAction(action: WorkflowAction) {
        scope.launch {
            val client = service.getApiClient() ?: return@launch

            val result =
                withContext(Dispatchers.IO) {
                    when (action) {
                        WorkflowAction.PLAN -> client.plan()
                        WorkflowAction.IMPLEMENT -> client.implement()
                        WorkflowAction.REVIEW -> client.review()
                        WorkflowAction.FINISH -> client.finish()
                        WorkflowAction.UNDO -> client.undo()
                        WorkflowAction.REDO -> client.redo()
                    }
                }

            result.onFailure { error ->
                JOptionPane.showMessageDialog(
                    this@TaskListPanel,
                    "Action failed: ${error.message}",
                    "Error",
                    JOptionPane.ERROR_MESSAGE
                )
            }

            // Refresh state after action
            service.refreshState()
        }
    }

    private fun showQuestionDialog(
        question: String,
        options: List<String>?
    ) {
        val answer =
            if (options != null && options.isNotEmpty()) {
                JOptionPane.showInputDialog(
                    this,
                    question,
                    "Agent Question",
                    JOptionPane.QUESTION_MESSAGE,
                    null,
                    options.toTypedArray(),
                    options.first()
                ) as? String
            } else {
                JOptionPane.showInputDialog(
                    this,
                    question,
                    "Agent Question",
                    JOptionPane.QUESTION_MESSAGE
                )
            }

        if (answer != null) {
            scope.launch {
                val client = service.getApiClient() ?: return@launch
                withContext(Dispatchers.IO) {
                    client.answer(answer)
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
 * Panel showing current task information.
 */
private class CurrentTaskPanel : JPanel(BorderLayout()) {
    private val titleLabel = JBLabel()
    private val stateLabel = JBLabel()
    private val refLabel = JBLabel()

    init {
        border =
            JBUI.Borders.compound(
                JBUI.Borders.customLine(JBColor.border(), 0, 0, 1, 0),
                JBUI.Borders.empty(8)
            )

        val infoPanel =
            JPanel().apply {
                layout = BoxLayout(this, BoxLayout.Y_AXIS)
                add(titleLabel)
                add(Box.createVerticalStrut(4))
                add(stateLabel)
                add(Box.createVerticalStrut(2))
                add(refLabel)
            }

        add(
            JBLabel("Current Task").apply {
                font = font.deriveFont(font.style or java.awt.Font.BOLD)
            },
            BorderLayout.NORTH
        )
        add(infoPanel, BorderLayout.CENTER)

        update(null, null, "idle")
    }

    fun update(
        task: TaskInfo?,
        work: TaskWork?,
        state: String
    ) {
        if (task == null) {
            titleLabel.text = "No active task"
            stateLabel.text = ""
            refLabel.text = ""
        } else {
            titleLabel.text = work?.title ?: task.ref
            stateLabel.text = "State: ${WorkflowUtils.formatState(state)}"
            stateLabel.foreground = WorkflowUtils.getStateColor(state)
            refLabel.text = "Ref: ${task.ref}"
            refLabel.foreground = JBColor.GRAY
        }
    }
}

/**
 * Panel with workflow action buttons.
 */
private class WorkflowButtonsPanel : JPanel(FlowLayout(FlowLayout.CENTER, 8, 8)) {
    private val planButton = JButton("Plan")
    private val implementButton = JButton("Implement")
    private val reviewButton = JButton("Review")
    private val finishButton = JButton("Finish")
    private val undoButton = JButton("Undo")
    private val redoButton = JButton("Redo")

    private var actionListener: ((WorkflowAction) -> Unit)? = null

    init {
        border = JBUI.Borders.emptyTop(8)

        add(planButton)
        add(implementButton)
        add(reviewButton)
        add(finishButton)
        add(undoButton)
        add(redoButton)

        planButton.addActionListener { actionListener?.invoke(WorkflowAction.PLAN) }
        implementButton.addActionListener { actionListener?.invoke(WorkflowAction.IMPLEMENT) }
        reviewButton.addActionListener { actionListener?.invoke(WorkflowAction.REVIEW) }
        finishButton.addActionListener { actionListener?.invoke(WorkflowAction.FINISH) }
        undoButton.addActionListener { actionListener?.invoke(WorkflowAction.UNDO) }
        redoButton.addActionListener { actionListener?.invoke(WorkflowAction.REDO) }

        updateState("idle")
    }

    fun setActionListener(listener: (WorkflowAction) -> Unit) {
        actionListener = listener
    }

    override fun setEnabled(enabled: Boolean) {
        super.setEnabled(enabled)
        components.filterIsInstance<JButton>().forEach { it.isEnabled = enabled }
    }

    fun updateState(state: String) {
        val buttonStates = WorkflowButtonState.getButtonStates(state)
        planButton.isEnabled = buttonStates.plan
        implementButton.isEnabled = buttonStates.implement
        reviewButton.isEnabled = buttonStates.review
        finishButton.isEnabled = buttonStates.finish
        undoButton.isEnabled = buttonStates.undo
        redoButton.isEnabled = buttonStates.redo
    }
}

enum class WorkflowAction {
    PLAN,
    IMPLEMENT,
    REVIEW,
    FINISH,
    UNDO,
    REDO
}

/**
 * Cell renderer for the task list.
 */
private class TaskListCellRenderer : ColoredListCellRenderer<TaskSummary>() {
    override fun customizeCellRenderer(
        list: JList<out TaskSummary>,
        value: TaskSummary?,
        index: Int,
        selected: Boolean,
        hasFocus: Boolean
    ) {
        value ?: return

        icon =
            when (value.state) {
                "done" -> AllIcons.RunConfigurations.TestPassed
                "failed" -> AllIcons.RunConfigurations.TestFailed
                "planning", "implementing", "reviewing" -> AllIcons.Process.Step_1
                else -> AllIcons.RunConfigurations.TestNotRan
            }

        append(value.title ?: value.id, SimpleTextAttributes.REGULAR_ATTRIBUTES)
        append(" ")
        append("[${value.state}]", SimpleTextAttributes.GRAYED_ATTRIBUTES)
    }
}
