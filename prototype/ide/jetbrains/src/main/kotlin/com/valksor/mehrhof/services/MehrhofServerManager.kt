package com.valksor.mehrhof.services

import com.intellij.openapi.diagnostic.Logger
import com.intellij.openapi.project.Project
import com.intellij.util.EnvironmentUtil
import com.valksor.mehrhof.settings.MehrhofSettings
import com.valksor.mehrhof.util.BinaryDetector
import kotlinx.coroutines.*
import java.io.File

/**
 * Manages the lifecycle of a `mehr serve` server process.
 *
 * Responsibilities:
 * - Finding the mehr binary (delegates to [BinaryDetector])
 * - Starting the server process and parsing port from output
 * - Stopping the server process with graceful shutdown
 *
 * Communicates back to the owning service via callbacks.
 */
class MehrhofServerManager(
    private val scope: CoroutineScope,
    private val onServerReady: (String) -> Unit,
    private val onError: (String) -> Unit,
    private val onInfo: (String) -> Unit,
    private val onProcessExited: () -> Unit
) {
    private val log = Logger.getInstance(MehrhofServerManager::class.java)

    private var serverProcess: Process? = null
    private var serverPort: Int? = null
    private var serverOutputJob: Job? = null

    /**
     * Check if the server process is currently alive.
     */
    fun isRunning(): Boolean = serverProcess?.isAlive == true

    /**
     * Get the server port, or null if not running.
     */
    fun getServerPort(): Int? = serverPort

    /**
     * Find the mehr binary, checking user config then default install locations.
     */
    fun findMehrBinary(settings: MehrhofSettings): String {
        val result = BinaryDetector.findMehrBinary(configuredPath = settings.mehrExecutable)
        return result.getOrThrow()
    }

    /**
     * Start the Mehrhof server for the given project.
     * Spawns `mehr serve --api` and captures the port from output.
     */
    fun startServer(
        project: Project,
        settings: MehrhofSettings
    ) {
        if (serverProcess?.isAlive == true) {
            log.info("Server already running")
            return
        }

        val projectPath = project.basePath
        if (projectPath == null) {
            onError("Cannot start server: no project path")
            return
        }

        val mehrBinary: String
        try {
            mehrBinary = findMehrBinary(settings)
        } catch (e: IllegalStateException) {
            onError(e.message ?: "mehr not found")
            return
        }

        log.info("Starting Mehrhof server in $projectPath using $mehrBinary")

        try {
            val processBuilder =
                ProcessBuilder(mehrBinary, "serve", "--api")
                    .directory(File(projectPath))
                    .redirectErrorStream(true)

            // Apply user's shell environment from EnvironmentUtil
            val env = processBuilder.environment()
            env.putAll(EnvironmentUtil.getEnvironmentMap())

            serverProcess = processBuilder.start()

            // Read stdout in background, parse for port
            serverOutputJob =
                scope.launch(Dispatchers.IO) {
                    val output = StringBuilder()
                    val process = serverProcess ?: return@launch

                    try {
                        process.inputStream?.bufferedReader()?.useLines { lines ->
                            for (line in lines) {
                                output.appendLine(line)
                                log.info("Server: $line")

                                // Parse: "Server running at: http://localhost:XXXXX"
                                val match = Regex("""Server running at: https?://[^:]+:(\d+)""").find(line)
                                if (match != null) {
                                    val port = match.groupValues[1].toIntOrNull()
                                    if (port != null) {
                                        serverPort = port
                                        val url = "http://localhost:$port"
                                        log.info("Server started on port $port")

                                        withContext(Dispatchers.Main) {
                                            onInfo("Server started on port $port")
                                        }

                                        onServerReady(url)
                                    }
                                }
                            }
                        }
                    } catch (e: Exception) {
                        if (e !is CancellationException) {
                            log.warn("Error reading server output: ${e.message}")
                        }
                    }

                    // Process ended - capture exit code
                    val exitCode =
                        try {
                            process.waitFor()
                        } catch (_: Exception) {
                            -1
                        }
                    val capturedPort = serverPort

                    withContext(Dispatchers.Main) {
                        if (capturedPort == null) {
                            val lastOutput = output.toString().takeLast(500)
                            onError("Server exited (code $exitCode):\n$lastOutput")
                        }
                        serverProcess = null
                        serverPort = null
                        onProcessExited()
                    }
                }
        } catch (e: Exception) {
            log.error("Failed to start server: ${e.message}")
            onError("Failed to start server: ${e.message}")
            serverProcess = null
        }
    }

    /**
     * Stop the Mehrhof server process.
     *
     * @param preShutdown callback invoked before process destruction (e.g., to disconnect clients)
     */
    fun stopServer(preShutdown: () -> Unit = {}) {
        log.info("Stopping Mehrhof server")

        preShutdown()

        // Cancel output reading job
        serverOutputJob?.cancel()
        serverOutputJob = null

        // Destroy the process
        serverProcess?.let { process ->
            process.destroy()
            // Wait a bit for graceful shutdown
            scope.launch(Dispatchers.IO) {
                try {
                    withTimeout(5000) {
                        while (process.isAlive) {
                            delay(100)
                        }
                    }
                } catch (_: TimeoutCancellationException) {
                    process.destroyForcibly()
                }
            }
        }

        serverProcess = null
        serverPort = null

        onInfo("Server stopped")
    }

    /**
     * Clean up server resources without graceful shutdown.
     * Used during disposal when the coroutine scope is about to be cancelled.
     */
    fun dispose() {
        serverOutputJob?.cancel()
        serverProcess?.destroy()
        serverProcess = null
        serverPort = null
    }
}
