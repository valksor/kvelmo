package com.valksor.mehrhof.api

import com.google.gson.Gson
import com.google.gson.GsonBuilder
import com.valksor.mehrhof.api.models.*
import org.junit.jupiter.api.*
import org.junit.jupiter.api.Assertions.*

/**
 * Unit tests for API model JSON deserialization.
 */
class ApiModelsTest {

    private val gson: Gson = GsonBuilder().setLenient().create()

    // ========================================================================
    // StatusResponse Tests
    // ========================================================================

    @Test
    fun `StatusResponse deserializes correctly`() {
        val json = """{"mode":"local","running":true,"port":3000,"state":"idle"}"""

        val response = gson.fromJson(json, StatusResponse::class.java)

        assertEquals("local", response.mode)
        assertTrue(response.running)
        assertEquals(3000, response.port)
        assertEquals("idle", response.state)
    }

    @Test
    fun `StatusResponse handles null state`() {
        val json = """{"mode":"local","running":false,"port":3000}"""

        val response = gson.fromJson(json, StatusResponse::class.java)

        assertNull(response.state)
    }

    // ========================================================================
    // TaskResponse Tests
    // ========================================================================

    @Test
    fun `TaskResponse deserializes with active task`() {
        val json = """
            {
                "active": true,
                "task": {
                    "id": "task-123",
                    "state": "planning",
                    "ref": "github:42",
                    "branch": "feature/test",
                    "worktree_path": "/path/to/worktree",
                    "started": "2024-01-15T10:00:00Z"
                },
                "work": {
                    "title": "Fix login bug",
                    "external_key": "42",
                    "created_at": "2024-01-15T10:00:00Z",
                    "updated_at": "2024-01-15T11:00:00Z"
                }
            }
        """.trimIndent()

        val response = gson.fromJson(json, TaskResponse::class.java)

        assertTrue(response.active)
        assertNotNull(response.task)
        assertEquals("task-123", response.task!!.id)
        assertEquals("planning", response.task!!.state)
        assertEquals("github:42", response.task!!.ref)
        assertEquals("feature/test", response.task!!.branch)
        assertEquals("/path/to/worktree", response.task!!.worktreePath)

        assertNotNull(response.work)
        assertEquals("Fix login bug", response.work!!.title)
        assertEquals("42", response.work!!.externalKey)
    }

    @Test
    fun `TaskResponse deserializes with pending question`() {
        val json = """
            {
                "active": true,
                "task": {"id": "task-1", "state": "waiting", "ref": "file:task.md"},
                "pending_question": {
                    "question": "Which approach?",
                    "options": ["Option A", "Option B"]
                }
            }
        """.trimIndent()

        val response = gson.fromJson(json, TaskResponse::class.java)

        assertNotNull(response.pendingQuestion)
        assertEquals("Which approach?", response.pendingQuestion!!.question)
        assertEquals(2, response.pendingQuestion!!.options!!.size)
    }

    @Test
    fun `TaskResponse deserializes with no active task`() {
        val json = """{"active": false}"""

        val response = gson.fromJson(json, TaskResponse::class.java)

        assertFalse(response.active)
        assertNull(response.task)
        assertNull(response.work)
    }

    // ========================================================================
    // TaskListResponse Tests
    // ========================================================================

    @Test
    fun `TaskListResponse deserializes task list`() {
        val json = """
            {
                "tasks": [
                    {"id": "task-1", "title": "Task 1", "state": "done", "created_at": "2024-01-01"},
                    {"id": "task-2", "title": "Task 2", "state": "planning", "worktree_path": "/path"}
                ],
                "count": 2
            }
        """.trimIndent()

        val response = gson.fromJson(json, TaskListResponse::class.java)

        assertEquals(2, response.count)
        assertEquals(2, response.tasks.size)
        assertEquals("task-1", response.tasks[0].id)
        assertEquals("Task 1", response.tasks[0].title)
        assertEquals("done", response.tasks[0].state)
        assertEquals("/path", response.tasks[1].worktreePath)
    }

    @Test
    fun `TaskListResponse handles empty list`() {
        val json = """{"tasks": [], "count": 0}"""

        val response = gson.fromJson(json, TaskListResponse::class.java)

        assertEquals(0, response.count)
        assertTrue(response.tasks.isEmpty())
    }

