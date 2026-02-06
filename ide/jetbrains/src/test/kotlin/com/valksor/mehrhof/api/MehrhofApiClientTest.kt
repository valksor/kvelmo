package com.valksor.mehrhof.api

import com.valksor.mehrhof.api.models.*
import okhttp3.mockwebserver.MockResponse
import okhttp3.mockwebserver.MockWebServer
import org.junit.jupiter.api.*
import org.junit.jupiter.api.Assertions.*

/**
 * Unit tests for MehrhofApiClient.
 */
class MehrhofApiClientTest {
    private lateinit var mockServer: MockWebServer
    private lateinit var client: MehrhofApiClient

    @BeforeEach
    fun setUp() {
        mockServer = MockWebServer()
        mockServer.start()
        client = MehrhofApiClient(mockServer.url("/").toString().trimEnd('/'))
    }

    @AfterEach
    fun tearDown() {
        mockServer.shutdown()
    }

    // ========================================================================
    // Status & Task Endpoint Tests
    // ========================================================================

    @Test
    fun `getStatus returns StatusResponse on success`() {
        mockServer.enqueue(
            MockResponse()
                .setResponseCode(200)
                .setHeader("Content-Type", "application/json")
                .setBody("""{"mode":"local","running":true,"port":3000,"state":"idle"}""")
        )

        val result = client.getStatus()

        assertTrue(result.isSuccess)
        val status = result.getOrNull()!!
        assertEquals("local", status.mode)
        assertTrue(status.running)
        assertEquals(3000, status.port)
        assertEquals("idle", status.state)

        val request = mockServer.takeRequest()
        assertEquals("GET", request.method)
        assertEquals("/api/v1/status", request.path)
    }

    @Test
    fun `getTask returns TaskResponse with active task`() {
        mockServer.enqueue(
            MockResponse()
                .setResponseCode(200)
                .setHeader("Content-Type", "application/json")
                .setBody(
                    """
                    {
                        "active": true,
                        "task": {
                            "id": "task-123",
                            "state": "planning",
                            "ref": "github:42",
                            "branch": "feature/test"
                        },
                        "work": {
                            "title": "Fix login bug",
                            "external_key": "42"
                        }
                    }
                    """.trimIndent()
                )
        )

        val result = client.getTask()

        assertTrue(result.isSuccess)
        val task = result.getOrNull()!!
        assertTrue(task.active)
        assertEquals("task-123", task.task?.id)
        assertEquals("planning", task.task?.state)
        assertEquals("Fix login bug", task.work?.title)
    }

    @Test
    fun `getTask returns TaskResponse with no active task`() {
        mockServer.enqueue(
            MockResponse()
                .setResponseCode(200)
                .setHeader("Content-Type", "application/json")
                .setBody("""{"active": false}""")
        )

        val result = client.getTask()

        assertTrue(result.isSuccess)
        val task = result.getOrNull()!!
        assertFalse(task.active)
        assertNull(task.task)
    }

    @Test
    fun `getTasks returns list of tasks`() {
        mockServer.enqueue(
            MockResponse()
                .setResponseCode(200)
                .setHeader("Content-Type", "application/json")
                .setBody(
                    """
                    {
                        "tasks": [
                            {"id": "task-1", "title": "Task 1", "state": "done"},
                            {"id": "task-2", "title": "Task 2", "state": "planning"}
                        ],
                        "count": 2
                    }
                    """.trimIndent()
                )
        )

        val result = client.getTasks()

        assertTrue(result.isSuccess)
        val response = result.getOrNull()!!
        assertEquals(2, response.count)
        assertEquals(2, response.tasks.size)
        assertEquals("task-1", response.tasks[0].id)
        assertEquals("done", response.tasks[0].state)
    }

    @Test
    fun `getGuide returns guidance response`() {
        mockServer.enqueue(
            MockResponse()
                .setResponseCode(200)
                .setHeader("Content-Type", "application/json")
                .setBody(
                    """
                    {
                        "has_task": true,
                        "task_id": "task-123",
                        "title": "Fix bug",
                        "state": "planning",
                        "specifications": 2,
                        "next_actions": [
                            {"command": "plan", "description": "Run planning", "endpoint": "/api/v1/workflow/plan"}
                        ]
                    }
                    """.trimIndent()
                )
        )

        val result = client.getGuide()

        assertTrue(result.isSuccess)
        val guide = result.getOrNull()!!
        assertTrue(guide.hasTask)
        assertEquals("task-123", guide.taskId)
        assertEquals(2, guide.specifications)
        assertEquals(1, guide.nextActions.size)
        assertEquals("plan", guide.nextActions[0].command)
    }

    // ========================================================================
    // Workflow Endpoint Tests
    // ========================================================================

    @Test
    fun `plan sends POST request and returns WorkflowResponse`() {
        mockServer.enqueue(
            MockResponse()
                .setResponseCode(200)
                .setHeader("Content-Type", "application/json")
                .setBody("""{"success": true, "state": "planning"}""")
        )

        val result = client.plan()

        assertTrue(result.isSuccess)
        val response = result.getOrNull()!!
        assertTrue(response.success)
        assertEquals("planning", response.state)

        val request = mockServer.takeRequest()
        assertEquals("POST", request.method)
        assertEquals("/api/v1/workflow/plan", request.path)
    }

    @Test
    fun `plan with agent sends agent in request body`() {
        mockServer.enqueue(
            MockResponse()
                .setResponseCode(200)
                .setHeader("Content-Type", "application/json")
                .setBody("""{"success": true, "state": "planning"}""")
        )

        client.plan(agent = "claude")

        val request = mockServer.takeRequest()
        assertTrue(request.body.readUtf8().contains("\"agent\":\"claude\""))
    }

    @Test
    fun `implement sends POST request`() {
        mockServer.enqueue(
            MockResponse()
                .setResponseCode(200)
                .setHeader("Content-Type", "application/json")
                .setBody("""{"success": true, "state": "implementing"}""")
        )

        val result = client.implement()

        assertTrue(result.isSuccess)

        val request = mockServer.takeRequest()
        assertEquals("POST", request.method)
        assertEquals("/api/v1/workflow/implement", request.path)
    }

    @Test
    fun `review sends POST request`() {
        mockServer.enqueue(
            MockResponse()
                .setResponseCode(200)
                .setHeader("Content-Type", "application/json")
                .setBody("""{"success": true, "state": "reviewing"}""")
        )

        val result = client.review()

        assertTrue(result.isSuccess)

        val request = mockServer.takeRequest()
        assertEquals("/api/v1/workflow/review", request.path)
    }

