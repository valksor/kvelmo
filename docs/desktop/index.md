# Desktop App

The kvelmo Desktop App provides a native experience for managing your development workflow without using the terminal.

## Why Use the Desktop App?

- **No terminal required** — Launch from your dock or start menu
- **Native notifications** — System-level alerts for task progress
- **Offline-ready** — Works without a browser
- **Same workflow** — Identical features to Web UI and CLI

## Download

Download the latest version for your platform:

| Platform | Download |
|----------|----------|
| macOS (Apple Silicon) | [kvelmo-darwin-arm64.dmg](https://github.com/valksor/kvelmo/releases/latest/download/kvelmo-darwin-arm64.dmg) |
| macOS (Intel) | [kvelmo-darwin-amd64.dmg](https://github.com/valksor/kvelmo/releases/latest/download/kvelmo-darwin-amd64.dmg) |
| Windows | [kvelmo-windows-amd64.exe](https://github.com/valksor/kvelmo/releases/latest/download/kvelmo-windows-amd64.exe) |
| Linux | [kvelmo-linux-amd64.AppImage](https://github.com/valksor/kvelmo/releases/latest/download/kvelmo-linux-amd64.AppImage) |

Or visit the [Releases page](https://github.com/valksor/kvelmo/releases) for all versions.

## Prerequisites

Before using the Desktop App, you need an AI agent CLI installed:

1. Install [Claude](https://claude.ai/code) (recommended) or another supported agent
2. Verify it works: `claude --version`

The Desktop App uses your existing agent CLI — no additional API keys needed.

## Getting Started

1. Download and install for your platform
2. Launch kvelmo from your applications
3. Select your project folder
4. Start your first task

See [Getting Started](/desktop/getting-started.md) for a complete walkthrough.

## Shared Workflow

The Desktop App uses the same workflow as CLI and Web UI:

```
START → PLAN → IMPLEMENT → REVIEW → SUBMIT
```

Your work is portable — start a task in the Desktop App, continue it in CLI, review it in Web UI. All interfaces share the same underlying state.

## Documentation

| Guide | Description |
|-------|-------------|
| [Getting Started](/desktop/getting-started.md) | First launch and project setup |
| [Workflow](/concepts/workflow.md) | Understanding the process |
| [Configuration](/configuration/index.md) | Customize behavior |

---

## Also Available

- **[Web UI](/web-ui/getting-started.md)** — Browser-based interface, run `kvelmo serve --open`
- **[CLI](/cli/index.md)** — Terminal-based for scripting and automation
