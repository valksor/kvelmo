package com.valksor.mehrhof.toolwindow

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
import javax.swing.JOptionPane
import javax.swing.JPanel

/**
 * Unit tests for [ActionsPanel].
 *
 * Tests button creation and command callback behavior.
 */
class ActionsPanelTest {
    private val capturedCommands = mutableListOf<Pair<String, List<String>>>()
    private lateinit var panel: ActionsPanel

    @BeforeEach
    fun setUp() {
        capturedCommands.clear()
        panel = ActionsPanel { command, args ->
            capturedCommands.add(command to args)
        }

        // Mock JOptionPane for dialogs
        mockkStatic(JOptionPane::class)
    }

    @AfterEach
    fun tearDown() {
        unmockkAll()
    }

    // ========================================================================
    // Button Creation Tests
    // ========================================================================

    @Test
    fun `panel contains workflow section buttons`() {
        val buttons = findAllButtons()
        val buttonTexts = buttons.map { it.text }

        assertTrue(buttonTexts.contains("Start Task..."))
        assertTrue(buttonTexts.contains("Plan"))
        assertTrue(buttonTexts.contains("Implement"))
        assertTrue(buttonTexts.contains("Review"))
        assertTrue(buttonTexts.contains("Continue"))
        assertTrue(buttonTexts.contains("Finish"))
        assertTrue(buttonTexts.contains("Abandon"))
    }

    @Test
    fun `panel contains checkpoint section buttons`() {
        val buttons = findAllButtons()
        val buttonTexts = buttons.map { it.text }

        assertTrue(buttonTexts.contains("Undo"))
        assertTrue(buttonTexts.contains("Redo"))
    }

    @Test
    fun `panel contains info section buttons`() {
        val buttons = findAllButtons()
        val buttonTexts = buttons.map { it.text }

        assertTrue(buttonTexts.contains("Status"))
        assertTrue(buttonTexts.contains("Cost"))
        assertTrue(buttonTexts.contains("Budget"))
        assertTrue(buttonTexts.contains("List Tasks"))
        assertTrue(buttonTexts.contains("Specifications"))
    }

    @Test
    fun `panel contains tools section buttons`() {
        val buttons = findAllButtons()
        val buttonTexts = buttons.map { it.text }

        assertTrue(buttonTexts.contains("Find Code..."))
        assertTrue(buttonTexts.contains("Search Memory..."))
        assertTrue(buttonTexts.contains("Library"))
        assertTrue(buttonTexts.contains("Quick Task..."))
        assertTrue(buttonTexts.contains("Simplify"))
        assertTrue(buttonTexts.contains("Add Note..."))
    }

    // ========================================================================
    // Simple Command Tests (no dialog)
    // ========================================================================

    @Test
    fun `plan button triggers plan command`() {
        clickButton("Plan")

        assertEquals(1, capturedCommands.size)
        assertEquals("plan" to emptyList<String>(), capturedCommands[0])
    }

    @Test
    fun `implement button triggers implement command`() {
        clickButton("Implement")

        assertEquals(1, capturedCommands.size)
        assertEquals("implement" to emptyList<String>(), capturedCommands[0])
    }

    @Test
    fun `review button triggers review command`() {
        clickButton("Review")

        assertEquals(1, capturedCommands.size)
        assertEquals("review" to emptyList<String>(), capturedCommands[0])
    }

    @Test
    fun `continue button triggers continue command`() {
        clickButton("Continue")

        assertEquals(1, capturedCommands.size)
        assertEquals("continue" to emptyList<String>(), capturedCommands[0])
    }

    @Test
    fun `undo button triggers undo command`() {
        clickButton("Undo")

        assertEquals(1, capturedCommands.size)
        assertEquals("undo" to emptyList<String>(), capturedCommands[0])
    }

    @Test
    fun `redo button triggers redo command`() {
        clickButton("Redo")

        assertEquals(1, capturedCommands.size)
        assertEquals("redo" to emptyList<String>(), capturedCommands[0])
    }

    @Test
    fun `status button triggers status command`() {
        clickButton("Status")

        assertEquals(1, capturedCommands.size)
        assertEquals("status" to emptyList<String>(), capturedCommands[0])
    }

    @Test
    fun `cost button triggers cost command`() {
        clickButton("Cost")

        assertEquals(1, capturedCommands.size)
        assertEquals("cost" to emptyList<String>(), capturedCommands[0])
    }

    @Test
    fun `budget button triggers budget command`() {
        clickButton("Budget")

        assertEquals(1, capturedCommands.size)
        assertEquals("budget" to emptyList<String>(), capturedCommands[0])
    }

    @Test
    fun `list button triggers list command`() {
        clickButton("List Tasks")

        assertEquals(1, capturedCommands.size)
        assertEquals("list" to emptyList<String>(), capturedCommands[0])
    }

    @Test
    fun `specification button triggers specification command`() {
        clickButton("Specifications")

        assertEquals(1, capturedCommands.size)
        assertEquals("specification" to emptyList<String>(), capturedCommands[0])
    }

    @Test
    fun `library button triggers library command`() {
        clickButton("Library")

        assertEquals(1, capturedCommands.size)
        assertEquals("library" to emptyList<String>(), capturedCommands[0])
    }

