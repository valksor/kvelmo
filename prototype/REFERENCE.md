# REFERENCE.md

Comprehensive reference for LLMs working with go-mehrhof. This file documents **what exists** — commands, endpoints, tools, interfaces, and data flows — so you can work effectively without exploring the codebase.

- For **coding rules and guidelines**, see [CLAUDE.md](CLAUDE.md)
- For **detailed documentation**, see [docs/](docs/)
- For **agent configuration**, see [AGENTS.md](AGENTS.md)

---

## 1. CLI Commands

All commands are invoked as `mehr <command>`. Global flags: `--verbose/-v`, `--quiet/-q`, `--no-color`, `--sandbox`.

### Workflow

| Command | Description | Key Flags | Conductor Method |
|---------|-------------|-----------|-----------------|
| `mehr start <ref>` | Start a new task from file, directory, or provider | `--agent/-A`, `--no-branch`, `--worktree/-w`, `--stash`, `--key/-k`, `--title`, `--slug`, `--commit-prefix`, `--branch-pattern`, `--template`, `--depends-on`, `--agent-plan`, `--agent-implement`, `--agent-review`, `--parallel/-p` | `Start()` |
| `mehr plan` | Generate implementation specifications | `--standalone`, `--seed/-s`, `--full-context`, `--agent-plan`, `--auto-approve`, `--optimize`, `--force`, `--with-parent`, `--without-parent`, `--with-siblings`, `--max-siblings` | `Plan()` |
| `mehr implement` | Execute specifications with AI agent | `--dry-run/-n`, `--agent-implement`, `--optimize`, `--only`, `--parallel`, `--force`, `--with-parent`, `--without-parent`, `--with-siblings`, `--max-siblings` | `Implement()` |
| `mehr implement review <n>` | Implement fixes from a specific review | `--dry-run/-n`, `--agent-implement`, `--optimize`, `--force` | `ImplementReview()` |
| `mehr review` | Run code review on changes | `--tool`, `--output/-o`, `--agent-review`, `--optimize`, `--force`, `--standalone`, `--branch`, `--range`, `--context`, `--fix`, `--checkpoint` | `Review()` |
| `mehr review view <n>` | View content of a specific review | `--output/-o`, `--all` | Storage only |
| `mehr finish` | Complete task, create PR | `--yes/-y`, `--merge`, `--delete`, `--push`, `--squash`, `--target/-t`, `--no-quality`, `--quality-target`, `--delete-work`, `--draft`, `--pr-title`, `--pr-body` | `Finish()` |
| `mehr continue` | Resume work on task (alias: `c`) | `--auto` | `Continue()` |
| `mehr abandon` | Discard current task | `--yes/-y`, `--keep-branch`, `--keep-work` | `Delete()` |
| `mehr reset` | Reset workflow state to idle | `--yes/-y` | `ResetState()` |
| `mehr undo` | Revert to previous checkpoint | `--yes/-y` | `Undo()` |
| `mehr redo` | Restore next checkpoint | `--yes/-y` | `Redo()` |
| `mehr auto <ref>` | Full automation: start, plan, implement, quality, finish | `--agent/-a`, `--no-branch`, `--worktree/-w`, `--max-retries`, `--no-push`, `--no-delete`, `--no-squash`, `--target/-t`, `--quality-target`, `--no-quality`, `--optimize`, `--agent-quality` | `RunAuto()` |
| `mehr workflow` | Show workflow state diagram | | `GetMachine()` |
| `mehr guide` | Show recommended next step | | `Guide()` |
| `mehr sync <task-id>` | Sync task from provider and generate delta spec | | `Sync()` |

### Task Management

| Command | Description | Key Flags | Conductor Method |
|---------|-------------|-----------|-----------------|
| `mehr list` | List all tasks in workspace | `--worktrees/-w`, `--search`, `--filter`, `--sort`, `--format`, `--json`, `--label`, `--label-any`, `--no-label`, `--running` | Storage only |
| `mehr status` | Show task details (alias: `st`) | `--all/-a`, `--json`, `--diagram` | `Status()` |
| `mehr delete` | Delete a queue task | `--task` | `ParseQueueTaskRef()` |
| `mehr cost` | Show token usage and costs | `--breakdown`, `--all`, `--summary/-s`, `--json`, `--chart` | Storage only |
| `mehr note` | Add note or answer agent question (alias: `answer`) | `--task`, `--running` | `AppendNote()` |
| `mehr question` | Ask the active agent a question (aliases: `ask`, `q`) | | `AskQuestion()` |
| `mehr find` | AI-powered code search | `--path/-p`, `--pattern`, `--format`, `--stream`, `--agent`, `--context/-C` | `Find()` |
| `mehr export` | Export queue task to markdown | `--task`, `--output` | `ExportQueueTask()` |
| `mehr label add` | Add labels to active task | | Storage only |
| `mehr label remove` | Remove labels from active task | | Storage only |
| `mehr label set` | Set all labels on active task | | Storage only |
| `mehr label list` | List labels on active task | | Storage only |