    // ========================================================================
    // WorkflowResponse Tests
    // ========================================================================

    @Test
    fun `WorkflowResponse deserializes success response`() {
        val json = """{"success": true, "state": "planning", "message": "Started planning"}"""

        val response = gson.fromJson(json, WorkflowResponse::class.java)

        assertTrue(response.success)
        assertEquals("planning", response.state)
        assertEquals("Started planning", response.message)
        assertNull(response.error)
    }

    @Test
    fun `WorkflowResponse deserializes error response`() {
        val json = """{"success": false, "error": "No active task"}"""

        val response = gson.fromJson(json, WorkflowResponse::class.java)

        assertFalse(response.success)
        assertEquals("No active task", response.error)
    }

    // ========================================================================
    // ContinueResponse Tests
    // ========================================================================

    @Test
    fun `ContinueResponse deserializes correctly`() {
        val json = """
            {
                "success": true,
                "state": "implementing",
                "action": "implement",
                "next_actions": ["review", "finish"],
                "message": "Implementation started"
            }
        """.trimIndent()

        val response = gson.fromJson(json, ContinueResponse::class.java)

        assertTrue(response.success)
        assertEquals("implementing", response.state)
        assertEquals("implement", response.action)
        assertEquals(2, response.nextActions.size)
        assertEquals("review", response.nextActions[0])
    }

    // ========================================================================
    // GuideResponse Tests
    // ========================================================================

    @Test
    fun `GuideResponse deserializes with next actions`() {
        val json = """
            {
                "has_task": true,
                "task_id": "task-123",
                "title": "Fix bug",
                "state": "idle",
                "specifications": 3,
                "next_actions": [
                    {"command": "plan", "description": "Start planning", "endpoint": "/api/v1/workflow/plan"},
                    {"command": "abandon", "description": "Abandon task"}
                ]
            }
        """.trimIndent()

        val response = gson.fromJson(json, GuideResponse::class.java)

        assertTrue(response.hasTask)
        assertEquals("task-123", response.taskId)
        assertEquals(3, response.specifications)
        assertEquals(2, response.nextActions.size)
        assertEquals("plan", response.nextActions[0].command)
        assertEquals("/api/v1/workflow/plan", response.nextActions[0].endpoint)
    }

    @Test
    fun `GuideResponse deserializes with pending question`() {
        val json = """
            {
                "has_task": true,
                "task_id": "task-1",
                "state": "waiting",
                "specifications": 0,
                "pending_question": {
                    "question": "How should we proceed?",
                    "options": ["Continue", "Stop"]
                },
                "next_actions": []
            }
        """.trimIndent()

        val response = gson.fromJson(json, GuideResponse::class.java)

        assertNotNull(response.pendingQuestion)
        assertEquals("How should we proceed?", response.pendingQuestion!!.question)
        assertEquals(2, response.pendingQuestion!!.options!!.size)
    }

    // ========================================================================
    // Cost Models Tests
    // ========================================================================

    @Test
    fun `CostInfo deserializes correctly`() {
        val json = """
            {
                "total_tokens": 1500,
                "input_tokens": 1000,
                "output_tokens": 500,
                "cached_tokens": 200,
                "total_cost_usd": 0.075
            }
        """.trimIndent()

        val response = gson.fromJson(json, CostInfo::class.java)

        assertEquals(1500, response.totalTokens)
        assertEquals(1000, response.inputTokens)
        assertEquals(500, response.outputTokens)
        assertEquals(200, response.cachedTokens)
        assertEquals(0.075, response.totalCostUsd, 0.0001)
    }

    @Test
    fun `CostInfo has zero defaults`() {
        val json = """{}"""

        val response = gson.fromJson(json, CostInfo::class.java)

        assertEquals(0, response.totalTokens)
        assertEquals(0.0, response.totalCostUsd, 0.0001)
    }

