package com.valksor.mehrhof.toolwindow

import com.intellij.openapi.project.Project
import com.intellij.ui.components.JBScrollPane
import com.intellij.ui.components.JBTextArea
import com.intellij.util.ui.JBUI
import com.valksor.mehrhof.services.MehrhofProjectService
import java.awt.BorderLayout
import java.awt.Font
import javax.swing.JButton
import javax.swing.JPanel
import javax.swing.SwingUtilities

/**
 * Panel showing agent output and logs.
 */
class OutputPanel(
    @Suppress("unused") private val project: Project,
    private val service: MehrhofProjectService
) : JPanel(BorderLayout()),
    MehrhofProjectService.StateListener {
    private val outputArea =
        JBTextArea().apply {
            isEditable = false
            font = Font(Font.MONOSPACED, Font.PLAIN, 12)
            lineWrap = true
            wrapStyleWord = true
        }

    private val clearButton = JButton("Clear")

    init {
        border = JBUI.Borders.empty(8)

        // Output area with scroll
        add(JBScrollPane(outputArea), BorderLayout.CENTER)

        // Clear button at bottom
        val buttonPanel =
            JPanel().apply {
                add(clearButton)
            }
        add(buttonPanel, BorderLayout.SOUTH)

        // Button actions
        clearButton.addActionListener {
            outputArea.text = ""
        }

        // Register as listener
        service.addStateListener(this)
    }

    override fun onAgentMessage(
        content: String,
        type: String?
    ) {
        SwingUtilities.invokeLater {
            appendOutput(content)
        }
    }

    override fun onWorkflowStateChanged(
        state: String,
        previousState: String?
    ) {
        SwingUtilities.invokeLater {
            appendOutput("\n--- State changed: $previousState → $state ---\n")
        }
    }

    override fun onError(message: String) {
        SwingUtilities.invokeLater {
            appendOutput("\n[ERROR] $message\n")
        }
    }

    override fun onQuestionReceived(
        question: String,
        options: List<String>?
    ) {
        SwingUtilities.invokeLater {
            appendOutput("\n[QUESTION] $question\n")
            options?.forEachIndexed { index, option ->
                appendOutput("  ${index + 1}. $option\n")
            }
        }
    }

    private fun appendOutput(text: String) {
        outputArea.append(text)
        // Auto-scroll to bottom
        outputArea.caretPosition = outputArea.document.length
    }

    fun dispose() {
        service.removeStateListener(this)
    }
}
