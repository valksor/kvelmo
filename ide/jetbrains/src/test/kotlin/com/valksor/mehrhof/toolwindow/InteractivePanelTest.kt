package com.valksor.mehrhof.toolwindow

import com.intellij.openapi.actionSystem.ActionManager
import com.intellij.openapi.actionSystem.ActionToolbar
import com.intellij.openapi.actionSystem.AnAction
import com.intellij.openapi.project.Project
import com.valksor.mehrhof.api.MehrhofApiClient
import com.valksor.mehrhof.api.models.InteractiveStateResponse
import com.valksor.mehrhof.api.models.TaskInfo
import com.valksor.mehrhof.api.models.TaskWork
import com.valksor.mehrhof.services.MehrhofProjectService
import io.mockk.every
import io.mockk.mockk
import io.mockk.mockkStatic
import io.mockk.unmockkAll
import io.mockk.verify
import org.junit.jupiter.api.AfterEach
import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.Assertions.assertTrue
import org.junit.jupiter.api.BeforeEach
import org.junit.jupiter.api.Test
import javax.swing.JButton
import javax.swing.JEditorPane
import javax.swing.SwingUtilities

/**
 * Unit tests for [InteractivePanel].
 *
 * Tests StateListener callbacks, command dispatch, and UI state updates.
 */
class InteractivePanelTest {
    private lateinit var project: Project
    private lateinit var service: MehrhofProjectService
    private lateinit var client: MehrhofApiClient
    private lateinit var panel: InteractivePanel

    @BeforeEach
    fun setUp() {
        project = mockk(relaxed = true)
        service = mockk(relaxed = true)
        client = mockk(relaxed = true)

        mockkStatic(MehrhofProjectService::class)
        every { MehrhofProjectService.getInstance(project) } returns service
        every { service.getApiClient() } returns client
        every { service.isConnected() } returns false
        every { service.isServerRunning() } returns false
        every { service.workflowState } returns "idle"
        every { service.currentTask } returns null
        every { service.currentTaskWork } returns null

        // Mock API calls that may be triggered during state listener callbacks
        every { client.getInteractiveState() } returns
            Result.success(
                InteractiveStateResponse(success = true, state = "idle"),
            )

        // Mock ActionManager for toolbar
        mockkStatic(ActionManager::class)
        val actionManager = mockk<ActionManager>(relaxed = true)
        every { ActionManager.getInstance() } returns actionManager
        every { actionManager.getAction(any<String>()) } returns mockk<AnAction>(relaxed = true)
        every { actionManager.createActionToolbar(any(), any(), any()) } returns mockk<ActionToolbar>(relaxed = true)

        panel = InteractivePanel(project, service)
    }

    @AfterEach
    fun tearDown() {
        panel.dispose()
        unmockkAll()
    }

    // ========================================================================
    // Initialization Tests
    // ========================================================================

    @Test
    fun `init registers as state listener`() {
        verify { service.addStateListener(panel) }
    }

    @Test
    fun `dispose removes state listener`() {
        panel.dispose()

        verify { service.removeStateListener(panel) }
    }

    @Test
    fun `initial welcome message is shown`() {
        val messagesPane = getField<JEditorPane>("messagesPane")
        assertTrue(messagesPane.text.contains("Welcome to Mehrhof Interactive Mode"))
    }

    // ========================================================================
    // Connection Status Tests
    // ========================================================================

    @Test
    fun `start stop button shows Start Server when not running`() {
        every { service.isServerRunning() } returns false

        panel = InteractivePanel(project, service)

        val button = getField<JButton>("startStopButton")
        assertEquals("Start Server", button.text)
    }

    @Test
    fun `start stop button shows Stop Server when running`() {
        every { service.isServerRunning() } returns true
        every { service.isConnected() } returns true
        every { service.getServerPort() } returns 8080

        panel = InteractivePanel(project, service)

        val button = getField<JButton>("startStopButton")
        assertEquals("Stop Server", button.text)
    }