    @Test
    fun `finish sends POST request with options`() {
        mockServer.enqueue(
            MockResponse()
                .setResponseCode(200)
                .setHeader("Content-Type", "application/json")
                .setBody("""{"success": true, "state": "done"}""")
        )

        val result = client.finish(FinishRequest(squashMerge = true, deleteBranch = false))

        assertTrue(result.isSuccess)

        val request = mockServer.takeRequest()
        val body = request.body.readUtf8()
        assertTrue(body.contains("\"squash_merge\":true"))
        assertTrue(body.contains("\"delete_branch\":false"))
    }

    @Test
    fun `undo sends POST request`() {
        mockServer.enqueue(
            MockResponse()
                .setResponseCode(200)
                .setHeader("Content-Type", "application/json")
                .setBody("""{"success": true, "message": "Reverted to checkpoint"}""")
        )

        val result = client.undo()

        assertTrue(result.isSuccess)

        val request = mockServer.takeRequest()
        assertEquals("POST", request.method)
        assertEquals("/api/v1/workflow/undo", request.path)
    }

    @Test
    fun `redo sends POST request`() {
        mockServer.enqueue(
            MockResponse()
                .setResponseCode(200)
                .setHeader("Content-Type", "application/json")
                .setBody("""{"success": true, "message": "Restored checkpoint"}""")
        )

        val result = client.redo()

        assertTrue(result.isSuccess)

        val request = mockServer.takeRequest()
        assertEquals("/api/v1/workflow/redo", request.path)
    }

    @Test
    fun `answer sends answer in request body`() {
        mockServer.enqueue(
            MockResponse()
                .setResponseCode(200)
                .setHeader("Content-Type", "application/json")
                .setBody("""{"success": true}""")
        )

        client.answer("Yes, proceed")

        val request = mockServer.takeRequest()
        assertTrue(request.body.readUtf8().contains("\"answer\":\"Yes, proceed\""))
    }

    @Test
    fun `startTask sends ref and content`() {
        mockServer.enqueue(
            MockResponse()
                .setResponseCode(200)
                .setHeader("Content-Type", "application/json")
                .setBody("""{"success": true, "state": "idle"}""")
        )

        client.startTask(ref = "github:123", content = "Fix the bug")

        val request = mockServer.takeRequest()
        val body = request.body.readUtf8()
        assertTrue(body.contains("\"ref\":\"github:123\""))
        assertTrue(body.contains("\"content\":\"Fix the bug\""))
    }

    // ========================================================================
    // Error Handling Tests
    // ========================================================================

    @Test
    fun `returns failure on HTTP 400 error`() {
        mockServer.enqueue(
            MockResponse()
                .setResponseCode(400)
                .setHeader("Content-Type", "application/json")
                .setBody("""{"error": "Invalid request"}""")
        )

        val result = client.getStatus()

        assertTrue(result.isFailure)
        val error = result.exceptionOrNull() as ApiException
        assertEquals(400, error.statusCode)
        assertEquals("Invalid request", error.message)
    }

    @Test
    fun `returns failure on HTTP 500 error`() {
        mockServer.enqueue(
            MockResponse()
                .setResponseCode(500)
                .setHeader("Content-Type", "application/json")
                .setBody("""{"error": "Internal server error"}""")
        )

        val result = client.plan()

        assertTrue(result.isFailure)
        val error = result.exceptionOrNull() as ApiException
        assertEquals(500, error.statusCode)
    }

    @Test
    fun `returns failure on HTTP error without error field`() {
        mockServer.enqueue(
            MockResponse()
                .setResponseCode(404)
                .setBody("Not Found")
        )

        val result = client.getStatus()

        assertTrue(result.isFailure)
        val error = result.exceptionOrNull() as ApiException
        assertEquals(404, error.statusCode)
        assertTrue(error.message.contains("404"))
    }

    @Test
    fun `returns failure on empty response body`() {
        mockServer.enqueue(
            MockResponse()
                .setResponseCode(200)
                .setBody("")
        )

        val result = client.getStatus()

        assertTrue(result.isFailure)
        val error = result.exceptionOrNull() as ApiException
        assertTrue(error.message.contains("Empty response"))
    }

    @Test
    fun `returns failure on invalid JSON`() {
        mockServer.enqueue(
            MockResponse()
                .setResponseCode(200)
                .setHeader("Content-Type", "application/json")
                .setBody("not valid json")
        )

        val result = client.getStatus()

        assertTrue(result.isFailure)
        val error = result.exceptionOrNull() as ApiException
        assertTrue(error.message.contains("Failed to parse"))
    }

    @Test
    fun `returns failure on network error`() {
        mockServer.shutdown() // Force connection failure

        val result = client.getStatus()

        assertTrue(result.isFailure)
        val error = result.exceptionOrNull() as ApiException
        assertTrue(error.message.contains("Network error") || error.message.contains("Unexpected error"))
    }

    // ========================================================================
    // isReachable Tests
    // ========================================================================

    @Test
    fun `isReachable returns true when server responds`() {
        mockServer.enqueue(
            MockResponse()
                .setResponseCode(200)
                .setBody("OK")
        )

        assertTrue(client.isReachable())

        val request = mockServer.takeRequest()
        assertEquals("/health", request.path)
    }

    @Test
    fun `isReachable returns false on server error`() {
        mockServer.enqueue(
            MockResponse()
                .setResponseCode(500)
        )

        assertFalse(client.isReachable())
    }

    @Test
    fun `isReachable returns false when server is down`() {
        mockServer.shutdown()

        assertFalse(client.isReachable())
    }

    // ========================================================================
    // Cost Endpoint Tests
    // ========================================================================

    @Test
    fun `getTaskCosts returns cost information`() {
        mockServer.enqueue(
            MockResponse()
                .setResponseCode(200)
                .setHeader("Content-Type", "application/json")
                .setBody(
                    """
                    {
                        "task_id": "task-123",
                        "title": "Fix bug",
                        "total_tokens": 1000,
                        "input_tokens": 800,
                        "output_tokens": 200,
                        "cached_tokens": 100,
                        "total_cost_usd": 0.05
                    }
                    """.trimIndent()
                )
        )

        val result = client.getTaskCosts("task-123")

        assertTrue(result.isSuccess)
        val costs = result.getOrNull()!!
        assertEquals("task-123", costs.taskId)
        assertEquals(1000, costs.totalTokens)
        assertEquals(0.05, costs.totalCostUsd)
    }

