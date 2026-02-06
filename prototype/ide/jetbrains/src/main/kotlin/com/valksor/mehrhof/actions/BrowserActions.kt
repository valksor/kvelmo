package com.valksor.mehrhof.actions

import com.intellij.openapi.actionSystem.AnActionEvent
import com.intellij.openapi.ui.Messages
import com.valksor.mehrhof.api.*

// ============================================================================
// Browser Actions
// ============================================================================

class BrowserStatusAction : MehrhofAction() {
    override fun actionPerformed(e: AnActionEvent) {
        val service = getService(e) ?: return
        val client = service.getApiClient() ?: return

        runInBackground(e, "Checking browser status...") {
            client
                .browserStatus()
                .onSuccess { response ->
                    if (!response.connected) {
                        val error = response.error?.let { " ($it)" } ?: ""
                        Messages.showInfoMessage(e.project, "Browser: Not connected$error", "Mehrhof")
                    } else {
                        val message =
                            buildString {
                                appendLine("Browser: Connected")
                                appendLine("Host: ${response.host}:${response.port}")
                                appendLine("Tabs: ${response.tabs?.size ?: 0}")
                            }
                        Messages.showInfoMessage(e.project, message, "Browser Status")
                    }
                }.onFailure { error ->
                    showError(e, "Failed to get browser status: ${error.message}")
                }
        }
    }

    override fun update(e: AnActionEvent) {
        e.presentation.isEnabled = isConnected(e)
    }
}

class BrowserTabsAction : MehrhofAction() {
    override fun actionPerformed(e: AnActionEvent) {
        val service = getService(e) ?: return
        val client = service.getApiClient() ?: return

        runInBackground(e, "Fetching browser tabs...") {
            client
                .browserTabs()
                .onSuccess { response ->
                    if (response.count == 0) {
                        Messages.showInfoMessage(e.project, "No browser tabs open", "Mehrhof")
                    } else {
                        val message =
                            buildString {
                                appendLine("${response.count} tab(s):")
                                response.tabs.forEach { tab ->
                                    val url = truncateUrl(tab.url, 50)
                                    appendLine("• ${tab.title.take(30)} - $url")
                                }
                            }
                        Messages.showInfoMessage(e.project, message, "Browser Tabs")
                    }
                }.onFailure { error ->
                    showError(e, "Failed to list tabs: ${error.message}")
                }
        }
    }

    override fun update(e: AnActionEvent) {
        e.presentation.isEnabled = isConnected(e)
    }

    private fun truncateUrl(
        url: String,
        maxLen: Int
    ): String = if (url.length <= maxLen) url else url.take(maxLen - 3) + "..."
}

class BrowserGotoAction : MehrhofAction() {
    override fun actionPerformed(e: AnActionEvent) {
        val service = getService(e) ?: return
        val client = service.getApiClient() ?: return

        val url =
            Messages.showInputDialog(
                e.project,
                "Enter URL to open:",
                "Browser Go To",
                null
            ) ?: return

        if (url.isBlank()) return

        runInBackground(e, "Opening URL...") {
            client
                .browserGoto(url)
                .onSuccess { response ->
                    if (response.success && response.tab != null) {
                        Messages.showInfoMessage(
                            e.project,
                            "Opened: ${response.tab.title.take(50)}",
                            "Mehrhof"
                        )
                    }
                }.onFailure { error ->
                    showError(e, "Failed to open URL: ${error.message}")
                }
        }
    }

    override fun update(e: AnActionEvent) {
        e.presentation.isEnabled = isConnected(e)
    }
}

class BrowserNavigateAction : MehrhofAction() {
    override fun actionPerformed(e: AnActionEvent) {
        val service = getService(e) ?: return
        val client = service.getApiClient() ?: return

        val url =
            Messages.showInputDialog(
                e.project,
                "Enter URL to navigate current tab to:",
                "Browser Navigate",
                null
            ) ?: return

        if (url.isBlank()) return

        runInBackground(e, "Navigating...") {
            client
                .browserNavigate(url)
                .onSuccess { response ->
                    if (response.success) {
                        Messages.showInfoMessage(e.project, response.message ?: "Navigated", "Mehrhof")
                    }
                }.onFailure { error ->
                    showError(e, "Navigation failed: ${error.message}")
                }
        }
    }

    override fun update(e: AnActionEvent) {
        e.presentation.isEnabled = isConnected(e)
    }
}

class BrowserReloadAction : MehrhofAction() {
    override fun actionPerformed(e: AnActionEvent) {
        val service = getService(e) ?: return
        val client = service.getApiClient() ?: return

        runInBackground(e, "Reloading page...") {
            client
                .browserReload()
                .onSuccess { response ->
                    if (response.success) {
                        Messages.showInfoMessage(e.project, response.message ?: "Page reloaded", "Mehrhof")
                    }
                }.onFailure { error ->
                    showError(e, "Reload failed: ${error.message}")
                }
        }
    }

    override fun update(e: AnActionEvent) {
        e.presentation.isEnabled = isConnected(e)
    }
}

class BrowserScreenshotAction : MehrhofAction() {
    override fun actionPerformed(e: AnActionEvent) {
        val service = getService(e) ?: return
        val client = service.getApiClient() ?: return

        runInBackground(e, "Taking screenshot...") {
            client
                .browserScreenshot()
                .onSuccess { response ->
                    if (response.success && response.data != null) {
                        val sizeKb = (response.size ?: 0) / 1024
                        Messages.showInfoMessage(
                            e.project,
                            "Screenshot captured: ${response.format ?: "png"}, $sizeKb KB",
                            "Mehrhof"
                        )
                    } else {
                        showError(e, "Screenshot failed")
                    }
                }.onFailure { error ->
                    showError(e, "Screenshot failed: ${error.message}")
                }
        }
    }

