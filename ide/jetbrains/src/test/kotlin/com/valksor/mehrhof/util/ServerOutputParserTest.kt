package com.valksor.mehrhof.util

import org.junit.jupiter.api.Assertions.*
import org.junit.jupiter.api.Test

class ServerOutputParserTest {
    // ========================================================================
    // parseServerPort tests
    // ========================================================================

    @Test
    fun `parseServerPort extracts port from standard message`() {
        val port = ServerOutputParser.parseServerPort("Server running at: http://localhost:3000")
        assertEquals(3000, port)
    }

    @Test
    fun `parseServerPort extracts port from https url`() {
        val port = ServerOutputParser.parseServerPort("Server running at: https://localhost:8443")
        assertEquals(8443, port)
    }

    @Test
    fun `parseServerPort extracts port with IP address`() {
        val port = ServerOutputParser.parseServerPort("Server running at: http://192.168.1.100:8080")
        assertEquals(8080, port)
    }

    @Test
    fun `parseServerPort extracts port with 0 address`() {
        val port = ServerOutputParser.parseServerPort("Server running at: http://0.0.0.0:3000")
        assertEquals(3000, port)
    }

    @Test
    fun `parseServerPort returns null for non-matching line`() {
        assertNull(ServerOutputParser.parseServerPort("Starting server..."))
    }

    @Test
    fun `parseServerPort returns null for empty line`() {
        assertNull(ServerOutputParser.parseServerPort(""))
    }

    @Test
    fun `parseServerPort returns null for line without port`() {
        assertNull(ServerOutputParser.parseServerPort("Server running at: http://localhost"))
    }

    @Test
    fun `parseServerPort handles large port numbers`() {
        val port = ServerOutputParser.parseServerPort("Server running at: http://localhost:65535")
        assertEquals(65535, port)
    }

    @Test
    fun `parseServerPort handles port 80`() {
        // Note: This won't match because our pattern requires explicit port
        assertNull(ServerOutputParser.parseServerPort("Server running at: http://localhost"))
    }

    // ========================================================================
    // parseServerUrl tests
    // ========================================================================

    @Test
    fun `parseServerUrl extracts http url`() {
        val url = ServerOutputParser.parseServerUrl("Server running at: http://localhost:3000")
        assertEquals("http://localhost:3000", url)
    }

    @Test
    fun `parseServerUrl extracts https url`() {
        val url = ServerOutputParser.parseServerUrl("Server running at: https://example.com:8443")
        assertEquals("https://example.com:8443", url)
    }

    @Test
    fun `parseServerUrl extracts url with path`() {
        val url = ServerOutputParser.parseServerUrl("API available at: http://localhost:3000/api/v1")
        assertEquals("http://localhost:3000/api/v1", url)
    }

    @Test
    fun `parseServerUrl returns null for no url`() {
        assertNull(ServerOutputParser.parseServerUrl("Starting server..."))
    }

    @Test
    fun `parseServerUrl returns null for empty line`() {
        assertNull(ServerOutputParser.parseServerUrl(""))
    }

    @Test
    fun `parseServerUrl extracts first url from line with multiple`() {
        val url = ServerOutputParser.parseServerUrl("Visit http://localhost:3000 or https://example.com")
        assertEquals("http://localhost:3000", url)
    }

    // ========================================================================
    // isServerStartMessage tests
    // ========================================================================

    @Test
    fun `isServerStartMessage returns true for valid message`() {
        assertTrue(ServerOutputParser.isServerStartMessage("Server running at: http://localhost:3000"))
    }

    @Test
    fun `isServerStartMessage returns true for https`() {
        assertTrue(ServerOutputParser.isServerStartMessage("Server running at: https://localhost:8443"))
    }

    @Test
    fun `isServerStartMessage returns false for other messages`() {
        assertFalse(ServerOutputParser.isServerStartMessage("Loading configuration..."))
    }

    @Test
    fun `isServerStartMessage returns false for empty string`() {
        assertFalse(ServerOutputParser.isServerStartMessage(""))
    }

    @Test
    fun `isServerStartMessage returns false for partial match`() {
        assertFalse(ServerOutputParser.isServerStartMessage("Server starting..."))
    }

    // ========================================================================
    // parseErrorMessage tests
    // ========================================================================

    @Test
    fun `parseErrorMessage extracts Error message`() {
        val msg = ServerOutputParser.parseErrorMessage("Error: Connection refused")
        assertEquals("Connection refused", msg)
    }

    @Test
    fun `parseErrorMessage extracts error lowercase`() {
        val msg = ServerOutputParser.parseErrorMessage("error: Something went wrong")
        assertEquals("Something went wrong", msg)
    }

    @Test
    fun `parseErrorMessage extracts FATAL message`() {
        val msg = ServerOutputParser.parseErrorMessage("FATAL: Cannot start server")
        assertEquals("Cannot start server", msg)
    }

    @Test
    fun `parseErrorMessage extracts Failed message`() {
        val msg = ServerOutputParser.parseErrorMessage("Failed: Authentication error")
        assertEquals("Authentication error", msg)
    }

    @Test
    fun `parseErrorMessage returns null for non-error line`() {
        assertNull(ServerOutputParser.parseErrorMessage("Server started successfully"))
    }

    @Test
    fun `parseErrorMessage returns null for empty line`() {
        assertNull(ServerOutputParser.parseErrorMessage(""))
    }

    @Test
    fun `parseErrorMessage trims whitespace`() {
        val msg = ServerOutputParser.parseErrorMessage("Error:   Trimmed message  ")
        assertEquals("Trimmed message", msg)
    }

    @Test
    fun `parseErrorMessage handles error in middle of line`() {
        // Our pattern requires Error: at the start
        assertNull(ServerOutputParser.parseErrorMessage("Something Error: happened"))
    }

    // ========================================================================
    // isErrorLine tests
    // ========================================================================

    @Test
    fun `isErrorLine returns true for Error line`() {
        assertTrue(ServerOutputParser.isErrorLine("Error: Something went wrong"))
    }

    @Test
    fun `isErrorLine returns true for FATAL line`() {
        assertTrue(ServerOutputParser.isErrorLine("FATAL: Critical failure"))
    }

    @Test
    fun `isErrorLine returns true for Failed line`() {
        assertTrue(ServerOutputParser.isErrorLine("Failed: Operation failed"))
    }

    @Test
    fun `isErrorLine returns false for normal line`() {
        assertFalse(ServerOutputParser.isErrorLine("Server started"))
    }

    @Test
    fun `isErrorLine returns false for empty line`() {
        assertFalse(ServerOutputParser.isErrorLine(""))
    }

    @Test
    fun `isErrorLine is case insensitive`() {
        assertTrue(ServerOutputParser.isErrorLine("ERROR: caps"))
        assertTrue(ServerOutputParser.isErrorLine("fatal: lowercase"))
    }
}