### Project & Stack

| Command | Description | Key Flags | Conductor Method |
|---------|-------------|-----------|-----------------|
| `mehr project plan` | Plan a project from reference | | `ProjectPlan()` |
| `mehr project submit` | Submit project for execution | | `ProjectSubmit()` |
| `mehr project start` | Start working on project tasks | | `ProjectStart()` |
| `mehr project sync` | Sync project state with provider | | `ProjectSync()` |
| `mehr project tasks` | List project tasks | | Storage only |
| `mehr project edit` | Edit a project task | | Storage only |
| `mehr project reorder` | Reorder project tasks | | Storage only |
| `mehr stack` | List stacked features | | Stack methods |
| `mehr stack sync` | Sync stack with remote | | `StackSync()` |
| `mehr stack rebase` | Rebase stack on base branch | | `StackRebase()` |

### Configuration & Discovery

| Command | Description | Key Flags | Conductor Method |
|---------|-------------|-----------|-----------------|
| `mehr init` | Initialize workspace (creates `.mehrhof/`) | | Config only |
| `mehr config validate` | Validate workspace config | | Config only |
| `mehr config init` | Initialize workspace config | | Config only |
| `mehr config explain` | Explain a config option | | Config only |
| `mehr agents` | List available agents | | Registry only |
| `mehr agents list` | List agents with details | | Registry only |
| `mehr agents explain` | Explain agent configuration | | Registry only |
| `mehr providers` | List available providers | | Registry only |
| `mehr providers list` | List providers with details | | Registry only |
| `mehr providers info` | Show provider info | | Registry only |
| `mehr providers status` | Show provider auth status | | Registry only |
| `mehr templates` | List available templates | | Registry only |
| `mehr templates show` | Show template details | | Registry only |
| `mehr templates apply` | Apply template to task | | Registry only |
| `mehr plugins list` | List installed plugins | | Plugin registry |
| `mehr plugins install` | Install a plugin | | Plugin registry |
| `mehr plugins remove` | Remove a plugin | | Plugin registry |
| `mehr plugins info` | Show plugin info | | Plugin registry |
| `mehr plugins validate` | Validate plugin manifest | | Plugin registry |

### Browser Automation

All browser subcommands share: `--host`, `--port`, `--headless`, `--strict-certs`, `--keep-alive`, `--cookie-profile`.

| Command | Description | Key Flags | Conductor Method |
|---------|-------------|-----------|-----------------|
| `mehr browser status` | Check browser connection | | `Browser.ListTabs()` |
| `mehr browser tabs` | List open browser tabs | | `Browser.ListTabs()` |
| `mehr browser goto <url>` | Open URL in new tab | | `Browser.OpenTab()` |
| `mehr browser navigate <url>` | Navigate current tab | | `Browser.Navigate()` |
| `mehr browser close` | Close a browser tab | | `Browser.CloseTab()` |
| `mehr browser switch <id>` | Switch to a tab | | `Browser.SwitchTab()` |
| `mehr browser reload` | Reload current tab | `--hard` | `Browser.Reload()` |
| `mehr browser screenshot` | Capture screenshot | `--output/-o`, `--format/-f`, `--quality`, `--full-page/-F` | `Browser.Screenshot()` |
| `mehr browser dom` | Query DOM elements | `--selector`, `--all`, `--html`, `--limit` | `Browser.QuerySelector()` |
| `mehr browser click` | Click an element | `--selector` | `Browser.Click()` |
| `mehr browser type` | Type text into element | `--selector`, `--clear` | `Browser.Type()` |
| `mehr browser eval` | Evaluate JavaScript | | `Browser.Eval()` |
| `mehr browser console` | Capture console logs | `--duration/-d`, `--level` | `Browser.GetConsoleLogs()` |
| `mehr browser network` | Capture network requests | `--duration/-d`, `--capture-body`, `--max-body-size` | `Browser.GetNetworkRequests()` |
| `mehr browser source` | Get page HTML source | | `Browser.GetPageSource()` |
| `mehr browser scripts` | List loaded JavaScript sources | `--include-source` | `Browser.GetScriptSources()` |
| `mehr browser websocket` | Monitor WebSocket frames | `--duration/-d` | `Browser.GetWebSocketFrames()` |
| `mehr browser styles` | Inspect CSS styles | `--selector`, `--computed`, `--matched` | `Browser.GetComputedStyles()` |
| `mehr browser coverage` | Track JS/CSS code coverage | `--duration/-d`, `--js`, `--css` | `Browser.GetCoverage()` |
| `mehr browser cookies export` | Export cookies to file | `--output/-o`, `--profile` | `Browser.GetCookies()` |
| `mehr browser cookies import` | Import cookies from file | `--file/-f`, `--profile` | `Browser.SetCookies()` |

