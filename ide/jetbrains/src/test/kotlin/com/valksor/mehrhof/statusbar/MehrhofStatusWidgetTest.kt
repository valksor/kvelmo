package com.valksor.mehrhof.statusbar

import com.intellij.openapi.project.Project
import com.intellij.openapi.wm.StatusBar
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
import java.awt.event.MouseEvent

/**
 * Unit tests for [MehrhofStatusWidget] and [MehrhofStatusWidgetFactory].
 *
 * Tests status bar widget text, tooltip, and state change handling.
 */
class MehrhofStatusWidgetTest {
    private lateinit var project: Project
    private lateinit var service: MehrhofProjectService
    private lateinit var widget: MehrhofStatusWidget

    @BeforeEach
    fun setUp() {
        project = mockk(relaxed = true)
        service = mockk(relaxed = true)

        mockkStatic(MehrhofProjectService::class)
        every { MehrhofProjectService.getInstance(project) } returns service

        // Default to connected state
        every { service.isConnected() } returns true
        every { service.workflowState } returns "idle"
        every { service.currentTask } returns null
        every { service.currentTaskWork } returns null
        every { service.pendingQuestion } returns null

        widget = MehrhofStatusWidget(project)
    }

    @AfterEach
    fun tearDown() {
        unmockkAll()
    }

    // ========================================================================
    // Factory Tests
    // ========================================================================

    @Test
    fun `factory getId returns correct ID`() {
        val factory = MehrhofStatusWidgetFactory()
        assertEquals("MehrhofStatusWidget", factory.id)
    }

    @Test
    fun `factory getDisplayName returns correct name`() {
        val factory = MehrhofStatusWidgetFactory()
        assertEquals("Mehrhof Status", factory.displayName)
    }

    @Test
    fun `factory isAvailable returns true`() {
        val factory = MehrhofStatusWidgetFactory()
        assertTrue(factory.isAvailable(project))
    }

    @Test
    fun `factory canBeEnabledOn returns true`() {
        val factory = MehrhofStatusWidgetFactory()
        val statusBar = mockk<StatusBar>(relaxed = true)
        assertTrue(factory.canBeEnabledOn(statusBar))
    }

    @Test
    fun `factory createWidget returns MehrhofStatusWidget`() {
        val factory = MehrhofStatusWidgetFactory()
        val createdWidget = factory.createWidget(project)
        assertTrue(createdWidget is MehrhofStatusWidget)
    }

    // ========================================================================
    // Widget ID Tests
    // ========================================================================

    @Test
    fun `ID returns correct widget ID`() {
        assertEquals("MehrhofStatusWidget", widget.ID())
    }

    // ========================================================================
    // getText Tests
    // ========================================================================

    @Test
    fun `getText shows disconnected when not connected`() {
        every { service.isConnected() } returns false

        assertEquals("Mehrhof: Disconnected", widget.getText())
    }

    @Test
    fun `getText shows state when connected with no task`() {
        every { service.isConnected() } returns true
        every { service.workflowState } returns "idle"
        every { service.currentTask } returns null
        every { service.currentTaskWork } returns null

        assertEquals("Mehrhof: Idle", widget.getText())
    }

    @Test
    fun `getText shows state and task ref when connected with task`() {
        every { service.isConnected() } returns true
        every { service.workflowState } returns "implementing"
        every { service.currentTask } returns TaskInfo(id = "t1", state = "implementing", ref = "fix-bug-123")
        every { service.currentTaskWork } returns null

        val text = widget.getText()
        assertTrue(text.contains("Implementing"))
        assertTrue(text.contains("fix-bug-123"))
    }

    @Test
    fun `getText shows task title from work when available`() {
        every { service.isConnected() } returns true
        every { service.workflowState } returns "planning"
        every { service.currentTask } returns TaskInfo(id = "t1", state = "planning", ref = "ref")
        every { service.currentTaskWork } returns TaskWork(title = "Add user auth")

        val text = widget.getText()
        assertTrue(text.contains("Planning"))
        assertTrue(text.contains("Add user auth"))
    }

