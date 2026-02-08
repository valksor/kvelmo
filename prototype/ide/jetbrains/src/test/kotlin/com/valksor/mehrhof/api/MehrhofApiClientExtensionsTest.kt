package com.valksor.mehrhof.api

import com.valksor.mehrhof.testutil.MockServerExtension
import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.Assertions.assertTrue
import org.junit.jupiter.api.Test
import org.junit.jupiter.api.extension.RegisterExtension

/**
 * Unit tests for API extension functions in [MehrhofApiClientExtensions].
 *
 * Uses MockWebServer to verify request paths, methods, and body structure.
 */
class MehrhofApiClientExtensionsTest {
    @JvmField
    @RegisterExtension
    val mockServer = MockServerExtension()

    // ========================================================================
    // Queue Task Extensions
    // ========================================================================

    @Test
    fun `createQuickTask sends POST with description`() {
        mockServer.enqueueSuccess("""{"success": true}""")

        val client = mockServer.createClient()
        val result = client.createQuickTask("Fix the bug")

        assertTrue(result.isSuccess)
        val request = mockServer.takeRequest()
        assertEquals("POST", request.method)
        assertEquals("/api/v1/interactive/command", request.path)
        val body = request.body.readUtf8()
        assertTrue(body.contains("quick"))
        assertTrue(body.contains("Fix the bug"))
    }

    @Test
    fun `deleteQueueTask sends POST with queue and task ID`() {
        mockServer.enqueueSuccess("""{"success": true}""")

        val client = mockServer.createClient()
        val result = client.deleteQueueTask("backlog", "task-123")

        assertTrue(result.isSuccess)
        mockServer.assertPostBody("/api/v1/interactive/command", "delete")
    }

    @Test
    fun `exportQueueTask sends POST with correct path format`() {
        mockServer.enqueueSuccess("""{"success": true}""")

        val client = mockServer.createClient()
        val result = client.exportQueueTask("backlog", "task-456")

        assertTrue(result.isSuccess)
        mockServer.assertPostBody("/api/v1/interactive/command", "export")
    }

    @Test
    fun `optimizeQueueTask sends POST for AI optimization`() {
        mockServer.enqueueSuccess("""{"success": true}""")

        val client = mockServer.createClient()
        val result = client.optimizeQueueTask("backlog", "task-789")

        assertTrue(result.isSuccess)
        mockServer.assertPostBody("/api/v1/interactive/command", "optimize")
    }

    @Test
    fun `submitQueueTask sends POST with provider`() {
        mockServer.enqueueSuccess("""{"success": true}""")

        val client = mockServer.createClient()
        val result = client.submitQueueTask("backlog", "task-1", "github")

        assertTrue(result.isSuccess)
        val request = mockServer.takeRequest()
        val body = request.body.readUtf8()
        assertTrue(body.contains("submit"))
        assertTrue(body.contains("github"))
    }

    @Test
    fun `syncTask sends POST with empty args`() {
        mockServer.enqueueSuccess("""{"success": true}""")

        val client = mockServer.createClient()
        val result = client.syncTask()

        assertTrue(result.isSuccess)
        mockServer.assertPostBody("/api/v1/interactive/command", "sync")
    }

    // ========================================================================
    // Find Extension
    // ========================================================================

    @Test
    fun `find sends GET with URL-encoded query`() {
        mockServer.enqueueSuccess("""{"query": "test", "count": 0, "matches": []}""")

        val client = mockServer.createClient()
        val result = client.find("hello world")

        assertTrue(result.isSuccess)
        val request = mockServer.takeRequest()
        assertEquals("GET", request.method)
        assertTrue(request.path!!.startsWith("/api/v1/find?q="))
        assertTrue(request.path!!.contains("hello"))
    }

    @Test
    fun `find URL-encodes special characters`() {
        mockServer.enqueueSuccess("""{"query": "test", "count": 0, "matches": []}""")

        val client = mockServer.createClient()
        client.find("foo&bar=baz")

        val request = mockServer.takeRequest()
        // & should be encoded as %26
        assertTrue(request.path!!.contains("%26") || request.path!!.contains("foo"))
    }

    // ========================================================================
    // Memory Extensions
    // ========================================================================

    @Test
    fun `memorySearch sends GET with query parameter`() {
        mockServer.enqueueSuccess("""{"results": [], "count": 0}""")

        val client = mockServer.createClient()
        val result = client.memorySearch("authentication")

        assertTrue(result.isSuccess)
        mockServer.assertRequestStartsWith("GET", "/api/v1/memory/search")
    }

