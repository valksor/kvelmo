# Installation Guide

> **Last verified:** February 2026

Complete guide to install Mehrhof and all prerequisites from scratch.

---

## Quick Start (Experienced Users)

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
| macOS | Intel (amd64) | Full |
| macOS | Apple Silicon (arm64) | Full |
| Windows | via WSL2 | Full (run Linux binary in WSL) |

**Time estimates:**
- Fresh Windows system (needs WSL): ~30-45 minutes
- Fresh Linux/macOS (needs Git + Claude): ~15-20 minutes
- Has prerequisites: ~2 minutes

---

## Windows (WSL Setup)

Windows users must use WSL2 (Windows Subsystem for Linux). Native Windows is not supported.

### Step 1: Enable WSL2

Open **PowerShell as Administrator** and run:

```powershell
wsl --install
```

This installs WSL2 with Ubuntu by default. Restart your computer when prompted.

### Step 2: First Launch

After restart, Ubuntu will launch automatically. If not, search for "Ubuntu" in the Start menu.

Create your Linux username and password when prompted. This is separate from your Windows login.

### Step 3: Update Ubuntu

```bash
sudo apt update && sudo apt upgrade -y
```

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

**Option 1: Xcode Command Line Tools** (easiest)

```bash
xcode-select --install
```

A popup will appear. Click "Install" and wait for completion.

**Option 2: Homebrew**

```bash
brew install git
```

### Configure Git

After installation, set your identity:

```bash
git config --global user.name "Your Name"
git config --global user.email "your.email@example.com"
```

### Verify

```bash
git --version
# Expected: git version 2.x.x
```

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

**Option 1: Homebrew** (recommended)

```bash
brew install go
```

**Option 2: Official Installer**

Download from [go.dev/dl](https://go.dev/dl/) and run the installer.

### Verify

```bash
go version
# Expected: go version go1.25.x ...
```

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

Follow the prompts to complete authentication.

### Verify

```bash
claude --version
# Expected: claude-code x.x.x
```

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
| macOS Intel | [mehr-darwin-amd64](https://github.com/valksor/go-mehrhof/releases/latest/download/mehr-darwin-amd64) |
| macOS Apple Silicon | [mehr-darwin-arm64](https://github.com/valksor/go-mehrhof/releases/latest/download/mehr-darwin-arm64) |

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
# Expected: mehr version x.x.x
```

---

## Post-Installation Setup

### Initialize a Project

Navigate to your project directory and run:

```bash
cd /path/to/your/project
mehr init
```

This creates a `.mehrhof/` directory with default configuration.

### Configure Your Agent

Edit `.mehrhof/config.yaml`:

```yaml
agent:
  default: claude  # or codex (experimental)
```

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
