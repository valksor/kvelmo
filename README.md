# Mehrhof — Structured Creation Environment

[![valksor](https://badgen.net/static/org/valksor/green)](https://github.com/valksor)
[![BSD-3-Clause](https://img.shields.io/badge/BSD--3--Clause-green?style=flat)](https://github.com/valksor/go-mehrhof/blob/master/LICENSE)
[![GitHub Release](https://img.shields.io/github/release/valksor/go-mehrhof.svg?style=flat)](https://github.com/valksor/go-mehrhof/releases/latest)
[![GitHub last commit](https://img.shields.io/github/last-commit/valksor/go-mehrhof.svg?style=flat)](https://github.com/valksor/go-mehrhof/commits/master)
[![zread](https://img.shields.io/badge/Ask_Zread-_.svg?style=flat&color=00b0aa&labelColor=000000&logo=data%3Aimage%2Fsvg%2Bxml%3Bbase64%2CPHN2ZyB3aWR0aD0iMTYiIGhlaWdodD0iMTYiIHZpZXdCb3g9IjAgMCAxNiAxNiIgZmlsbD0ibm9uZSIgeG1sbnM9Imh0dHA6Ly93d3cudzMub3JnLzIwMDAvc3ZnIj4KPHBhdGggZD0iTTQuOTYxNTYgMS42MDAxSDIuMjQxNTZDMS44ODgxIDEuNjAwMSAxLjYwMTU2IDEuODg2NjQgMS42MDE1NiAyLjI0MDFWNC45NjAxQzEuNjAxNTYgNS4zMTM1NiAxLjg4ODEgNS42MDAxIDIuMjQxNTYgNS42MDAxSDQuOTYxNTZDNS4zMTUwMiA1LjYwMDEgNS42MDE1NiA1LjMxMzU2IDUuNjAxNTYgNC45NjAxVjIuMjQwMUM1LjYwMTU2IDEuODg2NjQgNS4zMTUwMiAxLjYwMDEgNC45NjE1NiAxLjYwMDFaIiBmaWxsPSIjZmZmIi8%2BCjxwYXRoIGQ9Ik00Ljk2MTU2IDEwLjM5OTlIMi4yNDE1NkMxLjg4ODEgMTAuMzk5OSAxLjYwMTU2IDEwLjY4NjQgMS42MDE1NiAxMS4wMzk5VjEzLjc1OTlDMS42MDE1NiAxNC4xMTM0IDEuODg4MSAxNC4zOTk5IDIuMjQxNTYgMTQuMzk5OUg0Ljk2MTU2QzUuMzE1MDIgMTQuMzk5OSA1LjYwMTU2IDE0LjExMzQgNS42MDE1NiAxMy43NTk5VjExLjAzOTlDNS42MDE1NiAxMC42ODY0IDUuMzE1MDIgMTAuMzk5OSA0Ljk2MTU2IDEwLjM5OTlaIiBmaWxsPSIjZmZmIi8%2BCjxwYXRoIGQ9Ik0xMy43NTg0IDEuNjAwMUgxMS4wMzg0QzEwLjY4NSAxLjYwMDEgMTAuMzk4NCAxLjg4NjY0IDEwLjM5ODQgMi4yNDAxVjQuOTYwMUMxMC4zOTg0IDUuMzEzNTYgMTAuNjg1IDUuNjAwMSAxMS4wMzg0IDUuNjAwMUgxMy43NTg0QzE0LjExMTkgNS42MDAxIDE0LjM5ODQgNS4zMTM1NiAxNC4zOTg0IDQuOTYwMVYyLjI0MDFDMTQuMzk4NCAxLjg4NjY0IDE0LjExMTkgMS42MDAxIDEzLjc1ODQgMS42MDAxWiIgZmlsbD0iI2ZmZiIvPgo8cGF0aCBkPSJNNCAxMkwxMiA0TDQgMTJaIiBmaWxsPSIjZmZmIi8%2BCjxwYXRoIGQ9Ik00IDEyTDEyIDQiIHN0cm9rZT0iI2ZmZiIgc3Ryb2tlLXdpZHRoPSIxLjUiIHN0cm9rZS1saW5lY2FwPSJyb3VuZCIvPgo8L3N2Zz4K&logoColor=ffffff)](https://zread.ai/valksor/go-mehrhof)

[![Coverage Status](https://coveralls.io/repos/github/valksor/go-mehrhof/badge.svg?branch=master)](https://coveralls.io/github/valksor/go-mehrhof?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/valksor/go-mehrhof)](https://goreportcard.com/report/github.com/valksor/go-mehrhof)
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/valksor/go-mehrhof)

---

**Mehrhof** is a free, open-source structured creation environment designed for people who know what they want to build—whether that's feature changes, documentation updates, or well-defined modifications—without requiring deep technical skills.

Rather than being an "AI automation tool," Mehrhof brings clarity, guardrails, and repeatable workflows to the process of transforming human intent into reliable, reviewable output. It orchestrates your local agent CLI (Claude, Codex, or custom) to handle the mechanics of planning and transformation, while you stay in control of meaning, quality, and approval.

We build Mehrhof with the Valksor philosophy: **clarity before speed, quality before spectacle, long-term thinking over quick wins**. Technology should be understandable, dependable, and trustworthy—not coated in hype.

---

## What Mehrhof Is

Mehrhof is:

- **A workflow framework** that turns requests into structured, reviewable outputs
- **A structured process** that breaks creation into clear steps: plan → create → review → finish
- **A wrapper for agent CLIs**—your already-installed local tools (Claude, Codex, or custom); no extra subscriptions required
- **Completely free**—BSD-3 licensed, no usage fees, no account needed; if you already pay for an agent CLI, Mehrhof adds zero cost
- **A comfortable interface** for non-technical contributors via Web UI, with CLI available for power users

Mehrhof is not magic. It's a structured way to work with tools you already have, with clear checkpoints and human oversight at every step.

---

## How It Works

```
┌─────────────────────────────────────────────────────────────────────┐
│                                                                     │
│   1. TASK          2. PLAN          3. CREATE         4. REVIEW     │
│   ─────────        ────────         ────────          ────────      │
│   Describe what    Break it into    Execute via       Human         │
│   you want         actionable       agent CLI         approval      │
│                    steps                              before ship   │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

**1. Define a Task**
Start with a plain-language description of what you want to create or change. No code required — write what you need in your own words.

**2. Generate a Plan**
Mehrhof helps break your intent into actionable steps. You review this plan before any changes are made. Nothing happens without your approval.

**3. Create via Agent CLI**
Mehrhof structures calls to your configured agent (Claude by default, or Codex/custom). The agent handles the text transformations—creating code, documentation, or configuration files based on your approved plan.

**4. Review and Adjust**
All output is presented for human review before acceptance. You can undo, adjust, add notes, and iterate until you're satisfied. Then finish to merge your changes.

No hidden AI access, no servers, no extra fees—just structured assistance using tools you already have.

---

## Safety and Responsible Use

Mehrhof is built on principles that keep you in control:

- **Human review at every step** — Output is always reviewable and adjustable before it becomes permanent
- **No autonomous execution** — Nothing ships without your explicit approval
- **Transparent process** — You can see what's happening at each stage; no black boxes
- **Configurable guardrails** — Match your company's policies, guidelines, and approval workflows
- **Undo/redo support** — Made a mistake? Roll back to any checkpoint
- **Security by default** — CSRF protection, per-IP rate limiting, and session-based authentication for network-accessible deployments

We believe AI is a useful support tool for routine work and exploration, but critical decisions require human understanding and oversight. Mehrhof is designed to help, not replace, your judgment.

---

## Web UI vs CLI

Choose the interface that fits your workflow:

| Web UI                                      | CLI                                      |
|---------------------------------------------|------------------------------------------|
| Comfortable browser-based experience        | Text-based workflow for power users      |
| Click-through workflow with visual feedback | Scriptable commands for automation       |
| Ideal for non-technical contributors        | Ideal for developers and CI/CD pipelines |
| Real-time progress and streaming output     | Composable with other tools              |

Both interfaces use the same underlying engine and configuration. Your work is portable between them.

### Getting Started with Web UI

```bash
# 1. Install Mehrhof
curl -fsSL https://raw.githubusercontent.com/valksor/go-mehrhof/master/install.sh | bash

# 2. Navigate to your project
cd /path/to/your/project

# 3. Initialize and start
mehr init           # One-time setup
mehr serve --open   # Opens browser automatically
```

Click **"Create Task"** in your browser and you're ready to go.

**Web UI Features:**

| Feature                 | Description                                             |
|-------------------------|---------------------------------------------------------|
| 📊 **Dashboard**        | See all tasks at a glance with real-time progress       |
| 🎨 **Workflow Diagram** | Visual state showing current step and available actions |
| 📝 **Task Creation**    | Write tasks directly in browser or upload files         |
| 📜 **Live Output**      | Watch progress as it happens                            |
| 🔙 **Undo/Redo**        | Easy checkpoint navigation                              |
| 💬 **Notes**            | Add context for the workflow at any point               |
| 💰 **Budgets**          | Set cost/token limits with warnings                     |
| 🔍 **History**          | Browse and search past tasks                            |
| 🌓 **Dark Mode**        | Toggle between light and dark themes                    |
| 📱 **Mobile Ready**     | Full functionality on phone or tablet                   |

### Getting Started with CLI

```bash
# Create a task file
cat > task.md << 'EOF'
---
title: Add user authentication
---
Add login and signup pages with JWT tokens.
EOF

# Run the workflow
mehr start task.md
mehr plan
mehr implement
mehr finish
```

---

## CLI Commands

### Workflow Commands

| Command            | Description                                           |
|--------------------|-------------------------------------------------------|
| `mehr init`        | Initialize workspace (creates `.mehrhof/config.yaml`) |
| `mehr start <ref>` | Start task from file, directory, or provider          |
| `mehr plan`        | Generate structured plan from task description        |
| `mehr implement`   | Execute the plan to create changes                    |
| `mehr review`      | Run quality checks on changes                         |
| `mehr finish`      | Complete task and merge changes                       |
| `mehr auto <ref>`  | Full workflow: plan → implement → review → finish     |

### Task Management

| Command                   | Description                                   |
|---------------------------|-----------------------------------------------|
| `mehr status`             | Show current task state with workflow diagram |
| `mehr list`               | List all tasks with search, filter, and sort  |
| `mehr continue`           | Resume work on task                           |
| `mehr guide`              | Get suggestion for next action                |
| `mehr note <msg>`         | Add context notes for the workflow            |
| `mehr question <msg>`     | Ask questions during workflow steps           |
| `mehr undo` / `mehr redo` | Navigate checkpoints                          |
| `mehr reset`              | Reset stuck workflow state to idle            |
| `mehr abandon`            | Abandon task without merging                  |

### Providers

| Command                     | Description            |
|-----------------------------|------------------------|
| `mehr start file:task.md`   | Load from file         |
| `mehr start github:123`     | Load from GitHub issue |
| `mehr start jira:PROJ-123`  | Load from Jira         |
| `mehr start linear:ENG-123` | Load from Linear       |
| `mehr start notion:<uuid>`  | Load from Notion       |

Supports 16+ sources: Empty, File, Directory, GitHub, GitLab, Bitbucket, Jira, Linear, Asana, ClickUp, Azure DevOps, Notion, Trello, Wrike, YouTrack, and Research.

### Advanced

| Command               | Description                                            |
|-----------------------|--------------------------------------------------------|
| `mehr serve`          | Start Web UI server                                    |
| `mehr interactive`    | REPL mode with chat and workflow control               |
| `mehr project plan`   | Create task breakdown with dependencies                |
| `mehr project submit` | Submit tasks to provider                               |
| `mehr browser`        | Browser automation commands                            |
| `mehr mcp`            | Start MCP server for external integration              |
| `mehr scan`           | Run security scanners (SAST, secrets, vulnerabilities) |
| `mehr cost`           | View token usage and costs                             |
| `mehr memory <query>` | Search semantic memory for similar past tasks          |

**Tip:** Use command shortcuts for faster typing: `mehr gu` → `guide`, `mehr config:v` → `config validate`.

See [CLI Reference](https://valksor.com/docs/mehrhof/nightly/#/cli/index) for all commands and flags.

---

## Use Cases

Mehrhof helps when you need to:

- **Ship changes safely** — Review gates ensure nothing goes live without human approval
- **Turn requirements into change sets** — Describe what you need, get structured output you can review
- **Enable non-technical contributors** — Product managers, ops, and business users can create changes within guardrails
- **Maintain repeatability** — Same workflow every time; auditable history for enterprise settings
- **Reduce single points of failure** — Less reliance on one person who "knows how everything works"

---

## Non-Goals

Mehrhof is intentionally limited. It is **not**:

- **A fully autonomous coding system** — Human review is required at every step
- **An AI API proxy or hosted service** — Mehrhof wraps your local agent CLI; it doesn't provide AI access
- **An unsupervised production code writer** — All output must be reviewed before merging
- **A replacement for understanding** — You should understand what changes are being made

These limitations are features, not bugs. They keep you in control.

---

## Installation

### Install Script (Recommended)

```bash
curl -fsSL https://raw.githubusercontent.com/valksor/go-mehrhof/master/install.sh | bash
```

### Pre-built Binary

```bash
# Download for your platform (macOS ARM64 example)
curl -L https://github.com/valksor/go-mehrhof/releases/latest/download/mehr-darwin-arm64 -o mehr
chmod +x mehr
sudo mv mehr /usr/local/bin/
```

### Build from Source

```bash
git clone https://github.com/valksor/go-mehrhof.git
cd go-mehrhof
make install
```

**Prerequisite**: You need an agent CLI installed separately (e.g., [Claude](https://claude.ai/code)). Mehrhof orchestrates it but does not include it. See [Agents](https://valksor.com/docs/mehrhof/nightly/#/agents/index) for supported agents.

---

## Verifying Binary Authenticity

All stable releases are signed with [Minisign](https://github.com/jedisct1/minisign) to ensure authenticity.

**Public Key**: `RWTFiZ4b+sgoFLiIMuMrTZr1mmropNlDsnwKl5RfoUtyUWUk4zyVpPw2`

```bash
# Download and verify
curl -L -O https://github.com/valksor/go-mehrhof/releases/latest/download/mehr-linux-amd64
curl -L -O https://github.com/valksor/go-mehrhof/releases/latest/download/checksums.txt
curl -L -O https://github.com/valksor/go-mehrhof/releases/latest/download/checksums.txt.minisig

minisign -Vm checksums.txt -P "RWTFiZ4b+sgoFLiIMuMrTZr1mmropNlDsnwKl5RfoUtyUWUk4zyVpPw2" \
    -x checksums.txt.minisig
sha256sum -c checksums.txt --ignore-missing
```

---

## Configuration

Mehrhof is configurable to match your team's needs. Create `.mehrhof/config.yaml` in your project:

```yaml
# Git integration
git:
  auto_commit: true
  branch_pattern: "{type}/{key}--{slug}"
  target_branch: "main"

# Workflow settings
workflow:
  require_review: true    # Enforce a review step before finishing

# Save plans in a project for version control
specification:
  save_in_project: true
  project_dir: "plans"
```

See [Configuration Guide](https://valksor.com/docs/mehrhof/nightly/#/configuration/index) for all options.

---

## Documentation

- [Full Documentation](https://valksor.com/docs/mehrhof/nightly)
- [Quickstart](https://valksor.com/docs/mehrhof/nightly/#/quickstart) — Install and run your first task
- [Web UI Guide](https://valksor.com/docs/mehrhof/nightly/#/web-ui/getting-started) — Visual walkthrough
- [CLI Reference](https://valksor.com/docs/mehrhof/nightly/#/cli/index) — All commands and flags
- [Workflow Concepts](https://valksor.com/docs/mehrhof/nightly/#/concepts/workflow) — Understanding the process
- [Configuration](https://valksor.com/docs/mehrhof/nightly/#/configuration/index) — Customize behavior

---

## Development

```bash
make build        # Build binary
make test         # Run tests
make quality      # Lint and security checks
make install      # Install to $GOPATH/bin
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for development guidelines.

---

## Contributing

Contributions are welcome. Please read [CONTRIBUTING.md](CONTRIBUTING.md) before submitting changes.

By contributing to Mehrhof, you agree that your contributions will be licensed under the [BSD 3-Clause License](LICENSE).

---

## License

[BSD 3-Clause License](LICENSE)
