package com.valksor.mehrhof.actions

import com.intellij.openapi.project.Project
import com.intellij.openapi.ui.Messages
import com.valksor.mehrhof.api.browserClick
import com.valksor.mehrhof.api.browserConsole
import com.valksor.mehrhof.api.browserEval
import com.valksor.mehrhof.api.browserGoto
import com.valksor.mehrhof.api.browserNavigate
import com.valksor.mehrhof.api.browserNetwork
import com.valksor.mehrhof.api.browserReload
import com.valksor.mehrhof.api.browserScreenshot
import com.valksor.mehrhof.api.browserStatus
import com.valksor.mehrhof.api.browserTabs
import com.valksor.mehrhof.api.browserType
import com.valksor.mehrhof.api.models.BrowserClickResponse
import com.valksor.mehrhof.api.models.BrowserConsoleMessage
import com.valksor.mehrhof.api.models.BrowserConsoleResponse
import com.valksor.mehrhof.api.models.BrowserEvalResponse
import com.valksor.mehrhof.api.models.BrowserGotoResponse
import com.valksor.mehrhof.api.models.BrowserNavigateResponse
import com.valksor.mehrhof.api.models.BrowserNetworkEntry
import com.valksor.mehrhof.api.models.BrowserNetworkResponse
import com.valksor.mehrhof.api.models.BrowserReloadResponse
import com.valksor.mehrhof.api.models.BrowserScreenshotResponse
import com.valksor.mehrhof.api.models.BrowserStatusResponse
import com.valksor.mehrhof.api.models.BrowserTab
import com.valksor.mehrhof.api.models.BrowserTabsResponse
import com.valksor.mehrhof.api.models.BrowserTypeResponse
import com.valksor.mehrhof.testutil.ActionTestFixture
import io.mockk.every
import io.mockk.verify
import org.junit.jupiter.api.AfterEach
import org.junit.jupiter.api.BeforeEach
import org.junit.jupiter.api.Test

/**
 * Unit tests for browser-related actions in BrowserActions.kt.
 *
 * Uses [ActionTestFixture] for common mock setup.
 */
class BrowserActionsTest {
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
    // BrowserStatusAction Tests
    // ========================================================================

    @Test
    fun `BrowserStatusAction shows not connected message when disconnected`() {
        every { fixture.client.browserStatus() } returns
            Result.success(BrowserStatusResponse(connected = false, error = "No browser"))

        BrowserStatusAction().actionPerformed(fixture.event)

        verify { Messages.showInfoMessage(any<Project>(), any<String>(), eq("Mehrhof")) }
    }

    @Test
    fun `BrowserStatusAction shows connection info when connected`() {
        every { fixture.client.browserStatus() } returns
            Result.success(
                BrowserStatusResponse(
                    connected = true,
                    host = "localhost",
                    port = 9222,
                    tabs = listOf(BrowserTab(id = "t1", title = "Test Page", url = "https://test.com")),
                ),
            )

        BrowserStatusAction().actionPerformed(fixture.event)

        verify { Messages.showInfoMessage(any<Project>(), any<String>(), eq("Browser Status")) }
    }

    @Test
    fun `BrowserStatusAction shows error on failure`() {
        every { fixture.client.browserStatus() } returns Result.failure(Exception("Connection failed"))

        BrowserStatusAction().actionPerformed(fixture.event)

        verify { Messages.showErrorDialog(any<Project>(), any<String>(), any<String>()) }
    }

    // ========================================================================
    // BrowserTabsAction Tests
    // ========================================================================

    @Test
    fun `BrowserTabsAction shows no tabs message when empty`() {
        every { fixture.client.browserTabs() } returns
            Result.success(BrowserTabsResponse(tabs = emptyList(), count = 0))

        BrowserTabsAction().actionPerformed(fixture.event)

        verify { Messages.showInfoMessage(any<Project>(), eq("No browser tabs open"), any<String>()) }
    }

    @Test
    fun `BrowserTabsAction shows tabs when found`() {
        every { fixture.client.browserTabs() } returns
            Result.success(
                BrowserTabsResponse(
                    tabs =
                        listOf(
                            BrowserTab(id = "t1", title = "Google", url = "https://google.com"),
                            BrowserTab(id = "t2", title = "GitHub", url = "https://github.com"),
                        ),
                    count = 2,
                ),
            )

        BrowserTabsAction().actionPerformed(fixture.event)

        verify { Messages.showInfoMessage(any<Project>(), any<String>(), eq("Browser Tabs")) }
    }

    // ========================================================================
    // BrowserGotoAction Tests
    // ========================================================================

    @Test
    fun `BrowserGotoAction returns early when dialog is cancelled`() {
        fixture.setInputDialogResult("Browser Go To", null)

        BrowserGotoAction().actionPerformed(fixture.event)

        verify(exactly = 0) { fixture.client.browserGoto(any()) }
    }

    @Test
    fun `BrowserGotoAction returns early for blank URL`() {
        fixture.setInputDialogResult("Browser Go To", "   ")

        BrowserGotoAction().actionPerformed(fixture.event)

        verify(exactly = 0) { fixture.client.browserGoto(any()) }
    }