    // ========================================================================
    // StateListener Callback Tests
    // ========================================================================

    // Note: onConnectionChanged(true) triggers refreshInteractiveState() which uses
    // coroutines with Dispatchers.IO. Testing this properly requires kotlinx-coroutines-test.
    // Instead, we test the disconnect path which doesn't trigger async operations.

    @Test
    fun `onConnectionChanged appends disconnected message on false`() {
        panel.onConnectionChanged(false)
        SwingUtilities.invokeAndWait { }

        val messagesPane = getField<JEditorPane>("messagesPane")
        assertTrue(messagesPane.text.contains("Disconnected from server"))
    }

    @Test
    fun `onWorkflowStateChanged shows state transition message`() {
        panel.onWorkflowStateChanged("planning", "idle")
        SwingUtilities.invokeAndWait { }

        val messagesPane = getField<JEditorPane>("messagesPane")
        assertTrue(messagesPane.text.contains("idle"))
        assertTrue(messagesPane.text.contains("planning"))
    }

    @Test
    fun `onTaskChanged updates task label with task info`() {
        val task = TaskInfo(id = "abc123def", state = "planning", ref = "feature/auth")
        val work = TaskWork(title = "Add authentication")

        every { service.currentTask } returns task
        every { service.currentTaskWork } returns work

        panel.onTaskChanged(task, work)
        SwingUtilities.invokeAndWait { }

        val taskIdLabel = getField<com.intellij.ui.components.JBLabel>("taskIdLabel")
        assertTrue(taskIdLabel.text.contains("abc123d"))
        assertTrue(taskIdLabel.text.contains("Add authentication"))
    }

    @Test
    fun `onTaskChanged shows No active task when null`() {
        every { service.currentTask } returns null
        every { service.currentTaskWork } returns null

        panel.onTaskChanged(null, null)
        SwingUtilities.invokeAndWait { }

        val taskIdLabel = getField<com.intellij.ui.components.JBLabel>("taskIdLabel")
        assertEquals("No active task", taskIdLabel.text)
    }

    @Test
    fun `onQuestionReceived shows question in messages`() {
        panel.onQuestionReceived("What database should I use?", null)
        SwingUtilities.invokeAndWait { }

        val messagesPane = getField<JEditorPane>("messagesPane")
        assertTrue(messagesPane.text.contains("Agent asks: What database should I use?"))
    }

    @Test
    fun `onQuestionReceived shows options when provided`() {
        panel.onQuestionReceived("Choose framework:", listOf("React", "Vue", "Angular"))
        SwingUtilities.invokeAndWait { }

        val messagesPane = getField<JEditorPane>("messagesPane")
        assertTrue(messagesPane.text.contains("Options: React, Vue, Angular"))
    }

    @Test
    fun `onAgentMessage shows assistant message`() {
        panel.onAgentMessage("I will implement the feature now.", null)
        SwingUtilities.invokeAndWait { }

        val messagesPane = getField<JEditorPane>("messagesPane")
        assertTrue(messagesPane.text.contains("Agent:"))
        assertTrue(messagesPane.text.contains("I will implement the feature now"))
    }

    @Test
    fun `onError shows error message`() {
        panel.onError("Failed to connect to server")
        SwingUtilities.invokeAndWait { }

        val messagesPane = getField<JEditorPane>("messagesPane")
        assertTrue(messagesPane.text.contains("Error:"))
        assertTrue(messagesPane.text.contains("Failed to connect to server"))
    }

    // ========================================================================
    // Command History Tests
    // ========================================================================

    @Test
    fun `command history starts empty`() {
        val commandHistory = getField<MutableList<String>>("commandHistory")
        assertTrue(commandHistory.isEmpty())
    }

    // ========================================================================
    // Helper Methods
    // ========================================================================

    @Suppress("UNCHECKED_CAST")
    private fun <T> getField(name: String): T {
        val field = InteractivePanel::class.java.getDeclaredField(name)
        field.isAccessible = true
        return field.get(panel) as T
    }
}
