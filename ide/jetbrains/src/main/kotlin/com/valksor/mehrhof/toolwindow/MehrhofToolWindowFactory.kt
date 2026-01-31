package com.valksor.mehrhof.toolwindow

import com.intellij.openapi.Disposable
import com.intellij.openapi.project.DumbAware
import com.intellij.openapi.project.Project
import com.intellij.openapi.wm.ToolWindow
import com.intellij.openapi.wm.ToolWindowFactory
import com.intellij.ui.content.ContentFactory
import com.valksor.mehrhof.services.MehrhofProjectService

/**
 * Factory for creating the Mehrhof tool window.
 */
class MehrhofToolWindowFactory :
    ToolWindowFactory,
    DumbAware {
    override fun createToolWindowContent(
        project: Project,
        toolWindow: ToolWindow
    ) {
        val service = MehrhofProjectService.getInstance(project)
        val contentFactory = ContentFactory.getInstance()

        // Interactive tab - main interactive terminal
        val interactivePanel = InteractivePanel(project, service)
        val interactiveContent = contentFactory.createContent(interactivePanel, "Interactive", false)
        interactiveContent.setDisposer(DisposableWrapper { interactivePanel.dispose() })
        toolWindow.contentManager.addContent(interactiveContent)

        // Tasks tab - task list view (secondary)
        val tasksPanel = TaskListPanel(project, service)
        val tasksContent = contentFactory.createContent(tasksPanel, "Tasks", false)
        tasksContent.setDisposer(DisposableWrapper { tasksPanel.dispose() })
        toolWindow.contentManager.addContent(tasksContent)

        // Output tab - agent output and logs
        val outputPanel = OutputPanel(project, service)
        val outputContent = contentFactory.createContent(outputPanel, "Output", false)
        outputContent.setDisposer(DisposableWrapper { outputPanel.dispose() })
        toolWindow.contentManager.addContent(outputContent)

        // Auto-connect when tool window is opened
        if (!service.isConnected()) {
            service.connect()
        }
    }

    override fun shouldBeAvailable(project: Project): Boolean = true
}

/**
 * Simple Disposable wrapper for cleanup callbacks.
 */
private class DisposableWrapper(
    private val onDispose: () -> Unit
) : Disposable {
    override fun dispose() {
        onDispose()
    }
}