    @Test
    fun `BrowserGotoAction calls browserGoto with URL`() {
        fixture.setInputDialogResult("Browser Go To", "https://example.com")
        every { fixture.client.browserGoto("https://example.com") } returns
            Result.success(
                BrowserGotoResponse(
                    success = true,
                    tab = BrowserTab(id = "new", title = "Example", url = "https://example.com"),
                ),
            )

        BrowserGotoAction().actionPerformed(fixture.event)

        verify { fixture.client.browserGoto("https://example.com") }
        verify { Messages.showInfoMessage(any<Project>(), any<String>(), eq("Mehrhof")) }
    }

    // ========================================================================
    // BrowserNavigateAction Tests
    // ========================================================================

    @Test
    fun `BrowserNavigateAction returns early when dialog is cancelled`() {
        fixture.setInputDialogResult("Browser Navigate", null)

        BrowserNavigateAction().actionPerformed(fixture.event)

        verify(exactly = 0) { fixture.client.browserNavigate(any(), any()) }
    }

    @Test
    fun `BrowserNavigateAction calls browserNavigate with URL`() {
        fixture.setInputDialogResult("Browser Navigate", "https://new-url.com")
        every { fixture.client.browserNavigate("https://new-url.com", null) } returns
            Result.success(BrowserNavigateResponse(success = true, message = "Navigated"))

        BrowserNavigateAction().actionPerformed(fixture.event)

        verify { fixture.client.browserNavigate("https://new-url.com", any()) }
        verify { Messages.showInfoMessage(any<Project>(), any<String>(), eq("Mehrhof")) }
    }

    // ========================================================================
    // BrowserReloadAction Tests
    // ========================================================================

    @Test
    fun `BrowserReloadAction calls browserReload`() {
        every { fixture.client.browserReload(any(), any()) } returns
            Result.success(BrowserReloadResponse(success = true, message = "Page reloaded"))

        BrowserReloadAction().actionPerformed(fixture.event)

        verify { fixture.client.browserReload(any(), any()) }
        verify { Messages.showInfoMessage(any<Project>(), any<String>(), eq("Mehrhof")) }
    }

    @Test
    fun `BrowserReloadAction shows error on failure`() {
        every { fixture.client.browserReload(any(), any()) } returns Result.failure(Exception("Reload failed"))

        BrowserReloadAction().actionPerformed(fixture.event)

        verify { Messages.showErrorDialog(any<Project>(), any<String>(), any<String>()) }
    }

    // ========================================================================
    // BrowserScreenshotAction Tests
    // ========================================================================

    @Test
    fun `BrowserScreenshotAction shows success message with size`() {
        every { fixture.client.browserScreenshot(any(), any(), any()) } returns
            Result.success(
                BrowserScreenshotResponse(
                    success = true,
                    format = "png",
                    data = "base64data",
                    size = 51200,
                ),
            )

        BrowserScreenshotAction().actionPerformed(fixture.event)

        verify { Messages.showInfoMessage(any<Project>(), any<String>(), eq("Mehrhof")) }
    }

    @Test
    fun `BrowserScreenshotAction shows error when no data returned`() {
        every { fixture.client.browserScreenshot(any(), any(), any()) } returns
            Result.success(BrowserScreenshotResponse(success = true, data = null))

        BrowserScreenshotAction().actionPerformed(fixture.event)

        verify { Messages.showErrorDialog(any<Project>(), any<String>(), any<String>()) }
    }

    // ========================================================================
    // BrowserClickAction Tests
    // ========================================================================

    @Test
    fun `BrowserClickAction returns early when dialog is cancelled`() {
        fixture.setInputDialogResult("Browser Click", null)

        BrowserClickAction().actionPerformed(fixture.event)

        verify(exactly = 0) { fixture.client.browserClick(any(), any()) }
    }

    @Test
    fun `BrowserClickAction calls browserClick with selector`() {
        fixture.setInputDialogResult("Browser Click", "#submit-btn")
        every { fixture.client.browserClick("#submit-btn", any()) } returns
            Result.success(BrowserClickResponse(success = true, selector = "#submit-btn"))

        BrowserClickAction().actionPerformed(fixture.event)

        verify { fixture.client.browserClick("#submit-btn", any()) }
        verify { Messages.showInfoMessage(any<Project>(), any<String>(), eq("Mehrhof")) }
    }

    // ========================================================================
    // BrowserTypeAction Tests
    // ========================================================================

    @Test
    fun `BrowserTypeAction returns early when selector dialog is cancelled`() {
        fixture.setInputDialogResult("Browser Type", null)

        BrowserTypeAction().actionPerformed(fixture.event)

        verify(exactly = 0) { fixture.client.browserType(any(), any(), any(), any()) }
    }

    @Test
    fun `BrowserTypeAction calls browserType with selector and text`() {
        // Both dialogs use "Browser Type" as title
        fixture.setInputDialogResult("Browser Type", "#username")
        every { fixture.client.browserType("#username", any(), any(), any()) } returns
            Result.success(BrowserTypeResponse(success = true, selector = "#username"))

        // We need to call it - the mock will return #username for both dialogs
        BrowserTypeAction().actionPerformed(fixture.event)

        verify { fixture.client.browserType(eq("#username"), any(), any(), any()) }
    }