    @Test
    fun `getAllCosts returns all costs with grand total`() {
        mockServer.enqueue(
            MockResponse()
                .setResponseCode(200)
                .setHeader("Content-Type", "application/json")
                .setBody(
                    """
                    {
                        "tasks": [],
                        "grand_total": {
                            "input_tokens": 5000,
                            "output_tokens": 1000,
                            "total_tokens": 6000,
                            "cached_tokens": 500,
                            "cost_usd": 0.25
                        }
                    }
                    """.trimIndent()
                )
        )

        val result = client.getAllCosts()

        assertTrue(result.isSuccess)
        val costs = result.getOrNull()!!
        assertEquals(6000, costs.grandTotal.totalTokens)
        assertEquals(0.25, costs.grandTotal.costUsd)
    }

    // ========================================================================
    // Agent & Provider Tests
    // ========================================================================

    @Test
    fun `getAgents returns agent list`() {
        mockServer.enqueue(
            MockResponse()
                .setResponseCode(200)
                .setHeader("Content-Type", "application/json")
                .setBody(
                    """
                    {
                        "agents": [
                            {"name": "claude", "type": "claude", "available": true}
                        ],
                        "count": 1
                    }
                    """.trimIndent()
                )
        )

        val result = client.getAgents()

        assertTrue(result.isSuccess)
        val agents = result.getOrNull()!!
        assertEquals(1, agents.count)
        assertEquals("claude", agents.agents[0].name)
    }

    @Test
    fun `getProviders returns provider list`() {
        mockServer.enqueue(
            MockResponse()
                .setResponseCode(200)
                .setHeader("Content-Type", "application/json")
                .setBody(
                    """
                    {
                        "providers": [
                            {"scheme": "github", "name": "GitHub", "description": "GitHub issues"}
                        ],
                        "count": 1
                    }
                    """.trimIndent()
                )
        )

        val result = client.getProviders()

        assertTrue(result.isSuccess)
        val providers = result.getOrNull()!!
        assertEquals(1, providers.count)
        assertEquals("github", providers.providers[0].scheme)
    }

    // ========================================================================
    // Browser Endpoint Tests
    // ========================================================================

    @Test
    fun `browserStatus returns status response`() {
        mockServer.enqueue(
            MockResponse()
                .setResponseCode(200)
                .setHeader("Content-Type", "application/json")
                .setBody(
                    """
                    {
                        "connected": true,
                        "host": "127.0.0.1",
                        "port": 9222,
                        "tabs": [
                            {"id": "tab-1", "title": "Google", "url": "https://google.com"}
                        ]
                    }
                    """.trimIndent()
                )
        )

        val result = client.browserStatus()

        assertTrue(result.isSuccess)
        val status = result.getOrNull()!!
        assertTrue(status.connected)
        assertEquals("127.0.0.1", status.host)
        assertEquals(9222, status.port)
        assertEquals(1, status.tabs!!.size)
        assertEquals("tab-1", status.tabs!![0].id)

        val request = mockServer.takeRequest()
        assertEquals("GET", request.method)
        assertEquals("/api/v1/browser/status", request.path)
    }

    @Test
    fun `browserTabs returns tab list`() {
        mockServer.enqueue(
            MockResponse()
                .setResponseCode(200)
                .setHeader("Content-Type", "application/json")
                .setBody(
                    """
                    {
                        "tabs": [
                            {"id": "tab-1", "title": "Page A", "url": "https://a.com"},
                            {"id": "tab-2", "title": "Page B", "url": "https://b.com"}
                        ],
                        "count": 2
                    }
                    """.trimIndent()
                )
        )

        val result = client.browserTabs()

        assertTrue(result.isSuccess)
        val tabs = result.getOrNull()!!
        assertEquals(2, tabs.count)
        assertEquals("tab-1", tabs.tabs[0].id)
        assertEquals("Page B", tabs.tabs[1].title)

        val request = mockServer.takeRequest()
        assertEquals("GET", request.method)
        assertEquals("/api/v1/browser/tabs", request.path)
    }

    @Test
    fun `browserGoto sends URL and returns tab`() {
        mockServer.enqueue(
            MockResponse()
                .setResponseCode(200)
                .setHeader("Content-Type", "application/json")
                .setBody(
                    """
                    {
                        "success": true,
                        "tab": {"id": "tab-new", "title": "Example", "url": "https://example.com"}
                    }
                    """.trimIndent()
                )
        )

        val result = client.browserGoto("https://example.com")

        assertTrue(result.isSuccess)
        val response = result.getOrNull()!!
        assertTrue(response.success)
        assertEquals("tab-new", response.tab!!.id)
        assertEquals("https://example.com", response.tab!!.url)

        val request = mockServer.takeRequest()
        assertEquals("POST", request.method)
        assertEquals("/api/v1/browser/goto", request.path)
        assertTrue(request.body.readUtf8().contains("\"url\":\"https://example.com\""))
    }

    @Test
    fun `browserNavigate sends URL with optional tab ID`() {
        mockServer.enqueue(
            MockResponse()
                .setResponseCode(200)
                .setHeader("Content-Type", "application/json")
                .setBody("""{"success": true, "message": "Navigated"}""")
        )

        val result = client.browserNavigate("https://test.com", tabId = "tab-1")

        assertTrue(result.isSuccess)
        val response = result.getOrNull()!!
        assertTrue(response.success)

        val request = mockServer.takeRequest()
        assertEquals("POST", request.method)
        assertEquals("/api/v1/browser/navigate", request.path)
        val body = request.body.readUtf8()
        assertTrue(body.contains("\"url\":\"https://test.com\""))
        assertTrue(body.contains("\"tab_id\":\"tab-1\""))
    }

    @Test
    fun `browserClick sends selector`() {
        mockServer.enqueue(
            MockResponse()
                .setResponseCode(200)
                .setHeader("Content-Type", "application/json")
                .setBody("""{"success": true, "selector": "#submit-btn"}""")
        )

        val result = client.browserClick("#submit-btn")

        assertTrue(result.isSuccess)
        val response = result.getOrNull()!!
        assertTrue(response.success)
        assertEquals("#submit-btn", response.selector)

        val request = mockServer.takeRequest()
        assertEquals("POST", request.method)
        assertEquals("/api/v1/browser/click", request.path)
    }

