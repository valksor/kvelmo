package com.valksor.mehrhof.actions

import com.intellij.openapi.actionSystem.AnActionEvent
import com.intellij.openapi.ui.Messages
import com.valksor.mehrhof.api.*
import java.util.Locale

// ============================================================================
// Library Actions
// ============================================================================

/** Shared byte formatting utility for library actions. */
private object ByteFormatter {
    fun format(bytes: Long): String {
        if (bytes < 1024) return "$bytes B"
        val kb = bytes / 1024.0
        if (kb < 1024) return String.format(Locale.US, "%.1f KB", kb)
        val mb = kb / 1024.0
        if (mb < 1024) return String.format(Locale.US, "%.1f MB", mb)
        val gb = mb / 1024.0
        return String.format(Locale.US, "%.1f GB", gb)
    }
}

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
                                    val size = ByteFormatter.format(coll.totalSize)
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
                            appendLine("Size: ${ByteFormatter.format(coll.totalSize)}")
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
                                appendLine("Total size: ${ByteFormatter.format(response.totalSize)}")
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
}
