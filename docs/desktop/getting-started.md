# Getting Started with Desktop App

This guide walks you through your first task using the kvelmo Desktop App.

## Installation

### macOS

1. Download the `.dmg` file for your Mac (Apple Silicon or Intel)
2. Open the downloaded file
3. Drag kvelmo to your Applications folder
4. Launch from Applications or Spotlight

**First launch:** macOS may show a security warning. Go to **System Settings > Privacy & Security** and click **Open Anyway**.

### Windows

1. Download the `.exe` installer
2. Run the installer
3. Launch from the Start menu

### Linux

1. Download the `.AppImage` file
2. Make it executable: `chmod +x kvelmo-linux-amd64.AppImage`
3. Run the AppImage

---

## First Launch

When you open kvelmo for the first time:

1. **Select a project folder** — Click **Open Folder** and choose your project directory
2. **Project initializes** — kvelmo creates a `.kvelmo/` folder to track your work
3. **Dashboard appears** — You're ready to create your first task

---

## Your First Task

### 1. Create a Task

Click **New Task** and describe what you want to build:

```
Add a login page with email and password fields
```

Or import from your issue tracker (GitHub, GitLab, Linear, etc.).

### 2. Plan

Click **Plan** to generate a specification. Review it before any code changes happen.

**Why this matters:** You approve the plan before any code is written. If the plan misunderstands your intent, refine it now.

### 3. Implement

Click **Implement** to build the changes. Watch progress in real-time.

If something goes wrong, click **Undo** to revert and try again.

### 4. Review

Click **Review** to see all changes. Security scanning runs automatically if configured.

Approve the changes or undo to try a different approach.

### 5. Submit

Click **Submit** to create a PR and mark the task complete in your issue tracker.

---

## Navigation

| Area | What It Shows |
|------|---------------|
| **Sidebar** | Task list, project selector |
| **Main Panel** | Current task details and controls |
| **Output Panel** | Real-time agent output and logs |
| **Status Bar** | Current state, undo/redo controls |

---

## Switching Between Interfaces

Your work is portable. You can:

- Start a task in Desktop App, continue in CLI
- Review changes in Web UI, submit from Desktop App
- Use whatever interface fits your current context

All interfaces share the same workflow state.

---

## Keyboard Shortcuts

| Action | macOS | Windows/Linux |
|--------|-------|---------------|
| New Task | Cmd+N | Ctrl+N |
| Undo | Cmd+Z | Ctrl+Z |
| Redo | Cmd+Shift+Z | Ctrl+Shift+Z |
| Settings | Cmd+, | Ctrl+, |

---

## What's Next?

- [Workflow Concepts](/concepts/workflow.md) — Understand the full process
- [Configuration](/configuration/index.md) — Customize kvelmo for your team
- [Providers](/providers/index.md) — Connect to GitHub, GitLab, Linear, Wrike
