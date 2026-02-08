package com.valksor.mehrhof.actions

import com.intellij.openapi.project.Project
import com.intellij.openapi.ui.Messages
import com.valksor.mehrhof.api.linksList
import com.valksor.mehrhof.api.linksRebuild
import com.valksor.mehrhof.api.linksSearch
import com.valksor.mehrhof.api.linksStats
import com.valksor.mehrhof.api.models.EntityResult
import com.valksor.mehrhof.api.models.InteractiveCommandResponse
import com.valksor.mehrhof.api.models.LinkData
import com.valksor.mehrhof.api.models.LinksListResponse
import com.valksor.mehrhof.api.models.LinksSearchResponse
import com.valksor.mehrhof.api.models.LinksStatsResponse
import com.valksor.mehrhof.testutil.ActionTestFixture
import io.mockk.every
import io.mockk.verify
import org.junit.jupiter.api.AfterEach
import org.junit.jupiter.api.BeforeEach
import org.junit.jupiter.api.Test

/**
 * Unit tests for links-related actions in LinksActions.kt.
 *
 * Uses [ActionTestFixture] for common mock setup.
 */
class LinksActionsTest {
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
    // LinksListAction Tests
    // ========================================================================

    @Test
    fun `LinksListAction shows no links message when count is zero`() {
        every { fixture.client.linksList() } returns
            Result.success(LinksListResponse(links = emptyList(), count = 0))

        LinksListAction().actionPerformed(fixture.event)

        verify { Messages.showInfoMessage(any<Project>(), eq("No links found"), any<String>()) }
    }

    @Test
    fun `LinksListAction shows links when found`() {
        every { fixture.client.linksList() } returns
            Result.success(
                LinksListResponse(
                    links = listOf(
                        LinkData(source = "spec:1", target = "impl:1", context = "implements", createdAt = "2024-01-01"),
                        LinkData(source = "spec:2", target = "impl:2", context = "implements", createdAt = "2024-01-02"),
                    ),
                    count = 2,
                ),
            )

        LinksListAction().actionPerformed(fixture.event)

        verify { Messages.showInfoMessage(any<Project>(), any<String>(), eq("Links")) }
    }

    @Test
    fun `LinksListAction shows error on failure`() {
        every { fixture.client.linksList() } returns Result.failure(Exception("Network error"))

        LinksListAction().actionPerformed(fixture.event)

        verify { Messages.showErrorDialog(any<Project>(), any<String>(), any<String>()) }
    }

    // ========================================================================
    // LinksSearchAction Tests
    // ========================================================================

    @Test
    fun `LinksSearchAction returns early when dialog is cancelled`() {
        fixture.setInputDialogResult("Links Search", null)

        LinksSearchAction().actionPerformed(fixture.event)

        verify(exactly = 0) { fixture.client.linksSearch(any()) }
    }

    @Test
    fun `LinksSearchAction returns early for blank query`() {
        fixture.setInputDialogResult("Links Search", "   ")

        LinksSearchAction().actionPerformed(fixture.event)

        verify(exactly = 0) { fixture.client.linksSearch(any()) }
    }

    @Test
    fun `LinksSearchAction shows no results message when count is zero`() {
        fixture.setInputDialogResult("Links Search", "auth")
        every { fixture.client.linksSearch("auth") } returns
            Result.success(LinksSearchResponse(query = "auth", results = emptyList(), count = 0))

        LinksSearchAction().actionPerformed(fixture.event)

        verify { Messages.showInfoMessage(any<Project>(), eq("No entities found"), any<String>()) }
    }

    @Test
    fun `LinksSearchAction shows results when matches found`() {
        fixture.setInputDialogResult("Links Search", "authentication")
        every { fixture.client.linksSearch("authentication") } returns
            Result.success(
                LinksSearchResponse(
                    query = "authentication",
                    results = listOf(
                        EntityResult(entityId = "spec:auth", type = "spec", name = "Auth Spec", totalLinks = 5),
                        EntityResult(entityId = "impl:auth", type = "impl", name = "Auth Impl", totalLinks = 3),
                    ),
                    count = 2,
                ),
            )

        LinksSearchAction().actionPerformed(fixture.event)

        verify { Messages.showInfoMessage(any<Project>(), any<String>(), eq("Links Search Results")) }
    }