    @Test
    fun `simplify button triggers simplify command`() {
        clickButton("Simplify")

        assertEquals(1, capturedCommands.size)
        assertEquals("simplify" to emptyList<String>(), capturedCommands[0])
    }

    // ========================================================================
    // Prompt Dialog Tests
    // ========================================================================

    @Test
    fun `start button shows input dialog and passes ref`() {
        every { JOptionPane.showInputDialog(any(), any(), eq("Start Task"), any<Int>()) } returns "github:123"

        clickButton("Start Task...")

        assertEquals(1, capturedCommands.size)
        assertEquals("start", capturedCommands[0].first)
        assertEquals(listOf("github:123"), capturedCommands[0].second)
    }

    @Test
    fun `start button does not trigger command when cancelled`() {
        every { JOptionPane.showInputDialog(any(), any(), eq("Start Task"), any<Int>()) } returns null

        clickButton("Start Task...")

        assertTrue(capturedCommands.isEmpty())
    }

    @Test
    fun `start button does not trigger command for blank input`() {
        every { JOptionPane.showInputDialog(any(), any(), eq("Start Task"), any<Int>()) } returns "   "

        clickButton("Start Task...")

        assertTrue(capturedCommands.isEmpty())
    }

    @Test
    fun `find button shows input dialog and passes query`() {
        every { JOptionPane.showInputDialog(any(), any(), eq("Find Code"), any<Int>()) } returns "authentication"

        clickButton("Find Code...")

        assertEquals(1, capturedCommands.size)
        assertEquals("find", capturedCommands[0].first)
        assertEquals(listOf("authentication"), capturedCommands[0].second)
    }

    @Test
    fun `memory button shows input dialog and passes query`() {
        every { JOptionPane.showInputDialog(any(), any(), eq("Search Memory"), any<Int>()) } returns "login flow"

        clickButton("Search Memory...")

        assertEquals(1, capturedCommands.size)
        assertEquals("memory", capturedCommands[0].first)
        assertEquals(listOf("login flow"), capturedCommands[0].second)
    }

    @Test
    fun `quick task button shows input dialog and passes description`() {
        every { JOptionPane.showInputDialog(any(), any(), eq("Create Quick Task"), any<Int>()) } returns "Fix bug in login"

        clickButton("Quick Task...")

        assertEquals(1, capturedCommands.size)
        assertEquals("quick", capturedCommands[0].first)
        assertEquals(listOf("Fix bug in login"), capturedCommands[0].second)
    }

    @Test
    fun `note button shows input dialog and passes note text`() {
        every { JOptionPane.showInputDialog(any(), any(), eq("Add Note"), any<Int>()) } returns "Remember to test edge cases"

        clickButton("Add Note...")

        assertEquals(1, capturedCommands.size)
        assertEquals("note", capturedCommands[0].first)
        assertEquals(listOf("Remember to test edge cases"), capturedCommands[0].second)
    }

    // ========================================================================
    // Confirmation Dialog Tests
    // ========================================================================

    @Test
    fun `finish button shows confirmation and triggers on YES`() {
        every {
            JOptionPane.showConfirmDialog(any(), any(), eq("Confirm"), eq(JOptionPane.YES_NO_OPTION))
        } returns JOptionPane.YES_OPTION

        clickButton("Finish")

        assertEquals(1, capturedCommands.size)
        assertEquals("finish" to emptyList<String>(), capturedCommands[0])
    }

    @Test
    fun `finish button does not trigger on NO`() {
        every {
            JOptionPane.showConfirmDialog(any(), any(), eq("Confirm"), eq(JOptionPane.YES_NO_OPTION))
        } returns JOptionPane.NO_OPTION

        clickButton("Finish")

        assertTrue(capturedCommands.isEmpty())
    }

    @Test
    fun `abandon button shows confirmation and triggers on YES`() {
        every {
            JOptionPane.showConfirmDialog(any(), any(), eq("Confirm"), eq(JOptionPane.YES_NO_OPTION))
        } returns JOptionPane.YES_OPTION

        clickButton("Abandon")

        assertEquals(1, capturedCommands.size)
        assertEquals("abandon" to emptyList<String>(), capturedCommands[0])
    }

    @Test
    fun `abandon button does not trigger on NO`() {
        every {
            JOptionPane.showConfirmDialog(any(), any(), eq("Confirm"), eq(JOptionPane.YES_NO_OPTION))
        } returns JOptionPane.NO_OPTION

        clickButton("Abandon")

        assertTrue(capturedCommands.isEmpty())
    }

    // ========================================================================
    // Helper Methods
    // ========================================================================

    private fun findAllButtons(): List<JButton> {
        val buttons = mutableListOf<JButton>()
        findButtonsRecursive(panel, buttons)
        return buttons
    }

    private fun findButtonsRecursive(container: java.awt.Container, buttons: MutableList<JButton>) {
        for (component in container.components) {
            if (component is JButton) {
                buttons.add(component)
            } else if (component is java.awt.Container) {
                findButtonsRecursive(component, buttons)
            }
        }
    }

    private fun clickButton(text: String) {
        val button = findAllButtons().find { it.text == text }
            ?: throw AssertionError("Button '$text' not found")
        button.doClick()
    }
}
