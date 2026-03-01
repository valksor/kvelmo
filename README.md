# kvelmo

[![valksor](https://badgen.net/static/org/valksor/green)](https://github.com/valksor)
[![BSD-3-Clause](https://img.shields.io/badge/BSD--3--Clause-green?style=flat)](https://github.com/valksor/kvelmo/blob/master/LICENSE)
[![GitHub Release](https://img.shields.io/github/release/valksor/kvelmo.svg?style=flat)](https://github.com/valksor/kvelmo/releases/latest)
[![GitHub last commit](https://img.shields.io/github/last-commit/valksor/kvelmo.svg?style=flat)](https://github.com/valksor/kvelmo/commits/master)

[![Go Report Card](https://goreportcard.com/badge/github.com/valksor/kvelmo)](https://goreportcard.com/report/github.com/valksor/kvelmo)
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/valksor/kvelmo)

kvelmo helps you turn ideas into reliable changes through a structured workflow of **plan → implement → review → submit**.

Whether you're a product manager describing a feature, an ops person fixing a process, or a developer automating routine work — kvelmo provides the guardrails to make changes safely. It coordinates your local AI agent (Claude, Codex, or custom) to handle the mechanics while you stay in control of review and approval.

**Free and open-source.** If you already have Claude or another agent CLI working, kvelmo adds zero extra cost — just install and use.

---

## Choose Your Path

### For Non-Technical Contributors

Start with the **Desktop App** or **Web UI** — comfortable visual experiences that don't require command-line expertise.

**Desktop App** — Download from [Releases](https://github.com/valksor/kvelmo/releases) for a fully native experience.

**Web UI** — Run one command to start:

```bash
kvelmo serve --open
```

Open http://localhost:6337 and use the dashboard to create tasks, monitor progress, and manage your workflow.

**[Web UI Guide](https://valksor.com/docs/kvelmo/nightly/#/web-ui/getting-started)** — Visual walkthrough of your first task

### For Developers and Power Users

Use the **CLI** for scriptable workflows, automation, and integration with other tools.

```bash
kvelmo start --from file:task.md
kvelmo plan
kvelmo implement
kvelmo review
kvelmo submit
```

**[CLI Reference](https://valksor.com/docs/kvelmo/nightly/#/cli/index)** — All commands and options

All three interfaces — CLI, Web UI, and Desktop App — share the same workflow. Your work is portable between them.

---

## Key Features

| Feature | What It Does |
|---------|--------------|
| **Structured workflow** | Breaks work into plan → implement → review → submit steps |
| **Human review gates** | Nothing ships without your explicit approval |
| **Undo/redo support** | Roll back to any checkpoint if something goes wrong |
| **Multiple providers** | Pull tasks from files, GitHub, GitLab, Linear, or Wrike |
| **Git integration** | Automatic branches, commits, and PR creation |
| **Multi-interface** | Same workflow in CLI, Web UI, and Desktop App |

---

## Working Within Company Guidelines

kvelmo is designed for settings where changes need oversight:

- **Review requirements** — Configure approval workflows that match your policies
- **Audit trails** — Full history of who planned what and when
- **Tool integration** — Works with your existing Git, issue trackers, and CI/CD

**[Configuration Guide](https://valksor.com/docs/kvelmo/nightly/#/configuration/index)** — Customize for your team

---

## How It Works

```
┌─────────┐   ┌─────────┐   ┌─────────────┐   ┌──────────┐   ┌──────────┐
│  START  │ → │  PLAN   │ → │  IMPLEMENT  │ → │  REVIEW  │ → │  SUBMIT  │
│  task   │   │  spec   │   │   changes   │   │  approve │   │    PR    │
└─────────┘   └─────────┘   └─────────────┘   └──────────┘   └──────────┘
                            ↑               ↓
                            └─── undo/redo ─┘
```

1. **Start** — Load a task from a file, GitHub issue, or other source
2. **Plan** — Review the generated specification before any changes happen
3. **Implement** — Watch as your changes are built; undo anytime to try a different approach
4. **Review** — Approve the output before it ships
5. **Submit** — Create a PR and submit to your provider

Every step creates a checkpoint. Use `kvelmo undo` to revert; `kvelmo redo` to restore.

---

## Installation

```bash
curl -fsSL https://raw.githubusercontent.com/valksor/kvelmo/master/install.sh | bash
```

**Prerequisite**: You need an agent CLI installed (e.g., [Claude](https://claude.ai/code)). kvelmo uses your CLI subscription directly — no API keys to configure. If `claude --version` works, you're ready.

**[Full Installation Guide](INSTALL.md)** — Platform-specific instructions, WSL setup, and troubleshooting

---

## Documentation

| Guide | Description |
|-------|-------------|
| [Quickstart](https://valksor.com/docs/kvelmo/nightly/#/quickstart) | Install and run your first task |
| [Web UI Guide](https://valksor.com/docs/kvelmo/nightly/#/web-ui/getting-started) | Visual walkthrough |
| [Desktop App](https://valksor.com/docs/kvelmo/nightly/#/desktop/index) | Native application guide |
| [CLI Reference](https://valksor.com/docs/kvelmo/nightly/#/cli/index) | All commands and options |
| [Workflow Concepts](https://valksor.com/docs/kvelmo/nightly/#/concepts/workflow) | Understanding the process |
| [Providers](https://valksor.com/docs/kvelmo/nightly/#/providers/index) | GitHub, GitLab, Linear, Wrike |
| [Configuration](https://valksor.com/docs/kvelmo/nightly/#/configuration/index) | Customize behavior |

---

## CLI Commands

```bash
# Workflow
kvelmo start --from <provider>:<ref>   # Load task from provider
kvelmo plan                            # Generate implementation spec
kvelmo implement                       # Build changes from spec
kvelmo review                          # Review before shipping
kvelmo submit                          # Create PR and submit
kvelmo refresh                         # Check PR status
kvelmo finish                          # Clean up after merge

# Navigation
kvelmo undo                            # Revert to previous checkpoint
kvelmo redo                            # Restore undone checkpoint
kvelmo status                          # Show current state

# Management
kvelmo serve                           # Start web server
kvelmo config                          # Configuration management
kvelmo workers                         # Worker pool status
```

**[Full CLI Reference](https://valksor.com/docs/kvelmo/nightly/#/cli/index)** — All commands, flags, and options

---

## Development

```bash
make build        # Full build: web + Go binary
make test         # Run all tests
make lint         # golangci-lint with --fix
make quality      # fmt + vet + lint
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for complete guidelines.

---

## Prototype

The `prototype/` directory contains the original working prototype that informed kvelmo's design. This code is preserved for historical reference only and is not maintained. See [flow_v2.md](flow_v2.md) for how the prototype's patterns were adapted into the current architecture.

---

## License

[BSD 3-Clause License](LICENSE)
