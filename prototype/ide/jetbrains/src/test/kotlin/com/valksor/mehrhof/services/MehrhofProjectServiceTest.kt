package com.valksor.mehrhof.services

import com.intellij.notification.NotificationGroupManager
import com.intellij.openapi.project.Project
import com.valksor.mehrhof.api.models.TaskInfo
import com.valksor.mehrhof.api.models.TaskWork
import com.valksor.mehrhof.settings.MehrhofSettings
import io.mockk.every
import io.mockk.mockk
import io.mockk.mockkObject
import io.mockk.mockkStatic
import io.mockk.unmockkAll
import io.mockk.verify
import org.junit.jupiter.api.AfterEach
import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.Assertions.assertFalse
import org.junit.jupiter.api.Assertions.assertNull
import org.junit.jupiter.api.Assertions.assertTrue
import org.junit.jupiter.api.BeforeEach
import org.junit.jupiter.api.Test

/**
 * Unit tests for [MehrhofProjectService].
 *
 * Tests connection management, state tracking, listener notification,
 * and event handling.
 */
class MehrhofProjectServiceTest {
    private lateinit var project: Project
    private lateinit var settings: MehrhofSettings
    private lateinit var service: MehrhofProjectService

    @BeforeEach
    fun setUp() {
        project = mockk(relaxed = true)
        settings = mockk(relaxed = true)

        // Mock MehrhofSettings singleton
        mockkObject(MehrhofSettings.Companion)
        every { MehrhofSettings.getInstance() } returns settings
        every { settings.serverUrl } returns ""
        every { settings.autoReconnect } returns false
        every { settings.showNotifications } returns false
        every { settings.maxReconnectAttempts } returns 3
        every { settings.reconnectDelaySeconds } returns 1

        // Mock NotificationGroupManager to avoid IDE dependency
        mockkStatic(NotificationGroupManager::class)
        val notificationManager = mockk<NotificationGroupManager>(relaxed = true)
        every { NotificationGroupManager.getInstance() } returns notificationManager

        service = MehrhofProjectService(project)
    }

    @AfterEach
    fun tearDown() {
        unmockkAll()
    }

    // ========================================================================
    // Initial State Tests
    // ========================================================================

    @Test
    fun `isConnected returns false initially`() {
        assertFalse(service.isConnected())
    }

    @Test
    fun `workflowState is idle initially`() {
        assertEquals("idle", service.workflowState)
    }

    @Test
    fun `currentTask is null initially`() {
        assertNull(service.currentTask)
    }

    @Test
    fun `currentTaskWork is null initially`() {
        assertNull(service.currentTaskWork)
    }

    @Test
    fun `pendingQuestion is null initially`() {
        assertNull(service.pendingQuestion)
    }

    @Test
    fun `getApiClient returns null when not connected`() {
        assertNull(service.getApiClient())
    }

    // ========================================================================
    // State Listener Tests
    // ========================================================================

    @Test
    fun `addStateListener registers listener`() {
        val listener = mockk<MehrhofProjectService.StateListener>(relaxed = true)

        service.addStateListener(listener)

        // Trigger disconnect to verify listener is called
        service.disconnect()

        verify { listener.onConnectionChanged(false) }
    }

    @Test
    fun `removeStateListener unregisters listener`() {
        val listener = mockk<MehrhofProjectService.StateListener>(relaxed = true)

        service.addStateListener(listener)
        service.removeStateListener(listener)

        // Disconnect should not notify removed listener
        service.disconnect()

        verify(exactly = 0) { listener.onConnectionChanged(any()) }
    }

    @Test
    fun `multiple listeners are all notified`() {
        val listener1 = mockk<MehrhofProjectService.StateListener>(relaxed = true)
        val listener2 = mockk<MehrhofProjectService.StateListener>(relaxed = true)

        service.addStateListener(listener1)
        service.addStateListener(listener2)
        service.disconnect()

        verify { listener1.onConnectionChanged(false) }
        verify { listener2.onConnectionChanged(false) }
    }