    @Test
    fun `TaskCostResponse deserializes with step breakdown`() {
        val json = """
            {
                "task_id": "task-123",
                "title": "Test task",
                "total_tokens": 5000,
                "input_tokens": 4000,
                "output_tokens": 1000,
                "cached_tokens": 500,
                "cached_percent": 12.5,
                "total_cost_usd": 0.25,
                "by_step": {
                    "planning": {
                        "input_tokens": 2000,
                        "output_tokens": 500,
                        "cached_tokens": 200,
                        "total_tokens": 2500,
                        "cost_usd": 0.125,
                        "calls": 1
                    }
                }
            }
        """.trimIndent()

        val response = gson.fromJson(json, TaskCostResponse::class.java)

        assertEquals("task-123", response.taskId)
        assertEquals(5000, response.totalTokens)
        assertEquals(12.5, response.cachedPercent!!, 0.01)
        assertNotNull(response.byStep)
        assertNotNull(response.byStep!!["planning"])
        assertEquals(1, response.byStep!!["planning"]!!.calls)
    }

    @Test
    fun `AllCostsResponse deserializes with grand total`() {
        val json = """
            {
                "tasks": [],
                "grand_total": {
                    "input_tokens": 10000,
                    "output_tokens": 2000,
                    "total_tokens": 12000,
                    "cached_tokens": 1000,
                    "cost_usd": 0.50
                },
                "monthly": {
                    "month": "2024-01",
                    "spent": 5.25,
                    "max_cost": 100.0,
                    "warning_at": 80.0,
                    "warning_sent": false
                }
            }
        """.trimIndent()

        val response = gson.fromJson(json, AllCostsResponse::class.java)

        assertEquals(12000, response.grandTotal.totalTokens)
        assertEquals(0.50, response.grandTotal.costUsd, 0.001)
        assertNotNull(response.monthly)
        assertEquals("2024-01", response.monthly!!.month)
        assertEquals(5.25, response.monthly!!.spent, 0.001)
    }

    // ========================================================================
    // Specification Models Tests
    // ========================================================================

    @Test
    fun `SpecificationsResponse deserializes correctly`() {
        val json = """
            {
                "specifications": [
                    {
                        "id": 1,
                        "title": "Add login feature",
                        "content": "## Specification\nAdd a login form...",
                        "created_at": "2024-01-15T10:00:00Z",
                        "status": "implemented"
                    }
                ]
            }
        """.trimIndent()

        val response = gson.fromJson(json, SpecificationsResponse::class.java)

        assertEquals(1, response.specifications.size)
        assertEquals(1, response.specifications[0].id)
        assertEquals("Add login feature", response.specifications[0].title)
        assertTrue(response.specifications[0].content.contains("Specification"))
        assertEquals("implemented", response.specifications[0].status)
    }

    // ========================================================================
    // Session Models Tests
    // ========================================================================

    @Test
    fun `SessionsResponse deserializes correctly`() {
        val json = """
            {
                "sessions": [
                    {
                        "id": "session-1",
                        "step": "planning",
                        "started_at": "2024-01-15T10:00:00Z",
                        "ended_at": "2024-01-15T10:30:00Z",
                        "status": "completed"
                    },
                    {
                        "id": "session-2",
                        "step": "implementing",
                        "started_at": "2024-01-15T11:00:00Z"
                    }
                ]
            }
        """.trimIndent()

        val response = gson.fromJson(json, SessionsResponse::class.java)

        assertEquals(2, response.sessions.size)
        assertEquals("session-1", response.sessions[0].id)
        assertEquals("planning", response.sessions[0].step)
        assertEquals("completed", response.sessions[0].status)
        assertNull(response.sessions[1].endedAt)
    }

    // ========================================================================
    // Agent Models Tests
    // ========================================================================

