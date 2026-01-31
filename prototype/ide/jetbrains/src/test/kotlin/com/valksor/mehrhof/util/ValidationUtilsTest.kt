package com.valksor.mehrhof.util

import org.junit.jupiter.api.Assertions.*
import org.junit.jupiter.api.Test
import java.io.File

class ValidationUtilsTest {
    // ========================================================================
    // validateServerUrl tests
    // ========================================================================

    @Test
    fun `validateServerUrl accepts http`() {
        val result = ValidationUtils.validateServerUrl("http://localhost:3000")
        assertTrue(result.isSuccess)
    }

    @Test
    fun `validateServerUrl accepts https`() {
        val result = ValidationUtils.validateServerUrl("https://example.com")
        assertTrue(result.isSuccess)
    }

    @Test
    fun `validateServerUrl accepts http with path`() {
        val result = ValidationUtils.validateServerUrl("http://localhost:3000/api/v1")
        assertTrue(result.isSuccess)
    }

    @Test
    fun `validateServerUrl accepts IP address`() {
        val result = ValidationUtils.validateServerUrl("http://192.168.1.1:8080")
        assertTrue(result.isSuccess)
    }

    @Test
    fun `validateServerUrl rejects ftp scheme`() {
        val result = ValidationUtils.validateServerUrl("ftp://example.com")
        assertTrue(result.isFailure)
        assertTrue(result.exceptionOrNull()?.message?.contains("http or https") == true)
    }

    @Test
    fun `validateServerUrl rejects ws scheme`() {
        val result = ValidationUtils.validateServerUrl("ws://example.com")
        assertTrue(result.isFailure)
    }

    @Test
    fun `validateServerUrl rejects missing scheme`() {
        val result = ValidationUtils.validateServerUrl("example.com")
        assertTrue(result.isFailure)
    }

    @Test
    fun `validateServerUrl rejects URL without proper host`() {
        // http:/// has empty host after parsing
        val result = ValidationUtils.validateServerUrl("http:///path")
        assertTrue(result.isFailure)
        assertTrue(result.exceptionOrNull()?.message?.contains("host") == true)
    }

    @Test
    fun `validateServerUrl accepts empty as auto-detect`() {
        val result = ValidationUtils.validateServerUrl("")
        assertTrue(result.isSuccess)
    }

    @Test
    fun `validateServerUrl rejects invalid format`() {
        val result = ValidationUtils.validateServerUrl("not a url at all")
        assertTrue(result.isFailure)
    }

    // ========================================================================
    // validateExecutablePath tests
    // ========================================================================

    @Test
    fun `validateExecutablePath accepts empty as auto-detect`() {
        val result = ValidationUtils.validateExecutablePath("")
        assertTrue(result.isSuccess)
    }

    @Test
    fun `validateExecutablePath accepts existing executable`() {
        // Use /bin/bash which should exist and be executable on all Unix systems
        val result = ValidationUtils.validateExecutablePath("/bin/bash")
        assertTrue(result.isSuccess)
    }

    @Test
    fun `validateExecutablePath rejects nonexistent path`() {
        val result = ValidationUtils.validateExecutablePath("/nonexistent/path/to/binary")
        assertTrue(result.isFailure)
        assertTrue(result.exceptionOrNull()?.message?.contains("not found") == true)
    }

    @Test
    fun `validateExecutablePath rejects non-executable file`() {
        // Create a temp file that is not executable
        val tempFile = File.createTempFile("test", ".txt")
        tempFile.deleteOnExit()
        tempFile.setExecutable(false)

        val result = ValidationUtils.validateExecutablePath(tempFile.absolutePath)
        assertTrue(result.isFailure)
        assertTrue(result.exceptionOrNull()?.message?.contains("not executable") == true)
    }

    @Test
    fun `validateExecutablePath accepts executable file`() {
        // Create a temp file and make it executable
        val tempFile = File.createTempFile("test", ".sh")
        tempFile.deleteOnExit()
        tempFile.setExecutable(true)

        val result = ValidationUtils.validateExecutablePath(tempFile.absolutePath)
        assertTrue(result.isSuccess)
    }

