# Windows & WSL2 Guide

Mehrhof runs inside WSL2 (Windows Subsystem for Linux) on Windows. This guide covers setup, best practices, and tips for a smooth development experience.

## Why WSL2?

Mehrhof depends on tools that require a Unix environment: Git with POSIX-compatible branching, Claude CLI, and optional features like sandboxing and semantic memory. Rather than maintaining a partially-compatible Windows build, Mehrhof runs as a native Linux binary inside WSL2 — giving you full feature parity with Linux and macOS.

WSL2 runs a real Linux kernel. Performance is near-native for CPU and memory workloads. The main thing to get right is **where you store your project files** (see below).

## Initial Setup

If you haven't installed WSL2 yet, follow the [Installation Guide](/INSTALL.md#windows-wsl-setup) for step-by-step instructions covering PowerShell admin access, `wsl --install`, Ubuntu setup, and Mehrhof installation.

Once WSL2 is running, install Mehrhof from inside your Linux shell:

```bash
curl -fsSL https://raw.githubusercontent.com/valksor/go-mehrhof/master/install.sh | bash
```

---

## Project Location (Important)

**Store your projects on the Linux filesystem, not on the Windows drive.**

This is the single most impactful tip for Windows/WSL2 users. Files on the mounted Windows filesystem (`/mnt/c/Users/...`) go through a translation layer that makes Git, Go builds, and file watchers significantly slower.

```bash
# Good — native Linux filesystem (fast)
~/projects/my-app/

# Avoid — mounted Windows drive (slow for development)
/mnt/c/Users/yourname/Documents/my-app/
```

**What to expect:**
- Git operations: 5-20x faster on Linux filesystem
- Go builds: 3-10x faster
- File watchers (`mehr serve` live reload): responsive vs noticeable lag

**If you have existing projects on the Windows drive**, clone a fresh copy to the Linux filesystem:

```bash
mkdir -p ~/projects
cd ~/projects
git clone <your-repo-url>
```

You can still access your Linux files from Windows Explorer at `\\wsl$\Ubuntu\home\yourname\projects\` — the files are visible from both sides, but the *native* side is where performance matters.

---

## IDE Integration

Your IDE runs on Windows; Mehrhof runs inside WSL. Modern IDEs handle this seamlessly.

### VS Code (Recommended for WSL)

VS Code's WSL extension makes the boundary nearly invisible. The extension server runs inside WSL, so terminals, file access, and extensions all operate natively in Linux.

1. Install the [WSL extension](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-wsl) in VS Code
2. Open your project from the WSL terminal:
   ```bash
   cd ~/projects/my-app
   code .
   ```
   Or use the Command Palette: **WSL: Connect to WSL**
3. The Mehrhof VS Code extension works normally — it finds `mehr` from WSL's PATH

See [VS Code: Windows (WSL)](/ide/vscode.md#windows-wsl) for detailed setup.

### JetBrains IDEs

JetBrains 2025.1+ supports WSL as a remote backend. The plugin connects to `mehr serve` running inside WSL.

1. Open your project via the WSL path: `\\wsl$\Ubuntu\home\yourname\project`
2. Or use **File > Remote Development > WSL** to connect

See [JetBrains: Windows (WSL)](/ide/jetbrains.md#windows-wsl) for detailed setup.

---

## Web UI Access

When you run `mehr serve` inside WSL, the server binds to localhost. On modern Windows (11 22H2+ with WSL 2.0+), ports are automatically forwarded — open `http://localhost:8080` in your Windows browser and it works.

```bash
# Inside WSL
mehr serve --open
```

**Note:** The `--open` flag may not launch your Windows browser automatically from WSL. If it doesn't, copy the URL from the terminal output and paste it into your Windows browser manually.

### Older Windows Builds

If `localhost` doesn't reach WSL, your system may use the older NAT networking mode. Options:

1. **Update WSL** to get mirrored networking:
   ```bash
   wsl --update
   ```

2. **Find the WSL IP** and use that:
   ```bash
   hostname -I
   # Use the IP shown, e.g., http://172.x.x.x:8080
   ```

3. **Bind to all interfaces** (allows Windows to connect via any address):
   ```bash
   mehr serve --host 0.0.0.0
   ```

---

## Docker Integration

Docker Desktop for Windows uses WSL2 as its backend. Mehrhof's browser automation and other Docker-dependent features work when Docker Desktop is configured correctly.

### Setup

1. Install [Docker Desktop](https://docs.docker.com/desktop/install/windows-install/)
2. Open Docker Desktop **Settings > Resources > WSL Integration**
3. Enable integration with your WSL distro (e.g., Ubuntu)
4. Verify from inside WSL:
   ```bash
   docker --version
   docker run hello-world
   ```

Once configured, `docker` commands inside WSL use Docker Desktop's engine. No separate Linux Docker installation is needed.

---

## Clipboard

WSL can access the Windows clipboard via `clip.exe`:

```bash
# Copy command output to Windows clipboard
mehr status | clip.exe

# Copy a file's contents
cat .mehrhof/config.yaml | clip.exe
```

Pasting into WSL depends on your terminal: Windows Terminal supports `Ctrl+V`, while other terminals may use `Shift+Insert` or right-click.

---

## Multiple Distros

WSL2 supports multiple Linux distributions. Mehrhof works in any of them — Ubuntu, Debian, Fedora, Arch, etc. Install Mehrhof separately in each distro where you want to use it.

```bash
# List installed distros
wsl --list --verbose

# Open a specific distro
wsl -d Ubuntu
```

Each distro has its own filesystem. Projects in one distro are not automatically visible in another (though you can access them via `/mnt/wsl/`).

---

## Tips

**Terminal**: [Windows Terminal](https://apps.microsoft.com/detail/9N0DX20HK701) provides tabs, split panes, and proper Unicode/color support. It's the recommended terminal for WSL.

**Shell profile**: Your `~/.bashrc` or `~/.zshrc` inside WSL works the same as on any Linux system. Add `mehr` to your PATH here if the installer didn't do it automatically:

```bash
export PATH="$PATH:$HOME/.local/bin"
```

**Git credentials**: If you use HTTPS Git remotes, install [Git Credential Manager](https://github.com/git-ecosystem/git-credential-manager) which integrates with Windows credential storage from inside WSL.

**File watchers**: If you see "too many open files" errors, increase the inotify limit:

```bash
echo "fs.inotify.max_user_watches=524288" | sudo tee -a /etc/sysctl.conf
sudo sysctl -p
```

---

## Troubleshooting

For WSL-specific issues, see [Troubleshooting: Windows / WSL2](/troubleshooting/windows-wsl.md).

## See Also

- [Installation Guide](/INSTALL.md#windows-wsl-setup) — Initial WSL2 and Mehrhof installation
- [Quickstart](/quickstart.md) — Get started in 5 minutes
- [VS Code Extension](/ide/vscode.md) — VS Code integration
- [JetBrains Plugin](/ide/jetbrains.md) — JetBrains integration
