package export

import (
	"context"
	"regexp"
	"strconv"
	"strings"

	"github.com/valksor/go-mehrhof/internal/agent"
	"github.com/valksor/go-mehrhof/internal/export/schema"
	"github.com/valksor/go-mehrhof/internal/storage"
)

// ParsedPlan represents the structured output from AI task planning.
type ParsedPlan struct {
	Tasks     []*storage.QueuedTask
	Questions []string
	Blockers  []string
}

// ParseProjectPlan parses AI output into a structured plan with tasks.
// Expected format:
//
//	## Tasks
//	### task-1: Task title
//	- **Priority**: 1 (high)
//	- **Status**: ready
//	- **Labels**: backend, security
//	- **Depends on**: task-0
//	- **Assignee**: user@example.com
//	- **Description**: Detailed description here...
//
//	## Questions
//	1. Question one?
//	2. Question two?
//
//	## Blockers
//	- Blocker one
//	- Blocker two
func ParseProjectPlan(content string) *ParsedPlan {
	return ParseProjectPlanWithSchema(context.Background(), content, nil)
}

// ParseProjectPlanWithSchema parses AI output into a structured plan with tasks.
// It attempts schema-driven extraction first (if an agent is provided), then falls back
// to regex-based parsing if schema extraction fails or no agent is provided.
func ParseProjectPlanWithSchema(ctx context.Context, content string, ag agent.Agent) *ParsedPlan {
	// Try schema-driven extraction if agent is available
	if ag != nil {
		extractor := schema.NewExtractor(ag)
		plan, err := extractor.ExtractPlan(ctx, content)
		if err == nil && plan != nil && len(plan.Tasks) > 0 {
			tasks, questions, blockers := schema.ToStorageTasks(plan)

			return &ParsedPlan{
				Tasks:     tasks,
				Questions: questions,
				Blockers:  blockers,
			}
		}
		// Schema extraction failed or returned empty, fall through to regex parsing
	}

	// Fallback to regex-based parsing
	return parseProjectPlanRegex(content)
}

// parseProjectPlanRegex parses AI output using regex patterns.
// This is the original parsing logic, maintained as a fallback.
func parseProjectPlanRegex(content string) *ParsedPlan {
	plan := &ParsedPlan{
		Tasks:     make([]*storage.QueuedTask, 0),
		Questions: make([]string, 0),
		Blockers:  make([]string, 0),
	}

	// Parse tasks section
	plan.Tasks = parseTasks(content)

	// Parse questions section
	plan.Questions = parseQuestions(content)

	// Parse blockers section
	plan.Blockers = parseBlockers(content)

	return plan
}

// parseTasks extracts tasks from the content.
func parseTasks(content string) []*storage.QueuedTask {
	var tasks []*storage.QueuedTask

	// Find the Tasks section
	tasksSection := extractSection(content, "Tasks")
	if tasksSection == "" {
		return tasks
	}

	// Pattern for task headers: ### task-N: Title
	taskPattern := regexp.MustCompile(`(?m)^###\s+(task-\d+):\s*(.+)$`)
	matches := taskPattern.FindAllStringSubmatchIndex(tasksSection, -1)

	for i, match := range matches {
		taskID := tasksSection[match[2]:match[3]]
		title := strings.TrimSpace(tasksSection[match[4]:match[5]])

		// Find the end of this task section
		var taskContent string
		if i+1 < len(matches) {
			taskContent = tasksSection[match[1]:matches[i+1][0]]
		} else {
			// Check for next section header
			nextSection := regexp.MustCompile(`(?m)^##\s+[^#]`).FindStringIndex(tasksSection[match[1]:])
			if len(nextSection) == 2 {
				taskContent = tasksSection[match[1] : match[1]+nextSection[0]]
			} else {
				taskContent = tasksSection[match[1]:]
			}
		}

		task := parseTaskContent(taskID, title, taskContent)
		tasks = append(tasks, task)
	}

	return tasks
}