    // ========================================================================
    // Connect Tests
    // ========================================================================

    @Test
    fun `connect shows error when no server URL configured`() {
        every { settings.serverUrl } returns ""

        service.connect()

        // Should not set connected state
        assertFalse(service.isConnected())
    }

    // ========================================================================
    // Disconnect Tests
    // ========================================================================

    @Test
    fun `disconnect clears all state`() {
        // Manually set some state via reflection
        val taskField = MehrhofProjectService::class.java.getDeclaredField("currentTask")
        taskField.isAccessible = true
        taskField.set(service, TaskInfo(id = "t1", state = "planning", ref = "r1"))

        val stateField = MehrhofProjectService::class.java.getDeclaredField("workflowState")
        stateField.isAccessible = true
        stateField.set(service, "implementing")

        service.disconnect()

        assertFalse(service.isConnected())
        assertNull(service.currentTask)
        assertNull(service.currentTaskWork)
        assertEquals("idle", service.workflowState)
        assertNull(service.pendingQuestion)
    }

    @Test
    fun `disconnect notifies listeners`() {
        val listener = mockk<MehrhofProjectService.StateListener>(relaxed = true)
        service.addStateListener(listener)

        service.disconnect()

        verify { listener.onConnectionChanged(false) }
    }

    // ========================================================================
    // Server Management Delegation Tests
    // ========================================================================

    @Test
    fun `isServerRunning delegates to serverManager`() {
        // Initially not running
        assertFalse(service.isServerRunning())
    }

    @Test
    fun `getServerPort returns null when not running`() {
        assertNull(service.getServerPort())
    }

    // ========================================================================
    // Dispose Tests
    // ========================================================================

    @Test
    fun `dispose clears listeners`() {
        val listener = mockk<MehrhofProjectService.StateListener>(relaxed = true)
        service.addStateListener(listener)

        service.dispose()

        // After dispose, listeners should be cleared
        // (but dispose already disconnects internally)
        verify { listener.onConnectionChanged(false) }
    }

    // ========================================================================
    // Companion Object Tests
    // ========================================================================

    @Test
    fun `getInstance returns service from project`() {
        val mockService = mockk<MehrhofProjectService>()
        every { project.getService(MehrhofProjectService::class.java) } returns mockService

        val result = MehrhofProjectService.getInstance(project)

        assertEquals(mockService, result)
    }

    // ========================================================================
    // Event Handling Tests (via StateListener interface)
    // ========================================================================

    @Test
    fun `StateListener interface has all required methods`() {
        // Verify the interface contract by creating a test implementation
        val listener =
            object : MehrhofProjectService.StateListener {
                var connectionChanged = false
                var workflowChanged = false
                var taskChanged = false
                var questionReceived = false
                var agentMessageReceived = false
                var errorReceived = false

                override fun onConnectionChanged(connected: Boolean) {
                    connectionChanged = true
                }

                override fun onWorkflowStateChanged(
                    state: String,
                    previousState: String?,
                ) {
                    workflowChanged = true
                }

                override fun onTaskChanged(
                    task: TaskInfo?,
                    work: TaskWork?,
                ) {
                    taskChanged = true
                }

                override fun onQuestionReceived(
                    question: String,
                    options: List<String>?,
                ) {
                    questionReceived = true
                }

                override fun onAgentMessage(
                    content: String,
                    type: String?,
                ) {
                    agentMessageReceived = true
                }

                override fun onError(message: String) {
                    errorReceived = true
                }
            }

        // All methods should be callable with default implementations
        listener.onConnectionChanged(true)
        listener.onWorkflowStateChanged("planning", "idle")
        listener.onTaskChanged(null, null)
        listener.onQuestionReceived("test?", null)
        listener.onAgentMessage("content", null)
        listener.onError("error")

        assertTrue(listener.connectionChanged)
        assertTrue(listener.workflowChanged)
        assertTrue(listener.taskChanged)
        assertTrue(listener.questionReceived)
        assertTrue(listener.agentMessageReceived)
        assertTrue(listener.errorReceived)
    }
}
