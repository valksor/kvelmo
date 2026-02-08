package com.valksor.mehrhof.services

import com.intellij.openapi.project.Project
import com.valksor.mehrhof.settings.MehrhofSettings
import com.valksor.mehrhof.util.BinaryDetector
import io.mockk.every
import io.mockk.mockk
import io.mockk.mockkObject
import io.mockk.unmockkAll
import io.mockk.verify
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import org.junit.jupiter.api.AfterEach
import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.Assertions.assertFalse
import org.junit.jupiter.api.Assertions.assertNull
import org.junit.jupiter.api.Assertions.assertThrows
import org.junit.jupiter.api.Assertions.assertTrue
import org.junit.jupiter.api.BeforeEach
import org.junit.jupiter.api.Test

/**
 * Unit tests for [MehrhofServerManager].
 *
 * Tests server lifecycle management: binary detection, process startup/shutdown,
 * port parsing from server output, and callback invocation.
 */
class MehrhofServerManagerTest {
    private lateinit var scope: CoroutineScope
    private lateinit var settings: MehrhofSettings
    private lateinit var project: Project

    private var serverReadyUrl: String? = null
    private var lastError: String? = null
    private var lastInfo: String? = null
    private var processExitedCalled: Boolean = false

    private lateinit var manager: MehrhofServerManager

    @BeforeEach
    fun setUp() {
        scope = CoroutineScope(Dispatchers.IO + SupervisorJob())
        settings = mockk(relaxed = true)
        project = mockk(relaxed = true)

        serverReadyUrl = null
        lastError = null
        lastInfo = null
        processExitedCalled = false

        manager =
            MehrhofServerManager(
                scope = scope,
                onServerReady = { url -> serverReadyUrl = url },
                onError = { message -> lastError = message },
                onInfo = { message -> lastInfo = message },
                onProcessExited = { processExitedCalled = true },
            )

        mockkObject(BinaryDetector)
    }

    @AfterEach
    fun tearDown() {
        unmockkAll()
    }

    // ========================================================================
    // Initial State Tests
    // ========================================================================

    @Test
    fun `isRunning returns false when no process started`() {
        assertFalse(manager.isRunning())
    }

    @Test
    fun `getServerPort returns null when not running`() {
        assertNull(manager.getServerPort())
    }

    // ========================================================================
    // findMehrBinary Tests
    // ========================================================================

    @Test
    fun `findMehrBinary delegates to BinaryDetector`() {
        every { settings.mehrExecutable } returns "/custom/path/mehr"
        every {
            BinaryDetector.findMehrBinary(configuredPath = "/custom/path/mehr")
        } returns Result.success("/custom/path/mehr")

        val result = manager.findMehrBinary(settings)

        assertEquals("/custom/path/mehr", result)
        verify { BinaryDetector.findMehrBinary(configuredPath = "/custom/path/mehr") }
    }

    @Test
    fun `findMehrBinary throws when binary not found`() {
        every { settings.mehrExecutable } returns ""
        every {
            BinaryDetector.findMehrBinary(configuredPath = "")
        } returns Result.failure(IllegalStateException("mehr binary not found"))

        assertThrows(IllegalStateException::class.java) {
            manager.findMehrBinary(settings)
        }
    }

    // ========================================================================
    // startServer Tests
    // ========================================================================

    @Test
    fun `startServer calls onError when project has no basePath`() {
        every { project.basePath } returns null

        manager.startServer(project, settings)

        assertEquals("Cannot start server: no project path", lastError)
    }

    @Test
    fun `startServer calls onError when binary not found`() {
        every { project.basePath } returns "/project"
        every { settings.mehrExecutable } returns ""
        every {
            BinaryDetector.findMehrBinary(configuredPath = "")
        } returns Result.failure(IllegalStateException("mehr not found in PATH"))

        manager.startServer(project, settings)

        assertEquals("mehr not found in PATH", lastError)
    }

    @Test
    fun `startServer does nothing if already running`() {
        // Create a mock process that is alive
        val mockProcess = mockk<Process>(relaxed = true)
        every { mockProcess.isAlive } returns true

        // Use reflection to set the private serverProcess field
        val field = MehrhofServerManager::class.java.getDeclaredField("serverProcess")
        field.isAccessible = true
        field.set(manager, mockProcess)

        every { project.basePath } returns "/project"
        every { settings.mehrExecutable } returns "/usr/local/bin/mehr"
        every {
            BinaryDetector.findMehrBinary(configuredPath = "/usr/local/bin/mehr")
        } returns Result.success("/usr/local/bin/mehr")

        manager.startServer(project, settings)

        // Should not attempt to start a new process (no error, no info)
        assertNull(lastError)
        assertNull(lastInfo)
    }

    // ========================================================================
    // Port Parsing Tests
    // ========================================================================

    @Test
    fun `parsePort extracts port from server output`() {
        // Test the regex pattern used in startServer
        val regex = Regex("""Server running at: https?://[^:]+:(\d+)""")

        val testCases =
            listOf(
                "Server running at: http://localhost:8080" to 8080,
                "Server running at: https://127.0.0.1:3000" to 3000,
                "Server running at: http://0.0.0.0:12345" to 12345,
            )

        for ((input, expectedPort) in testCases) {
            val match = regex.find(input)
            val port = match?.groupValues?.get(1)?.toIntOrNull()
            assertEquals(expectedPort, port, "Failed for input: $input")
        }
    }

    @Test
    fun `parsePort returns null for invalid output`() {
        val regex = Regex("""Server running at: https?://[^:]+:(\d+)""")

        val invalidInputs =
            listOf(
                "Starting server...",
                "Server running at: http://localhost", // no port
                "Error: port already in use",
            )

        for (input in invalidInputs) {
            val match = regex.find(input)
            assertNull(match, "Should not match: $input")
        }
    }

    // ========================================================================
    // stopServer Tests
    // ========================================================================

    @Test
    fun `stopServer calls preShutdown callback`() {
        var preShutdownCalled = false

        manager.stopServer(preShutdown = { preShutdownCalled = true })

        assertTrue(preShutdownCalled)
    }

    @Test
    fun `stopServer clears serverProcess and serverPort`() {
        // Set up a mock process
        val mockProcess = mockk<Process>(relaxed = true)
        every { mockProcess.isAlive } returns true

        val processField = MehrhofServerManager::class.java.getDeclaredField("serverProcess")
        processField.isAccessible = true
        processField.set(manager, mockProcess)

        val portField = MehrhofServerManager::class.java.getDeclaredField("serverPort")
        portField.isAccessible = true
        portField.set(manager, 8080)

        manager.stopServer()

        assertFalse(manager.isRunning())
        assertNull(manager.getServerPort())
        assertEquals("Server stopped", lastInfo)
    }

    // ========================================================================
    // dispose Tests
    // ========================================================================

    @Test
    fun `dispose clears all resources`() {
        // Set up mock resources
        val mockProcess = mockk<Process>(relaxed = true)

        val processField = MehrhofServerManager::class.java.getDeclaredField("serverProcess")
        processField.isAccessible = true
        processField.set(manager, mockProcess)

        val portField = MehrhofServerManager::class.java.getDeclaredField("serverPort")
        portField.isAccessible = true
        portField.set(manager, 8080)

        manager.dispose()

        verify { mockProcess.destroy() }
        assertFalse(manager.isRunning())
        assertNull(manager.getServerPort())
    }
}
