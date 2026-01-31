package com.valksor.mehrhof.util

import java.io.File

/**
 * Utility for finding the mehr binary executable.
 */
object BinaryDetector {
    /**
     * Default installation locations to check for the mehr binary.
     */
    fun getDefaultCandidates(): List<String> {
        val home = System.getProperty("user.home")
        return listOf(
            "$home/.local/bin/mehr",
            "$home/bin/mehr",
            "/usr/local/bin/mehr"
        )
    }

    /**
     * Finds the mehr binary, checking the configured path first,
     * then falling back to default installation locations.
     *
     * @param configuredPath User-configured path (may be empty for auto-detect)
     * @param candidates List of candidate paths to check (defaults to standard locations)
     * @return Result.success with the path, or Result.failure with an error message
     */
    fun findMehrBinary(
        configuredPath: String = "",
        candidates: List<String> = getDefaultCandidates()
    ): Result<String> {
        // User configured path takes priority
        if (configuredPath.isNotEmpty()) {
            val file = File(configuredPath)
            return if (file.canExecute()) {
                Result.success(configuredPath)
            } else if (!file.exists()) {
                Result.failure(IllegalStateException("Configured mehr path not found: $configuredPath"))
            } else {
                Result.failure(IllegalStateException("Configured mehr path is not executable: $configuredPath"))
            }
        }

        // Try default install locations
        for (path in candidates) {
            if (File(path).canExecute()) {
                return Result.success(path)
            }
        }

        return Result.failure(
            IllegalStateException(
                "mehr not found. Install with 'curl -fsSL https://valksor.com/install | bash' " +
                    "or configure path in Settings → Tools → Mehrhof"
            )
        )
    }

    /**
     * Checks if a path is a valid executable.
     */
    fun isValidExecutable(path: String): Boolean {
        if (path.isEmpty()) return false
        val file = File(path)
        return file.exists() && file.canExecute()
    }

    /**
     * Attempts to auto-detect the mehr binary.
     * Returns the first valid path from default candidates, or null if not found.
     */
    fun autoDetect(): String? = getDefaultCandidates().firstOrNull { isValidExecutable(it) }
}