    override fun update(e: AnActionEvent) {
        e.presentation.isEnabled = isConnected(e)
    }
}

class BrowserClickAction : MehrhofAction() {
    override fun actionPerformed(e: AnActionEvent) {
        val service = getService(e) ?: return
        val client = service.getApiClient() ?: return

        val selector =
            Messages.showInputDialog(
                e.project,
                "Enter CSS selector to click:",
                "Browser Click",
                null
            ) ?: return

        if (selector.isBlank()) return

        runInBackground(e, "Clicking element...") {
            client
                .browserClick(selector)
                .onSuccess { response ->
                    if (response.success) {
                        Messages.showInfoMessage(
                            e.project,
                            "Clicked: ${response.selector ?: selector}",
                            "Mehrhof"
                        )
                    }
                }.onFailure { error ->
                    showError(e, "Click failed: ${error.message}")
                }
        }
    }

    override fun update(e: AnActionEvent) {
        e.presentation.isEnabled = isConnected(e)
    }
}

class BrowserTypeAction : MehrhofAction() {
    override fun actionPerformed(e: AnActionEvent) {
        val service = getService(e) ?: return
        val client = service.getApiClient() ?: return

        val selector =
            Messages.showInputDialog(
                e.project,
                "Enter CSS selector for input element:",
                "Browser Type",
                null
            ) ?: return

        if (selector.isBlank()) return

        val text =
            Messages.showInputDialog(
                e.project,
                "Enter text to type:",
                "Browser Type",
                null
            ) ?: return

        runInBackground(e, "Typing...") {
            client
                .browserType(selector, text)
                .onSuccess { response ->
                    if (response.success) {
                        Messages.showInfoMessage(
                            e.project,
                            "Typed into: ${response.selector ?: selector}",
                            "Mehrhof"
                        )
                    }
                }.onFailure { error ->
                    showError(e, "Type failed: ${error.message}")
                }
        }
    }

    override fun update(e: AnActionEvent) {
        e.presentation.isEnabled = isConnected(e)
    }
}

class BrowserEvalAction : MehrhofAction() {
    override fun actionPerformed(e: AnActionEvent) {
        val service = getService(e) ?: return
        val client = service.getApiClient() ?: return

        val expression =
            Messages.showInputDialog(
                e.project,
                "Enter JavaScript expression to evaluate:",
                "Browser Eval",
                null
            ) ?: return

        if (expression.isBlank()) return

        runInBackground(e, "Evaluating...") {
            client
                .browserEval(expression)
                .onSuccess { response ->
                    if (response.success) {
                        Messages.showInfoMessage(
                            e.project,
                            "Result: ${response.result}",
                            "Browser Eval"
                        )
                    }
                }.onFailure { error ->
                    showError(e, "Eval failed: ${error.message}")
                }
        }
    }

    override fun update(e: AnActionEvent) {
        e.presentation.isEnabled = isConnected(e)
    }
}

class BrowserConsoleAction : MehrhofAction() {
    override fun actionPerformed(e: AnActionEvent) {
        val service = getService(e) ?: return
        val client = service.getApiClient() ?: return

        runInBackground(e, "Fetching console logs...") {
            client
                .browserConsole()
                .onSuccess { response ->
                    val messages = response.messages
                    if (messages.isNullOrEmpty()) {
                        Messages.showInfoMessage(e.project, "No console messages", "Mehrhof")
                    } else {
                        val message =
                            buildString {
                                appendLine("${messages.size} message(s):")
                                messages.take(20).forEach { msg ->
                                    appendLine("[${msg.level.uppercase()}] ${msg.text.take(80)}")
                                }
                                if (messages.size > 20) {
                                    appendLine("... and ${messages.size - 20} more")
                                }
                            }
                        Messages.showInfoMessage(e.project, message, "Browser Console")
                    }
                }.onFailure { error ->
                    showError(e, "Failed to fetch console: ${error.message}")
                }
        }
    }

    override fun update(e: AnActionEvent) {
        e.presentation.isEnabled = isConnected(e)
    }
}

class BrowserNetworkAction : MehrhofAction() {
    override fun actionPerformed(e: AnActionEvent) {
        val service = getService(e) ?: return
        val client = service.getApiClient() ?: return

        runInBackground(e, "Fetching network requests...") {
            client
                .browserNetwork()
                .onSuccess { response ->
                    val requests = response.requests
                    if (requests.isNullOrEmpty()) {
                        Messages.showInfoMessage(e.project, "No network requests", "Mehrhof")
                    } else {
                        val message =
                            buildString {
                                appendLine("${requests.size} request(s):")
                                requests.take(15).forEach { req ->
                                    val status = req.status?.toString() ?: "..."
                                    val url = truncateUrl(req.url, 50)
                                    appendLine("${req.method} $status - $url")
                                }
                                if (requests.size > 15) {
                                    appendLine("... and ${requests.size - 15} more")
                                }
                            }
                        Messages.showInfoMessage(e.project, message, "Browser Network")
                    }
                }.onFailure { error ->
                    showError(e, "Failed to fetch network: ${error.message}")
                }
        }
    }

    override fun update(e: AnActionEvent) {
        e.presentation.isEnabled = isConnected(e)
    }

    private fun truncateUrl(
        url: String,
        maxLen: Int
    ): String = if (url.length <= maxLen) url else url.take(maxLen - 3) + "..."
}