    @Test
    fun `browserType sends selector and text`() {
        mockServer.enqueue(
            MockResponse()
                .setResponseCode(200)
                .setHeader("Content-Type", "application/json")
                .setBody("""{"success": true, "selector": "#email"}""")
        )

        val result = client.browserType(selector = "#email", text = "user@test.com", clear = true)

        assertTrue(result.isSuccess)

        val request = mockServer.takeRequest()
        assertEquals("POST", request.method)
        assertEquals("/api/v1/browser/type", request.path)
        val body = request.body.readUtf8()
        assertTrue(body.contains("\"selector\":\"#email\""))
        assertTrue(body.contains("\"text\":\"user@test.com\""))
        assertTrue(body.contains("\"clear\":true"))
    }

    @Test
    fun `browserEval sends expression and returns result`() {
        mockServer.enqueue(
            MockResponse()
                .setResponseCode(200)
                .setHeader("Content-Type", "application/json")
                .setBody("""{"success": true, "result": "Hello World"}""")
        )

        val result = client.browserEval("document.title")

        assertTrue(result.isSuccess)
        val response = result.getOrNull()!!
        assertTrue(response.success)
        assertEquals("Hello World", response.result)

        val request = mockServer.takeRequest()
        assertEquals("POST", request.method)
        assertEquals("/api/v1/browser/eval", request.path)
        assertTrue(request.body.readUtf8().contains("\"expression\":\"document.title\""))
    }

    @Test
    fun `browserScreenshot returns screenshot data`() {
        mockServer.enqueue(
            MockResponse()
                .setResponseCode(200)
                .setHeader("Content-Type", "application/json")
                .setBody(
                    """
                    {
                        "success": true,
                        "format": "png",
                        "data": "iVBORw0KGgoAAAANS...",
                        "size": 4096,
                        "encoding": "base64"
                    }
                    """.trimIndent()
                )
        )

        val result = client.browserScreenshot(format = "png", fullPage = true)

        assertTrue(result.isSuccess)
        val response = result.getOrNull()!!
        assertTrue(response.success)
        assertEquals("png", response.format)
        assertEquals("base64", response.encoding)
        assertEquals(4096, response.size)

        val request = mockServer.takeRequest()
        assertEquals("POST", request.method)
        assertEquals("/api/v1/browser/screenshot", request.path)
        val body = request.body.readUtf8()
        assertTrue(body.contains("\"full_page\":true"))
    }

    @Test
    fun `browserConsole returns console messages`() {
        mockServer.enqueue(
            MockResponse()
                .setResponseCode(200)
                .setHeader("Content-Type", "application/json")
                .setBody(
                    """
                    {
                        "success": true,
                        "messages": [
                            {"level": "error", "text": "Uncaught TypeError", "timestamp": "2026-01-15T10:00:00Z"},
                            {"level": "warn", "text": "Deprecated API", "timestamp": "2026-01-15T10:00:01Z"}
                        ],
                        "count": 2
                    }
                    """.trimIndent()
                )
        )

        val result = client.browserConsole(level = "error")

        assertTrue(result.isSuccess)
        val response = result.getOrNull()!!
        assertTrue(response.success)
        assertEquals(2, response.count)
        assertEquals("error", response.messages!![0].level)
        assertEquals("Uncaught TypeError", response.messages!![0].text)

        val request = mockServer.takeRequest()
        assertEquals("POST", request.method)
        assertEquals("/api/v1/browser/console", request.path)
    }

    @Test
    fun `browserNetwork returns network entries`() {
        mockServer.enqueue(
            MockResponse()
                .setResponseCode(200)
                .setHeader("Content-Type", "application/json")
                .setBody(
                    """
                    {
                        "success": true,
                        "requests": [
                            {
                                "method": "GET",
                                "url": "https://api.example.com/data",
                                "status": 200,
                                "status_text": "OK",
                                "timestamp": "2026-01-15T10:00:00Z"
                            }
                        ],
                        "count": 1
                    }
                    """.trimIndent()
                )
        )

        val result = client.browserNetwork(captureBody = true)

        assertTrue(result.isSuccess)
        val response = result.getOrNull()!!
        assertTrue(response.success)
        assertEquals(1, response.count)
        assertEquals("GET", response.requests!![0].method)
        assertEquals(200, response.requests!![0].status)

        val request = mockServer.takeRequest()
        assertEquals("POST", request.method)
        assertEquals("/api/v1/browser/network", request.path)
        assertTrue(request.body.readUtf8().contains("\"capture_body\":true"))
    }

    @Test
    fun `browserReload sends reload request`() {
        mockServer.enqueue(
            MockResponse()
                .setResponseCode(200)
                .setHeader("Content-Type", "application/json")
                .setBody("""{"success": true, "message": "Page reloaded"}""")
        )

        val result = client.browserReload(tabId = "tab-1", hard = true)

        assertTrue(result.isSuccess)
        val response = result.getOrNull()!!
        assertTrue(response.success)

        val request = mockServer.takeRequest()
        assertEquals("POST", request.method)
        assertEquals("/api/v1/browser/reload", request.path)
        val body = request.body.readUtf8()
        assertTrue(body.contains("\"hard\":true"))
        assertTrue(body.contains("\"tab_id\":\"tab-1\""))
    }

    @Test
    fun `browserClose sends close request for tab`() {
        mockServer.enqueue(
            MockResponse()
                .setResponseCode(200)
                .setHeader("Content-Type", "application/json")
                .setBody("""{"success": true, "message": "Tab closed"}""")
        )

        val result = client.browserClose("tab-42")

        assertTrue(result.isSuccess)

        val request = mockServer.takeRequest()
        assertEquals("POST", request.method)
        assertEquals("/api/v1/browser/close", request.path)
        assertTrue(request.body.readUtf8().contains("\"tab_id\":\"tab-42\""))
    }

    // ========================================================================
    // Memory Endpoint Tests
    // ========================================================================

    @Test
    fun `memorySearch returns search results`() {
        mockServer.enqueue(
            MockResponse()
                .setResponseCode(200)
                .setHeader("Content-Type", "application/json")
                .setBody(
                    """
                    {
                        "results": [
                            {
                                "task_id": "task-old",
                                "type": "specification",
                                "score": 0.92,
                                "content": "Implemented caching layer for API responses"
                            }
                        ],
                        "count": 1
                    }
                    """.trimIndent()
                )
        )

        val result = client.memorySearch("caching")

        assertTrue(result.isSuccess)
        val response = result.getOrNull()!!
        assertEquals(1, response.count)
        assertEquals("task-old", response.results[0].taskId)
        assertEquals(0.92, response.results[0].score)

        val request = mockServer.takeRequest()
        assertEquals("GET", request.method)
        assertTrue(request.path!!.startsWith("/api/v1/memory/search"))
        assertTrue(request.path!!.contains("q=caching"))
    }

