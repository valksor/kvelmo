package server

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/valksor/go-mehrhof/internal/browser"
	"github.com/valksor/go-mehrhof/internal/conductor"
	"github.com/valksor/go-mehrhof/internal/stack"
	"github.com/valksor/go-mehrhof/internal/storage"
	"github.com/valksor/go-mehrhof/internal/template"
	"github.com/valksor/go-mehrhof/internal/validation"
)

// executeInteractiveToolsCommand handles tools and meta interactive commands.
// Commands: browser, project, stack, config, agents, providers, templates, scan, commit, help.
func (s *Server) executeInteractiveToolsCommand(ctx context.Context, command string, args []string) (string, error) {
	cond := s.config.Conductor

	switch command {
	case "browser":
		ctrl := cond.GetBrowser(ctx)
		if ctrl == nil {
			return "", errors.New("browser not configured. Start the browser with 'mehr browser --keep-alive status'")
		}
		subcommand := "status"
		subArgs := []string{}
		if len(args) > 0 {
			subcommand = strings.ToLower(args[0])
			subArgs = args[1:]
		}

		switch subcommand {
		case "status":
			tabs, tabErr := ctrl.ListTabs(ctx)
			if tabErr != nil {
				return "", tabErr
			}
			var lines []string
			lines = append(lines, fmt.Sprintf("Connected to Chrome (port %d)", ctrl.GetPort()))
			lines = append(lines, fmt.Sprintf("Tabs: %d", len(tabs)))
			for i, tab := range tabs {
				title := tab.Title
				if len(title) > 50 {
					title = title[:47] + "..."
				}
				lines = append(lines, fmt.Sprintf("  %d. [%s] %s", i+1, tab.ID[:8], title))
			}

			return strings.Join(lines, "\n"), nil

		case "tabs":
			tabs, tabErr := ctrl.ListTabs(ctx)
			if tabErr != nil {
				return "", tabErr
			}
			if len(tabs) == 0 {
				return "No tabs open", nil
			}
			var lines []string
			for i, tab := range tabs {
				lines = append(lines, fmt.Sprintf("%d. [%s] %s\n   %s", i+1, tab.ID, tab.Title, tab.URL))
			}

			return strings.Join(lines, "\n"), nil

		case "goto":
			if len(subArgs) == 0 {
				return "", errors.New("usage: browser goto <url>")
			}
			tab, openErr := ctrl.OpenTab(ctx, subArgs[0])
			if openErr != nil {
				return "", openErr
			}

			return fmt.Sprintf("Opened: %s\nTab ID: %s", tab.Title, tab.ID), nil

		case "navigate":
			if len(subArgs) == 0 {
				return "", errors.New("usage: browser navigate <url>")
			}
			tabs, tabErr := ctrl.ListTabs(ctx)
			if tabErr != nil || len(tabs) == 0 {
				return "", errors.New("no tabs open")
			}
			if navErr := ctrl.Navigate(ctx, tabs[0].ID, subArgs[0]); navErr != nil {
				return "", navErr
			}

			return "Navigated to: " + subArgs[0], nil

		case "close":
			if len(subArgs) == 0 {
				return "", errors.New("usage: browser close <tab-id>")
			}
			if closeErr := ctrl.CloseTab(ctx, subArgs[0]); closeErr != nil {
				return "", closeErr
			}

			return "Closed tab: " + subArgs[0], nil

		case "reload":
			tabs, tabErr := ctrl.ListTabs(ctx)
			if tabErr != nil || len(tabs) == 0 {
				return "", errors.New("no tabs open")
			}
			hard := len(subArgs) > 0 && subArgs[0] == "--hard"
			if reloadErr := ctrl.Reload(ctx, tabs[0].ID, hard); reloadErr != nil {
				return "", reloadErr
			}

			return "Page reloaded", nil

		case "screenshot":
			tabs, tabErr := ctrl.ListTabs(ctx)
			if tabErr != nil || len(tabs) == 0 {
				return "", errors.New("no tabs open")
			}
			opts := browser.ScreenshotOptions{Format: "png", Quality: 80}
			data, ssErr := ctrl.Screenshot(ctx, tabs[0].ID, opts)
			if ssErr != nil {
				return "", ssErr
			}

			return fmt.Sprintf("Screenshot captured (%d bytes, use Web UI to view)", len(data)), nil

		case "click":
			if len(subArgs) == 0 {
				return "", errors.New("usage: browser click <selector>")
			}
			tabs, tabErr := ctrl.ListTabs(ctx)
			if tabErr != nil || len(tabs) == 0 {
				return "", errors.New("no tabs open")
			}
			selector := strings.Join(subArgs, " ")
			if clickErr := ctrl.Click(ctx, tabs[0].ID, selector); clickErr != nil {
				return "", clickErr
			}

			return "Clicked: " + selector, nil

		case "type":
			if len(subArgs) < 2 {
				return "", errors.New("usage: browser type <selector> <text>")
			}
			tabs, tabErr := ctrl.ListTabs(ctx)
			if tabErr != nil || len(tabs) == 0 {
				return "", errors.New("no tabs open")
			}
			selector := subArgs[0]
			text := strings.Join(subArgs[1:], " ")
			if typeErr := ctrl.Type(ctx, tabs[0].ID, selector, text, false); typeErr != nil {
				return "", typeErr
			}

			return "Typed into: " + selector, nil

		case "dom":
			if len(subArgs) == 0 {
				return "", errors.New("usage: browser dom <selector>")
			}
			tabs, tabErr := ctrl.ListTabs(ctx)
			if tabErr != nil || len(tabs) == 0 {
				return "", errors.New("no tabs open")
			}
			selector := strings.Join(subArgs, " ")
			elem, domErr := ctrl.QuerySelector(ctx, tabs[0].ID, selector)
			if domErr != nil {
				return "", domErr
			}
			if elem == nil {
				return "No element found for: " + selector, nil
			}
			text := elem.TextContent
			if len(text) > 100 {
				text = text[:97] + "..."
			}

			return fmt.Sprintf("<%s>\nText: %s\nVisible: %v", elem.TagName, text, elem.Visible), nil

		case "eval":
			if len(subArgs) == 0 {
				return "", errors.New("usage: browser eval <expression>")
			}
			tabs, tabErr := ctrl.ListTabs(ctx)
			if tabErr != nil || len(tabs) == 0 {
				return "", errors.New("no tabs open")
			}
			expression := strings.Join(subArgs, " ")
			evalResult, evalErr := ctrl.Eval(ctx, tabs[0].ID, expression)
			if evalErr != nil {
				return "", evalErr
			}

			return fmt.Sprintf("Result: %v", evalResult), nil

		case "console":
			tabs, tabErr := ctrl.ListTabs(ctx)
			if tabErr != nil || len(tabs) == 0 {
				return "", errors.New("no tabs open")
			}
			duration := 3 * time.Second
			messages, consoleErr := ctrl.GetConsoleLogs(ctx, tabs[0].ID, duration)
			if consoleErr != nil {
				return "", consoleErr
			}
			if len(messages) == 0 {
				return "No console messages captured", nil
			}
			var lines []string
			for _, msg := range messages {
				lines = append(lines, fmt.Sprintf("[%s] %s", msg.Level, msg.Text))
			}

			return strings.Join(lines, "\n"), nil

		case "network":
			tabs, tabErr := ctrl.ListTabs(ctx)
			if tabErr != nil || len(tabs) == 0 {
				return "", errors.New("no tabs open")
			}
			duration := 3 * time.Second
			requests, netErr := ctrl.GetNetworkRequests(ctx, tabs[0].ID, duration)
			if netErr != nil {
				return "", netErr
			}
			if len(requests) == 0 {
				return "No network requests captured", nil
			}
			var lines []string
			for _, req := range requests {
				status := ""
				if req.Status > 0 {
					status = fmt.Sprintf(" → %d", req.Status)
				}
				lines = append(lines, fmt.Sprintf("[%s] %s %s%s",
					req.Timestamp.Format("15:04:05"), req.Method, truncateURL(req.URL, 60), status))
			}

			return strings.Join(lines, "\n"), nil

		case "source":
			tabs, tabErr := ctrl.ListTabs(ctx)
			if tabErr != nil || len(tabs) == 0 {
				return "", errors.New("no tabs open")
			}
			source, sourceErr := ctrl.GetPageSource(ctx, tabs[0].ID)
			if sourceErr != nil {
				return "", sourceErr
			}
			if len(source) > 2000 {
				source = source[:2000] + "\n... (truncated)"
			}

			return fmt.Sprintf("Page source (%d bytes):\n%s", len(source), source), nil

		default:
			return fmt.Sprintf("Unknown browser subcommand: %s\nUse: status, tabs, goto, navigate, close, reload, screenshot, click, type, dom, eval, console, network, source", subcommand), nil
		}

	case "project":
		ws := cond.GetWorkspace()
		if ws == nil {
			return "", errors.New("workspace not initialized")
		}
		subcommand := "help"
		subArgs := []string{}
		if len(args) > 0 {
			subcommand = strings.ToLower(args[0])
			subArgs = args[1:]
		}

		switch subcommand {
		case "plan":
			if len(subArgs) == 0 {
				return "", errors.New("usage: project plan <source>")
			}
			source := subArgs[0]
			opts := conductor.ProjectPlanOptions{}
			// Parse optional --title flag
			for i := 1; i < len(subArgs); i++ {
				if subArgs[i] == "--title" && i+1 < len(subArgs) {
					opts.Title = subArgs[i+1]
					i++
				}
			}
			planResult, planErr := cond.CreateProjectPlan(ctx, source, opts)
			if planErr != nil {
				return "", planErr
			}
			result := fmt.Sprintf("Created queue: %s\n  %d tasks identified", planResult.Queue.ID, len(planResult.Tasks))
			if len(planResult.Questions) > 0 {
				result += fmt.Sprintf("\n  %d questions to resolve", len(planResult.Questions))
			}

			return result, nil

		case "tasks":
			var queueID string
			var err error
			if len(subArgs) > 0 {
				queueID = subArgs[0]
			} else {
				queues, listErr := ws.ListQueues()
				if listErr != nil {
					err = listErr
				} else if len(queues) == 0 {
					err = errors.New("no queues found")
				} else {
					queueID = queues[len(queues)-1]
				}
			}
			if err != nil {
				return "", err
			}
			if queueID != "" {
				queue, loadErr := storage.LoadTaskQueue(ws, queueID)
				if loadErr != nil {
					return "", loadErr
				}
				var lines []string
				lines = append(lines, fmt.Sprintf("Queue: %s (%d tasks)", queue.ID, len(queue.Tasks)))
				for _, task := range queue.Tasks {
					lines = append(lines, fmt.Sprintf("  • %s: %s [%s]", task.ID, truncateStr(task.Title, 40), task.Status))
				}

				return strings.Join(lines, "\n"), nil
			}

			return "", nil

		case "edit":
			if len(subArgs) == 0 {
				return "", errors.New("usage: project edit <task-id> [--title <title>] [--status <status>]")
			}
			taskID := subArgs[0]
			queues, listErr := ws.ListQueues()
			if listErr != nil {
				return "", listErr
			}
			if len(queues) == 0 {
				return "", errors.New("no queues found")
			}
			queueID := queues[len(queues)-1]
			queue, loadErr := storage.LoadTaskQueue(ws, queueID)
			if loadErr != nil {
				return "", loadErr
			}
			updateErr := queue.UpdateTask(taskID, func(task *storage.QueuedTask) {
				for i := 1; i < len(subArgs); i++ {
					switch subArgs[i] {
					case "--title":
						if i+1 < len(subArgs) {
							task.Title = subArgs[i+1]
							i++
						}
					case "--status":
						if i+1 < len(subArgs) {
							task.Status = storage.TaskStatus(subArgs[i+1])
							i++
						}
					case "--priority":
						if i+1 < len(subArgs) {
							if p, pErr := strconv.Atoi(subArgs[i+1]); pErr == nil {
								task.Priority = p
							}
							i++
						}
					}
				}
			})
			if updateErr != nil {
				return "", updateErr
			}
			if saveErr := queue.Save(); saveErr != nil {
				return "", saveErr
			}

			return "Updated task: " + taskID, nil

		case "reorder":
			if len(subArgs) == 0 {
				return "", errors.New("usage: project reorder <task-id> --before|--after <target-id>")
			}
			if len(subArgs) >= 3 {
				taskID := subArgs[0]
				queues, listErr := ws.ListQueues()
				if listErr != nil {
					return "", listErr
				}
				if len(queues) == 0 {
					return "", errors.New("no queues found")
				}
				queueID := queues[len(queues)-1]
				queue, loadErr := storage.LoadTaskQueue(ws, queueID)
				if loadErr != nil {
					return "", loadErr
				}
				var targetIndex int
				for i := 1; i < len(subArgs); i++ {
					if subArgs[i] == "--before" && i+1 < len(subArgs) {
						for j, t := range queue.Tasks {
							if t.ID == subArgs[i+1] {
								targetIndex = j

								break
							}
						}
					} else if subArgs[i] == "--after" && i+1 < len(subArgs) {
						for j, t := range queue.Tasks {
							if t.ID == subArgs[i+1] {
								targetIndex = j + 1

								break
							}
						}
					}
				}
				if reorderErr := queue.ReorderTask(taskID, targetIndex); reorderErr != nil {
					return "", reorderErr
				}
				if saveErr := queue.Save(); saveErr != nil {
					return "", saveErr
				}

				return fmt.Sprintf("Moved task %s to position %d", taskID, targetIndex+1), nil
			}

			return "", errors.New("usage: project reorder <task-id> --before|--after <target-id>")

		case "submit":
			if len(subArgs) < 1 {
				return "", errors.New("usage: project submit --provider <provider>")
			}
			var providerName string
			for i := 0; i < len(subArgs); i++ {
				if subArgs[i] == "--provider" && i+1 < len(subArgs) {
					providerName = subArgs[i+1]
					i++
				}
			}
			if providerName == "" {
				return "", errors.New("--provider is required")
			}
			queues, listErr := ws.ListQueues()
			if listErr != nil {
				return "", listErr
			}
			if len(queues) == 0 {
				return "", errors.New("no queues found")
			}
			queueID := queues[len(queues)-1]
			opts := conductor.SubmitOptions{Provider: providerName}
			submitResult, submitErr := cond.SubmitProjectTasks(ctx, queueID, opts)
			if submitErr != nil {
				return "", submitErr
			}

			return fmt.Sprintf("Submitted %d tasks to %s", len(submitResult.Tasks), providerName), nil

		case "start":
			queues, listErr := ws.ListQueues()
			if listErr != nil {
				return "", listErr
			}
			if len(queues) == 0 {
				return "", errors.New("no queues found")
			}
			queueID := queues[len(queues)-1]
			task, startErr := cond.StartNextTask(ctx, queueID)
			if startErr != nil {
				return "", startErr
			}

			return fmt.Sprintf("Started task: %s - %s", task.ID, task.Title), nil

		case "sync":
			if len(subArgs) == 0 {
				return "", errors.New("usage: project sync <provider:reference>")
			}
			reference := subArgs[0]
			opts := conductor.SyncProjectOptions{}
			syncResult, syncErr := cond.SyncProject(ctx, reference, opts)
			if syncErr != nil {
				return "", syncErr
			}

			return fmt.Sprintf("Synced project: %s\n  Queue: %s\n  Tasks: %d synced", syncResult.Queue.Title, syncResult.Queue.ID, syncResult.TasksSync), nil

		default:
			return `Project commands:
• project plan <source> - Create task breakdown from source
• project tasks [queue-id] - View tasks in queue
• project edit <task-id> --title|--status|--priority - Edit task
• project reorder <task-id> --before|--after <target> - Reorder tasks
• project submit --provider <name> - Submit to provider
• project start - Start next task from queue
• project sync <provider:ref> - Sync from provider`, nil
		}

	case "stack":
		ws := cond.GetWorkspace()
		if ws == nil {
			return "", errors.New("workspace not initialized")
		}
		subcommand := "list"
		subArgs := []string{}
		if len(args) > 0 {
			subcommand = strings.ToLower(args[0])
			subArgs = args[1:]
		}

		stackStorage := stack.NewStorage(ws.DataRoot())
		if loadErr := stackStorage.Load(); loadErr != nil {
			return "", fmt.Errorf("load stacks: %w", loadErr)
		}
		switch subcommand {
		case "list", "ls":
			stacks := stackStorage.ListStacks()
			if len(stacks) == 0 {
				return "No stacked features found.\nUse 'mehr start <task> --depends-on <parent>' to create a stacked feature.", nil
			}
			var lines []string
			for _, s := range stacks {
				lines = append(lines, fmt.Sprintf("Stack: %s (%d tasks)", s.ID, s.TaskCount()))
				for _, t := range s.Tasks {
					lines = append(lines, fmt.Sprintf("  • %s [%s] %s", t.ID, t.State, t.Branch))
				}
			}

			return strings.Join(lines, "\n"), nil

		case "rebase":
			git := cond.GetGit()
			if git == nil {
				return "", errors.New("not in a git repository")
			}
			rebaser := stack.NewRebaser(stackStorage, git)
			if len(subArgs) > 0 {
				// Single task rebase
				taskID := subArgs[0]
				preview, previewErr := rebaser.PreviewTask(ctx, taskID)
				if previewErr != nil {
					return "", previewErr
				}
				if preview.WouldConflict {
					return "", fmt.Errorf("cannot rebase %s: conflicts detected", taskID)
				}
				rebaseResult, rebaseErr := rebaser.RebaseTask(ctx, taskID)
				if rebaseErr != nil {
					return "", rebaseErr
				}
				if rebaseResult.FailedTask != nil {
					return "", fmt.Errorf("rebase failed for %s", rebaseResult.FailedTask.TaskID)
				}

				return "Rebased task " + taskID, nil
			}
			// Rebase all needing it
			var stacksWithRebase []*stack.Stack
			for _, s := range stackStorage.ListStacks() {
				if len(s.GetTasksNeedingRebase()) > 0 {
					stacksWithRebase = append(stacksWithRebase, s)
				}
			}
			if len(stacksWithRebase) == 0 {
				return "No tasks need rebasing", nil
			}
			var rebased int
			for _, s := range stacksWithRebase {
				rebaseResult, rebaseErr := rebaser.RebaseAll(ctx, s.ID)
				if rebaseErr != nil {
					return "", rebaseErr
				}
				rebased += len(rebaseResult.RebasedTasks)
			}

			return fmt.Sprintf("Rebased %d task(s)", rebased), nil

		case "sync":
			return "Stack sync requires provider configuration. Use CLI for full sync.", nil

		default:
			return `Stack commands:
• stack - List stacked features
• stack rebase [task-id] - Rebase stacked tasks
• stack sync - Sync PR status`, nil
		}

	case "config":
		// Config validate/explain commands
		if len(args) > 0 {
			subCmd := args[0]
			switch subCmd {
			case "validate":
				ws := s.config.Conductor.GetWorkspace()
				if ws == nil {
					return "Workspace not initialized", nil
				}
				validator := validation.New(ws.Root(), validation.Options{})
				validationResult, validateErr := validator.Validate(ctx)
				if validateErr != nil {
					return fmt.Sprintf("Validation error: %s", validateErr), nil
				}
				if validationResult.Valid {
					return "✓ Configuration is valid", nil
				}

				return validationResult.Format("text"), nil
			case "explain":
				// Config explain requires step argument and complex resolution
				return "Config explain requires step argument. Use CLI: mehr config explain --agent <planning|implementing|reviewing>", nil
			default:
				return `Config commands:
• config validate - Check configuration for issues
• config explain --agent <step> - Explain agent resolution (CLI only)`, nil
			}
		}

		return `Config commands:
• config validate - Check configuration for issues
• config explain --agent <step> - Explain agent resolution (CLI only)`, nil

	case "agents":
		// Agents list/explain commands
		if len(args) > 0 {
			subCmd := args[0]
			switch subCmd {
			case "list":
				registry := s.config.Conductor.GetAgentRegistry()
				if registry == nil {
					return "Agent registry not initialized", nil
				}
				agentNames := registry.List()
				if len(agentNames) == 0 {
					return "No agents configured", nil
				}
				var sb strings.Builder
				sb.WriteString("Available agents:\n")
				for _, name := range agentNames {
					sb.WriteString("• " + name)
					// Check availability
					if ag, agErr := registry.Get(name); agErr == nil {
						if ag.Available() != nil {
							sb.WriteString(" (unavailable)")
						}
					}
					sb.WriteString("\n")
				}

				return sb.String(), nil
			case "explain":
				if len(args) > 1 {
					agentName := args[1]
					registry := s.config.Conductor.GetAgentRegistry()
					if registry == nil {
						return "Agent registry not initialized", nil
					}
					ag, agErr := registry.Get(agentName)
					if agErr != nil {
						return "Agent not found: " + agentName, nil //nolint:nilerr // interactive UI converts lookup errors to user messages
					}
					var sb strings.Builder
					sb.WriteString(fmt.Sprintf("Agent: %s\n", ag.Name()))
					if ag.Available() == nil {
						sb.WriteString("Status: Available\n")
					} else {
						sb.WriteString(fmt.Sprintf("Status: Unavailable (%s)\n", ag.Available()))
					}

					return sb.String(), nil
				}

				return "Usage: agents explain <name>", nil
			default:
				return `Agents commands:
• agents list - List available agents
• agents explain <name> - Show agent details`, nil
			}
		}

		return `Agents commands:
• agents list - List available agents
• agents explain <name> - Show agent details`, nil

	case "providers":
		// Providers list/info/status commands
		if len(args) > 0 {
			subCmd := args[0]
			switch subCmd {
			case "list":
				return `Available providers:
• file (f) - Single markdown file
• dir (d) - Directory with README.md
• github (gh) - GitHub issues and pull requests
• gitlab - GitLab issues and merge requests
• jira - Atlassian Jira tickets
• linear - Linear issues
• notion - Notion pages and databases
• wrike - Wrike tasks
• youtrack (yt) - JetBrains YouTrack issues

Usage: mehr start <scheme>:<reference>`, nil
			case "info":
				if len(args) > 1 {
					providerName := strings.ToLower(args[1])
					info := getProviderInfoText(providerName)
					if info != "" {
						return info, nil
					}

					return fmt.Sprintf("Unknown provider: %s\nRun 'providers list' to see available providers.", providerName), nil
				}

				return "Usage: providers info <name>", nil
			case "status":
				// Provider status requires conductor initialization and health checks
				return "Provider status check requires CLI. Use: mehr providers status", nil
			default:
				return `Providers commands:
• providers list - List available providers
• providers info <name> - Show provider details
• providers status - Check connection status (CLI only)`, nil
			}
		}

		return `Providers commands:
• providers list - List available providers
• providers info <name> - Show provider details
• providers status - Check connection status (CLI only)`, nil

	case "templates":
		// Templates list/show/apply commands
		if len(args) > 0 {
			subCmd := args[0]
			switch subCmd {
			case "list":
				templateNames := template.BuiltInTemplates()
				if len(templateNames) == 0 {
					return "No templates available", nil
				}
				var sb strings.Builder
				sb.WriteString("Available templates:\n")
				for _, name := range templateNames {
					if tpl, tplErr := template.LoadBuiltIn(name); tplErr == nil {
						sb.WriteString("• " + name)
						if tpl.Description != "" {
							sb.WriteString(" - " + tpl.Description)
						}
						sb.WriteString("\n")
					} else {
						sb.WriteString(fmt.Sprintf("• %s\n", name))
					}
				}

				return sb.String(), nil
			case "show":
				if len(args) > 1 {
					templateName := args[1]
					tpl, tplErr := template.LoadBuiltIn(templateName)
					if tplErr != nil {
						return "Template not found: " + templateName, nil //nolint:nilerr // interactive UI converts lookup errors to user messages
					}

					return tpl.GetDescription(), nil
				}

				return "Usage: templates show <name>", nil
			case "apply":
				return "Template apply requires file selection. Use CLI: mehr templates apply <name> <file>", nil
			default:
				return `Templates commands:
• templates list - List available templates
• templates show <name> - Show template content
• templates apply <name> <file> - Apply template (CLI only)`, nil
			}
		}

		return `Templates commands:
• templates list - List available templates
• templates show <name> - Show template content
• templates apply <name> <file> - Apply template (CLI only)`, nil

	case "scan":
		// Security scanning
		return "Security scanning requires CLI for full output. Use: mehr scan [--gosec] [--gitleaks] [--govulncheck]", nil

	case "commit":
		// AI commit assistance
		return "Commit assistance requires CLI for git integration. Use: mehr commit [--analyze] [--preview] [--execute]", nil

	case "help":
		return `Commands:
• start <ref> - Start a task
• plan - Run planning
• implement - Run implementation
• implement review <n> - Fix issues from review
• review - Run code review
• review <n> - View review content
• continue - Resume paused
• finish - Complete task
• abandon - Discard task
• auto - Auto-execute next step
• reset - Reset workflow to idle
• undo/redo - Checkpoints
• status - Show status
• cost - Show token usage
• budget - Show budget status
• list - List tasks
• note <msg> - Add a note
• question <msg> - Ask agent a question
• answer <resp> - Answer agent question
• find <query> - Search code
• memory search <query> - Search similar tasks
• memory index <task-id> - Index task to memory
• memory stats - Show memory statistics
• library list/show/search/pull/remove/stats - Documentation library
• links list/backlinks/search/stats/rebuild - Entity links
• browser status/tabs/goto/navigate/close/reload - Browser automation
• browser screenshot/click/type/dom/eval - Page interaction
• browser console/network/source - DevTools data
• project plan/tasks/edit/reorder/submit/start/sync - Project planning
• stack/stack rebase/stack sync - Stacked features
• specification [n] - View specifications
• quick <desc> - Create quick task
• delete <queue>/<id> - Delete queue task
• export <queue>/<id> - Export task to markdown
• optimize <queue>/<id> - AI optimize task
• submit <queue>/<id> <provider> - Submit to provider
• sync - Sync task from provider
• simplify - Simplify code
• label [add|rm] <labels> - Manage labels
• config validate/explain - Configuration validation
• agents list/explain - Agent management
• providers list/info/status - Provider management
• templates list/show - Template management
• scan - Security scanning (CLI)
• commit - AI commit assistance (CLI)
• help - Show this help`, nil

	default:
		return "", fmt.Errorf("unknown tools command: %s", command)
	}
}
