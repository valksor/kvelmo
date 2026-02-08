package com.valksor.mehrhof.testutil

import com.intellij.openapi.actionSystem.AnActionEvent
import com.intellij.openapi.actionSystem.Presentation
import com.intellij.openapi.progress.ProgressIndicator
import com.intellij.openapi.progress.ProgressManager
import com.intellij.openapi.progress.Task
import com.intellij.openapi.project.Project
import com.intellij.openapi.ui.Messages
import com.valksor.mehrhof.api.MehrhofApiClient
import com.valksor.mehrhof.api.models.AddNoteResponse
import com.valksor.mehrhof.api.models.AllCostsResponse
import com.valksor.mehrhof.api.models.BrowserCloseResponse
import com.valksor.mehrhof.api.models.BrowserClickResponse
import com.valksor.mehrhof.api.models.BrowserConsoleResponse
import com.valksor.mehrhof.api.models.BrowserEvalResponse
import com.valksor.mehrhof.api.models.BrowserGotoResponse
import com.valksor.mehrhof.api.models.BrowserNavigateResponse
import com.valksor.mehrhof.api.models.BrowserNetworkResponse
import com.valksor.mehrhof.api.models.BrowserReloadResponse
import com.valksor.mehrhof.api.models.BrowserScreenshotResponse
import com.valksor.mehrhof.api.models.BrowserStatusResponse
import com.valksor.mehrhof.api.models.BrowserTabsResponse
import com.valksor.mehrhof.api.models.BrowserTypeResponse
import com.valksor.mehrhof.api.models.ContinueResponse
import com.valksor.mehrhof.api.models.EntityLinksResponse
import com.valksor.mehrhof.api.models.GrandTotal
import com.valksor.mehrhof.api.models.FindSearchResponse
import com.valksor.mehrhof.api.models.InteractiveCommandResponse
import com.valksor.mehrhof.api.models.LibraryCollection
import com.valksor.mehrhof.api.models.LibraryListResponse
import com.valksor.mehrhof.api.models.LibraryShowResponse
import com.valksor.mehrhof.api.models.LibraryStatsResponse
import com.valksor.mehrhof.api.models.LinksListResponse
import com.valksor.mehrhof.api.models.LinksSearchResponse
import com.valksor.mehrhof.api.models.LinksStatsResponse
import com.valksor.mehrhof.api.models.MemoryIndexResponse
import com.valksor.mehrhof.api.models.MemorySearchResponse
import com.valksor.mehrhof.api.models.MemoryStatsResponse
import com.valksor.mehrhof.api.models.TaskCostResponse
import com.valksor.mehrhof.api.models.WorkflowResponse
import com.valksor.mehrhof.api.browserClick
import com.valksor.mehrhof.api.browserClose
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
import com.valksor.mehrhof.api.createQuickTask
import com.valksor.mehrhof.api.deleteQueueTask
import com.valksor.mehrhof.api.exportQueueTask
import com.valksor.mehrhof.api.find
import com.valksor.mehrhof.api.libraryList
import com.valksor.mehrhof.api.libraryPull
import com.valksor.mehrhof.api.libraryRemove
import com.valksor.mehrhof.api.libraryShow
import com.valksor.mehrhof.api.libraryStats
import com.valksor.mehrhof.api.linksGet
import com.valksor.mehrhof.api.linksList
import com.valksor.mehrhof.api.linksRebuild
import com.valksor.mehrhof.api.linksSearch
import com.valksor.mehrhof.api.linksStats
import com.valksor.mehrhof.api.memoryIndex
import com.valksor.mehrhof.api.memorySearch
import com.valksor.mehrhof.api.memoryStats
import com.valksor.mehrhof.api.optimizeQueueTask
import com.valksor.mehrhof.api.submitQueueTask
import com.valksor.mehrhof.api.syncTask
import com.valksor.mehrhof.services.MehrhofProjectService
import com.intellij.openapi.ui.InputValidator
import io.mockk.Runs
import io.mockk.every
import io.mockk.just
import io.mockk.mockk
import io.mockk.mockkStatic
import io.mockk.unmockkAll
import javax.swing.Icon

/**
 * Shared test fixture for action tests.
 *
 * Provides pre-configured mocks for:
 * - [AnActionEvent] and [Presentation]
 * - [Project]
 * - [MehrhofProjectService] (via mockkStatic)
 * - [MehrhofApiClient] with default success responses
 * - [ProgressManager] (runs tasks synchronously)
 * - [Messages] (stubs all dialog methods)
 *
 * Usage:
 * ```kotlin
 * class MyActionsTest {
 *     private lateinit var fixture: ActionTestFixture
 *
 *     @BeforeEach
 *     fun setUp() {
 *         fixture = ActionTestFixture()
 *         fixture.setUp()
 *     }
 *
 *     @AfterEach
 *     fun tearDown() {
 *         fixture.tearDown()
 *     }
 *
 *     @Test
 *     fun `test my action`() {
 *         MyAction().actionPerformed(fixture.event)
 *         verify { fixture.client.someMethod() }
 *     }
 * }
 * ```
 */
