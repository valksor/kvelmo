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

    protected fun isConnected(e: AnActionEvent): Boolean = getService(e)?.isConnected() == true

    protected fun runInBackground(
        e: AnActionEvent,
        title: String,
        action: () -> Unit
    ) {
        val project = e.project ?: return

        ProgressManager.getInstance().run(
            object : Task.Backgroundable(project, title, true) {
                override fun run(indicator: ProgressIndicator) {
                    action()
                }
            }
        )
    }

    protected fun showError(
        e: AnActionEvent,
        message: String
    ) {
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

        val ref =
            Messages.showInputDialog(
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

        val result =
            Messages.showYesNoDialog(
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
                                appendLine("Cost: \$${String.format("%.4f", response.totalCostUsd)}")
                                appendLine(
                                    "Tokens: ${response.totalTokens} (${response.inputTokens} in, ${response.outputTokens} out)"
                                )
                                appendLine(
                                    "Cached: ${response.cachedTokens} (${response.cachedPercent?.let {
                                        String.format(
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
                                appendLine("Total Cost: \$${String.format("%.4f", total.costUsd)}")
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

// ============================================================================
// Memory Actions
// ============================================================================

class MemorySearchAction : MehrhofAction() {
    override fun actionPerformed(e: AnActionEvent) {
        val service = getService(e) ?: return
        val client = service.getApiClient() ?: return

        val query =
            Messages.showInputDialog(
                e.project,
                "Enter search query for similar tasks:",
                "Memory Search",
                null
            ) ?: return

        if (query.isBlank()) return

        runInBackground(e, "Searching memory...") {
            client
                .memorySearch(query)
                .onSuccess { response ->
                    if (response.count == 0) {
                        Messages.showInfoMessage(e.project, "No similar tasks found", "Mehrhof")
                    } else {
                        val message =
                            buildString {
                                appendLine("Found ${response.count} similar task(s):")
                                response.results.forEach { result ->
                                    val similarity = (result.score * 100).toInt()
                                    appendLine("• ${result.taskId} ($similarity% similar)")
                                }
                            }
                        Messages.showInfoMessage(e.project, message, "Memory Search Results")
                    }
                }.onFailure { error ->
                    showError(e, "Memory search failed: ${error.message}")
                }
        }
    }

    override fun update(e: AnActionEvent) {
        e.presentation.isEnabled = isConnected(e)
    }
}

class MemoryIndexAction : MehrhofAction() {
    override fun actionPerformed(e: AnActionEvent) {
        val service = getService(e) ?: return
        val client = service.getApiClient() ?: return

        val defaultTaskId = service.currentTask?.id ?: ""
        val taskId =
            Messages.showInputDialog(
                e.project,
                "Enter task ID to index:",
                "Memory Index Task",
                null,
                defaultTaskId,
                null
            ) ?: return

        if (taskId.isBlank()) return

        runInBackground(e, "Indexing task...") {
            client
                .memoryIndex(taskId)
                .onSuccess { response ->
                    if (response.success) {
                        Messages.showInfoMessage(
                            e.project,
                            response.message ?: "Task indexed successfully",
                            "Mehrhof"
                        )
                    } else {
                        showError(e, response.error ?: "Failed to index task")
                    }
                }.onFailure { error ->
                    showError(e, "Memory index failed: ${error.message}")
                }
        }
    }

    override fun update(e: AnActionEvent) {
        e.presentation.isEnabled = isConnected(e)
    }
}

class MemoryStatsAction : MehrhofAction() {
    override fun actionPerformed(e: AnActionEvent) {
        val service = getService(e) ?: return
        val client = service.getApiClient() ?: return

        runInBackground(e, "Fetching memory stats...") {
            client
                .memoryStats()
                .onSuccess { response ->
                    if (!response.enabled) {
                        Messages.showInfoMessage(e.project, "Memory system is not enabled", "Mehrhof")
                    } else {
                        val message =
                            buildString {
                                appendLine("Total documents: ${response.totalDocuments}")
                                if (response.byType.isNotEmpty()) {
                                    appendLine("\nBy type:")
                                    response.byType.forEach { (type, count) ->
                                        appendLine("  • $type: $count")
                                    }
                                }
                            }
                        Messages.showInfoMessage(e.project, message, "Memory Statistics")
                    }
                }.onFailure { error ->
                    showError(e, "Failed to fetch memory stats: ${error.message}")
                }
        }
    }

    override fun update(e: AnActionEvent) {
        e.presentation.isEnabled = isConnected(e)
    }
}

// ============================================================================
// Library Actions
// ============================================================================

class LibraryListAction : MehrhofAction() {
    override fun actionPerformed(e: AnActionEvent) {
        val service = getService(e) ?: return
        val client = service.getApiClient() ?: return

        runInBackground(e, "Fetching library...") {
            client
                .libraryList()
                .onSuccess { response ->
                    if (response.count == 0) {
                        Messages.showInfoMessage(e.project, "No collections in library", "Mehrhof")
                    } else {
                        val message =
                            buildString {
                                appendLine("${response.count} collection(s):")
                                response.collections.forEach { coll ->
                                    val size = formatBytes(coll.totalSize)
                                    appendLine("• ${coll.name} (${coll.pageCount} pages, $size)")
                                }
                            }
                        Messages.showInfoMessage(e.project, message, "Library Collections")
                    }
                }.onFailure { error ->
                    showError(e, "Failed to list library: ${error.message}")
                }
        }
    }

    override fun update(e: AnActionEvent) {
        e.presentation.isEnabled = isConnected(e)
    }

    private fun formatBytes(bytes: Long): String {
        if (bytes < 1024) return "$bytes B"
        val kb = bytes / 1024.0
        if (kb < 1024) return String.format("%.1f KB", kb)
        val mb = kb / 1024.0
        if (mb < 1024) return String.format("%.1f MB", mb)
        val gb = mb / 1024.0
        return String.format("%.1f GB", gb)
    }
}

class LibraryShowAction : MehrhofAction() {
    override fun actionPerformed(e: AnActionEvent) {
        val service = getService(e) ?: return
        val client = service.getApiClient() ?: return

        val nameOrId =
            Messages.showInputDialog(
                e.project,
                "Enter collection name or ID:",
                "Library Show",
                null
            ) ?: return

        if (nameOrId.isBlank()) return

        runInBackground(e, "Fetching collection...") {
            client
                .libraryShow(nameOrId)
                .onSuccess { response ->
                    val coll = response.collection
                    val message =
                        buildString {
                            appendLine("Name: ${coll.name}")
                            appendLine("Source: ${coll.source}")
                            appendLine("Type: ${coll.sourceType}")
                            appendLine("Mode: ${coll.includeMode}")
                            appendLine("Pages: ${coll.pageCount}")
                            appendLine("Size: ${formatBytes(coll.totalSize)}")
                            appendLine("Location: ${coll.location}")
                            coll.pulledAt?.let { appendLine("Pulled: $it") }
                            if (response.pages.isNotEmpty()) {
                                appendLine("\nPages (first 10):")
                                response.pages.take(10).forEach { page ->
                                    appendLine("  • $page")
                                }
                                if (response.pages.size > 10) {
                                    appendLine("  ... and ${response.pages.size - 10} more")
                                }
                            }
                        }
                    Messages.showInfoMessage(e.project, message, "Collection: ${coll.name}")
                }.onFailure { error ->
                    showError(e, "Failed to show collection: ${error.message}")
                }
        }
    }

    override fun update(e: AnActionEvent) {
        e.presentation.isEnabled = isConnected(e)
    }

    private fun formatBytes(bytes: Long): String {
        if (bytes < 1024) return "$bytes B"
        val kb = bytes / 1024.0
        if (kb < 1024) return String.format("%.1f KB", kb)
        val mb = kb / 1024.0
        if (mb < 1024) return String.format("%.1f MB", mb)
        val gb = mb / 1024.0
        return String.format("%.1f GB", gb)
    }
}

class LibraryPullAction : MehrhofAction() {
    override fun actionPerformed(e: AnActionEvent) {
        val service = getService(e) ?: return
        val client = service.getApiClient() ?: return

        val source =
            Messages.showInputDialog(
                e.project,
                "Enter source URL or path:",
                "Library Pull",
                null
            ) ?: return

        if (source.isBlank()) return

        val name =
            Messages.showInputDialog(
                e.project,
                "Collection name (leave empty for auto):",
                "Library Pull",
                null
            )

        val shared =
            Messages.showYesNoDialog(
                e.project,
                "Make collection shared (available to all projects)?",
                "Library Pull",
                Messages.getQuestionIcon()
            ) == Messages.YES

        runInBackground(e, "Pulling documentation...") {
            client
                .libraryPull(source, name?.takeIf { it.isNotBlank() }, shared)
                .onSuccess { response ->
                    if (response.success) {
                        Messages.showInfoMessage(e.project, response.message ?: "Collection pulled", "Mehrhof")
                    } else {
                        showError(e, response.error ?: "Failed to pull collection")
                    }
                }.onFailure { error ->
                    showError(e, "Pull failed: ${error.message}")
                }
        }
    }

    override fun update(e: AnActionEvent) {
        e.presentation.isEnabled = isConnected(e)
    }
}

class LibraryRemoveAction : MehrhofAction() {
    override fun actionPerformed(e: AnActionEvent) {
        val service = getService(e) ?: return
        val client = service.getApiClient() ?: return

        val nameOrId =
            Messages.showInputDialog(
                e.project,
                "Enter collection name or ID to remove:",
                "Library Remove",
                null
            ) ?: return

        if (nameOrId.isBlank()) return

        val confirm =
            Messages.showYesNoDialog(
                e.project,
                "Remove collection '$nameOrId'? This cannot be undone.",
                "Library Remove",
                Messages.getWarningIcon()
            )

        if (confirm != Messages.YES) return

        runInBackground(e, "Removing collection...") {
            client
                .libraryRemove(nameOrId)
                .onSuccess { response ->
                    if (response.success) {
                        Messages.showInfoMessage(e.project, response.message ?: "Collection removed", "Mehrhof")
                    } else {
                        showError(e, response.error ?: "Failed to remove collection")
                    }
                }.onFailure { error ->
                    showError(e, "Remove failed: ${error.message}")
                }
        }
    }

    override fun update(e: AnActionEvent) {
        e.presentation.isEnabled = isConnected(e)
    }
}

class LibraryStatsAction : MehrhofAction() {
    override fun actionPerformed(e: AnActionEvent) {
        val service = getService(e) ?: return
        val client = service.getApiClient() ?: return

        runInBackground(e, "Fetching library stats...") {
            client
                .libraryStats()
                .onSuccess { response ->
                    if (!response.enabled) {
                        Messages.showInfoMessage(e.project, "Library system is not enabled", "Mehrhof")
                    } else {
                        val message =
                            buildString {
                                appendLine("Collections: ${response.totalCollections}")
                                appendLine("Pages: ${response.totalPages}")
                                appendLine("Total size: ${formatBytes(response.totalSize)}")
                                appendLine("Project: ${response.projectCount}")
                                appendLine("Shared: ${response.sharedCount}")
                                if (response.byMode.isNotEmpty()) {
                                    appendLine("\nBy mode:")
                                    response.byMode.forEach { (mode, count) ->
                                        appendLine("  • $mode: $count")
                                    }
                                }
                            }
                        Messages.showInfoMessage(e.project, message, "Library Statistics")
                    }
                }.onFailure { error ->
                    showError(e, "Failed to fetch library stats: ${error.message}")
                }
        }
    }

    override fun update(e: AnActionEvent) {
        e.presentation.isEnabled = isConnected(e)
    }

    private fun formatBytes(bytes: Long): String {
        if (bytes < 1024) return "$bytes B"
        val kb = bytes / 1024.0
        if (kb < 1024) return String.format("%.1f KB", kb)
        val mb = kb / 1024.0
        if (mb < 1024) return String.format("%.1f MB", mb)
        val gb = mb / 1024.0
        return String.format("%.1f GB", gb)
    }
}

// ============================================================================
// Links Actions
// ============================================================================

class LinksListAction : MehrhofAction() {
    override fun actionPerformed(e: AnActionEvent) {
        val service = getService(e) ?: return
        val client = service.getApiClient() ?: return

        runInBackground(e, "Fetching links...") {
            client
                .linksList()
                .onSuccess { response ->
                    if (response.count == 0) {
                        Messages.showInfoMessage(e.project, "No links found", "Mehrhof")
                    } else {
                        val message =
                            buildString {
                                appendLine("${response.count} link(s):")
                                response.links.take(15).forEach { link ->
                                    appendLine("• ${link.source} → ${link.target}")
                                }
                                if (response.count > 15) {
                                    appendLine("... and ${response.count - 15} more")
                                }
                            }
                        Messages.showInfoMessage(e.project, message, "Links")
                    }
                }.onFailure { error ->
                    showError(e, "Failed to list links: ${error.message}")
                }
        }
    }

    override fun update(e: AnActionEvent) {
        e.presentation.isEnabled = isConnected(e)
    }
}

class LinksSearchAction : MehrhofAction() {
    override fun actionPerformed(e: AnActionEvent) {
        val service = getService(e) ?: return
        val client = service.getApiClient() ?: return

        val query =
            Messages.showInputDialog(
                e.project,
                "Enter search query (entity ID or name pattern):",
                "Links Search",
                null
            ) ?: return

        if (query.isBlank()) return

        runInBackground(e, "Searching links...") {
            client
                .linksSearch(query)
                .onSuccess { response ->
                    if (response.count == 0) {
                        Messages.showInfoMessage(e.project, "No entities found", "Mehrhof")
                    } else {
                        val message =
                            buildString {
                                appendLine("Found ${response.count} entity(ies):")
                                response.results.forEach { entity ->
                                    val links = entity.totalLinks?.let { " ($it links)" } ?: ""
                                    appendLine("• ${entity.entityId} [${entity.type}]$links")
                                }
                            }
                        Messages.showInfoMessage(e.project, message, "Links Search Results")
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

class LinksStatsAction : MehrhofAction() {
    override fun actionPerformed(e: AnActionEvent) {
        val service = getService(e) ?: return
        val client = service.getApiClient() ?: return

        runInBackground(e, "Fetching links stats...") {
            client
                .linksStats()
                .onSuccess { response ->
                    if (!response.enabled) {
                        Messages.showInfoMessage(e.project, "Links system is not enabled", "Mehrhof")
                    } else {
                        val message =
                            buildString {
                                appendLine("Total links: ${response.totalLinks}")
                                appendLine("Sources: ${response.totalSources}")
                                appendLine("Targets: ${response.totalTargets}")
                                appendLine("Orphans: ${response.orphanEntities}")
                                if (response.mostLinked.isNotEmpty()) {
                                    appendLine("\nMost linked:")
                                    response.mostLinked.take(5).forEach { entity ->
                                        val links = entity.totalLinks ?: 0
                                        appendLine("  • ${entity.entityId}: $links links")
                                    }
                                }
                            }
                        Messages.showInfoMessage(e.project, message, "Links Statistics")
                    }
                }.onFailure { error ->
                    showError(e, "Failed to fetch links stats: ${error.message}")
                }
        }
    }

    override fun update(e: AnActionEvent) {
        e.presentation.isEnabled = isConnected(e)
    }
}

class LinksRebuildAction : MehrhofAction() {
    override fun actionPerformed(e: AnActionEvent) {
        val service = getService(e) ?: return
        val client = service.getApiClient() ?: return

        val confirm =
            Messages.showYesNoDialog(
                e.project,
                "Rebuild links index? This will rescan all tasks.",
                "Links Rebuild",
                Messages.getQuestionIcon()
            )

        if (confirm != Messages.YES) return

        runInBackground(e, "Rebuilding links index...") {
            client
                .linksRebuild()
                .onSuccess { response ->
                    if (response.success) {
                        Messages.showInfoMessage(e.project, response.message ?: "Links index rebuilt", "Mehrhof")
                    } else {
                        showError(e, response.error ?: "Failed to rebuild links")
                    }
                }.onFailure { error ->
                    showError(e, "Rebuild failed: ${error.message}")
                }
        }
    }

    override fun update(e: AnActionEvent) {
        e.presentation.isEnabled = isConnected(e)
    }
}

// ============================================================================
// Browser Actions
// ============================================================================

class BrowserStatusAction : MehrhofAction() {
    override fun actionPerformed(e: AnActionEvent) {
        val service = getService(e) ?: return
        val client = service.getApiClient() ?: return

        runInBackground(e, "Checking browser status...") {
            client
                .browserStatus()
                .onSuccess { response ->
                    if (!response.connected) {
                        val error = response.error?.let { " ($it)" } ?: ""
                        Messages.showInfoMessage(e.project, "Browser: Not connected$error", "Mehrhof")
                    } else {
                        val message =
                            buildString {
                                appendLine("Browser: Connected")
                                appendLine("Host: ${response.host}:${response.port}")
                                appendLine("Tabs: ${response.tabs?.size ?: 0}")
                            }
                        Messages.showInfoMessage(e.project, message, "Browser Status")
                    }
                }.onFailure { error ->
                    showError(e, "Failed to get browser status: ${error.message}")
                }
        }
    }

    override fun update(e: AnActionEvent) {
        e.presentation.isEnabled = isConnected(e)
    }
}

class BrowserTabsAction : MehrhofAction() {
    override fun actionPerformed(e: AnActionEvent) {
        val service = getService(e) ?: return
        val client = service.getApiClient() ?: return

        runInBackground(e, "Fetching browser tabs...") {
            client
                .browserTabs()
                .onSuccess { response ->
                    if (response.count == 0) {
                        Messages.showInfoMessage(e.project, "No browser tabs open", "Mehrhof")
                    } else {
                        val message =
                            buildString {
                                appendLine("${response.count} tab(s):")
                                response.tabs.forEach { tab ->
                                    val url = truncateUrl(tab.url, 50)
                                    appendLine("• ${tab.title.take(30)} - $url")
                                }
                            }
                        Messages.showInfoMessage(e.project, message, "Browser Tabs")
                    }
                }.onFailure { error ->
                    showError(e, "Failed to list tabs: ${error.message}")
                }
        }
    }

    override fun update(e: AnActionEvent) {
        e.presentation.isEnabled = isConnected(e)
    }

    private fun truncateUrl(
        url: String,
        maxLen: Int
    ): String = if (url.length <= maxLen) url else url.take(maxLen - 3) + "..."
}

class BrowserGotoAction : MehrhofAction() {
    override fun actionPerformed(e: AnActionEvent) {
        val service = getService(e) ?: return
        val client = service.getApiClient() ?: return

        val url =
            Messages.showInputDialog(
                e.project,
                "Enter URL to open:",
                "Browser Go To",
                null
            ) ?: return

        if (url.isBlank()) return

        runInBackground(e, "Opening URL...") {
            client
                .browserGoto(url)
                .onSuccess { response ->
                    if (response.success && response.tab != null) {
                        Messages.showInfoMessage(
                            e.project,
                            "Opened: ${response.tab.title.take(50)}",
                            "Mehrhof"
                        )
                    }
                }.onFailure { error ->
                    showError(e, "Failed to open URL: ${error.message}")
                }
        }
    }

    override fun update(e: AnActionEvent) {
        e.presentation.isEnabled = isConnected(e)
    }
}

class BrowserNavigateAction : MehrhofAction() {
    override fun actionPerformed(e: AnActionEvent) {
        val service = getService(e) ?: return
        val client = service.getApiClient() ?: return

        val url =
            Messages.showInputDialog(
                e.project,
                "Enter URL to navigate current tab to:",
                "Browser Navigate",
                null
            ) ?: return

        if (url.isBlank()) return

        runInBackground(e, "Navigating...") {
            client
                .browserNavigate(url)
                .onSuccess { response ->
                    if (response.success) {
                        Messages.showInfoMessage(e.project, response.message ?: "Navigated", "Mehrhof")
                    }
                }.onFailure { error ->
                    showError(e, "Navigation failed: ${error.message}")
                }
        }
    }

    override fun update(e: AnActionEvent) {
        e.presentation.isEnabled = isConnected(e)
    }
}

class BrowserReloadAction : MehrhofAction() {
    override fun actionPerformed(e: AnActionEvent) {
        val service = getService(e) ?: return
        val client = service.getApiClient() ?: return

        runInBackground(e, "Reloading page...") {
            client
                .browserReload()
                .onSuccess { response ->
                    if (response.success) {
                        Messages.showInfoMessage(e.project, response.message ?: "Page reloaded", "Mehrhof")
                    }
                }.onFailure { error ->
                    showError(e, "Reload failed: ${error.message}")
                }
        }
    }

    override fun update(e: AnActionEvent) {
        e.presentation.isEnabled = isConnected(e)
    }
}

class BrowserScreenshotAction : MehrhofAction() {
    override fun actionPerformed(e: AnActionEvent) {
        val service = getService(e) ?: return
        val client = service.getApiClient() ?: return

        runInBackground(e, "Taking screenshot...") {
            client
                .browserScreenshot()
                .onSuccess { response ->
                    if (response.success && response.data != null) {
                        val sizeKb = (response.size ?: 0) / 1024
                        Messages.showInfoMessage(
                            e.project,
                            "Screenshot captured: ${response.format ?: "png"}, $sizeKb KB",
                            "Mehrhof"
                        )
                    } else {
                        showError(e, "Screenshot failed")
                    }
                }.onFailure { error ->
                    showError(e, "Screenshot failed: ${error.message}")
                }
        }
    }

    override fun update(e: AnActionEvent) {
        e.presentation.isEnabled = isConnected(e)
    }
}

class BrowserClickAction : MehrhofAction() {
    override fun actionPerformed(e: AnActionEvent) {
        val service = getService(e) ?: return
        val client = service.getApiClient() ?: return

        val selector =
            Messages.showInputDialog(
                e.project,
                "Enter CSS selector to click:",
                "Browser Click",
                null
            ) ?: return

        if (selector.isBlank()) return

        runInBackground(e, "Clicking element...") {
            client
                .browserClick(selector)
                .onSuccess { response ->
                    if (response.success) {
                        Messages.showInfoMessage(
                            e.project,
                            "Clicked: ${response.selector ?: selector}",
                            "Mehrhof"
                        )
                    }
                }.onFailure { error ->
                    showError(e, "Click failed: ${error.message}")
                }
        }
    }

    override fun update(e: AnActionEvent) {
        e.presentation.isEnabled = isConnected(e)
    }
}

class BrowserTypeAction : MehrhofAction() {
    override fun actionPerformed(e: AnActionEvent) {
        val service = getService(e) ?: return
        val client = service.getApiClient() ?: return

        val selector =
            Messages.showInputDialog(
                e.project,
                "Enter CSS selector for input element:",
                "Browser Type",
                null
            ) ?: return

        if (selector.isBlank()) return

        val text =
            Messages.showInputDialog(
                e.project,
                "Enter text to type:",
                "Browser Type",
                null
            ) ?: return

        runInBackground(e, "Typing...") {
            client
                .browserType(selector, text)
                .onSuccess { response ->
                    if (response.success) {
                        Messages.showInfoMessage(
                            e.project,
                            "Typed into: ${response.selector ?: selector}",
                            "Mehrhof"
                        )
                    }
                }.onFailure { error ->
                    showError(e, "Type failed: ${error.message}")
                }
        }
    }

    override fun update(e: AnActionEvent) {
        e.presentation.isEnabled = isConnected(e)
    }
}

class BrowserEvalAction : MehrhofAction() {
    override fun actionPerformed(e: AnActionEvent) {
        val service = getService(e) ?: return
        val client = service.getApiClient() ?: return

        val expression =
            Messages.showInputDialog(
                e.project,
                "Enter JavaScript expression to evaluate:",
                "Browser Eval",
                null
            ) ?: return

        if (expression.isBlank()) return

        runInBackground(e, "Evaluating...") {
            client
                .browserEval(expression)
                .onSuccess { response ->
                    if (response.success) {
                        Messages.showInfoMessage(
                            e.project,
                            "Result: ${response.result}",
                            "Browser Eval"
                        )
                    }
                }.onFailure { error ->
                    showError(e, "Eval failed: ${error.message}")
                }
        }
    }

    override fun update(e: AnActionEvent) {
        e.presentation.isEnabled = isConnected(e)
    }
}

class BrowserConsoleAction : MehrhofAction() {
    override fun actionPerformed(e: AnActionEvent) {
        val service = getService(e) ?: return
        val client = service.getApiClient() ?: return

        runInBackground(e, "Fetching console logs...") {
            client
                .browserConsole()
                .onSuccess { response ->
                    val messages = response.messages
                    if (messages.isNullOrEmpty()) {
                        Messages.showInfoMessage(e.project, "No console messages", "Mehrhof")
                    } else {
                        val message =
                            buildString {
                                appendLine("${messages.size} message(s):")
                                messages.take(20).forEach { msg ->
                                    appendLine("[${msg.level.uppercase()}] ${msg.text.take(80)}")
                                }
                                if (messages.size > 20) {
                                    appendLine("... and ${messages.size - 20} more")
                                }
                            }
                        Messages.showInfoMessage(e.project, message, "Browser Console")
                    }
                }.onFailure { error ->
                    showError(e, "Failed to fetch console: ${error.message}")
                }
        }
    }

    override fun update(e: AnActionEvent) {
        e.presentation.isEnabled = isConnected(e)
    }
}

class BrowserNetworkAction : MehrhofAction() {
    override fun actionPerformed(e: AnActionEvent) {
        val service = getService(e) ?: return
        val client = service.getApiClient() ?: return

        runInBackground(e, "Fetching network requests...") {
            client
                .browserNetwork()
                .onSuccess { response ->
                    val requests = response.requests
                    if (requests.isNullOrEmpty()) {
                        Messages.showInfoMessage(e.project, "No network requests", "Mehrhof")
                    } else {
                        val message =
                            buildString {
                                appendLine("${requests.size} request(s):")
                                requests.take(15).forEach { req ->
                                    val status = req.status?.toString() ?: "..."
                                    val url = truncateUrl(req.url, 50)
                                    appendLine("${req.method} $status - $url")
                                }
                                if (requests.size > 15) {
                                    appendLine("... and ${requests.size - 15} more")
                                }
                            }
                        Messages.showInfoMessage(e.project, message, "Browser Network")
                    }
                }.onFailure { error ->
                    showError(e, "Failed to fetch network: ${error.message}")
                }
        }
    }

    override fun update(e: AnActionEvent) {
        e.presentation.isEnabled = isConnected(e)
    }

    private fun truncateUrl(
        url: String,
        maxLen: Int
    ): String = if (url.length <= maxLen) url else url.take(maxLen - 3) + "..."
}

// ============================================================================
// Project Actions
// ============================================================================

class ProjectPlanAction : MehrhofAction() {
    override fun actionPerformed(e: AnActionEvent) {
        val service = getService(e) ?: return
        val client = service.getApiClient() ?: return

        val source =
            Messages.showInputDialog(
                e.project,
                "Enter source (file path, URL, or GitHub issue reference):",
                "Project Plan",
                null
            ) ?: return

        if (source.isBlank()) return

        runInBackground(e, "Creating project plan...") {
            client
                .executeCommand("project", listOf("plan", source))
                .onSuccess { response ->
                    if (response.success) {
                        Messages.showInfoMessage(e.project, response.message ?: "Project plan created", "Mehrhof")
                    } else {
                        showError(e, response.error ?: "Failed to create project plan")
                    }
                }.onFailure { error ->
                    showError(e, "Project plan failed: ${error.message}")
                }
        }
    }

    override fun update(e: AnActionEvent) {
        e.presentation.isEnabled = isConnected(e)
    }
}

class ProjectTasksAction : MehrhofAction() {
    override fun actionPerformed(e: AnActionEvent) {
        val service = getService(e) ?: return
        val client = service.getApiClient() ?: return

        runInBackground(e, "Fetching project tasks...") {
            client
                .executeCommand("project", listOf("tasks"))
                .onSuccess { response ->
                    if (response.success) {
                        Messages.showInfoMessage(e.project, response.message ?: "Project tasks retrieved", "Mehrhof")
                    } else {
                        showError(e, response.error ?: "Failed to get project tasks")
                    }
                }.onFailure { error ->
                    showError(e, "Project tasks failed: ${error.message}")
                }
        }
    }

    override fun update(e: AnActionEvent) {
        e.presentation.isEnabled = isConnected(e)
    }
}

class ProjectEditAction : MehrhofAction() {
    override fun actionPerformed(e: AnActionEvent) {
        val service = getService(e) ?: return
        val client = service.getApiClient() ?: return

        val taskId =
            Messages.showInputDialog(
                e.project,
                "Enter task ID to edit:",
                "Project Edit Task",
                null
            ) ?: return

        if (taskId.isBlank()) return

        val fields = arrayOf("title", "priority", "status")
        val field =
            Messages.showEditableChooseDialog(
                "Select field to edit:",
                "Project Edit Task",
                Messages.getQuestionIcon(),
                fields,
                fields[0],
                null
            ) ?: return

        val value =
            Messages.showInputDialog(
                e.project,
                "Enter new $field:",
                "Project Edit Task",
                null
            ) ?: return

        if (value.isBlank()) return

        runInBackground(e, "Updating project task...") {
            client
                .executeCommand("project", listOf("edit", taskId, "--$field", value))
                .onSuccess { response ->
                    if (response.success) {
                        Messages.showInfoMessage(e.project, response.message ?: "Task updated", "Mehrhof")
                    } else {
                        showError(e, response.error ?: "Failed to update task")
                    }
                }.onFailure { error ->
                    showError(e, "Edit task failed: ${error.message}")
                }
        }
    }

    override fun update(e: AnActionEvent) {
        e.presentation.isEnabled = isConnected(e)
    }
}

class ProjectSubmitAction : MehrhofAction() {
    override fun actionPerformed(e: AnActionEvent) {
        val service = getService(e) ?: return
        val client = service.getApiClient() ?: return

        val providers = arrayOf("github", "gitlab", "linear", "jira")
        val provider =
            Messages.showEditableChooseDialog(
                "Select provider to submit tasks to:",
                "Project Submit",
                Messages.getQuestionIcon(),
                providers,
                providers[0],
                null
            ) ?: return

        runInBackground(e, "Submitting project tasks...") {
            client
                .executeCommand("project", listOf("submit", provider))
                .onSuccess { response ->
                    if (response.success) {
                        Messages.showInfoMessage(e.project, response.message ?: "Tasks submitted", "Mehrhof")
                    } else {
                        showError(e, response.error ?: "Failed to submit tasks")
                    }
                }.onFailure { error ->
                    showError(e, "Submit tasks failed: ${error.message}")
                }
        }
    }

    override fun update(e: AnActionEvent) {
        e.presentation.isEnabled = isConnected(e)
    }
}

class ProjectStartAction : MehrhofAction() {
    override fun actionPerformed(e: AnActionEvent) {
        val service = getService(e) ?: return
        val client = service.getApiClient() ?: return

        runInBackground(e, "Starting next project task...") {
            client
                .executeCommand("project", listOf("start"))
                .onSuccess { response ->
                    if (response.success) {
                        Messages.showInfoMessage(e.project, response.message ?: "Started next task", "Mehrhof")
                    } else {
                        showError(e, response.error ?: "Failed to start next task")
                    }
                }.onFailure { error ->
                    showError(e, "Project start failed: ${error.message}")
                }
            service.refreshState()
        }
    }

    override fun update(e: AnActionEvent) {
        e.presentation.isEnabled = isConnected(e)
    }
}

class ProjectSyncAction : MehrhofAction() {
    override fun actionPerformed(e: AnActionEvent) {
        val service = getService(e) ?: return
        val client = service.getApiClient() ?: return

        val reference =
            Messages.showInputDialog(
                e.project,
                "Enter provider reference to sync from:",
                "Project Sync",
                null
            ) ?: return

        if (reference.isBlank()) return

        runInBackground(e, "Syncing project...") {
            client
                .executeCommand("project", listOf("sync", reference))
                .onSuccess { response ->
                    if (response.success) {
                        Messages.showInfoMessage(e.project, response.message ?: "Project synced", "Mehrhof")
                    } else {
                        showError(e, response.error ?: "Failed to sync project")
                    }
                }.onFailure { error ->
                    showError(e, "Project sync failed: ${error.message}")
                }
        }
    }

    override fun update(e: AnActionEvent) {
        e.presentation.isEnabled = isConnected(e)
    }
}

// ============================================================================
// Stack Actions
// ============================================================================

class StackListAction : MehrhofAction() {
    override fun actionPerformed(e: AnActionEvent) {
        val service = getService(e) ?: return
        val client = service.getApiClient() ?: return

        runInBackground(e, "Fetching stacks...") {
            client
                .executeCommand("stack", listOf("list"))
                .onSuccess { response ->
                    if (response.success) {
                        Messages.showInfoMessage(e.project, response.message ?: "Stacks retrieved", "Mehrhof")
                    } else {
                        showError(e, response.error ?: "Failed to list stacks")
                    }
                }.onFailure { error ->
                    showError(e, "Stack list failed: ${error.message}")
                }
        }
    }

    override fun update(e: AnActionEvent) {
        e.presentation.isEnabled = isConnected(e)
    }
}

class StackRebaseAction : MehrhofAction() {
    override fun actionPerformed(e: AnActionEvent) {
        val service = getService(e) ?: return
        val client = service.getApiClient() ?: return

        val taskId =
            Messages.showInputDialog(
                e.project,
                "Enter task ID to rebase (leave empty to rebase all):",
                "Stack Rebase",
                null
            )

        val args = if (taskId.isNullOrBlank()) listOf("rebase") else listOf("rebase", taskId)

        runInBackground(e, "Rebasing stack...") {
            client
                .executeCommand("stack", args)
                .onSuccess { response ->
                    if (response.success) {
                        Messages.showInfoMessage(e.project, response.message ?: "Stack rebased", "Mehrhof")
                    } else {
                        showError(e, response.error ?: "Failed to rebase stack")
                    }
                }.onFailure { error ->
                    showError(e, "Stack rebase failed: ${error.message}")
                }
        }
    }

    override fun update(e: AnActionEvent) {
        e.presentation.isEnabled = isConnected(e)
    }
}

class StackSyncAction : MehrhofAction() {
    override fun actionPerformed(e: AnActionEvent) {
        val service = getService(e) ?: return
        val client = service.getApiClient() ?: return

        runInBackground(e, "Syncing stacks...") {
            client
                .executeCommand("stack", listOf("sync"))
                .onSuccess { response ->
                    if (response.success) {
                        Messages.showInfoMessage(e.project, response.message ?: "Stacks synced", "Mehrhof")
                    } else {
                        showError(e, response.error ?: "Failed to sync stacks")
                    }
                }.onFailure { error ->
                    showError(e, "Stack sync failed: ${error.message}")
                }
        }
    }

    override fun update(e: AnActionEvent) {
        e.presentation.isEnabled = isConnected(e)
    }
}

// ============================================================================
// Configuration Actions
// ============================================================================

class ConfigValidateAction : MehrhofAction() {
    override fun actionPerformed(e: AnActionEvent) {
        val service = getService(e) ?: return
        val client = service.getApiClient() ?: return

        runInBackground(e, "Validating configuration...") {
            client
                .executeCommand("config", listOf("validate"))
                .onSuccess { response ->
                    if (response.success) {
                        Messages.showInfoMessage(e.project, response.message ?: "Configuration valid", "Mehrhof")
                    } else {
                        showError(e, response.error ?: "Configuration validation failed")
                    }
                }.onFailure { error ->
                    showError(e, "Config validate failed: ${error.message}")
                }
        }
    }

    override fun update(e: AnActionEvent) {
        e.presentation.isEnabled = isConnected(e)
    }
}

class AgentsListAction : MehrhofAction() {
    override fun actionPerformed(e: AnActionEvent) {
        val service = getService(e) ?: return
        val client = service.getApiClient() ?: return

        runInBackground(e, "Listing agents...") {
            client
                .executeCommand("agents", listOf("list"))
                .onSuccess { response ->
                    if (response.success) {
                        Messages.showInfoMessage(e.project, response.message ?: "Agents retrieved", "Mehrhof")
                    } else {
                        showError(e, response.error ?: "Failed to list agents")
                    }
                }.onFailure { error ->
                    showError(e, "Agents list failed: ${error.message}")
                }
        }
    }

    override fun update(e: AnActionEvent) {
        e.presentation.isEnabled = isConnected(e)
    }
}

class AgentsExplainAction : MehrhofAction() {
    override fun actionPerformed(e: AnActionEvent) {
        val service = getService(e) ?: return
        val client = service.getApiClient() ?: return

        val name =
            Messages.showInputDialog(
                e.project,
                "Enter agent name to explain:",
                "Agents Explain",
                null
            ) ?: return

        if (name.isBlank()) return

        runInBackground(e, "Getting agent info...") {
            client
                .executeCommand("agents", listOf("explain", name))
                .onSuccess { response ->
                    if (response.success) {
                        Messages.showInfoMessage(e.project, response.message ?: "Agent info retrieved", "Mehrhof")
                    } else {
                        showError(e, response.error ?: "Failed to explain agent")
                    }
                }.onFailure { error ->
                    showError(e, "Agents explain failed: ${error.message}")
                }
        }
    }

    override fun update(e: AnActionEvent) {
        e.presentation.isEnabled = isConnected(e)
    }
}

class ProvidersListAction : MehrhofAction() {
    override fun actionPerformed(e: AnActionEvent) {
        val service = getService(e) ?: return
        val client = service.getApiClient() ?: return

        runInBackground(e, "Listing providers...") {
            client
                .executeCommand("providers", listOf("list"))
                .onSuccess { response ->
                    if (response.success) {
                        Messages.showInfoMessage(e.project, response.message ?: "Providers retrieved", "Mehrhof")
                    } else {
                        showError(e, response.error ?: "Failed to list providers")
                    }
                }.onFailure { error ->
                    showError(e, "Providers list failed: ${error.message}")
                }
        }
    }

    override fun update(e: AnActionEvent) {
        e.presentation.isEnabled = isConnected(e)
    }
}

class ProvidersInfoAction : MehrhofAction() {
    override fun actionPerformed(e: AnActionEvent) {
        val service = getService(e) ?: return
        val client = service.getApiClient() ?: return

        val name =
            Messages.showInputDialog(
                e.project,
                "Enter provider name (github, jira, linear, etc.):",
                "Providers Info",
                null
            ) ?: return

        if (name.isBlank()) return

        runInBackground(e, "Getting provider info...") {
            client
                .executeCommand("providers", listOf("info", name))
                .onSuccess { response ->
                    if (response.success) {
                        Messages.showInfoMessage(e.project, response.message ?: "Provider info retrieved", "Mehrhof")
                    } else {
                        showError(e, response.error ?: "Failed to get provider info")
                    }
                }.onFailure { error ->
                    showError(e, "Providers info failed: ${error.message}")
                }
        }
    }

    override fun update(e: AnActionEvent) {
        e.presentation.isEnabled = isConnected(e)
    }
}

class TemplatesListAction : MehrhofAction() {
    override fun actionPerformed(e: AnActionEvent) {
        val service = getService(e) ?: return
        val client = service.getApiClient() ?: return

        runInBackground(e, "Listing templates...") {
            client
                .executeCommand("templates", listOf("list"))
                .onSuccess { response ->
                    if (response.success) {
                        Messages.showInfoMessage(e.project, response.message ?: "Templates retrieved", "Mehrhof")
                    } else {
                        showError(e, response.error ?: "Failed to list templates")
                    }
                }.onFailure { error ->
                    showError(e, "Templates list failed: ${error.message}")
                }
        }
    }

    override fun update(e: AnActionEvent) {
        e.presentation.isEnabled = isConnected(e)
    }
}

class TemplatesShowAction : MehrhofAction() {
    override fun actionPerformed(e: AnActionEvent) {
        val service = getService(e) ?: return
        val client = service.getApiClient() ?: return

        val name =
            Messages.showInputDialog(
                e.project,
                "Enter template name (bug-fix, feature, refactor, etc.):",
                "Templates Show",
                null
            ) ?: return

        if (name.isBlank()) return

        runInBackground(e, "Getting template...") {
            client
                .executeCommand("templates", listOf("show", name))
                .onSuccess { response ->
                    if (response.success) {
                        Messages.showInfoMessage(e.project, response.message ?: "Template retrieved", "Mehrhof")
                    } else {
                        showError(e, response.error ?: "Failed to get template")
                    }
                }.onFailure { error ->
                    showError(e, "Templates show failed: ${error.message}")
                }
        }
    }

    override fun update(e: AnActionEvent) {
        e.presentation.isEnabled = isConnected(e)
    }
}

class ScanAction : MehrhofAction() {
    override fun actionPerformed(e: AnActionEvent) {
        val service = getService(e) ?: return
        val client = service.getApiClient() ?: return

        runInBackground(e, "Running security scan...") {
            client
                .executeCommand("scan", emptyList())
                .onSuccess { response ->
                    if (response.success) {
                        Messages.showInfoMessage(e.project, response.message ?: "Scan complete", "Mehrhof")
                    } else {
                        showError(e, response.error ?: "Scan failed")
                    }
                }.onFailure { error ->
                    showError(e, "Scan failed: ${error.message}")
                }
        }
    }

    override fun update(e: AnActionEvent) {
        e.presentation.isEnabled = isConnected(e)
    }
}

class CommitAction : MehrhofAction() {
    override fun actionPerformed(e: AnActionEvent) {
        val service = getService(e) ?: return
        val client = service.getApiClient() ?: return

        runInBackground(e, "Running commit analysis...") {
            client
                .executeCommand("commit", emptyList())
                .onSuccess { response ->
                    if (response.success) {
                        Messages.showInfoMessage(e.project, response.message ?: "Commit analysis complete", "Mehrhof")
                    } else {
                        showError(e, response.error ?: "Commit analysis failed")
                    }
                }.onFailure { error ->
                    showError(e, "Commit analysis failed: ${error.message}")
                }
        }
    }

    override fun update(e: AnActionEvent) {
        e.presentation.isEnabled = isConnected(e)
    }
}
