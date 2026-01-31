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
}
