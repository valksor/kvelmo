package com.valksor.mehrhof.api

import com.google.gson.Gson
import com.google.gson.GsonBuilder
import com.google.gson.JsonObject
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.Response
import okhttp3.sse.EventSource
import okhttp3.sse.EventSourceListener
import okhttp3.sse.EventSources
import java.util.concurrent.TimeUnit
import java.util.concurrent.atomic.AtomicBoolean

/**
 * SSE (Server-Sent Events) client for real-time updates from the Mehrhof server.
 *
 * Connects to /api/v1/events for workflow state changes and agent output streaming.
 */
class EventStreamClient(
    private val baseUrl: String,
    private val onEvent: (EventType, JsonObject) -> Unit,
    private val onError: (String) -> Unit = {},
    private val onConnected: () -> Unit = {},
    private val onDisconnected: () -> Unit = {}
) {
    private val client =
        OkHttpClient
            .Builder()
            .connectTimeout(10, TimeUnit.SECONDS)
            .readTimeout(0, TimeUnit.SECONDS) // No timeout for SSE
            .build()

    private val gson: Gson = GsonBuilder().setLenient().create()

    private var eventSource: EventSource? = null
    private val connected = AtomicBoolean(false)
    private val intentionalDisconnect = AtomicBoolean(false)

    @Volatile
    private var sessionCookie: String? = null

    fun setSessionCookie(cookie: String?) {
        this.sessionCookie = cookie
    }

    /**
     * Connect to the SSE event stream.
     * Call this from a background thread or use coroutines.
     */
    fun connect() {
        if (connected.get()) {
            return
        }

        val sseUrl = "$baseUrl/api/v1/events"
        val requestBuilder =
            Request
                .Builder()
                .url(sseUrl)
                .header("Accept", "text/event-stream")

        sessionCookie?.let { requestBuilder.addHeader("Cookie", it) }

        val request = requestBuilder.build()

        println("Connecting to SSE: $sseUrl")

        val listener =
            object : EventSourceListener() {
                override fun onOpen(
                    eventSource: EventSource,
                    response: Response
                ) {
                    connected.set(true)
                    onConnected()
                }

                override fun onEvent(
                    eventSource: EventSource,
                    id: String?,
                    type: String?,
                    data: String
                ) {
                    try {
                        val eventType = EventType.fromString(type ?: "message")
                        val jsonData =
                            try {
                                gson.fromJson(data, JsonObject::class.java) ?: JsonObject()
                            } catch (_: Exception) {
                                // If data is not JSON, wrap it
                                JsonObject().apply { addProperty("data", data) }
                            }
                        onEvent(eventType, jsonData)
                    } catch (e: Exception) {
                        onError("Failed to parse event: ${e.message}")
                    }
                }

                override fun onClosed(eventSource: EventSource) {
                    connected.set(false)
                    onDisconnected()
                }

                override fun onFailure(
                    eventSource: EventSource,
                    t: Throwable?,
                    response: Response?
                ) {
                    connected.set(false)
                    // Only report error if not an intentional disconnect
                    if (!intentionalDisconnect.getAndSet(false)) {
                        // Don't report EOF on successful responses as error - it's just a graceful close
                        val isGracefulClose =
                            response?.isSuccessful == true &&
                                (t is java.io.EOFException || t?.cause is java.io.EOFException)

                        if (!isGracefulClose) {
                            val errorMsg =
                                buildString {
                                    if (response != null) {
                                        append("HTTP ${response.code}")
                                        if (response.message.isNotEmpty()) append(" ${response.message}")
                                        try {
                                            response.body?.string()?.takeIf { it.isNotBlank() }?.let { body ->
                                                append(" - $body")
                                            }
                                        } catch (_: Exception) {
                                            // ignore
                                        }
                                    }
                                    if (t != null) {
                                        if (isNotEmpty()) append(": ")
                                        append(t.javaClass.simpleName)
                                        t.message?.let { append(": $it") }
                                    }
                                    if (isEmpty()) append("Unknown error")
                                }
                            onError("SSE connection failed: $errorMsg")
                        }
                    }
                    onDisconnected()
                }
            }

        val factory = EventSources.createFactory(client)
        eventSource = factory.newEventSource(request, listener)
    }

    /**
     * Disconnect from the SSE event stream.
     */
    fun disconnect() {
        intentionalDisconnect.set(true)
        eventSource?.cancel()
        eventSource = null
        connected.set(false)
    }

    /**
     * Check if connected to the event stream.
     */
    fun isConnected(): Boolean = connected.get()

    /**
     * Reconnect to the event stream.
     */
    fun reconnect() {
        disconnect()
        connect()
    }
}