### Server & Integration

| Command | Description | Key Flags | Conductor Method |
|---------|-------------|-----------|-----------------|
| `mehr serve` | Start web UI server | `--port/-p`, `--global`, `--open`, `--api` | `Server.Start()` |
| `mehr serve register` | Register project in global registry | `--list/-l` | `Registry.Register()` |
| `mehr serve unregister` | Remove from global registry | | `Registry.Unregister()` |
| `mehr interactive` | Start interactive CLI (REPL + chat) | | Full workflow |
| `mehr mcp` | Start MCP server for AI agents | | MCP integration |
| `mehr scan` | Security/quality scan | | `Scan()` |
| `mehr login` | Authenticate with provider | | Provider auth |
| `mehr submit` | Submit task to external provider | `--provider`, `--task`, `--source` | `Submit()` |

### Utilities

| Command | Description | Key Flags | Conductor Method |
|---------|-------------|-----------|-----------------|
| `mehr commit` | Create logical commits from changes | `--push/-p`, `--all/-a`, `--dry-run/-n`, `--note/-m`, `--agent-commit` | `GenerateCommitMessageForGroup()` |
| `mehr quick` | Create quick/temporary tasks | | Quick task creation |
| `mehr simplify` | Simplify code or specifications | | `Simplify()` |
| `mehr optimize` | Optimize specifications or prompts | | `Optimize()` |
| `mehr links` | Manage bidirectional `[[links]]` | | Links methods |
| `mehr memory` | Semantic memory management | | Memory methods |
| `mehr library pull <source>` | Pull documentation from URL, file, or git | `--name`, `--mode`, `--shared`, `--paths`, `--tag`, `--max-depth`, `--max-pages`, `--dry-run` | Library methods |
| `mehr library list` | List documentation collections | `--shared`, `--project`, `--verbose` | Library methods |
| `mehr library show <name>` | Show collection details | `[page]` | Library methods |
| `mehr library remove <name>` | Remove a collection | `--force` | Library methods |
| `mehr library update [name]` | Update collection(s) from source | | Library methods |
| `mehr review-pr` | Standalone PR review | | `RunPRReview()` |
| `mehr specification view` | View specifications | | Storage only |
| `mehr budget status` | Show budget status | | Budget methods |
| `mehr budget set` | Set token/cost budget | | Budget methods |
| `mehr budget resume` | Resume after budget pause | | Budget methods |
| `mehr budget reset` | Reset budget counters | | Budget methods |
| `mehr generate-secret` | Generate secure API secret | | Utility |
| `mehr update` | Update mehr to latest version | | Utility |
| `mehr license` | Show license information | | Utility |
| `mehr version` | Show version | | Utility |

---

## 2. Web API Endpoints

Base URL: `http://host:port`. Response format: `{"success": bool, "data": object, "error": string}`. Streaming uses SSE.

### Workflow

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/workflow/start` | Start new task |
| POST | `/api/v1/workflow/plan` | Run planning phase |
| POST | `/api/v1/workflow/implement` | Run implementation phase |
| POST | `/api/v1/workflow/review` | Run review phase |
| POST | `/api/v1/workflow/finish` | Complete task |
| POST | `/api/v1/workflow/continue` | Continue from previous state |
| POST | `/api/v1/workflow/undo` | Undo to previous checkpoint |
| POST | `/api/v1/workflow/redo` | Redo to next checkpoint |
| POST | `/api/v1/workflow/abandon` | Abandon current task |
| POST | `/api/v1/workflow/reset` | Reset workflow state |
| POST | `/api/v1/workflow/auto` | Full automation cycle |
| POST | `/api/v1/workflow/answer` | Answer pending question |
| POST | `/api/v1/workflow/resume` | Resume from waiting/paused |
| POST | `/api/v1/workflow/question` | Submit question to agent |
| POST | `/api/v1/workflow/sync` | Sync task state |
| POST | `/api/v1/workflow/simplify` | Simplify specifications |
| POST | `/api/v1/workflow/review/standalone` | Standalone PR review |
| POST | `/api/v1/workflow/simplify/standalone` | Standalone simplify |
| GET | `/api/v1/workflow/diagram` | Get workflow state diagram |

### Tasks & Status

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/status` | Server and workspace status |
| GET | `/api/v1/context` | Server context (worktree, task) |
| GET | `/api/v1/task` | Get active task |
| GET | `/api/v1/tasks` | List all tasks |
| GET | `/api/v1/tasks/{id}/specs` | Get task specifications |
| GET | `/api/v1/tasks/{id}/sessions` | Get task sessions |
| GET | `/api/v1/tasks/{id}/notes` | Get task notes |
| POST | `/api/v1/tasks/{id}/notes` | Add note to task |
| GET | `/api/v1/tasks/{id}/costs` | Get task costs |
| GET | `/api/v1/task/labels` | Get task labels |
| POST | `/api/v1/task/labels` | Set task labels |
| GET | `/api/v1/labels` | List all labels |
| GET | `/api/v1/task/hierarchy` | Get task hierarchy |
| GET | `/api/v1/guide` | Get workflow guide/next step |

