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
     * Recognized workflow commands (and their aliases).
     */
    val workflowCommands: Set<String> =
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
            "simplify",
            "spec",
            "specification",
            "label"
        )

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

    /**
     * Normalizes command aliases to their canonical form.
     * e.g., "impl" -> "implement", "st" -> "status", "cont" -> "continue"
     */
    fun normalizeCommand(command: String): String =
        when (command.lowercase()) {
            "impl" -> "implement"
            "st" -> "status"
            "cont" -> "continue"
            "spec" -> "specification"
            else -> command.lowercase()
        }
}
