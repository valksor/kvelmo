package com.valksor.mehrhof.actions

import com.intellij.openapi.actionSystem.AnActionEvent
import com.intellij.openapi.actionSystem.Presentation
import com.intellij.openapi.progress.ProgressIndicator
import com.intellij.openapi.progress.ProgressManager
import com.intellij.openapi.progress.Task
import com.intellij.openapi.project.Project
import com.intellij.openapi.ui.Messages
import com.valksor.mehrhof.api.*
import com.valksor.mehrhof.api.MehrhofApiClient
import com.valksor.mehrhof.api.models.AddNoteResponse
import com.valksor.mehrhof.api.models.ContinueResponse
import com.valksor.mehrhof.api.models.InteractiveCommandResponse
import com.valksor.mehrhof.api.models.TaskInfo
import com.valksor.mehrhof.api.models.WorkflowResponse
import com.valksor.mehrhof.services.MehrhofProjectService
import io.mockk.*
import org.junit.jupiter.api.AfterEach
import org.junit.jupiter.api.Assertions.*
import org.junit.jupiter.api.BeforeEach
import org.junit.jupiter.api.Test
import javax.swing.Icon

class WorkflowActionsTest {
    private lateinit var event: AnActionEvent
    private lateinit var project: Project
    private lateinit var service: MehrhofProjectService
    private lateinit var client: MehrhofApiClient
    private lateinit var presentation: Presentation

    @BeforeEach
    fun setUp() {
        event = mockk(relaxed = true)
        project = mockk(relaxed = true)
        service = mockk(relaxed = true)
        client = mockk(relaxed = true)
        presentation = Presentation()

        every { event.project } returns project
        every { event.presentation } returns presentation

        mockkStatic(MehrhofProjectService::class)
        every { MehrhofProjectService.getInstance(project) } returns service
        every { service.getApiClient() } returns client
        every { service.isConnected() } returns true

        // Mock ProgressManager so runInBackground() executes the task synchronously
        mockkStatic(ProgressManager::class)
        val progressManager = mockk<ProgressManager>(relaxed = true)
        every { ProgressManager.getInstance() } returns progressManager
        every { progressManager.run(any<Task>()) } answers {
            (firstArg<Task>() as Task.Backgroundable).run(mockk<ProgressIndicator>(relaxed = true))
        }

        // Mock Messages with default stubs for all commonly used methods
        mockkStatic(Messages::class)
        every {
            Messages.showErrorDialog(any<Project>(), any<String>(), any<String>())
        } just Runs
        every {
            Messages.showInfoMessage(any<Project>(), any<String>(), any<String>())
        } just Runs
        every {
            Messages.showInputDialog(
                any<Project>(),
                any<String>(),
                any<String>(),
                any<Icon>(),
            )
        } returns null
        every {
            Messages.showYesNoDialog(
                any<Project>(),
                any<String>(),
                any<String>(),
                any<Icon>(),
            )
        } returns Messages.NO

        // Stub Result<T> returns — relaxed mocks lose generic type info (type erasure),
        // causing ClassCastException when .onSuccess { response -> response.success } runs.
        val okWorkflow = Result.success(WorkflowResponse(success = true))
        val okCommand = Result.success(InteractiveCommandResponse(success = true))
        val okContinue =
            Result.success(
                ContinueResponse(
                    success = true,
                    state = "idle",
                    nextActions = emptyList(),
                    message = "",
                ),
            )
        every { client.plan(any()) } returns okWorkflow
        every { client.implement(any()) } returns okWorkflow
        every { client.review(any()) } returns okWorkflow
        every { client.finish(any()) } returns okWorkflow
        every { client.continueWorkflow(any()) } returns okContinue
        every { client.undo() } returns okWorkflow
        every { client.redo() } returns okWorkflow
        every { client.abandon() } returns okWorkflow
        every { client.reset() } returns okWorkflow
        every { client.question(any()) } returns okWorkflow
        every { client.addNote(any(), any()) } returns
            Result.success(
                AddNoteResponse(success = true),
            )
        every { client.executeCommand(any(), any()) } returns okCommand

        mockkStatic("com.valksor.mehrhof.api.MehrhofApiClientExtensionsKt")
        every { client.createQuickTask(any()) } returns okCommand
        every { client.deleteQueueTask(any(), any()) } returns okCommand
        every { client.syncTask() } returns okCommand
        every { client.linksRebuild() } returns okCommand
    }