    @Test
    fun `memoryIndex sends task ID for indexing`() {
        mockServer.enqueue(
            MockResponse()
                .setResponseCode(200)
                .setHeader("Content-Type", "application/json")
                .setBody(
                    """
                    {
                        "success": true,
                        "message": "Task indexed successfully",
                        "task_id": "task-456"
                    }
                    """.trimIndent()
                )
        )

        val result = client.memoryIndex("task-456")

        assertTrue(result.isSuccess)
        val response = result.getOrNull()!!
        assertTrue(response.success)
        assertEquals("task-456", response.taskId)

        val request = mockServer.takeRequest()
        assertEquals("POST", request.method)
        assertEquals("/api/v1/memory/index", request.path)
        assertTrue(request.body.readUtf8().contains("\"task_id\":\"task-456\""))
    }

    @Test
    fun `memoryStats returns memory statistics`() {
        mockServer.enqueue(
            MockResponse()
                .setResponseCode(200)
                .setHeader("Content-Type", "application/json")
                .setBody(
                    """
                    {
                        "total_documents": 150,
                        "by_type": {"specification": 80, "review": 40, "note": 30},
                        "enabled": true
                    }
                    """.trimIndent()
                )
        )

        val result = client.memoryStats()

        assertTrue(result.isSuccess)
        val stats = result.getOrNull()!!
        assertEquals(150, stats.totalDocuments)
        assertTrue(stats.enabled)
        assertEquals(80, stats.byType["specification"])

        val request = mockServer.takeRequest()
        assertEquals("GET", request.method)
        assertEquals("/api/v1/memory/stats", request.path)
    }

    // ========================================================================
    // Library Endpoint Tests
    // ========================================================================

    @Test
    fun `libraryList returns collection list`() {
        mockServer.enqueue(
            MockResponse()
                .setResponseCode(200)
                .setHeader("Content-Type", "application/json")
                .setBody(
                    """
                    {
                        "collections": [
                            {
                                "id": "col-1",
                                "name": "react-docs",
                                "source": "https://react.dev/docs",
                                "source_type": "web",
                                "include_mode": "full",
                                "page_count": 42,
                                "total_size": 1048576,
                                "location": "project"
                            }
                        ],
                        "count": 1
                    }
                    """.trimIndent()
                )
        )

        val result = client.libraryList()

        assertTrue(result.isSuccess)
        val response = result.getOrNull()!!
        assertEquals(1, response.count)
        assertEquals("react-docs", response.collections[0].name)
        assertEquals(42, response.collections[0].pageCount)

        val request = mockServer.takeRequest()
        assertEquals("GET", request.method)
        assertEquals("/api/v1/library", request.path)
    }

    @Test
    fun `libraryShow returns collection details with pages`() {
        mockServer.enqueue(
            MockResponse()
                .setResponseCode(200)
                .setHeader("Content-Type", "application/json")
                .setBody(
                    """
                    {
                        "collection": {
                            "id": "col-1",
                            "name": "react-docs",
                            "source": "https://react.dev/docs",
                            "source_type": "web",
                            "include_mode": "full",
                            "page_count": 3,
                            "total_size": 32768,
                            "location": "project"
                        },
                        "pages": ["getting-started.md", "hooks.md", "components.md"]
                    }
                    """.trimIndent()
                )
        )

        val result = client.libraryShow("react-docs")

        assertTrue(result.isSuccess)
        val response = result.getOrNull()!!
        assertEquals("react-docs", response.collection.name)
        assertEquals(3, response.pages.size)
        assertEquals("hooks.md", response.pages[1])

        val request = mockServer.takeRequest()
        assertEquals("GET", request.method)
        assertTrue(request.path!!.startsWith("/api/v1/library/"))
    }

    @Test
    fun `libraryStats returns library statistics`() {
        mockServer.enqueue(
            MockResponse()
                .setResponseCode(200)
                .setHeader("Content-Type", "application/json")
                .setBody(
                    """
                    {
                        "total_collections": 5,
                        "total_pages": 200,
                        "total_size": 5242880,
                        "project_count": 3,
                        "shared_count": 2,
                        "by_mode": {"full": 3, "summary": 2},
                        "enabled": true
                    }
                    """.trimIndent()
                )
        )

        val result = client.libraryStats()

        assertTrue(result.isSuccess)
        val stats = result.getOrNull()!!
        assertEquals(5, stats.totalCollections)
        assertEquals(200, stats.totalPages)
        assertEquals(3, stats.projectCount)
        assertTrue(stats.enabled)

        val request = mockServer.takeRequest()
        assertEquals("GET", request.method)
        assertEquals("/api/v1/library/stats", request.path)
    }

    @Test
    fun `libraryPull sends pull command with arguments`() {
        mockServer.enqueue(
            MockResponse()
                .setResponseCode(200)
                .setHeader("Content-Type", "application/json")
                .setBody("""{"success": true, "message": "Collection pulled"}""")
        )

        val result =
            client.libraryPull(
                source = "https://docs.example.com",
                name = "example-docs",
                shared = true
            )

        assertTrue(result.isSuccess)

        val request = mockServer.takeRequest()
        assertEquals("POST", request.method)
        assertEquals("/api/v1/interactive/command", request.path)
        val body = request.body.readUtf8()
        assertTrue(body.contains("\"command\":\"library\""))
        assertTrue(body.contains("pull"))
        assertTrue(body.contains("https://docs.example.com"))
        assertTrue(body.contains("--name"))
        assertTrue(body.contains("example-docs"))
        assertTrue(body.contains("--shared"))
    }

    @Test
    fun `libraryRemove sends remove command`() {
        mockServer.enqueue(
            MockResponse()
                .setResponseCode(200)
                .setHeader("Content-Type", "application/json")
                .setBody("""{"success": true, "message": "Collection removed"}""")
        )

        val result = client.libraryRemove("react-docs")

        assertTrue(result.isSuccess)

        val request = mockServer.takeRequest()
        assertEquals("POST", request.method)
        assertEquals("/api/v1/interactive/command", request.path)
        val body = request.body.readUtf8()
        assertTrue(body.contains("\"command\":\"library\""))
        assertTrue(body.contains("remove"))
        assertTrue(body.contains("react-docs"))
    }

    // ========================================================================
    // Links Endpoint Tests
    // ========================================================================

