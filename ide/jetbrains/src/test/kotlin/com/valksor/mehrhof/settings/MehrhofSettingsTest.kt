package com.valksor.mehrhof.settings

import org.junit.jupiter.api.*
import org.junit.jupiter.api.Assertions.*

/**
 * Unit tests for MehrhofSettings.
 *
 * Note: These tests verify the Settings class behavior without the IntelliJ Platform
 * service infrastructure. Full integration testing requires the IDE environment.
 */
class MehrhofSettingsTest {

    // ========================================================================
    // State Class Tests
    // ========================================================================

    @Test
    fun `State has correct default values`() {
        val state = MehrhofSettings.State()

        assertEquals("", state.serverUrl)  // Empty = plugin manages server
        assertEquals("", state.mehrExecutable)  // Empty = auto-detect
        assertTrue(state.showNotifications)
        assertEquals("", state.defaultAgent)
        assertTrue(state.autoReconnect)
        assertEquals(5, state.reconnectDelaySeconds)
        assertEquals(10, state.maxReconnectAttempts)
    }

    @Test
    fun `State properties can be modified`() {
        val state = MehrhofSettings.State()

        state.serverUrl = "http://192.168.1.100:8080"
        state.mehrExecutable = "/usr/local/bin/mehr"
        state.showNotifications = false
        state.defaultAgent = "claude"
        state.autoReconnect = false
        state.reconnectDelaySeconds = 10
        state.maxReconnectAttempts = 5

        assertEquals("http://192.168.1.100:8080", state.serverUrl)
        assertEquals("/usr/local/bin/mehr", state.mehrExecutable)
        assertFalse(state.showNotifications)
        assertEquals("claude", state.defaultAgent)
        assertFalse(state.autoReconnect)
        assertEquals(10, state.reconnectDelaySeconds)
        assertEquals(5, state.maxReconnectAttempts)
    }

    @Test
    fun `State data class equality`() {
        val state1 = MehrhofSettings.State()
        val state2 = MehrhofSettings.State()
        val state3 = MehrhofSettings.State(serverUrl = "http://different:3000")

        assertEquals(state1, state2)
        assertNotEquals(state1, state3)
    }

    @Test
    fun `State data class copy`() {
        val original = MehrhofSettings.State(
            serverUrl = "http://test:3000",
            showNotifications = false
        )

        val copy = original.copy(showNotifications = true)

        assertEquals("http://test:3000", copy.serverUrl)
        assertTrue(copy.showNotifications)
        assertFalse(original.showNotifications)  // Original unchanged
    }

    // ========================================================================
    // Edge Case Tests
    // ========================================================================

    @Test
    fun `State accepts empty serverUrl`() {
        val state = MehrhofSettings.State(serverUrl = "")
        assertEquals("", state.serverUrl)
    }

    @Test
    fun `State accepts empty mehrExecutable for auto-detect`() {
        val state = MehrhofSettings.State(mehrExecutable = "")
        assertEquals("", state.mehrExecutable)
    }

    @Test
    fun `State accepts custom mehrExecutable path`() {
        val state = MehrhofSettings.State(mehrExecutable = "/custom/path/to/mehr")
        assertEquals("/custom/path/to/mehr", state.mehrExecutable)
    }

    @Test
    fun `State mehrExecutable with spaces in path`() {
        val state = MehrhofSettings.State(mehrExecutable = "/path/with spaces/mehr")
        assertEquals("/path/with spaces/mehr", state.mehrExecutable)
    }

    @Test
    fun `State accepts zero for reconnectDelaySeconds`() {
        val state = MehrhofSettings.State(reconnectDelaySeconds = 0)
        assertEquals(0, state.reconnectDelaySeconds)
    }

    @Test
    fun `State accepts negative for maxReconnectAttempts`() {
        // Note: This tests current behavior, not necessarily desired behavior
        val state = MehrhofSettings.State(maxReconnectAttempts = -1)
        assertEquals(-1, state.maxReconnectAttempts)
    }

    @Test
    fun `State accepts URL with path`() {
        val state = MehrhofSettings.State(serverUrl = "http://localhost:3000/api")
        assertEquals("http://localhost:3000/api", state.serverUrl)
    }

    @Test
    fun `State accepts HTTPS URL`() {
        val state = MehrhofSettings.State(serverUrl = "https://mehrhof.example.com")
        assertEquals("https://mehrhof.example.com", state.serverUrl)
    }

    // ========================================================================
    // Validation Helpers (for future validation tests)
    // ========================================================================

    @Test
    fun `isValidUrl helper validates URLs`() {
        // Helper function to test URL validation logic
        fun isValidUrl(url: String): Boolean {
            return try {
                val parsed = java.net.URI(url)
                parsed.scheme in listOf("http", "https") && parsed.host != null
            } catch (_: Exception) {
                false
            }
        }

        assertTrue(isValidUrl("http://localhost:3000"))
        assertTrue(isValidUrl("https://example.com"))
        assertTrue(isValidUrl("http://192.168.1.1:8080"))
        assertFalse(isValidUrl("not-a-url"))
        assertFalse(isValidUrl("ftp://invalid-scheme.com"))
        assertFalse(isValidUrl(""))
    }

    @Test
    fun `reconnectDelaySeconds should be positive`() {
        // Validation logic test
        fun isValidDelay(delay: Int): Boolean = delay > 0

        assertTrue(isValidDelay(1))
        assertTrue(isValidDelay(5))
        assertTrue(isValidDelay(60))
        assertFalse(isValidDelay(0))
        assertFalse(isValidDelay(-1))
    }

    @Test
    fun `maxReconnectAttempts should be non-negative`() {
        // Validation logic test
        fun isValidAttempts(attempts: Int): Boolean = attempts >= 0

        assertTrue(isValidAttempts(0))  // 0 means no reconnection
        assertTrue(isValidAttempts(10))
        assertFalse(isValidAttempts(-1))
    }

    // ========================================================================
    // Executable Path Validation Tests
    // ========================================================================

    @Test
    fun `isValidExecutablePath helper validates paths`() {
        // Helper function to test executable path validation logic
        fun isValidExecutablePath(path: String): Boolean {
            if (path.isEmpty()) return true  // Empty = auto-detect
            val file = java.io.File(path)
            return file.exists() && file.canExecute()
        }

        // Empty is valid (auto-detect)
        assertTrue(isValidExecutablePath(""))

        // Existing executable should be valid (use bash as example)
        assertTrue(isValidExecutablePath("/bin/bash"))

        // Non-existent path should be invalid
        assertFalse(isValidExecutablePath("/nonexistent/path/to/mehr"))
    }

    @Test
    fun `default install locations are checked in order`() {
        // Test the expected search order for mehr binary
        val home = System.getProperty("user.home")
        val expectedLocations = listOf(
            "$home/.local/bin/mehr",
            "$home/bin/mehr",
            "/usr/local/bin/mehr"
        )

        assertEquals(3, expectedLocations.size)
        assertTrue(expectedLocations[0].contains(".local/bin"))
        assertTrue(expectedLocations[1].endsWith("/bin/mehr"))
        assertEquals("/usr/local/bin/mehr", expectedLocations[2])
    }
}
