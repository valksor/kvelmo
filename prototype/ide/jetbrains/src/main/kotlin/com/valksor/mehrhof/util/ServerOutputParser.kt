package com.valksor.mehrhof.util

/**
 * Utilities for parsing server output and detecting connection information.
 */
object ServerOutputParser {
    /**
     * Pattern to match server startup messages like:
     * "Server running at: http://localhost:3000"
     * "Server running at: https://localhost:8080"
     */
    private val serverStartPattern = Regex("""Server running at: https?://[^:]+:(\d+)""")

    /**
     * Parses a server output line to extract the port number.
     * Returns null if the line doesn't contain a server start message.
     *
     * @param line A line from the server's stdout
     * @return The port number if found, null otherwise
     */
    fun parseServerPort(line: String): Int? {
        val match = serverStartPattern.find(line)
        return match?.groupValues?.getOrNull(1)?.toIntOrNull()
    }

    /**
     * Extracts the full URL from a server startup line.
     * Returns null if the line doesn't contain a server start message.
     *
     * @param line A line from the server's stdout
     * @return The full server URL if found, null otherwise
     */
    fun parseServerUrl(line: String): String? {
        val urlPattern = Regex("""(https?://[^\s]+)""")
        val match = urlPattern.find(line)
        return match?.value
    }

    /**
     * Checks if a line indicates the server has started successfully.
     */
    fun isServerStartMessage(line: String): Boolean = serverStartPattern.containsMatchIn(line)

    /**
     * Extracts error messages from server output.
     * Common error patterns:
     * - "Error: <message>"
     * - "error: <message>"
     * - "FATAL: <message>"
     */
    fun parseErrorMessage(line: String): String? {
        val errorPatterns =
            listOf(
                Regex("""(?i)^Error:\s*(.+)$"""),
                Regex("""(?i)^FATAL:\s*(.+)$"""),
                Regex("""(?i)^Failed:\s*(.+)$""")
            )

        for (pattern in errorPatterns) {
            val match = pattern.find(line)
            if (match != null) {
                return match.groupValues.getOrNull(1)?.trim()
            }
        }
        return null
    }

    /**
     * Checks if a line indicates an error condition.
     */
    fun isErrorLine(line: String): Boolean = parseErrorMessage(line) != null
}