### Browser

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/browser/status` | Browser connection status |
| GET | `/api/v1/browser/tabs` | List open tabs |
| POST | `/api/v1/browser/goto` | Open URL in new tab |
| POST | `/api/v1/browser/navigate` | Navigate current tab |
| POST | `/api/v1/browser/screenshot` | Take screenshot |
| POST | `/api/v1/browser/click` | Click element |
| POST | `/api/v1/browser/type` | Type text |
| POST | `/api/v1/browser/eval` | Evaluate JavaScript |
| POST | `/api/v1/browser/dom` | Query DOM structure |
| POST | `/api/v1/browser/reload` | Reload page |
| POST | `/api/v1/browser/close` | Close browser/tab |
| POST | `/api/v1/browser/network` | Monitor network requests |
| POST | `/api/v1/browser/console` | Monitor console logs |
| POST | `/api/v1/browser/websocket` | Monitor WebSocket frames |
| POST | `/api/v1/browser/source` | Get page HTML source |
| POST | `/api/v1/browser/scripts` | List loaded JavaScript sources |
| POST | `/api/v1/browser/styles` | Inspect CSS styles |
| POST | `/api/v1/browser/coverage` | Measure JS/CSS code coverage |

### Memory & Links

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/memory/search` | Search memory embeddings |
| POST | `/api/v1/memory/index` | Index content for memory |
| GET | `/api/v1/memory/stats` | Get memory statistics |
| GET | `/api/v1/links` | List all links |
| GET | `/api/v1/links/{entity}` | Get entity links |
| GET | `/api/v1/links/search` | Search links |
| GET | `/api/v1/links/stats` | Get links statistics |
| POST | `/api/v1/links/rebuild` | Rebuild link index |

### Library

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/library` | List documentation collections |
| GET | `/api/v1/library/stats` | Get library statistics |
| GET | `/api/v1/library/{name}` | Get collection details |
| POST | `/api/v1/library/pull` | Pull documentation from source |
| POST | `/api/v1/library/pull/preview` | Preview pull operation |
| DELETE | `/api/v1/library/{name}` | Remove a collection |

### Costs & Budget

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/costs` | Get all costs |
| GET | `/api/v1/budget/monthly/status` | Monthly budget status |
| POST | `/api/v1/budget/monthly/reset` | Reset monthly budget |

### Security

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/scan` | Run security/quality scan |

### Project & Stack

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/project/upload` | Upload project data |
| POST | `/api/v1/project/source` | Set project source |
| POST | `/api/v1/project/plan` | Plan project |
| GET | `/api/v1/project/queues` | List project queues |
| GET | `/api/v1/project/queue/{id}` | Get queue details |
| DELETE | `/api/v1/project/queue/{id}` | Delete queue |
| GET | `/api/v1/project/tasks` | List project tasks |
| PUT | `/api/v1/project/tasks/{id}` | Edit project task |
| POST | `/api/v1/project/reorder` | Reorder tasks |
| POST | `/api/v1/project/submit` | Submit project |
| POST | `/api/v1/project/start` | Start project workflow |
| POST | `/api/v1/project/sync` | Sync project |
| GET | `/api/v1/stack` | List stacked features |
| POST | `/api/v1/stack/sync` | Sync stack |
| POST | `/api/v1/stack/rebase` | Rebase stack |

