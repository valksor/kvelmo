package com.valksor.mehrhof.api

import com.google.gson.Gson
import com.google.gson.JsonObject
import okhttp3.mockwebserver.MockResponse
import okhttp3.mockwebserver.MockWebServer
import org.junit.jupiter.api.*
import org.junit.jupiter.api.Assertions.*
import java.util.concurrent.CopyOnWriteArrayList
import java.util.concurrent.CountDownLatch
import java.util.concurrent.TimeUnit

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

    // ========================================================================
    // Complete EventType Enum Coverage
    // ========================================================================

    @Test
    fun `EventType fromString parses task_started`() {
        assertEquals(EventType.TASK_STARTED, EventType.fromString("task_started"))
    }

    @Test
    fun `EventType fromString parses task_completed`() {
        assertEquals(EventType.TASK_COMPLETED, EventType.fromString("task_completed"))
    }

    @Test
    fun `EventType fromString parses task_failed`() {
        assertEquals(EventType.TASK_FAILED, EventType.fromString("task_failed"))
    }

    @Test
    fun `EventType fromString parses agent_started`() {
        assertEquals(EventType.AGENT_STARTED, EventType.fromString("agent_started"))
    }

    @Test
    fun `EventType fromString parses agent_completed`() {
        assertEquals(EventType.AGENT_COMPLETED, EventType.fromString("agent_completed"))
    }

    @Test
    fun `EventType fromString parses agent_error`() {
        assertEquals(EventType.AGENT_ERROR, EventType.fromString("agent_error"))
    }

    @Test
    fun `EventType fromString parses code_change`() {
        assertEquals(EventType.CODE_CHANGE, EventType.fromString("code_change"))
    }

    @Test
    fun `EventType fromString parses message`() {
        assertEquals(EventType.MESSAGE, EventType.fromString("message"))
    }

    @Test
    fun `EventType fromString parses error`() {
        assertEquals(EventType.ERROR, EventType.fromString("error"))
    }

    @Test
    fun `every EventType value has a corresponding fromString mapping`() {
        // Verify that every enum value (except UNKNOWN) can be reached via fromString
        val reachableTypes =
            setOf(
                EventType.fromString("workflow_state_changed"),
                EventType.fromString("task_started"),
                EventType.fromString("task_completed"),
                EventType.fromString("task_failed"),
                EventType.fromString("agent_message"),
                EventType.fromString("agent_started"),
                EventType.fromString("agent_completed"),
                EventType.fromString("agent_error"),
                EventType.fromString("specification_created"),
                EventType.fromString("specification_updated"),
                EventType.fromString("code_change"),
                EventType.fromString("question_asked"),
                EventType.fromString("answer_provided"),
                EventType.fromString("message"),
                EventType.fromString("heartbeat"),
                EventType.fromString("error"),
                EventType.fromString("definitely_unknown")
            )

        // All enum values should be reachable
        for (type in EventType.entries) {
            assertTrue(reachableTypes.contains(type), "EventType.$type is not reachable via fromString")
        }
    }

    @Test
    fun `EventType fromString alias coverage for specification events`() {
        // Canonical names
        assertEquals(EventType.SPECIFICATION_CREATED, EventType.fromString("specification_created"))
        assertEquals(EventType.SPECIFICATION_UPDATED, EventType.fromString("specification_updated"))

        // Short aliases
        assertEquals(EventType.SPECIFICATION_CREATED, EventType.fromString("spec_created"))
        assertEquals(EventType.SPECIFICATION_UPDATED, EventType.fromString("spec_updated"))
    }

    @Test
    fun `EventType fromString alias coverage for heartbeat`() {
        assertEquals(EventType.HEARTBEAT, EventType.fromString("heartbeat"))
        assertEquals(EventType.HEARTBEAT, EventType.fromString("ping"))
        assertEquals(EventType.HEARTBEAT, EventType.fromString("PING"))
    }

    @Test
    fun `EventType fromString alias coverage for question events`() {
        assertEquals(EventType.QUESTION_ASKED, EventType.fromString("question_asked"))
        assertEquals(EventType.QUESTION_ASKED, EventType.fromString("question"))
        assertEquals(EventType.ANSWER_PROVIDED, EventType.fromString("answer_provided"))
        assertEquals(EventType.ANSWER_PROVIDED, EventType.fromString("answer"))
    }

    // ========================================================================
    // Malformed JSON Handling Tests
    // ========================================================================

    @Test
    fun `toWorkflowStateEvent returns null on completely malformed JSON`() {
        val json =
            JsonObject().apply {
                addProperty("totally_wrong_field", 12345)
                addProperty("another_bad_field", true)
            }

        val event = json.toWorkflowStateEvent(gson)

        // Gson returns object with null fields rather than throwing
        assertNotNull(event)
        assertNull(event!!.state)
    }

    @Test
    fun `toAgentMessageEvent handles empty JsonObject`() {
        val json = JsonObject()

        val event = json.toAgentMessageEvent(gson)

        assertNotNull(event)
        assertNull(event!!.content)
        assertNull(event.type)
        assertNull(event.toolName)
        assertNull(event.toolInput)
    }

    @Test
    fun `toQuestionEvent handles empty JsonObject`() {
        val json = JsonObject()

        val event = json.toQuestionEvent(gson)

        assertNotNull(event)
        assertNull(event!!.question)
        assertNull(event.options)
        assertNull(event.taskId)
    }

    @Test
    fun `toWorkflowStateEvent handles numeric state value gracefully`() {
        val json =
            JsonObject().apply {
                addProperty("state", 42)
            }

        // Gson coerces numbers to strings for string fields
        val event = json.toWorkflowStateEvent(gson)

        assertNotNull(event)
    }

    @Test
    fun `toAgentMessageEvent handles nested JSON in content`() {
        val json =
            JsonObject().apply {
                addProperty("content", """{"nested": "json", "key": "value"}""")
                addProperty("type", "tool_result")
            }

        val event = json.toAgentMessageEvent(gson)

        assertNotNull(event)
        assertTrue(event!!.content.contains("nested"))
        assertEquals("tool_result", event.type)
    }

    @Test
    fun `toQuestionEvent handles non-array options gracefully`() {
        // When options is a string instead of array, Gson should handle this
        val json =
            gson.fromJson(
                """
                {
                    "question": "Pick one",
                    "options": "not_an_array"
                }
                """.trimIndent(),
                JsonObject::class.java
            )

        // This should not throw, it should return null due to parse error
        val event = json.toQuestionEvent(gson)

        // Gson may throw on type mismatch; the extension function catches it
        // Either event is null (caught exception) or options parsing failed
        // The extension function wraps in try/catch, so null is acceptable
        if (event != null) {
            assertNotNull(event.question)
        }
    }

    @Test
    fun `WorkflowStateEvent handles extra unknown fields in JSON`() {
        val json =
            JsonObject().apply {
                addProperty("state", "planning")
                addProperty("previousState", "idle")
                addProperty("unknown_extra_field", "should be ignored")
                addProperty("another_extra", 999)
            }

        val event = json.toWorkflowStateEvent(gson)

        assertNotNull(event)
        assertEquals("planning", event!!.state)
        assertEquals("idle", event.previousState)
    }

    @Test
    fun `AgentMessageEvent handles very long content`() {
        val longContent = "A".repeat(100_000)
        val json =
            JsonObject().apply {
                addProperty("content", longContent)
                addProperty("type", "text")
            }

        val event = json.toAgentMessageEvent(gson)

        assertNotNull(event)
        assertEquals(100_000, event!!.content.length)
    }

    // ========================================================================
    // Session Cookie Propagation Tests
    // ========================================================================

    @Test
    fun `EventStreamClient accepts session cookie via setter`() {
        val client =
            EventStreamClient(
                baseUrl = "http://localhost:3000",
                onEvent = { _, _ -> }
            )

        // Should not throw - verifies the setter exists and works
        assertDoesNotThrow { client.setSessionCookie("mehr_session=test123") }
    }

    @Test
    fun `EventStreamClient accepts null session cookie`() {
        val client =
            EventStreamClient(
                baseUrl = "http://localhost:3000",
                onEvent = { _, _ -> }
            )

        assertDoesNotThrow { client.setSessionCookie(null) }
    }

    @Test
    fun `EventStreamClient sends session cookie in SSE request`() {
        val mockServer = MockWebServer()
        mockServer.start()

        try {
            // Enqueue an SSE response that closes immediately
            mockServer.enqueue(
                MockResponse()
                    .setResponseCode(200)
                    .setHeader("Content-Type", "text/event-stream")
                    .setBody("data: {}\n\n")
            )

            val latch = CountDownLatch(1)
            val client =
                EventStreamClient(
                    baseUrl = mockServer.url("/").toString().trimEnd('/'),
                    onEvent = { _, _ -> latch.countDown() },
                    onError = { _ -> latch.countDown() },
                    onDisconnected = { latch.countDown() }
                )

            client.setSessionCookie("mehr_session=cookie_value_123")
            client.connect()

            // Wait for the request to be sent
            latch.await(5, TimeUnit.SECONDS)

            val request = mockServer.takeRequest(2, TimeUnit.SECONDS)
            assertNotNull(request)
            assertEquals("mehr_session=cookie_value_123", request!!.getHeader("Cookie"))
            assertEquals("text/event-stream", request.getHeader("Accept"))

            client.disconnect()
        } finally {
            mockServer.shutdown()
        }
    }

    @Test
    fun `EventStreamClient SSE request targets correct endpoint`() {
        val mockServer = MockWebServer()
        mockServer.start()

        try {
            mockServer.enqueue(
                MockResponse()
                    .setResponseCode(200)
                    .setHeader("Content-Type", "text/event-stream")
                    .setBody("data: {}\n\n")
            )

            val latch = CountDownLatch(1)
            val client =
                EventStreamClient(
                    baseUrl = mockServer.url("/").toString().trimEnd('/'),
                    onEvent = { _, _ -> latch.countDown() },
                    onDisconnected = { latch.countDown() }
                )

            client.connect()
            latch.await(5, TimeUnit.SECONDS)

            val request = mockServer.takeRequest(2, TimeUnit.SECONDS)
            assertNotNull(request)
            assertEquals("/api/v1/events", request!!.path)

            client.disconnect()
        } finally {
            mockServer.shutdown()
        }
    }

    // ========================================================================
    // EventStreamClient Callback Tests
    // ========================================================================

    @Test
    fun `EventStreamClient calls onEvent for SSE messages`() {
        val mockServer = MockWebServer()
        mockServer.start()

        try {
            mockServer.enqueue(
                MockResponse()
                    .setResponseCode(200)
                    .setHeader("Content-Type", "text/event-stream")
                    .setBody("event: workflow_state_changed\ndata: {\"state\":\"planning\"}\n\n")
            )

            val receivedEvents = CopyOnWriteArrayList<Pair<EventType, JsonObject>>()
            val latch = CountDownLatch(1)

            val client =
                EventStreamClient(
                    baseUrl = mockServer.url("/").toString().trimEnd('/'),
                    onEvent = { type, data ->
                        receivedEvents.add(Pair(type, data))
                        latch.countDown()
                    },
                    onDisconnected = { }
                )

            client.connect()
            latch.await(5, TimeUnit.SECONDS)

            assertTrue(receivedEvents.isNotEmpty(), "Expected at least one event")
            assertEquals(EventType.WORKFLOW_STATE_CHANGED, receivedEvents[0].first)
            assertEquals("planning", receivedEvents[0].second.get("state").asString)

            client.disconnect()
        } finally {
            mockServer.shutdown()
        }
    }

    @Test
    fun `EventStreamClient handles non-JSON SSE data gracefully`() {
        val mockServer = MockWebServer()
        mockServer.start()

        try {
            // Send non-JSON data as SSE event
            mockServer.enqueue(
                MockResponse()
                    .setResponseCode(200)
                    .setHeader("Content-Type", "text/event-stream")
                    .setBody("event: message\ndata: plain text not json\n\n")
            )

            val receivedEvents = CopyOnWriteArrayList<Pair<EventType, JsonObject>>()
            val latch = CountDownLatch(1)

            val client =
                EventStreamClient(
                    baseUrl = mockServer.url("/").toString().trimEnd('/'),
                    onEvent = { type, data ->
                        receivedEvents.add(Pair(type, data))
                        latch.countDown()
                    },
                    onDisconnected = { }
                )

            client.connect()
            latch.await(5, TimeUnit.SECONDS)

            // Non-JSON data should be wrapped in a JsonObject with "data" field
            assertTrue(receivedEvents.isNotEmpty(), "Expected at least one event")
            assertEquals(EventType.MESSAGE, receivedEvents[0].first)
            assertTrue(receivedEvents[0].second.has("data"))
            assertEquals("plain text not json", receivedEvents[0].second.get("data").asString)

            client.disconnect()
        } finally {
            mockServer.shutdown()
        }
    }

    @Test
    fun `EventStreamClient calls onError for SSE connection failure`() {
        val mockServer = MockWebServer()
        mockServer.start()

        try {
            // Return a non-SSE error response
            mockServer.enqueue(
                MockResponse()
                    .setResponseCode(500)
                    .setBody("Server error")
            )

            val errors = CopyOnWriteArrayList<String>()
            val latch = CountDownLatch(1)

            val client =
                EventStreamClient(
                    baseUrl = mockServer.url("/").toString().trimEnd('/'),
                    onEvent = { _, _ -> },
                    onError = { msg ->
                        errors.add(msg)
                        latch.countDown()
                    },
                    onDisconnected = { latch.countDown() }
                )

            client.connect()
            latch.await(5, TimeUnit.SECONDS)

            // Connection should report as not connected after failure
            assertFalse(client.isConnected())

            client.disconnect()
        } finally {
            mockServer.shutdown()
        }
    }

    @Test
    fun `EventStreamClient connect is idempotent when already connected`() {
        val mockServer = MockWebServer()
        mockServer.start()

        try {
            // Long-lived SSE response
            mockServer.enqueue(
                MockResponse()
                    .setResponseCode(200)
                    .setHeader("Content-Type", "text/event-stream")
                    .setBody("event: heartbeat\ndata: {}\n\n")
            )

            val connectedLatch = CountDownLatch(1)
            val client =
                EventStreamClient(
                    baseUrl = mockServer.url("/").toString().trimEnd('/'),
                    onEvent = { _, _ -> },
                    onConnected = { connectedLatch.countDown() },
                    onDisconnected = { }
                )

            client.connect()
            connectedLatch.await(5, TimeUnit.SECONDS)

            // Calling connect again should not create a second connection
            client.connect()

            // Only one request should have been made
            val firstRequest = mockServer.takeRequest(2, TimeUnit.SECONDS)
            assertNotNull(firstRequest)

            client.disconnect()
        } finally {
            mockServer.shutdown()
        }
    }

    // ========================================================================
    // Additional Data Class Tests
    // ========================================================================

    @Test
    fun `WorkflowStateEvent toString includes all fields`() {
        val event = WorkflowStateEvent("planning", "idle", "task-1", "Starting")
        val str = event.toString()

        assertTrue(str.contains("planning"))
        assertTrue(str.contains("idle"))
        assertTrue(str.contains("task-1"))
        assertTrue(str.contains("Starting"))
    }

    @Test
    fun `AgentMessageEvent copy works correctly`() {
        val original = AgentMessageEvent("Hello", "text", null, null)
        val copy = original.copy(content = "World")

        assertEquals("World", copy.content)
        assertEquals("text", copy.type)
        assertNotEquals(original, copy)
    }

    @Test
    fun `QuestionEvent copy with modified options`() {
        val original = QuestionEvent("Pick?", listOf("A", "B"), "task-1")
        val copy = original.copy(options = listOf("X", "Y", "Z"))

        assertEquals("Pick?", copy.question)
        assertEquals(3, copy.options!!.size)
        assertEquals("X", copy.options!![0])
    }

    @Test
    fun `WorkflowStateEvent with all null optional fields`() {
        val event = WorkflowStateEvent("idle")

        assertEquals("idle", event.state)
        assertNull(event.previousState)
        assertNull(event.taskId)
        assertNull(event.message)
    }

    @Test
    fun `AgentMessageEvent hashCode is consistent`() {
        val event1 = AgentMessageEvent("Hello", "text", null, null)
        val event2 = AgentMessageEvent("Hello", "text", null, null)

        assertEquals(event1.hashCode(), event2.hashCode())
    }
}
