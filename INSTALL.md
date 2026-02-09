# Installation Guide

> **Last verified:** February 2026

Complete guide to install Mehrhof and all prerequisites from scratch.

---

## Before You Begin (First-Time Users)

**New to command lines?** This section is for you. If you've used a terminal before, skip to [Quick Start](#quick-start-experienced-users).

### What is a Terminal?

A **terminal** (also called "command line", "console", or "shell") is a text-based way to control your computer. Instead of clicking buttons and icons, you type commands.

It looks like a window with text, usually with a **prompt** like this:
```
username@computer:~$
```
or on Mac:
```
username@MacBook ~ %
```

The `$` or `%` at the end means the computer is waiting for you to type something.

### How to Open a Terminal

**macOS:**
1. Press **Cmd + Space** to open Spotlight search
2. Type **Terminal**
3. Press **Enter**

**Windows (after WSL is installed - see [Windows section](#windows-wsl-setup)):**
1. Click the **Start** button
2. Type **Ubuntu**
3. Click to open

**Ubuntu/Linux:**
1. Press the **Super** key (the Windows logo key on your keyboard)
2. Type **Terminal**
3. Click to open

### Running Commands

When you see a command in this guide like:
```bash
sudo apt update
```

Here's what to do:
1. **Copy** the command (highlight it, then Ctrl+C or Cmd+C)
2. **Paste** it into your terminal (Ctrl+Shift+V on Linux, Cmd+V on Mac, or right-click → Paste)
3. Press **Enter** to run it

**What to expect:** You'll see text scroll by - this is normal! Commands often print information as they work.

### Passwords in the Terminal

Some commands start with `sudo` (like `sudo apt update`). This runs the command with administrator privileges.

When you run a `sudo` command:
- You'll be asked for your password
- **The characters won't appear as you type** - this is a security feature, not a bug
- Just type your password and press Enter

On Linux/WSL, this is your Linux password (the one you created during setup), not your Windows password.

---

## Which Computer Do I Have?

Some downloads depend on your computer's **architecture** (the type of processor inside). Here's how to check:

**Mac:**
1. Click the **Apple menu** (top-left corner)
2. Click **About This Mac**
3. Look for **Chip** or **Processor**:
   - **Apple M1, M2, M3, or M4** → You have **Apple Silicon (arm64)**
   - **Intel** → You have **Intel (amd64)**

**Linux:**
Open a terminal and run:
```bash
uname -m
```
- `x86_64` → You have **amd64**
- `aarch64` → You have **arm64**

**Windows:**
Don't worry about this - WSL handles it automatically.

---

## Key Terms Reference

| Term | What It Means |
|------|---------------|
| **Terminal** | A text-based window where you type commands |
| **CLI** | "Command Line Interface" - a program you control by typing, not clicking |
| **`sudo`** | "Super user do" - runs a command with admin privileges |
| **`curl`** | A program that downloads files from the internet |
| **`bash`** | A program that runs commands (a "shell") |
| **PATH** | A list of folders where your computer looks for programs |
| **`~`** | Shortcut for your home folder (e.g., `/home/yourname/` or `/Users/yourname/`) |
| **Binary** | A ready-to-run program file |
| **Repository/Repo** | A project folder managed by Git (version control) |

---

## Desktop App (Easiest Option)

Prefer a native app with zero terminal? The Desktop App bundles everything — no commands needed after download.

1. Download from [releases](https://github.com/valksor/go-mehrhof/releases)
2. Install and launch
3. Pick a folder and start working

**Windows users:** The app requires WSL2 but will auto-install `mehr` for you.

See [Desktop App documentation](docs/desktop/index.md) for details.

---

## Quick Start (Experienced Users)

> **New to terminals?** Skip this section and start at [Before You Begin](#before-you-begin-first-time-users) instead.

Already have Git and an AI agent CLI installed? Run:

```bash
curl -fsSL https://raw.githubusercontent.com/valksor/go-mehrhof/master/install.sh | bash
```

### Requirements Checklist

| Requirement | Check Command | Required? |
|-------------|---------------|-----------|
| Git | `git --version` | Yes |
| Agent CLI (Claude recommended) | `claude --version` | Yes |
| Go 1.25+ | `go version` | Only for building from source |
| curl | `curl --version` | Yes (for install script) |

If any are missing, follow the relevant sections below.

---

## Table of Contents

**Quickest Options:**
- [Desktop App](#desktop-app-easiest-option) - Zero terminal, just download and run
- [Quick Start](#quick-start-experienced-users) - For experienced users

**First-Time Users:**
- [Before You Begin](#before-you-begin-first-time-users) - What is a terminal? How to open it
- [Which Computer Do I Have?](#which-computer-do-i-have) - Check your architecture
- [Key Terms Reference](#key-terms-reference) - Glossary of technical terms

**Installation:**
- [Platform Support](#platform-support)
- [Windows (WSL Setup)](#windows-wsl-setup)
- [Git Installation](#git-installation)
- [Go Installation](#go-installation-optional)
- [Claude CLI Installation](#claude-cli-installation)
- [Codex Installation](#codex-installation-optional)
- [Mehrhof Installation](#mehrhof-installation)
- [Post-Installation Setup](#post-installation-setup)
- [Troubleshooting](#troubleshooting)

---

## Platform Support

| Platform | Architecture | Support |
|----------|--------------|---------|
| Linux (all distros) | x86_64 (amd64) | Full |
| Linux (all distros) | ARM64 (aarch64) | Full |
| macOS | Apple Silicon (arm64) | Full |
| macOS | Intel (amd64) | Full |
| Windows | via WSL2 | Full (run Linux binary in WSL) |

**Time estimates:**
- Fresh Windows system (needs WSL): ~30-45 minutes
- Fresh Linux/macOS (needs Git + Claude): ~15-20 minutes
- Has prerequisites: ~2 minutes

---

## Windows (WSL Setup)

Windows users must use WSL2 (Windows Subsystem for Linux). Native Windows is not supported.

### Quick Install (PowerShell)

If you already have WSL2 set up with a Linux distribution, run this in PowerShell to install Mehrhof:

```powershell
irm https://raw.githubusercontent.com/valksor/go-mehrhof/master/install.ps1 | iex
```

The script checks your WSL2 setup and runs the Linux installer inside WSL. If WSL2 isn't installed, it will give you setup instructions.

### Manual Setup

If you're setting up WSL2 for the first time, follow these steps:

### Step 1: Enable WSL2

First, open **PowerShell as Administrator**:

1. **Right-click** the **Start** button (Windows icon, bottom-left)
2. Click **Terminal (Admin)** or **Windows PowerShell (Admin)**
3. If a popup asks "Do you want to allow this app to make changes?" click **Yes**

Now type (or paste) this command and press Enter:

```powershell
wsl --install
```

This installs WSL2 with Ubuntu by default. **Restart your computer when prompted.**

### Step 2: First Launch

After restart, Ubuntu will launch automatically. If not:
1. Click **Start**
2. Type **Ubuntu**
3. Click to open

You'll be asked to create a **Linux username and password**:
- This is **separate from your Windows login** - you can use different credentials
- Pick something simple you'll remember
- **When typing your password, no characters will appear** - this is normal, just type and press Enter
- You'll be asked to type it twice to confirm

### Step 3: Update Ubuntu

This command updates Ubuntu's software list and installs any available updates:

```bash
sudo apt update && sudo apt upgrade -y
```

- `sudo` = run with admin privileges (you'll enter your Linux password)
- `apt update` = refresh the list of available software
- `apt upgrade -y` = install updates (`-y` means "yes to all")

**Expect:** Lots of text will scroll by. This can take a few minutes. When you see the prompt again (ending in `$`), it's done.

### Step 4: Verify WSL2

```bash
wsl --version
```

You should see "WSL version: 2.x.x" or similar.

**All remaining instructions should be run inside WSL/Ubuntu**, not in PowerShell.

*Reference: [Microsoft WSL Documentation](https://learn.microsoft.com/en-us/windows/wsl/install) for troubleshooting.*

---

## Git Installation

### Debian/Ubuntu (including WSL)

```bash
sudo apt update
sudo apt install git -y
```

### Fedora/RHEL/CentOS

```bash
# Fedora / RHEL 8+
sudo dnf install git -y

# Older RHEL/CentOS (7 and below)
sudo yum install git -y
```

### Arch Linux

```bash
sudo pacman -S git
```

### Alpine Linux

```bash
apk add git
```

### macOS

> **Beginners:** Use Option 1 - just run the command and click "Install" when prompted.

**Option 1: Xcode Command Line Tools** (recommended for beginners)

```bash
xcode-select --install
```

A popup will appear. Click "Install" and wait for completion (~5-10 minutes).

**Option 2: Homebrew** (if you already use Homebrew)

```bash
brew install git
```

### Configure Git

Git needs to know who you are so it can label your changes. Run these two commands, **replacing the example values with your actual name and email**:

```bash
git config --global user.name "Jane Smith"
git config --global user.email "jane.smith@example.com"
```

- `--global` means this applies to all projects on your computer
- Use the email associated with your GitHub/GitLab account if you have one

### Verify

```bash
git --version
```

**Success looks like:**
```
git version 2.43.0
```
(Any version 2.x.x is fine)

**If you see** `command not found` **or** `git is not recognized`, the installation didn't work. Try the steps again for your platform.

---

## Go Installation (Optional)

**Only required if building Mehrhof from source.** Skip this section if using the install script or pre-built binaries.

Mehrhof requires **Go 1.25 or higher**.

### Debian/Ubuntu

```bash
# Remove old version if installed via apt (usually outdated)
sudo apt remove golang-go -y

# Download and install Go 1.25+
curl -LO https://go.dev/dl/go1.25.6.linux-amd64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.25.6.linux-amd64.tar.gz
rm go1.25.6.linux-amd64.tar.gz

# Add to PATH (add to ~/.bashrc for persistence)
export PATH=$PATH:/usr/local/go/bin
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
```

### Fedora/RHEL

```bash
sudo dnf install golang -y
# Note: Check version - may need manual install if < 1.25
```

### Arch Linux

```bash
sudo pacman -S go
```

### Alpine Linux

```bash
apk add go
```

### macOS

> **Beginners:** If you don't have Homebrew, use Option 2 - download and run the installer.

**Option 1: Homebrew** (if you have Homebrew installed)

```bash
brew install go
```

**Option 2: Official Installer** (recommended for beginners)

1. Go to [go.dev/dl](https://go.dev/dl/)
2. Download the macOS installer (`.pkg` file)
3. Double-click to run and follow the prompts

### Verify

```bash
go version
```

**Success looks like:**
```
go version go1.25.6 linux/amd64
```
(Version must be 1.25 or higher)

**If you see** `command not found`, close and reopen your terminal, then try again. If it still doesn't work, check that Go is in your PATH (see [Troubleshooting](#path-issues)).

---

## Claude CLI Installation

Claude is the recommended (and primary supported) AI agent for Mehrhof.

### Installation

Visit [claude.ai/code](https://claude.ai/code) and follow the installation instructions for your platform.

Typical installation:

```bash
# macOS/Linux - check the official site for current command
curl -fsSL https://claude.ai/install.sh | bash
```

### Authentication

After installation, authenticate with your Anthropic account:

```bash
claude login
```

Follow the prompts to complete authentication. This uses your **Claude subscription** — no separate API key setup required. Once authenticated, Mehrhof will use your CLI directly with no additional configuration.

### Verify

```bash
claude --version
```

**Success looks like:**
```
claude-code 1.2.3
```

**If you see** `command not found`, close and reopen your terminal. If it still doesn't work, revisit the installation steps on [claude.ai/code](https://claude.ai/code).

*Reference: [docs/agents/claude.md](docs/agents/claude.md) for advanced configuration.*

---

## Codex Installation (Optional)

> **⚠️ EXPERIMENTAL WARNING**
>
> Codex support in Mehrhof is **untested and experimental**. The implementation is based on CLI documentation only and has NOT been validated against an actual Codex CLI. File operations and output parsing may not work correctly.
>
> **Claude is strongly recommended for production use.**

If you still want to try Codex:

### Prerequisites

- Node.js 18+ installed
- OpenAI API key

### Installation

```bash
npm install -g @openai/codex
```

### Configuration

Set your OpenAI API key:

```bash
export OPENAI_API_KEY="your-api-key-here"
# Add to ~/.bashrc or ~/.zshrc for persistence
```

### Verify

```bash
codex --version
```

### Test the Integration

Before using with Mehrhof, verify the CLI works:

```bash
codex exec --json "echo test"
```

*Reference: [docs/agents/codex.md](docs/agents/codex.md) for details and known limitations.*

---

## Mehrhof Installation

> **Beginners:** Use Method 1 (Install Script) - just run one command and you're done.

### Method 1: Install Script (Recommended)

The install script automatically:
- Detects your OS and architecture
- Downloads the correct binary
- Verifies SHA256 checksum
- Verifies Minisign signature (if minisign is installed)
- Installs to `~/.local/bin`, `~/bin`, or `/usr/local/bin`

```bash
curl -fsSL https://raw.githubusercontent.com/valksor/go-mehrhof/master/install.sh | bash
```

**Script options:**

```bash
# Install specific version
curl -fsSL https://raw.githubusercontent.com/valksor/go-mehrhof/master/install.sh | bash -s -- -v v1.2.3

# Install nightly build
curl -fsSL https://raw.githubusercontent.com/valksor/go-mehrhof/master/install.sh | bash -s -- --nightly
```

**For security-conscious users:** You can download and inspect the script before running:

```bash
curl -fsSL https://raw.githubusercontent.com/valksor/go-mehrhof/master/install.sh -o install.sh
less install.sh  # Review the script
bash install.sh
```

### Method 2: Pre-built Binaries

Download directly from GitHub releases:

| Platform | Download |
|----------|----------|
| Linux x86_64 | [mehr-linux-amd64](https://github.com/valksor/go-mehrhof/releases/latest/download/mehr-linux-amd64) |
| Linux ARM64 | [mehr-linux-arm64](https://github.com/valksor/go-mehrhof/releases/latest/download/mehr-linux-arm64) |
| macOS Apple Silicon | [mehr-darwin-arm64](https://github.com/valksor/go-mehrhof/releases/latest/download/mehr-darwin-arm64) |
| macOS Intel | [mehr-darwin-amd64](https://github.com/valksor/go-mehrhof/releases/latest/download/mehr-darwin-amd64) |

**Manual installation:**

```bash
# Example for Linux x86_64
curl -L https://github.com/valksor/go-mehrhof/releases/latest/download/mehr-linux-amd64 -o mehr
chmod +x mehr
sudo mv mehr /usr/local/bin/
```

**Verify checksums (recommended):**

```bash
curl -L -O https://github.com/valksor/go-mehrhof/releases/latest/download/mehr-linux-amd64
curl -L -O https://github.com/valksor/go-mehrhof/releases/latest/download/checksums.txt

# Verify checksum
sha256sum -c checksums.txt --ignore-missing
```

**Verify Minisign signature (optional):**

```bash
# Install minisign first (apt install minisign, brew install minisign, etc.)
curl -L -O https://github.com/valksor/go-mehrhof/releases/latest/download/checksums.txt.minisig

minisign -Vm checksums.txt -P "RWTFiZ4b+sgoFLiIMuMrTZr1mmropNlDsnwKl5RfoUtyUWUk4zyVpPw2" \
    -x checksums.txt.minisig
```

### Method 3: Build from Source

Requires [Go 1.25+](#go-installation-optional).

```bash
git clone https://github.com/valksor/go-mehrhof.git
cd go-mehrhof
make install
```

The binary is installed to `$GOPATH/bin` (usually `~/go/bin`).

### Verify Installation

```bash
mehr --version
```

**Success looks like:**
```
mehr version 1.0.0
```

**If you see** `command not found`, see [Troubleshooting - PATH Issues](#path-issues).

---

## Post-Installation Setup

### Initialize a Project

First, navigate to your project folder using the `cd` (change directory) command:

```bash
cd /path/to/your/project
```

**Examples of real paths:**
- `cd ~/projects/my-website` (Linux/Mac - `~` means your home folder)
- `cd ~/Documents/code/my-app` (your Documents folder)
- `cd /home/jane/work/api-server` (full path on Linux)

**Tip:** You can drag a folder from your file manager into the terminal to paste its path.

Once you're in your project folder, initialize Mehrhof:

```bash
mehr init
```

This creates a `.mehrhof/` directory with default configuration.

> **Note:** Folders starting with `.` (like `.mehrhof/`) are **hidden** by default. In your file manager, enable "Show Hidden Files" to see them (usually Ctrl+H on Linux, Cmd+Shift+. on Mac).

### Configure Your Agent

The configuration file is at `.mehrhof/config.yaml`. To edit it:

**Option 1: Using nano (beginner-friendly)**
```bash
nano .mehrhof/config.yaml
```
- Make your changes
- Press **Ctrl+O** then **Enter** to save
- Press **Ctrl+X** to exit

**Option 2: Using a text editor**
- Open VS Code, Sublime Text, or any text editor
- Open the `.mehrhof` folder in your project
- Edit `config.yaml`

The default configuration looks like this:

```yaml
agent:
  default: claude  # or codex (experimental)
```

- Lines starting with `#` are comments (ignored by the program)
- YAML uses indentation (spaces, not tabs) to show structure

### Quick Verification

Run a quick test to verify everything works:

```bash
# Create a simple task
echo "# Test Task\n\nCreate a hello world file." > task.md

# Start the workflow
mehr start file:task.md
mehr status
```

If you see the task loaded and status shows "planning", everything is working!

### Agent Auto-Detection

Mehrhof automatically detects installed agents. Run `mehr agents` to see which agents are available:

```bash
mehr agents
```

---

## Troubleshooting

### PATH Issues

If `mehr` command is not found after installation:

**Bash:**
```bash
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc
```

**Zsh:**
```bash
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.zshrc
source ~/.zshrc
```

### Permission Denied

If you get permission errors during installation:

```bash
# Option 1: Install to user directory
mkdir -p ~/.local/bin
# Then re-run the install script

# Option 2: Use sudo for /usr/local/bin
sudo mv mehr /usr/local/bin/
```

### Agent Not Found

If Mehrhof can't find your agent:

1. Verify the agent CLI is installed: `claude --version` or `codex --version`
2. Ensure the agent is in your PATH
3. Check your `.mehrhof/config.yaml` agent configuration

### WSL-Specific Issues

**"Cannot find WSL" after installation:**
- Restart your computer
- Run `wsl --update` in PowerShell as Administrator

**Slow file access:**
- Store your projects inside WSL (e.g., `~/projects/`) not in `/mnt/c/`
- Windows filesystem access through WSL is slower

**Network issues:**
- Check Windows Firewall isn't blocking WSL
- Try `wsl --shutdown` then relaunch Ubuntu

### Alpine Linux / Container Issues

The pre-built binaries are compiled with `CGO_ENABLED=0` (pure Go, no C dependencies), so they should work on Alpine's musl libc without issues.

If you encounter problems:

```bash
# Verify the binary runs
./mehr --version

# Check for missing dependencies (should be none for static binary)
ldd ./mehr
# Expected: "not a dynamic executable" or "statically linked"
```

### Container Deployment

For Docker/containerized environments:

```dockerfile
# Example Dockerfile snippet
FROM alpine:latest

RUN apk add --no-cache git curl

# Install mehrhof
RUN curl -fsSL https://raw.githubusercontent.com/valksor/go-mehrhof/master/install.sh | bash

# Install Claude CLI (check official docs for container-friendly method)
# ...
```

---

## See Also

- [README.md](README.md) - Project overview and quick reference
- [docs/quickstart.md](docs/quickstart.md) - Getting started guide
- [docs/agents/](docs/agents/) - Agent configuration details
- [docs/configuration/](docs/configuration/) - Full configuration reference
