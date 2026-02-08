package com.valksor.mehrhof.actions

import com.intellij.openapi.actionSystem.AnActionEvent
import com.intellij.openapi.ui.Messages
import com.valksor.mehrhof.api.*
import java.util.Locale

// ============================================================================
// Note & Question Actions
// ============================================================================

class NoteAction : MehrhofAction() {
    override fun actionPerformed(e: AnActionEvent) {
        val service = getService(e) ?: return
        val client = service.getApiClient() ?: return
        val taskId =
            service.currentTask?.id ?: run {
                showError(e, "No active task")
                return
            }

        val message =
            Messages.showInputDialog(
                e.project,
                "Enter note message:",
                "Add Note",
                null
            ) ?: return

        if (message.isBlank()) return

        runInBackground(e, "Adding note...") {
            client
                .addNote(taskId, message)
                .onSuccess { response ->
                    if (response.success) {
                        Messages.showInfoMessage(
                            e.project,
                            "Note #${response.noteNumber ?: "N/A"} added",
                            "Mehrhof"
                        )
                    } else {
                        showError(e, response.error ?: "Failed to add note")
                    }
                }.onFailure { error ->
                    showError(e, "Add note failed: ${error.message}")
                }
        }
    }

    override fun update(e: AnActionEvent) {
        e.presentation.isEnabled = isConnected(e)
    }
}

class QuestionAction : MehrhofAction() {
    override fun actionPerformed(e: AnActionEvent) {
        val service = getService(e) ?: return
        val client = service.getApiClient() ?: return

        val message =
            Messages.showInputDialog(
                e.project,
                "Enter question for the agent:",
                "Ask Question",
                null
            ) ?: return

        if (message.isBlank()) return

        runInBackground(e, "Asking question...") {
            client.question(message).onFailure { error ->
                showError(e, "Question failed: ${error.message}")
            }
            service.refreshState()
        }
    }

    override fun update(e: AnActionEvent) {
        e.presentation.isEnabled = isConnected(e)
    }
}

class ResetAction : MehrhofAction() {
    override fun actionPerformed(e: AnActionEvent) {
        val service = getService(e) ?: return
        val client = service.getApiClient() ?: return

        val result =
            Messages.showYesNoDialog(
                e.project,
                "Reset workflow state to idle? This will not lose your work.",
                "Reset Workflow",
                Messages.getQuestionIcon()
            )

        if (result != Messages.YES) return

        runInBackground(e, "Resetting...") {
            client
                .reset()
                .onSuccess {
                    Messages.showInfoMessage(e.project, "Workflow reset to idle", "Mehrhof")
                }.onFailure { error ->
                    showError(e, "Reset failed: ${error.message}")
                }
            service.refreshState()
        }
    }

    override fun update(e: AnActionEvent) {
        e.presentation.isEnabled = isConnected(e)
    }
}

class CostAction : MehrhofAction() {
    override fun actionPerformed(e: AnActionEvent) {
        val service = getService(e) ?: return
        val client = service.getApiClient() ?: return
        val taskId = service.currentTask?.id

        runInBackground(e, "Fetching costs...") {
            if (taskId != null) {
                // Show task-specific costs
                client
                    .getTaskCosts(taskId)
                    .onSuccess { response ->
                        val message =
                            buildString {
                                appendLine("Task: ${response.title ?: taskId}")
                                appendLine("Cost: \$${String.format(Locale.US, "%.4f", response.totalCostUsd)}")
                                appendLine(
                                    "Tokens: ${response.totalTokens} (${response.inputTokens} in, ${response.outputTokens} out)"
                                )
                                appendLine(
                                    "Cached: ${response.cachedTokens} (${response.cachedPercent?.let {
                                        String.format(
                                            Locale.US,
                                            "%.1f",
                                            it
                                        )
                                    } ?: "0"}%)"
                                )
                            }
                        Messages.showInfoMessage(e.project, message, "Task Costs")
                    }.onFailure { error ->
                        showError(e, "Failed to fetch costs: ${error.message}")
                    }
            } else {
                // Show all costs
                client
                    .getAllCosts()
                    .onSuccess { response ->
                        val total = response.grandTotal
                        val message =
                            buildString {
                                appendLine("Total Cost: \$${String.format(Locale.US, "%.4f", total.costUsd)}")
                                val totalIn = total.inputTokens
                                val totalOut = total.outputTokens
                                appendLine("Tokens: ${total.totalTokens} ($totalIn in, $totalOut out)")
                                appendLine("Cached: ${total.cachedTokens}")
                            }
                        Messages.showInfoMessage(e.project, message, "All Costs")
                    }.onFailure { error ->
                        showError(e, "Failed to fetch costs: ${error.message}")
                    }
            }
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

