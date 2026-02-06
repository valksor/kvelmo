package com.valksor.mehrhof.api.models

import com.google.gson.annotations.SerializedName

// ============================================================================
// Status & Task Models
// ============================================================================

data class StatusResponse(
    val mode: String,
    val running: Boolean,
    val port: Int,
    val state: String? = null
)

data class TaskResponse(
    val active: Boolean,
    val task: TaskInfo? = null,
    val work: TaskWork? = null,
    @SerializedName("pending_question")
    val pendingQuestion: PendingQuestion? = null
)

data class TaskInfo(
    val id: String,
    val state: String,
    val ref: String,
    val branch: String? = null,
    @SerializedName("worktree_path")
    val worktreePath: String? = null,
    val started: String? = null
)

data class TaskWork(
    val title: String? = null,
    @SerializedName("external_key")
    val externalKey: String? = null,
    @SerializedName("created_at")
    val createdAt: String? = null,
    @SerializedName("updated_at")
    val updatedAt: String? = null,
    val costs: CostInfo? = null
)

data class PendingQuestion(
    val question: String,
    val options: List<String>? = null
)

data class TaskListResponse(
    val tasks: List<TaskSummary>,
    val count: Int
)

data class TaskSummary(
    val id: String,
    val title: String? = null,
    val state: String,
    @SerializedName("created_at")
    val createdAt: String? = null,
    @SerializedName("worktree_path")
    val worktreePath: String? = null
)

// ============================================================================
// Workflow Models
// ============================================================================

data class WorkflowResponse(
    val success: Boolean,
    val state: String? = null,
    val message: String? = null,
    val error: String? = null
)

data class ContinueResponse(
    val success: Boolean,
    val state: String,
    val action: String? = null,
    @SerializedName("next_actions")
    val nextActions: List<String>,
    val message: String
)

data class WorkflowRequest(
    val agent: String? = null
)

data class FinishRequest(
    @SerializedName("squash_merge")
    val squashMerge: Boolean = false,
    @SerializedName("delete_branch")
    val deleteBranch: Boolean = true,
    @SerializedName("target_branch")
    val targetBranch: String? = null,
    @SerializedName("push_after")
    val pushAfter: Boolean = true,
    @SerializedName("force_merge")
    val forceMerge: Boolean = false,
    @SerializedName("draft_pr")
    val draftPr: Boolean = false,
    @SerializedName("pr_title")
    val prTitle: String? = null,
    @SerializedName("pr_body")
    val prBody: String? = null
)

data class AnswerRequest(
    val answer: String
)

data class AddNoteRequest(
    val message: String
)

data class AddNoteResponse(
    val success: Boolean,
    @SerializedName("note_number")
    val noteNumber: Int? = null,
    val error: String? = null
)

data class QuestionRequest(
    val message: String
)

data class StartTaskRequest(
    val ref: String? = null,
    val content: String? = null
)

// ============================================================================
// Guide Models
// ============================================================================

data class GuideResponse(
    @SerializedName("has_task")
    val hasTask: Boolean,
    @SerializedName("task_id")
    val taskId: String? = null,
    val title: String? = null,
    val state: String? = null,
    val specifications: Int,
    @SerializedName("pending_question")
    val pendingQuestion: PendingQuestionInfo? = null,
    @SerializedName("next_actions")
    val nextActions: List<GuideAction>
)

data class PendingQuestionInfo(
    val question: String,
    val options: List<String>? = null
)

data class GuideAction(
    val command: String,
    val description: String,
    val endpoint: String? = null
)

// ============================================================================
// Cost Models
// ============================================================================

data class CostInfo(
    @SerializedName("total_tokens")
    val totalTokens: Int = 0,
    @SerializedName("input_tokens")
    val inputTokens: Int = 0,
    @SerializedName("output_tokens")
    val outputTokens: Int = 0,
    @SerializedName("cached_tokens")
    val cachedTokens: Int = 0,
    @SerializedName("total_cost_usd")
    val totalCostUsd: Double = 0.0
)

data class TaskCostResponse(
    @SerializedName("task_id")
    val taskId: String,
    val title: String? = null,
    @SerializedName("total_tokens")
    val totalTokens: Int,
    @SerializedName("input_tokens")
    val inputTokens: Int,
    @SerializedName("output_tokens")
    val outputTokens: Int,
    @SerializedName("cached_tokens")
    val cachedTokens: Int,
    @SerializedName("cached_percent")
    val cachedPercent: Double? = null,
    @SerializedName("total_cost_usd")
    val totalCostUsd: Double,
    @SerializedName("by_step")
    val byStep: Map<String, StepCost>? = null
)

data class StepCost(
    @SerializedName("input_tokens")
    val inputTokens: Int,
    @SerializedName("output_tokens")
    val outputTokens: Int,
    @SerializedName("cached_tokens")
    val cachedTokens: Int,
    @SerializedName("total_tokens")
    val totalTokens: Int,
    @SerializedName("cost_usd")
    val costUsd: Double,
    val calls: Int
)

data class AllCostsResponse(
    val tasks: List<TaskCostResponse>,
    @SerializedName("grand_total")
    val grandTotal: GrandTotal,
    val monthly: MonthlyBudgetInfo? = null
)

data class GrandTotal(
    @SerializedName("input_tokens")
    val inputTokens: Int,
    @SerializedName("output_tokens")
    val outputTokens: Int,
    @SerializedName("total_tokens")
    val totalTokens: Int,
    @SerializedName("cached_tokens")
    val cachedTokens: Int,
    @SerializedName("cost_usd")
    val costUsd: Double
)

