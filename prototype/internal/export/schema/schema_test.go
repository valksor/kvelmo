// Package schema provides JSON Schema-based extraction for project plans.
package schema

import (
	"encoding/json"
	"testing"
)

func TestParseStatus(t *testing.T) {
	tests := []struct {
		name   string
		status string
		want   string
	}{
		{
			name:   "ready status",
			status: "ready",
			want:   "ready",
		},
		{
			name:   "READY uppercase",
			status: "READY",
			want:   "ready",
		},
		{
			name:   "Ready mixed case",
			status: "Ready",
			want:   "ready",
		},
		{
			name:   "blocked status",
			status: "blocked",
			want:   "blocked",
		},
		{
			name:   "submitted status",
			status: "submitted",
			want:   "submitted",
		},
		{
			name:   "pending status",
			status: "pending",
			want:   "pending",
		},
		{
			name:   "empty status defaults to pending",
			status: "",
			want:   "pending",
		},
		{
			name:   "unknown status defaults to pending",
			status: "in_progress",
			want:   "pending",
		},
		{
			name:   "status with whitespace",
			status: "  ready  ",
			want:   "ready",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseStatus(tt.status)
			if string(got) != tt.want {
				t.Errorf("parseStatus(%q) = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}

func TestToStorageTasksInternal(t *testing.T) {
	tests := []struct {
		name        string
		schemaTasks []*Task
		wantCount   int
		wantID      string
		wantTitle   string
	}{
		{
			name:        "nil tasks",
			schemaTasks: nil,
			wantCount:   0,
		},
		{
			name:        "empty tasks",
			schemaTasks: []*Task{},
			wantCount:   0,
		},
		{
			name: "single task",
			schemaTasks: []*Task{
				{
					ID:          "task-1",
					Title:       "Build auth",
					Priority:    1,
					Status:      "ready",
					Description: "Implement OAuth2",
				},
			},
			wantCount: 1,
			wantID:    "task-1",
			wantTitle: "Build auth",
		},
		{
			name: "multiple tasks",
			schemaTasks: []*Task{
				{
					ID:    "task-1",
					Title: "First task",
				},
				{
					ID:    "task-2",
					Title: "Second task",
				},
			},
			wantCount: 2,
		},
		{
			name: "task with nil in slice",
			schemaTasks: []*Task{
				{
					ID:    "task-1",
					Title: "Valid task",
				},
				nil,
				{
					ID:    "task-2",
					Title: "Another valid task",
				},
			},
			wantCount: 2, // Nil task should be skipped
		},
		{
			name: "task with all fields",
			schemaTasks: []*Task{
				{
					ID:          "task-1",
					Title:       "Full task",
					Priority:    1,
					Status:      "ready",
					Labels:      []string{"backend", "security"},
					DependsOn:   []string{"task-0"},
					Assignee:    "user@example.com",
					Description: "Full description",
				},
			},
			wantCount: 1,
			wantID:    "task-1",
			wantTitle: "Full task",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toStorageTasks(tt.schemaTasks)
			if len(got) != tt.wantCount {
				t.Errorf("toStorageTasks() returned %d tasks, want %d", len(got), tt.wantCount)
			}

			if tt.wantID != "" && len(got) > 0 {
				if got[0].ID != tt.wantID {
					t.Errorf("toStorageTasks()[0].ID = %v, want %v", got[0].ID, tt.wantID)
				}
				if got[0].Title != tt.wantTitle {
					t.Errorf("toStorageTasks()[0].Title = %v, want %v", got[0].Title, tt.wantTitle)
				}
			}
		})
	}
}

func TestToStorageTasks(t *testing.T) {
	tests := []struct {
		name          string
		plan          *ParsedPlan
		wantTaskCount int
		wantQuestions int
		wantBlockers  int
		wantNil       bool
	}{
		{
			name:    "nil plan",
			plan:    nil,
			wantNil: true,
		},
		{
			name:          "empty plan",
			plan:          &ParsedPlan{},
			wantTaskCount: 0,
			wantQuestions: 0,
			wantBlockers:  0,
			wantNil:       true, // Empty slices return nil from toStorageTasks
		},
		{
			name: "plan with tasks",
			plan: &ParsedPlan{
				Tasks: []*Task{
					{ID: "task-1", Title: "Test task"},
				},
				Questions: []string{"Question 1?"},
				Blockers:  []string{"Blocker 1"},
			},
			wantTaskCount: 1,
			wantQuestions: 1,
			wantBlockers:  1,
			wantNil:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tasks, questions, blockers := ToStorageTasks(tt.plan)
			if (tasks == nil && questions == nil && blockers == nil) != tt.wantNil {
				t.Errorf("ToStorageTasks() nil = %v, want nil %v", tasks == nil && questions == nil && blockers == nil, tt.wantNil)
			}

			if !tt.wantNil {
				if len(tasks) != tt.wantTaskCount {
					t.Errorf("ToStorageTasks() Tasks count = %v, want %v", len(tasks), tt.wantTaskCount)
				}
				if len(questions) != tt.wantQuestions {
					t.Errorf("ToStorageTasks() Questions count = %v, want %v", len(questions), tt.wantQuestions)
				}
				if len(blockers) != tt.wantBlockers {
					t.Errorf("ToStorageTasks() Blockers count = %v, want %v", len(blockers), tt.wantBlockers)
				}
			}
		})
	}
}

func TestExtractJSON(t *testing.T) {
	tests := []struct {
		name     string
		response string
		want     string
	}{
		{
			name:     "plain JSON",
			response: `{"tasks": []}`,
			want:     `{"tasks": []}`,
		},
		{
			name:     "JSON in markdown code block",
			response: "```json\n{\"tasks\": []}\n```",
			want:     `{"tasks": []}`,
		},
		{
			name:     "JSON in generic code block",
			response: "```\n{\"tasks\": []}\n```",
			want:     `{"tasks": []}`,
		},
		{
			name:     "JSON with surrounding text",
			response: "Here's the result:\n```json\n{\"tasks\": []}\n```\nDone!",
			want:     `{"tasks": []}`,
		},
		{
			name:     "JSON with leading/trailing whitespace",
			response: "  \n  {\"tasks\": []}  \n  ",
			want:     `{"tasks": []}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractJSON(tt.response)
			if got != tt.want {
				t.Errorf("extractJSON() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTrimWhitespace(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want string
	}{
		{
			name: "no whitespace",
			s:    "hello",
			want: "hello",
		},
		{
			name: "leading spaces",
			s:    "  hello",
			want: "hello",
		},
		{
			name: "trailing spaces",
			s:    "hello  ",
			want: "hello",
		},
		{
			name: "leading and trailing spaces",
			s:    "  hello  ",
			want: "hello",
		},
		{
			name: "tabs and newlines",
			s:    "\t\nhello\n\t",
			want: "hello",
		},
		{
			name: "only whitespace",
			s:    "   \n\t   ",
			want: "",
		},
		{
			name: "empty string",
			s:    "",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := trimWhitespace(tt.s)
			if got != tt.want {
				t.Errorf("trimWhitespace() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTaskSchemaIsValid(t *testing.T) {
	// Verify the schema is valid JSON
	if !isValidJSON(string(TaskSchema)) {
		t.Errorf("TaskSchema is not valid JSON: %s", string(TaskSchema))
	}
}

// isValidJSON checks if a string is valid JSON.
func isValidJSON(s string) bool {
	var js map[string]interface{}

	return json.Unmarshal([]byte(s), &js) == nil
}