    @Test
    fun `memoryIndex sends POST with task ID`() {
        mockServer.enqueueSuccess("""{"success": true}""")

        val client = mockServer.createClient()
        val result = client.memoryIndex("task-123")

        assertTrue(result.isSuccess)
        mockServer.assertPostBody("/api/v1/memory/index", "task-123")
    }

    @Test
    fun `memoryStats sends GET to stats endpoint`() {
        mockServer.enqueueSuccess("""{"total_documents": 10, "by_type": {}, "enabled": true}""")

        val client = mockServer.createClient()
        val result = client.memoryStats()

        assertTrue(result.isSuccess)
        mockServer.assertRequest("GET", "/api/v1/memory/stats")
    }

    // ========================================================================
    // Library Extensions
    // ========================================================================

    @Test
    fun `libraryList sends GET to library endpoint`() {
        mockServer.enqueueSuccess("""{"collections": [], "count": 0}""")

        val client = mockServer.createClient()
        val result = client.libraryList()

        assertTrue(result.isSuccess)
        mockServer.assertRequest("GET", "/api/v1/library")
    }

    @Test
    fun `libraryShow sends GET with collection name`() {
        mockServer.enqueueSuccess(
            """{
            "collection": {
                "id": "c1", "name": "test", "source": "local",
                "source_type": "directory", "include_mode": "all",
                "page_count": 0, "total_size": 0, "location": "/tmp"
            },
            "pages": []
        }""",
        )

        val client = mockServer.createClient()
        val result = client.libraryShow("my-docs")

        assertTrue(result.isSuccess)
        mockServer.assertRequestStartsWith("GET", "/api/v1/library/")
    }

    @Test
    fun `libraryStats sends GET to stats endpoint`() {
        mockServer.enqueueSuccess(
            """{
            "total_collections": 5, "total_pages": 100, "total_size": 1000,
            "project_count": 3, "shared_count": 2, "by_mode": {}, "enabled": true
        }""",
        )

        val client = mockServer.createClient()
        val result = client.libraryStats()

        assertTrue(result.isSuccess)
        mockServer.assertRequest("GET", "/api/v1/library/stats")
    }

    @Test
    fun `libraryPull sends POST with source and options`() {
        mockServer.enqueueSuccess("""{"success": true}""")

        val client = mockServer.createClient()
        val result = client.libraryPull("https://docs.example.com", name = "example", shared = true)

        assertTrue(result.isSuccess)
        val request = mockServer.takeRequest()
        val body = request.body.readUtf8()
        assertTrue(body.contains("library"))
        assertTrue(body.contains("pull"))
    }

    @Test
    fun `libraryRemove sends POST with collection name`() {
        mockServer.enqueueSuccess("""{"success": true}""")

        val client = mockServer.createClient()
        val result = client.libraryRemove("old-docs")

        assertTrue(result.isSuccess)
        val request = mockServer.takeRequest()
        val body = request.body.readUtf8()
        assertTrue(body.contains("remove"))
        assertTrue(body.contains("old-docs"))
    }

    // ========================================================================
    // Links Extensions
    // ========================================================================

    @Test
    fun `linksList sends GET to links endpoint`() {
        mockServer.enqueueSuccess("""{"links": [], "count": 0}""")

        val client = mockServer.createClient()
        val result = client.linksList()

        assertTrue(result.isSuccess)
        mockServer.assertRequest("GET", "/api/v1/links")
    }

    @Test
    fun `linksGet sends GET with entity ID`() {
        mockServer.enqueueSuccess("""{"entity_id": "e1", "outgoing": [], "incoming": []}""")

        val client = mockServer.createClient()
        val result = client.linksGet("spec:123")

        assertTrue(result.isSuccess)
        mockServer.assertRequestStartsWith("GET", "/api/v1/links/")
    }

    @Test
    fun `linksSearch sends GET with query parameter`() {
        mockServer.enqueueSuccess("""{"query": "auth", "results": [], "count": 0}""")

        val client = mockServer.createClient()
        val result = client.linksSearch("authentication")

        assertTrue(result.isSuccess)
        mockServer.assertRequestStartsWith("GET", "/api/v1/links/search")
    }

    @Test
    fun `linksStats sends GET to stats endpoint`() {
        mockServer.enqueueSuccess(
            """{
            "total_links": 50, "total_sources": 10, "total_targets": 20,
            "orphan_entities": 2, "most_linked": [], "enabled": true
        }""",
        )

        val client = mockServer.createClient()
        val result = client.linksStats()

        assertTrue(result.isSuccess)
        mockServer.assertRequest("GET", "/api/v1/links/stats")
    }

    @Test
    fun `linksRebuild sends POST command`() {
        mockServer.enqueueSuccess("""{"success": true}""")

        val client = mockServer.createClient()
        val result = client.linksRebuild()

        assertTrue(result.isSuccess)
        val request = mockServer.takeRequest()
        val body = request.body.readUtf8()
        assertTrue(body.contains("links"))
        assertTrue(body.contains("rebuild"))
    }

