package com.valksor.mehrhof.toolwindow

import com.intellij.ui.JBColor
import com.intellij.ui.components.JBLabel
import com.intellij.ui.components.JBScrollPane
import com.intellij.util.ui.JBUI
import java.awt.*
import javax.swing.*

/**
 * Actions panel with workflow buttons.
 */
internal class ActionsPanel(
    private val onCommand: (String, List<String>) -> Unit
) : JPanel(BorderLayout()) {
    init {
        border = JBUI.Borders.empty(0, 8)
        preferredSize = Dimension(200, 0)

        val content =
            JPanel().apply {
                layout = BoxLayout(this, BoxLayout.Y_AXIS)
                border = JBUI.Borders.empty(8)
            }

        // Actions section
        content.add(createSectionLabel("Workflow"))
        content.add(Box.createVerticalStrut(8))
        content.add(createButton("Start Task...", "start"))
        content.add(createButton("Plan", "plan"))
        content.add(createButton("Implement", "implement"))
        content.add(createButton("Review", "review"))
        content.add(createButton("Continue", "continue"))
        content.add(Box.createVerticalStrut(8))
        content.add(createButton("Finish", "finish", JBColor.GREEN.darker()))
        content.add(createButton("Abandon", "abandon", JBColor.RED))
        content.add(Box.createVerticalStrut(16))

        // Checkpoints section
        content.add(createSectionLabel("Checkpoints"))
        content.add(Box.createVerticalStrut(8))
        content.add(createButton("Undo", "undo"))
        content.add(createButton("Redo", "redo"))
        content.add(Box.createVerticalStrut(16))

        // Info section
        content.add(createSectionLabel("Info"))
        content.add(Box.createVerticalStrut(8))
        content.add(createButton("Status", "status"))
        content.add(createButton("Cost", "cost"))
        content.add(createButton("Budget", "budget"))
        content.add(createButton("List Tasks", "list"))
        content.add(createButton("Specifications", "specification"))
        content.add(Box.createVerticalStrut(16))

        // Tools section
        content.add(createSectionLabel("Tools"))
        content.add(Box.createVerticalStrut(8))
        content.add(createButton("Find Code...", "find"))
        content.add(createButton("Search Memory...", "memory"))
        content.add(createButton("Library", "library"))
        content.add(createButton("Quick Task...", "quick"))
        content.add(createButton("Simplify", "simplify"))
        content.add(createButton("Add Note...", "note"))

        content.add(Box.createVerticalGlue())

        add(JBScrollPane(content), BorderLayout.CENTER)
    }

    private fun createSectionLabel(text: String): JComponent =
        JBLabel(text).apply {
            font = font.deriveFont(Font.BOLD, 12f)
            alignmentX = Component.LEFT_ALIGNMENT
        }

    private fun createButton(
        text: String,
        command: String,
        color: Color? = null
    ): JButton =
        JButton(text).apply {
            alignmentX = Component.LEFT_ALIGNMENT
            maximumSize = Dimension(Int.MAX_VALUE, preferredSize.height)
            color?.let { foreground = it }
            addActionListener { handleCommand(command) }
        }

    private fun handleCommand(command: String) {
        when (command) {
            "start" -> promptAndExecute("Enter task reference (e.g., github:123):", "Start Task", command)
            "find" -> promptAndExecute("Enter search query:", "Find Code", command)
            "memory" -> promptAndExecute("Enter search query:", "Search Memory", command)
            "quick" -> promptAndExecute("Enter task description:", "Create Quick Task", command)
            "note" -> promptAndExecute("Enter note:", "Add Note", command)
            "finish" -> confirmAndExecute("Complete this task?", command)
            "abandon" -> confirmAndExecute("Discard this task? This will delete the branch!", command)
            else -> onCommand(command, emptyList())
        }
    }

    private fun promptAndExecute(
        prompt: String,
        title: String,
        command: String
    ) {
        val input = JOptionPane.showInputDialog(this, prompt, title, JOptionPane.PLAIN_MESSAGE)
        if (!input.isNullOrBlank()) {
            onCommand(command, listOf(input))
        }
    }

    private fun confirmAndExecute(
        message: String,
        command: String
    ) {
        val result = JOptionPane.showConfirmDialog(this, message, "Confirm", JOptionPane.YES_NO_OPTION)
        if (result == JOptionPane.YES_OPTION) {
            onCommand(command, emptyList())
        }
    }
}
