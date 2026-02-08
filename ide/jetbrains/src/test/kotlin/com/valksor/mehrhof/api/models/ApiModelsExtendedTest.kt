package com.valksor.mehrhof.api.models

import com.google.gson.Gson
import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.Assertions.assertNull
import org.junit.jupiter.api.Assertions.assertTrue
import org.junit.jupiter.api.Test

/**
 * Unit tests for extended API model data classes.
 *
 * Tests JSON serialization/deserialization for Memory, Library, Links, and Project models.
 */
class ApiModelsExtendedTest {
    private val gson = Gson()

    // ========================================================================
    // Find Search Models
    // ========================================================================

    @Test
    fun `FindSearchResponse deserializes correctly`() {
        val json =
            """{
            "query": "function foo",
            "count": 2,
            "matches": [
                {"file": "src/main.kt", "line": 42, "snippet": "fun foo() {", "reason": "exact match"},
                {"file": "src/util.kt", "line": 10, "snippet": "fun fooBar() {"}
            ]
        }"""

        val response = gson.fromJson(json, FindSearchResponse::class.java)

        assertEquals("function foo", response.query)
        assertEquals(2, response.count)
        assertEquals(2, response.matches.size)
        assertEquals("src/main.kt", response.matches[0].file)
        assertEquals(42, response.matches[0].line)
        assertEquals("exact match", response.matches[0].reason)
        assertNull(response.matches[1].reason)
    }

    // ========================================================================
    // Memory Models
    // ========================================================================

    @Test
    fun `MemorySearchResponse deserializes correctly`() {
        val json =
            """{
            "results": [
                {"task_id": "t1", "type": "spec", "score": 0.95, "content": "Auth spec"}
            ],
            "count": 1
        }"""

        val response = gson.fromJson(json, MemorySearchResponse::class.java)

        assertEquals(1, response.count)
        assertEquals(1, response.results.size)
        assertEquals("t1", response.results[0].taskId)
        assertEquals("spec", response.results[0].type)
        assertEquals(0.95, response.results[0].score, 0.01)
    }

    @Test
    fun `MemoryIndexRequest serializes correctly`() {
        val request = MemoryIndexRequest(taskId = "task-123")

        val json = gson.toJson(request)

        assertTrue(json.contains("task_id") || json.contains("task-123"))
    }

    @Test
    fun `MemoryStatsResponse deserializes correctly`() {
        val json =
            """{
            "total_documents": 100,
            "by_type": {"spec": 50, "impl": 30, "review": 20},
            "enabled": true
        }"""

        val response = gson.fromJson(json, MemoryStatsResponse::class.java)

        assertEquals(100, response.totalDocuments)
        assertEquals(50, response.byType["spec"])
        assertTrue(response.enabled)
    }

    // ========================================================================
    // Library Models
    // ========================================================================

    @Test
    fun `LibraryListResponse deserializes correctly`() {
        val json =
            """{
            "collections": [
                {
                    "id": "c1",
                    "name": "React Docs",
                    "source": "https://react.dev",
                    "source_type": "website",
                    "include_mode": "all",
                    "page_count": 50,
                    "total_size": 1024000,
                    "location": "/docs/react"
                }
            ],
            "count": 1
        }"""

        val response = gson.fromJson(json, LibraryListResponse::class.java)

        assertEquals(1, response.count)
        assertEquals("React Docs", response.collections[0].name)
        assertEquals("website", response.collections[0].sourceType)
        assertEquals(50, response.collections[0].pageCount)
    }

    @Test
    fun `LibraryCollection handles optional fields`() {
        val json =
            """{
            "id": "c1",
            "name": "Test",
            "source": "local",
            "source_type": "directory",
            "include_mode": "all",
            "page_count": 10,
            "total_size": 1000,
            "location": "/tmp",
            "pulled_at": "2024-01-01T00:00:00Z",
            "tags": ["api", "docs"],
            "paths": ["/api", "/guide"]
        }"""

        val collection = gson.fromJson(json, LibraryCollection::class.java)

        assertEquals("2024-01-01T00:00:00Z", collection.pulledAt)
        assertEquals(listOf("api", "docs"), collection.tags)
        assertEquals(listOf("/api", "/guide"), collection.paths)
    }

    @Test
    fun `LibraryShowResponse deserializes correctly`() {
        val json =
            """{
            "collection": {
                "id": "c1", "name": "Test", "source": "local",
                "source_type": "directory", "include_mode": "all",
                "page_count": 5, "total_size": 500, "location": "/tmp"
            },
            "pages": ["page1.md", "page2.md"]
        }"""

        val response = gson.fromJson(json, LibraryShowResponse::class.java)

        assertEquals("Test", response.collection.name)
        assertEquals(2, response.pages.size)
    }

    @Test
    fun `LibraryStatsResponse deserializes correctly`() {
        val json =
            """{
            "total_collections": 10,
            "total_pages": 500,
            "total_size": 5000000,
            "project_count": 7,
            "shared_count": 3,
            "by_mode": {"all": 8, "selective": 2},
            "enabled": true
        }"""

        val response = gson.fromJson(json, LibraryStatsResponse::class.java)

        assertEquals(10, response.totalCollections)
        assertEquals(500, response.totalPages)
        assertEquals(5000000, response.totalSize)
        assertEquals(7, response.projectCount)
        assertEquals(3, response.sharedCount)
        assertEquals(8, response.byMode["all"])
    }

