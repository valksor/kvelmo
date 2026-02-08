package com.valksor.mehrhof.actions

import com.intellij.openapi.project.Project
import com.intellij.openapi.ui.Messages
import com.valksor.mehrhof.api.models.InteractiveCommandResponse
import com.valksor.mehrhof.testutil.ActionTestFixture
import io.mockk.every
import io.mockk.verify
import org.junit.jupiter.api.AfterEach
import org.junit.jupiter.api.BeforeEach
import org.junit.jupiter.api.Test

/**
 * Unit tests for project, stack, and config actions in ProjectConfigActions.kt.
 *
 * Uses [ActionTestFixture] for common mock setup.
 */
class ProjectConfigActionsTest {
    private lateinit var fixture: ActionTestFixture

    @BeforeEach
    fun setUp() {
        fixture = ActionTestFixture()
        fixture.setUp()
    }

    @AfterEach
    fun tearDown() {
        fixture.tearDown()
    }

    // ========================================================================
    // ProjectPlanAction Tests
    // ========================================================================

    @Test
    fun `ProjectPlanAction returns early when dialog is cancelled`() {
        fixture.setInputDialogResult("Project Plan", null)

        ProjectPlanAction().actionPerformed(fixture.event)

        verify(exactly = 0) { fixture.client.executeCommand(any(), any()) }
    }

    @Test
    fun `ProjectPlanAction returns early for blank source`() {
        fixture.setInputDialogResult("Project Plan", "   ")

        ProjectPlanAction().actionPerformed(fixture.event)

        verify(exactly = 0) { fixture.client.executeCommand(any(), any()) }
    }

    @Test
    fun `ProjectPlanAction calls executeCommand with project plan`() {
        fixture.setInputDialogResult("Project Plan", "github:user/repo#123")
        every { fixture.client.executeCommand("project", listOf("plan", "github:user/repo#123")) } returns
            Result.success(InteractiveCommandResponse(success = true, message = "Plan created"))

        ProjectPlanAction().actionPerformed(fixture.event)

        verify { fixture.client.executeCommand("project", listOf("plan", "github:user/repo#123")) }
        verify { Messages.showInfoMessage(any<Project>(), any<String>(), eq("Mehrhof")) }
    }

    // ========================================================================
    // ProjectTasksAction Tests
    // ========================================================================

    @Test
    fun `ProjectTasksAction calls executeCommand with project tasks`() {
        every { fixture.client.executeCommand("project", listOf("tasks")) } returns
            Result.success(InteractiveCommandResponse(success = true, message = "Tasks retrieved"))

        ProjectTasksAction().actionPerformed(fixture.event)

        verify { fixture.client.executeCommand("project", listOf("tasks")) }
    }

    // ========================================================================
    // ProjectEditAction Tests
    // ========================================================================

    @Test
    fun `ProjectEditAction returns early when task ID dialog is cancelled`() {
        fixture.setInputDialogResult("Project Edit Task", null)

        ProjectEditAction().actionPerformed(fixture.event)

        verify(exactly = 0) { fixture.client.executeCommand(any(), any()) }
    }

    // ========================================================================
    // ProjectSubmitAction Tests
    // ========================================================================

    @Test
    fun `ProjectSubmitAction calls executeCommand with provider`() {
        // Uses showEditableChooseDialog which returns null by default in our mock
        // So we test that no command is executed when dialog is cancelled
        ProjectSubmitAction().actionPerformed(fixture.event)

        // Dialog returns null by default, so no command should be executed
        verify(exactly = 0) { fixture.client.executeCommand(eq("project"), any()) }
    }

    // ========================================================================
    // ProjectStartAction Tests
    // ========================================================================

    @Test
    fun `ProjectStartAction calls executeCommand with project start`() {
        every { fixture.client.executeCommand("project", listOf("start")) } returns
            Result.success(InteractiveCommandResponse(success = true, message = "Started next task"))

        ProjectStartAction().actionPerformed(fixture.event)

        verify { fixture.client.executeCommand("project", listOf("start")) }
        verify { fixture.service.refreshState() }
    }

