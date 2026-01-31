package com.valksor.mehrhof.util

/**
 * Manages command history for the interactive terminal.
 * Allows navigation through previously entered commands.
 */
class CommandHistory(
    private val maxSize: Int = 100
) {
    private val history = mutableListOf<String>()
    private var index = 0

    /**
     * The number of commands in history.
     */
    val size: Int get() = history.size

    /**
     * Whether history is empty.
     */
    val isEmpty: Boolean get() = history.isEmpty()

    /**
     * Add a command to history.
     * Resets the navigation index to the end.
     */
    fun add(command: String) {
        if (command.isBlank()) return

        // Don't add duplicate of last command
        if (history.isNotEmpty() && history.last() == command) {
            index = history.size
            return
        }

        // Trim to max size
        if (history.size >= maxSize) {
            history.removeAt(0)
        }

        history.add(command)
        index = history.size
    }

    /**
     * Navigate up (to previous command).
     * Returns the command at the new position, or null if at the beginning.
     */
    fun navigateUp(): String? {
        if (history.isEmpty()) return null
        if (index > 0) {
            index--
        }
        return history.getOrNull(index)
    }

    /**
     * Navigate down (to next command).
     * Returns the command at the new position, or null if at the end.
     */
    fun navigateDown(): String? {
        if (history.isEmpty()) return null
        if (index < history.size) {
            index++
        }
        return if (index < history.size) history[index] else null
    }

    /**
     * Reset navigation to the end of history.
     */
    fun resetNavigation() {
        index = history.size
    }

    /**
     * Get current position in history (for testing).
     */
    fun currentIndex(): Int = index

    /**
     * Get command at specific index (for testing).
     */
    fun getAt(i: Int): String? = history.getOrNull(i)

    /**
     * Clear all history.
     */
    fun clear() {
        history.clear()
        index = 0
    }
}
