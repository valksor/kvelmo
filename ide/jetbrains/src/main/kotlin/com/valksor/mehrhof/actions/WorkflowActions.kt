package com.valksor.mehrhof.actions

import com.intellij.openapi.actionSystem.ActionUpdateThread
import com.intellij.openapi.actionSystem.AnAction
import com.intellij.openapi.actionSystem.AnActionEvent
import com.intellij.openapi.progress.ProgressIndicator
import com.intellij.openapi.progress.ProgressManager
import com.intellij.openapi.progress.Task
import com.intellij.openapi.ui.Messages
import com.valksor.mehrhof.services.MehrhofProjectService

/**
 * Base class for Mehrhof workflow actions.
 */
abstract class MehrhofAction : AnAction() {
    override fun getActionUpdateThread(): ActionUpdateThread = ActionUpdateThread.BGT

    protected fun getService(e: AnActionEvent): MehrhofProjectService? {
        val project = e.project ?: return null
        return MehrhofProjectService.getInstance(project)
    }

    protected fun isConnected(e: AnActionEvent): Boolean {
        return getService(e)?.isConnected() == true
    }

    protected fun runInBackground(e: AnActionEvent, title: String, action: () -> Unit) {
        val project = e.project ?: return

        ProgressManager.getInstance().run(object : Task.Backgroundable(project, title, true) {
            override fun run(indicator: ProgressIndicator) {
                action()
            }
        })
    }

    protected fun showError(e: AnActionEvent, message: String) {
        Messages.showErrorDialog(e.project, message, "Mehrhof Error")
    }
}

// ============================================================================
// Connection Actions
// ============================================================================

class ConnectAction : MehrhofAction() {
    override fun actionPerformed(e: AnActionEvent) {
        getService(e)?.connect()
    }

    override fun update(e: AnActionEvent) {
        e.presentation.isEnabled = !isConnected(e)
        e.presentation.text = if (isConnected(e)) "Connected" else "Connect to Server"
    }
}

class DisconnectAction : MehrhofAction() {
    override fun actionPerformed(e: AnActionEvent) {
        getService(e)?.disconnect()
    }

    override fun update(e: AnActionEvent) {
        e.presentation.isEnabled = isConnected(e)
    }
}

// ============================================================================
// Workflow Actions
// ============================================================================

class StartTaskAction : MehrhofAction() {
    override fun actionPerformed(e: AnActionEvent) {
        val service = getService(e) ?: return
        val client = service.getApiClient() ?: return

        val ref = Messages.showInputDialog(
            e.project,
            "Enter task reference (e.g., github:123, file:task.md):",
            "Start Task",
            null
        ) ?: return

        if (ref.isBlank()) return

        runInBackground(e, "Starting task...") {
            client.executeCommand("start", listOf(ref)).onFailure { error ->
                showError(e, "Start failed: ${error.message}")
            }
            service.refreshState()
        }
    }

    override fun update(e: AnActionEvent) {
        e.presentation.isEnabled = isConnected(e)
    }
}

class PlanAction : MehrhofAction() {
    override fun actionPerformed(e: AnActionEvent) {
        val service = getService(e) ?: return
        val client = service.getApiClient() ?: return

        runInBackground(e, "Planning...") {
            client.plan().onFailure { error ->
                showError(e, "Plan failed: ${error.message}")
            }
            service.refreshState()
        }
    }

    override fun update(e: AnActionEvent) {
        e.presentation.isEnabled = isConnected(e)
    }
}

class ImplementAction : MehrhofAction() {
    override fun actionPerformed(e: AnActionEvent) {
        val service = getService(e) ?: return
        val client = service.getApiClient() ?: return

        runInBackground(e, "Implementing...") {
            client.implement().onFailure { error ->
                showError(e, "Implement failed: ${error.message}")
            }
            service.refreshState()
        }
    }

    override fun update(e: AnActionEvent) {
        e.presentation.isEnabled = isConnected(e)
    }
}

class ReviewAction : MehrhofAction() {
    override fun actionPerformed(e: AnActionEvent) {
        val service = getService(e) ?: return
        val client = service.getApiClient() ?: return

        runInBackground(e, "Reviewing...") {
            client.review().onFailure { error ->
                showError(e, "Review failed: ${error.message}")
            }
            service.refreshState()
        }
    }

    override fun update(e: AnActionEvent) {
        e.presentation.isEnabled = isConnected(e)
    }
}

class FinishAction : MehrhofAction() {
    override fun actionPerformed(e: AnActionEvent) {
        val service = getService(e) ?: return
        val client = service.getApiClient() ?: return

        runInBackground(e, "Finishing...") {
            client.finish().onFailure { error ->
                showError(e, "Finish failed: ${error.message}")
            }
            service.refreshState()
        }
    }

    override fun update(e: AnActionEvent) {
        e.presentation.isEnabled = isConnected(e)
    }
}

class ContinueAction : MehrhofAction() {
    override fun actionPerformed(e: AnActionEvent) {
        val service = getService(e) ?: return
        val client = service.getApiClient() ?: return

        runInBackground(e, "Continuing...") {
            client.continueWorkflow().onFailure { error ->
                showError(e, "Continue failed: ${error.message}")
            }
            service.refreshState()
        }
    }

    override fun update(e: AnActionEvent) {
        e.presentation.isEnabled = isConnected(e)
    }
}

class AbandonAction : MehrhofAction() {
    override fun actionPerformed(e: AnActionEvent) {
        val service = getService(e) ?: return
        val client = service.getApiClient() ?: return

        val result = Messages.showYesNoDialog(
            e.project,
            "Discard this task? This will delete the branch and work directory!",
            "Abandon Task",
            Messages.getWarningIcon()
        )

        if (result != Messages.YES) return

        runInBackground(e, "Abandoning...") {
            client.abandon().onFailure { error ->
                showError(e, "Abandon failed: ${error.message}")
            }
            service.refreshState()
        }
    }

    override fun update(e: AnActionEvent) {
        e.presentation.isEnabled = isConnected(e)
    }
}

// ============================================================================
// Checkpoint Actions
// ============================================================================

class UndoAction : MehrhofAction() {
    override fun actionPerformed(e: AnActionEvent) {
        val service = getService(e) ?: return
        val client = service.getApiClient() ?: return

        runInBackground(e, "Undoing...") {
            client.undo().onFailure { error ->
                showError(e, "Undo failed: ${error.message}")
            }
            service.refreshState()
        }
    }

    override fun update(e: AnActionEvent) {
        e.presentation.isEnabled = isConnected(e)
    }
}

class RedoAction : MehrhofAction() {
    override fun actionPerformed(e: AnActionEvent) {
        val service = getService(e) ?: return
        val client = service.getApiClient() ?: return

        runInBackground(e, "Redoing...") {
            client.redo().onFailure { error ->
                showError(e, "Redo failed: ${error.message}")
            }
            service.refreshState()
        }
    }

    override fun update(e: AnActionEvent) {
        e.presentation.isEnabled = isConnected(e)
    }
}

// ============================================================================
// Utility Actions
// ============================================================================

class RefreshAction : MehrhofAction() {
    override fun actionPerformed(e: AnActionEvent) {
        getService(e)?.refreshState()
    }

    override fun update(e: AnActionEvent) {
        e.presentation.isEnabled = isConnected(e)
    }
}
