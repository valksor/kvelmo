package com.valksor.mehrhof.actions

import com.intellij.openapi.project.Project
import com.intellij.openapi.ui.Messages
import com.valksor.mehrhof.api.createQuickTask
import com.valksor.mehrhof.api.deleteQueueTask
import com.valksor.mehrhof.api.exportQueueTask
import com.valksor.mehrhof.api.find
import com.valksor.mehrhof.api.models.AllCostsResponse
import com.valksor.mehrhof.api.models.FindMatch
import com.valksor.mehrhof.api.models.FindSearchResponse
import com.valksor.mehrhof.api.models.GrandTotal
import com.valksor.mehrhof.api.models.TaskCostResponse
import com.valksor.mehrhof.api.models.TaskInfo
import com.valksor.mehrhof.api.optimizeQueueTask
import com.valksor.mehrhof.api.syncTask
import com.valksor.mehrhof.testutil.ActionTestFixture
import io.mockk.every
import io.mockk.verify
import org.junit.jupiter.api.AfterEach
import org.junit.jupiter.api.BeforeEach
import org.junit.jupiter.api.Test

/**
 * Unit tests for task-related actions in TaskActions.kt.
 *
 * Uses [ActionTestFixture] for common mock setup.
 */
class TaskActionsTest {
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
    // NoteAction Tests
    // ========================================================================

    @Test
    fun `NoteAction shows error when no active task`() {
        every { fixture.service.currentTask } returns null

        NoteAction().actionPerformed(fixture.event)

        verify { Messages.showErrorDialog(any<Project>(), eq("No active task"), any<String>()) }
    }

    @Test
    fun `NoteAction returns early when dialog is cancelled`() {
        every { fixture.service.currentTask } returns TaskInfo(id = "t1", state = "implementing", ref = "r")
        fixture.setInputDialogResult("Add Note", null)

        NoteAction().actionPerformed(fixture.event)

        verify(exactly = 0) { fixture.client.addNote(any(), any()) }
    }

    @Test
    fun `NoteAction calls addNote with task id and message`() {
        every { fixture.service.currentTask } returns TaskInfo(id = "task-42", state = "implementing", ref = "r")
        fixture.setInputDialogResult("Add Note", "my note")

        NoteAction().actionPerformed(fixture.event)

        verify { fixture.client.addNote("task-42", "my note") }
    }

    // ========================================================================
    // QuestionAction Tests
    // ========================================================================

    @Test
    fun `QuestionAction returns early when dialog is cancelled`() {
        fixture.setInputDialogResult("Ask Question", null)

        QuestionAction().actionPerformed(fixture.event)

        verify(exactly = 0) { fixture.client.question(any()) }
    }

    @Test
    fun `QuestionAction calls client question with user message`() {
        fixture.setInputDialogResult("Ask Question", "What is the API key format?")

        QuestionAction().actionPerformed(fixture.event)

        verify { fixture.client.question("What is the API key format?") }
    }

    // ========================================================================
    // ResetAction Tests
    // ========================================================================

    @Test
    fun `ResetAction returns early when user cancels confirmation`() {
        fixture.setYesNoDialogResult("Reset Workflow", Messages.NO)

        ResetAction().actionPerformed(fixture.event)

        verify(exactly = 0) { fixture.client.reset() }
    }

    @Test
    fun `ResetAction calls client reset when user confirms`() {
        fixture.setYesNoDialogResult("Reset Workflow", Messages.YES)

        ResetAction().actionPerformed(fixture.event)

        verify { fixture.client.reset() }
    }

    // ========================================================================
    // CostAction Tests
    // ========================================================================

    @Test
    fun `CostAction shows task costs when task is active`() {
        every { fixture.service.currentTask } returns TaskInfo(id = "t1", state = "implementing", ref = "r")
        every { fixture.client.getTaskCosts("t1") } returns
            Result.success(
                TaskCostResponse(
                    taskId = "t1",
                    title = "My Task",
                    totalCostUsd = 0.05,
                    inputTokens = 1000,
                    outputTokens = 500,
                    cachedTokens = 100,
                    cachedPercent = 10.0,
                    totalTokens = 1500,
                ),
            )

        CostAction().actionPerformed(fixture.event)

        verify { fixture.client.getTaskCosts("t1") }
        verify { Messages.showInfoMessage(any<Project>(), any<String>(), eq("Task Costs")) }
    }

    @Test
    fun `CostAction shows all costs when no task active`() {
        every { fixture.service.currentTask } returns null
        every { fixture.client.getAllCosts() } returns
            Result.success(
                AllCostsResponse(
                    tasks = emptyList(),
                    grandTotal =
                        GrandTotal(
                            inputTokens = 5000,
                            outputTokens = 2500,
                            totalTokens = 7500,
                            cachedTokens = 500,
                            costUsd = 0.25,
                        ),
                ),
            )

        CostAction().actionPerformed(fixture.event)

        verify { fixture.client.getAllCosts() }
        verify { Messages.showInfoMessage(any<Project>(), any<String>(), eq("All Costs")) }
    }

    // ========================================================================
    // RefreshAction Tests
    // ========================================================================

    @Test
    fun `RefreshAction calls service refreshState`() {
        RefreshAction().actionPerformed(fixture.event)

        verify { fixture.service.refreshState() }
    }

    // ========================================================================
    // QuickTaskAction Tests
    // ========================================================================