    // ========================================================================
    // validateReconnectDelay tests
    // ========================================================================

    @Test
    fun `validateReconnectDelay accepts positive integer`() {
        val result = ValidationUtils.validateReconnectDelay("5")
        assertTrue(result.isSuccess)
        assertEquals(5, result.getOrNull())
    }

    @Test
    fun `validateReconnectDelay accepts 1`() {
        val result = ValidationUtils.validateReconnectDelay("1")
        assertTrue(result.isSuccess)
        assertEquals(1, result.getOrNull())
    }

    @Test
    fun `validateReconnectDelay accepts large number`() {
        val result = ValidationUtils.validateReconnectDelay("3600")
        assertTrue(result.isSuccess)
        assertEquals(3600, result.getOrNull())
    }

    @Test
    fun `validateReconnectDelay rejects zero`() {
        val result = ValidationUtils.validateReconnectDelay("0")
        assertTrue(result.isFailure)
        assertTrue(result.exceptionOrNull()?.message?.contains("positive") == true)
    }

    @Test
    fun `validateReconnectDelay rejects negative`() {
        val result = ValidationUtils.validateReconnectDelay("-1")
        assertTrue(result.isFailure)
        assertTrue(result.exceptionOrNull()?.message?.contains("positive") == true)
    }

    @Test
    fun `validateReconnectDelay rejects non-numeric`() {
        val result = ValidationUtils.validateReconnectDelay("abc")
        assertTrue(result.isFailure)
        assertTrue(result.exceptionOrNull()?.message?.contains("valid integer") == true)
    }

    @Test
    fun `validateReconnectDelay rejects float`() {
        val result = ValidationUtils.validateReconnectDelay("5.5")
        assertTrue(result.isFailure)
    }

    @Test
    fun `validateReconnectDelay rejects empty string`() {
        val result = ValidationUtils.validateReconnectDelay("")
        assertTrue(result.isFailure)
    }

    @Test
    fun `validateReconnectDelay rejects whitespace`() {
        val result = ValidationUtils.validateReconnectDelay("  ")
        assertTrue(result.isFailure)
    }

    // ========================================================================
    // validateMaxAttempts tests
    // ========================================================================

    @Test
    fun `validateMaxAttempts accepts zero`() {
        val result = ValidationUtils.validateMaxAttempts("0")
        assertTrue(result.isSuccess)
        assertEquals(0, result.getOrNull())
    }

    @Test
    fun `validateMaxAttempts accepts positive`() {
        val result = ValidationUtils.validateMaxAttempts("10")
        assertTrue(result.isSuccess)
        assertEquals(10, result.getOrNull())
    }

    @Test
    fun `validateMaxAttempts accepts large number`() {
        val result = ValidationUtils.validateMaxAttempts("100")
        assertTrue(result.isSuccess)
        assertEquals(100, result.getOrNull())
    }

    @Test
    fun `validateMaxAttempts rejects negative`() {
        val result = ValidationUtils.validateMaxAttempts("-1")
        assertTrue(result.isFailure)
        assertTrue(result.exceptionOrNull()?.message?.contains("non-negative") == true)
    }

    @Test
    fun `validateMaxAttempts rejects non-numeric`() {
        val result = ValidationUtils.validateMaxAttempts("ten")
        assertTrue(result.isFailure)
        assertTrue(result.exceptionOrNull()?.message?.contains("valid integer") == true)
    }

    @Test
    fun `validateMaxAttempts rejects empty string`() {
        val result = ValidationUtils.validateMaxAttempts("")
        assertTrue(result.isFailure)
    }

    @Test
    fun `validateMaxAttempts rejects float`() {
        val result = ValidationUtils.validateMaxAttempts("5.0")
        assertTrue(result.isFailure)
    }
}
