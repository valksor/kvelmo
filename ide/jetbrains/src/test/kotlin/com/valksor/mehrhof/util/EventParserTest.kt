package com.valksor.mehrhof.util

import com.google.gson.JsonArray
import com.google.gson.JsonNull
import com.google.gson.JsonObject
import com.valksor.mehrhof.api.EventType
import org.junit.jupiter.api.Assertions.*
import org.junit.jupiter.api.Test

class EventParserTest {
    // ========================================================================
    // WORKFLOW_STATE_CHANGED tests
    // ========================================================================

    @Test
    fun `WORKFLOW_STATE_CHANGED extracts state string`() {
        val data = JsonObject().apply { addProperty("state", "planning") }
        val result = EventParser.parse(EventType.WORKFLOW_STATE_CHANGED, data)

        assertTrue(result is EventParser.ParsedEvent.WorkflowStateChanged)
        assertEquals("planning", (result as EventParser.ParsedEvent.WorkflowStateChanged).newState)
    }

    @Test
    fun `WORKFLOW_STATE_CHANGED with implementing state`() {
        val data = JsonObject().apply { addProperty("state", "implementing") }
        val result = EventParser.parse(EventType.WORKFLOW_STATE_CHANGED, data)

        assertTrue(result is EventParser.ParsedEvent.WorkflowStateChanged)
        assertEquals("implementing", (result as EventParser.ParsedEvent.WorkflowStateChanged).newState)
    }

    @Test
    fun `WORKFLOW_STATE_CHANGED with missing state returns Ignored`() {
        val data = JsonObject()
        val result = EventParser.parse(EventType.WORKFLOW_STATE_CHANGED, data)

        assertTrue(result is EventParser.ParsedEvent.Ignored)
    }

    @Test
    fun `WORKFLOW_STATE_CHANGED with null state returns Ignored`() {
        val data = JsonObject().apply { add("state", JsonNull.INSTANCE) }
        val result = EventParser.parse(EventType.WORKFLOW_STATE_CHANGED, data)

        assertTrue(result is EventParser.ParsedEvent.Ignored)
    }

    // ========================================================================
    // Task lifecycle tests
    // ========================================================================

    @Test
    fun `TASK_STARTED returns TaskLifecycleEvent`() {
        val data = JsonObject()
        val result = EventParser.parse(EventType.TASK_STARTED, data)

        assertTrue(result is EventParser.ParsedEvent.TaskLifecycleEvent)
    }

    @Test
    fun `TASK_COMPLETED returns TaskLifecycleEvent`() {
        val data = JsonObject()
        val result = EventParser.parse(EventType.TASK_COMPLETED, data)

        assertTrue(result is EventParser.ParsedEvent.TaskLifecycleEvent)
    }

    @Test
    fun `TASK_FAILED returns TaskLifecycleEvent`() {
        val data = JsonObject()
        val result = EventParser.parse(EventType.TASK_FAILED, data)

        assertTrue(result is EventParser.ParsedEvent.TaskLifecycleEvent)
    }

    // ========================================================================
    // QUESTION_ASKED tests
    // ========================================================================

    @Test
    fun `QUESTION_ASKED extracts question and options`() {
        val options =
            JsonArray().apply {
                add("Yes")
                add("No")
                add("Maybe")
            }
        val data =
            JsonObject().apply {
                addProperty("question", "Should we proceed?")
                add("options", options)
            }
        val result = EventParser.parse(EventType.QUESTION_ASKED, data)

        assertTrue(result is EventParser.ParsedEvent.QuestionAsked)
        val qa = result as EventParser.ParsedEvent.QuestionAsked
        assertEquals("Should we proceed?", qa.question)
        assertEquals(listOf("Yes", "No", "Maybe"), qa.options)
    }

    @Test
    fun `QUESTION_ASKED without options sets options to null`() {
        val data =
            JsonObject().apply {
                addProperty("question", "What do you think?")
            }
        val result = EventParser.parse(EventType.QUESTION_ASKED, data)

        assertTrue(result is EventParser.ParsedEvent.QuestionAsked)
        val qa = result as EventParser.ParsedEvent.QuestionAsked
        assertEquals("What do you think?", qa.question)
        assertNull(qa.options)
    }

    @Test
    fun `QUESTION_ASKED with missing question returns Ignored`() {
        val data =
            JsonObject().apply {
                add("options", JsonArray())
            }
        val result = EventParser.parse(EventType.QUESTION_ASKED, data)

        assertTrue(result is EventParser.ParsedEvent.Ignored)
    }

    // ========================================================================
    // ANSWER_PROVIDED tests
    // ========================================================================

    @Test
    fun `ANSWER_PROVIDED returns AnswerProvided`() {
        val data = JsonObject()
        val result = EventParser.parse(EventType.ANSWER_PROVIDED, data)

        assertTrue(result is EventParser.ParsedEvent.AnswerProvided)
    }

