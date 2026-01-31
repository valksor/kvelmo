package com.valksor.mehrhof.statusbar

import com.intellij.openapi.project.Project
import com.intellij.openapi.wm.StatusBar
import com.intellij.openapi.wm.StatusBarWidget
import com.intellij.openapi.wm.StatusBarWidgetFactory
import com.intellij.util.Consumer
import com.valksor.mehrhof.api.models.TaskInfo
import com.valksor.mehrhof.api.models.TaskWork
import com.valksor.mehrhof.services.MehrhofProjectService
import java.awt.event.MouseEvent
import javax.swing.SwingUtilities

/**
 * Factory for creating Mehrhof status bar widgets.
 */
class MehrhofStatusWidgetFactory : StatusBarWidgetFactory {
    override fun getId(): String = "MehrhofStatusWidget"

    override fun getDisplayName(): String = "Mehrhof Status"

    override fun isAvailable(project: Project): Boolean = true

    override fun createWidget(project: Project): StatusBarWidget {
        return MehrhofStatusWidget(project)
    }

    override fun disposeWidget(widget: StatusBarWidget) {
        (widget as? MehrhofStatusWidget)?.dispose()
    }

    override fun canBeEnabledOn(statusBar: StatusBar): Boolean = true
}

/**
 * Status bar widget showing current Mehrhof connection and workflow state.
 */
class MehrhofStatusWidget(
    private val project: Project
) : StatusBarWidget, StatusBarWidget.TextPresentation, MehrhofProjectService.StateListener {

    private var statusBar: StatusBar? = null
    private val service = MehrhofProjectService.getInstance(project)

    init {
        service.addStateListener(this)
    }

    override fun ID(): String = "MehrhofStatusWidget"

    override fun install(statusBar: StatusBar) {
        this.statusBar = statusBar
    }

    override fun getPresentation(): StatusBarWidget.WidgetPresentation = this

    override fun getText(): String {
        return if (!service.isConnected()) {
            "Mehrhof: Disconnected"
        } else {
            val state = formatState(service.workflowState)
            val task = service.currentTaskWork?.title ?: service.currentTask?.ref
            if (task != null) {
                "Mehrhof: $state - ${truncate(task, 20)}"
            } else {
                "Mehrhof: $state"
            }
        }
    }

    override fun getTooltipText(): String {
        return if (!service.isConnected()) {
            "Click to connect to Mehrhof server"
        } else {
            buildString {
                append("State: ${service.workflowState}\n")
                service.currentTask?.let { task ->
                    append("Task: ${task.id}\n")
                    append("Ref: ${task.ref}\n")
                }
                service.currentTaskWork?.let { work ->
                    work.title?.let { append("Title: $it\n") }
                }
                service.pendingQuestion?.let { q ->
                    append("\nPending question: $q")
                }
            }
        }
    }

    override fun getAlignment(): Float = 0f

    override fun getClickConsumer(): Consumer<MouseEvent> = Consumer { _ ->
        if (!service.isConnected()) {
            service.connect()
        } else {
            // Toggle tool window or show quick actions popup
            // For now, just refresh state
            service.refreshState()
        }
    }

    override fun dispose() {
        service.removeStateListener(this)
        statusBar = null
    }

    // StateListener callbacks
    override fun onConnectionChanged(connected: Boolean) {
        updateWidget()
    }

    override fun onWorkflowStateChanged(state: String, previousState: String?) {
        updateWidget()
    }

    override fun onTaskChanged(task: TaskInfo?, work: TaskWork?) {
        updateWidget()
    }

    private fun updateWidget() {
        SwingUtilities.invokeLater {
            statusBar?.updateWidget(ID())
        }
    }

    private fun formatState(state: String): String {
        return state.replace("_", " ").replaceFirstChar { it.uppercase() }
    }

    private fun truncate(text: String, maxLength: Int): String {
        return if (text.length <= maxLength) text
        else text.take(maxLength - 3) + "..."
    }
}