    // ========================================================================
    // LinksStatsAction Tests
    // ========================================================================

    @Test
    fun `LinksStatsAction shows not enabled message when disabled`() {
        every { fixture.client.linksStats() } returns
            Result.success(
                LinksStatsResponse(
                    totalLinks = 0,
                    totalSources = 0,
                    totalTargets = 0,
                    orphanEntities = 0,
                    mostLinked = emptyList(),
                    enabled = false,
                ),
            )

        LinksStatsAction().actionPerformed(fixture.event)

        verify { Messages.showInfoMessage(any<Project>(), eq("Links system is not enabled"), any<String>()) }
    }

    @Test
    fun `LinksStatsAction shows stats when enabled`() {
        every { fixture.client.linksStats() } returns
            Result.success(
                LinksStatsResponse(
                    totalLinks = 100,
                    totalSources = 50,
                    totalTargets = 60,
                    orphanEntities = 5,
                    mostLinked = listOf(
                        EntityResult(entityId = "spec:main", type = "spec", totalLinks = 20),
                    ),
                    enabled = true,
                ),
            )

        LinksStatsAction().actionPerformed(fixture.event)

        verify { Messages.showInfoMessage(any<Project>(), any<String>(), eq("Links Statistics")) }
    }

    // ========================================================================
    // LinksRebuildAction Tests
    // ========================================================================

    @Test
    fun `LinksRebuildAction returns early when user cancels confirmation`() {
        fixture.setYesNoDialogResult("Links Rebuild", Messages.NO)

        LinksRebuildAction().actionPerformed(fixture.event)

        verify(exactly = 0) { fixture.client.linksRebuild() }
    }

    @Test
    fun `LinksRebuildAction calls linksRebuild when user confirms`() {
        fixture.setYesNoDialogResult("Links Rebuild", Messages.YES)
        every { fixture.client.linksRebuild() } returns
            Result.success(InteractiveCommandResponse(success = true, message = "Links index rebuilt"))

        LinksRebuildAction().actionPerformed(fixture.event)

        verify { fixture.client.linksRebuild() }
        verify { Messages.showInfoMessage(any<Project>(), any<String>(), eq("Mehrhof")) }
    }

    @Test
    fun `LinksRebuildAction shows error on failure`() {
        fixture.setYesNoDialogResult("Links Rebuild", Messages.YES)
        every { fixture.client.linksRebuild() } returns
            Result.success(InteractiveCommandResponse(success = false, error = "Rebuild failed"))

        LinksRebuildAction().actionPerformed(fixture.event)

        verify { Messages.showErrorDialog(any<Project>(), any<String>(), any<String>()) }
    }

    // ========================================================================
    // Update Tests (presentation state)
    // ========================================================================

    @Test
    fun `links actions disable when not connected`() {
        fixture.setConnected(false)

        val actions = listOf(
            LinksListAction(),
            LinksSearchAction(),
            LinksStatsAction(),
            LinksRebuildAction(),
        )

        for (action in actions) {
            fixture.resetPresentation()
            action.update(fixture.event)
            assert(!fixture.presentation.isEnabled) { "${action::class.simpleName} should be disabled" }
        }
    }

    @Test
    fun `links actions enable when connected`() {
        fixture.setConnected(true)

        val actions = listOf(
            LinksListAction(),
            LinksSearchAction(),
            LinksStatsAction(),
            LinksRebuildAction(),
        )

        for (action in actions) {
            fixture.resetPresentation()
            action.update(fixture.event)
            assert(fixture.presentation.isEnabled) { "${action::class.simpleName} should be enabled" }
        }
    }
}