    @AfterEach
    fun tearDown() {
        unmockkAll()
    }

    // ========================================================================
    // Base class: null-safety guards
    // ========================================================================

    @Test
    fun `action does nothing when project is null`() {
        every { event.project } returns null

        PlanAction().actionPerformed(event)

        verify(exactly = 0) { client.plan(any()) }
    }

    @Test
    fun `action does nothing when api client is null`() {
        every { service.getApiClient() } returns null

        ImplementAction().actionPerformed(event)

        verify(exactly = 0) { client.implement(any()) }
    }

    // ========================================================================
    // Connection Actions
    // ========================================================================

    @Test
    fun `ConnectAction calls service connect`() {
        ConnectAction().actionPerformed(event)

        verify { service.connect() }
    }

    @Test
    fun `ConnectAction update reflects connection state`() {
        every { service.isConnected() } returns true
        ConnectAction().update(event)
        assertFalse(presentation.isEnabled)
        assertEquals("Connected", presentation.text)

        presentation = Presentation()
        every { event.presentation } returns presentation
        every { service.isConnected() } returns false
        ConnectAction().update(event)
        assertTrue(presentation.isEnabled)
        assertEquals("Connect to Server", presentation.text)
    }

    @Test
    fun `DisconnectAction calls service disconnect`() {
        DisconnectAction().actionPerformed(event)

        verify { service.disconnect() }
    }

    @Test
    fun `DisconnectAction update reflects connection state`() {
        every { service.isConnected() } returns true
        DisconnectAction().update(event)
        assertTrue(presentation.isEnabled)

        presentation = Presentation()
        every { event.presentation } returns presentation
        every { service.isConnected() } returns false
        DisconnectAction().update(event)
        assertFalse(presentation.isEnabled)
    }

    // ========================================================================
    // Simple Workflow Actions - correct API method is called
    // ========================================================================

    @Test
    fun `PlanAction calls client plan`() {
        PlanAction().actionPerformed(event)
        verify { client.plan() }
    }

    @Test
    fun `ImplementAction calls client implement`() {
        ImplementAction().actionPerformed(event)
        verify { client.implement() }
    }

    @Test
    fun `ReviewAction calls client review`() {
        ReviewAction().actionPerformed(event)
        verify { client.review() }
    }

    @Test
    fun `FinishAction calls client finish`() {
        FinishAction().actionPerformed(event)
        verify { client.finish(any()) }
    }

    @Test
    fun `ContinueAction calls client continueWorkflow`() {
        ContinueAction().actionPerformed(event)
        verify { client.continueWorkflow() }
    }

    @Test
    fun `UndoAction calls client undo`() {
        UndoAction().actionPerformed(event)
        verify { client.undo() }
    }

    @Test
    fun `RedoAction calls client redo`() {
        RedoAction().actionPerformed(event)
        verify { client.redo() }
    }

    // ========================================================================
    // Workflow actions refresh state after execution
    // ========================================================================

    @Test
    fun `PlanAction refreshes state after execution`() {
        PlanAction().actionPerformed(event)
        verify { service.refreshState() }
    }

    @Test
    fun `ContinueAction refreshes state after execution`() {
        ContinueAction().actionPerformed(event)
        verify { service.refreshState() }
    }

    // ========================================================================
    // Workflow actions disable when not connected
    // ========================================================================

    @Test
    fun `workflow actions disable when not connected`() {
        every { service.isConnected() } returns false

        val actions =
            listOf(
                PlanAction(),
                ImplementAction(),
                ReviewAction(),
                FinishAction(),
                ContinueAction(),
                UndoAction(),
                RedoAction()
            )

        for (action in actions) {
            presentation = Presentation()
            every { event.presentation } returns presentation
            action.update(event)
            assertFalse(presentation.isEnabled, "${action::class.simpleName} should be disabled")
        }
    }

    @Test
    fun `workflow actions enable when connected`() {
        every { service.isConnected() } returns true

        val actions =
            listOf(
                PlanAction(),
                ImplementAction(),
                ReviewAction(),
                FinishAction(),
                ContinueAction(),
                UndoAction(),
                RedoAction()
            )

        for (action in actions) {
            presentation = Presentation()
            every { event.presentation } returns presentation
            action.update(event)
            assertTrue(presentation.isEnabled, "${action::class.simpleName} should be enabled")
        }
    }