class ActionTestFixture {
    lateinit var event: AnActionEvent
    lateinit var project: Project
    lateinit var service: MehrhofProjectService
    lateinit var client: MehrhofApiClient
    lateinit var presentation: Presentation

    /**
     * Initialize all mocks and set up default behaviors.
     * Call this in @BeforeEach.
     */
    fun setUp() {
        event = mockk(relaxed = true)
        project = mockk(relaxed = true)
        service = mockk(relaxed = true)
        client = mockk(relaxed = true)
        presentation = Presentation()

        every { event.project } returns project
        every { event.presentation } returns presentation

        setUpServiceMock()
        setUpProgressManagerMock()
        setUpMessagesMock()
        setUpClientMock()
        setUpExtensionMocks()
    }

    /**
     * Clean up all mocks.
     * Call this in @AfterEach.
     */
    fun tearDown() {
        unmockkAll()
    }

    /**
     * Reset presentation to a fresh instance.
     * Useful when testing multiple update() calls.
     */
    fun resetPresentation() {
        presentation = Presentation()
        every { event.presentation } returns presentation
    }

    /**
     * Configure the service to return a connected state.
     */
    fun setConnected(connected: Boolean) {
        every { service.isConnected() } returns connected
    }

    /**
     * Configure Messages.showInputDialog to return a specific value.
     */
    fun setInputDialogResult(
        title: String,
        result: String?,
    ) {
        every {
            Messages.showInputDialog(any<Project>(), any<String>(), eq(title), any<Icon>())
        } returns result
    }

    /**
     * Configure Messages.showInputDialog (6-arg version with default value) to return a specific value.
     */
    fun setInputDialogWithDefaultResult(
        title: String,
        result: String?,
    ) {
        every {
            Messages.showInputDialog(
                any<Project>(),
                any<String>(),
                eq(title),
                any<Icon>(),
                any<String>(),
                any<InputValidator>(),
            )
        } returns result
    }

    /**
     * Configure Messages.showYesNoDialog to return YES or NO.
     */
    fun setYesNoDialogResult(
        title: String,
        result: Int,
    ) {
        every {
            Messages.showYesNoDialog(any<Project>(), any<String>(), eq(title), any<Icon>())
        } returns result
    }

    private fun setUpServiceMock() {
        mockkStatic(MehrhofProjectService::class)
        every { MehrhofProjectService.getInstance(project) } returns service
        every { service.getApiClient() } returns client
        every { service.isConnected() } returns true
    }

    private fun setUpProgressManagerMock() {
        mockkStatic(ProgressManager::class)
        val progressManager = mockk<ProgressManager>(relaxed = true)
        every { ProgressManager.getInstance() } returns progressManager
        every { progressManager.run(any<Task>()) } answers {
            (firstArg<Task>() as Task.Backgroundable).run(mockk<ProgressIndicator>(relaxed = true))
        }
    }

    private fun setUpMessagesMock() {
        mockkStatic(Messages::class)
        every {
            Messages.showErrorDialog(any<Project>(), any<String>(), any<String>())
        } just Runs
        every {
            Messages.showInfoMessage(any<Project>(), any<String>(), any<String>())
        } just Runs
        // 4-arg version of showInputDialog (Project, message, title, icon)
        every {
            Messages.showInputDialog(any<Project>(), any<String>(), any<String>(), any<Icon>())
        } returns null
        // 6-arg version of showInputDialog with initialValue and validator
        every {
            Messages.showInputDialog(
                any<Project>(),
                any<String>(),
                any<String>(),
                any<Icon>(),
                any<String>(),
                any<InputValidator>(),
            )
        } returns null
        every {
            Messages.showYesNoDialog(any<Project>(), any<String>(), any<String>(), any<Icon>())
        } returns Messages.NO
        // showEditableChooseDialog for dropdown selections
        every {
            Messages.showEditableChooseDialog(
                any<String>(),
                any<String>(),
                any<Icon>(),
                any<Array<String>>(),
                any<String>(),
                any<InputValidator>(),
            )
        } returns null
    }

