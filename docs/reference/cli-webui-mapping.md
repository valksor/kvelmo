# CLI to Web UI Mapping

Quick reference for finding Web UI equivalents of CLI commands.

## Mapping Overview

The CLI uses a command-per-file structure (54 docs), while the Web UI uses a workflow-oriented structure (35 docs). This mapping shows where each CLI command's functionality is documented in the Web UI.

> **Note:** Some CLI commands are CLI-only (no Web UI equivalent). These are marked with **CLI Only** in the notes column.

## Complete Mapping Table

### Task Management

| CLI Command                  | Web UI Doc                                    | UI Location                    | Notes                       |
|------------------------------|-----------------------------------------------|--------------------------------|-----------------------------|
| [start](/cli/start.md)       | [Creating Tasks](/web-ui/creating-tasks.md)   | Dashboard → Create Task button | Task creation workflows     |
| [status](/cli/status.md)     | [Dashboard](/web-ui/dashboard.md)             | Dashboard → Active Task card   | Always visible on dashboard |
| [continue](/cli/continue.md) | [Continuing](/web-ui/continuing.md)           | Dashboard → Continue button    | Context-aware next step     |
| [sync](/cli/sync.md)         | [Syncing](/web-ui/sync.md)                    | Task Detail → Actions → Sync   | Refresh from provider       |
| [abandon](/cli/abandon.md)   | [Getting Started](/web-ui/getting-started.md) | Dashboard → Abandon button     | In Quick Actions section    |
| [list](/cli/list.md)         | [Task History](/web-ui/task-history.md)       | Dashboard → Task History       | Search and filter tasks     |
| [delete](/cli/delete.md)     | [Task History](/web-ui/task-history.md)       | Task History → Delete          | Remove completed tasks      |

### Quick Tasks & Optimization

| CLI Command                  | Web UI Doc                            | UI Location                   | Notes                    |
|------------------------------|---------------------------------------|-------------------------------|--------------------------|
| [quick](/cli/quick.md)       | [Quick Tasks](/web-ui/quick-tasks.md) | Dashboard → Quick Tasks       | Lightweight task capture |
| [optimize](/cli/optimize.md) | [Quick Tasks](/web-ui/quick-tasks.md) | Quick Tasks → Optimize button | AI-refine task details   |
| [export](/cli/export.md)     | [Quick Tasks](/web-ui/quick-tasks.md) | Quick Tasks → Export button   | Save to markdown         |
| [submit](/cli/submit.md)     | [Quick Tasks](/web-ui/quick-tasks.md) | Quick Tasks → Submit button   | Send to provider         |

### Workflow Steps

| CLI Command                    | Web UI Doc                              | UI Location                    | Notes                    |
|--------------------------------|-----------------------------------------|--------------------------------|--------------------------|
| [plan](/cli/plan.md)           | [Planning](/web-ui/planning.md)         | Active Task → Plan button      | Generate specifications  |
| [implement](/cli/implement.md) | [Implementing](/web-ui/implementing.md) | Active Task → Implement button | Execute specifications   |
| [review](/cli/review.md)       | [Reviewing](/web-ui/reviewing.md)       | Active Task → Review button    | Quality checks           |
| [finish](/cli/finish.md)       | [Finishing](/web-ui/finishing.md)       | Active Task → Finish button    | Merge and complete       |
| [note](/cli/note.md)           | [Notes](/web-ui/notes.md)               | Active Task → Add Note button  | Context for agent        |
| [question](/cli/question.md)   | [Questions](/web-ui/questions.md)       | Waiting state panel            | Answer agent questions   |
| [auto](/cli/auto.md)           | [Auto Mode](/web-ui/auto.md)            | Active Task → Auto button      | Full autonomous workflow |
| [guide](/cli/guide.md)         | [Dashboard](/web-ui/dashboard.md)       | Active Task → state hints      | Suggested next actions   |

### History & Checkpoints

| CLI Command            | Web UI Doc                          | UI Location              | Notes                    |
|------------------------|-------------------------------------|--------------------------|--------------------------|
| [undo](/cli/undo.md)   | [Undo & Redo](/web-ui/undo-redo.md) | Dashboard → Undo button  | Revert to checkpoint     |
| [redo](/cli/redo.md)   | [Undo & Redo](/web-ui/undo-redo.md) | Dashboard → Redo button  | Restore from redo stack  |
| [reset](/cli/reset.md) | [Reset State](/web-ui/reset.md)     | Dashboard → Reset button | Recover from stuck state |

### Project Planning

