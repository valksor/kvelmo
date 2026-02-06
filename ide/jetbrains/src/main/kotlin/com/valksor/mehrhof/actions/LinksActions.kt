package com.valksor.mehrhof.actions

import com.intellij.openapi.actionSystem.AnActionEvent
import com.intellij.openapi.ui.Messages
import com.valksor.mehrhof.api.*

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
