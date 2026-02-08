package com.valksor.mehrhof.api.models

import com.google.gson.annotations.SerializedName

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

// ============================================================================
// Budget Models
// ============================================================================

data class BudgetStatusResponse(
    val enabled: Boolean,
    @SerializedName("max_cost")
    val maxCost: Double? = null,
    val spent: Double? = null,
    val remaining: Double? = null,
    val currency: String? = null,
    @SerializedName("warning_at")
    val warningAt: Double? = null,
    val warned: Boolean? = null,
    @SerializedName("limit_hit")
    val limitHit: Boolean? = null
)

// ============================================================================
// Label Models
// ============================================================================

data class LabelsListResponse(
    val labels: List<String>,
    val count: Int
)
