# Desktop App

> **Zero terminal required.** Launch the app, pick a folder, and start working.

The Mehrhof Desktop App gives you the full Web UI experience as a native application. No browser tabs, no terminal commands — just install, launch, and go.

---

## Installation

### macOS

1. Download the `.dmg` from the [releases page](https://github.com/valksor/go-mehrhof/releases)
2. Open the file and drag **Mehrhof** to your Applications folder
3. Launch from Applications

> **First launch:** macOS may show a security warning. Right-click the app and select **Open**, then click **Open** again in the dialog. This is only needed once. Proper code signing is coming soon.

### Windows

1. Make sure you have [WSL2 installed](https://learn.microsoft.com/en-us/windows/wsl/install)
2. Download the installer from the [releases page](https://github.com/valksor/go-mehrhof/releases)
3. Run the installer and launch Mehrhof

> **First launch:** If Mehrhof isn't installed in WSL yet, the app will install it for you automatically.

### Linux

1. Download the `.deb` package from the [releases page](https://github.com/valksor/go-mehrhof/releases)
2. Double-click to install, or use your package manager

---

## Getting Started

### Opening a Project

1. Launch the app — you'll see the project picker
2. Click **Open Folder** and select your project directory
3. That's it! You're ready to create tasks

Your projects are remembered automatically. Next time you launch, just click on a recent project.

### Favorites

Keep your most-used projects at the top:

1. Hover over a project in the list
2. Click the star icon
3. Starred projects appear first

### Switching Projects

To work on a different project, use the project menu in the top bar to switch or open a new folder.

---

## Using the App

Once you've opened a project, the desktop app works exactly like the Web UI. All features are available:

| Feature           | Learn More                                  |
|-------------------|---------------------------------------------|
| Creating tasks    | [Creating Tasks](/web-ui/creating-tasks.md) |
| Planning workflow | [Planning](/web-ui/planning.md)             |
| Implementing      | [Implementing](/web-ui/implementing.md)     |
| Code review       | [Reviewing](/web-ui/reviewing.md)           |
| Undo & redo       | [Undo & Redo](/web-ui/undo-redo.md)         |
| Settings          | [Settings](/web-ui/settings.md)             |

For the complete feature list, see [Web UI Overview](/web-ui/index.md).

---

## Windows Notes

On Windows, the app runs natively but uses WSL (Windows Subsystem for Linux) behind the scenes. A few things to know:

- **Use local drives only** — Network drives and mapped shares aren't supported
- **First launch takes longer** — The app may need to install components in WSL
- **Projects on Windows drives work fine** — The app handles path conversion automatically

---

## Troubleshooting

### macOS: Security warning on first launch

Right-click the app → **Open** → Click **Open** in the dialog. This tells macOS you trust the app.

### Windows: "WSL is not installed"

You need Windows Subsystem for Linux. Open PowerShell and run:

```
wsl --install
```

Restart your computer when prompted.

### App won't start or shows error

1. Try launching again — sometimes the first launch takes time
2. Make sure you have a working internet connection (needed for first-time setup on Windows)
3. Check that no firewall is blocking the app

### Projects not showing up

Projects appear after you open them at least once. Click **Open Folder** to add a new project.

---

## Also Available

| Option                               | Best For                                     |
|--------------------------------------|----------------------------------------------|
| [Web UI](/web-ui/index.md)           | Using in a browser without installing an app |
| [CLI](/cli/index.md)                 | Power users who prefer the terminal          |
| [VS Code](/ide/vscode/index.md)      | Working inside VS Code                       |
| [JetBrains](/ide/jetbrains/index.md) | Working inside IntelliJ, PyCharm, etc.       |