    @Test
    fun `getText truncates long task titles`() {
        every { service.isConnected() } returns true
        every { service.workflowState } returns "implementing"
        every { service.currentTask } returns null
        every { service.currentTaskWork } returns TaskWork(title = "This is a very long task title that should be truncated")

        val text = widget.getText()
        // WorkflowUtils.truncate limits to 20 chars + ellipsis
        assertTrue(text.length < 60)
    }

    // ========================================================================
    // getTooltipText Tests
    // ========================================================================

    @Test
    fun `getTooltipText shows connect message when disconnected`() {
        every { service.isConnected() } returns false

        assertEquals("Click to connect to Mehrhof server", widget.getTooltipText())
    }

    @Test
    fun `getTooltipText shows state when connected`() {
        every { service.isConnected() } returns true
        every { service.workflowState } returns "reviewing"

        val tooltip = widget.getTooltipText()
        assertTrue(tooltip!!.contains("State: reviewing"))
    }

    @Test
    fun `getTooltipText shows task info when available`() {
        every { service.isConnected() } returns true
        every { service.workflowState } returns "implementing"
        every { service.currentTask } returns TaskInfo(id = "task-42", state = "implementing", ref = "feature/auth")

        val tooltip = widget.getTooltipText()
        assertTrue(tooltip!!.contains("Task: task-42"))
        assertTrue(tooltip.contains("Ref: feature/auth"))
    }

    @Test
    fun `getTooltipText shows work title when available`() {
        every { service.isConnected() } returns true
        every { service.workflowState } returns "planning"
        every { service.currentTask } returns null
        every { service.currentTaskWork } returns TaskWork(title = "Implement login form")

        val tooltip = widget.getTooltipText()
        assertTrue(tooltip!!.contains("Title: Implement login form"))
    }

    @Test
    fun `getTooltipText shows pending question when available`() {
        every { service.isConnected() } returns true
        every { service.workflowState } returns "waiting"
        every { service.pendingQuestion } returns "Should I proceed with the changes?"

        val tooltip = widget.getTooltipText()
        assertTrue(tooltip!!.contains("Pending question: Should I proceed with the changes?"))
    }

    // ========================================================================
    // Click Behavior Tests
    // ========================================================================

    @Test
    fun `clicking when disconnected calls connect`() {
        every { service.isConnected() } returns false

        val clickConsumer = widget.getClickConsumer()
        clickConsumer!!.consume(mockk<MouseEvent>(relaxed = true))

        verify { service.connect() }
    }

    @Test
    fun `clicking when connected calls refreshState`() {
        every { service.isConnected() } returns true

        val clickConsumer = widget.getClickConsumer()
        clickConsumer!!.consume(mockk<MouseEvent>(relaxed = true))

        verify { service.refreshState() }
    }

    // ========================================================================
    // Listener Tests
    // ========================================================================

    @Test
    fun `widget registers as state listener on init`() {
        // Widget is created in setUp, which should register as listener
        verify { service.addStateListener(widget) }
    }

    @Test
    fun `dispose removes state listener`() {
        widget.dispose()

        verify { service.removeStateListener(widget) }
    }

    // ========================================================================
    // Install Tests
    // ========================================================================

    @Test
    fun `install stores status bar reference`() {
        val statusBar = mockk<StatusBar>(relaxed = true)

        widget.install(statusBar)
        // Trigger an update to verify statusBar is used
        widget.onConnectionChanged(true)

        // The widget should call statusBar.updateWidget in SwingUtilities.invokeLater
        // Since we can't easily test SwingUtilities, we just verify no exception
    }

    // ========================================================================
    // Presentation Tests
    // ========================================================================

    @Test
    fun `getPresentation returns this widget`() {
        assertEquals(widget, widget.getPresentation())
    }

    @Test
    fun `getAlignment returns 0`() {
        assertEquals(0f, widget.getAlignment())
    }
}