    private fun setUpClientMock() {
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
        every { client.addNote(any(), any()) } returns Result.success(AddNoteResponse(success = true))
        every { client.executeCommand(any(), any()) } returns okCommand
        every { client.getTaskCosts(any()) } returns
            Result.success(
                TaskCostResponse(
                    taskId = "t1",
                    title = "Test",
                    totalCostUsd = 0.01,
                    inputTokens = 100,
                    outputTokens = 50,
                    cachedTokens = 10,
                    cachedPercent = 10.0,
                    totalTokens = 150,
                ),
            )
        every { client.getAllCosts() } returns
            Result.success(
                AllCostsResponse(
                    tasks = emptyList(),
                    grandTotal =
                        GrandTotal(
                            costUsd = 0.05,
                            inputTokens = 500,
                            outputTokens = 250,
                            cachedTokens = 50,
                            totalTokens = 750,
                        ),
                ),
            )
    }

    private fun setUpExtensionMocks() {
        mockkStatic("com.valksor.mehrhof.api.MehrhofApiClientExtensionsKt")

        val okCommand = Result.success(InteractiveCommandResponse(success = true))

        // Queue task extensions
        every { client.createQuickTask(any()) } returns okCommand
        every { client.deleteQueueTask(any(), any()) } returns okCommand
        every { client.exportQueueTask(any(), any()) } returns okCommand
        every { client.optimizeQueueTask(any(), any()) } returns okCommand
        every { client.submitQueueTask(any(), any(), any()) } returns okCommand
        every { client.syncTask() } returns okCommand

        // Find extension
        every { client.find(any()) } returns
            Result.success(FindSearchResponse(query = "", count = 0, matches = emptyList()))

        // Memory extensions
        every { client.memorySearch(any()) } returns
            Result.success(MemorySearchResponse(results = emptyList(), count = 0))
        every { client.memoryIndex(any()) } returns
            Result.success(MemoryIndexResponse(success = true))
        every { client.memoryStats() } returns
            Result.success(MemoryStatsResponse(totalDocuments = 0, byType = emptyMap(), enabled = true))

        // Library extensions
        every { client.libraryList() } returns
            Result.success(LibraryListResponse(collections = emptyList(), count = 0))
        every { client.libraryShow(any()) } returns
            Result.success(
                LibraryShowResponse(
                    collection =
                        LibraryCollection(
                            id = "c1",
                            name = "test",
                            source = "local",
                            sourceType = "directory",
                            includeMode = "all",
                            pageCount = 0,
                            totalSize = 0,
                            location = "/tmp",
                        ),
                    pages = emptyList(),
                ),
            )
        every { client.libraryStats() } returns
            Result.success(
                LibraryStatsResponse(
                    totalCollections = 0,
                    totalPages = 0,
                    totalSize = 0,
                    projectCount = 0,
                    sharedCount = 0,
                    byMode = emptyMap(),
                    enabled = true,
                ),
            )
        every { client.libraryPull(any(), any(), any()) } returns okCommand
        every { client.libraryRemove(any()) } returns okCommand

        // Links extensions
        every { client.linksList() } returns
            Result.success(LinksListResponse(links = emptyList(), count = 0))
        every { client.linksGet(any()) } returns
            Result.success(EntityLinksResponse(entityId = "e1", outgoing = emptyList(), incoming = emptyList()))
        every { client.linksSearch(any()) } returns
            Result.success(LinksSearchResponse(query = "", results = emptyList(), count = 0))
        every { client.linksStats() } returns
            Result.success(
                LinksStatsResponse(
                    totalLinks = 0,
                    totalSources = 0,
                    totalTargets = 0,
                    orphanEntities = 0,
                    mostLinked = emptyList(),
                    enabled = true,
                ),
            )
        every { client.linksRebuild() } returns okCommand

        // Browser extensions
        every { client.browserStatus() } returns
            Result.success(BrowserStatusResponse(connected = false, tabs = emptyList()))
        every { client.browserTabs() } returns
            Result.success(BrowserTabsResponse(tabs = emptyList(), count = 0))
        every { client.browserGoto(any()) } returns
            Result.success(BrowserGotoResponse(success = true, tab = null))
        every { client.browserNavigate(any(), any()) } returns
            Result.success(BrowserNavigateResponse(success = true))
        every { client.browserClick(any(), any()) } returns
            Result.success(BrowserClickResponse(success = true))
        every { client.browserType(any(), any(), any(), any()) } returns
            Result.success(BrowserTypeResponse(success = true))
        every { client.browserEval(any(), any()) } returns
            Result.success(BrowserEvalResponse(success = true, result = null))
        every { client.browserScreenshot(any(), any(), any()) } returns
            Result.success(BrowserScreenshotResponse(success = true))
        every { client.browserReload(any(), any()) } returns
            Result.success(BrowserReloadResponse(success = true))
        every { client.browserClose(any()) } returns
            Result.success(BrowserCloseResponse(success = true))
        every { client.browserConsole(any(), any(), any()) } returns
            Result.success(BrowserConsoleResponse(success = true, messages = emptyList()))
        every { client.browserNetwork(any(), any(), any()) } returns
            Result.success(BrowserNetworkResponse(success = true, requests = emptyList()))
    }
}
