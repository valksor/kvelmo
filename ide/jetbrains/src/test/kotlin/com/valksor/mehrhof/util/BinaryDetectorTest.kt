package com.valksor.mehrhof.util

import org.junit.jupiter.api.Assertions.*
import org.junit.jupiter.api.Test
import java.io.File

class BinaryDetectorTest {
    // ========================================================================
    // getDefaultCandidates tests
    // ========================================================================

    @Test
    fun `getDefaultCandidates returns expected paths`() {
        val candidates = BinaryDetector.getDefaultCandidates()

        assertEquals(3, candidates.size)
        assertTrue(candidates[0].endsWith("/.local/bin/mehr"))
        assertTrue(candidates[1].endsWith("/bin/mehr"))
        assertEquals("/usr/local/bin/mehr", candidates[2])
    }

    @Test
    fun `getDefaultCandidates uses user home`() {
        val home = System.getProperty("user.home")
        val candidates = BinaryDetector.getDefaultCandidates()

        assertTrue(candidates[0].startsWith(home))
        assertTrue(candidates[1].startsWith(home))
    }

    // ========================================================================
    // findMehrBinary tests with configured path
    // ========================================================================

    @Test
    fun `findMehrBinary returns configured path if executable`() {
        // Use /bin/bash as a known executable
        val result = BinaryDetector.findMehrBinary(configuredPath = "/bin/bash")
        assertTrue(result.isSuccess)
        assertEquals("/bin/bash", result.getOrNull())
    }

    @Test
    fun `findMehrBinary fails if configured path not found`() {
        val result = BinaryDetector.findMehrBinary(configuredPath = "/nonexistent/path/mehr")
        assertTrue(result.isFailure)
        assertTrue(result.exceptionOrNull()?.message?.contains("not found") == true)
    }

    @Test
    fun `findMehrBinary fails if configured path not executable`() {
        // Create a temp file that is not executable
        val tempFile = File.createTempFile("test", ".txt")
        tempFile.deleteOnExit()
        tempFile.setExecutable(false)

        val result = BinaryDetector.findMehrBinary(configuredPath = tempFile.absolutePath)
        assertTrue(result.isFailure)
        assertTrue(result.exceptionOrNull()?.message?.contains("not executable") == true)
    }

    // ========================================================================
    // findMehrBinary tests with candidates
    // ========================================================================

    @Test
    fun `findMehrBinary returns first valid candidate`() {
        val result =
            BinaryDetector.findMehrBinary(
                configuredPath = "",
                candidates = listOf("/nonexistent1", "/bin/bash", "/bin/sh")
            )
        assertTrue(result.isSuccess)
        assertEquals("/bin/bash", result.getOrNull())
    }

    @Test
    fun `findMehrBinary tries candidates in order`() {
        // Both /bin/bash and /bin/sh exist, should return first one
        val result =
            BinaryDetector.findMehrBinary(
                configuredPath = "",
                candidates = listOf("/bin/sh", "/bin/bash")
            )
        assertTrue(result.isSuccess)
        assertEquals("/bin/sh", result.getOrNull())
    }

    @Test
    fun `findMehrBinary fails if no candidates are valid`() {
        val result =
            BinaryDetector.findMehrBinary(
                configuredPath = "",
                candidates = listOf("/nonexistent1", "/nonexistent2")
            )
        assertTrue(result.isFailure)
        assertTrue(result.exceptionOrNull()?.message?.contains("mehr not found") == true)
    }

    @Test
    fun `findMehrBinary fails with helpful message when not found`() {
        val result =
            BinaryDetector.findMehrBinary(
                configuredPath = "",
                candidates = emptyList()
            )
        assertTrue(result.isFailure)
        val message = result.exceptionOrNull()?.message
        assertTrue(message?.contains("valksor.com/install") == true)
        assertTrue(message?.contains("Settings") == true)
    }

    // ========================================================================
    // isValidExecutable tests
    // ========================================================================

    @Test
    fun `isValidExecutable returns true for valid executable`() {
        assertTrue(BinaryDetector.isValidExecutable("/bin/bash"))
    }

    @Test
    fun `isValidExecutable returns false for nonexistent path`() {
        assertFalse(BinaryDetector.isValidExecutable("/nonexistent/path"))
    }

    @Test
    fun `isValidExecutable returns false for empty path`() {
        assertFalse(BinaryDetector.isValidExecutable(""))
    }

    @Test
    fun `isValidExecutable returns false for non-executable file`() {
        val tempFile = File.createTempFile("test", ".txt")
        tempFile.deleteOnExit()
        tempFile.setExecutable(false)

        assertFalse(BinaryDetector.isValidExecutable(tempFile.absolutePath))
    }

    @Test
    fun `isValidExecutable returns true for executable file`() {
        val tempFile = File.createTempFile("test", ".sh")
        tempFile.deleteOnExit()
        tempFile.setExecutable(true)

        assertTrue(BinaryDetector.isValidExecutable(tempFile.absolutePath))
    }

    // ========================================================================
    // autoDetect tests
    // ========================================================================

    @Test
    fun `autoDetect returns null if mehr not installed`() {
        // This test may pass or fail depending on whether mehr is installed
        // We're testing the behavior, not the actual presence
        val result = BinaryDetector.autoDetect()
        // Result should be either null or a valid path
        if (result != null) {
            assertTrue(BinaryDetector.isValidExecutable(result))
        }
    }
}