| CLI Command                          | Web UI Doc                                      | UI Location                    | Notes                  |
|--------------------------------------|-------------------------------------------------|--------------------------------|------------------------|
| [project plan](/cli/project.md)      | [Project Planning](/web-ui/project-planning.md) | Dashboard → Project Planning   | Break down large tasks |
| [project sync](/cli/project-sync.md) | [Project Planning](/web-ui/project-planning.md) | Project → Sync from Provider   | Pull external project  |
| [project tasks](/cli/project.md)     | [Project Planning](/web-ui/project-planning.md) | Dashboard → Tasks → Queue view | View queue tasks       |
| [project edit](/cli/project.md)      | [Project Planning](/web-ui/project-planning.md) | Queue view → Edit task         | Modify queue tasks     |
| [project reorder](/cli/project.md)   | [Project Planning](/web-ui/project-planning.md) | Queue view → Reorder           | AI-assisted reordering |
| [project submit](/cli/project.md)    | [Project Planning](/web-ui/project-planning.md) | Queue view → Submit            | Send to provider       |
| [project start](/cli/project.md)     | [Project Planning](/web-ui/project-planning.md) | Queue view → Start             | Begin implementation   |

### Stacked Features

| CLI Command                      | Web UI Doc                | UI Location              | Notes                        |
|----------------------------------|---------------------------|--------------------------|------------------------------|
| [stack](/cli/stack.md)           | [Stack](/web-ui/stack.md) | Tools → Stack tab        | Manage dependent branches    |
| [stack --graph](/cli/stack.md)   | [Stack](/web-ui/stack.md) | Stack page visualization | **CLI Only**: ASCII graph    |
| [stack --mermaid](/cli/stack.md) | —                         | —                        | **CLI Only**: Mermaid output |

### Web UI & Server

| CLI Command                        | Web UI Doc                                    | UI Location              | Notes                           |
|------------------------------------|-----------------------------------------------|--------------------------|---------------------------------|
| [serve](/cli/serve.md)             | [Getting Started](/web-ui/getting-started.md) | —                        | **CLI Only**: Starts server     |
| [serve --register](/cli/serve.md)  | [Settings](/web-ui/settings.md)               | —                        | **CLI Only**: Register instance |
| [register](/cli/register.md)       | [Settings](/web-ui/settings.md)               | —                        | **CLI Only**: Legacy alias      |
| [interactive](/cli/interactive.md) | [Chat](/web-ui/interactive.md)                | Workflow dropdown → Chat | REPL mode / Chat page           |

### Configuration & Utilities

| CLI Command                    | Web UI Doc                        | UI Location                     | Notes                              |
|--------------------------------|-----------------------------------|---------------------------------|------------------------------------|
| [init](/cli/init.md)           | —                                 | —                               | **CLI Only**: Initialize workspace |
| [config](/cli/config.md)       | [Settings](/web-ui/settings.md)   | Navigation → Settings           | Configuration UI                   |
| [agents](/cli/agents.md)       | [Settings](/web-ui/settings.md)   | Settings → Agent section        | Agent management                   |
| [providers](/cli/providers.md) | [Settings](/web-ui/settings.md)   | Settings → Providers section    | Provider config                    |
| [templates](/cli/templates.md) | [Templates](/web-ui/templates.md) | Create Task → Template dropdown | Task templates                     |
| [cost](/cli/cost.md)           | [Dashboard](/web-ui/dashboard.md) | Active Task → cost display      | Token usage                        |
| [budget](/cli/budget.md)       | [Settings](/web-ui/settings.md)   | Settings → Budget section       | Spending limits                    |
| [license](/cli/license.md)     | [Settings](/web-ui/settings.md)   | Settings → About                | License info                       |
| [version](/cli/version.md)     | —                                 | —                               | **CLI Only**: Version output       |

### Code Search & Simplification

| CLI Command                  | Web UI Doc                                        | UI Location           | Notes                  |
|------------------------------|---------------------------------------------------|-----------------------|------------------------|
| [find](/cli/find.md)         | [Find](/web-ui/find.md)                           | Navigation bar → Find | AI-powered code search |
| [simplify](/cli/simplify.md) | [Security & Quality](/web-ui/security-quality.md) | Dashboard → Simplify  | Code refactoring       |

### Specifications & Labels

| CLI Command                                   | Web UI Doc                              | UI Location                  | Notes                |
|-----------------------------------------------|-----------------------------------------|------------------------------|----------------------|
| [specification](/cli/specification.md)        | [Implementing](/web-ui/implementing.md) | Task Detail → Specifications | View/manage specs    |
| [specification view](/cli/specification.md)   | [Planning](/web-ui/planning.md)         | Specifications panel         | View spec content    |
| [specification add](/cli/specification.md)    | [Planning](/web-ui/planning.md)         | Add Specification button     | Manual spec creation |
| [specification delete](/cli/specification.md) | [Planning](/web-ui/planning.md)         | Spec → Delete button         | Remove specification |
| [label](/cli/label.md)                        | [Dashboard](/web-ui/dashboard.md)       | Active Task → Labels         | Task labeling        |
| [label add](/cli/label.md)                    | [Dashboard](/web-ui/dashboard.md)       | Labels → Add                 | Add label            |
| [label remove](/cli/label.md)                 | [Dashboard](/web-ui/dashboard.md)       | Labels → Remove              | Remove label         |
| [label set](/cli/label.md)                    | [Dashboard](/web-ui/dashboard.md)       | Labels → Set                 | Replace all labels   |
| [label list](/cli/label.md)                   | [Dashboard](/web-ui/dashboard.md)       | Labels display               | View labels          |

### Standalone Reviews

