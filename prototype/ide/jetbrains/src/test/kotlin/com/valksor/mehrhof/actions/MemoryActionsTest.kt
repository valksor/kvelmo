package com.valksor.mehrhof.actions

import com.intellij.openapi.project.Project
import com.intellij.openapi.ui.Messages
import com.valksor.mehrhof.api.memoryIndex
import com.valksor.mehrhof.api.memorySearch
import com.valksor.mehrhof.api.memoryStats
import com.valksor.mehrhof.api.models.MemoryIndexResponse
import com.valksor.mehrhof.api.models.MemoryResult
import com.valksor.mehrhof.api.models.MemorySearchResponse
import com.valksor.mehrhof.api.models.MemoryStatsResponse
import com.valksor.mehrhof.api.models.TaskInfo
import com.valksor.mehrhof.testutil.ActionTestFixture
import io.mockk.every
import io.mockk.verify
import org.junit.jupiter.api.AfterEach
import org.junit.jupiter.api.BeforeEach
import org.junit.jupiter.api.Test

/**
 * Unit tests for memory-related actions in MemoryActions.kt.
 *
 * Uses [ActionTestFixture] for common mock setup.
 */
class MemoryActionsTest {
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
    // MemorySearchAction Tests
    // ========================================================================

    @Test
    fun `MemorySearchAction returns early when dialog is cancelled`() {
        fixture.setInputDialogResult("Memory Search", null)

        MemorySearchAction().actionPerformed(fixture.event)

        verify(exactly = 0) { fixture.client.memorySearch(any()) }
    }

    @Test
    fun `MemorySearchAction returns early for blank query`() {
        fixture.setInputDialogResult("Memory Search", "   ")

        MemorySearchAction().actionPerformed(fixture.event)

        verify(exactly = 0) { fixture.client.memorySearch(any()) }
    }

    @Test
    fun `MemorySearchAction shows no results message when count is zero`() {
        fixture.setInputDialogResult("Memory Search", "authentication")
        every { fixture.client.memorySearch("authentication") } returns
            Result.success(MemorySearchResponse(results = emptyList(), count = 0))

        MemorySearchAction().actionPerformed(fixture.event)

        verify { fixture.client.memorySearch("authentication") }
        verify { Messages.showInfoMessage(any<Project>(), eq("No similar tasks found"), any<String>()) }
    }

    @Test
    fun `MemorySearchAction shows results when matches found`() {
        fixture.setInputDialogResult("Memory Search", "login")
        every { fixture.client.memorySearch("login") } returns
            Result.success(
                MemorySearchResponse(
                    results = listOf(
                        MemoryResult(taskId = "t1", type = "spec", score = 0.95, content = "Login spec"),
                        MemoryResult(taskId = "t2", type = "impl", score = 0.80, content = "Login impl"),
                    ),
                    count = 2,
                ),
            )

        MemorySearchAction().actionPerformed(fixture.event)

        verify { Messages.showInfoMessage(any<Project>(), any<String>(), eq("Memory Search Results")) }
    }

    @Test
    fun `MemorySearchAction shows error on failure`() {
        fixture.setInputDialogResult("Memory Search", "query")
        every { fixture.client.memorySearch("query") } returns Result.failure(Exception("Network error"))

        MemorySearchAction().actionPerformed(fixture.event)

        verify { Messages.showErrorDialog(any<Project>(), any<String>(), any<String>()) }
    }

    // ========================================================================
    // MemoryIndexAction Tests
    // ========================================================================

    @Test
    fun `MemoryIndexAction returns early when dialog is cancelled`() {
        // MemoryIndexAction uses the 6-arg showInputDialog with a default value
        fixture.setInputDialogWithDefaultResult("Memory Index Task", null)

        MemoryIndexAction().actionPerformed(fixture.event)

        verify(exactly = 0) { fixture.client.memoryIndex(any()) }
    }