    @Test
    fun `linksList returns list of links`() {
        mockServer.enqueue(
            MockResponse()
                .setResponseCode(200)
                .setHeader("Content-Type", "application/json")
                .setBody(
                    """
                    {
                        "links": [
                            {
                                "source": "spec:1",
                                "target": "decision:cache-strategy",
                                "context": "Specification references caching decision",
                                "created_at": "2026-01-15T10:00:00Z"
                            }
                        ],
                        "count": 1
                    }
                    """.trimIndent()
                )
        )

        val result = client.linksList()

        assertTrue(result.isSuccess)
        val response = result.getOrNull()!!
        assertEquals(1, response.count)
        assertEquals("spec:1", response.links[0].source)
        assertEquals("decision:cache-strategy", response.links[0].target)

        val request = mockServer.takeRequest()
        assertEquals("GET", request.method)
        assertEquals("/api/v1/links", request.path)
    }

    @Test
    fun `linksSearch returns matching entities`() {
        mockServer.enqueue(
            MockResponse()
                .setResponseCode(200)
                .setHeader("Content-Type", "application/json")
                .setBody(
                    """
                    {
                        "query": "cache",
                        "results": [
                            {
                                "entity_id": "decision:cache-strategy",
                                "type": "decision",
                                "name": "cache-strategy",
                                "total_links": 3
                            }
                        ],
                        "count": 1
                    }
                    """.trimIndent()
                )
        )

        val result = client.linksSearch("cache")

        assertTrue(result.isSuccess)
        val response = result.getOrNull()!!
        assertEquals("cache", response.query)
        assertEquals(1, response.count)
        assertEquals("decision:cache-strategy", response.results[0].entityId)

        val request = mockServer.takeRequest()
        assertEquals("GET", request.method)
        assertTrue(request.path!!.contains("/api/v1/links/search"))
        assertTrue(request.path!!.contains("q=cache"))
    }

    @Test
    fun `linksStats returns links statistics`() {
        mockServer.enqueue(
            MockResponse()
                .setResponseCode(200)
                .setHeader("Content-Type", "application/json")
                .setBody(
                    """
                    {
                        "total_links": 50,
                        "total_sources": 20,
                        "total_targets": 35,
                        "orphan_entities": 5,
                        "most_linked": [
                            {"entity_id": "spec:1", "type": "specification", "total_links": 8}
                        ],
                        "enabled": true
                    }
                    """.trimIndent()
                )
        )

        val result = client.linksStats()

        assertTrue(result.isSuccess)
        val stats = result.getOrNull()!!
        assertEquals(50, stats.totalLinks)
        assertEquals(20, stats.totalSources)
        assertEquals(5, stats.orphanEntities)
        assertTrue(stats.enabled)

        val request = mockServer.takeRequest()
        assertEquals("GET", request.method)
        assertEquals("/api/v1/links/stats", request.path)
    }

    @Test
    fun `linksRebuild sends rebuild command`() {
        mockServer.enqueue(
            MockResponse()
                .setResponseCode(200)
                .setHeader("Content-Type", "application/json")
                .setBody("""{"success": true, "message": "Links index rebuilt"}""")
        )

        val result = client.linksRebuild()

        assertTrue(result.isSuccess)

        val request = mockServer.takeRequest()
        assertEquals("POST", request.method)
        assertEquals("/api/v1/interactive/command", request.path)
        val body = request.body.readUtf8()
        assertTrue(body.contains("\"command\":\"links\""))
        assertTrue(body.contains("rebuild"))
    }

    // ========================================================================
    // Additional Error Scenario Tests
    // ========================================================================

    @Test
    fun `returns failure on HTTP 401 unauthorized`() {
        mockServer.enqueue(
            MockResponse()
                .setResponseCode(401)
                .setHeader("Content-Type", "application/json")
                .setBody("""{"error": "Authentication required"}""")
        )

        val result = client.getTask()

        assertTrue(result.isFailure)
        val error = result.exceptionOrNull() as ApiException
        assertEquals(401, error.statusCode)
        assertEquals("Authentication required", error.message)
    }

    @Test
    fun `returns failure on HTTP 403 forbidden`() {
        mockServer.enqueue(
            MockResponse()
                .setResponseCode(403)
                .setHeader("Content-Type", "application/json")
                .setBody("""{"error": "CSRF token invalid"}""")
        )

        val result = client.plan()

        assertTrue(result.isFailure)
        val error = result.exceptionOrNull() as ApiException
        assertEquals(403, error.statusCode)
        assertEquals("CSRF token invalid", error.message)
    }

    @Test
    fun `returns failure on HTTP 404 for missing task`() {
        mockServer.enqueue(
            MockResponse()
                .setResponseCode(404)
                .setHeader("Content-Type", "application/json")
                .setBody("""{"error": "Task not found"}""")
        )

        val result = client.getTaskCosts("nonexistent")

        assertTrue(result.isFailure)
        val error = result.exceptionOrNull() as ApiException
        assertEquals(404, error.statusCode)
        assertEquals("Task not found", error.message)
    }

    @Test
    fun `returns failure on HTTP 429 rate limited`() {
        mockServer.enqueue(
            MockResponse()
                .setResponseCode(429)
                .setHeader("Content-Type", "application/json")
                .setBody("""{"error": "Rate limit exceeded"}""")
        )

        val result = client.getStatus()

        assertTrue(result.isFailure)
        val error = result.exceptionOrNull() as ApiException
        assertEquals(429, error.statusCode)
        assertEquals("Rate limit exceeded", error.message)
    }

    @Test
    fun `returns failure on HTTP 500 with non-JSON body`() {
        mockServer.enqueue(
            MockResponse()
                .setResponseCode(500)
                .setBody("Internal Server Error")
        )

        val result = client.implement()

        assertTrue(result.isFailure)
        val error = result.exceptionOrNull() as ApiException
        assertEquals(500, error.statusCode)
        assertTrue(error.message.contains("500"))
    }

    @Test
    fun `returns failure on HTTP 502 bad gateway`() {
        mockServer.enqueue(
            MockResponse()
                .setResponseCode(502)
                .setBody("<html>Bad Gateway</html>")
        )

        val result = client.getStatus()

        assertTrue(result.isFailure)
        val error = result.exceptionOrNull() as ApiException
        assertEquals(502, error.statusCode)
    }

    @Test
    fun `POST error returns failure with correct status code`() {
        mockServer.enqueue(
            MockResponse()
                .setResponseCode(409)
                .setHeader("Content-Type", "application/json")
                .setBody("""{"error": "Task already running"}""")
        )

        val result = client.startTask(ref = "github:1")

        assertTrue(result.isFailure)
        val error = result.exceptionOrNull() as ApiException
        assertEquals(409, error.statusCode)
        assertEquals("Task already running", error.message)
    }