| CLI Command                           | Web UI Doc                        | UI Location               | Notes                |
|---------------------------------------|-----------------------------------|---------------------------|----------------------|
| [review view](/cli/review.md)         | [Reviewing](/web-ui/reviewing.md) | Reviews panel → View      | View previous review |
| [implement review](/cli/implement.md) | [Reviewing](/web-ui/reviewing.md) | Reviews → Implement Fixes | Fix review issues    |

### Tools & Features

| CLI Command                | Web UI Doc                                        | UI Location               | Notes                     |
|----------------------------|---------------------------------------------------|---------------------------|---------------------------|
| [browser](/cli/browser.md) | [Browser Control](/web-ui/browser.md)             | Tools → Browser tab       | Chrome automation         |
| [memory](/cli/memory.md)   | [Memory](/web-ui/memory.md)                       | Tools → Memory tab        | Semantic search           |
| [library](/cli/library.md) | [Library](/web-ui/library.md)                     | More dropdown → Library   | Documentation collections |
| [links](/cli/links.md)     | [Links](/web-ui/links.md)                         | More dropdown → Links     | Bidirectional linking     |
| [scan](/cli/scan.md)       | [Security & Quality](/web-ui/security-quality.md) | Tools → Security tab      | Security scanning         |
| [plugins](/cli/plugins.md) | [Settings](/web-ui/settings.md)                   | Settings → Plugins        | Plugin management         |
| [commit](/cli/commit.md)   | [Commit](/web-ui/commit.md)                       | Dashboard → Commit button | AI-generated commits      |

### Provider Authentication

| CLI Command                     | Web UI Doc                      | UI Location                     | Notes               |
|---------------------------------|---------------------------------|---------------------------------|---------------------|
| [github login](/cli/login.md)   | [Settings](/web-ui/settings.md) | Settings → Providers → GitHub   | Token configuration |
| [gitlab login](/cli/login.md)   | [Settings](/web-ui/settings.md) | Settings → Providers → GitLab   | Token configuration |
| [notion login](/cli/login.md)   | [Settings](/web-ui/settings.md) | Settings → Providers → Notion   | Token configuration |
| [jira login](/cli/login.md)     | [Settings](/web-ui/settings.md) | Settings → Providers → Jira     | Token configuration |
| [linear login](/cli/login.md)   | [Settings](/web-ui/settings.md) | Settings → Providers → Linear   | Token configuration |
| [wrike login](/cli/login.md)    | [Settings](/web-ui/settings.md) | Settings → Providers → Wrike    | Token configuration |
| [youtrack login](/cli/login.md) | [Settings](/web-ui/settings.md) | Settings → Providers → YouTrack | Token configuration |

### CLI-Only Commands

These commands have no Web UI equivalent by design:

| CLI Command                                | Purpose                  | Why CLI Only           |
|--------------------------------------------|--------------------------|------------------------|
| [mcp](/cli/mcp.md)                         | MCP server for AI agents | stdio-based protocol   |
| [generate-secret](/cli/generate-secret.md) | Generate shared secret   | One-time utility       |
| [update](/cli/update.md)                   | Self-update CLI          | Binary management      |
| [workflow](/cli/workflow.md)               | Display state diagram    | Terminal visualization |

## Finding Coverage

### By Web UI Page

| Web UI Doc                                        | CLI Commands Covered                              |
|---------------------------------------------------|---------------------------------------------------|
| [Dashboard](/web-ui/dashboard.md)                 | status, continue, guide, cost, label, undo, redo  |
| [Creating Tasks](/web-ui/creating-tasks.md)       | start                                             |
| [Planning](/web-ui/planning.md)                   | plan, specification                               |
| [Implementing](/web-ui/implementing.md)           | implement, specification                          |
| [Reviewing](/web-ui/reviewing.md)                 | review, implement review                          |
| [Finishing](/web-ui/finishing.md)                 | finish                                            |
| [Notes](/web-ui/notes.md)                         | note                                              |
| [Chat](/web-ui/interactive.md)                    | interactive                                       |
| [Settings](/web-ui/settings.md)                   | config, agents, providers, budget, plugins, login |
| [Quick Tasks](/web-ui/quick-tasks.md)             | quick, optimize, export, submit                   |
| [Project Planning](/web-ui/project-planning.md)   | project (all subcommands)                         |
| [Stack](/web-ui/stack.md)                         | stack                                             |
| [Find](/web-ui/find.md)                           | find                                              |
| [Memory](/web-ui/memory.md)                       | memory                                            |
| [Library](/web-ui/library.md)                     | library                                           |
| [Links](/web-ui/links.md)                         | links                                             |
| [Browser Control](/web-ui/browser.md)             | browser                                           |
| [Security & Quality](/web-ui/security-quality.md) | scan, simplify                                    |
| [Templates](/web-ui/templates.md)                 | templates                                         |

## See Also

- [CLI Overview](/cli/index.md) - Full CLI command reference
- [Web UI Overview](/web-ui/index.md) - Web interface guide
- [Feature Parity](/reference/feature-parity.md) - Implementation status