// ============================================================================
// Queue Task Actions
// ============================================================================

class QuickTaskAction : MehrhofAction() {
    override fun actionPerformed(e: AnActionEvent) {
        val service = getService(e) ?: return
        val client = service.getApiClient() ?: return

        val description =
            Messages.showInputDialog(
                e.project,
                "Enter task description:",
                "Quick Task",
                null
            ) ?: return

        if (description.isBlank()) return

        runInBackground(e, "Creating quick task...") {
            client
                .createQuickTask(description)
                .onSuccess { response ->
                    if (response.success) {
                        Messages.showInfoMessage(e.project, response.message ?: "Task created", "Mehrhof")
                    } else {
                        showError(e, response.error ?: "Failed to create task")
                    }
                }.onFailure { error ->
                    showError(e, "Create task failed: ${error.message}")
                }
        }
    }

    override fun update(e: AnActionEvent) {
        e.presentation.isEnabled = isConnected(e)
    }
}

class DeleteQueueTaskAction : MehrhofAction() {
    override fun actionPerformed(e: AnActionEvent) {
        val service = getService(e) ?: return
        val client = service.getApiClient() ?: return

        val taskRef =
            Messages.showInputDialog(
                e.project,
                "Enter task reference (queue/task-id):",
                "Delete Queue Task",
                null
            ) ?: return

        if (taskRef.isBlank()) return

        val parts = taskRef.split("/")
        if (parts.size != 2) {
            showError(e, "Invalid task reference format (expected: queue/task-id)")
            return
        }

        val confirm =
            Messages.showYesNoDialog(
                e.project,
                "Delete task $taskRef? This cannot be undone.",
                "Delete Queue Task",
                Messages.getWarningIcon()
            )

        if (confirm != Messages.YES) return

        runInBackground(e, "Deleting task...") {
            client
                .deleteQueueTask(parts[0], parts[1])
                .onSuccess { response ->
                    if (response.success) {
                        Messages.showInfoMessage(e.project, response.message ?: "Task deleted", "Mehrhof")
                    } else {
                        showError(e, response.error ?: "Failed to delete task")
                    }
                }.onFailure { error ->
                    showError(e, "Delete task failed: ${error.message}")
                }
        }
    }

    override fun update(e: AnActionEvent) {
        e.presentation.isEnabled = isConnected(e)
    }
}

class ExportQueueTaskAction : MehrhofAction() {
    override fun actionPerformed(e: AnActionEvent) {
        val service = getService(e) ?: return
        val client = service.getApiClient() ?: return

        val taskRef =
            Messages.showInputDialog(
                e.project,
                "Enter task reference (queue/task-id):",
                "Export Queue Task",
                null
            ) ?: return

        if (taskRef.isBlank()) return

        val parts = taskRef.split("/")
        if (parts.size != 2) {
            showError(e, "Invalid task reference format (expected: queue/task-id)")
            return
        }

        runInBackground(e, "Exporting task...") {
            client
                .exportQueueTask(parts[0], parts[1])
                .onSuccess { response ->
                    if (response.success) {
                        Messages.showInfoMessage(e.project, response.message ?: "Task exported", "Mehrhof")
                    } else {
                        showError(e, response.error ?: "Failed to export task")
                    }
                }.onFailure { error ->
                    showError(e, "Export task failed: ${error.message}")
                }
        }
    }

    override fun update(e: AnActionEvent) {
        e.presentation.isEnabled = isConnected(e)
    }
}

