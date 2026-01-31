package com.valksor.mehrhof.util

import com.intellij.ui.JBColor
import java.awt.Color

/**
 * Utility functions for workflow state display.
 * Extracted from UI components for testability.
 */
object WorkflowUtils {
    /**
     * Formats a workflow state for display.
     * Converts snake_case to "Title Case" (e.g., "waiting_for_input" -> "Waiting for input").
     */
    fun formatState(state: String): String {
        if (state.isEmpty()) return ""
        return state.replace("_", " ").replaceFirstChar { it.uppercase() }
    }

    /**
     * Truncates text with ellipsis if it exceeds maxLength.
     * Returns original text if it fits within maxLength.
     *
     * @param text The text to truncate
     * @param maxLength Maximum length including ellipsis (minimum 4 to fit "...")
     * @return Truncated text with "..." suffix, or original if short enough
     */
    fun truncate(
        text: String,
        maxLength: Int
    ): String {
        if (maxLength < 4) return text.take(maxLength)
        return if (text.length <= maxLength) {
            text
        } else {
            text.take(maxLength - 3) + "..."
        }
    }

    /**
     * Returns the display color for a workflow state.
     * Used for simple foreground coloring.
     */
    fun getStateColor(state: String): Color =
        when (state) {
            "planning" -> JBColor.BLUE
            "implementing" -> JBColor.ORANGE
            "reviewing" -> JBColor.MAGENTA
            "done" -> JBColor.GREEN.darker()
            "failed" -> JBColor.RED
            "waiting" -> JBColor.YELLOW.darker()
            else -> JBColor.GRAY
        }

    /**
     * Returns the background color for a workflow state badge.
     * Provides light/dark theme variants via JBColor.
     */
    fun getStateBackground(state: String): JBColor =
        when (state) {
            "idle" -> JBColor(Color(224, 224, 224), Color(66, 66, 66))
            "planning" -> JBColor(Color(187, 222, 251), Color(21, 101, 192))
            "implementing" -> JBColor(Color(255, 224, 178), Color(230, 81, 0))
            "reviewing" -> JBColor(Color(225, 190, 231), Color(123, 31, 162))
            "waiting" -> JBColor(Color(255, 245, 157), Color(245, 127, 23))
            "done" -> JBColor(Color(200, 230, 201), Color(27, 94, 32))
            "failed" -> JBColor(Color(255, 205, 210), Color(183, 28, 28))
            else -> JBColor(Color(224, 224, 224), Color(66, 66, 66))
        }

    /**
     * Returns the foreground color for a workflow state badge.
     * Provides light/dark theme variants via JBColor.
     */
    fun getStateForeground(state: String): JBColor =
        when (state) {
            "idle" -> JBColor(Color(97, 97, 97), Color(189, 189, 189))
            "planning" -> JBColor(Color(13, 71, 161), Color(187, 222, 251))
            "implementing" -> JBColor(Color(230, 81, 0), Color(255, 224, 178))
            "reviewing" -> JBColor(Color(74, 20, 140), Color(225, 190, 231))
            "waiting" -> JBColor(Color(245, 127, 23), Color(255, 245, 157))
            "done" -> JBColor(Color(27, 94, 32), Color(200, 230, 201))
            "failed" -> JBColor(Color(183, 28, 28), Color(255, 205, 210))
            else -> JBColor(Color(97, 97, 97), Color(189, 189, 189))
        }
}