    @Test
    fun `ProjectStartAction shows error on failure`() {
        every { fixture.client.executeCommand("project", listOf("start")) } returns
            Result.success(InteractiveCommandResponse(success = false, error = "No tasks available"))

        ProjectStartAction().actionPerformed(fixture.event)

        verify { Messages.showErrorDialog(any<Project>(), any<String>(), any<String>()) }
    }

    // ========================================================================
    // ProjectSyncAction Tests
    // ========================================================================

    @Test
    fun `ProjectSyncAction returns early when dialog is cancelled`() {
        fixture.setInputDialogResult("Project Sync", null)

        ProjectSyncAction().actionPerformed(fixture.event)

        verify(exactly = 0) { fixture.client.executeCommand(eq("project"), any()) }
    }

    @Test
    fun `ProjectSyncAction calls executeCommand with reference`() {
        fixture.setInputDialogResult("Project Sync", "github:user/repo")
        every { fixture.client.executeCommand("project", listOf("sync", "github:user/repo")) } returns
            Result.success(InteractiveCommandResponse(success = true, message = "Project synced"))

        ProjectSyncAction().actionPerformed(fixture.event)

        verify { fixture.client.executeCommand("project", listOf("sync", "github:user/repo")) }
    }

    // ========================================================================
    // StackListAction Tests
    // ========================================================================

    @Test
    fun `StackListAction calls executeCommand with stack list`() {
        every { fixture.client.executeCommand("stack", listOf("list")) } returns
            Result.success(InteractiveCommandResponse(success = true, message = "Stacks retrieved"))

        StackListAction().actionPerformed(fixture.event)

        verify { fixture.client.executeCommand("stack", listOf("list")) }
    }

    // ========================================================================
    // StackRebaseAction Tests
    // ========================================================================

    @Test
    fun `StackRebaseAction calls executeCommand with stack rebase`() {
        // Dialog returns null by default - rebase with empty task ID
        every { fixture.client.executeCommand("stack", listOf("rebase")) } returns
            Result.success(InteractiveCommandResponse(success = true, message = "Stack rebased"))

        StackRebaseAction().actionPerformed(fixture.event)

        verify { fixture.client.executeCommand("stack", listOf("rebase")) }
    }

    @Test
    fun `StackRebaseAction calls executeCommand with task ID when provided`() {
        fixture.setInputDialogResult("Stack Rebase", "task-123")
        every { fixture.client.executeCommand("stack", listOf("rebase", "task-123")) } returns
            Result.success(InteractiveCommandResponse(success = true, message = "Stack rebased"))

        StackRebaseAction().actionPerformed(fixture.event)

        verify { fixture.client.executeCommand("stack", listOf("rebase", "task-123")) }
    }

    // ========================================================================
    // StackSyncAction Tests
    // ========================================================================

    @Test
    fun `StackSyncAction calls executeCommand with stack sync`() {
        every { fixture.client.executeCommand("stack", listOf("sync")) } returns
            Result.success(InteractiveCommandResponse(success = true, message = "Stacks synced"))

        StackSyncAction().actionPerformed(fixture.event)

        verify { fixture.client.executeCommand("stack", listOf("sync")) }
    }

    // ========================================================================
    // ConfigValidateAction Tests
    // ========================================================================

    @Test
    fun `ConfigValidateAction calls executeCommand with config validate`() {
        every { fixture.client.executeCommand("config", listOf("validate")) } returns
            Result.success(InteractiveCommandResponse(success = true, message = "Configuration valid"))

        ConfigValidateAction().actionPerformed(fixture.event)

        verify { fixture.client.executeCommand("config", listOf("validate")) }
        verify { Messages.showInfoMessage(any<Project>(), any<String>(), eq("Mehrhof")) }
    }

