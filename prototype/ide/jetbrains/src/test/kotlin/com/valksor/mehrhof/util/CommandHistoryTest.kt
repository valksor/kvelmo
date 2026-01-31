package com.valksor.mehrhof.util

import org.junit.jupiter.api.Assertions.*
import org.junit.jupiter.api.BeforeEach
import org.junit.jupiter.api.Test

class CommandHistoryTest {
    private lateinit var history: CommandHistory

    @BeforeEach
    fun setUp() {
        history = CommandHistory()
    }

    // ========================================================================
    // Basic add and size tests
    // ========================================================================

    @Test
    fun `new history is empty`() {
        assertTrue(history.isEmpty)
        assertEquals(0, history.size)
    }

    @Test
    fun `add increases size`() {
        history.add("command1")
        assertEquals(1, history.size)
        history.add("command2")
        assertEquals(2, history.size)
    }

    @Test
    fun `add blank command does nothing`() {
        history.add("")
        assertTrue(history.isEmpty)
        history.add("   ")
        assertTrue(history.isEmpty)
    }

    @Test
    fun `add duplicate of last command does not add`() {
        history.add("command1")
        history.add("command1")
        assertEquals(1, history.size)
    }

    @Test
    fun `add same command with different commands in between adds both`() {
        history.add("command1")
        history.add("command2")
        history.add("command1")
        assertEquals(3, history.size)
    }

    // ========================================================================
    // Navigation tests
    // ========================================================================

    @Test
    fun `navigateUp on empty history returns null`() {
        assertNull(history.navigateUp())
    }

    @Test
    fun `navigateDown on empty history returns null`() {
        assertNull(history.navigateDown())
    }

    @Test
    fun `navigateUp returns previous command`() {
        history.add("command1")
        history.add("command2")
        history.add("command3")

        assertEquals("command3", history.navigateUp())
        assertEquals("command2", history.navigateUp())
        assertEquals("command1", history.navigateUp())
    }

    @Test
    fun `navigateUp stops at beginning`() {
        history.add("command1")
        history.add("command2")

        history.navigateUp() // -> command2
        history.navigateUp() // -> command1
        history.navigateUp() // still command1
        assertEquals("command1", history.getAt(history.currentIndex()))
    }

    @Test
    fun `navigateDown returns next command`() {
        history.add("command1")
        history.add("command2")

        history.navigateUp() // -> command2
        history.navigateUp() // -> command1
        assertEquals("command2", history.navigateDown())
    }

    @Test
    fun `navigateDown at end returns null`() {
        history.add("command1")
        // Index starts at end (past last command)
        assertNull(history.navigateDown())
    }

    @Test
    fun `navigateDown after navigateUp returns next command`() {
        history.add("command1")
        history.add("command2")
        history.add("command3")

        history.navigateUp() // -> command3
        history.navigateUp() // -> command2
        assertEquals("command3", history.navigateDown())
    }

    @Test
    fun `after navigating down past end returns null`() {
        history.add("command1")
        history.add("command2")

        history.navigateUp() // -> command2
        history.navigateUp() // -> command1
        history.navigateDown() // -> command2
        history.navigateDown() // -> end (null)
        assertNull(history.navigateDown())
    }

    // ========================================================================
    // Reset navigation tests
    // ========================================================================

    @Test
    fun `resetNavigation moves to end`() {
        history.add("command1")
        history.add("command2")

        history.navigateUp() // -> command2
        history.navigateUp() // -> command1

        history.resetNavigation()
        assertEquals("command2", history.navigateUp())
    }

    @Test
    fun `adding command resets navigation`() {
        history.add("command1")
        history.add("command2")

        history.navigateUp() // -> command2
        history.navigateUp() // -> command1

        history.add("command3")
        assertEquals("command3", history.navigateUp())
    }

    // ========================================================================
    // Max size tests
    // ========================================================================

    @Test
    fun `history trims to max size`() {
        val smallHistory = CommandHistory(maxSize = 3)
        smallHistory.add("cmd1")
        smallHistory.add("cmd2")
        smallHistory.add("cmd3")
        smallHistory.add("cmd4")

        assertEquals(3, smallHistory.size)
        assertEquals("cmd2", smallHistory.getAt(0)) // cmd1 was removed
        assertEquals("cmd4", smallHistory.getAt(2))
    }

    @Test
    fun `max size of 1 keeps only last command`() {
        val tinyHistory = CommandHistory(maxSize = 1)
        tinyHistory.add("cmd1")
        tinyHistory.add("cmd2")
        tinyHistory.add("cmd3")

        assertEquals(1, tinyHistory.size)
        assertEquals("cmd3", tinyHistory.getAt(0))
    }

    // ========================================================================
    // Clear tests
    // ========================================================================

    @Test
    fun `clear removes all history`() {
        history.add("command1")
        history.add("command2")

        history.clear()

        assertTrue(history.isEmpty)
        assertEquals(0, history.size)
    }

    @Test
    fun `clear resets index`() {
        history.add("command1")
        history.navigateUp()

        history.clear()

        assertEquals(0, history.currentIndex())
    }

    // ========================================================================
    // getAt tests
    // ========================================================================

    @Test
    fun `getAt returns command at index`() {
        history.add("command1")
        history.add("command2")

        assertEquals("command1", history.getAt(0))
        assertEquals("command2", history.getAt(1))
    }

    @Test
    fun `getAt returns null for out of bounds`() {
        history.add("command1")

        assertNull(history.getAt(-1))
        assertNull(history.getAt(5))
    }

    // ========================================================================
    // Edge case tests
    // ========================================================================

    @Test
    fun `single command navigation works`() {
        history.add("only")

        assertEquals("only", history.navigateUp())
        assertEquals("only", history.navigateUp()) // stays at same
        assertNull(history.navigateDown()) // at end
    }

    @Test
    fun `multiple navigateUp then navigateDown sequence`() {
        history.add("a")
        history.add("b")
        history.add("c")

        assertEquals("c", history.navigateUp())
        assertEquals("b", history.navigateUp())
        assertEquals("c", history.navigateDown())
        assertEquals("b", history.navigateUp())
        assertEquals("a", history.navigateUp())
        assertEquals("a", history.navigateUp()) // stuck at beginning
        assertEquals("b", history.navigateDown())
    }
}