class OptimizeQueueTaskAction : MehrhofAction() {
    override fun actionPerformed(e: AnActionEvent) {
        val service = getService(e) ?: return
        val client = service.getApiClient() ?: return

        val taskRef =
            Messages.showInputDialog(
                e.project,
                "Enter task reference (queue/task-id):",
                "Optimize Queue Task",
                null
            ) ?: return

        if (taskRef.isBlank()) return

        val parts = taskRef.split("/")
        if (parts.size != 2) {
            showError(e, "Invalid task reference format (expected: queue/task-id)")
            return
        }

        runInBackground(e, "Optimizing task with AI...") {
            client
                .optimizeQueueTask(parts[0], parts[1])
                .onSuccess { response ->
                    if (response.success) {
                        Messages.showInfoMessage(e.project, response.message ?: "Task optimized", "Mehrhof")
                    } else {
                        showError(e, response.error ?: "Failed to optimize task")
                    }
                }.onFailure { error ->
                    showError(e, "Optimize task failed: ${error.message}")
                }
        }
    }

    override fun update(e: AnActionEvent) {
        e.presentation.isEnabled = isConnected(e)
    }
}

class SubmitQueueTaskAction : MehrhofAction() {
    override fun actionPerformed(e: AnActionEvent) {
        val service = getService(e) ?: return
        val client = service.getApiClient() ?: return

        val taskRef =
            Messages.showInputDialog(
                e.project,
                "Enter task reference (queue/task-id):",
                "Submit Queue Task",
                null
            ) ?: return

        if (taskRef.isBlank()) return

        val parts = taskRef.split("/")
        if (parts.size != 2) {
            showError(e, "Invalid task reference format (expected: queue/task-id)")
            return
        }

        val provider =
            Messages.showInputDialog(
                e.project,
                "Enter provider name (github, jira, wrike, etc.):",
                "Submit Queue Task",
                null
            ) ?: return

        if (provider.isBlank()) return

        runInBackground(e, "Submitting task...") {
            client
                .submitQueueTask(parts[0], parts[1], provider)
                .onSuccess { response ->
                    if (response.success) {
                        val msg =
                            buildString {
                                append(response.message ?: "Task submitted")
                            }
                        Messages.showInfoMessage(e.project, msg, "Mehrhof")
                    } else {
                        showError(e, response.error ?: "Failed to submit task")
                    }
                }.onFailure { error ->
                    showError(e, "Submit task failed: ${error.message}")
                }
        }
    }

    override fun update(e: AnActionEvent) {
        e.presentation.isEnabled = isConnected(e)
    }
}

class SyncTaskAction : MehrhofAction() {
    override fun actionPerformed(e: AnActionEvent) {
        val service = getService(e) ?: return
        val client = service.getApiClient() ?: return
        val task = service.currentTask

        if (task == null) {
            showError(e, "No active task to sync")
            return
        }

        runInBackground(e, "Syncing task...") {
            client
                .syncTask()
                .onSuccess { response ->
                    if (response.success) {
                        Messages.showInfoMessage(e.project, response.message ?: "Task synced", "Mehrhof")
                    } else {
                        showError(e, response.error ?: "Failed to sync task")
                    }
                }.onFailure { error ->
                    showError(e, "Sync task failed: ${error.message}")
                }
        }
    }

    override fun update(e: AnActionEvent) {
        e.presentation.isEnabled = isConnected(e)
    }
}

// ============================================================================
// Find & Search Actions
// ============================================================================

class FindAction : MehrhofAction() {
    override fun actionPerformed(e: AnActionEvent) {
        val service = getService(e) ?: return
        val client = service.getApiClient() ?: return

        val query =
            Messages.showInputDialog(
                e.project,
                "Enter search query (regex supported):",
                "Find in Codebase",
                null
            ) ?: return

        if (query.isBlank()) return

        runInBackground(e, "Searching codebase...") {
            client
                .find(query)
                .onSuccess { response ->
                    if (response.count == 0) {
                        Messages.showInfoMessage(e.project, "No matches found", "Mehrhof")
                    } else {
                        val message =
                            buildString {
                                appendLine("Found ${response.count} match(es):")
                                response.matches.take(10).forEach { match ->
                                    appendLine("• ${match.file}:${match.line} - ${match.reason ?: match.snippet}")
                                }
                                if (response.count > 10) {
                                    appendLine("... and ${response.count - 10} more")
                                }
                            }
                        Messages.showInfoMessage(e.project, message, "Search Results")
                    }
                }.onFailure { error ->
                    showError(e, "Search failed: ${error.message}")
                }
        }
    }

    override fun update(e: AnActionEvent) {
        e.presentation.isEnabled = isConnected(e)
    }
}