// parseTaskContent extracts task fields from the task content block.
func parseTaskContent(id, title, content string) *storage.QueuedTask {
	task := &storage.QueuedTask{
		ID:        id,
		Title:     title,
		Status:    storage.TaskStatusPending,
		Priority:  0, // Will be set if found
		DependsOn: make([]string, 0),
		Labels:    make([]string, 0),
	}

	// Extract priority
	priorityPattern := regexp.MustCompile(`(?i)\*\*Priority\*\*:\s*(\d+)`)
	if match := priorityPattern.FindStringSubmatch(content); len(match) > 1 {
		if p, err := strconv.Atoi(match[1]); err == nil {
			task.Priority = p
		}
	}

	// Extract status
	statusPattern := regexp.MustCompile(`(?i)\*\*Status\*\*:\s*(\w+)`)
	if match := statusPattern.FindStringSubmatch(content); len(match) > 1 {
		status := strings.ToLower(match[1])
		switch status {
		case "ready":
			task.Status = storage.TaskStatusReady
		case "blocked":
			task.Status = storage.TaskStatusBlocked
		case "submitted":
			task.Status = storage.TaskStatusSubmitted
		default:
			task.Status = storage.TaskStatusPending
		}
	}

	// Extract labels
	labelsPattern := regexp.MustCompile(`(?i)\*\*Labels\*\*:\s*(.+)`)
	if match := labelsPattern.FindStringSubmatch(content); len(match) > 1 {
		labels := strings.Split(match[1], ",")
		for _, label := range labels {
			label = strings.TrimSpace(label)
			if label != "" {
				task.Labels = append(task.Labels, label)
			}
		}
	}

	// Extract depends on
	dependsPattern := regexp.MustCompile(`(?i)\*\*Depends on\*\*:\s*(.+)`)
	if match := dependsPattern.FindStringSubmatch(content); len(match) > 1 {
		deps := strings.Split(match[1], ",")
		for _, dep := range deps {
			dep = strings.TrimSpace(dep)
			if dep != "" && strings.HasPrefix(dep, "task-") {
				task.DependsOn = append(task.DependsOn, dep)
			}
		}
	}

	// Extract parent (for subtask hierarchy)
	parentPattern := regexp.MustCompile(`(?i)\*\*Parent\*\*:\s*(task-\d+)`)
	if match := parentPattern.FindStringSubmatch(content); len(match) > 1 {
		task.ParentID = strings.TrimSpace(match[1])
	}

	// Extract assignee
	assigneePattern := regexp.MustCompile(`(?i)\*\*Assignee\*\*:\s*(.+)`)
	if match := assigneePattern.FindStringSubmatch(content); len(match) > 1 {
		task.Assignee = strings.TrimSpace(match[1])
	}

	// Extract description (everything after the bullet points)
	descPattern := regexp.MustCompile(`(?i)\*\*Description\*\*:\s*(.+)`)
	if match := descPattern.FindStringSubmatch(content); len(match) > 1 {
		task.Description = strings.TrimSpace(match[1])
	} else {
		// Fallback: try to find description as multi-line content
		// Look for content after all the metadata fields
		lines := strings.Split(content, "\n")
		var descLines []string
		inDescription := false

		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				if inDescription {
					descLines = append(descLines, "")
				}

				continue
			}

			// Skip metadata lines
			if strings.HasPrefix(line, "- **") || strings.HasPrefix(line, "* **") {
				continue
			}

			// Skip task header
			if strings.HasPrefix(line, "###") {
				continue
			}

			// Everything else is description
			inDescription = true
			descLines = append(descLines, line)
		}

		if len(descLines) > 0 {
			task.Description = strings.TrimSpace(strings.Join(descLines, "\n"))
		}
	}

	return task
}

// parseQuestions extracts questions from the content.
func parseQuestions(content string) []string {
	var questions []string

	questionsSection := extractSection(content, "Questions")
	if questionsSection == "" {
		return questions
	}

	// Match numbered items or bullet points
	lines := strings.Split(questionsSection, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Remove numbering or bullet points
		questionPattern := regexp.MustCompile(`^(?:\d+\.\s*|[-*]\s*)(.+)`)
		if match := questionPattern.FindStringSubmatch(line); len(match) > 1 {
			questions = append(questions, strings.TrimSpace(match[1]))
		}
	}

	return questions
}

// parseBlockers extracts blockers from the content.
func parseBlockers(content string) []string {
	var blockers []string

	blockersSection := extractSection(content, "Blockers")
	if blockersSection == "" {
		return blockers
	}

	// Match bullet points
	lines := strings.Split(blockersSection, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Remove bullet points
		blockerPattern := regexp.MustCompile(`^[-*]\s*(.+)`)
		if match := blockerPattern.FindStringSubmatch(line); len(match) > 1 {
			blockers = append(blockers, strings.TrimSpace(match[1]))
		}
	}

	return blockers
}

// extractSection extracts content from a markdown section.
func extractSection(content string, sectionHeader string) string {
	// Try to find ## Section header
	pattern := regexp.MustCompile(`(?m)^##\s+` + regexp.QuoteMeta(sectionHeader) + `\s*$`)
	loc := pattern.FindStringIndex(content)
	if len(loc) < 2 {
		return ""
	}

	sectionStart := loc[1]

	// Find next ## header or end of content
	nextPattern := regexp.MustCompile(`(?m)^##\s+[^#]`)
	nextLoc := nextPattern.FindStringIndex(content[sectionStart:])

	var end int
	if len(nextLoc) >= 2 {
		end = sectionStart + nextLoc[0]
	} else {
		end = len(content)
	}

	return strings.TrimSpace(content[sectionStart:end])
}

// ParseTaskOrder parses AI output to extract the recommended task order and reasoning.
// Expected format:
//
//	## Recommended Order
//	1. task-3 - Brief reason
//	2. task-1 - Brief reason
//	...
//
//	## Reasoning
//	A 2-3 sentence explanation of the reordering strategy.
func ParseTaskOrder(content string) ([]string, string, error) {
	var order []string

	// Extract the Recommended Order section
	orderSection := extractSection(content, "Recommended Order")
	if orderSection == "" {
		return nil, "", errNoOrderSection
	}

	// Pattern to match numbered task items: 1. task-N - reason
	taskPattern := regexp.MustCompile(`(?m)^\d+\.\s+(task-\d+)`)
	matches := taskPattern.FindAllStringSubmatch(orderSection, -1)

	for _, match := range matches {
		if len(match) > 1 {
			order = append(order, match[1])
		}
	}

	if len(order) == 0 {
		return nil, "", errNoTasksFound
	}

	// Extract reasoning
	reasoning := extractSection(content, "Reasoning")

	return order, reasoning, nil
}

var (
	errNoOrderSection = &ParseError{msg: "no 'Recommended Order' section found in AI response"}
	errNoTasksFound   = &ParseError{msg: "no task IDs found in AI response"}
)

// ParseError represents a parsing error.
type ParseError struct {
	msg string
}

func (e *ParseError) Error() string {
	return e.msg
}
