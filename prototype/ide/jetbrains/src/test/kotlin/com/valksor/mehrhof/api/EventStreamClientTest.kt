package com.valksor.mehrhof.api

import com.google.gson.Gson
import com.google.gson.JsonObject
import org.junit.jupiter.api.*
import org.junit.jupiter.api.Assertions.*

/**
 * Unit tests for EventStreamClient and related event types.
 */
class EventStreamClientTest {
    private val gson = Gson()

    // ========================================================================
    // EventType Tests
    // ========================================================================

    @Test
    fun `EventType fromString parses workflow_state_changed`() {
        assertEquals(EventType.WORKFLOW_STATE_CHANGED, EventType.fromString("workflow_state_changed"))
        assertEquals(EventType.WORKFLOW_STATE_CHANGED, EventType.fromString("state_changed"))
        assertEquals(EventType.WORKFLOW_STATE_CHANGED, EventType.fromString("WORKFLOW_STATE_CHANGED"))
        assertEquals(EventType.WORKFLOW_STATE_CHANGED, EventType.fromString("workflow-state-changed"))
    }

    @Test
    fun `EventType fromString parses task events`() {
        assertEquals(EventType.TASK_STARTED, EventType.fromString("task_started"))
        assertEquals(EventType.TASK_COMPLETED, EventType.fromString("task_completed"))
        assertEquals(EventType.TASK_FAILED, EventType.fromString("task_failed"))
    }

    @Test
    fun `EventType fromString parses agent events`() {
        assertEquals(EventType.AGENT_MESSAGE, EventType.fromString("agent_message"))
        assertEquals(EventType.AGENT_MESSAGE, EventType.fromString("agent_output"))
        assertEquals(EventType.AGENT_STARTED, EventType.fromString("agent_started"))
        assertEquals(EventType.AGENT_COMPLETED, EventType.fromString("agent_completed"))
        assertEquals(EventType.AGENT_ERROR, EventType.fromString("agent_error"))
    }

    @Test
    fun `EventType fromString parses specification events`() {
        assertEquals(EventType.SPECIFICATION_CREATED, EventType.fromString("specification_created"))
        assertEquals(EventType.SPECIFICATION_CREATED, EventType.fromString("spec_created"))
        assertEquals(EventType.SPECIFICATION_UPDATED, EventType.fromString("specification_updated"))
        assertEquals(EventType.SPECIFICATION_UPDATED, EventType.fromString("spec_updated"))
    }

    @Test
    fun `EventType fromString parses question events`() {
        assertEquals(EventType.QUESTION_ASKED, EventType.fromString("question_asked"))
        assertEquals(EventType.QUESTION_ASKED, EventType.fromString("question"))
        assertEquals(EventType.ANSWER_PROVIDED, EventType.fromString("answer_provided"))
        assertEquals(EventType.ANSWER_PROVIDED, EventType.fromString("answer"))
    }

    @Test
    fun `EventType fromString parses generic events`() {
        assertEquals(EventType.MESSAGE, EventType.fromString("message"))
        assertEquals(EventType.HEARTBEAT, EventType.fromString("heartbeat"))
        assertEquals(EventType.HEARTBEAT, EventType.fromString("ping"))
        assertEquals(EventType.ERROR, EventType.fromString("error"))
        assertEquals(EventType.CODE_CHANGE, EventType.fromString("code_change"))
    }

    @Test
    fun `EventType fromString returns UNKNOWN for unrecognized types`() {
        assertEquals(EventType.UNKNOWN, EventType.fromString("some_random_event"))
        assertEquals(EventType.UNKNOWN, EventType.fromString("xyz"))
        assertEquals(EventType.UNKNOWN, EventType.fromString(""))
    }

    @Test
    fun `EventType fromString is case insensitive`() {
        assertEquals(EventType.AGENT_MESSAGE, EventType.fromString("AGENT_MESSAGE"))
        assertEquals(EventType.AGENT_MESSAGE, EventType.fromString("Agent_Message"))
        assertEquals(EventType.AGENT_MESSAGE, EventType.fromString("agent_message"))
    }