    // ========================================================================
    // StartTaskAction - input dialog behavior
    // ========================================================================

    @Test
    fun `StartTaskAction calls executeCommand with user-provided ref`() {
        every {
            Messages.showInputDialog(any<Project>(), any<String>(), eq("Start Task"), any())
        } returns "github:42"

        StartTaskAction().actionPerformed(event)

        verify { client.executeCommand("start", listOf("github:42")) }
    }

    @Test
    fun `StartTaskAction returns early when dialog is cancelled`() {
        every {
            Messages.showInputDialog(any<Project>(), any<String>(), eq("Start Task"), any())
        } returns null

        StartTaskAction().actionPerformed(event)

        verify(exactly = 0) { client.executeCommand(any(), any()) }
    }

    @Test
    fun `StartTaskAction returns early when input is blank`() {
        every {
            Messages.showInputDialog(any<Project>(), any<String>(), eq("Start Task"), any())
        } returns "   "

        StartTaskAction().actionPerformed(event)

        verify(exactly = 0) { client.executeCommand(any(), any()) }
    }

    // ========================================================================
    // NoteAction - input dialog and task dependency
    // ========================================================================

    @Test
    fun `NoteAction shows error when no active task`() {
        every { service.currentTask } returns null

        NoteAction().actionPerformed(event)

        verify { Messages.showErrorDialog(any<Project>(), eq("No active task"), any<String>()) }
        verify(exactly = 0) { client.addNote(any(), any()) }
    }

    @Test
    fun `NoteAction returns early when dialog is cancelled`() {
        every { service.currentTask } returns TaskInfo(id = "t1", state = "implementing", ref = "r")
        every {
            Messages.showInputDialog(any<Project>(), any<String>(), eq("Add Note"), any())
        } returns null

        NoteAction().actionPerformed(event)

        verify(exactly = 0) { client.addNote(any(), any()) }
    }

    @Test
    fun `NoteAction calls addNote with task id and message`() {
        every { service.currentTask } returns TaskInfo(id = "task-42", state = "implementing", ref = "r")
        every {
            Messages.showInputDialog(any<Project>(), any<String>(), eq("Add Note"), any())
        } returns "my note"

        NoteAction().actionPerformed(event)

        verify { client.addNote("task-42", "my note") }
    }

    // ========================================================================
    // QuestionAction - input dialog behavior
    // ========================================================================

    @Test
    fun `QuestionAction returns early when dialog is cancelled`() {
        every {
            Messages.showInputDialog(any<Project>(), any<String>(), eq("Ask Question"), any())
        } returns null

        QuestionAction().actionPerformed(event)

        verify(exactly = 0) { client.question(any()) }
    }

    @Test
    fun `QuestionAction calls client question with user message`() {
        every {
            Messages.showInputDialog(any<Project>(), any<String>(), eq("Ask Question"), any())
        } returns "What is the API key format?"

        QuestionAction().actionPerformed(event)

        verify { client.question("What is the API key format?") }
    }

    // ========================================================================
    // AbandonAction - confirmation dialog behavior
    // ========================================================================

    @Test
    fun `AbandonAction returns early when user cancels confirmation`() {
        every {
            Messages.showYesNoDialog(any<Project>(), any<String>(), eq("Abandon Task"), any())
        } returns Messages.NO

        AbandonAction().actionPerformed(event)

        verify(exactly = 0) { client.abandon() }
    }

    @Test
    fun `AbandonAction calls client abandon when user confirms`() {
        every {
            Messages.showYesNoDialog(any<Project>(), any<String>(), eq("Abandon Task"), any())
        } returns Messages.YES

        AbandonAction().actionPerformed(event)

        verify { client.abandon() }
    }

    // ========================================================================
    // ResetAction - confirmation dialog behavior
    // ========================================================================

    @Test
    fun `ResetAction returns early when user cancels confirmation`() {
        every {
            Messages.showYesNoDialog(any<Project>(), any<String>(), eq("Reset Workflow"), any())
        } returns Messages.NO

        ResetAction().actionPerformed(event)

        verify(exactly = 0) { client.reset() }
    }

