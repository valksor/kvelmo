package com.valksor.mehrhof.settings

import com.intellij.openapi.fileChooser.FileChooserDescriptorFactory
import com.intellij.openapi.options.Configurable
import com.intellij.openapi.options.ConfigurationException
import com.intellij.openapi.ui.TextFieldWithBrowseButton
import com.intellij.ui.components.JBCheckBox
import com.intellij.ui.components.JBLabel
import com.intellij.ui.components.JBTextField
import com.intellij.util.ui.FormBuilder
import com.valksor.mehrhof.util.ValidationUtils
import javax.swing.JComponent
import javax.swing.JPanel

/**
 * Settings page for configuring the Mehrhof plugin.
 */
class MehrhofConfigurable : Configurable {
    private val settings = MehrhofSettings.getInstance()

    private val mehrExecutableField = TextFieldWithBrowseButton()
    private val serverUrlField = JBTextField()
    private val showNotificationsCheckbox = JBCheckBox("Show notifications")
    private val autoReconnectCheckbox = JBCheckBox("Auto-reconnect on disconnect")
    private val defaultAgentField = JBTextField()
    private val reconnectDelayField = JBTextField()
    private val maxReconnectAttemptsField = JBTextField()

    override fun getDisplayName(): String = "Mehrhof"

    override fun createComponent(): JComponent {
        // Set up executable file chooser
        mehrExecutableField.addBrowseFolderListener(
            "Select mehr Executable",
            "Choose the path to the mehr binary",
            null,
            FileChooserDescriptorFactory.createSingleFileDescriptor()
        )
        (mehrExecutableField.textField as? JBTextField)?.emptyText?.text =
            "Auto-detect from ~/.local/bin, ~/bin, /usr/local/bin"

        serverUrlField.emptyText.text = "Leave empty to use Start Server button"

        return FormBuilder
            .createFormBuilder()
            .addLabeledComponent(JBLabel("mehr executable:"), mehrExecutableField, 1, false)
            .addLabeledComponent(JBLabel("Server URL (optional):"), serverUrlField, 1, false)
            .addComponent(showNotificationsCheckbox, 1)
            .addComponent(autoReconnectCheckbox, 1)
            .addLabeledComponent(JBLabel("Default agent:"), defaultAgentField, 1, false)
            .addLabeledComponent(JBLabel("Reconnect delay (seconds):"), reconnectDelayField, 1, false)
            .addLabeledComponent(JBLabel("Max reconnect attempts:"), maxReconnectAttemptsField, 1, false)
            .addComponentFillVertically(JPanel(), 0)
            .panel
    }

    override fun isModified(): Boolean =
        mehrExecutableField.text != settings.mehrExecutable ||
            serverUrlField.text != settings.serverUrl ||
            showNotificationsCheckbox.isSelected != settings.showNotifications ||
            autoReconnectCheckbox.isSelected != settings.autoReconnect ||
            defaultAgentField.text != settings.defaultAgent ||
            reconnectDelayField.text != settings.reconnectDelaySeconds.toString() ||
            maxReconnectAttemptsField.text != settings.maxReconnectAttempts.toString()

    @Throws(ConfigurationException::class)
    override fun apply() {
        val execPath = mehrExecutableField.text.trim()
        val url = serverUrlField.text.trim()

        // Validate executable path
        ValidationUtils.validateExecutablePath(execPath).onFailure {
            throw ConfigurationException(it.message)
        }

        // Validate URL
        ValidationUtils.validateServerUrl(url).onFailure {
            throw ConfigurationException(it.message)
        }

        // Validate reconnect delay
        val delay =
            ValidationUtils.validateReconnectDelay(reconnectDelayField.text).getOrElse {
                throw ConfigurationException(it.message)
            }

        // Validate max reconnect attempts
        val attempts =
            ValidationUtils.validateMaxAttempts(maxReconnectAttemptsField.text).getOrElse {
                throw ConfigurationException(it.message)
            }

        // Apply validated settings
        settings.mehrExecutable = execPath
        settings.serverUrl = url
        settings.showNotifications = showNotificationsCheckbox.isSelected
        settings.autoReconnect = autoReconnectCheckbox.isSelected
        settings.defaultAgent = defaultAgentField.text.trim()
        settings.reconnectDelaySeconds = delay
        settings.maxReconnectAttempts = attempts
    }

    override fun reset() {
        mehrExecutableField.text = settings.mehrExecutable
        serverUrlField.text = settings.serverUrl
        showNotificationsCheckbox.isSelected = settings.showNotifications
        autoReconnectCheckbox.isSelected = settings.autoReconnect
        defaultAgentField.text = settings.defaultAgent
        reconnectDelayField.text = settings.reconnectDelaySeconds.toString()
        maxReconnectAttemptsField.text = settings.maxReconnectAttempts.toString()
    }
}
