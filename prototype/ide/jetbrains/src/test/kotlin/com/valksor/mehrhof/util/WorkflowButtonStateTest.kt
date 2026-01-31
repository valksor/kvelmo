package com.valksor.mehrhof.util

import org.junit.jupiter.api.Assertions.*
import org.junit.jupiter.api.Test

class WorkflowButtonStateTest {
    // ========================================================================
    // Idle state tests
    // ========================================================================

    @Test
    fun `idle state enables only plan button`() {
        val states = WorkflowButtonState.getButtonStates("idle")
        assertTrue(states.plan, "Plan should be enabled in idle")
        assertFalse(states.implement, "Implement should be disabled in idle")
        assertFalse(states.review, "Review should be disabled in idle")
        assertFalse(states.finish, "Finish should be disabled in idle")
        assertFalse(states.undo, "Undo should be disabled in idle")
        assertFalse(states.redo, "Redo should be disabled in idle")
    }

    // ========================================================================
    // In-progress states (should disable all buttons)
    // ========================================================================

    @Test
    fun `planning state disables all buttons`() {
        val states = WorkflowButtonState.getButtonStates("planning")
        assertEquals(ButtonStates.ALL_DISABLED, states)
    }

    @Test
    fun `implementing state disables all buttons`() {
        val states = WorkflowButtonState.getButtonStates("implementing")
        assertEquals(ButtonStates.ALL_DISABLED, states)
    }

    @Test
    fun `reviewing state disables all buttons`() {
        val states = WorkflowButtonState.getButtonStates("reviewing")
        assertEquals(ButtonStates.ALL_DISABLED, states)
    }

    @Test
    fun `checkpointing state disables all buttons`() {
        val states = WorkflowButtonState.getButtonStates("checkpointing")
        assertEquals(ButtonStates.ALL_DISABLED, states)
    }

    @Test
    fun `reverting state disables all buttons`() {
        val states = WorkflowButtonState.getButtonStates("reverting")
        assertEquals(ButtonStates.ALL_DISABLED, states)
    }

    @Test
    fun `restoring state disables all buttons`() {
        val states = WorkflowButtonState.getButtonStates("restoring")
        assertEquals(ButtonStates.ALL_DISABLED, states)
    }

    // ========================================================================
    // Waiting state
    // ========================================================================

    @Test
    fun `waiting state enables only undo`() {
        val states = WorkflowButtonState.getButtonStates("waiting")
        assertFalse(states.plan, "Plan should be disabled in waiting")
        assertFalse(states.implement, "Implement should be disabled in waiting")
        assertFalse(states.review, "Review should be disabled in waiting")
        assertFalse(states.finish, "Finish should be disabled in waiting")
        assertTrue(states.undo, "Undo should be enabled in waiting")
        assertFalse(states.redo, "Redo should be disabled in waiting")
    }

    // ========================================================================
    // Completion states
    // ========================================================================

    @Test
    fun `planned state enables plan implement undo`() {
        val states = WorkflowButtonState.getButtonStates("planned")
        assertTrue(states.plan, "Can re-plan")
        assertTrue(states.implement, "Ready to implement")
        assertFalse(states.review, "Review not available yet")
        assertFalse(states.finish, "Finish not available yet")
        assertTrue(states.undo, "Can undo")
        assertFalse(states.redo, "Redo should be disabled")
    }

    @Test
    fun `implemented state enables implement review undo`() {
        val states = WorkflowButtonState.getButtonStates("implemented")
        assertFalse(states.plan, "Plan should be disabled")
        assertTrue(states.implement, "Can re-implement")
        assertTrue(states.review, "Ready to review")
        assertFalse(states.finish, "Finish not available yet")
        assertTrue(states.undo, "Can undo")
        assertFalse(states.redo, "Redo should be disabled")
    }

    @Test
    fun `reviewed state enables review finish undo`() {
        val states = WorkflowButtonState.getButtonStates("reviewed")
        assertFalse(states.plan, "Plan should be disabled")
        assertFalse(states.implement, "Implement should be disabled")
        assertTrue(states.review, "Can re-review")
        assertTrue(states.finish, "Ready to finish")
        assertTrue(states.undo, "Can undo")
        assertFalse(states.redo, "Redo should be disabled")
    }

    @Test
    fun `done state enables review finish undo`() {
        val states = WorkflowButtonState.getButtonStates("done")
        assertFalse(states.plan, "Plan should be disabled")
        assertFalse(states.implement, "Implement should be disabled")
        assertTrue(states.review, "Can re-review")
        assertTrue(states.finish, "Can finish")
        assertTrue(states.undo, "Can undo")
        assertFalse(states.redo, "Redo should be disabled")
    }

    // ========================================================================
    // Failed state
    // ========================================================================

    @Test
    fun `failed state enables all except redo`() {
        val states = WorkflowButtonState.getButtonStates("failed")
        assertTrue(states.plan, "Can retry planning")
        assertTrue(states.implement, "Can retry implementation")
        assertTrue(states.review, "Can retry review")
        assertTrue(states.finish, "Can finish anyway")
        assertTrue(states.undo, "Can undo")
        assertFalse(states.redo, "Redo should be disabled")
    }

    // ========================================================================
    // Unknown state (fallback)
    // ========================================================================

    @Test
    fun `unknown state has default enablement`() {
        val states = WorkflowButtonState.getButtonStates("some_unknown_state")
        assertTrue(states.plan, "Plan should be enabled as fallback")
        assertFalse(states.implement, "Implement should be disabled")
        assertFalse(states.review, "Review should be disabled")
        assertFalse(states.finish, "Finish should be disabled")
        assertTrue(states.undo, "Undo should be enabled")
        assertTrue(states.redo, "Redo should be enabled in unknown state")
    }

    @Test
    fun `empty state has default enablement`() {
        val states = WorkflowButtonState.getButtonStates("")
        assertTrue(states.plan)
        assertTrue(states.undo)
        assertTrue(states.redo)
    }

    // ========================================================================
    // ButtonStates data class tests
    // ========================================================================

    @Test
    fun `ALL_DISABLED has all fields false`() {
        val disabled = ButtonStates.ALL_DISABLED
        assertFalse(disabled.plan)
        assertFalse(disabled.implement)
        assertFalse(disabled.review)
        assertFalse(disabled.finish)
        assertFalse(disabled.undo)
        assertFalse(disabled.redo)
    }

    @Test
    fun `ButtonStates equality works correctly`() {
        val states1 = ButtonStates(true, false, false, false, true, false)
        val states2 = ButtonStates(true, false, false, false, true, false)
        val states3 = ButtonStates(false, false, false, false, true, false)

        assertEquals(states1, states2)
        assertNotEquals(states1, states3)
    }
}