### Quick Tasks

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/quick` | List quick tasks |
| POST | `/api/v1/quick` | Create quick task |
| POST | `/api/v1/quick/submit-source` | Submit quick task source |
| GET | `/api/v1/quick/{taskId}` | Get quick task |
| POST | `/api/v1/quick/{taskId}/note` | Add note to quick task |
| POST | `/api/v1/quick/{taskId}/optimize` | Optimize quick task |
| POST | `/api/v1/quick/{taskId}/export` | Export quick task |
| POST | `/api/v1/quick/{taskId}/submit` | Submit quick task |
| POST | `/api/v1/quick/{taskId}/start` | Start quick task |
| DELETE | `/api/v1/quick/{taskId}` | Delete quick task |

### Interactive & Streaming

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/interactive/chat` | Send chat message |
| POST | `/api/v1/interactive/command` | Execute command |
| POST | `/api/v1/interactive/answer` | Answer question |
| GET | `/api/v1/interactive/state` | Get interactive state |
| GET | `/api/v1/interactive/commands` | List available commands (discovery API) |
| POST | `/api/v1/interactive/stop` | Stop interactive session |
| GET | `/api/v1/events` | SSE event stream |
| GET | `/api/v1/agent/logs/stream` | Stream agent logs (SSE) |
| GET | `/api/v1/agent/logs/history` | Get agent log history |

### Commit & Running Tasks

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/commit/plan` | Plan commit operations |
| POST | `/api/v1/commit/execute` | Execute commit |
| GET | `/api/v1/running` | List running tasks |
| GET | `/api/v1/running/{id}/stream` | Stream running task output (SSE) |
| POST | `/api/v1/running/{id}/cancel` | Cancel running task |
| POST | `/api/v1/parallel` | Start parallel tasks |

### Settings & Configuration

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/settings` | Get settings |
| POST | `/api/v1/settings` | Save settings |
| GET | `/api/v1/settings/explain` | Explain config option |
| GET | `/api/v1/settings/provider-health` | Check provider health |
| GET | `/api/v1/templates` | List templates |
| POST | `/api/v1/templates/apply` | Apply template |

### Agents & Providers

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/agents` | List agents |
| GET | `/api/v1/providers` | List providers |
| GET | `/api/v1/agents/aliases` | List agent aliases |
| POST | `/api/v1/agents/aliases` | Create agent alias |
| DELETE | `/api/v1/agents/aliases/{alias}` | Delete agent alias |

### Health & CSRF

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/auth/csrf` | Get CSRF token |
| GET | `/health` | Health check |
| GET | `/api/v1/license` | License info |

### Global Mode (multi-project)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/projects` | List projects |
| POST | `/api/v1/projects/select` | Select project |
| POST | `/api/v1/projects/switch` | Switch project |

### Sandbox

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/sandbox/status` | Sandbox status |
| POST | `/api/v1/sandbox/enable` | Enable sandbox |
| POST | `/api/v1/sandbox/disable` | Disable sandbox |

---

## 3. MCP Tools

The MCP server (`mehr mcp`) exposes these tools for AI agents over stdio JSON-RPC:

### Workspace Data Tools

| Tool | Description |
|------|-------------|
| `workspace_get_active_task` | Get current active task info |
| `workspace_list_tasks` | List all tasks in workspace |
| `workspace_get_specs` | Get task specifications |
| `workspace_get_sessions` | Get task session history |
| `workspace_get_notes` | Get task notes |

### Registry Tools

| Tool | Description |
|------|-------------|
| `agents_list` | List all registered AI agents |
| `agents_get_default` | Get default agent |
| `providers_list` | List all task providers |
| `providers_resolve` | Resolve provider from task reference |

### CLI Commands as MCP Tools

These CLI commands are registered as callable MCP tools:

| Tool | Description |
|------|-------------|
| `status` | Show workflow status |
| `list` | List tasks |
| `guide` | Show workflow guide |
| `version` | Show version |
| `config` | Manage configuration |
| `providers` | List providers |
| `templates` | List templates |
| `agents` | List agents |
| `cost` | Show costs |
| `scan` | Run security scan |
| `find` | Search codebase |
| `browser_status` | Check browser status |
| `browser_tabs` | List browser tabs |
| `browser_goto` | Navigate to URL |
| `browser_navigate` | Navigate page |
| `browser_close` | Close browser |
| `browser_switch` | Switch tab |
| `browser_reload` | Reload page |
| `browser_screenshot` | Take screenshot |
| `browser_dom` | Query DOM |
| `browser_click` | Click element |
| `browser_type` | Type text |
| `browser_eval` | Evaluate JavaScript |
| `browser_console` | Get console output |
| `browser_network` | Get network requests |
| `browser_source` | Get page HTML source |
| `browser_scripts` | List loaded JavaScript sources |
| `browser_websocket` | Monitor WebSocket frames |
| `browser_styles` | Inspect CSS styles |
| `browser_coverage` | Measure JS/CSS code coverage |
| `browser_cookies` | Cookie management |
| `browser_cookies_export` | Export cookies to file |
| `browser_cookies_import` | Import cookies from file |
| `library_list` | List documentation collections |
| `library_show` | Show collection details and pages |
| `library_get_docs` | Get documentation context for file paths or query |

---

## 4. Key Internal Interfaces

### Conductor (`internal/conductor/`)

The main orchestration facade. All CLI commands and Web UI handlers call Conductor methods.

```
Workflow Methods:
  Start(ctx, ref, opts)        Plan(ctx, opts)          Implement(ctx, opts)
  Review(ctx, opts)            Finish(ctx, opts)        Continue(ctx)
  Undo(ctx)                    Redo(ctx)                Reset(ctx)
  Abandon(ctx)                 Question(ctx, q)         Answer(ctx, a)
  RunAuto(ctx, ref, opts)      RunPRReview(ctx, opts)

