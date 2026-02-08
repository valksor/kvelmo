package commands

import (
	"fmt"
	"strings"

	"github.com/valksor/go-mehrhof/internal/agent"
	routercommands "github.com/valksor/go-mehrhof/internal/conductor/commands"
	mehrhofdisplay "github.com/valksor/go-mehrhof/internal/display"
	"github.com/valksor/go-mehrhof/internal/events"
	"github.com/valksor/go-mehrhof/internal/workflow"
	"github.com/valksor/go-toolkit/display"
	"github.com/valksor/go-toolkit/eventbus"
)

// handleAgentEvent processes an agent streaming event.
func (s *InteractiveSession) handleAgentEvent(event agent.Event) error {
	switch event.Type {
	case agent.EventText:
		fmt.Print(event.Text)
		s.transcript.WriteString(event.Text)

	case agent.EventToolUse:
		if event.ToolCall != nil {
			s.printf(false, "\n→ %s\n", display.Muted(event.ToolCall.Name))
		}

	case agent.EventToolResult, agent.EventFile, agent.EventError, agent.EventUsage, agent.EventComplete:
		// Ignore other event types for display purposes
	}

	// Also publish to eventbus for other listeners
	s.cond.GetEventBus().PublishRaw(eventbus.Event{
		Type: events.TypeAgentMessage,
		Data: map[string]any{"event": event},
	})

	return nil
}

