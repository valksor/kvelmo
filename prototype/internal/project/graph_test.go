package project

import (
	"testing"
)

func TestASCIIGraph(t *testing.T) {
	tests := []struct {
		name     string
		graph    *DependencyGraph
		contains []string // Substrings that should be in output
	}{
		{
			name:     "empty nodes returns message",
			graph:    &DependencyGraph{},
			contains: []string{"No tasks to display"},
		},
		{
			name: "single node with no edges",
			graph: &DependencyGraph{
				Nodes: []GraphNode{
					{ID: "1", Title: "Task 1", Status: "pending", Priority: 1},
				},
				Edges: []GraphEdge{},
			},
			contains: []string{"Task Dependency Graph", "Level 0:", "Task 1"},
		},
		{
			name: "multiple nodes at different levels",
			graph: &DependencyGraph{
				Nodes: []GraphNode{
					{ID: "1", Title: "Root Task", Status: "pending", Priority: 1},
					{ID: "2", Title: "Child Task", Status: "in_progress", Priority: 2},
					{ID: "3", Title: "Leaf Task", Status: "done", Priority: 3},
				},
				Edges: []GraphEdge{
					{From: "1", To: "2"},
					{From: "2", To: "3"},
				},
			},
			contains: []string{"Level 0:", "Root Task", "Level 1:", "Child Task", "Leaf Task"},
		},
		{
			name: "long title truncation",
			graph: &DependencyGraph{
				Nodes: []GraphNode{
					{ID: "1", Title: "This is a very long task title that should be truncated because it exceeds the maximum length", Status: "pending", Priority: 1},
				},
				Edges: []GraphEdge{},
			},
			contains: []string{"..."},
		},
		{
			name: "node with outgoing edges shows arrow",
			graph: &DependencyGraph{
				Nodes: []GraphNode{
					{ID: "1", Title: "Parent", Status: "pending", Priority: 1},
					{ID: "2", Title: "Child", Status: "pending", Priority: 2},
				},
				Edges: []GraphEdge{
					{From: "1", To: "2"},
				},
			},
			contains: []string{"→", "Child"},
		},
		{
			name: "multiple dependencies on same node",
			graph: &DependencyGraph{
				Nodes: []GraphNode{
					{ID: "1", Title: "Task 1", Status: "pending", Priority: 1},
					{ID: "2", Title: "Task 2", Status: "pending", Priority: 2},
					{ID: "3", Title: "Task 3", Status: "pending", Priority: 3},
				},
				Edges: []GraphEdge{
					{From: "1", To: "3"},
					{From: "2", To: "3"},
				},
			},
			contains: []string{"Level 0:", "Task 1", "Task 2", "Level 2:", "Task 3"},
		},
		{
			name: "diamond dependency pattern",
			graph: &DependencyGraph{
				Nodes: []GraphNode{
					{ID: "1", Title: "Root", Status: "pending", Priority: 1},
					{ID: "2", Title: "Branch A", Status: "pending", Priority: 2},
					{ID: "3", Title: "Branch B", Status: "pending", Priority: 3},
					{ID: "4", Title: "Converge", Status: "pending", Priority: 4},
				},
				Edges: []GraphEdge{
					{From: "1", To: "2"},
					{From: "1", To: "3"},
					{From: "2", To: "4"},
					{From: "3", To: "4"},
				},
			},
			contains: []string{"Level 0:", "Root", "Level 1:", "Branch A", "Branch B", "Level 2:", "Converge"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ASCIIGraph(tt.graph)
			for _, expected := range tt.contains {
				if !containsString(result, expected) {
					t.Errorf("ASCIIGraph() output should contain %q", expected)
				}
			}
		})
	}
}

