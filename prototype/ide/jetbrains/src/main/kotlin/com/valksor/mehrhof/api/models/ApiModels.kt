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
    val status: String? = null
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

// ============================================================================
// Find Search Models
// ============================================================================

data class FindSearchResponse(
    val query: String,
    val count: Int,
    val matches: List<FindMatch>
)

data class FindMatch(
    val file: String,
    val line: Int,
    val snippet: String,
    val context: String? = null,
    val reason: String? = null
)

// ============================================================================
// Memory Models
// ============================================================================

data class MemorySearchResponse(
    val results: List<MemoryResult>,
    val count: Int
)

data class MemoryResult(
    @SerializedName("task_id")
    val taskId: String,
    val type: String,
    val score: Double,
    val content: String,
    val metadata: Map<String, Any>? = null
)

data class MemoryIndexRequest(
    @SerializedName("task_id")
    val taskId: String
)

data class MemoryIndexResponse(
    val success: Boolean,
    val message: String? = null,
    @SerializedName("task_id")
    val taskId: String? = null,
    val error: String? = null
)

data class MemoryStatsResponse(
    @SerializedName("total_documents")
    val totalDocuments: Int,
    @SerializedName("by_type")
    val byType: Map<String, Int>,
    val enabled: Boolean
)

// ============================================================================
// Library Models
// ============================================================================

data class LibraryListResponse(
    val collections: List<LibraryCollection>,
    val count: Int
)

data class LibraryCollection(
    val id: String,
    val name: String,
    val source: String,
    @SerializedName("source_type")
    val sourceType: String,
    @SerializedName("include_mode")
    val includeMode: String,
    @SerializedName("page_count")
    val pageCount: Int,
    @SerializedName("total_size")
    val totalSize: Long,
    val location: String,
    @SerializedName("pulled_at")
    val pulledAt: String? = null,
    val tags: List<String>? = null,
    val paths: List<String>? = null
)

data class LibraryShowResponse(
    val collection: LibraryCollection,
    val pages: List<String>
)

data class LibraryStatsResponse(
    @SerializedName("total_collections")
    val totalCollections: Int,
    @SerializedName("total_pages")
    val totalPages: Int,
    @SerializedName("total_size")
    val totalSize: Long,
    @SerializedName("project_count")
    val projectCount: Int,
    @SerializedName("shared_count")
    val sharedCount: Int,
    @SerializedName("by_mode")
    val byMode: Map<String, Int>,
    val enabled: Boolean
)

// ============================================================================
// Links Models
// ============================================================================

data class LinksListResponse(
    val links: List<LinkData>,
    val count: Int
)

data class LinkData(
    val source: String,
    val target: String,
    val context: String,
    @SerializedName("created_at")
    val createdAt: String
)

data class EntityLinksResponse(
    @SerializedName("entity_id")
    val entityId: String,
    val outgoing: List<LinkData>,
    val incoming: List<LinkData>
)

data class LinksSearchResponse(
    val query: String,
    val results: List<EntityResult>,
    val count: Int
)

data class EntityResult(
    @SerializedName("entity_id")
    val entityId: String,
    val type: String,
    val name: String? = null,
    @SerializedName("task_id")
    val taskId: String? = null,
    val id: String? = null,
    @SerializedName("full_type")
    val fullType: String? = null,
    @SerializedName("total_links")
    val totalLinks: Int? = null
)

data class LinksStatsResponse(
    @SerializedName("total_links")
    val totalLinks: Int,
    @SerializedName("total_sources")
    val totalSources: Int,
    @SerializedName("total_targets")
    val totalTargets: Int,
    @SerializedName("orphan_entities")
    val orphanEntities: Int,
    @SerializedName("most_linked")
    val mostLinked: List<EntityResult>,
    val enabled: Boolean
)

// ============================================================================
// Browser Models
// ============================================================================

data class BrowserStatusResponse(
    val connected: Boolean,
    val host: String? = null,
    val port: Int? = null,
    val tabs: List<BrowserTab>? = null,
    val error: String? = null
)

data class BrowserTab(
    val id: String,
    val title: String,
    val url: String
)

data class BrowserTabsResponse(
    val tabs: List<BrowserTab>,
    val count: Int
)

data class BrowserGotoRequest(
    val url: String
)

data class BrowserGotoResponse(
    val success: Boolean,
    val tab: BrowserTab? = null
)

data class BrowserNavigateRequest(
    @SerializedName("tab_id")
    val tabId: String? = null,
    val url: String
)

data class BrowserNavigateResponse(
    val success: Boolean,
    val message: String? = null
)

data class BrowserClickRequest(
    @SerializedName("tab_id")
    val tabId: String? = null,
    val selector: String
)

data class BrowserClickResponse(
    val success: Boolean,
    val selector: String? = null
)

data class BrowserTypeRequest(
    @SerializedName("tab_id")
    val tabId: String? = null,
    val selector: String,
    val text: String,
    val clear: Boolean = false
)

data class BrowserTypeResponse(
    val success: Boolean,
    val selector: String? = null
)