    @Test
    fun `QuickTaskAction returns early when dialog is cancelled`() {
        fixture.setInputDialogResult("Quick Task", null)

        QuickTaskAction().actionPerformed(fixture.event)

        verify(exactly = 0) { fixture.client.createQuickTask(any()) }
    }

    @Test
    fun `QuickTaskAction calls createQuickTask with description`() {
        fixture.setInputDialogResult("Quick Task", "Fix the login page")

        QuickTaskAction().actionPerformed(fixture.event)

        verify { fixture.client.createQuickTask("Fix the login page") }
    }

    // ========================================================================
    // DeleteQueueTaskAction Tests
    // ========================================================================

    @Test
    fun `DeleteQueueTaskAction shows error for invalid format`() {
        fixture.setInputDialogResult("Delete Queue Task", "invalid-no-slash")

        DeleteQueueTaskAction().actionPerformed(fixture.event)

        verify {
            Messages.showErrorDialog(
                any<Project>(),
                eq("Invalid task reference format (expected: queue/task-id)"),
                any<String>(),
            )
        }
    }

    @Test
    fun `DeleteQueueTaskAction calls deleteQueueTask with parsed parts`() {
        fixture.setInputDialogResult("Delete Queue Task", "backlog/fix-123")
        fixture.setYesNoDialogResult("Delete Queue Task", Messages.YES)

        DeleteQueueTaskAction().actionPerformed(fixture.event)

        verify { fixture.client.deleteQueueTask("backlog", "fix-123") }
    }

    // ========================================================================
    // ExportQueueTaskAction Tests
    // ========================================================================

    @Test
    fun `ExportQueueTaskAction shows error for invalid format`() {
        fixture.setInputDialogResult("Export Queue Task", "invalid")

        ExportQueueTaskAction().actionPerformed(fixture.event)

        verify { Messages.showErrorDialog(any<Project>(), any<String>(), any<String>()) }
    }

    @Test
    fun `ExportQueueTaskAction calls exportQueueTask with parsed parts`() {
        fixture.setInputDialogResult("Export Queue Task", "queue/task-1")

        ExportQueueTaskAction().actionPerformed(fixture.event)

        verify { fixture.client.exportQueueTask("queue", "task-1") }
    }

    // ========================================================================
    // OptimizeQueueTaskAction Tests
    // ========================================================================

    @Test
    fun `OptimizeQueueTaskAction calls optimizeQueueTask with parsed parts`() {
        fixture.setInputDialogResult("Optimize Queue Task", "backlog/task-789")

        OptimizeQueueTaskAction().actionPerformed(fixture.event)

        verify { fixture.client.optimizeQueueTask("backlog", "task-789") }
    }

    // ========================================================================
    // SyncTaskAction Tests
    // ========================================================================

    @Test
    fun `SyncTaskAction shows error when no active task`() {
        every { fixture.service.currentTask } returns null

        SyncTaskAction().actionPerformed(fixture.event)

        verify { Messages.showErrorDialog(any<Project>(), eq("No active task to sync"), any<String>()) }
    }

    @Test
    fun `SyncTaskAction calls syncTask when active task exists`() {
        every { fixture.service.currentTask } returns TaskInfo(id = "t1", state = "implementing", ref = "r")

        SyncTaskAction().actionPerformed(fixture.event)

        verify { fixture.client.syncTask() }
    }

    // ========================================================================
    // FindAction Tests
    // ========================================================================

    @Test
    fun `FindAction returns early when dialog is cancelled`() {
        fixture.setInputDialogResult("Find in Codebase", null)

        FindAction().actionPerformed(fixture.event)

        verify(exactly = 0) { fixture.client.find(any()) }
    }

    @Test
    fun `FindAction shows no matches message when count is zero`() {
        fixture.setInputDialogResult("Find in Codebase", "nonexistent")
        every { fixture.client.find("nonexistent") } returns
            Result.success(FindSearchResponse(query = "nonexistent", count = 0, matches = emptyList()))

        FindAction().actionPerformed(fixture.event)

        verify { Messages.showInfoMessage(any<Project>(), eq("No matches found"), any<String>()) }
    }

    @Test
    fun `FindAction shows results when matches found`() {
        fixture.setInputDialogResult("Find in Codebase", "foo")
        every { fixture.client.find("foo") } returns
            Result.success(
                FindSearchResponse(
                    query = "foo",
                    count = 2,
                    matches =
                        listOf(
                            FindMatch(file = "src/main.kt", line = 10, snippet = "fun foo()"),
                            FindMatch(file = "src/util.kt", line = 20, snippet = "val foo = 1"),
                        ),
                ),
            )

        FindAction().actionPerformed(fixture.event)

        verify { Messages.showInfoMessage(any<Project>(), any<String>(), eq("Search Results")) }
    }

    // ========================================================================
    // Update Tests (presentation state)
    // ========================================================================

    @Test
    fun `task actions disable when not connected`() {
        fixture.setConnected(false)

        val actions =
            listOf(
                NoteAction(),
                QuestionAction(),
                ResetAction(),
                CostAction(),
                RefreshAction(),
                QuickTaskAction(),
                SyncTaskAction(),
                FindAction(),
            )

        for (action in actions) {
            fixture.resetPresentation()
            action.update(fixture.event)
            assert(!fixture.presentation.isEnabled) { "${action::class.simpleName} should be disabled" }
        }
    }
}