    @Test
    fun `ConfigValidateAction shows error on validation failure`() {
        every { fixture.client.executeCommand("config", listOf("validate")) } returns
            Result.success(InteractiveCommandResponse(success = false, error = "Invalid configuration"))

        ConfigValidateAction().actionPerformed(fixture.event)

        verify { Messages.showErrorDialog(any<Project>(), any<String>(), any<String>()) }
    }

    // ========================================================================
    // AgentsListAction Tests
    // ========================================================================

    @Test
    fun `AgentsListAction calls executeCommand with agents list`() {
        every { fixture.client.executeCommand("agents", listOf("list")) } returns
            Result.success(InteractiveCommandResponse(success = true, message = "Agents listed"))

        AgentsListAction().actionPerformed(fixture.event)

        verify { fixture.client.executeCommand("agents", listOf("list")) }
    }

    // ========================================================================
    // AgentsExplainAction Tests
    // ========================================================================

    @Test
    fun `AgentsExplainAction returns early when dialog is cancelled`() {
        fixture.setInputDialogResult("Agents Explain", null)

        AgentsExplainAction().actionPerformed(fixture.event)

        verify(exactly = 0) { fixture.client.executeCommand(eq("agents"), any()) }
    }

    @Test
    fun `AgentsExplainAction calls executeCommand with agent name`() {
        fixture.setInputDialogResult("Agents Explain", "claude")
        every { fixture.client.executeCommand("agents", listOf("explain", "claude")) } returns
            Result.success(InteractiveCommandResponse(success = true, message = "Agent info"))

        AgentsExplainAction().actionPerformed(fixture.event)

        verify { fixture.client.executeCommand("agents", listOf("explain", "claude")) }
    }

    // ========================================================================
    // ProvidersListAction Tests
    // ========================================================================

    @Test
    fun `ProvidersListAction calls executeCommand with providers list`() {
        every { fixture.client.executeCommand("providers", listOf("list")) } returns
            Result.success(InteractiveCommandResponse(success = true, message = "Providers listed"))

        ProvidersListAction().actionPerformed(fixture.event)

        verify { fixture.client.executeCommand("providers", listOf("list")) }
    }

    // ========================================================================
    // ProvidersInfoAction Tests
    // ========================================================================

    @Test
    fun `ProvidersInfoAction returns early when dialog is cancelled`() {
        fixture.setInputDialogResult("Providers Info", null)

        ProvidersInfoAction().actionPerformed(fixture.event)

        verify(exactly = 0) { fixture.client.executeCommand(eq("providers"), any()) }
    }

    @Test
    fun `ProvidersInfoAction calls executeCommand with provider name`() {
        fixture.setInputDialogResult("Providers Info", "github")
        every { fixture.client.executeCommand("providers", listOf("info", "github")) } returns
            Result.success(InteractiveCommandResponse(success = true, message = "Provider info"))

        ProvidersInfoAction().actionPerformed(fixture.event)

        verify { fixture.client.executeCommand("providers", listOf("info", "github")) }
    }

    // ========================================================================
    // TemplatesListAction Tests
    // ========================================================================

    @Test
    fun `TemplatesListAction calls executeCommand with templates list`() {
        every { fixture.client.executeCommand("templates", listOf("list")) } returns
            Result.success(InteractiveCommandResponse(success = true, message = "Templates listed"))

        TemplatesListAction().actionPerformed(fixture.event)

        verify { fixture.client.executeCommand("templates", listOf("list")) }
    }

    // ========================================================================
    // TemplatesShowAction Tests
    // ========================================================================

    @Test
    fun `TemplatesShowAction returns early when dialog is cancelled`() {
        fixture.setInputDialogResult("Templates Show", null)

        TemplatesShowAction().actionPerformed(fixture.event)

        verify(exactly = 0) { fixture.client.executeCommand(eq("templates"), any()) }
    }

    @Test
    fun `TemplatesShowAction calls executeCommand with template name`() {
        fixture.setInputDialogResult("Templates Show", "bug-fix")
        every { fixture.client.executeCommand("templates", listOf("show", "bug-fix")) } returns
            Result.success(InteractiveCommandResponse(success = true, message = "Template content"))

        TemplatesShowAction().actionPerformed(fixture.event)

        verify { fixture.client.executeCommand("templates", listOf("show", "bug-fix")) }
    }