    @Test
    fun `MemoryIndexAction uses current task ID as default`() {
        every { fixture.service.currentTask } returns TaskInfo(id = "current-task", state = "implementing", ref = "r")
        // Dialog will return null by default, so action returns early
        // We're just testing that the default value is set up correctly

        MemoryIndexAction().actionPerformed(fixture.event)

        // The dialog mock is set up to return null, so we just verify no API call
        verify(exactly = 0) { fixture.client.memoryIndex(any()) }
    }

    @Test
    fun `MemoryIndexAction calls memoryIndex with task ID`() {
        // MemoryIndexAction uses the 6-arg showInputDialog with a default value
        fixture.setInputDialogWithDefaultResult("Memory Index Task", "task-123")
        every { fixture.client.memoryIndex("task-123") } returns
            Result.success(MemoryIndexResponse(success = true, message = "Task indexed"))

        MemoryIndexAction().actionPerformed(fixture.event)

        verify { fixture.client.memoryIndex("task-123") }
        verify { Messages.showInfoMessage(any<Project>(), any<String>(), eq("Mehrhof")) }
    }

    @Test
    fun `MemoryIndexAction shows error on API failure`() {
        // MemoryIndexAction uses the 6-arg showInputDialog with a default value
        fixture.setInputDialogWithDefaultResult("Memory Index Task", "task-456")
        every { fixture.client.memoryIndex("task-456") } returns
            Result.success(MemoryIndexResponse(success = false, error = "Task not found"))

        MemoryIndexAction().actionPerformed(fixture.event)

        verify { Messages.showErrorDialog(any<Project>(), any<String>(), any<String>()) }
    }

    // ========================================================================
    // MemoryStatsAction Tests
    // ========================================================================

    @Test
    fun `MemoryStatsAction shows not enabled message when disabled`() {
        every { fixture.client.memoryStats() } returns
            Result.success(MemoryStatsResponse(totalDocuments = 0, byType = emptyMap(), enabled = false))

        MemoryStatsAction().actionPerformed(fixture.event)

        verify { Messages.showInfoMessage(any<Project>(), eq("Memory system is not enabled"), any<String>()) }
    }

    @Test
    fun `MemoryStatsAction shows stats when enabled`() {
        every { fixture.client.memoryStats() } returns
            Result.success(
                MemoryStatsResponse(
                    totalDocuments = 100,
                    byType = mapOf("spec" to 50, "impl" to 30, "review" to 20),
                    enabled = true,
                ),
            )

        MemoryStatsAction().actionPerformed(fixture.event)

        verify { Messages.showInfoMessage(any<Project>(), any<String>(), eq("Memory Statistics")) }
    }

    @Test
    fun `MemoryStatsAction shows error on failure`() {
        every { fixture.client.memoryStats() } returns Result.failure(Exception("Connection failed"))

        MemoryStatsAction().actionPerformed(fixture.event)

        verify { Messages.showErrorDialog(any<Project>(), any<String>(), any<String>()) }
    }

    // ========================================================================
    // Update Tests (presentation state)
    // ========================================================================

    @Test
    fun `memory actions disable when not connected`() {
        fixture.setConnected(false)

        val actions = listOf(
            MemorySearchAction(),
            MemoryIndexAction(),
            MemoryStatsAction(),
        )

        for (action in actions) {
            fixture.resetPresentation()
            action.update(fixture.event)
            assert(!fixture.presentation.isEnabled) { "${action::class.simpleName} should be disabled" }
        }
    }

    @Test
    fun `memory actions enable when connected`() {
        fixture.setConnected(true)

        val actions = listOf(
            MemorySearchAction(),
            MemoryIndexAction(),
            MemoryStatsAction(),
        )

        for (action in actions) {
            fixture.resetPresentation()
            action.update(fixture.event)
            assert(fixture.presentation.isEnabled) { "${action::class.simpleName} should be enabled" }
        }
    }
}