    // ========================================================================
    // Additional Workflow Endpoint Tests
    // ========================================================================

    @Test
    fun `continueWorkflow sends auto flag`() {
        mockServer.enqueue(
            MockResponse()
                .setResponseCode(200)
                .setHeader("Content-Type", "application/json")
                .setBody(
                    """
                    {
                        "success": true,
                        "state": "implementing",
                        "action": "implement",
                        "next_actions": ["review"],
                        "message": "Continuing with implementation"
                    }
                    """.trimIndent()
                )
        )

        val result = client.continueWorkflow(auto = true)

        assertTrue(result.isSuccess)
        val response = result.getOrNull()!!
        assertTrue(response.success)
        assertEquals("implementing", response.state)

        val request = mockServer.takeRequest()
        assertEquals("POST", request.method)
        assertEquals("/api/v1/workflow/continue", request.path)
        assertTrue(request.body.readUtf8().contains("\"auto\":true"))
    }

    @Test
    fun `resume sends POST request`() {
        mockServer.enqueue(
            MockResponse()
                .setResponseCode(200)
                .setHeader("Content-Type", "application/json")
                .setBody("""{"success": true, "state": "implementing"}""")
        )

        val result = client.resume()

        assertTrue(result.isSuccess)

        val request = mockServer.takeRequest()
        assertEquals("POST", request.method)
        assertEquals("/api/v1/workflow/resume", request.path)
    }

    @Test
    fun `abandon sends POST request`() {
        mockServer.enqueue(
            MockResponse()
                .setResponseCode(200)
                .setHeader("Content-Type", "application/json")
                .setBody("""{"success": true, "state": "idle"}""")
        )

        val result = client.abandon()

        assertTrue(result.isSuccess)
        val response = result.getOrNull()!!
        assertEquals("idle", response.state)

        val request = mockServer.takeRequest()
        assertEquals("POST", request.method)
        assertEquals("/api/v1/workflow/abandon", request.path)
    }

    @Test
    fun `reset sends POST request`() {
        mockServer.enqueue(
            MockResponse()
                .setResponseCode(200)
                .setHeader("Content-Type", "application/json")
                .setBody("""{"success": true, "state": "idle", "message": "Reset to idle"}""")
        )

        val result = client.reset()

        assertTrue(result.isSuccess)
        val response = result.getOrNull()!!
        assertEquals("idle", response.state)

        val request = mockServer.takeRequest()
        assertEquals("POST", request.method)
        assertEquals("/api/v1/workflow/reset", request.path)
    }

    @Test
    fun `question sends message in request body`() {
        mockServer.enqueue(
            MockResponse()
                .setResponseCode(200)
                .setHeader("Content-Type", "application/json")
                .setBody("""{"success": true}""")
        )

        client.question("Can you also add error handling?")

        val request = mockServer.takeRequest()
        assertEquals("POST", request.method)
        assertEquals("/api/v1/workflow/question", request.path)
        assertTrue(request.body.readUtf8().contains("Can you also add error handling?"))
    }

    @Test
    fun `addNote sends task ID and message`() {
        mockServer.enqueue(
            MockResponse()
                .setResponseCode(200)
                .setHeader("Content-Type", "application/json")
                .setBody("""{"success": true, "note_number": 3}""")
        )

        val result = client.addNote("task-123", "Remember to check edge cases")

        assertTrue(result.isSuccess)
        val response = result.getOrNull()!!
        assertTrue(response.success)
        assertEquals(3, response.noteNumber)

        val request = mockServer.takeRequest()
        assertEquals("POST", request.method)
        assertEquals("/api/v1/tasks/task-123/notes", request.path)
        assertTrue(request.body.readUtf8().contains("Remember to check edge cases"))
    }

    // ========================================================================
    // Interactive API Endpoint Tests
    // ========================================================================

    @Test
    fun `executeCommand sends command and args`() {
        mockServer.enqueue(
            MockResponse()
                .setResponseCode(200)
                .setHeader("Content-Type", "application/json")
                .setBody("""{"success": true, "message": "Command executed", "state": "idle"}""")
        )

        val result = client.executeCommand("status", listOf("--verbose"))

        assertTrue(result.isSuccess)
        val response = result.getOrNull()!!
        assertTrue(response.success)

        val request = mockServer.takeRequest()
        assertEquals("POST", request.method)
        assertEquals("/api/v1/interactive/command", request.path)
        val body = request.body.readUtf8()
        assertTrue(body.contains("\"command\":\"status\""))
        assertTrue(body.contains("--verbose"))
    }

    @Test
    fun `chat sends message`() {
        mockServer.enqueue(
            MockResponse()
                .setResponseCode(200)
                .setHeader("Content-Type", "application/json")
                .setBody(
                    """
                    {
                        "success": true,
                        "message": "The current task is in planning state.",
                        "messages": [
                            {"role": "user", "content": "What is the status?"},
                            {"role": "assistant", "content": "The current task is in planning state."}
                        ]
                    }
                    """.trimIndent()
                )
        )

        val result = client.chat("What is the status?")

        assertTrue(result.isSuccess)
        val response = result.getOrNull()!!
        assertTrue(response.success)
        assertEquals(2, response.messages!!.size)

        val request = mockServer.takeRequest()
        assertEquals("POST", request.method)
        assertEquals("/api/v1/interactive/chat", request.path)
    }

    @Test
    fun `getInteractiveState returns current state`() {
        mockServer.enqueue(
            MockResponse()
                .setResponseCode(200)
                .setHeader("Content-Type", "application/json")
                .setBody(
                    """
                    {
                        "success": true,
                        "state": "planning",
                        "task_id": "task-789",
                        "title": "Add caching"
                    }
                    """.trimIndent()
                )
        )

        val result = client.getInteractiveState()

        assertTrue(result.isSuccess)
        val state = result.getOrNull()!!
        assertEquals("planning", state.state)
        assertEquals("task-789", state.taskId)

        val request = mockServer.takeRequest()
        assertEquals("GET", request.method)
        assertEquals("/api/v1/interactive/state", request.path)
    }

    @Test
    fun `stopOperation sends POST request`() {
        mockServer.enqueue(
            MockResponse()
                .setResponseCode(200)
                .setHeader("Content-Type", "application/json")
                .setBody("""{"success": true, "message": "Operation stopped"}""")
        )

        val result = client.stopOperation()

        assertTrue(result.isSuccess)

        val request = mockServer.takeRequest()
        assertEquals("POST", request.method)
        assertEquals("/api/v1/interactive/stop", request.path)
    }