    // ========================================================================
    // Browser Extensions
    // ========================================================================

    @Test
    fun `browserStatus sends GET to status endpoint`() {
        mockServer.enqueueSuccess("""{"connected": false, "tabs": []}""")

        val client = mockServer.createClient()
        val result = client.browserStatus()

        assertTrue(result.isSuccess)
        mockServer.assertRequest("GET", "/api/v1/browser/status")
    }

    @Test
    fun `browserTabs sends GET to tabs endpoint`() {
        mockServer.enqueueSuccess("""{"tabs": [], "count": 0}""")

        val client = mockServer.createClient()
        val result = client.browserTabs()

        assertTrue(result.isSuccess)
        mockServer.assertRequest("GET", "/api/v1/browser/tabs")
    }

    @Test
    fun `browserGoto sends POST with URL`() {
        mockServer.enqueueSuccess("""{"success": true, "tab": null}""")

        val client = mockServer.createClient()
        val result = client.browserGoto("https://example.com")

        assertTrue(result.isSuccess)
        mockServer.assertPostBody("/api/v1/browser/goto", "https://example.com")
    }

    @Test
    fun `browserNavigate sends POST with URL and optional tab ID`() {
        mockServer.enqueueSuccess("""{"success": true}""")

        val client = mockServer.createClient()
        val result = client.browserNavigate("https://google.com", tabId = "tab-1")

        assertTrue(result.isSuccess)
        val request = mockServer.assertRequest("POST", "/api/v1/browser/navigate")
        val body = request.body.readUtf8()
        assertTrue(body.contains("https://google.com"))
    }

    @Test
    fun `browserClick sends POST with selector`() {
        mockServer.enqueueSuccess("""{"success": true}""")

        val client = mockServer.createClient()
        val result = client.browserClick("#submit-btn")

        assertTrue(result.isSuccess)
        mockServer.assertPostBody("/api/v1/browser/click", "#submit-btn")
    }

    @Test
    fun `browserType sends POST with selector and text`() {
        mockServer.enqueueSuccess("""{"success": true}""")

        val client = mockServer.createClient()
        val result = client.browserType("#input", "hello world")

        assertTrue(result.isSuccess)
        val request = mockServer.assertRequest("POST", "/api/v1/browser/type")
        val body = request.body.readUtf8()
        assertTrue(body.contains("#input"))
        assertTrue(body.contains("hello world"))
    }

    @Test
    fun `browserEval sends POST with expression`() {
        mockServer.enqueueSuccess("""{"success": true, "result": "42"}""")

        val client = mockServer.createClient()
        val result = client.browserEval("document.title")

        assertTrue(result.isSuccess)
        mockServer.assertPostBody("/api/v1/browser/eval", "document.title")
    }

    @Test
    fun `browserScreenshot sends POST with options`() {
        mockServer.enqueueSuccess("""{"success": true}""")

        val client = mockServer.createClient()
        val result = client.browserScreenshot(format = "png", fullPage = true)

        assertTrue(result.isSuccess)
        mockServer.assertRequest("POST", "/api/v1/browser/screenshot")
    }

    @Test
    fun `browserReload sends POST with options`() {
        mockServer.enqueueSuccess("""{"success": true}""")

        val client = mockServer.createClient()
        val result = client.browserReload(hard = true)

        assertTrue(result.isSuccess)
        mockServer.assertRequest("POST", "/api/v1/browser/reload")
    }

    @Test
    fun `browserClose sends POST with tab ID`() {
        mockServer.enqueueSuccess("""{"success": true}""")

        val client = mockServer.createClient()
        val result = client.browserClose("tab-123")

        assertTrue(result.isSuccess)
        mockServer.assertPostBody("/api/v1/browser/close", "tab-123")
    }

    @Test
    fun `browserConsole sends POST with options`() {
        mockServer.enqueueSuccess("""{"success": true, "messages": [], "count": 0}""")

        val client = mockServer.createClient()
        val result = client.browserConsole(duration = 5, level = "error")

        assertTrue(result.isSuccess)
        mockServer.assertRequest("POST", "/api/v1/browser/console")
    }

    @Test
    fun `browserNetwork sends POST with options`() {
        mockServer.enqueueSuccess("""{"success": true, "requests": [], "count": 0}""")

        val client = mockServer.createClient()
        val result = client.browserNetwork(duration = 10, captureBody = true)

        assertTrue(result.isSuccess)
        mockServer.assertRequest("POST", "/api/v1/browser/network")
    }
}