func TestASCIICriticalPath(t *testing.T) {
	tests := []struct {
		name          string
		graph         *DependencyGraph
		expectedCount int
		contains      []string
	}{
		{
			name: "empty graph returns empty",
			graph: &DependencyGraph{
				Nodes: []GraphNode{},
				Edges: []GraphEdge{},
			},
			expectedCount: 0,
		},
		{
			name: "single node with no dependencies",
			graph: &DependencyGraph{
				Nodes: []GraphNode{
					{ID: "1", Title: "Single Task", Status: "pending", Priority: 1},
				},
				Edges: []GraphEdge{},
			},
			expectedCount: 1,
			contains:      []string{"Single Task"},
		},
		{
			name: "linear chain returns single path",
			graph: &DependencyGraph{
				Nodes: []GraphNode{
					{ID: "1", Title: "Task A", Status: "pending", Priority: 1},
					{ID: "2", Title: "Task B", Status: "pending", Priority: 2},
					{ID: "3", Title: "Task C", Status: "pending", Priority: 3},
				},
				Edges: []GraphEdge{
					{From: "1", To: "2"},
					{From: "2", To: "3"},
				},
			},
			expectedCount: 1,
			contains:      []string{"Task A", "→", "Task B", "Task C"},
		},
		{
			name: "diamond pattern picks longest path",
			graph: &DependencyGraph{
				Nodes: []GraphNode{
					{ID: "1", Title: "Root", Status: "pending", Priority: 1},
					{ID: "2", Title: "Branch A", Status: "pending", Priority: 2},
					{ID: "3", Title: "Branch B", Status: "pending", Priority: 3},
					{ID: "4", Title: "Leaf A", Status: "pending", Priority: 4},
					{ID: "5", Title: "Leaf B", Status: "pending", Priority: 5},
				},
				Edges: []GraphEdge{
					{From: "1", To: "2"},
					{From: "1", To: "3"},
					{From: "2", To: "4"},
					{From: "3", To: "5"},
				},
			},
			expectedCount: 1,                        // One root, returns one longest path
			contains:      []string{"Root", "Leaf"}, // Both branches are same length, picks one
		},
		{
			name: "multiple roots return multiple paths",
			graph: &DependencyGraph{
				Nodes: []GraphNode{
					{ID: "1", Title: "Root 1", Status: "pending", Priority: 1},
					{ID: "2", Title: "Root 2", Status: "pending", Priority: 2},
					{ID: "3", Title: "Child of 1", Status: "pending", Priority: 3},
					{ID: "4", Title: "Child of 2", Status: "pending", Priority: 4},
				},
				Edges: []GraphEdge{
					{From: "1", To: "3"},
					{From: "2", To: "4"},
				},
			},
			expectedCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ASCIICriticalPath(tt.graph)
			if len(result) != tt.expectedCount {
				t.Errorf("ASCIICriticalPath() returned %d paths, expected %d", len(result), tt.expectedCount)
			}
			for _, expected := range tt.contains {
				found := false
				for _, path := range result {
					if containsString(path, expected) {
						found = true

						break
					}
				}
				if !found {
					t.Errorf("ASCIICriticalPath() paths should contain %q", expected)
				}
			}
		})
	}
}

func TestGenerateMermaid(t *testing.T) {
	tests := []struct {
		name     string
		graph    *DependencyGraph
		contains []string
	}{
		{
			name: "empty graph returns header",
			graph: &DependencyGraph{
				Nodes: []GraphNode{},
				Edges: []GraphEdge{},
			},
			contains: []string{"graph TD"},
		},
		{
			name: "single node",
			graph: &DependencyGraph{
				Nodes: []GraphNode{
					{ID: "task-1", Title: "My Task", Status: "pending", Priority: 1},
				},
				Edges: []GraphEdge{},
			},
			contains: []string{"graph TD", "task_1", "My Task"},
		},
		{
			name: "multiple nodes and edges",
			graph: &DependencyGraph{
				Nodes: []GraphNode{
					{ID: "task-1", Title: "Task 1", Status: "pending", Priority: 1},
					{ID: "task-2", Title: "Task 2", Status: "pending", Priority: 2},
				},
				Edges: []GraphEdge{
					{From: "task-1", To: "task-2"},
				},
			},
			contains: []string{"task_1", "task_2", "-->", "Task 1", "Task 2"},
		},
		{
			name: "hyphens in IDs are replaced",
			graph: &DependencyGraph{
				Nodes: []GraphNode{
					{ID: "my-task-id", Title: "My Task", Status: "pending", Priority: 1},
				},
				Edges: []GraphEdge{},
			},
			contains: []string{"my_task_id"},
		},
		{
			name: "quotes in titles are replaced",
			graph: &DependencyGraph{
				Nodes: []GraphNode{
					{ID: "1", Title: `Task with "quotes"`, Status: "pending", Priority: 1},
				},
				Edges: []GraphEdge{},
			},
			contains: []string{"Task with 'quotes'"},
		},
		{
			name: "long titles are truncated",
			graph: &DependencyGraph{
				Nodes: []GraphNode{
					{ID: "1", Title: "This is a very long task title that exceeds twenty characters", Status: "pending", Priority: 1},
				},
				Edges: []GraphEdge{},
			},
			contains: []string{"..."},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateMermaid(tt.graph)
			for _, expected := range tt.contains {
				if !containsString(result, expected) {
					t.Errorf("GenerateMermaid() output should contain %q, got:\n%s", expected, result)
				}
			}
		})
	}
}

