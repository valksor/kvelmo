package com.valksor.mehrhof.util

import org.junit.jupiter.api.Assertions.*
import org.junit.jupiter.api.Test

class ReconnectionPolicyTest {
    // ========================================================================
    // Constructor and initial state tests
    // ========================================================================

    @Test
    fun `fresh policy has zero attempts`() {
        val policy = ReconnectionPolicy(maxAttempts = 5, delaySeconds = 10)
        assertEquals(0, policy.currentAttempts())
    }

    @Test
    fun `fresh policy shouldReconnect is true`() {
        val policy = ReconnectionPolicy(maxAttempts = 5, delaySeconds = 10)
        assertTrue(policy.shouldReconnect())
    }

    // ========================================================================
    // recordAttempt tests
    // ========================================================================

    @Test
    fun `recordAttempt increments and returns new count`() {
        val policy = ReconnectionPolicy(maxAttempts = 5, delaySeconds = 10)
        assertEquals(1, policy.recordAttempt())
        assertEquals(2, policy.recordAttempt())
        assertEquals(3, policy.recordAttempt())
    }

    @Test
    fun `recordAttempt is reflected in currentAttempts`() {
        val policy = ReconnectionPolicy(maxAttempts = 5, delaySeconds = 10)
        policy.recordAttempt()
        policy.recordAttempt()
        assertEquals(2, policy.currentAttempts())
    }

    // ========================================================================
    // shouldReconnect boundary tests
    // ========================================================================

    @Test
    fun `shouldReconnect true when under max`() {
        val policy = ReconnectionPolicy(maxAttempts = 3, delaySeconds = 5)
        policy.recordAttempt() // 1
        policy.recordAttempt() // 2
        assertTrue(policy.shouldReconnect())
    }

    @Test
    fun `shouldReconnect false when at max`() {
        val policy = ReconnectionPolicy(maxAttempts = 3, delaySeconds = 5)
        policy.recordAttempt() // 1
        policy.recordAttempt() // 2
        policy.recordAttempt() // 3
        assertFalse(policy.shouldReconnect())
    }

    @Test
    fun `shouldReconnect false when over max`() {
        val policy = ReconnectionPolicy(maxAttempts = 2, delaySeconds = 5)
        policy.recordAttempt() // 1
        policy.recordAttempt() // 2
        policy.recordAttempt() // 3
        assertFalse(policy.shouldReconnect())
    }

    @Test
    fun `maxAttempts of zero means never reconnect`() {
        val policy = ReconnectionPolicy(maxAttempts = 0, delaySeconds = 5)
        assertFalse(policy.shouldReconnect())
    }

    @Test
    fun `maxAttempts of one allows single attempt`() {
        val policy = ReconnectionPolicy(maxAttempts = 1, delaySeconds = 5)
        assertTrue(policy.shouldReconnect())
        policy.recordAttempt()
        assertFalse(policy.shouldReconnect())
    }

    // ========================================================================
    // nextDelayMs tests
    // ========================================================================

    @Test
    fun `nextDelayMs converts seconds to milliseconds`() {
        val policy = ReconnectionPolicy(maxAttempts = 5, delaySeconds = 10)
        assertEquals(10000L, policy.nextDelayMs())
    }

    @Test
    fun `nextDelayMs with 1 second delay`() {
        val policy = ReconnectionPolicy(maxAttempts = 5, delaySeconds = 1)
        assertEquals(1000L, policy.nextDelayMs())
    }

    @Test
    fun `nextDelayMs with 30 second delay`() {
        val policy = ReconnectionPolicy(maxAttempts = 5, delaySeconds = 30)
        assertEquals(30000L, policy.nextDelayMs())
    }

    @Test
    fun `nextDelayMs with zero delay`() {
        val policy = ReconnectionPolicy(maxAttempts = 5, delaySeconds = 0)
        assertEquals(0L, policy.nextDelayMs())
    }

    // ========================================================================
    // reset tests
    // ========================================================================

    @Test
    fun `reset clears attempt counter`() {
        val policy = ReconnectionPolicy(maxAttempts = 3, delaySeconds = 5)
        policy.recordAttempt()
        policy.recordAttempt()
        policy.recordAttempt()
        assertFalse(policy.shouldReconnect())

        policy.reset()
        assertEquals(0, policy.currentAttempts())
        assertTrue(policy.shouldReconnect())
    }

    @Test
    fun `reset allows recording new attempts`() {
        val policy = ReconnectionPolicy(maxAttempts = 2, delaySeconds = 5)
        policy.recordAttempt()
        policy.recordAttempt()
        assertFalse(policy.shouldReconnect())

        policy.reset()
        assertEquals(1, policy.recordAttempt())
        assertTrue(policy.shouldReconnect())
    }

    @Test
    fun `multiple resets work correctly`() {
        val policy = ReconnectionPolicy(maxAttempts = 1, delaySeconds = 5)
        policy.recordAttempt()
        assertFalse(policy.shouldReconnect())

        policy.reset()
        assertTrue(policy.shouldReconnect())

        policy.recordAttempt()
        assertFalse(policy.shouldReconnect())

        policy.reset()
        assertTrue(policy.shouldReconnect())
    }

    // ========================================================================
    // Thread safety (basic validation)
    // ========================================================================

    @Test
    fun `concurrent recordAttempt calls are safe`() {
        val policy = ReconnectionPolicy(maxAttempts = 1000, delaySeconds = 1)
        val threads =
            (1..10).map {
                Thread {
                    repeat(100) { policy.recordAttempt() }
                }
            }
        threads.forEach { it.start() }
        threads.forEach { it.join() }

        assertEquals(1000, policy.currentAttempts())
    }
}