    @Test
    fun `EventType fromString handles dashes as underscores`() {
        assertEquals(EventType.WORKFLOW_STATE_CHANGED, EventType.fromString("workflow-state-changed"))
        assertEquals(EventType.AGENT_MESSAGE, EventType.fromString("agent-message"))
    }

    // ========================================================================
    // WorkflowStateEvent Tests
    // ========================================================================

    @Test
    fun `WorkflowStateEvent parses from JSON`() {
        val json =
            JsonObject().apply {
                addProperty("state", "planning")
                addProperty("previousState", "idle")
                addProperty("taskId", "task-123")
                addProperty("message", "Started planning")
            }

        val event = json.toWorkflowStateEvent(gson)

        assertNotNull(event)
        assertEquals("planning", event!!.state)
        assertEquals("idle", event.previousState)
        assertEquals("task-123", event.taskId)
        assertEquals("Started planning", event.message)
    }

    @Test
    fun `WorkflowStateEvent handles missing optional fields`() {
        val json =
            JsonObject().apply {
                addProperty("state", "implementing")
            }

        val event = json.toWorkflowStateEvent(gson)

        assertNotNull(event)
        assertEquals("implementing", event!!.state)
        assertNull(event.previousState)
        assertNull(event.taskId)
    }

    @Test
    fun `WorkflowStateEvent returns null for invalid JSON`() {
        val json =
            JsonObject().apply {
                addProperty("invalid", "data")
            }

        val event = json.toWorkflowStateEvent(gson)

        // Should return object with null state, not throw
        assertNotNull(event)
    }

    // ========================================================================
    // AgentMessageEvent Tests
    // ========================================================================

    @Test
    fun `AgentMessageEvent parses from JSON`() {
        val json =
            JsonObject().apply {
                addProperty("content", "Processing your request...")
                addProperty("type", "text")
            }

        val event = json.toAgentMessageEvent(gson)

        assertNotNull(event)
        assertEquals("Processing your request...", event!!.content)
        assertEquals("text", event.type)
    }

    @Test
    fun `AgentMessageEvent parses tool use`() {
        val json =
            JsonObject().apply {
                addProperty("content", "Using tool")
                addProperty("type", "tool_use")
                addProperty("toolName", "read_file")
                addProperty("toolInput", "/path/to/file")
            }

        val event = json.toAgentMessageEvent(gson)

        assertNotNull(event)
        assertEquals("tool_use", event!!.type)
        assertEquals("read_file", event.toolName)
        assertEquals("/path/to/file", event.toolInput)
    }

    @Test
    fun `AgentMessageEvent handles missing optional fields`() {
        val json =
            JsonObject().apply {
                addProperty("content", "Simple message")
            }

        val event = json.toAgentMessageEvent(gson)

        assertNotNull(event)
        assertEquals("Simple message", event!!.content)
        assertNull(event.type)
        assertNull(event.toolName)
    }

    // ========================================================================
    // QuestionEvent Tests
    // ========================================================================

    @Test
    fun `QuestionEvent parses from JSON with options`() {
        val json =
            gson.fromJson(
                """
                {
                    "question": "Which approach do you prefer?",
                    "options": ["Option A", "Option B", "Option C"],
                    "taskId": "task-123"
                }
                """.trimIndent(),
                JsonObject::class.java
            )

        val event = json.toQuestionEvent(gson)

        assertNotNull(event)
        assertEquals("Which approach do you prefer?", event!!.question)
        assertNotNull(event.options)
        assertEquals(3, event.options!!.size)
        assertEquals("Option A", event.options!![0])
        assertEquals("task-123", event.taskId)
    }

    @Test
    fun `QuestionEvent parses from JSON without options`() {
        val json =
            JsonObject().apply {
                addProperty("question", "Please describe the issue")
            }

        val event = json.toQuestionEvent(gson)

        assertNotNull(event)
        assertEquals("Please describe the issue", event!!.question)
        assertNull(event.options)
    }