data class BrowserEvalRequest(
    @SerializedName("tab_id")
    val tabId: String? = null,
    val expression: String
)

data class BrowserEvalResponse(
    val success: Boolean,
    val result: Any? = null
)

data class BrowserScreenshotRequest(
    @SerializedName("tab_id")
    val tabId: String? = null,
    val format: String? = null,
    val quality: Int? = null,
    @SerializedName("full_page")
    val fullPage: Boolean = false
)

data class BrowserScreenshotResponse(
    val success: Boolean,
    val format: String? = null,
    val data: String? = null,
    val size: Int? = null,
    val encoding: String? = null
)

data class BrowserReloadRequest(
    @SerializedName("tab_id")
    val tabId: String? = null,
    val hard: Boolean = false
)

data class BrowserReloadResponse(
    val success: Boolean,
    val message: String? = null
)

data class BrowserCloseRequest(
    @SerializedName("tab_id")
    val tabId: String
)

data class BrowserCloseResponse(
    val success: Boolean,
    val message: String? = null
)

data class BrowserConsoleRequest(
    @SerializedName("tab_id")
    val tabId: String? = null,
    val duration: Int? = null,
    val level: String? = null
)

data class BrowserConsoleMessage(
    val level: String,
    val text: String,
    val timestamp: String? = null
)

data class BrowserConsoleResponse(
    val success: Boolean,
    val messages: List<BrowserConsoleMessage>? = null,
    val count: Int? = null
)

data class BrowserNetworkRequest(
    @SerializedName("tab_id")
    val tabId: String? = null,
    val duration: Int? = null,
    @SerializedName("capture_body")
    val captureBody: Boolean = false,
    @SerializedName("max_body_size")
    val maxBodySize: Int? = null
)

data class BrowserNetworkEntry(
    val method: String,
    val url: String,
    val status: Int? = null,
    @SerializedName("status_text")
    val statusText: String? = null,
    val timestamp: String,
    @SerializedName("request_body")
    val requestBody: String? = null,
    @SerializedName("response_body")
    val responseBody: String? = null
)

data class BrowserNetworkResponse(
    val success: Boolean,
    val requests: List<BrowserNetworkEntry>? = null,
    val count: Int? = null
)

// ============================================================================
// Project Models
// ============================================================================

data class ProjectPlanRequest(
    val source: String,
    val title: String? = null,
    val instructions: String? = null
)

data class ProjectPlanResponse(
    val success: Boolean,
    @SerializedName("queue_id")
    val queueId: String? = null,
    @SerializedName("task_count")
    val taskCount: Int? = null,
    val questions: List<String>? = null,
    val error: String? = null
)

data class ProjectTasksResponse(
    @SerializedName("queue_id")
    val queueId: String,
    val tasks: List<ProjectQueueTask>,
    val count: Int
)

data class ProjectQueueTask(
    val id: String,
    val title: String,
    val status: String,
    val priority: Int,
    @SerializedName("parent_id")
    val parentId: String? = null,
    @SerializedName("depends_on")
    val dependsOn: List<String>? = null
)

data class ProjectSubmitRequest(
    val provider: String,
    @SerializedName("queue_id")
    val queueId: String? = null,
    @SerializedName("create_epic")
    val createEpic: Boolean = false,
    val labels: List<String>? = null
)

data class ProjectSubmitResponse(
    val success: Boolean,
    @SerializedName("submitted_count")
    val submittedCount: Int? = null,
    val tasks: List<ProjectSubmittedTask>? = null,
    val error: String? = null
)

data class ProjectSubmittedTask(
    @SerializedName("local_id")
    val localId: String,
    @SerializedName("external_id")
    val externalId: String,
    val title: String
)

data class ProjectSyncRequest(
    val reference: String
)

data class ProjectSyncResponse(
    val success: Boolean,
    @SerializedName("queue_id")
    val queueId: String? = null,
    @SerializedName("queue_title")
    val queueTitle: String? = null,
    @SerializedName("tasks_synced")
    val tasksSynced: Int? = null,
    val error: String? = null
)

// ============================================================================
// Stack Models
// ============================================================================

data class StackListResponse(
    val stacks: List<StackInfo>,
    val count: Int
)

data class StackInfo(
    val id: String,
    @SerializedName("task_count")
    val taskCount: Int,
    val tasks: List<StackTask>
)

data class StackTask(
    val id: String,
    val branch: String,
    val state: String,
    @SerializedName("depends_on")
    val dependsOn: String? = null,
    @SerializedName("pr_number")
    val prNumber: Int? = null
)

data class StackRebaseResponse(
    val success: Boolean,
    @SerializedName("rebased_count")
    val rebasedCount: Int? = null,
    @SerializedName("rebased_tasks")
    val rebasedTasks: List<String>? = null,
    val error: String? = null
)

data class StackSyncResponse(
    val success: Boolean,
    @SerializedName("updated_count")
    val updatedCount: Int? = null,
    val error: String? = null
)