    @Test
    fun `AgentsListResponse deserializes correctly`() {
        val json = """
            {
                "agents": [
                    {
                        "name": "claude",
                        "type": "claude",
                        "available": true,
                        "description": "Claude AI assistant",
                        "version": "1.0.0",
                        "capabilities": {
                            "streaming": true,
                            "tool_use": true,
                            "file_operations": true,
                            "code_execution": true,
                            "multi_turn": true,
                            "system_prompt": true,
                            "allowed_tools": ["read", "write", "execute"]
                        },
                        "models": [
                            {
                                "id": "claude-3-opus",
                                "name": "Claude 3 Opus",
                                "default": true,
                                "max_tokens": 200000,
                                "input_cost_usd": 0.015,
                                "output_cost_usd": 0.075
                            }
                        ]
                    }
                ],
                "count": 1
            }
        """.trimIndent()

        val response = gson.fromJson(json, AgentsListResponse::class.java)

        assertEquals(1, response.count)
        val agent = response.agents[0]
        assertEquals("claude", agent.name)
        assertTrue(agent.available)

        assertNotNull(agent.capabilities)
        assertTrue(agent.capabilities!!.streaming)
        assertTrue(agent.capabilities!!.toolUse)
        assertEquals(3, agent.capabilities!!.allowedTools!!.size)

        assertNotNull(agent.models)
        assertEquals(1, agent.models!!.size)
        assertTrue(agent.models!![0].default)
        assertEquals(200000, agent.models!![0].maxTokens)
    }

    // ========================================================================
    // Provider Models Tests
    // ========================================================================

    @Test
    fun `ProvidersListResponse deserializes correctly`() {
        val json = """
            {
                "providers": [
                    {
                        "scheme": "github",
                        "shorthand": "gh",
                        "name": "GitHub",
                        "description": "GitHub issues and pull requests",
                        "env_vars": ["GITHUB_TOKEN"]
                    },
                    {
                        "scheme": "file",
                        "name": "File",
                        "description": "Local markdown files"
                    }
                ],
                "count": 2
            }
        """.trimIndent()

        val response = gson.fromJson(json, ProvidersListResponse::class.java)

        assertEquals(2, response.count)
        assertEquals("github", response.providers[0].scheme)
        assertEquals("gh", response.providers[0].shorthand)
        assertEquals(1, response.providers[0].envVars!!.size)
        assertNull(response.providers[1].shorthand)
        assertNull(response.providers[1].envVars)
    }

    // ========================================================================
    // Request Models Tests
    // ========================================================================

    @Test
    fun `FinishRequest serializes correctly`() {
        val request = FinishRequest(
            squashMerge = true,
            deleteBranch = false,
            targetBranch = "main",
            pushAfter = true,
            forceMerge = false,
            draftPr = true,
            prTitle = "Fix: Login bug",
            prBody = "This PR fixes the login issue"
        )

        val json = gson.toJson(request)

        assertTrue(json.contains("\"squash_merge\":true"))
        assertTrue(json.contains("\"delete_branch\":false"))
        assertTrue(json.contains("\"target_branch\":\"main\""))
        assertTrue(json.contains("\"draft_pr\":true"))
    }

    @Test
    fun `FinishRequest has correct defaults`() {
        val request = FinishRequest()

        assertFalse(request.squashMerge)
        assertTrue(request.deleteBranch)
        assertNull(request.targetBranch)
        assertTrue(request.pushAfter)
        assertFalse(request.forceMerge)
        assertFalse(request.draftPr)
    }

    @Test
    fun `StartTaskRequest serializes correctly`() {
        val request = StartTaskRequest(ref = "github:123", content = "Fix the bug")

        val json = gson.toJson(request)

        assertTrue(json.contains("\"ref\":\"github:123\""))
        assertTrue(json.contains("\"content\":\"Fix the bug\""))
    }

    @Test
    fun `WorkflowRequest serializes correctly`() {
        val request = WorkflowRequest(agent = "claude-opus")

        val json = gson.toJson(request)

        assertTrue(json.contains("\"agent\":\"claude-opus\""))
    }

    @Test
    fun `AnswerRequest serializes correctly`() {
        val request = AnswerRequest(answer = "Yes, proceed with option A")

        val json = gson.toJson(request)

        assertTrue(json.contains("\"answer\":\"Yes, proceed with option A\""))
    }

    // ========================================================================
    // Interactive API Models Tests
    // ========================================================================

    @Test
    fun `InteractiveCommandRequest serializes correctly`() {
        val request = InteractiveCommandRequest(command = "start", args = listOf("github:123"))

        val json = gson.toJson(request)

        assertTrue(json.contains("\"command\":\"start\""))
        assertTrue(json.contains("\"args\":[\"github:123\"]"))
    }

    @Test
    fun `InteractiveCommandRequest with empty args serializes correctly`() {
        val request = InteractiveCommandRequest(command = "plan")

        val json = gson.toJson(request)

        assertTrue(json.contains("\"command\":\"plan\""))
        assertTrue(json.contains("\"args\":[]"))
    }