Accessors:
  GetProviderRegistry()        GetAgentRegistry()       GetEventBus()
  GetWorkspace()               GetGit()                 GetActiveTask()
  GetTaskWork()                GetActiveAgent()         GetMachine()
  GetBrowser(ctx)              GetStdout()              GetStderr()

Configuration:
  SetAgent(name)               ClearAgent()
  SetImplementationOptions(component, parallel)
  ClearImplementationOptions()

Lifecycle:
  New(opts ...Option)          Initialize(ctx)          Close()
```

### Agent (`internal/agent/`)

```go
type Agent interface {
    Name() string
    Run(ctx context.Context, prompt string) (*Response, error)
    RunStream(ctx context.Context, prompt string) (<-chan Event, <-chan error)
    RunWithCallback(ctx context.Context, prompt string, cb StreamCallback) (*Response, error)
    Available() error
    WithEnv(key, value string) Agent
    WithArgs(args ...string) Agent
}

// Response aggregates agent output
type Response struct {
    Files    []FileChange    // Files created/modified/deleted
    Summary  string          // Agent's summary
    Messages []string        // Agent messages
    Usage    *UsageStats     // Token/cost stats
    Duration time.Duration   // Execution time
    Question *Question       // Pending question (if asked)
}

// Event types for streaming: text, tool_use, tool_result, file, error, usage, complete
```

### Provider (`internal/provider/`)

```go
// Minimum provider
type ReadOnlyProvider interface {
    Fetch(ctx, id string) (*WorkUnit, error)          // Reader
    Parse(input string) (string, error)               // Identifier
    Match(input string) bool                          // Identifier
}

// Extended capabilities (detected via type assertion)
Lister:              List(ctx, opts) ([]*WorkUnit, error)
Commenter:           AddComment(ctx, id, body) (*Comment, error)
StatusUpdater:       UpdateStatus(ctx, id, status) error
LabelManager:        AddLabels/RemoveLabels(ctx, id, labels) error
PRCreator:           CreatePullRequest(ctx, opts) (*PullRequest, error)
PRFetcher:           FetchPullRequest/FetchPullRequestDiff(ctx, number) (*PR, error)
PRCommenter:         AddPullRequestComment(ctx, number, body) (*Comment, error)
PRReviewer:          SubmitReview(ctx, opts) (*ReviewSubmission, error)
SubtaskFetcher:      FetchSubtasks(ctx, id) ([]*WorkUnit, error)
ParentFetcher:       FetchParent(ctx, id) (*WorkUnit, error)
ProjectFetcher:      FetchProject(ctx, ref) (*ProjectStructure, error)
Snapshotter:         Snapshot(ctx, id) (*Snapshot, error)
WorkUnitCreator:     CreateWorkUnit(ctx, opts) (*WorkUnit, error)
BranchLinker:        LinkBranch/UnlinkBranch/GetLinkedBranch(ctx, id, branch) error
DependencyCreator:   CreateDependency(ctx, predecessorID, successorID) error
DependencyFetcher:   GetDependencies(ctx, id) ([]string, error)
```

### Workflow Machine (`internal/workflow/`)

```
States:  idle → planning → implementing → reviewing → done | failed
         + waiting, paused, checkpointing, reverting, restoring

Events:  start, plan, implement, review, finish,
         plan_done, implement_done, review_done,
         checkpoint, checkpoint_done, undo, undo_done, redo, redo_done,
         error, abort, wait, answer, pause, resume, reset

