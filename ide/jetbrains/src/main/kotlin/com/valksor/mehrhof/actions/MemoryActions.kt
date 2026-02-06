package com.valksor.mehrhof.actions

import com.intellij.openapi.actionSystem.AnActionEvent
import com.intellij.openapi.ui.Messages
import com.valksor.mehrhof.api.*

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
