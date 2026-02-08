package com.valksor.mehrhof.actions

import com.intellij.openapi.actionSystem.AnActionEvent
import com.intellij.openapi.ui.Messages
import com.valksor.mehrhof.api.*
import java.util.Locale

// ============================================================================
// Auto Workflow Action
// ============================================================================

class AutoAction : MehrhofAction() {
    override fun actionPerformed(e: AnActionEvent) {
        val service = getService(e) ?: return
        val client = service.getApiClient() ?: return

        val loopsStr =
            Messages.showInputDialog(
                e.project,
                "Enter number of loops (0 for continuous, or leave empty):",
                "Auto Workflow",
                null
            )

        val loops = loopsStr?.toIntOrNull() ?: 0

        runInBackground(e, "Running auto workflow...") {
            client
                .auto(loops)
                .onSuccess { response ->
                    if (response.success) {
                        Messages.showInfoMessage(
                            e.project,
                            response.message ?: "Auto workflow completed",
                            "Mehrhof"
                        )
                    } else {
                        showError(e, response.error ?: "Auto workflow failed")
                    }
                }.onFailure { error ->
                    showError(e, "Auto workflow failed: ${error.message}")
                }
            service.refreshState()
        }
    }

    override fun update(e: AnActionEvent) {
        e.presentation.isEnabled = isConnected(e)
    }
}

// ============================================================================
// Budget Actions
// ============================================================================

class BudgetStatusAction : MehrhofAction() {
    override fun actionPerformed(e: AnActionEvent) {
        val service = getService(e) ?: return
        val client = service.getApiClient() ?: return

        runInBackground(e, "Fetching budget status...") {
            client
                .budgetStatus()
                .onSuccess { response ->
                    val message =
                        if (response.enabled) {
                            buildString {
                                appendLine("Monthly Budget Status")
                                appendLine("─".repeat(30))
                                appendLine(
                                    "Spent: ${response.currency ?: "USD"} ${
                                        String.format(
                                            Locale.US,
                                            "%.2f",
                                            response.spent ?: 0.0
                                        )
                                    }"
                                )
                                appendLine(
                                    "Limit: ${response.currency ?: "USD"} ${
                                        String.format(
                                            Locale.US,
                                            "%.2f",
                                            response.maxCost ?: 0.0
                                        )
                                    }"
                                )
                                appendLine(
                                    "Remaining: ${response.currency ?: "USD"} ${
                                        String.format(
                                            Locale.US,
                                            "%.2f",
                                            response.remaining ?: 0.0
                                        )
                                    }"
                                )
                                if (response.limitHit == true) {
                                    appendLine("\n⚠️ Budget limit reached!")
                                } else if (response.warned == true) {
                                    appendLine("\n⚠️ Warning threshold reached")
                                }
                            }
                        } else {
                            "Monthly budget is not enabled.\n\nEnable it in .mehrhof/config.yaml:\n  budget:\n    enabled: true\n    monthly:\n      max_cost: 50.00"
                        }
                    Messages.showInfoMessage(e.project, message, "Budget Status")
                }.onFailure { error ->
                    showError(e, "Failed to fetch budget: ${error.message}")
                }
        }
    }

    override fun update(e: AnActionEvent) {
        e.presentation.isEnabled = isConnected(e)
    }
}

class BudgetResetAction : MehrhofAction() {
    override fun actionPerformed(e: AnActionEvent) {
        val service = getService(e) ?: return
        val client = service.getApiClient() ?: return

        val confirm =
            Messages.showYesNoDialog(
                e.project,
                "Reset the monthly budget spending counter?\n\nThis will set the spent amount back to zero.",
                "Reset Budget",
                Messages.getQuestionIcon()
            )

        if (confirm != Messages.YES) return

        runInBackground(e, "Resetting budget...") {
            client
                .budgetReset()
                .onSuccess { response ->
                    if (response.success) {
                        Messages.showInfoMessage(
                            e.project,
                            response.message ?: "Budget reset successfully",
                            "Mehrhof"
                        )
                    } else {
                        showError(e, response.error ?: "Failed to reset budget")
                    }
                }.onFailure { error ->
                    showError(e, "Reset budget failed: ${error.message}")
                }
        }
    }