    @Test
    fun `getCommands returns available commands`() {
        mockServer.enqueue(
            MockResponse()
                .setResponseCode(200)
                .setHeader("Content-Type", "application/json")
                .setBody(
                    """
                    {
                        "commands": [
                            {
                                "name": "plan",
                                "description": "Run planning step",
                                "category": "workflow",
                                "requires_task": true
                            },
                            {
                                "name": "status",
                                "description": "Show current status",
                                "category": "info",
                                "requires_task": false
                            }
                        ]
                    }
                    """.trimIndent()
                )
        )

        val result = client.getCommands()

        assertTrue(result.isSuccess)
        val commands = result.getOrNull()!!
        assertEquals(2, commands.commands.size)
        assertEquals("plan", commands.commands[0].name)
        assertTrue(commands.commands[0].requiresTask)

        val request = mockServer.takeRequest()
        assertEquals("GET", request.method)
        assertEquals("/api/v1/interactive/commands", request.path)
    }

    // ========================================================================
    // Find & Specifications & Sessions Tests
    // ========================================================================

    @Test
    fun `find returns search results`() {
        mockServer.enqueue(
            MockResponse()
                .setResponseCode(200)
                .setHeader("Content-Type", "application/json")
                .setBody(
                    """
                    {
                        "query": "handleError",
                        "count": 1,
                        "matches": [
                            {
                                "file": "internal/server/handlers.go",
                                "line": 42,
                                "snippet": "func handleError(w http.ResponseWriter, err error) {",
                                "reason": "Function definition"
                            }
                        ]
                    }
                    """.trimIndent()
                )
        )

        val result = client.find("handleError")

        assertTrue(result.isSuccess)
        val response = result.getOrNull()!!
        assertEquals("handleError", response.query)
        assertEquals(1, response.count)
        assertEquals("internal/server/handlers.go", response.matches[0].file)
        assertEquals(42, response.matches[0].line)

        val request = mockServer.takeRequest()
        assertEquals("GET", request.method)
        assertTrue(request.path!!.contains("/api/v1/find"))
        assertTrue(request.path!!.contains("q=handleError"))
    }

    @Test
    fun `getSpecifications returns specs for task`() {
        mockServer.enqueue(
            MockResponse()
                .setResponseCode(200)
                .setHeader("Content-Type", "application/json")
                .setBody(
                    """
                    {
                        "specifications": [
                            {
                                "id": 1,
                                "title": "Add login form validation",
                                "content": "Validate email format and password length",
                                "status": "implemented"
                            },
                            {
                                "id": 2,
                                "title": "Add error messages",
                                "content": "Show user-friendly error messages",
                                "status": "pending"
                            }
                        ]
                    }
                    """.trimIndent()
                )
        )

        val result = client.getSpecifications("task-123")

        assertTrue(result.isSuccess)
        val specs = result.getOrNull()!!
        assertEquals(2, specs.specifications.size)
        assertEquals("Add login form validation", specs.specifications[0].title)
        assertEquals("implemented", specs.specifications[0].status)

        val request = mockServer.takeRequest()
        assertEquals("GET", request.method)
        assertEquals("/api/v1/tasks/task-123/specs", request.path)
    }

    @Test
    fun `getSessions returns session history`() {
        mockServer.enqueue(
            MockResponse()
                .setResponseCode(200)
                .setHeader("Content-Type", "application/json")
                .setBody(
                    """
                    {
                        "sessions": [
                            {
                                "id": "sess-1",
                                "step": "planning",
                                "started_at": "2026-01-15T10:00:00Z",
                                "ended_at": "2026-01-15T10:05:00Z",
                                "status": "completed"
                            }
                        ]
                    }
                    """.trimIndent()
                )
        )

        val result = client.getSessions("task-123")

        assertTrue(result.isSuccess)
        val sessions = result.getOrNull()!!
        assertEquals(1, sessions.sessions.size)
        assertEquals("planning", sessions.sessions[0].step)
        assertEquals("completed", sessions.sessions[0].status)

        val request = mockServer.takeRequest()
        assertEquals("GET", request.method)
        assertEquals("/api/v1/tasks/task-123/sessions", request.path)
    }

    // ========================================================================
    // CSRF Token & Session Cookie Tests
    // ========================================================================

    @Test
    fun `POST requests include CSRF token when set`() {
        mockServer.enqueue(
            MockResponse()
                .setResponseCode(200)
                .setHeader("Content-Type", "application/json")
                .setBody("""{"success": true, "state": "planning"}""")
        )

        client.setCsrfToken("test-csrf-token")
        client.plan()

        val request = mockServer.takeRequest()
        assertEquals("test-csrf-token", request.getHeader("X-Csrf-Token"))
    }

    @Test
    fun `GET requests include session cookie when set`() {
        mockServer.enqueue(
            MockResponse()
                .setResponseCode(200)
                .setHeader("Content-Type", "application/json")
                .setBody("""{"mode":"local","running":true,"port":3000}""")
        )

        client.setSessionCookie("mehr_session=abc123")
        client.getStatus()

        val request = mockServer.takeRequest()
        assertEquals("mehr_session=abc123", request.getHeader("Cookie"))
    }

    @Test
    fun `POST requests include both CSRF token and session cookie`() {
        mockServer.enqueue(
            MockResponse()
                .setResponseCode(200)
                .setHeader("Content-Type", "application/json")
                .setBody("""{"success": true}""")
        )

        client.setSessionCookie("mehr_session=abc123")
        client.setCsrfToken("csrf-token-xyz")
        client.plan()

        val request = mockServer.takeRequest()
        assertEquals("mehr_session=abc123", request.getHeader("Cookie"))
        assertEquals("csrf-token-xyz", request.getHeader("X-Csrf-Token"))
    }

    @Test
    fun `session cookie is extracted from Set-Cookie header`() {
        mockServer.enqueue(
            MockResponse()
                .setResponseCode(200)
                .setHeader("Content-Type", "application/json")
                .setHeader("Set-Cookie", "mehr_session=newtoken123; Path=/; HttpOnly")
                .setBody("""{"mode":"local","running":true,"port":3000}""")
        )

        // Second request to verify cookie was extracted
        mockServer.enqueue(
            MockResponse()
                .setResponseCode(200)
                .setHeader("Content-Type", "application/json")
                .setBody("""{"mode":"local","running":true,"port":3000}""")
        )

        client.getStatus()
        client.getStatus()

        mockServer.takeRequest() // First request
        val secondRequest = mockServer.takeRequest()
        assertEquals("mehr_session=newtoken123", secondRequest.getHeader("Cookie"))
    }
}
