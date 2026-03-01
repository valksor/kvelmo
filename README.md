# kvelmo

[![valksor](https://badgen.net/static/org/valksor/green)](https://github.com/valksor)
[![BSD-3-Clause](https://img.shields.io/badge/BSD--3--Clause-green?style=flat)](https://github.com/valksor/kvelmo/blob/master/LICENSE)
[![GitHub Release](https://img.shields.io/github/release/valksor/kvelmo.svg?style=flat)](https://github.com/valksor/kvelmo/releases/latest)
[![GitHub last commit](https://img.shields.io/github/last-commit/valksor/kvelmo.svg?style=flat)](https://github.com/valksor/kvelmo/commits/master)

[![Go Report Card](https://goreportcard.com/badge/github.com/valksor/kvelmo)](https://goreportcard.com/report/github.com/valksor/kvelmo)
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/valksor/kvelmo)

---

**kvelmo turns your ideas into merged pull requests — with human oversight at every step.**

Whether you're a product manager describing a feature, an ops person fixing a process, or a developer automating routine work — kvelmo provides the guardrails to ship changes safely. It coordinates your local AI agent to handle the mechanics while you stay in control of review and approval.

**Completely free.** kvelmo uses your existing Claude CLI directly — no API keys to configure, no usage fees, no account needed. If `claude --version` works, you're ready. Codex and other agents are also supported.

---

We built kvelmo with a simple belief: **the best tools make hard things approachable without hiding what's happening.**

AI can handle the mechanics of planning and implementation. But understanding, judgment, and approval? Those stay with you. kvelmo brings structure to that collaboration — clear steps, visible progress, and the ability to undo anything that doesn't look right.

No magic. No "just press go." A structured way to work with tools you already have.

---

## How It Works

```
          ┌────────────────────────────────────────────────────────┐
          │                    YOU'RE IN CONTROL                    │
          │   undo ←─────────────────────────────────────→ redo     │
          └────────────────────────────────────────────────────────┘
                              ↓ checkpoints ↓
┌─────────────┐   ┌─────────────┐   ┌─────────────┐   ┌─────────────┐
│    START    │ → │    PLAN     │ → │  IMPLEMENT  │ → │   REVIEW    │ → SUBMIT
│  describe   │   │   approve   │   │   watch it  │   │   approve   │
│  your idea  │   │  the spec   │   │   happen    │   │  the result │
└─────────────┘   └─────────────┘   └─────────────┘   └─────────────┘
```

1. **Start** — Describe what you want in plain language, or pull a task from GitHub, GitLab, Linear, or Wrike
2. **Plan** — Review the generated specification before any changes happen. Add notes to refine direction. Nothing proceeds without your approval
3. **Implement** — Watch as your agent builds the changes. Every step creates a checkpoint — undo anytime to try a different approach
4. **Review** — Approve the output before it ships. Make adjustments if needed
5. **Submit** — Create a PR and push to your repository

Close your laptop, come back tomorrow. `kvelmo status` shows where you left off.

---

## Choose Your Path

### For Non-Technical Contributors

Start with the **Desktop App** or **Web UI** — comfortable visual experiences that don't require command-line expertise.

**Desktop App** — Download from [Releases](https://github.com/valksor/kvelmo/releases) for a fully native experience. No terminal needed after installation.

**Web UI** — Run one command to start:

```bash
kvelmo serve --open
```

Open http://localhost:6337 and use the dashboard to create tasks, watch progress, and manage your workflow visually.

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

All three interfaces share the same workflow. Start in the Web UI, finish in CLI — your work is portable.

---

## What kvelmo Is

- **A guided workflow** that breaks work into reviewable steps: plan → implement → review → submit
- **Human-gated** — nothing ships without your explicit approval at each phase
- **Recoverable** — every step creates a git checkpoint; undo mistakes, try different approaches
- **Multi-source** — pull tasks from files, GitHub, GitLab, Linear, or Wrike
- **Multi-interface** — same workflow in CLI, Web UI, and Desktop App

## What kvelmo Is Not

- **Not autonomous** — it doesn't make decisions for you or ship code without approval
- **Not a hosted service** — kvelmo runs locally and uses your existing agent CLI
- **Not a replacement for understanding** — you should review and understand what changes are being made

kvelmo is an orchestrator, not an AI. It coordinates tools you already have through a structured, reviewable process. The limitations are features — they keep you in control.

---

## Installation

```bash
curl -fsSL https://raw.githubusercontent.com/valksor/kvelmo/master/install.sh | bash
```

**That's it.** If your Claude CLI works (`claude --version`), kvelmo works. No API keys, no configuration, no account setup.

**Other options:**

```bash
# Install specific version
curl -fsSL https://raw.githubusercontent.com/valksor/kvelmo/master/install.sh | bash -s -- -v v1.2.3

# Install nightly build
curl -fsSL https://raw.githubusercontent.com/valksor/kvelmo/master/install.sh | bash -s -- --nightly
```

**[Full Installation Guide](INSTALL.md)** — Platform-specific instructions, WSL setup, and troubleshooting

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
| **Real-time progress** | Watch your agent work in the Web UI or CLI |

---

## CLI Commands

```bash
# Workflow
kvelmo start --from <provider>:<ref>   # Load task from provider
kvelmo plan                            # Generate implementation spec
kvelmo implement                       # Build changes from spec
kvelmo review                          # Review before shipping
kvelmo submit                          # Create PR and submit
kvelmo finish                          # Clean up after merge

# Navigation
kvelmo undo                            # Revert to previous checkpoint
kvelmo redo                            # Restore undone checkpoint
kvelmo status                          # Show current state

# Management
kvelmo serve                           # Start web server
kvelmo config                          # Configuration management
```

**[Full CLI Reference](https://valksor.com/docs/kvelmo/nightly/#/cli/index)** — All commands, flags, and options

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

## Development

```bash
make build        # Full build: web + Go binary
make test         # Run all tests
make quality      # fmt + vet + lint
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for complete guidelines.

---

## Prototype

The `prototype/` directory contains the original working prototype that informed kvelmo's design. This code is preserved for historical reference only and is not maintained. See [flow_v2.md](flow_v2.md) for how the prototype's patterns were adapted into the current architecture.

---

## License

[BSD 3-Clause License](LICENSE)