    // ========================================================================
    // AGENT_MESSAGE tests
    // ========================================================================

    @Test
    fun `AGENT_MESSAGE extracts content and type`() {
        val data =
            JsonObject().apply {
                addProperty("content", "Working on the implementation...")
                addProperty("type", "progress")
            }
        val result = EventParser.parse(EventType.AGENT_MESSAGE, data)

        assertTrue(result is EventParser.ParsedEvent.AgentMessage)
        val msg = result as EventParser.ParsedEvent.AgentMessage
        assertEquals("Working on the implementation...", msg.content)
        assertEquals("progress", msg.type)
    }

    @Test
    fun `AGENT_MESSAGE without type sets type to null`() {
        val data =
            JsonObject().apply {
                addProperty("content", "Done")
            }
        val result = EventParser.parse(EventType.AGENT_MESSAGE, data)

        assertTrue(result is EventParser.ParsedEvent.AgentMessage)
        val msg = result as EventParser.ParsedEvent.AgentMessage
        assertEquals("Done", msg.content)
        assertNull(msg.type)
    }

    @Test
    fun `AGENT_MESSAGE with missing content returns Ignored`() {
        val data =
            JsonObject().apply {
                addProperty("type", "progress")
            }
        val result = EventParser.parse(EventType.AGENT_MESSAGE, data)

        assertTrue(result is EventParser.ParsedEvent.Ignored)
    }

    // ========================================================================
    // ERROR tests
    // ========================================================================

    @Test
    fun `ERROR extracts from error field`() {
        val data =
            JsonObject().apply {
                addProperty("error", "Something went wrong")
            }
        val result = EventParser.parse(EventType.ERROR, data)

        assertTrue(result is EventParser.ParsedEvent.Error)
        assertEquals("Something went wrong", (result as EventParser.ParsedEvent.Error).message)
    }

    @Test
    fun `ERROR falls back to message field`() {
        val data =
            JsonObject().apply {
                addProperty("message", "Connection lost")
            }
        val result = EventParser.parse(EventType.ERROR, data)

        assertTrue(result is EventParser.ParsedEvent.Error)
        assertEquals("Connection lost", (result as EventParser.ParsedEvent.Error).message)
    }

    @Test
    fun `ERROR prefers error field over message field`() {
        val data =
            JsonObject().apply {
                addProperty("error", "Primary error")
                addProperty("message", "Secondary message")
            }
        val result = EventParser.parse(EventType.ERROR, data)

        assertTrue(result is EventParser.ParsedEvent.Error)
        assertEquals("Primary error", (result as EventParser.ParsedEvent.Error).message)
    }

    @Test
    fun `ERROR with no fields returns Unknown error`() {
        val data = JsonObject()
        val result = EventParser.parse(EventType.ERROR, data)

        assertTrue(result is EventParser.ParsedEvent.Error)
        assertEquals("Unknown error", (result as EventParser.ParsedEvent.Error).message)
    }

    // ========================================================================
    // Unknown event tests
    // ========================================================================

    @Test
    fun `HEARTBEAT returns Ignored`() {
        val data = JsonObject()
        val result = EventParser.parse(EventType.HEARTBEAT, data)

        assertTrue(result is EventParser.ParsedEvent.Ignored)
    }

    @Test
    fun `MESSAGE returns Ignored`() {
        val data = JsonObject()
        val result = EventParser.parse(EventType.MESSAGE, data)

        assertTrue(result is EventParser.ParsedEvent.Ignored)
    }

    @Test
    fun `UNKNOWN returns Ignored`() {
        val data = JsonObject()
        val result = EventParser.parse(EventType.UNKNOWN, data)

        assertTrue(result is EventParser.ParsedEvent.Ignored)
    }

    @Test
    fun `AGENT_STARTED returns Ignored`() {
        val data = JsonObject()
        val result = EventParser.parse(EventType.AGENT_STARTED, data)

        assertTrue(result is EventParser.ParsedEvent.Ignored)
    }

    @Test
    fun `AGENT_COMPLETED returns Ignored`() {
        val data = JsonObject()
        val result = EventParser.parse(EventType.AGENT_COMPLETED, data)

        assertTrue(result is EventParser.ParsedEvent.Ignored)
    }

    @Test
    fun `CODE_CHANGE returns Ignored`() {
        val data = JsonObject()
        val result = EventParser.parse(EventType.CODE_CHANGE, data)

        assertTrue(result is EventParser.ParsedEvent.Ignored)
    }

    @Test
    fun `SPECIFICATION_CREATED returns Ignored`() {
        val data = JsonObject()
        val result = EventParser.parse(EventType.SPECIFICATION_CREATED, data)

        assertTrue(result is EventParser.ParsedEvent.Ignored)
    }
}
