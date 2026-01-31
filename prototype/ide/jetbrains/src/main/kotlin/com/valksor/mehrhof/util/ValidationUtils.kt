package com.valksor.mehrhof.util

import java.io.File
import java.net.URI

/**
 * Validation utilities for settings and configuration.
 * Extracted from MehrhofConfigurable for testability.
 */
object ValidationUtils {
    /**
     * Validates a server URL.
     *
     * Rules:
     * - Empty string is valid (means "auto-detect" / use Start Server button)
     * - Must use http or https scheme
     * - Must have a valid host
     *
     * @return Result.success(Unit) if valid, Result.failure with error message if invalid
     */
    fun validateServerUrl(url: String): Result<Unit> {
        if (url.isEmpty()) {
            return Result.success(Unit) // Empty = auto-detect
        }

        return try {
            val uri = URI(url)
            when {
                uri.scheme !in listOf("http", "https") -> {
                    Result.failure(IllegalArgumentException("Server URL must use http or https scheme"))
                }
                uri.host.isNullOrEmpty() -> {
                    Result.failure(IllegalArgumentException("Server URL must have a valid host"))
                }
                else -> Result.success(Unit)
            }
        } catch (e: Exception) {
            Result.failure(IllegalArgumentException("Invalid server URL: ${e.message}"))
        }
    }

    /**
     * Validates an executable path.
     *
     * Rules:
     * - Empty string is valid (means "auto-detect")
     * - If provided, file must exist and be executable
     *
     * @return Result.success(Unit) if valid, Result.failure with error message if invalid
     */
    fun validateExecutablePath(path: String): Result<Unit> {
        if (path.isEmpty()) {
            return Result.success(Unit) // Empty = auto-detect
        }

        val file = File(path)
        return when {
            !file.exists() -> {
                Result.failure(IllegalArgumentException("Executable not found: $path"))
            }
            !file.canExecute() -> {
                Result.failure(IllegalArgumentException("Path is not executable: $path"))
            }
            else -> Result.success(Unit)
        }
    }

    /**
     * Validates a reconnect delay value.
     *
     * Rules:
     * - Must be a positive integer (>= 1)
     *
     * @return Result.success(parsed int) if valid, Result.failure with error message if invalid
     */
    fun validateReconnectDelay(value: String): Result<Int> {
        val delay = value.toIntOrNull()
        return when {
            delay == null -> {
                Result.failure(IllegalArgumentException("Reconnect delay must be a valid integer"))
            }
            delay < 1 -> {
                Result.failure(
                    IllegalArgumentException("Reconnect delay must be a positive integer (minimum 1 second)")
                )
            }
            else -> Result.success(delay)
        }
    }

    /**
     * Validates a max reconnect attempts value.
     *
     * Rules:
     * - Must be a non-negative integer (>= 0)
     * - 0 means no reconnection attempts
     *
     * @return Result.success(parsed int) if valid, Result.failure with error message if invalid
     */
    fun validateMaxAttempts(value: String): Result<Int> {
        val attempts = value.toIntOrNull()
        return when {
            attempts == null -> {
                Result.failure(IllegalArgumentException("Max reconnect attempts must be a valid integer"))
            }
            attempts < 0 -> {
                Result.failure(IllegalArgumentException("Max reconnect attempts must be a non-negative integer"))
            }
            else -> Result.success(attempts)
        }
    }
}
