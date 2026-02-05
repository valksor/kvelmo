package com.valksor.mehrhof.util

/**
 * Represents parsed user input from the interactive terminal.
 */
sealed class ParsedInput {
    /**
     * A workflow command with optional arguments.
     */
    data class Command(
        val name: String,
        val args: List<String>
    ) : ParsedInput()

    /**
     * A chat message to send to the agent.
     */
    data class Chat(
        val message: String
    ) : ParsedInput()

    /**
     * An answer to an agent question.
     */
    data class Answer(
        val response: String
    ) : ParsedInput()

    /**
     * Request to show help.
     */
    data object Help : ParsedInput()

    /**
     * Request to clear the message pane.
     */
    data object Clear : ParsedInput()

    /**
     * Parse error with message.
     */
    data class Error(
        val message: String
    ) : ParsedInput()
}

/**
 * Parses user input into structured commands.
 * Extracted from InteractivePanel for testability.
 */
object CommandParser {
    /**
     * Default workflow commands (and their aliases).
     * Used as fallback when discovery API is unavailable.
     */
    private val defaultCommands: Set<String> =
        setOf(
            "start",
            "plan",
            "implement",
            "impl",
            "review",
            "continue",
            "cont",
            "finish",
            "abandon",
            "undo",
            "redo",
            "status",
            "st",
            "cost",
            "budget",
            "list",
            "quick",
            "note",
            "find",
            "memory",
            "library",
            "simplify",
            "spec",
            "specification",
            "label",
            "reset",
            "auto",
            "answer",
            "question"
        )

    /**
     * Current set of recognized workflow commands.
     * Initially populated with defaults, can be updated via [updateCommands].
     */
    @Volatile
    var workflowCommands: Set<String> = defaultCommands
        private set

    /**
     * Updates the known commands from the discovery API response.
     * Includes both command names and aliases.
     */
    fun updateCommands(commands: List<com.valksor.mehrhof.api.models.CommandInfo>) {
        val newCommands = mutableSetOf<String>()
        for (cmd in commands) {
            newCommands.add(cmd.name.lowercase())
            cmd.aliases?.forEach { alias ->
                newCommands.add(alias.lowercase())
            }
        }
        // Only update if we got some commands
        if (newCommands.isNotEmpty()) {
            workflowCommands = newCommands
        }
    }

    /**
     * Resets commands to defaults (for testing or error recovery).
     */
    fun resetToDefaults() {
        workflowCommands = defaultCommands
    }

    /**
     * Parses user input into a structured ParsedInput.
     *
     * Rules:
     * - Empty/whitespace input -> Error
     * - "help" or "?" -> Help
     * - "clear" -> Clear
     * - "answer <response>" or "a <response>" -> Answer (error if no response)
     * - "chat <message>", "ask <message>", or "c <message>" -> Chat (error if no message)
     * - Known workflow command -> Command with args
     * - Unrecognized input -> treated as Chat
     */
    fun parse(input: String): ParsedInput {
        val trimmed = input.trim()
        if (trimmed.isEmpty()) {
            return ParsedInput.Error("Empty input")
        }

        val parts = trimmed.split(" ", limit = 2)
        val command = parts[0].lowercase()
        val args = if (parts.size > 1) parts[1] else ""

        return when (command) {
            // Help commands
            "help", "?" -> ParsedInput.Help

            // Clear command
            "clear" -> ParsedInput.Clear

            // Answer commands
            "answer", "a" -> {
                if (args.isNotEmpty()) {
                    ParsedInput.Answer(args)
                } else {
                    ParsedInput.Error("Usage: answer <response>")
                }
            }

            // Chat commands
            "chat", "ask", "c" -> {
                if (args.isNotEmpty()) {
                    ParsedInput.Chat(args)
                } else {
                    ParsedInput.Error("Usage: chat <message>")
                }
            }

            // Workflow commands
            in workflowCommands -> {
                val argsList = if (args.isNotEmpty()) args.split(" ") else emptyList()
                ParsedInput.Command(command, argsList)
            }

            // Default: treat as chat
            else -> ParsedInput.Chat(trimmed)
        }
    }
}
