package com.valksor.mehrhof.api.models

import com.google.gson.annotations.SerializedName

// ============================================================================
// Browser Models
// ============================================================================

data class BrowserStatusResponse(
    val connected: Boolean,
    val host: String? = null,
    val port: Int? = null,
    val tabs: List<BrowserTab>? = null,
    val error: String? = null
)

data class BrowserTab(
    val id: String,
    val title: String,
    val url: String
)

data class BrowserTabsResponse(
    val tabs: List<BrowserTab>,
    val count: Int
)

data class BrowserGotoRequest(
    val url: String
)

data class BrowserGotoResponse(
    val success: Boolean,
    val tab: BrowserTab? = null
)

data class BrowserNavigateRequest(
    @SerializedName("tab_id")
    val tabId: String? = null,
    val url: String
)

data class BrowserNavigateResponse(
    val success: Boolean,
    val message: String? = null
)

data class BrowserClickRequest(
    @SerializedName("tab_id")
    val tabId: String? = null,
    val selector: String
)

data class BrowserClickResponse(
    val success: Boolean,
    val selector: String? = null
)

data class BrowserTypeRequest(
    @SerializedName("tab_id")
    val tabId: String? = null,
    val selector: String,
    val text: String,
    val clear: Boolean = false
)

data class BrowserTypeResponse(
    val success: Boolean,
    val selector: String? = null
)

data class BrowserEvalRequest(
    @SerializedName("tab_id")
    val tabId: String? = null,
    val expression: String
)

data class BrowserEvalResponse(
    val success: Boolean,
    val result: Any? = null
)

data class BrowserScreenshotRequest(
    @SerializedName("tab_id")
    val tabId: String? = null,
    val format: String? = null,
    val quality: Int? = null,
    @SerializedName("full_page")
    val fullPage: Boolean = false
)

data class BrowserScreenshotResponse(
    val success: Boolean,
    val format: String? = null,
    val data: String? = null,
    val size: Int? = null,
    val encoding: String? = null
)

data class BrowserReloadRequest(
    @SerializedName("tab_id")
    val tabId: String? = null,
    val hard: Boolean = false
)

data class BrowserReloadResponse(
    val success: Boolean,
    val message: String? = null
)

data class BrowserCloseRequest(
    @SerializedName("tab_id")
    val tabId: String
)

data class BrowserCloseResponse(
    val success: Boolean,
    val message: String? = null
)

data class BrowserConsoleRequest(
    @SerializedName("tab_id")
    val tabId: String? = null,
    val duration: Int? = null,
    val level: String? = null
)

data class BrowserConsoleMessage(
    val level: String,
    val text: String,
    val timestamp: String? = null
)

data class BrowserConsoleResponse(
    val success: Boolean,
    val messages: List<BrowserConsoleMessage>? = null,
    val count: Int? = null
)

data class BrowserNetworkRequest(
    @SerializedName("tab_id")
    val tabId: String? = null,
    val duration: Int? = null,
    @SerializedName("capture_body")
    val captureBody: Boolean = false,
    @SerializedName("max_body_size")
    val maxBodySize: Int? = null
)

data class BrowserNetworkEntry(
    val method: String,
    val url: String,
    val status: Int? = null,
    @SerializedName("status_text")
    val statusText: String? = null,
    val timestamp: String,
    @SerializedName("request_body")
    val requestBody: String? = null,
    @SerializedName("response_body")
    val responseBody: String? = null
)

data class BrowserNetworkResponse(
    val success: Boolean,
    val requests: List<BrowserNetworkEntry>? = null,
    val count: Int? = null
)
