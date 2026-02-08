package com.valksor.mehrhof.actions

import com.intellij.openapi.project.Project
import com.intellij.openapi.ui.Messages
import com.valksor.mehrhof.api.libraryList
import com.valksor.mehrhof.api.libraryPull
import com.valksor.mehrhof.api.libraryRemove
import com.valksor.mehrhof.api.libraryShow
import com.valksor.mehrhof.api.libraryStats
import com.valksor.mehrhof.api.models.InteractiveCommandResponse
import com.valksor.mehrhof.api.models.LibraryCollection
import com.valksor.mehrhof.api.models.LibraryListResponse
import com.valksor.mehrhof.api.models.LibraryShowResponse
import com.valksor.mehrhof.api.models.LibraryStatsResponse
import com.valksor.mehrhof.testutil.ActionTestFixture
import io.mockk.every
import io.mockk.verify
import org.junit.jupiter.api.AfterEach
import org.junit.jupiter.api.BeforeEach
import org.junit.jupiter.api.Test

/**
 * Unit tests for library-related actions in LibraryActions.kt.
 *
 * Uses [ActionTestFixture] for common mock setup.
 */
class LibraryActionsTest {
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
    // LibraryListAction Tests
    // ========================================================================

    @Test
    fun `LibraryListAction shows no collections message when count is zero`() {
        every { fixture.client.libraryList() } returns
            Result.success(LibraryListResponse(collections = emptyList(), count = 0))

        LibraryListAction().actionPerformed(fixture.event)

        verify { Messages.showInfoMessage(any<Project>(), eq("No collections in library"), any<String>()) }
    }

    @Test
    fun `LibraryListAction shows collections when found`() {
        every { fixture.client.libraryList() } returns
            Result.success(
                LibraryListResponse(
                    collections = listOf(
                        LibraryCollection(
                            id = "c1",
                            name = "React Docs",
                            source = "https://react.dev",
                            sourceType = "website",
                            includeMode = "all",
                            pageCount = 50,
                            totalSize = 1024000,
                            location = "/docs/react",
                        ),
                    ),
                    count = 1,
                ),
            )

        LibraryListAction().actionPerformed(fixture.event)

        verify { Messages.showInfoMessage(any<Project>(), any<String>(), eq("Library Collections")) }
    }

    @Test
    fun `LibraryListAction shows error on failure`() {
        every { fixture.client.libraryList() } returns Result.failure(Exception("Network error"))

        LibraryListAction().actionPerformed(fixture.event)

        verify { Messages.showErrorDialog(any<Project>(), any<String>(), any<String>()) }
    }

    // ========================================================================
    // LibraryShowAction Tests
    // ========================================================================

    @Test
    fun `LibraryShowAction returns early when dialog is cancelled`() {
        fixture.setInputDialogResult("Library Show", null)

        LibraryShowAction().actionPerformed(fixture.event)

        verify(exactly = 0) { fixture.client.libraryShow(any()) }
    }

    @Test
    fun `LibraryShowAction returns early for blank input`() {
        fixture.setInputDialogResult("Library Show", "   ")

        LibraryShowAction().actionPerformed(fixture.event)

        verify(exactly = 0) { fixture.client.libraryShow(any()) }
    }

    @Test
    fun `LibraryShowAction shows collection details`() {
        fixture.setInputDialogResult("Library Show", "react-docs")
        every { fixture.client.libraryShow("react-docs") } returns
            Result.success(
                LibraryShowResponse(
                    collection = LibraryCollection(
                        id = "c1",
                        name = "React Docs",
                        source = "https://react.dev",
                        sourceType = "website",
                        includeMode = "all",
                        pageCount = 50,
                        totalSize = 2048000,
                        location = "/docs/react",
                        pulledAt = "2024-01-15T10:00:00Z",
                    ),
                    pages = listOf("hooks.md", "components.md", "state.md"),
                ),
            )

        LibraryShowAction().actionPerformed(fixture.event)

        verify { Messages.showInfoMessage(any<Project>(), any<String>(), eq("Collection: React Docs")) }
    }

    // ========================================================================
    // LibraryPullAction Tests
    // ========================================================================

    @Test
    fun `LibraryPullAction returns early when source dialog is cancelled`() {
        fixture.setInputDialogResult("Library Pull", null)

        LibraryPullAction().actionPerformed(fixture.event)

        verify(exactly = 0) { fixture.client.libraryPull(any(), any(), any()) }
    }

    @Test
    fun `LibraryPullAction returns early for blank source`() {
        fixture.setInputDialogResult("Library Pull", "   ")

        LibraryPullAction().actionPerformed(fixture.event)

        verify(exactly = 0) { fixture.client.libraryPull(any(), any(), any()) }
    }

