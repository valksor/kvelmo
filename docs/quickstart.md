# Quickstart

Get kvelmo up and running in 5 minutes.

## What You'll Accomplish

By the end of this guide, you'll have:
1. Installed kvelmo on your computer
2. Started your first task
3. Seen the plan → implement → review workflow in action

No prior experience with command-line tools is required if you choose the Desktop App or Web UI path.

---

## Prerequisites

**Git** — kvelmo uses Git for version control and checkpoints. [Install Git](https://git-scm.com/downloads) if you don't have it.

**Agent CLI** — kvelmo coordinates your local AI agent. Claude is recommended, but Codex and custom agents are also supported.

> **Agent Ready?**
>
> kvelmo uses your agent CLI directly. If `claude --version` works, you're ready.

```bash
# Check if Claude is installed
claude --version
```

If you don't have Claude, follow the [Claude setup guide](https://claude.ai/code). For other agents, see [AI Agents](/agents/index.md).

---

## Install kvelmo

```bash
curl -fsSL https://raw.githubusercontent.com/valksor/kvelmo/master/install.sh | bash
```

Verify the installation:

```bash
kvelmo version
```

> **Starting from scratch?** See the complete [Installation Guide](/INSTALL.md) for WSL, Git, and agent CLI setup.

---

## Choose Your Interface

kvelmo works through multiple interfaces. Choose what matches your comfort level:

| Interface | Best For | Setup |
|-----------|----------|-------|
| **Desktop App** | Visual users, no terminal needed | [Download](https://github.com/valksor/kvelmo/releases) |
| **Web UI** | Visual users, one terminal command | `kvelmo serve --open` |
| **CLI** | Power users, scripting, CI/CD | Terminal commands |

All interfaces have the same features — the choice is about preference, not capability.

---

## Desktop App — Native Experience

The Desktop App is ideal if you want a fully native experience without using the terminal.

**Get started:**

1. Download from [Releases](https://github.com/valksor/kvelmo/releases) for your platform
2. Install and launch the app
3. Select your project folder
4. Click **New Task** to begin

**[Full Desktop App Guide](/desktop/getting-started.md)** — Complete walkthrough

---

## Web UI — Visual Workflow

The Web UI is ideal if you prefer a visual interface.

**Get started:**

```bash
kvelmo serve --open
```

1. Your browser opens automatically
2. Navigate to your project folder
3. Click **Create Task** to begin

**What you'll see:**
- A dashboard showing your current task
- Buttons to plan, implement, and review
- Real-time streaming of progress
- Undo/redo controls if something goes wrong

**[Full Web UI Guide](/web-ui/getting-started.md)** — Complete walkthrough with visual examples

---

## CLI — Power User Workflow

The CLI is ideal for text-based workflows, scripting, and CI/CD pipelines.

**Try it in 60 seconds:**

```bash
cd /path/to/your/project

cat > task.md << 'EOF'
---
title: Add hello world endpoint
---
Add a GET /hello endpoint that returns "Hello, World!".
EOF

kvelmo start --from file:task.md
kvelmo plan
kvelmo implement
kvelmo review
kvelmo submit
```

**[Full CLI Tutorial](/guides/first-task.md)** — Step-by-step guide

---

## The Workflow

### 1. Start

Load a task and prepare your workspace:

```bash
kvelmo start --from file:task.md
```

### 2. Plan

Generate a specification you can review before any changes happen:

```bash
kvelmo plan
```

**Why this matters:** You approve the plan before any code is written. Nothing proceeds without your review.

### 3. Implement

Build your changes based on the approved plan:

```bash
kvelmo implement
```

Watch the agent work in real-time. If something's wrong, use `kvelmo undo` to revert.

### 4. Review

Review the changes before they ship:

```bash
kvelmo review
```

### 5. Submit

Create a PR when you're satisfied:

```bash
kvelmo submit
```

---

## Check Status

At any point, see where you are:

```bash
kvelmo status
```

---

## What's Next?

- [Desktop App Guide](/desktop/getting-started.md) — Native application walkthrough
- [Web UI Guide](/web-ui/getting-started.md) — Browser interface walkthrough
- [CLI Reference](/cli/index.md) — All commands and options
- [Workflow Concepts](/concepts/workflow.md) — Understanding the process
- [Providers](/providers/index.md) — Pull tasks from GitHub, GitLab, Linear
