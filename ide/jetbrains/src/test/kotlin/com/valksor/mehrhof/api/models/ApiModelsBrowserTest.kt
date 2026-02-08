package com.valksor.mehrhof.api.models

import com.google.gson.Gson
import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.Assertions.assertFalse
import org.junit.jupiter.api.Assertions.assertNull
import org.junit.jupiter.api.Assertions.assertTrue
import org.junit.jupiter.api.Test

/**
 * Unit tests for browser API model data classes.
 *
 * Tests JSON serialization/deserialization and default values.
 */
class ApiModelsBrowserTest {
    private val gson = Gson()

    // ========================================================================
    // BrowserStatusResponse Tests
    // ========================================================================

    @Test
    fun `BrowserStatusResponse deserializes with all fields`() {
        val json =
            """{
            "connected": true,
            "host": "localhost",
            "port": 9222,
            "tabs": [{"id": "t1", "title": "Test", "url": "http://test.com"}]
        }"""

        val response = gson.fromJson(json, BrowserStatusResponse::class.java)

        assertTrue(response.connected)
        assertEquals("localhost", response.host)
        assertEquals(9222, response.port)
        assertEquals(1, response.tabs?.size)
        assertEquals("t1", response.tabs?.first()?.id)
    }

    @Test
    fun `BrowserStatusResponse handles null optional fields`() {
        val json = """{"connected": false}"""

        val response = gson.fromJson(json, BrowserStatusResponse::class.java)

        assertFalse(response.connected)
        assertNull(response.host)
        assertNull(response.port)
        assertNull(response.tabs)
    }

    // ========================================================================
    // BrowserTab Tests
    // ========================================================================

    @Test
    fun `BrowserTab deserializes correctly`() {
        val json = """{"id": "tab-123", "title": "My Page", "url": "https://example.com"}"""

        val tab = gson.fromJson(json, BrowserTab::class.java)

        assertEquals("tab-123", tab.id)
        assertEquals("My Page", tab.title)
        assertEquals("https://example.com", tab.url)
    }

    // ========================================================================
    // BrowserGotoRequest Tests
    // ========================================================================

    @Test
    fun `BrowserGotoRequest serializes correctly`() {
        val request = BrowserGotoRequest(url = "https://example.com")

        val json = gson.toJson(request)

        assertTrue(json.contains("https://example.com"))
    }

    // ========================================================================
    // BrowserNavigateRequest Tests
    // ========================================================================

    @Test
    fun `BrowserNavigateRequest serializes with optional tab_id`() {
        val request = BrowserNavigateRequest(tabId = "t1", url = "https://test.com")

        val json = gson.toJson(request)

        assertTrue(json.contains("t1") || json.contains("tab_id"))
        assertTrue(json.contains("https://test.com"))
    }

    @Test
    fun `BrowserNavigateRequest serializes without tab_id`() {
        val request = BrowserNavigateRequest(url = "https://test.com")

        val json = gson.toJson(request)

        assertTrue(json.contains("https://test.com"))
    }

    // ========================================================================
    // BrowserClickRequest Tests
    // ========================================================================

    @Test
    fun `BrowserClickRequest serializes selector correctly`() {
        val request = BrowserClickRequest(selector = "#submit-btn")

        val json = gson.toJson(request)

        assertTrue(json.contains("#submit-btn"))
    }

    // ========================================================================
    // BrowserTypeRequest Tests
    // ========================================================================

    @Test
    fun `BrowserTypeRequest serializes with all options`() {
        val request = BrowserTypeRequest(selector = "#input", text = "hello", clear = true)

        val json = gson.toJson(request)

        assertTrue(json.contains("#input"))
        assertTrue(json.contains("hello"))
        assertTrue(json.contains("true"))
    }

    @Test
    fun `BrowserTypeRequest has false as default for clear`() {
        val request = BrowserTypeRequest(selector = "#input", text = "hello")

        assertFalse(request.clear)
    }

    // ========================================================================
    // BrowserEvalRequest Tests
    // ========================================================================

