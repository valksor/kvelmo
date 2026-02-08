package com.valksor.mehrhof.toolwindow

import com.intellij.openapi.actionSystem.ActionManager
import com.intellij.openapi.actionSystem.ActionToolbar
import com.intellij.openapi.actionSystem.AnAction
import com.intellij.openapi.project.Project
import com.intellij.ui.components.JBLabel
import com.valksor.mehrhof.api.MehrhofApiClient
import com.valksor.mehrhof.api.models.TaskInfo
import com.valksor.mehrhof.api.models.TaskSummary
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
import javax.swing.DefaultListModel
import javax.swing.JButton
import javax.swing.SwingUtilities

/**
 * Unit tests for [TaskListPanel].
 *
 * Tests StateListener callbacks, workflow button states, and task list updates.
 */
class TaskListPanelTest {
    private lateinit var project: Project
    private lateinit var service: MehrhofProjectService
    private lateinit var client: MehrhofApiClient
    private lateinit var panel: TaskListPanel

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

        // Mock ActionManager for toolbar
        mockkStatic(ActionManager::class)
        val actionManager = mockk<ActionManager>(relaxed = true)
        every { ActionManager.getInstance() } returns actionManager
        every { actionManager.getAction(any<String>()) } returns mockk<AnAction>(relaxed = true)
        every { actionManager.createActionToolbar(any(), any(), any()) } returns mockk<ActionToolbar>(relaxed = true)

        panel = TaskListPanel(project, service)
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
    fun `initial connection status shows not running`() {
        every { service.isServerRunning() } returns false

        panel = TaskListPanel(project, service)

        val connectionLabel = getField<JBLabel>("connectionStatusLabel")
        assertEquals("Not running", connectionLabel.text)
    }

    // ========================================================================
    // Server Control Tests
    // ========================================================================

    @Test
    fun `start stop button shows Start Server when not running`() {
        every { service.isServerRunning() } returns false

        panel = TaskListPanel(project, service)

        val button = getField<JButton>("startStopButton")
        assertEquals("Start Server", button.text)
    }

    @Test
    fun `start stop button shows Stop Server when running`() {
        every { service.isServerRunning() } returns true
        every { service.isConnected() } returns true
        every { service.getServerPort() } returns 8080

        panel = TaskListPanel(project, service)

        val button = getField<JButton>("startStopButton")
        assertEquals("Stop Server", button.text)
    }

    // ========================================================================
    // StateListener Callback Tests
    // ========================================================================

    @Test
    fun `onError updates connection status to Error`() {
        panel.onError("Connection lost")
        SwingUtilities.invokeAndWait { }

        val connectionLabel = getField<JBLabel>("connectionStatusLabel")
        assertEquals("Error", connectionLabel.text)
    }

    @Test
    fun `onTaskChanged does not throw for null values`() {
        // Should not throw
        panel.onTaskChanged(null, null)
        SwingUtilities.invokeAndWait { }
    }

    @Test
    fun `onTaskChanged does not throw for valid task`() {
        val task = TaskInfo(id = "test-task", state = "planning", ref = "feature/x")
        val work = TaskWork(title = "Test Task")

        every { service.currentTask } returns task
        every { service.currentTaskWork } returns work
        every { service.workflowState } returns "planning"

        // Should not throw
        panel.onTaskChanged(task, work)
        SwingUtilities.invokeAndWait { }
    }

    @Test
    fun `onWorkflowStateChanged does not throw`() {
        every { service.workflowState } returns "implementing"

        // Should not throw
        panel.onWorkflowStateChanged("implementing", "planning")
        SwingUtilities.invokeAndWait { }
    }

    // ========================================================================
    // Task List Tests
    // ========================================================================

    @Test
    fun `task list model starts empty`() {
        val listModel = getField<DefaultListModel<TaskSummary>>("taskListModel")
        assertTrue(listModel.isEmpty)
    }

    // ========================================================================
    // WorkflowAction Enum Tests
    // ========================================================================

    @Test
    fun `WorkflowAction enum contains all actions`() {
        val actions = WorkflowAction.values()

        assertEquals(6, actions.size)
        assertTrue(actions.contains(WorkflowAction.PLAN))
        assertTrue(actions.contains(WorkflowAction.IMPLEMENT))
        assertTrue(actions.contains(WorkflowAction.REVIEW))
        assertTrue(actions.contains(WorkflowAction.FINISH))
        assertTrue(actions.contains(WorkflowAction.UNDO))
        assertTrue(actions.contains(WorkflowAction.REDO))
    }

    // ========================================================================
    // Helper Methods
    // ========================================================================

    @Suppress("UNCHECKED_CAST")
    private fun <T> getField(name: String): T {
        val field = TaskListPanel::class.java.getDeclaredField(name)
        field.isAccessible = true
        return field.get(panel) as T
    }
}
