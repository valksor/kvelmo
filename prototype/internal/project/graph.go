package project

import (
	"fmt"
	"strings"
)

// GraphNode represents a node in the dependency graph.
type GraphNode struct {
	ID       string
	Title    string
	Status   string
	Priority int
}

// GraphEdge represents a dependency edge between nodes.
type GraphEdge struct {
	From string
	To   string
}

// DependencyGraph represents a task dependency graph.
type DependencyGraph struct {
	Nodes []GraphNode
	Edges []GraphEdge
}

// ASCIIGraph generates an ASCII art representation of the dependency graph.
func ASCIIGraph(graph *DependencyGraph) string {
	if len(graph.Nodes) == 0 {
		return "No tasks to display"
	}

	var sb strings.Builder

	// Build a simple hierarchical layout
	// Group nodes by level (number of dependencies)
	nodeLevels := make(map[string]int)
	maxLevel := 0

	for _, node := range graph.Nodes {
		level := 0
		for _, edge := range graph.Edges {
			if edge.To == node.ID {
				level++
			}
		}
		nodeLevels[node.ID] = level
		if level > maxLevel {
			maxLevel = level
		}
	}

	// Group nodes by level
	levelNodes := make([][]GraphNode, maxLevel+1)
	for _, node := range graph.Nodes {
		level := nodeLevels[node.ID]
		levelNodes[level] = append(levelNodes[level], node)
	}

	// Render graph level by level
	sb.WriteString("Task Dependency Graph\n")
	sb.WriteString("======================\n\n")

	for level, nodes := range levelNodes {
		if len(nodes) == 0 {
			continue
		}

		sb.WriteString(fmt.Sprintf("Level %d:\n", level))

		for _, node := range nodes {
			statusIcon := getStatusIcon(node.Status)
			title := node.Title
			if len(title) > 30 {
				title = title[:27] + "..."
			}

			// Check if this node has outgoing edges
			var outgoing []string
			for _, edge := range graph.Edges {
				if edge.From == node.ID {
					outgoing = append(outgoing, edge.To)
				}
			}

			line := fmt.Sprintf("  [%s] %s", statusIcon, title)
			if len(outgoing) > 0 {
				line += " →"
				var lineSb90 strings.Builder
				for _, dep := range outgoing {
					depTitle := getNodeTitle(graph, dep)
					if len(depTitle) > 15 {
						depTitle = depTitle[:12] + "..."
					}
					lineSb90.WriteString(" " + depTitle)
				}
				line += lineSb90.String()
			}
			sb.WriteString(line + "\n")
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// ASCIICriticalPath generates an ASCII art representation showing the critical path.
func ASCIICriticalPath(graph *DependencyGraph) []string {
	// Find nodes with no dependencies (roots)
	var roots []GraphNode
	for _, node := range graph.Nodes {
		hasDependency := false
		for _, edge := range graph.Edges {
			if edge.To == node.ID {
				hasDependency = true

				break
			}
		}
		if !hasDependency {
			roots = append(roots, node)
		}
	}

	// For each root, find the longest path to any leaf
	var paths []string
	for _, root := range roots {
		path := findLongestPath(graph, root.ID, []string{})
		if len(path) > 0 {
			paths = append(paths, strings.Join(path, " → "))
		}
	}

	return paths
}

// findLongestPath finds the longest path from a node to any leaf using DFS.
func findLongestPath(graph *DependencyGraph, nodeID string, currentPath []string) []string {
	currentPath = append(currentPath, getNodeTitle(graph, nodeID))

	// Find all nodes this node depends on
	var children []string
	for _, edge := range graph.Edges {
		if edge.From == nodeID {
			children = append(children, edge.To)
		}
	}

	// If no children, this is a leaf
	if len(children) == 0 {
		return currentPath
	}

	// Find longest path through children
	var longestPath []string
	for _, child := range children {
		path := findLongestPath(graph, child, make([]string, len(currentPath)))
		copy(path, currentPath)
		if len(path) > len(longestPath) {
			longestPath = path
		}
	}

	return longestPath
}

func getNodeTitle(graph *DependencyGraph, nodeID string) string {
	for _, node := range graph.Nodes {
		if node.ID == nodeID {
			return node.Title
		}
	}

	return nodeID
}

func getStatusIcon(status string) string {
	switch status {
	case "done":
		return "●"
	case "in_progress":
		return "◑"
	case "pending":
		return "○"
	case "blocked":
		return "⊘"
	default:
		return "○"
	}
}

// GenerateMermaid generates a Mermaid diagram specification for the graph.
func GenerateMermaid(graph *DependencyGraph) string {
	var sb strings.Builder

	sb.WriteString("graph TD\n")

	// Define nodes
	for _, node := range graph.Nodes {
		label := strings.ReplaceAll(node.Title, "\"", "'")
		if len(label) > 20 {
			label = label[:17] + "..."
		}
		nodeID := strings.ReplaceAll(node.ID, "-", "_")
		sb.WriteString(fmt.Sprintf("  %s[\"%s\"]\n", nodeID, label))
	}

	// Define edges
	for _, edge := range graph.Edges {
		from := strings.ReplaceAll(edge.From, "-", "_")
		to := strings.ReplaceAll(edge.To, "-", "_")
		sb.WriteString(fmt.Sprintf("  %s --> %s\n", from, to))
	}

	return sb.String()
}
