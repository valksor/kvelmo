# Desktop App

The Mehrhof Desktop App provides a native application experience for managing your projects.

> **Zero terminal required.** Just launch the app, pick a folder, and start working. The Desktop App handles project initialization automatically.

## Features

- **Project picker** - Launch with recent projects and favorites at your fingertips
- **Native integration** - Opens project folders with native file dialogs
- **Auto-tracking** - Projects are automatically tracked when you open them
- **Cross-platform** - Available for Windows (via WSL), macOS, and Linux

## Installation

### macOS

Download the latest `.dmg` from the [releases page](https://github.com/valksor/go-mehrhof/releases) and drag to Applications.

> **Gatekeeper Warning:** macOS may show "App is from an unidentified developer" when you first open Mehrhof. Right-click the app and select **Open** to bypass this warning. Proper code signing is coming soon.

### Windows

1. Ensure [WSL2](https://learn.microsoft.com/en-us/windows/wsl/install) is installed
2. Download and run the installer from the [releases page](https://github.com/valksor/go-mehrhof/releases)

> **Note:** If mehr is not installed in WSL, the app will automatically install it for you on first launch. Nightly builds install nightly mehr; stable builds install stable mehr.

### Linux

Download the `.deb` package from the [releases page](https://github.com/valksor/go-mehrhof/releases) and install with:

```bash
sudo dpkg -i mehrhof_*.deb
```

## Usage

### Opening a Project

1. Launch the Mehrhof app
2. Select a recent project from the list, or click **Open Folder...** to choose a new project
3. The app automatically initializes your project and starts the web UI

No `cd` or `mehr init` required — the project picker handles everything.

### Favorites

Star frequently-used projects to keep them at the top of your list:

1. Hover over a project in the list
2. Click the star icon to toggle favorite status
3. Favorites persist across sessions

### Project Tracking

Projects are automatically tracked when you open them. The app maintains a list of up to 20 recent projects, stored in:

- **macOS/Linux**: `~/.valksor/mehrhof/projects.json`
- **Windows (WSL)**: Same location within WSL filesystem

This file is shared with the CLI and Web UI global mode, so your recent projects stay synchronized across all interfaces.

## Windows + WSL

On Windows, the desktop app runs natively but communicates with `mehr` running inside WSL:

1. The app converts Windows paths to WSL paths (e.g., `C:\Users\foo` → `/mnt/c/Users/foo`)
2. It spawns `wsl -e bash -c "cd '<path>' && mehr serve --port 0"`
3. Parses the port from stdout and navigates to the server

**Network paths** (UNC paths like `\\server\share`) are not supported. Use local drives only.

## Troubleshooting

### "WSL is not installed"

Install WSL2:
```powershell
wsl --install
```

Restart your computer and ensure a Linux distribution is set up.

### "mehr is not installed in WSL"

Open WSL terminal and run:
```bash
curl -fsSL https://install.mehrhof.dev | bash
```

### Server startup timeout

If the server takes longer than 30 seconds to start:

1. Check that `mehr` works: `mehr --version`
2. Ensure no firewall is blocking localhost connections
3. Try running `mehr serve` manually to see any errors

### Project not appearing in list

Projects are tracked when `mehr serve` runs. To add a project:

1. Open the project folder in the desktop app, or
2. Run `mehr serve` in the project directory from the CLI

---

## Also Available

| Interface | Description |
|-----------|-------------|
| [CLI](/cli/serve.md) | Command-line interface |
| [Web UI](/web-ui/index.md) | Browser-based interface |
| [VS Code](/ide/vscode/index.md) | Editor integration |
| [JetBrains](/ide/jetbrains/index.md) | IDE integration |
