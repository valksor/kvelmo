package com.valksor.mehrhof.util

import java.util.concurrent.atomic.AtomicInteger

/**
 * Manages reconnection attempt tracking and delay calculation.
 * Extracted from MehrhofProjectService for testability.
 *
 * @param maxAttempts Maximum number of reconnection attempts before giving up.
 * @param delaySeconds Delay in seconds between reconnection attempts.
 */
class ReconnectionPolicy(
    private val maxAttempts: Int,
    private val delaySeconds: Int
) {
    private val attempts = AtomicInteger(0)

    /**
     * Record a reconnection attempt.
     * @return The current attempt number (1-indexed).
     */
    fun recordAttempt(): Int = attempts.incrementAndGet()

    /**
     * Check if another reconnection attempt should be made.
     * @return true if the number of attempts has not exceeded [maxAttempts].
     */
    fun shouldReconnect(): Boolean = attempts.get() < maxAttempts

    /**
     * Get the current attempt count.
     */
    fun currentAttempts(): Int = attempts.get()

    /**
     * Get the delay in milliseconds before the next reconnection attempt.
     */
    fun nextDelayMs(): Long = delaySeconds * 1000L

    /**
     * Reset the attempt counter (call after successful connection).
     */
    fun reset() {
        attempts.set(0)
    }
}