    @Test
    fun `InteractiveCommandResponse deserializes success response`() {
        val json = """{"success": true, "message": "Task started", "state": "idle"}"""

        val response = gson.fromJson(json, InteractiveCommandResponse::class.java)

        assertTrue(response.success)
        assertEquals("Task started", response.message)
        assertEquals("idle", response.state)
        assertNull(response.error)
    }

    @Test
    fun `InteractiveCommandResponse deserializes error response`() {
        val json = """{"success": false, "error": "No active task"}"""

        val response = gson.fromJson(json, InteractiveCommandResponse::class.java)

        assertFalse(response.success)
        assertEquals("No active task", response.error)
    }

    @Test
    fun `InteractiveChatRequest serializes correctly`() {
        val request = InteractiveChatRequest(message = "Explain the auth flow")

        val json = gson.toJson(request)

        assertTrue(json.contains("\"message\":\"Explain the auth flow\""))
    }

    @Test
    fun `InteractiveChatResponse deserializes with messages`() {
        val json = """
            {
                "success": true,
                "messages": [
                    {"role": "user", "content": "Hello", "timestamp": "2024-01-15T10:00:00Z"},
                    {"role": "assistant", "content": "Hi there!", "timestamp": "2024-01-15T10:00:01Z"}
                ]
            }
        """.trimIndent()

        val response = gson.fromJson(json, InteractiveChatResponse::class.java)

        assertTrue(response.success)
        assertNotNull(response.messages)
        assertEquals(2, response.messages!!.size)
        assertEquals("user", response.messages!![0].role)
        assertEquals("Hello", response.messages!![0].content)
        assertEquals("assistant", response.messages!![1].role)
        assertEquals("Hi there!", response.messages!![1].content)
    }

    @Test
    fun `InteractiveChatResponse deserializes error`() {
        val json = """{"success": false, "error": "Agent not available"}"""

        val response = gson.fromJson(json, InteractiveChatResponse::class.java)

        assertFalse(response.success)
        assertEquals("Agent not available", response.error)
        assertNull(response.messages)
    }

    @Test
    fun `InteractiveAnswerRequest serializes correctly`() {
        val request = InteractiveAnswerRequest(response = "Option A")

        val json = gson.toJson(request)

        assertTrue(json.contains("\"response\":\"Option A\""))
    }

    @Test
    fun `InteractiveStateResponse deserializes correctly`() {
        val json = """
            {
                "success": true,
                "state": "planning",
                "task_id": "task-abc123",
                "title": "Fix login bug"
            }
        """.trimIndent()

        val response = gson.fromJson(json, InteractiveStateResponse::class.java)

        assertTrue(response.success)
        assertEquals("planning", response.state)
        assertEquals("task-abc123", response.taskId)
        assertEquals("Fix login bug", response.title)
    }

    @Test
    fun `InteractiveStateResponse deserializes with no active task`() {
        val json = """{"success": true, "state": "idle"}"""

        val response = gson.fromJson(json, InteractiveStateResponse::class.java)

        assertTrue(response.success)
        assertEquals("idle", response.state)
        assertNull(response.taskId)
        assertNull(response.title)
    }

    @Test
    fun `InteractiveStopResponse deserializes success`() {
        val json = """{"success": true, "message": "Operation cancelled: plan"}"""

        val response = gson.fromJson(json, InteractiveStopResponse::class.java)

        assertTrue(response.success)
        assertEquals("Operation cancelled: plan", response.message)
    }

    @Test
    fun `InteractiveStopResponse deserializes no operation running`() {
        val json = """{"success": false, "error": "No operation running"}"""

        val response = gson.fromJson(json, InteractiveStopResponse::class.java)

        assertFalse(response.success)
        assertEquals("No operation running", response.error)
    }

    // ========================================================================
    // Error Response Tests
    // ========================================================================

    @Test
    fun `ErrorResponse deserializes correctly`() {
        val json = """{"error": "Task not found"}"""

        val response = gson.fromJson(json, ErrorResponse::class.java)

        assertEquals("Task not found", response.error)
    }
}
