package com.valksor.mehrhof.util

/**
 * Utility for HTML escaping and message formatting.
 * Extracted from InteractivePanel for testability.
 */
object HtmlEscaper {
    /**
     * Escapes HTML special characters and converts newlines to <br> tags.
     *
     * Escapes: & < >
     * Converts: \n -> <br>
     */
    fun escape(text: String): String =
        text
            .replace("&", "&amp;")
            .replace("<", "&lt;")
            .replace(">", "&gt;")
            .replace("\n", "<br>")

    /**
     * Formats a message as an HTML div with the given CSS class.
     * The text content is escaped.
     *
     * @param cssClass The CSS class for the div
     * @param text The text content (will be escaped)
     * @return HTML string: <div class="cssClass">escaped text</div>
     */
    fun formatMessage(
        cssClass: String,
        text: String
    ): String = "<div class=\"$cssClass\">${escape(text)}</div>"

    /**
     * Common CSS classes for message types.
     */
    object CssClasses {
        const val USER = "user"
        const val ASSISTANT = "assistant"
        const val SYSTEM = "system"
        const val ERROR = "error"
        const val COMMAND = "command"
    }

    /**
     * Formats a user message.
     */
    fun formatUserMessage(text: String): String = formatMessage(CssClasses.USER, "You: $text")

    /**
     * Formats an assistant message.
     */
    fun formatAssistantMessage(text: String): String = formatMessage(CssClasses.ASSISTANT, "Agent: $text")

    /**
     * Formats a system message.
     */
    fun formatSystemMessage(text: String): String = formatMessage(CssClasses.SYSTEM, text)

    /**
     * Formats an error message.
     */
    fun formatErrorMessage(text: String): String = formatMessage(CssClasses.ERROR, "Error: $text")

    /**
     * Formats a command message (shows the command that was entered).
     */
    fun formatCommandMessage(text: String): String = formatMessage(CssClasses.COMMAND, "> $text")
}