    // ========================================================================
    // Links Models
    // ========================================================================

    @Test
    fun `LinksListResponse deserializes correctly`() {
        val json =
            """{
            "links": [
                {"source": "spec:1", "target": "impl:1", "context": "implements", "created_at": "2024-01-01T00:00:00Z"}
            ],
            "count": 1
        }"""

        val response = gson.fromJson(json, LinksListResponse::class.java)

        assertEquals(1, response.count)
        assertEquals("spec:1", response.links[0].source)
        assertEquals("impl:1", response.links[0].target)
        assertEquals("implements", response.links[0].context)
    }

    @Test
    fun `EntityLinksResponse deserializes correctly`() {
        val json =
            """{
            "entity_id": "spec:123",
            "outgoing": [
                {"source": "spec:123", "target": "impl:1", "context": "implements", "created_at": "2024-01-01T00:00:00Z"}
            ],
            "incoming": [
                {"source": "decision:1", "target": "spec:123", "context": "references", "created_at": "2024-01-01T00:00:00Z"}
            ]
        }"""

        val response = gson.fromJson(json, EntityLinksResponse::class.java)

        assertEquals("spec:123", response.entityId)
        assertEquals(1, response.outgoing.size)
        assertEquals(1, response.incoming.size)
        assertEquals("impl:1", response.outgoing[0].target)
        assertEquals("decision:1", response.incoming[0].source)
    }

    @Test
    fun `LinksSearchResponse deserializes correctly`() {
        val json =
            """{
            "query": "auth",
            "results": [
                {"entity_id": "spec:auth", "type": "spec", "name": "Authentication Spec"}
            ],
            "count": 1
        }"""

        val response = gson.fromJson(json, LinksSearchResponse::class.java)

        assertEquals("auth", response.query)
        assertEquals(1, response.count)
        assertEquals("spec:auth", response.results[0].entityId)
        assertEquals("Authentication Spec", response.results[0].name)
    }

    @Test
    fun `LinksStatsResponse deserializes correctly`() {
        val json =
            """{
            "total_links": 100,
            "total_sources": 50,
            "total_targets": 60,
            "orphan_entities": 5,
            "most_linked": [
                {"entity_id": "spec:main", "type": "spec", "total_links": 20}
            ],
            "enabled": true
        }"""

        val response = gson.fromJson(json, LinksStatsResponse::class.java)

        assertEquals(100, response.totalLinks)
        assertEquals(50, response.totalSources)
        assertEquals(60, response.totalTargets)
        assertEquals(5, response.orphanEntities)
        assertEquals(1, response.mostLinked.size)
        assertTrue(response.enabled)
    }

    // ========================================================================
    // Project Models
    // ========================================================================

    @Test
    fun `ProjectPlanRequest serializes correctly`() {
        val request = ProjectPlanRequest(source = "github:123", title = "My Plan", instructions = "Focus on tests")

        val json = gson.toJson(request)

        assertTrue(json.contains("github:123"))
        assertTrue(json.contains("My Plan"))
    }

    @Test
    fun `ProjectPlanResponse deserializes correctly`() {
        val json =
            """{
            "success": true,
            "queue_id": "q-123",
            "task_count": 5,
            "questions": ["What priority?", "Include tests?"]
        }"""

        val response = gson.fromJson(json, ProjectPlanResponse::class.java)

        assertTrue(response.success)
        assertEquals("q-123", response.queueId)
        assertEquals(5, response.taskCount)
        assertEquals(2, response.questions?.size)
    }

    @Test
    fun `ProjectQueueTask deserializes correctly`() {
        val json =
            """{
            "id": "t-1",
            "title": "Implement feature",
            "status": "pending",
            "priority": 1,
            "parent_id": "t-0",
            "depends_on": ["t-2", "t-3"]
        }"""

        val task = gson.fromJson(json, ProjectQueueTask::class.java)

        assertEquals("t-1", task.id)
        assertEquals("Implement feature", task.title)
        assertEquals("pending", task.status)
        assertEquals(1, task.priority)
        assertEquals("t-0", task.parentId)
        assertEquals(listOf("t-2", "t-3"), task.dependsOn)
    }

    // ========================================================================
    // Stack Models
    // ========================================================================

    @Test
    fun `StackListResponse deserializes correctly`() {
        val json =
            """{
            "stacks": [
                {
                    "id": "s-1",
                    "task_count": 3,
                    "tasks": [
                        {"id": "t-1", "branch": "feature/auth", "state": "done"},
                        {"id": "t-2", "branch": "feature/db", "state": "implementing", "depends_on": "t-1"}
                    ]
                }
            ],
            "count": 1
        }"""

        val response = gson.fromJson(json, StackListResponse::class.java)

        assertEquals(1, response.count)
        assertEquals("s-1", response.stacks[0].id)
        assertEquals(3, response.stacks[0].taskCount)
        assertEquals(2, response.stacks[0].tasks.size)
        assertEquals("feature/auth", response.stacks[0].tasks[0].branch)
    }

    @Test
    fun `StackTask handles optional fields`() {
        val json =
            """{
            "id": "t-1",
            "branch": "feature/test",
            "state": "done",
            "depends_on": "t-0",
            "pr_number": 42
        }"""

        val task = gson.fromJson(json, StackTask::class.java)

        assertEquals("t-0", task.dependsOn)
        assertEquals(42, task.prNumber)
    }
}