// renderResult formats and displays a router result for CLI output.
func (s *InteractiveSession) renderResult(result *routercommands.Result) {
	switch result.Type {
	case routercommands.ResultMessage:
		s.printf(true, "%s %s\n", display.SuccessMsg("✓"), result.Message)

	case routercommands.ResultStatus:
		if data, ok := result.Data.(routercommands.StatusData); ok {
			s.printf(true, "\n%s\n", display.Bold("Task Status:"))
			if data.TaskID != "" {
				s.printf(true, "  ID:      %s\n", data.TaskID)
			}
			if data.Title != "" {
				s.printf(true, "  Title:   %s\n", data.Title)
			}
			s.printf(true, "  State:   %s\n", mehrhofdisplay.ColorState(data.State, data.State))
			if data.Branch != "" {
				s.printf(true, "  Branch:  %s\n", display.Cyan(data.Branch))
			}
			if data.SpecCount > 0 {
				s.printf(true, "  Specs:   %d\n", data.SpecCount)
			}
			s.printf(true, "\n")
		} else {
			s.printf(true, "%s\n", result.Message)
		}

	case routercommands.ResultCost:
		if data, ok := result.Data.(routercommands.CostData); ok {
			s.printf(true, "\n%s\n", display.Bold("Cost Summary:"))
			s.printf(true, "  Input tokens:   %d\n", data.InputTokens)
			s.printf(true, "  Output tokens:  %d\n", data.OutputTokens)
			s.printf(true, "  Cached tokens:  %d\n", data.CachedTokens)
			s.printf(true, "  Total tokens:   %d\n", data.TotalTokens)
			s.printf(true, "  Total cost:     $%.4f\n", data.TotalCostUSD)
			s.printf(true, "\n")
		} else {
			s.printf(true, "%s\n", result.Message)
		}

	case routercommands.ResultBudget:
		if data, ok := result.Data.(routercommands.BudgetData); ok {
			s.printf(true, "\n%s\n", display.Bold("Budget Status:"))
			s.printf(true, "  Type:       %s\n", data.Type)
			s.printf(true, "  Used:       %s\n", data.Used)
			s.printf(true, "  Max:        %s\n", data.Max)
			s.printf(true, "  Percentage: %.1f%%\n", data.Percentage)
			if data.Warned {
				s.printf(true, "  %s\n", display.WarningMsg("Warning threshold reached"))
			}
			s.printf(true, "\n")
		} else {
			s.printf(true, "%s\n", result.Message)
		}

	case routercommands.ResultList:
		s.printf(true, "\n%s\n", display.Bold(result.Message))
		switch items := result.Data.(type) {
		case []routercommands.TaskListItem:
			for _, item := range items {
				prefix := "  "
				if result.TaskID != "" && item.ID == result.TaskID {
					prefix = "* "
				}
				title := item.Title
				if len(title) > 40 {
					title = title[:37] + "..."
				}
				s.printf(true, "%s%s [%s] %s\n",
					prefix, display.Cyan(item.ID),
					mehrhofdisplay.ColorState(item.State, item.State), title)
			}
		case []routercommands.SpecificationItem:
			for _, item := range items {
				icon := mehrhofdisplay.GetSpecificationStatusIcon(item.Status)
				title := item.Title
				if title == "" {
					title = "(untitled)"
				}
				if len(title) > 50 {
					title = title[:47] + "..."
				}
				s.printf(true, "  %s spec-%d: %s [%s]\n",
					icon, item.Number, title, item.Status)
			}
		default:
			// Generic list handling
			s.printf(true, "  %v\n", result.Data)
		}
		s.printf(true, "\n")

	case routercommands.ResultHelp:
		// Help output is handled by printHelp, but router can provide command info
		if cmds, ok := result.Data.([]routercommands.CommandInfo); ok {
			s.printf(true, "\n%s\n", display.Bold("Available Commands:"))
			categories := make(map[string][]routercommands.CommandInfo)
			for _, cmd := range cmds {
				categories[cmd.Category] = append(categories[cmd.Category], cmd)
			}
			for category, commands := range categories {
				s.printf(true, "\n%s\n", display.Bold(capitalizeFirst(category)+":"))
				for _, cmd := range commands {
					aliases := ""
					if len(cmd.Aliases) > 0 {
						aliases = " (aliases: " + strings.Join(cmd.Aliases, ", ") + ")"
					}
					s.printf(true, "  %-15s %s%s\n", cmd.Name, cmd.Description, display.Muted(aliases))
				}
			}
			s.printf(true, "\n")
		} else {
			s.printHelp()
		}

	case routercommands.ResultError:
		s.printf(false, "%s %s\n", display.ErrorMsg("Error:"), result.Message)

	case routercommands.ResultChat:
		// Chat responses display the message directly
		s.printf(true, "%s\n", result.Message)

	case routercommands.ResultSpecifications:
		// Specifications are handled as a list
		s.printf(true, "\n%s\n", display.Bold(result.Message))
		if items, ok := result.Data.([]routercommands.SpecificationItem); ok {
			for _, item := range items {
				icon := mehrhofdisplay.GetSpecificationStatusIcon(item.Status)
				title := item.Title
				if title == "" {
					title = "(untitled)"
				}
				if len(title) > 50 {
					title = title[:47] + "..."
				}
				s.printf(true, "  %s spec-%d: %s [%s]\n",
					icon, item.Number, title, item.Status)
			}
		}
		s.printf(true, "\n")

	case routercommands.ResultQuestion:
		// Question results require user input
		s.printf(true, "\n%s\n", display.WarningMsg("Question:"))
		s.printf(true, "  %s\n", result.Message)
		s.printf(true, "\n")

	case routercommands.ResultWaiting:
		s.printf(true, "\n%s\n", display.WarningMsg("Agent is waiting for your answer"))
		if waiting, ok := result.Data.(routercommands.WaitingData); ok {
			s.printf(true, "  %s\n", waiting.Question)
			for i, opt := range waiting.Options {
				s.printf(true, "  %d. %s\n", i+1, opt.Label)
			}
		} else if result.Message != "" {
			s.printf(true, "  %s\n", result.Message)
		}
		s.printf(true, "\n")

	case routercommands.ResultPaused:
		s.printf(true, "%s %s\n", display.WarningMsg("Paused:"), result.Message)

	case routercommands.ResultStopped:
		s.printf(false, "%s %s\n", display.ErrorMsg("Stopped:"), result.Message)

	case routercommands.ResultConflict:
		s.printf(false, "%s %s\n", display.ErrorMsg("Conflict:"), result.Message)

	case routercommands.ResultExit:
		// Exit is handled before renderResult is called, but include for exhaustiveness
		// Nothing to display
	}
}

// handleEvent processes events from the event bus.
func (s *InteractiveSession) handleEvent(e eventbus.Event) {
	switch e.Type {
	case events.TypeStateChanged:
		if to, ok := e.Data["to"].(string); ok {
			s.state = workflow.State(to)
			s.printf(true, "\n[%s]\n", mehrhofdisplay.ColorState(to, to))
		}

	case events.TypeProgress:
		if msg, ok := e.Data["message"].(string); ok {
			s.printf(false, "  %s\n", display.Muted(msg))
		}

	case events.TypeFileChanged:
		if path, ok := e.Data["path"].(string); ok {
			if op, ok := e.Data["operation"].(string); ok {
				s.printf(false, "  [%s] %s\n", display.Muted(op), display.Cyan(path))
			}
		}

	case events.TypeError:
		if errMsg, ok := e.Data["error"].(string); ok {
			s.printf(false, "%s %s\n", display.ErrorMsg("Error:"), errMsg)
		}
	}
}