Steps:   planning, implementing, reviewing, checkpointing,
         simplifying, optimizing, quality, pr_review

Machine Methods:
  State()              Dispatch(ctx, event)     CanDispatch(ctx, event)
  WorkUnit()           SetWorkUnit(wu)          AddListener(fn)
  History()            Reset()                  IsTerminal()
  PushUndo(checkpoint) PopUndo()                PopRedo()
  CanUndo()            CanRedo()                PhaseOrder()
```

### Browser Controller (`internal/browser/`)

```go
type Controller interface {
    // Connection
    Connect(ctx) error              Disconnect() error
    IsConnected() bool              GetPort() int

    // Tab management
    ListTabs(ctx) ([]Tab, error)    OpenTab(ctx, url) (*Tab, error)
    CloseTab(ctx, tabID) error      SwitchTab(ctx, tabID) (*Tab, error)
    Navigate(ctx, tabID, url) error Reload(ctx, tabID, hard) error

    // Page interaction
    Screenshot(ctx, tabID, opts) ([]byte, error)
    QuerySelector(ctx, tabID, sel) (*DOMElement, error)
    QuerySelectorAll(ctx, tabID, sel) ([]DOMElement, error)
    Click(ctx, tabID, sel) error    Type(ctx, tabID, sel, text, clear) error
    Eval(ctx, tabID, expr) (any, error)

    // Monitoring
    GetConsoleLogs(ctx, tabID, dur) ([]ConsoleMessage, error)
    GetNetworkRequests(ctx, tabID, dur) ([]NetworkRequest, error)
    SetNetworkMonitorOptions(opts NetworkMonitorOptions)
    GetWebSocketFrames(ctx, tabID, dur) ([]WebSocketFrame, error)

    // Source inspection
    GetPageSource(ctx, tabID) (string, error)
    GetScriptSources(ctx, tabID) ([]ScriptSource, error)

    // CSS inspection
    GetComputedStyles(ctx, tabID, sel) ([]ComputedStyle, error)
    GetMatchedStyles(ctx, tabID, sel) (*MatchedStyles, error)

    // Coverage
    GetCoverage(ctx, tabID, dur, trackJS, trackCSS) (*CoverageSummary, []JSCoverageEntry, []CSSCoverageEntry, error)

    // Authentication
    DetectAuth(ctx, tabID) (*AuthRequirement, error)
    WaitForLogin(ctx, tabID, auth) error

    // Cookies
    GetCookies(ctx) ([]Cookie, error)
    SetCookies(ctx, cookies) error
}
```

### Storage (`internal/storage/`)

```
Split storage:
  .mehrhof/                          → Project config (config.yaml, .env, browser.json)
  ~/.valksor/mehrhof/workspaces/<id>/ → Task data (active_task, work/, sessions/)

Three directory concept:
  Root()     → Project hub (where .mehrhof/ lives, tasks, queues)
  CodeRoot() → Code target (where agents edit, git operates, linters run)
  DataRoot() → Home dir data (~/.valksor/mehrhof/workspaces/<id>/)

CodeRoot() defaults to Root() when project.code_dir is not configured.

Workspace Methods:
  OpenWorkspace(ctx, root, cfg)    Root()           CodeRoot()
  TaskRoot()                       WorkRoot()       DataRoot()
  ConfigPath()                     CodeAbsolutePath(rel)
  HasConfig()                      LoadEnv()        EnsureInitialized()
  SaveFile(path, data)             DeleteFile(path) UpdateGitignore()

Key Types:
  ActiveTask      { ID, Ref, WorkDir, State, Branch, UseGit, WorktreePath, Started }
  TaskWork        { Version, Metadata, Source, Git, Agent, Costs, Budget, Hierarchy }
  Session         { Version, Kind, Metadata, Usage, Exchanges }
  Checkpoint      { ID, Commit, Message, State, CreatedAt }
  ProjectSettings { CodeDir }
  QueuedTask      { ID, ExternalID, Title, Description, SourcePath, Metadata, ... }
```

---

## 5. Data Flow Traces

### CLI Command → Result
```
User runs `mehr plan`
  → cmd/mehr/commands/plan.go (Cobra handler)
  → conductor.Plan(ctx, opts)
  → workflow.Machine.Dispatch(ctx, EventPlan)  [state: idle → planning]
  → agent.RunWithCallback(ctx, prompt, streamCallback)
  → storage.SaveSpecification(spec)
  → workflow.Machine.Dispatch(ctx, EventPlanDone)  [state: planning → idle]
  → events.Bus.Publish(PlanCompletedEvent)
