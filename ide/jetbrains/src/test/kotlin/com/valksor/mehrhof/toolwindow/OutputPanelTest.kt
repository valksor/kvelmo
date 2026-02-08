package com.valksor.mehrhof.toolwindow

import com.intellij.openapi.project.Project
import com.valksor.mehrhof.services.MehrhofProjectService
import io.mockk.every
import io.mockk.mockk
import io.mockk.mockkStatic
import io.mockk.unmockkAll
import io.mockk.verify
import org.junit.jupiter.api.AfterEach
import org.junit.jupiter.api.Assertions.assertFalse
import org.junit.jupiter.api.Assertions.assertTrue
import org.junit.jupiter.api.BeforeEach
import org.junit.jupiter.api.Test
import javax.swing.SwingUtilities

/**
 * Unit tests for [OutputPanel].
 *
 * Tests StateListener callbacks and message display formatting.
 */
class OutputPanelTest {
    private lateinit var project: Project
    private lateinit var service: MehrhofProjectService
    private lateinit var panel: OutputPanel

    @BeforeEach
    fun setUp() {
        project = mockk(relaxed = true)
        service = mockk(relaxed = true)

        mockkStatic(MehrhofProjectService::class)
        every { MehrhofProjectService.getInstance(project) } returns service

        panel = OutputPanel(project, service)
    }

    @AfterEach
    fun tearDown() {
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

    // ========================================================================
    // StateListener Callback Tests
    // ========================================================================

    @Test
    fun `onAgentMessage appends content to output`() {
        // Execute the callback
        panel.onAgentMessage("Hello from agent", "info")

        // Wait for SwingUtilities.invokeLater to complete
        SwingUtilities.invokeAndWait { }

        // Verify the panel contains the message by checking internal state
        val outputField = panel.javaClass.getDeclaredField("outputArea")
        outputField.isAccessible = true
        val outputArea = outputField.get(panel) as javax.swing.JTextArea
        assertTrue(outputArea.text.contains("Hello from agent"))
    }

    @Test
    fun `onWorkflowStateChanged appends state transition`() {
        panel.onWorkflowStateChanged("planning", "idle")

        SwingUtilities.invokeAndWait { }

        val outputField = panel.javaClass.getDeclaredField("outputArea")
        outputField.isAccessible = true
        val outputArea = outputField.get(panel) as javax.swing.JTextArea

        assertTrue(outputArea.text.contains("idle"))
        assertTrue(outputArea.text.contains("planning"))
        assertTrue(outputArea.text.contains("State changed"))
    }

    @Test
    fun `onError appends error message with ERROR prefix`() {
        panel.onError("Something went wrong")

        SwingUtilities.invokeAndWait { }

        val outputField = panel.javaClass.getDeclaredField("outputArea")
        outputField.isAccessible = true
        val outputArea = outputField.get(panel) as javax.swing.JTextArea

        assertTrue(outputArea.text.contains("[ERROR]"))
        assertTrue(outputArea.text.contains("Something went wrong"))
    }

    @Test
    fun `onQuestionReceived appends question with QUESTION prefix`() {
        panel.onQuestionReceived("What should I do?", null)

        SwingUtilities.invokeAndWait { }

        val outputField = panel.javaClass.getDeclaredField("outputArea")
        outputField.isAccessible = true
        val outputArea = outputField.get(panel) as javax.swing.JTextArea

        assertTrue(outputArea.text.contains("[QUESTION]"))
        assertTrue(outputArea.text.contains("What should I do?"))
    }

    @Test
    fun `onQuestionReceived shows options when provided`() {
        panel.onQuestionReceived("Choose one:", listOf("Option A", "Option B", "Option C"))

        SwingUtilities.invokeAndWait { }

        val outputField = panel.javaClass.getDeclaredField("outputArea")
        outputField.isAccessible = true
        val outputArea = outputField.get(panel) as javax.swing.JTextArea

        assertTrue(outputArea.text.contains("1. Option A"))
        assertTrue(outputArea.text.contains("2. Option B"))
        assertTrue(outputArea.text.contains("3. Option C"))
    }

    // ========================================================================
    // Clear Button Test
    // ========================================================================

    @Test
    fun `clear button clears output area`() {
        // Add some content first
        panel.onAgentMessage("Some message", null)
        SwingUtilities.invokeAndWait { }

        val outputField = panel.javaClass.getDeclaredField("outputArea")
        outputField.isAccessible = true
        val outputArea = outputField.get(panel) as javax.swing.JTextArea

        // Verify content exists
        assertTrue(outputArea.text.isNotEmpty())

        // Find and click clear button
        val clearButtonField = panel.javaClass.getDeclaredField("clearButton")
        clearButtonField.isAccessible = true
        val clearButton = clearButtonField.get(panel) as javax.swing.JButton
        clearButton.doClick()

        // Verify cleared
        assertTrue(outputArea.text.isEmpty())
    }

    // ========================================================================
    // Output Accumulation Tests
    // ========================================================================

    @Test
    fun `multiple messages accumulate in output`() {
        panel.onAgentMessage("First message", null)
        panel.onAgentMessage("Second message", null)
        panel.onError("An error")

        SwingUtilities.invokeAndWait { }

        val outputField = panel.javaClass.getDeclaredField("outputArea")
        outputField.isAccessible = true
        val outputArea = outputField.get(panel) as javax.swing.JTextArea

        assertTrue(outputArea.text.contains("First message"))
        assertTrue(outputArea.text.contains("Second message"))
        assertTrue(outputArea.text.contains("An error"))
    }

    // ========================================================================
    // No-Op Listener Methods
    // ========================================================================

    @Test
    fun `onConnectionChanged does not throw`() {
        // Default no-op implementation
        panel.onConnectionChanged(true)
        panel.onConnectionChanged(false)
        // No exception = pass
    }

    @Test
    fun `onTaskChanged does not throw`() {
        // Default no-op implementation
        panel.onTaskChanged(null, null)
        // No exception = pass
    }
}
