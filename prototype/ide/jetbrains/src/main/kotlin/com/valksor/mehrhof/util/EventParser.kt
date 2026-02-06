package com.valksor.mehrhof.util

import com.google.gson.JsonObject
import com.valksor.mehrhof.api.EventType

/**
 * Parses SSE event data into structured results.
 * Extracts pure parsing logic from MehrhofProjectService for testability.
 */
object EventParser {
    /** Safely get a string value from a JsonObject, returning null for missing keys and JSON nulls. */
    private fun JsonObject.getString(key: String): String? = get(key)?.takeIf { !it.isJsonNull }?.asString

    /**
     * Result of parsing an SSE event.
     */
    sealed class ParsedEvent {
        data class WorkflowStateChanged(
            val newState: String
        ) : ParsedEvent()

        data object TaskLifecycleEvent : ParsedEvent()

        data class QuestionAsked(
            val question: String,
            val options: List<String>?
        ) : ParsedEvent()

        data object AnswerProvided : ParsedEvent()

        data class AgentMessage(
            val content: String,
            val type: String?
        ) : ParsedEvent()

        data class Error(
            val message: String
        ) : ParsedEvent()

        data object Ignored : ParsedEvent()
    }

    /**
     * Parse an SSE event type and JSON data into a structured result.
     *
     * @param eventType The type of SSE event received.
     * @param data The JSON payload of the event.
     * @return A [ParsedEvent] representing the parsed data.
     */
    fun parse(
        eventType: EventType,
        data: JsonObject
    ): ParsedEvent =
        when (eventType) {
            EventType.WORKFLOW_STATE_CHANGED -> {
                val state =
                    data.getString("state")
                        ?: return ParsedEvent.Ignored
                ParsedEvent.WorkflowStateChanged(state)
            }

            EventType.TASK_STARTED,
            EventType.TASK_COMPLETED,
            EventType.TASK_FAILED -> ParsedEvent.TaskLifecycleEvent

            EventType.QUESTION_ASKED -> {
                val question =
                    data.getString("question")
                        ?: return ParsedEvent.Ignored
                val options = data.getAsJsonArray("options")?.map { it.asString }
                ParsedEvent.QuestionAsked(question, options)
            }

            EventType.ANSWER_PROVIDED -> ParsedEvent.AnswerProvided

            EventType.AGENT_MESSAGE -> {
                val content =
                    data.getString("content")
                        ?: return ParsedEvent.Ignored
                val type = data.getString("type")
                ParsedEvent.AgentMessage(content, type)
            }

            EventType.ERROR -> {
                val error =
                    data.getString("error")
                        ?: data.getString("message")
                        ?: "Unknown error"
                ParsedEvent.Error(error)
            }

            else -> ParsedEvent.Ignored
        }
}
