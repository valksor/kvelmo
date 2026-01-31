package com.valksor.mehrhof.util

/**
 * Represents the enabled/disabled state of workflow buttons.
 */
data class ButtonStates(
    val plan: Boolean,
    val implement: Boolean,
    val review: Boolean,
    val finish: Boolean,
    val undo: Boolean,
    val redo: Boolean
) {
    companion object {
        val ALL_DISABLED =
            ButtonStates(
                plan = false,
                implement = false,
                review = false,
                finish = false,
                undo = false,
                redo = false
            )
    }
}

/**
 * Determines which workflow buttons should be enabled based on current workflow state.
 * Extracted from TaskListPanel for testability.
 */
object WorkflowButtonState {
    /**
     * Returns which buttons should be enabled for a given workflow state.
     *
     * State transitions:
     * - idle: Can start planning
     * - planning/implementing/reviewing/checkpointing/reverting/restoring: In progress, wait
     * - waiting: Waiting for user input, can undo
     * - planned: Planning complete, can implement or re-plan
     * - implemented: Implementation complete, can review or re-implement
     * - reviewed/done: Review complete, can finish or re-review
     * - failed: Can retry any step
     */
    fun getButtonStates(state: String): ButtonStates =
        when (state) {
            "idle" ->
                ButtonStates(
                    plan = true,
                    implement = false,
                    review = false,
                    finish = false,
                    undo = false,
                    redo = false
                )

            "planning", "implementing", "reviewing", "checkpointing", "reverting", "restoring" -> {
                // Workflow in progress - disable all buttons
                ButtonStates.ALL_DISABLED
            }

            "waiting" ->
                ButtonStates(
                    plan = false,
                    implement = false,
                    review = false,
                    finish = false,
                    undo = true, // Can undo while waiting
                    redo = false
                )

            "planned" ->
                ButtonStates(
                    plan = true, // Can re-plan
                    implement = true, // Ready to implement
                    review = false,
                    finish = false,
                    undo = true,
                    redo = false
                )

            "implemented" ->
                ButtonStates(
                    plan = false,
                    implement = true, // Can re-implement
                    review = true, // Ready to review
                    finish = false,
                    undo = true,
                    redo = false
                )

            "reviewed", "done" ->
                ButtonStates(
                    plan = false,
                    implement = false,
                    review = true, // Can re-review
                    finish = true, // Ready to finish
                    undo = true,
                    redo = false
                )

            "failed" ->
                ButtonStates(
                    plan = true, // Can retry planning
                    implement = true, // Can retry implementation
                    review = true, // Can retry review
                    finish = true, // Can finish anyway
                    undo = true,
                    redo = false
                )

            else -> {
                // Unknown state - enable basic actions
                ButtonStates(
                    plan = true,
                    implement = false,
                    review = false,
                    finish = false,
                    undo = true,
                    redo = true
                )
            }
        }
}