```

### Web UI → SSE Stream
```
Browser POST /api/v1/workflow/plan
  → internal/server/handlers_workflow.go
  → conductor.Plan(ctx, opts)
  → events.Bus publishes progress events
  → internal/server/handlers_stream.go picks up events
  → SSE stream: data: {"type": "progress", "message": "..."}
  → React frontend updates UI via SSE events
```

### MCP Tool Call → Result
```
AI agent calls MCP tool "status"
  → internal/mcp/Server receives JSON-RPC request
  → mcp.ToolRegistry.CallTool(ctx, "status", args)
  → Cobra command statusCmd.RunE(cmd, args)
  → conductor.Status(ctx)
  → JSON-RPC response back to agent
```

### IDE Plugin → API
```
JetBrains/VS Code plugin
  → REST API call (e.g., POST /api/v1/workflow/implement)
  → SSE subscription (GET /api/v1/events)
  → Same handlers as Web UI
  → Same conductor methods
```

---

## 6. Provider & Agent Registry

### Providers (16)

| Provider | Schemes | Description | Key Capabilities |
|----------|---------|-------------|-----------------|
| `github` | `github`, `gh` | GitHub Issues & PRs | Read, List, Comment, Status, Labels, PR, Subtasks, Dependencies |
| `gitlab` | `gitlab`, `gl` | GitLab Issues & MRs | Read, List, Comment, Status, Labels, PR, Subtasks, Dependencies |
| `jira` | `jira`, `j` | Jira Issues | Read, List, Comment, Status, Labels, Subtasks, Parent, Project |
| `linear` | `linear`, `ln` | Linear Issues | Read, List, Comment, Status, Labels, Subtasks, Dependencies |
| `asana` | `asana`, `as` | Asana Tasks | Read, List, Comment, Status, Labels, Subtasks, Dependencies |
| `wrike` | `wrike`, `wk` | Wrike Tasks | Read, List, Comment, Status, Labels, PR, Subtasks, Project |
| `youtrack` | `youtrack`, `yt` | YouTrack Issues | Read, List, Comment, Status, Labels, Subtasks, Dependencies |
| `notion` | `notion`, `nt` | Notion Pages | Read, List, Comment, Status, Labels, Snapshot |
| `bitbucket` | `bitbucket`, `bb` | Bitbucket Issues | Read, List, Comment, Status, PR, Subtasks, Dependencies |
| `azuredevops` | `azuredevops` | Azure DevOps Work Items | Read, List, Comment, Status, Labels, Subtasks |
| `clickup` | `clickup` | ClickUp Tasks | Read, List, Comment, Status, Labels, Subtasks |
| `trello` | `trello` | Trello Cards | Read, List, Comment, Status, Labels |
| `file` | `file` | Local markdown files | Read, Snapshot |
| `directory` | `directory` | Local directory scanner | Read, List, Snapshot |
| `queue` | `queue` | Local project queue | Read, List |
| `empty` | `empty` | Empty task (create from scratch) | Read |

### Agents (3 + plugins)

| Agent | Description | Model |
|-------|-------------|-------|
| `claude` | Claude AI via Claude CLI | claude-sonnet-4 (default), configurable |
| `codex` | OpenAI Codex via Codex CLI | Codex models |
| `noop` | No-op agent for testing | N/A |

Custom agents via `.mehrhof/config.yaml`:
```yaml
agents:
  opus:
    extends: claude
    args: ["--model", "claude-opus-4"]
```

Agent resolution priority (highest → lowest):
1. CLI step flag (`--agent-plan`)
2. CLI global (`--agent`)
3. Task frontmatter step (`agent_steps.planning.agent`)
4. Task frontmatter default (`agent:`)
5. Workspace config step (`agent.steps.planning.name`)
6. Workspace config default (`agent.default`)
7. Auto-detect

---

## 7. Event Types

Events published on the event bus for real-time updates:

| Event | Description |
|-------|-------------|
| `state_changed` | Workflow state transition (from, to, event) |
| `progress` | Progress update (phase, message, current, total) |
| `error` | Error event (error, fatal flag) |
| `file_changed` | File modification (path, operation) |
| `agent_message` | Agent output (content, role) |
| `checkpoint` | Git checkpoint created (commit, message) |
| `blueprint_ready` | Specification ready |
| `branch_created` | Git branch created |
| `plan_completed` | Planning phase done |
| `implement_done` | Implementation done |
| `pr_created` | Pull request created |
| `browser_action` | Browser automation action |
| `browser_tab_opened` | Browser tab opened |
| `browser_screenshot` | Screenshot captured |
| `sandbox_status_changed` | Sandbox status change |
