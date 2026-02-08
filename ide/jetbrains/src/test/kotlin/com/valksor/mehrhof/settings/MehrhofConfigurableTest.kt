package com.valksor.mehrhof.settings

import com.intellij.openapi.options.ConfigurationException
import com.intellij.ui.components.JBCheckBox
import com.intellij.ui.components.JBTextField
import io.mockk.every
import io.mockk.mockkObject
import io.mockk.unmockkAll
import org.junit.jupiter.api.AfterEach
import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.Assertions.assertFalse
import org.junit.jupiter.api.Assertions.assertNotNull
import org.junit.jupiter.api.Assertions.assertThrows
import org.junit.jupiter.api.Assertions.assertTrue
import org.junit.jupiter.api.BeforeEach
import org.junit.jupiter.api.Test
import javax.swing.JPanel

/**
 * Unit tests for [MehrhofConfigurable] settings page.
 *
 * Tests the Settings > Mehrhof configuration panel behavior.
 */
class MehrhofConfigurableTest {
    private lateinit var settings: MehrhofSettings
    private lateinit var configurable: MehrhofConfigurable

    @BeforeEach
    fun setUp() {
        settings = MehrhofSettings()

        mockkObject(MehrhofSettings.Companion)
        every { MehrhofSettings.getInstance() } returns settings

        configurable = MehrhofConfigurable()
    }

    @AfterEach
    fun tearDown() {
        unmockkAll()
    }

    // ========================================================================
    // Display Name Tests
    // ========================================================================

    @Test
    fun `getDisplayName returns Mehrhof`() {
        assertEquals("Mehrhof", configurable.displayName)
    }

    // ========================================================================
    // Component Creation Tests
    // ========================================================================

    @Test
    fun `createComponent returns a panel`() {
        val component = configurable.createComponent()

        assertNotNull(component)
        assertTrue(component is JPanel)
    }

    @Test
    fun `createComponent can be called multiple times`() {
        val component1 = configurable.createComponent()
        val component2 = configurable.createComponent()

        assertNotNull(component1)
        assertNotNull(component2)
    }

    // ========================================================================
    // isModified Tests
    // ========================================================================

    @Test
    fun `isModified returns false when no changes`() {
        // Initialize fields by calling createComponent
        configurable.createComponent()

        // Reset to match settings
        configurable.reset()

        assertFalse(configurable.isModified)
    }

    @Test
    fun `isModified returns true when server URL changes`() {
        configurable.createComponent()
        configurable.reset()

        // Access and modify the serverUrlField via reflection
        val field = MehrhofConfigurable::class.java.getDeclaredField("serverUrlField")
        field.isAccessible = true
        val serverUrlField = field.get(configurable) as JBTextField
        serverUrlField.text = "http://localhost:9876"

        assertTrue(configurable.isModified)
    }

    @Test
    fun `isModified returns true when showNotifications changes`() {
        configurable.createComponent()
        configurable.reset()

        val field = MehrhofConfigurable::class.java.getDeclaredField("showNotificationsCheckbox")
        field.isAccessible = true
        val checkbox = field.get(configurable) as JBCheckBox

        // Toggle the checkbox
        checkbox.isSelected = !settings.showNotifications

        assertTrue(configurable.isModified)
    }

    @Test
    fun `isModified returns true when autoReconnect changes`() {
        configurable.createComponent()
        configurable.reset()

        val field = MehrhofConfigurable::class.java.getDeclaredField("autoReconnectCheckbox")
        field.isAccessible = true
        val checkbox = field.get(configurable) as JBCheckBox

        checkbox.isSelected = !settings.autoReconnect

        assertTrue(configurable.isModified)
    }

    // ========================================================================
    // Apply Tests
    // ========================================================================

    @Test
    fun `apply saves valid settings`() {
        configurable.createComponent()
        configurable.reset()

        // Modify settings via reflection
        val urlField = getDeclaredField<JBTextField>("serverUrlField")
        val notifCheckbox = getDeclaredField<JBCheckBox>("showNotificationsCheckbox")
        val autoReconnectCheckbox = getDeclaredField<JBCheckBox>("autoReconnectCheckbox")
        val defaultAgentField = getDeclaredField<JBTextField>("defaultAgentField")

        urlField.text = "http://localhost:8080"
        notifCheckbox.isSelected = false
        autoReconnectCheckbox.isSelected = true
        defaultAgentField.text = "custom-agent"

        configurable.apply()

        assertEquals("http://localhost:8080", settings.serverUrl)
        assertFalse(settings.showNotifications)
        assertTrue(settings.autoReconnect)
        assertEquals("custom-agent", settings.defaultAgent)
    }

    @Test
    fun `apply throws ConfigurationException for invalid URL`() {
        configurable.createComponent()
        configurable.reset()

        val urlField = getDeclaredField<JBTextField>("serverUrlField")
        urlField.text = "not-a-valid-url"

        assertThrows(ConfigurationException::class.java) {
            configurable.apply()
        }
    }

    @Test
    fun `apply throws ConfigurationException for invalid reconnect delay`() {
        configurable.createComponent()
        configurable.reset()

        val delayField = getDeclaredField<JBTextField>("reconnectDelayField")
        delayField.text = "not-a-number"

        assertThrows(ConfigurationException::class.java) {
            configurable.apply()
        }
    }

    @Test
    fun `apply throws ConfigurationException for invalid max attempts`() {
        configurable.createComponent()
        configurable.reset()

        val attemptsField = getDeclaredField<JBTextField>("maxReconnectAttemptsField")
        attemptsField.text = "-5"

        assertThrows(ConfigurationException::class.java) {
            configurable.apply()
        }
    }

    // ========================================================================
    // Reset Tests
    // ========================================================================

    @Test
    fun `reset restores values from settings`() {
        settings.serverUrl = "http://test-server:9999"
        settings.showNotifications = false
        settings.autoReconnect = true
        settings.defaultAgent = "test-agent"
        settings.reconnectDelaySeconds = 10
        settings.maxReconnectAttempts = 5

        configurable.createComponent()
        configurable.reset()

        val urlField = getDeclaredField<JBTextField>("serverUrlField")
        val notifCheckbox = getDeclaredField<JBCheckBox>("showNotificationsCheckbox")
        val autoReconnectCheckbox = getDeclaredField<JBCheckBox>("autoReconnectCheckbox")
        val defaultAgentField = getDeclaredField<JBTextField>("defaultAgentField")
        val delayField = getDeclaredField<JBTextField>("reconnectDelayField")
        val attemptsField = getDeclaredField<JBTextField>("maxReconnectAttemptsField")

        assertEquals("http://test-server:9999", urlField.text)
        assertFalse(notifCheckbox.isSelected)
        assertTrue(autoReconnectCheckbox.isSelected)
        assertEquals("test-agent", defaultAgentField.text)
        assertEquals("10", delayField.text)
        assertEquals("5", attemptsField.text)
    }

    @Test
    fun `reset clears isModified state`() {
        configurable.createComponent()
        configurable.reset()

        // Modify a field
        val urlField = getDeclaredField<JBTextField>("serverUrlField")
        urlField.text = "http://modified:1234"
        assertTrue(configurable.isModified)

        // Reset should clear the modification
        configurable.reset()
        assertFalse(configurable.isModified)
    }

    // ========================================================================
    // Helper Methods
    // ========================================================================

    @Suppress("UNCHECKED_CAST")
    private fun <T> getDeclaredField(name: String): T {
        val field = MehrhofConfigurable::class.java.getDeclaredField(name)
        field.isAccessible = true
        return field.get(configurable) as T
    }
}
