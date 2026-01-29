// Package schema provides JSON Schema-based extraction for project plans.
// It enables flexible parsing of AI-generated task lists using LLM-powered extraction
// with a fallback to regex-based parsing.
package schema

import "encoding/json"

// TaskSchema defines the JSON Schema for project plan extraction.
// This schema describes the expected structure of AI-generated project plans,
// allowing the LLM to extract structured data from varied input formats.
var TaskSchema = json.RawMessage(`{
    "type": "object",
    "properties": {
        "tasks": {
            "type": "array",
            "items": {
                "type": "object",
                "properties": {
                    "id": {"type": "string"},
                    "title": {"type": "string"},
                    "priority": {"type": "integer"},
                    "status": {"type": "string"},
                    "labels": {"type": "array", "items": {"type": "string"}},
                    "depends_on": {"type": "array", "items": {"type": "string"}},
                    "assignee": {"type": "string"},
                    "description": {"type": "string"}
                },
                "required": ["id", "title"]
            }
        },
        "questions": {
            "type": "array",
            "items": {"type": "string"}
        },
        "blockers": {
            "type": "array",
            "items": {"type": "string"}
        }
    }
}`)

// ParsedPlan represents a project plan extracted from AI content.
// It mirrors the structure in export.ParsedPlan but uses local types
// to avoid circular dependencies.
type ParsedPlan struct {
	Tasks     []*Task  `json:"tasks"`
	Questions []string `json:"questions"`
	Blockers  []string `json:"blockers"`
}

// Task represents a single task in a project plan.
type Task struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Priority    int      `json:"priority"`
	Status      string   `json:"status"`
	Labels      []string `json:"labels"`
	DependsOn   []string `json:"depends_on"`
	Assignee    string   `json:"assignee"`
	Description string   `json:"description"`
}