    @Test
    fun `BrowserEvalRequest serializes expression correctly`() {
        val request = BrowserEvalRequest(expression = "document.title")

        val json = gson.toJson(request)

        assertTrue(json.contains("document.title"))
    }

    // ========================================================================
    // BrowserScreenshotRequest Tests
    // ========================================================================

    @Test
    fun `BrowserScreenshotRequest serializes with options`() {
        val request = BrowserScreenshotRequest(format = "png", quality = 80, fullPage = true)

        val json = gson.toJson(request)

        assertTrue(json.contains("png"))
        assertTrue(json.contains("80"))
        assertTrue(json.contains("true"))
    }

    @Test
    fun `BrowserScreenshotRequest has false as default for fullPage`() {
        val request = BrowserScreenshotRequest()

        assertFalse(request.fullPage)
    }

    // ========================================================================
    // BrowserScreenshotResponse Tests
    // ========================================================================

    @Test
    fun `BrowserScreenshotResponse deserializes with data`() {
        val json =
            """{
            "success": true,
            "format": "png",
            "data": "base64encodeddata",
            "size": 12345,
            "encoding": "base64"
        }"""

        val response = gson.fromJson(json, BrowserScreenshotResponse::class.java)

        assertTrue(response.success)
        assertEquals("png", response.format)
        assertEquals("base64encodeddata", response.data)
        assertEquals(12345, response.size)
    }

    // ========================================================================
    // BrowserConsoleMessage Tests
    // ========================================================================

    @Test
    fun `BrowserConsoleMessage deserializes correctly`() {
        val json = """{"level": "error", "text": "Something went wrong", "timestamp": "2024-01-01T12:00:00Z"}"""

        val message = gson.fromJson(json, BrowserConsoleMessage::class.java)

        assertEquals("error", message.level)
        assertEquals("Something went wrong", message.text)
        assertEquals("2024-01-01T12:00:00Z", message.timestamp)
    }

    // ========================================================================
    // BrowserNetworkEntry Tests
    // ========================================================================

    @Test
    fun `BrowserNetworkEntry deserializes correctly`() {
        val json =
            """{
            "method": "GET",
            "url": "https://api.example.com/data",
            "status": 200,
            "status_text": "OK",
            "timestamp": "2024-01-01T12:00:00Z"
        }"""

        val entry = gson.fromJson(json, BrowserNetworkEntry::class.java)

        assertEquals("GET", entry.method)
        assertEquals("https://api.example.com/data", entry.url)
        assertEquals(200, entry.status)
        assertEquals("OK", entry.statusText)
    }

    // ========================================================================
    // Response Tests
    // ========================================================================

    @Test
    fun `BrowserGotoResponse deserializes with tab`() {
        val json =
            """{
            "success": true,
            "tab": {"id": "new-tab", "title": "New Page", "url": "https://new.com"}
        }"""

        val response = gson.fromJson(json, BrowserGotoResponse::class.java)

        assertTrue(response.success)
        assertEquals("new-tab", response.tab?.id)
    }

    @Test
    fun `BrowserConsoleResponse deserializes with messages`() {
        val json =
            """{
            "success": true,
            "messages": [
                {"level": "log", "text": "Hello"},
                {"level": "error", "text": "Oops"}
            ],
            "count": 2
        }"""

        val response = gson.fromJson(json, BrowserConsoleResponse::class.java)

        assertTrue(response.success)
        assertEquals(2, response.count)
        assertEquals(2, response.messages?.size)
    }

    @Test
    fun `BrowserNetworkResponse deserializes with requests`() {
        val json =
            """{
            "success": true,
            "requests": [
                {"method": "GET", "url": "https://api.com", "timestamp": "2024-01-01T00:00:00Z"}
            ],
            "count": 1
        }"""

        val response = gson.fromJson(json, BrowserNetworkResponse::class.java)

        assertTrue(response.success)
        assertEquals(1, response.count)
        assertEquals("GET", response.requests?.first()?.method)
    }
}