/**
 * Types of events received from the SSE stream.
 */
enum class EventType {
    // Workflow state events
    WORKFLOW_STATE_CHANGED,
    TASK_STARTED,
    TASK_COMPLETED,
    TASK_FAILED,

    // Agent events
    AGENT_MESSAGE,
    AGENT_STARTED,
    AGENT_COMPLETED,
    AGENT_ERROR,

    // Specification events
    SPECIFICATION_CREATED,
    SPECIFICATION_UPDATED,

    // Code change events
    CODE_CHANGE,

    // Question events
    QUESTION_ASKED,
    ANSWER_PROVIDED,

    // Generic events
    MESSAGE,
    HEARTBEAT,
    ERROR,
    UNKNOWN;

    companion object {
        fun fromString(type: String): EventType =
            when (type.lowercase().replace("-", "_")) {
                "workflow_state_changed", "state_changed" -> WORKFLOW_STATE_CHANGED
                "task_started" -> TASK_STARTED
                "task_completed" -> TASK_COMPLETED
                "task_failed" -> TASK_FAILED
                "agent_message", "agent_output" -> AGENT_MESSAGE
                "agent_started" -> AGENT_STARTED
                "agent_completed" -> AGENT_COMPLETED
                "agent_error" -> AGENT_ERROR
                "specification_created", "spec_created" -> SPECIFICATION_CREATED
                "specification_updated", "spec_updated" -> SPECIFICATION_UPDATED
                "code_change" -> CODE_CHANGE
                "question_asked", "question" -> QUESTION_ASKED
                "answer_provided", "answer" -> ANSWER_PROVIDED
                "message" -> MESSAGE
                "heartbeat", "ping" -> HEARTBEAT
                "error" -> ERROR
                else -> UNKNOWN
            }
    }
}

/**
 * Data class representing a workflow state change event.
 */
data class WorkflowStateEvent(
    val state: String,
    val previousState: String? = null,
    val taskId: String? = null,
    val message: String? = null
)

/**
 * Data class representing an agent message event.
 */
data class AgentMessageEvent(
    val content: String,
    val type: String? = null, // "text", "tool_use", "tool_result", etc.
    val toolName: String? = null,
    val toolInput: String? = null
)

/**
 * Data class representing a question event.
 */
data class QuestionEvent(
    val question: String,
    val options: List<String>? = null,
    val taskId: String? = null
)

/**
 * Extension functions for parsing event data.
 */
fun JsonObject.toWorkflowStateEvent(gson: Gson): WorkflowStateEvent? =
    try {
        gson.fromJson(this, WorkflowStateEvent::class.java)
    } catch (_: Exception) {
        null
    }

fun JsonObject.toAgentMessageEvent(gson: Gson): AgentMessageEvent? =
    try {
        gson.fromJson(this, AgentMessageEvent::class.java)
    } catch (_: Exception) {
        null
    }

fun JsonObject.toQuestionEvent(gson: Gson): QuestionEvent? =
    try {
        gson.fromJson(this, QuestionEvent::class.java)
    } catch (_: Exception) {
        null
    }
