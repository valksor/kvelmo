package com.valksor.mehrhof.actions

import com.intellij.openapi.actionSystem.AnActionEvent
import com.intellij.openapi.ui.Messages
import com.valksor.mehrhof.api.*

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