    override fun update(e: AnActionEvent) {
        e.presentation.isEnabled = isConnected(e)
    }
}

// ============================================================================
// Simplify Action
// ============================================================================

class SimplifyAction : MehrhofAction() {
    override fun actionPerformed(e: AnActionEvent) {
        val service = getService(e) ?: return
        val client = service.getApiClient() ?: return

        val path =
            Messages.showInputDialog(
                e.project,
                "Enter file or directory path to simplify (leave empty for current task):",
                "Simplify Code",
                null
            )

        val instructions =
            Messages.showInputDialog(
                e.project,
                "Enter simplification instructions (optional):",
                "Simplify Code",
                null
            )

        runInBackground(e, "Simplifying code...") {
            client
                .simplify(
                    path = path?.takeIf { it.isNotBlank() },
                    instructions = instructions?.takeIf { it.isNotBlank() }
                ).onSuccess { response ->
                    if (response.success) {
                        Messages.showInfoMessage(
                            e.project,
                            response.message ?: "Code simplification completed",
                            "Mehrhof"
                        )
                    } else {
                        showError(e, response.error ?: "Simplification failed")
                    }
                }.onFailure { error ->
                    showError(e, "Simplify failed: ${error.message}")
                }
            service.refreshState()
        }
    }

    override fun update(e: AnActionEvent) {
        e.presentation.isEnabled = isConnected(e)
    }
}

// ============================================================================
// Label Actions
// ============================================================================

class LabelListAction : MehrhofAction() {
    override fun actionPerformed(e: AnActionEvent) {
        val service = getService(e) ?: return
        val client = service.getApiClient() ?: return

        runInBackground(e, "Fetching labels...") {
            client
                .labelsList()
                .onSuccess { response ->
                    val message =
                        if (response.labels.isEmpty()) {
                            "No labels on current task"
                        } else {
                            buildString {
                                appendLine("Labels (${response.count}):")
                                response.labels.forEach { label ->
                                    appendLine("  • $label")
                                }
                            }
                        }
                    Messages.showInfoMessage(e.project, message, "Task Labels")
                }.onFailure { error ->
                    showError(e, "Failed to fetch labels: ${error.message}")
                }
        }
    }

    override fun update(e: AnActionEvent) {
        e.presentation.isEnabled = isConnected(e)
    }
}

class LabelAddAction : MehrhofAction() {
    override fun actionPerformed(e: AnActionEvent) {
        val service = getService(e) ?: return
        val client = service.getApiClient() ?: return

        val label =
            Messages.showInputDialog(
                e.project,
                "Enter label to add:",
                "Add Label",
                null
            ) ?: return

        if (label.isBlank()) return

        runInBackground(e, "Adding label...") {
            client
                .labelsAdd(label)
                .onSuccess { response ->
                    if (response.success) {
                        Messages.showInfoMessage(
                            e.project,
                            response.message ?: "Label '$label' added",
                            "Mehrhof"
                        )
                    } else {
                        showError(e, response.error ?: "Failed to add label")
                    }
                }.onFailure { error ->
                    showError(e, "Add label failed: ${error.message}")
                }
        }
    }

    override fun update(e: AnActionEvent) {
        e.presentation.isEnabled = isConnected(e)
    }
}

class LabelRemoveAction : MehrhofAction() {
    override fun actionPerformed(e: AnActionEvent) {
        val service = getService(e) ?: return
        val client = service.getApiClient() ?: return

        val label =
            Messages.showInputDialog(
                e.project,
                "Enter label to remove:",
                "Remove Label",
                null
            ) ?: return

        if (label.isBlank()) return

        runInBackground(e, "Removing label...") {
            client
                .labelsRemove(label)
                .onSuccess { response ->
                    if (response.success) {
                        Messages.showInfoMessage(
                            e.project,
                            response.message ?: "Label '$label' removed",
                            "Mehrhof"
                        )
                    } else {
                        showError(e, response.error ?: "Failed to remove label")
                    }
                }.onFailure { error ->
                    showError(e, "Remove label failed: ${error.message}")
                }
        }
    }

    override fun update(e: AnActionEvent) {
        e.presentation.isEnabled = isConnected(e)
    }
}