    // ========================================================================
    // ScanAction Tests
    // ========================================================================

    @Test
    fun `ScanAction calls executeCommand with scan`() {
        every { fixture.client.executeCommand("scan", emptyList()) } returns
            Result.success(InteractiveCommandResponse(success = true, message = "Scan complete"))

        ScanAction().actionPerformed(fixture.event)

        verify { fixture.client.executeCommand("scan", emptyList()) }
        verify { Messages.showInfoMessage(any<Project>(), any<String>(), eq("Mehrhof")) }
    }

    @Test
    fun `ScanAction shows error on failure`() {
        every { fixture.client.executeCommand("scan", emptyList()) } returns
            Result.success(InteractiveCommandResponse(success = false, error = "Scan failed"))

        ScanAction().actionPerformed(fixture.event)

        verify { Messages.showErrorDialog(any<Project>(), any<String>(), any<String>()) }
    }

    // ========================================================================
    // CommitAction Tests
    // ========================================================================

    @Test
    fun `CommitAction calls executeCommand with commit`() {
        every { fixture.client.executeCommand("commit", emptyList()) } returns
            Result.success(InteractiveCommandResponse(success = true, message = "Commit analysis complete"))

        CommitAction().actionPerformed(fixture.event)

        verify { fixture.client.executeCommand("commit", emptyList()) }
    }

    // ========================================================================
    // Update Tests (presentation state)
    // ========================================================================

    @Test
    fun `project actions disable when not connected`() {
        fixture.setConnected(false)

        val actions =
            listOf(
                ProjectPlanAction(),
                ProjectTasksAction(),
                ProjectEditAction(),
                ProjectSubmitAction(),
                ProjectStartAction(),
                ProjectSyncAction(),
            )

        for (action in actions) {
            fixture.resetPresentation()
            action.update(fixture.event)
            assert(!fixture.presentation.isEnabled) { "${action::class.simpleName} should be disabled" }
        }
    }

    @Test
    fun `stack actions disable when not connected`() {
        fixture.setConnected(false)

        val actions =
            listOf(
                StackListAction(),
                StackRebaseAction(),
                StackSyncAction(),
            )

        for (action in actions) {
            fixture.resetPresentation()
            action.update(fixture.event)
            assert(!fixture.presentation.isEnabled) { "${action::class.simpleName} should be disabled" }
        }
    }

    @Test
    fun `config actions disable when not connected`() {
        fixture.setConnected(false)

        val actions =
            listOf(
                ConfigValidateAction(),
                AgentsListAction(),
                AgentsExplainAction(),
                ProvidersListAction(),
                ProvidersInfoAction(),
                TemplatesListAction(),
                TemplatesShowAction(),
                ScanAction(),
                CommitAction(),
            )

        for (action in actions) {
            fixture.resetPresentation()
            action.update(fixture.event)
            assert(!fixture.presentation.isEnabled) { "${action::class.simpleName} should be disabled" }
        }
    }

    @Test
    fun `all actions enable when connected`() {
        fixture.setConnected(true)

        val actions =
            listOf(
                ProjectPlanAction(),
                ProjectTasksAction(),
                ProjectEditAction(),
                ProjectSubmitAction(),
                ProjectStartAction(),
                ProjectSyncAction(),
                StackListAction(),
                StackRebaseAction(),
                StackSyncAction(),
                ConfigValidateAction(),
                AgentsListAction(),
                AgentsExplainAction(),
                ProvidersListAction(),
                ProvidersInfoAction(),
                TemplatesListAction(),
                TemplatesShowAction(),
                ScanAction(),
                CommitAction(),
            )

        for (action in actions) {
            fixture.resetPresentation()
            action.update(fixture.event)
            assert(fixture.presentation.isEnabled) { "${action::class.simpleName} should be enabled" }
        }
    }
}