func TestGetStatusIcon(t *testing.T) {
	tests := []struct {
		status   string
		expected string
	}{
		{"done", "●"},
		{"in_progress", "◑"},
		{"pending", "○"},
		{"blocked", "⊘"},
		{"unknown", "○"},   // default case
		{"", "○"},          // empty string
		{"COMPLETED", "○"}, // case sensitive, not matched
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			result := getStatusIcon(tt.status)
			if result != tt.expected {
				t.Errorf("getStatusIcon(%q) = %q, expected %q", tt.status, result, tt.expected)
			}
		})
	}
}

func TestGetNodeTitle(t *testing.T) {
	tests := []struct {
		name     string
		graph    *DependencyGraph
		nodeID   string
		expected string
	}{
		{
			name: "found node returns title",
			graph: &DependencyGraph{
				Nodes: []GraphNode{
					{ID: "1", Title: "My Task", Status: "pending", Priority: 1},
				},
			},
			nodeID:   "1",
			expected: "My Task",
		},
		{
			name: "not found returns ID",
			graph: &DependencyGraph{
				Nodes: []GraphNode{
					{ID: "1", Title: "My Task", Status: "pending", Priority: 1},
				},
			},
			nodeID:   "999",
			expected: "999",
		},
		{
			name:     "empty graph returns ID",
			graph:    &DependencyGraph{Nodes: []GraphNode{}},
			nodeID:   "1",
			expected: "1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getNodeTitle(tt.graph, tt.nodeID)
			if result != tt.expected {
				t.Errorf("getNodeTitle(%q) = %q, expected %q", tt.nodeID, result, tt.expected)
			}
		})
	}
}

func TestFindLongestPath(t *testing.T) {
	tests := []struct {
		name          string
		graph         *DependencyGraph
		nodeID        string
		minLength     int    // minimum expected path length
		shouldContain string // substring that should be in path
	}{
		{
			name: "leaf node returns path with just itself",
			graph: &DependencyGraph{
				Nodes: []GraphNode{
					{ID: "1", Title: "Leaf", Status: "pending", Priority: 1},
				},
				Edges: []GraphEdge{},
			},
			nodeID:        "1",
			minLength:     1,
			shouldContain: "Leaf",
		},
		{
			name: "linear chain returns full path",
			graph: &DependencyGraph{
				Nodes: []GraphNode{
					{ID: "1", Title: "A", Status: "pending", Priority: 1},
					{ID: "2", Title: "B", Status: "pending", Priority: 2},
					{ID: "3", Title: "C", Status: "pending", Priority: 3},
				},
				Edges: []GraphEdge{
					{From: "1", To: "2"},
					{From: "2", To: "3"},
				},
			},
			nodeID:        "1",
			minLength:     3,
			shouldContain: "C", // Last node
		},
		{
			name: "diamond picks longest branch",
			graph: &DependencyGraph{
				Nodes: []GraphNode{
					{ID: "1", Title: "Root", Status: "pending", Priority: 1},
					{ID: "2", Title: "Short", Status: "pending", Priority: 2},
					{ID: "3", Title: "Long", Status: "pending", Priority: 3},
					{ID: "4", Title: "End", Status: "pending", Priority: 4},
				},
				Edges: []GraphEdge{
					{From: "1", To: "2"},
					{From: "1", To: "3"},
					{From: "2", To: "4"},
					{From: "3", To: "4"},
				},
			},
			nodeID:        "1",
			minLength:     3, // Root -> Long -> End OR Root -> Short -> End
			shouldContain: "End",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findLongestPath(tt.graph, tt.nodeID, []string{})
			if len(result) < tt.minLength {
				t.Errorf("findLongestPath() returned path of length %d, expected at least %d", len(result), tt.minLength)
			}
			if tt.shouldContain != "" && !containsStringSlice(result, tt.shouldContain) {
				t.Errorf("findLongestPath() path should contain %q, got %v", tt.shouldContain, result)
			}
		})
	}
}

// Helper function to check if a string contains a substring.
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}

	return false
}

// Helper function to check if a slice of strings contains a string.
func containsStringSlice(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}

	return false
}