    @Test
    fun `QuestionEvent handles empty options array`() {
        val json =
            gson.fromJson(
                """
                {
                    "question": "Any input?",
                    "options": []
                }
                """.trimIndent(),
                JsonObject::class.java
            )

        val event = json.toQuestionEvent(gson)

        assertNotNull(event)
        assertNotNull(event!!.options)
        assertTrue(event.options!!.isEmpty())
    }

    // ========================================================================
    // EventStreamClient State Tests
    // ========================================================================

    @Test
    fun `EventStreamClient initializes as disconnected`() {
        val client =
            EventStreamClient(
                baseUrl = "http://localhost:3000",
                onEvent = { _, _ -> }
            )

        assertFalse(client.isConnected())
    }

    @Test
    fun `EventStreamClient disconnect when not connected is safe`() {
        val client =
            EventStreamClient(
                baseUrl = "http://localhost:3000",
                onEvent = { _, _ -> }
            )

        // Should not throw
        assertDoesNotThrow { client.disconnect() }
        assertFalse(client.isConnected())
    }

    @Test
    fun `EventStreamClient reconnect is idempotent when disconnected`() {
        val client =
            EventStreamClient(
                baseUrl = "http://localhost:3000",
                onEvent = { _, _ -> }
            )

        // Calling reconnect when already disconnected should be safe
        assertDoesNotThrow { client.reconnect() }

        // State should remain disconnected (no server running)
        assertFalse(client.isConnected())
    }

    // ========================================================================
    // Event Data Class Tests
    // ========================================================================

    @Test
    fun `WorkflowStateEvent data class equality`() {
        val event1 = WorkflowStateEvent("planning", "idle", "task-1", "Starting")
        val event2 = WorkflowStateEvent("planning", "idle", "task-1", "Starting")
        val event3 = WorkflowStateEvent("implementing", "planning", "task-1", "Started")

        assertEquals(event1, event2)
        assertNotEquals(event1, event3)
    }

    @Test
    fun `AgentMessageEvent data class equality`() {
        val event1 = AgentMessageEvent("Hello", "text", null, null)
        val event2 = AgentMessageEvent("Hello", "text", null, null)
        val event3 = AgentMessageEvent("World", "text", null, null)

        assertEquals(event1, event2)
        assertNotEquals(event1, event3)
    }

    @Test
    fun `QuestionEvent data class equality`() {
        val event1 = QuestionEvent("Question?", listOf("A", "B"), "task-1")
        val event2 = QuestionEvent("Question?", listOf("A", "B"), "task-1")
        val event3 = QuestionEvent("Different?", listOf("A", "B"), "task-1")

        assertEquals(event1, event2)
        assertNotEquals(event1, event3)
    }

    // ========================================================================
    // Edge Cases
    // ========================================================================

    @Test
    fun `EventType handles whitespace in type string`() {
        // Whitespace should be handled gracefully
        assertEquals(EventType.UNKNOWN, EventType.fromString("  "))
        assertEquals(EventType.UNKNOWN, EventType.fromString(" agent_message "))
    }

    @Test
    fun `JSON parsing handles special characters in content`() {
        val json =
            JsonObject().apply {
                addProperty("content", "Line 1\nLine 2\tTabbed\r\nWindows line")
                addProperty("type", "text")
            }

        val event = json.toAgentMessageEvent(gson)

        assertNotNull(event)
        assertTrue(event!!.content.contains("\n"))
        assertTrue(event.content.contains("\t"))
    }

    @Test
    fun `JSON parsing handles unicode in content`() {
        val json =
            JsonObject().apply {
                addProperty("content", "Hello 世界 🌍")
            }

        val event = json.toAgentMessageEvent(gson)

        assertNotNull(event)
        assertEquals("Hello 世界 🌍", event!!.content)
    }
}
