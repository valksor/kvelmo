# kvelmo Documentation

Welcome to the kvelmo documentation. kvelmo is a structured development workflow that orchestrates AI agents through a reviewable process with human oversight at every step.

## Quick Links

- [Quickstart](/quickstart.md) — Get up and running in 5 minutes
- [Web UI Guide](/web-ui/getting-started.md) — Visual interface for non-technical users
- [CLI Reference](/cli/index.md) — Command-line interface for developers

## What is kvelmo?

kvelmo manages the complete lifecycle of development tasks:

1. **Start** — Load a task from a file, GitHub issue, or other provider
2. **Plan** — Generate a structured specification for review
3. **Implement** — Execute the specification with an AI agent
4. **Review** — Human review with security scanning
5. **Submit** — Create a PR and submit to the provider

Every step creates a git checkpoint. You can undo and redo at any point.

## Interfaces

kvelmo provides two interfaces that share the same underlying engine:

| Interface | Best For |
|-----------|----------|
| [Web UI](/web-ui/getting-started.md) | Non-technical contributors, visual workflows |
| [CLI](/cli/index.md) | Developers, automation, scripting |

## Core Concepts

- [Workflow](/concepts/workflow.md) — The 5-phase workflow philosophy
- [State Machine](/concepts/state-machine.md) — Task states and transitions
- [Checkpoints](/concepts/checkpoints.md) — Git-based undo/redo system
- [Sockets](/concepts/sockets.md) — Architecture overview

## Getting Help

- [FAQ](/faq.md) — Common questions
- [Troubleshooting](/guides/recovery.md) — Recovering from errors
- [GitHub Issues](https://github.com/valksor/kvelmo/issues) — Bug reports and feature requests