    @Test
    fun `LibraryPullAction calls libraryPull on success`() {
        // First call is for source URL
        fixture.setInputDialogResult("Library Pull", "https://docs.example.com")
        // Second call is for collection name (let it default to null)
        fixture.setYesNoDialogResult("Library Pull", Messages.NO)

        every { fixture.client.libraryPull("https://docs.example.com", null, false) } returns
            Result.success(InteractiveCommandResponse(success = true, message = "Collection pulled"))

        LibraryPullAction().actionPerformed(fixture.event)

        verify { fixture.client.libraryPull("https://docs.example.com", any(), eq(false)) }
    }

    @Test
    fun `LibraryPullAction shows error on failure`() {
        fixture.setInputDialogResult("Library Pull", "https://invalid.com")
        fixture.setYesNoDialogResult("Library Pull", Messages.NO)

        every { fixture.client.libraryPull(any(), any(), any()) } returns
            Result.success(InteractiveCommandResponse(success = false, error = "Failed to pull"))

        LibraryPullAction().actionPerformed(fixture.event)

        verify { Messages.showErrorDialog(any<Project>(), any<String>(), any<String>()) }
    }

    // ========================================================================
    // LibraryRemoveAction Tests
    // ========================================================================

    @Test
    fun `LibraryRemoveAction returns early when input dialog is cancelled`() {
        fixture.setInputDialogResult("Library Remove", null)

        LibraryRemoveAction().actionPerformed(fixture.event)

        verify(exactly = 0) { fixture.client.libraryRemove(any()) }
    }

    @Test
    fun `LibraryRemoveAction returns early when user cancels confirmation`() {
        fixture.setInputDialogResult("Library Remove", "old-docs")
        fixture.setYesNoDialogResult("Library Remove", Messages.NO)

        LibraryRemoveAction().actionPerformed(fixture.event)

        verify(exactly = 0) { fixture.client.libraryRemove(any()) }
    }

    @Test
    fun `LibraryRemoveAction calls libraryRemove when user confirms`() {
        fixture.setInputDialogResult("Library Remove", "old-docs")
        fixture.setYesNoDialogResult("Library Remove", Messages.YES)

        every { fixture.client.libraryRemove("old-docs") } returns
            Result.success(InteractiveCommandResponse(success = true, message = "Collection removed"))

        LibraryRemoveAction().actionPerformed(fixture.event)

        verify { fixture.client.libraryRemove("old-docs") }
        verify { Messages.showInfoMessage(any<Project>(), any<String>(), eq("Mehrhof")) }
    }

    // ========================================================================
    // LibraryStatsAction Tests
    // ========================================================================

    @Test
    fun `LibraryStatsAction shows not enabled message when disabled`() {
        every { fixture.client.libraryStats() } returns
            Result.success(
                LibraryStatsResponse(
                    totalCollections = 0,
                    totalPages = 0,
                    totalSize = 0,
                    projectCount = 0,
                    sharedCount = 0,
                    byMode = emptyMap(),
                    enabled = false,
                ),
            )

        LibraryStatsAction().actionPerformed(fixture.event)

        verify { Messages.showInfoMessage(any<Project>(), eq("Library system is not enabled"), any<String>()) }
    }

    @Test
    fun `LibraryStatsAction shows stats when enabled`() {
        every { fixture.client.libraryStats() } returns
            Result.success(
                LibraryStatsResponse(
                    totalCollections = 10,
                    totalPages = 500,
                    totalSize = 5000000,
                    projectCount = 7,
                    sharedCount = 3,
                    byMode = mapOf("all" to 8, "selective" to 2),
                    enabled = true,
                ),
            )

        LibraryStatsAction().actionPerformed(fixture.event)

        verify { Messages.showInfoMessage(any<Project>(), any<String>(), eq("Library Statistics")) }
    }

    @Test
    fun `LibraryStatsAction shows error on failure`() {
        every { fixture.client.libraryStats() } returns Result.failure(Exception("Connection failed"))

        LibraryStatsAction().actionPerformed(fixture.event)

        verify { Messages.showErrorDialog(any<Project>(), any<String>(), any<String>()) }
    }

    // ========================================================================
    // Update Tests (presentation state)
    // ========================================================================

    @Test
    fun `library actions disable when not connected`() {
        fixture.setConnected(false)

        val actions = listOf(
            LibraryListAction(),
            LibraryShowAction(),
            LibraryPullAction(),
            LibraryRemoveAction(),
            LibraryStatsAction(),
        )

        for (action in actions) {
            fixture.resetPresentation()
            action.update(fixture.event)
            assert(!fixture.presentation.isEnabled) { "${action::class.simpleName} should be disabled" }
        }
    }

    @Test
    fun `library actions enable when connected`() {
        fixture.setConnected(true)

        val actions = listOf(
            LibraryListAction(),
            LibraryShowAction(),
            LibraryPullAction(),
            LibraryRemoveAction(),
            LibraryStatsAction(),
        )

        for (action in actions) {
            fixture.resetPresentation()
            action.update(fixture.event)
            assert(fixture.presentation.isEnabled) { "${action::class.simpleName} should be enabled" }
        }
    }
}