data class MonthlyBudgetInfo(
    val month: String,
    val spent: Double,
    @SerializedName("max_cost")
    val maxCost: Double? = null,
    @SerializedName("warning_at")
    val warningAt: Double? = null,
    @SerializedName("warning_sent")
    val warningSent: Boolean? = null
)

// ============================================================================
// Specification Models
// ============================================================================

data class SpecificationsResponse(
    val specifications: List<Specification>
)

data class Specification(
    val id: Int,
    val title: String,
    val content: String,
    @SerializedName("created_at")
    val createdAt: String? = null,
    val status: String? = null,
    @SerializedName("implemented_files")
    val implementedFiles: List<String>? = null
)

// ============================================================================
// Session Models
// ============================================================================

data class SessionsResponse(
    val sessions: List<Session>
)

data class Session(
    val id: String,
    val step: String,
    @SerializedName("started_at")
    val startedAt: String,
    @SerializedName("ended_at")
    val endedAt: String? = null,
    val status: String? = null
)

// ============================================================================
// Agent & Provider Models
// ============================================================================

data class AgentsListResponse(
    val agents: List<AgentInfo>,
    val count: Int
)

data class AgentInfo(
    val name: String,
    val type: String,
    val extends: String? = null,
    val description: String? = null,
    val version: String? = null,
    val available: Boolean,
    val capabilities: AgentCapabilities? = null,
    val models: List<AgentModel>? = null
)

data class AgentCapabilities(
    val streaming: Boolean,
    @SerializedName("tool_use")
    val toolUse: Boolean,
    @SerializedName("file_operations")
    val fileOperations: Boolean,
    @SerializedName("code_execution")
    val codeExecution: Boolean,
    @SerializedName("multi_turn")
    val multiTurn: Boolean,
    @SerializedName("system_prompt")
    val systemPrompt: Boolean,
    @SerializedName("allowed_tools")
    val allowedTools: List<String>? = null
)

data class AgentModel(
    val id: String,
    val name: String,
    val default: Boolean,
    @SerializedName("max_tokens")
    val maxTokens: Int? = null,
    @SerializedName("input_cost_usd")
    val inputCostUsd: Double? = null,
    @SerializedName("output_cost_usd")
    val outputCostUsd: Double? = null
)

data class ProvidersListResponse(
    val providers: List<ProviderInfo>,
    val count: Int
)

data class ProviderInfo(
    val scheme: String,
    val shorthand: String? = null,
    val name: String,
    val description: String,
    @SerializedName("env_vars")
    val envVars: List<String>? = null
)

// ============================================================================
// Interactive API Models
// ============================================================================

data class InteractiveCommandRequest(
    val command: String,
    val args: List<String> = emptyList()
)

data class InteractiveCommandResponse(
    val success: Boolean,
    val message: String? = null,
    val state: String? = null,
    val error: String? = null
)

data class InteractiveChatRequest(
    val message: String
)

data class InteractiveChatMessage(
    val role: String,
    val content: String,
    val timestamp: String? = null
)

data class InteractiveChatResponse(
    val success: Boolean,
    val message: String? = null,
    val messages: List<InteractiveChatMessage>? = null,
    val error: String? = null
)

data class InteractiveAnswerRequest(
    val response: String
)

data class InteractiveStateResponse(
    val success: Boolean,
    val state: String? = null,
    @SerializedName("task_id")
    val taskId: String? = null,
    val title: String? = null,
    val error: String? = null
)

data class InteractiveStopResponse(
    val success: Boolean,
    val message: String? = null,
    val error: String? = null
)

// ============================================================================
// Command Discovery Models
// ============================================================================

data class CommandArg(
    val name: String,
    val required: Boolean,
    val description: String? = null
)

data class CommandInfo(
    val name: String,
    val aliases: List<String>? = null,
    val description: String,
    val category: String,
    val args: List<CommandArg>? = null,
    @SerializedName("requires_task")
    val requiresTask: Boolean,
    val subcommands: List<String>? = null
)

data class CommandsResponse(
    val commands: List<CommandInfo>
)

// ============================================================================
// Error Response
// ============================================================================

data class ErrorResponse(
    val error: String
)

// ============================================================================
// Queue Task Models (for quick tasks)
// ============================================================================

data class DeleteQueueTaskResponse(
    val success: Boolean,
    val message: String? = null,
    val error: String? = null
)

data class ExportQueueTaskResponse(
    val success: Boolean,
    val message: String? = null,
    val markdown: String? = null,
    val error: String? = null
)

data class OptimizeQueueTaskResponse(
    val success: Boolean,
    val message: String? = null,
    @SerializedName("original_title")
    val originalTitle: String? = null,
    @SerializedName("optimized_title")
    val optimizedTitle: String? = null,
    @SerializedName("added_labels")
    val addedLabels: List<String>? = null,
    @SerializedName("improvement_notes")
    val improvementNotes: List<String>? = null,
    val error: String? = null
)

data class SubmitQueueTaskResponse(
    val success: Boolean,
    val message: String? = null,
    @SerializedName("external_id")
    val externalId: String? = null,
    val url: String? = null,
    val error: String? = null
)

data class SyncTaskResponse(
    val success: Boolean,
    val message: String? = null,
    val error: String? = null
)
