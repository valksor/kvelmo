package com.valksor.mehrhof.util

import com.intellij.ui.JBColor
import org.junit.jupiter.api.Assertions.*
import org.junit.jupiter.api.Test

class WorkflowUtilsTest {
    // ========================================================================
    // formatState tests
    // ========================================================================

    @Test
    fun `formatState converts idle to Idle`() {
        assertEquals("Idle", WorkflowUtils.formatState("idle"))
    }

    @Test
    fun `formatState converts planning to Planning`() {
        assertEquals("Planning", WorkflowUtils.formatState("planning"))
    }

    @Test
    fun `formatState converts underscores to spaces`() {
        assertEquals("Waiting for input", WorkflowUtils.formatState("waiting_for_input"))
    }

    @Test
    fun `formatState capitalizes first letter only`() {
        assertEquals("Some long state", WorkflowUtils.formatState("some_long_state"))
    }

    @Test
    fun `formatState handles empty string`() {
        assertEquals("", WorkflowUtils.formatState(""))
    }

    @Test
    fun `formatState handles single character`() {
        assertEquals("A", WorkflowUtils.formatState("a"))
    }

    @Test
    fun `formatState handles already capitalized`() {
        assertEquals("IDLE", WorkflowUtils.formatState("IDLE"))
    }

    @Test
    fun `formatState handles multiple underscores`() {
        assertEquals("One two three", WorkflowUtils.formatState("one_two_three"))
    }

    // ========================================================================
    // truncate tests
    // ========================================================================

    @Test
    fun `truncate returns original when shorter than max`() {
        assertEquals("Hello", WorkflowUtils.truncate("Hello", 10))
    }

    @Test
    fun `truncate returns original when exactly max length`() {
        assertEquals("Hello", WorkflowUtils.truncate("Hello", 5))
    }

    @Test
    fun `truncate adds ellipsis when longer than max`() {
        assertEquals("Hello...", WorkflowUtils.truncate("Hello world", 8))
    }

    @Test
    fun `truncate handles maxLength of 4 minimum for ellipsis`() {
        assertEquals("H...", WorkflowUtils.truncate("Hello", 4))
    }

    @Test
    fun `truncate handles maxLength less than 4`() {
        assertEquals("Hel", WorkflowUtils.truncate("Hello", 3))
    }

    @Test
    fun `truncate handles maxLength of 0`() {
        assertEquals("", WorkflowUtils.truncate("Hello", 0))
    }

    @Test
    fun `truncate handles empty string`() {
        assertEquals("", WorkflowUtils.truncate("", 10))
    }

    @Test
    fun `truncate handles very long text`() {
        val longText = "a".repeat(1000)
        val result = WorkflowUtils.truncate(longText, 20)
        assertEquals(20, result.length)
        assertTrue(result.endsWith("..."))
    }

    // ========================================================================
    // getStateColor tests
    // ========================================================================

    @Test
    fun `getStateColor returns blue for planning`() {
        assertEquals(JBColor.BLUE, WorkflowUtils.getStateColor("planning"))
    }

    @Test
    fun `getStateColor returns orange for implementing`() {
        assertEquals(JBColor.ORANGE, WorkflowUtils.getStateColor("implementing"))
    }

    @Test
    fun `getStateColor returns magenta for reviewing`() {
        assertEquals(JBColor.MAGENTA, WorkflowUtils.getStateColor("reviewing"))
    }

    @Test
    fun `getStateColor returns dark green for done`() {
        assertEquals(JBColor.GREEN.darker(), WorkflowUtils.getStateColor("done"))
    }

    @Test
    fun `getStateColor returns red for failed`() {
        assertEquals(JBColor.RED, WorkflowUtils.getStateColor("failed"))
    }

    @Test
    fun `getStateColor returns dark yellow for waiting`() {
        assertEquals(JBColor.YELLOW.darker(), WorkflowUtils.getStateColor("waiting"))
    }

    @Test
    fun `getStateColor returns gray for unknown state`() {
        assertEquals(JBColor.GRAY, WorkflowUtils.getStateColor("unknown"))
    }

    @Test
    fun `getStateColor returns gray for idle`() {
        assertEquals(JBColor.GRAY, WorkflowUtils.getStateColor("idle"))
    }

    @Test
    fun `getStateColor returns gray for empty string`() {
        assertEquals(JBColor.GRAY, WorkflowUtils.getStateColor(""))
    }

    // ========================================================================
    // getStateBackground tests
    // ========================================================================

    @Test
    fun `getStateBackground returns JBColor for idle`() {
        val color = WorkflowUtils.getStateBackground("idle")
        assertNotNull(color)
        // Verify it's a JBColor (has both light and dark variants)
        assertTrue(color is JBColor)
    }

    @Test
    fun `getStateBackground returns different colors for different states`() {
        val idleColor = WorkflowUtils.getStateBackground("idle")
        val planningColor = WorkflowUtils.getStateBackground("planning")
        assertNotEquals(idleColor, planningColor)
    }

    @Test
    fun `getStateBackground returns blue-ish for planning`() {
        val color = WorkflowUtils.getStateBackground("planning")
        // Light theme color should be blue-ish (high blue component)
        assertTrue(color.blue > color.red)
    }

    @Test
    fun `getStateBackground returns same color for unknown as idle`() {
        val unknownColor = WorkflowUtils.getStateBackground("unknown_state")
        val idleColor = WorkflowUtils.getStateBackground("idle")
        assertEquals(idleColor, unknownColor)
    }

    // ========================================================================
    // getStateForeground tests
    // ========================================================================

    @Test
    fun `getStateForeground returns JBColor for all states`() {
        val states = listOf("idle", "planning", "implementing", "reviewing", "waiting", "done", "failed")
        for (state in states) {
            val color = WorkflowUtils.getStateForeground(state)
            assertNotNull(color)
            assertTrue(color is JBColor)
        }
    }

    @Test
    fun `getStateForeground returns gray-ish for idle`() {
        val color = WorkflowUtils.getStateForeground("idle")
        // Gray means R, G, B are similar
        val diff = maxOf(color.red, color.green, color.blue) - minOf(color.red, color.green, color.blue)
        assertTrue(diff < 50, "Idle foreground should be grayish")
    }
}