    @Test
    fun `ResetAction calls client reset when user confirms`() {
        every {
            Messages.showYesNoDialog(any<Project>(), any<String>(), eq("Reset Workflow"), any())
        } returns Messages.YES

        ResetAction().actionPerformed(event)

        verify { client.reset() }
    }

    // ========================================================================
    // RefreshAction
    // ========================================================================

    @Test
    fun `RefreshAction calls service refreshState`() {
        RefreshAction().actionPerformed(event)

        verify { service.refreshState() }
    }

    // ========================================================================
    // QuickTaskAction - input dialog behavior
    // ========================================================================

    @Test
    fun `QuickTaskAction returns early when dialog is cancelled`() {
        every {
            Messages.showInputDialog(any<Project>(), any<String>(), eq("Quick Task"), any())
        } returns null

        QuickTaskAction().actionPerformed(event)

        verify(exactly = 0) { client.createQuickTask(any()) }
    }

    @Test
    fun `QuickTaskAction calls createQuickTask with description`() {
        every {
            Messages.showInputDialog(any<Project>(), any<String>(), eq("Quick Task"), any())
        } returns "Fix the login page"

        QuickTaskAction().actionPerformed(event)

        verify { client.createQuickTask("Fix the login page") }
    }

    // ========================================================================
    // DeleteQueueTaskAction - input parsing and confirmation
    // ========================================================================

    @Test
    fun `DeleteQueueTaskAction shows error for invalid format`() {
        every {
            Messages.showInputDialog(any<Project>(), any<String>(), eq("Delete Queue Task"), any())
        } returns "invalid-no-slash"

        DeleteQueueTaskAction().actionPerformed(event)

        verify {
            Messages.showErrorDialog(
                any<Project>(),
                eq("Invalid task reference format (expected: queue/task-id)"),
                any<String>()
            )
        }
        verify(exactly = 0) { client.deleteQueueTask(any(), any()) }
    }

    @Test
    fun `DeleteQueueTaskAction calls deleteQueueTask with parsed parts`() {
        every {
            Messages.showInputDialog(any<Project>(), any<String>(), eq("Delete Queue Task"), any())
        } returns "backlog/fix-123"
        every {
            Messages.showYesNoDialog(any<Project>(), any<String>(), eq("Delete Queue Task"), any())
        } returns Messages.YES

        DeleteQueueTaskAction().actionPerformed(event)

        verify { client.deleteQueueTask("backlog", "fix-123") }
    }

    // ========================================================================
    // ScanAction and CommitAction - no-input actions via executeCommand
    // ========================================================================

    @Test
    fun `ScanAction calls executeCommand with scan`() {
        ScanAction().actionPerformed(event)
        verify { client.executeCommand("scan", emptyList()) }
    }

    @Test
    fun `CommitAction calls executeCommand with commit`() {
        CommitAction().actionPerformed(event)
        verify { client.executeCommand("commit", emptyList()) }
    }

    // ========================================================================
    // LinksRebuildAction - confirmation dialog behavior
    // ========================================================================

    @Test
    fun `LinksRebuildAction calls linksRebuild only when confirmed`() {
        every {
            Messages.showYesNoDialog(any<Project>(), any<String>(), eq("Links Rebuild"), any())
        } returns Messages.NO
        LinksRebuildAction().actionPerformed(event)
        verify(exactly = 0) { client.linksRebuild() }

        every {
            Messages.showYesNoDialog(any<Project>(), any<String>(), eq("Links Rebuild"), any())
        } returns Messages.YES
        LinksRebuildAction().actionPerformed(event)
        verify { client.linksRebuild() }
    }

    // ========================================================================
    // SyncTaskAction - requires active task
    // ========================================================================

    @Test
    fun `SyncTaskAction shows error when no active task`() {
        every { service.currentTask } returns null

        SyncTaskAction().actionPerformed(event)

        verify { Messages.showErrorDialog(any<Project>(), eq("No active task to sync"), any<String>()) }
        verify(exactly = 0) { client.syncTask() }
    }

    @Test
    fun `SyncTaskAction calls syncTask when active task exists`() {
        every { service.currentTask } returns TaskInfo(id = "t1", state = "implementing", ref = "r")

        SyncTaskAction().actionPerformed(event)

        verify { client.syncTask() }
    }
}