    // ========================================================================
    // BrowserEvalAction Tests
    // ========================================================================

    @Test
    fun `BrowserEvalAction returns early when dialog is cancelled`() {
        fixture.setInputDialogResult("Browser Eval", null)

        BrowserEvalAction().actionPerformed(fixture.event)

        verify(exactly = 0) { fixture.client.browserEval(any(), any()) }
    }

    @Test
    fun `BrowserEvalAction shows result on success`() {
        fixture.setInputDialogResult("Browser Eval", "document.title")
        every { fixture.client.browserEval("document.title", any()) } returns
            Result.success(BrowserEvalResponse(success = true, result = "My Page Title"))

        BrowserEvalAction().actionPerformed(fixture.event)

        verify { fixture.client.browserEval("document.title", any()) }
        verify { Messages.showInfoMessage(any<Project>(), any<String>(), eq("Browser Eval")) }
    }

    // ========================================================================
    // BrowserConsoleAction Tests
    // ========================================================================

    @Test
    fun `BrowserConsoleAction shows no messages when empty`() {
        every { fixture.client.browserConsole(any(), any(), any()) } returns
            Result.success(BrowserConsoleResponse(success = true, messages = emptyList()))

        BrowserConsoleAction().actionPerformed(fixture.event)

        verify { Messages.showInfoMessage(any<Project>(), eq("No console messages"), any<String>()) }
    }

    @Test
    fun `BrowserConsoleAction shows console messages`() {
        every { fixture.client.browserConsole(any(), any(), any()) } returns
            Result.success(
                BrowserConsoleResponse(
                    success = true,
                    messages =
                        listOf(
                            BrowserConsoleMessage(level = "log", text = "Hello world"),
                            BrowserConsoleMessage(level = "error", text = "Something went wrong"),
                        ),
                    count = 2,
                ),
            )

        BrowserConsoleAction().actionPerformed(fixture.event)

        verify { Messages.showInfoMessage(any<Project>(), any<String>(), eq("Browser Console")) }
    }

    // ========================================================================
    // BrowserNetworkAction Tests
    // ========================================================================

    @Test
    fun `BrowserNetworkAction shows no requests message when empty`() {
        every { fixture.client.browserNetwork(any(), any(), any()) } returns
            Result.success(BrowserNetworkResponse(success = true, requests = emptyList()))

        BrowserNetworkAction().actionPerformed(fixture.event)

        verify { Messages.showInfoMessage(any<Project>(), eq("No network requests"), any<String>()) }
    }

    @Test
    fun `BrowserNetworkAction shows network requests`() {
        every { fixture.client.browserNetwork(any(), any(), any()) } returns
            Result.success(
                BrowserNetworkResponse(
                    success = true,
                    requests =
                        listOf(
                            BrowserNetworkEntry(
                                method = "GET",
                                url = "https://api.example.com/data",
                                status = 200,
                                statusText = "OK",
                                timestamp = "2024-01-01T12:00:00Z",
                            ),
                            BrowserNetworkEntry(
                                method = "POST",
                                url = "https://api.example.com/submit",
                                status = 201,
                                statusText = "Created",
                                timestamp = "2024-01-01T12:00:01Z",
                            ),
                        ),
                    count = 2,
                ),
            )

        BrowserNetworkAction().actionPerformed(fixture.event)

        verify { Messages.showInfoMessage(any<Project>(), any<String>(), eq("Browser Network")) }
    }

    // ========================================================================
    // Update Tests (presentation state)
    // ========================================================================

    @Test
    fun `browser actions disable when not connected`() {
        fixture.setConnected(false)

        val actions =
            listOf(
                BrowserStatusAction(),
                BrowserTabsAction(),
                BrowserGotoAction(),
                BrowserNavigateAction(),
                BrowserReloadAction(),
                BrowserScreenshotAction(),
                BrowserClickAction(),
                BrowserTypeAction(),
                BrowserEvalAction(),
                BrowserConsoleAction(),
                BrowserNetworkAction(),
            )

        for (action in actions) {
            fixture.resetPresentation()
            action.update(fixture.event)
            assert(!fixture.presentation.isEnabled) { "${action::class.simpleName} should be disabled" }
        }
    }

    @Test
    fun `browser actions enable when connected`() {
        fixture.setConnected(true)

        val actions =
            listOf(
                BrowserStatusAction(),
                BrowserTabsAction(),
                BrowserGotoAction(),
                BrowserNavigateAction(),
                BrowserReloadAction(),
                BrowserScreenshotAction(),
                BrowserClickAction(),
                BrowserTypeAction(),
                BrowserEvalAction(),
                BrowserConsoleAction(),
                BrowserNetworkAction(),
            )

        for (action in actions) {
            fixture.resetPresentation()
            action.update(fixture.event)
            assert(fixture.presentation.isEnabled) { "${action::class.simpleName} should be enabled" }
        }
    }
}
