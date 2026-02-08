# Troubleshooting: Windows / WSL2

Solutions for common issues when running Mehrhof inside WSL2 on Windows.

## Quick Fixes

| Problem                          | Quick Fix                                        |
|----------------------------------|--------------------------------------------------|
| Browser won't open from `--open` | Copy URL from terminal, paste in Windows browser |
| Slow git/builds                  | Move project to `~/projects/`, not `/mnt/c/`     |
| `localhost` doesn't reach WSL    | Run `wsl --update` for mirrored networking       |
| Docker not found                 | Enable Docker Desktop WSL2 integration           |
| Claude CLI not found             | Install Claude inside WSL, not Windows           |
| Permission denied on files       | Check `umask` or remount with metadata           |
| Too many open files              | Increase inotify watches (see below)             |

---

## Browser Won't Open

When you run `mehr serve --open`, the `--open` flag uses `xdg-open` inside WSL to launch a browser. This may not work because WSL doesn't have a native browser.

**Fixes:**

1. **Copy the URL manually** — the terminal output shows the address (e.g., `http://localhost:8080`). Paste it into your Windows browser.

2. **Use `explorer.exe`** — Windows executables can be called from WSL:
   ```bash
   explorer.exe http://localhost:8080
   ```
   This opens the URL in your default Windows browser.

3. **Set `BROWSER` environment variable** — point WSL to a Windows browser:
   ```bash
   # Add to ~/.bashrc
   export BROWSER="explorer.exe"
   ```
   After this, `--open` may work automatically.

---

## Slow Git Operations and Builds

**Symptom:** `mehr plan`, `mehr implement`, `git status`, or `go build` are noticeably slow.

**Cause:** Your project is on the mounted Windows filesystem (`/mnt/c/`). File operations through the translation layer are 5-50x slower than on the native Linux filesystem.

**Fix:** Move your project to the Linux filesystem:

```bash
mkdir -p ~/projects
cd ~/projects
git clone <your-repo-url>
cd your-project
mehr init
```

**How to tell where your project is:**

```bash
pwd
# /mnt/c/Users/... = Windows filesystem (slow)
# /home/yourname/... = Linux filesystem (fast)
```

Your Linux files are accessible from Windows Explorer at `\\wsl$\Ubuntu\home\yourname\`.

---

## Cannot Connect to `mehr serve`

**Symptom:** `mehr serve` starts successfully inside WSL but `localhost:8080` doesn't load in your Windows browser.

### Modern Windows (11 22H2+, WSL 2.0+)

Ports should auto-forward. If they don't:

1. **Update WSL:**
   ```bash
   wsl --update
   ```

2. **Check WSL networking mode** — verify mirrored mode is active:
   ```bash
   # In PowerShell
   wsl --version
   # Look for WSL version 2.0.0 or higher
   ```

3. **Enable mirrored networking** if not active — create or edit `%USERPROFILE%\.wslconfig`:
   ```ini
   [wsl2]
   networkingMode=mirrored
   ```
   Then restart WSL: `wsl --shutdown` in PowerShell.

### Older Windows / NAT Networking

If mirrored mode isn't available:

1. **Find the WSL IP:**
   ```bash
   hostname -I
   ```
   Use the IP shown (e.g., `http://172.28.160.1:8080`) instead of `localhost`.

2. **Or bind to all interfaces:**
   ```bash
   mehr serve --host 0.0.0.0
   ```

### Windows Firewall

If neither works, Windows Firewall may be blocking WSL:

1. Open **Windows Defender Firewall > Advanced Settings**
2. Check **Inbound Rules** for rules blocking the port
3. Create an allow rule for the port (e.g., 8080) if needed

---

## Docker Not Found

**Symptom:** Docker-dependent features fail, or `docker` command not found inside WSL.

**Fix:**

1. Install [Docker Desktop](https://docs.docker.com/desktop/install/windows-install/) on Windows
2. Open Docker Desktop **Settings > Resources > WSL Integration**
3. Toggle on your WSL distro (e.g., Ubuntu)
4. Restart your WSL terminal
5. Verify:
   ```bash
   docker --version
   ```

**Do not install Docker inside WSL separately** — Docker Desktop's WSL2 backend is the supported approach on Windows.

---

## Claude CLI Not Found

**Symptom:** `mehr plan` fails with "claude CLI not found" or similar.

**Cause:** Claude CLI is installed on Windows but not inside WSL, or vice versa.

**Fix:** Install Claude inside your WSL environment:

```bash
# Inside WSL
claude --version  # Check if installed

# If not, follow the Claude CLI install instructions for Linux
# Visit https://claude.ai/code
```

The Windows installation of Claude (if any) is separate from WSL. Each environment has its own PATH and installed binaries.

---

## Permission Denied on Files

**Symptom:** Operations fail with "permission denied" on project files.

### Files on Linux Filesystem

Check ownership and permissions:

```bash
ls -la problematic-file
# Fix ownership
sudo chown -R $(whoami) ~/projects/your-project
```

### Files on Windows Filesystem (`/mnt/c/`)

Windows drives are mounted without Unix metadata by default. All files appear with the same permissions.

**Fix:** Enable metadata mount option. Edit `/etc/wsl.conf`:

```ini
[automount]
options = "metadata,umask=022,fmask=011"
```

Restart WSL (`wsl --shutdown` in PowerShell) for changes to take effect.

**Better fix:** Move your project to the Linux filesystem (see "Slow Git Operations" above).

---

## Too Many Open Files / Inotify Limit

**Symptom:** `mehr serve` or file watchers fail with "too many open files" or "inotify watch limit reached."

**Fix:** Increase the inotify watcher limit:

```bash
# Temporary (resets on WSL restart)
sudo sysctl fs.inotify.max_user_watches=524288

# Permanent
echo "fs.inotify.max_user_watches=524288" | sudo tee -a /etc/sysctl.conf
sudo sysctl -p
```

---

## PATH Pollution from Windows

**Symptom:** WSL finds wrong versions of tools (e.g., Windows `git.exe` instead of Linux `git`), or PATH is extremely long.

**Cause:** By default, WSL appends the Windows PATH to the Linux PATH. This means Windows executables like `git.exe`, `node.exe`, etc. are available inside WSL, which can cause confusion.

**Fix:** Disable Windows PATH inheritance if it's causing problems. Edit `/etc/wsl.conf`:

```ini
[interop]
appendWindowsPath = false
```

Restart WSL (`wsl --shutdown` in PowerShell).

**Note:** This also removes access to `explorer.exe`, `clip.exe`, and other Windows utilities from WSL. Only disable if the PATH pollution is causing real issues.

---

## WSL Hangs or Runs Out of Memory

**Symptom:** WSL becomes unresponsive during large builds or agent operations.

**Fix:** Limit WSL's memory usage. Edit `%USERPROFILE%\.wslconfig` on the Windows side:

```ini
[wsl2]
memory=8GB
processors=4
```

Restart WSL: `wsl --shutdown` in PowerShell.

Default: WSL uses up to 50% of system RAM (capped at 8GB on older builds, higher on newer). Adjust based on your system.

---

## See Also

- [Windows & WSL2 Guide](/guides/windows-wsl.md) — Setup and best practices
- [Common Issues](/troubleshooting/index.md) — General troubleshooting (not WSL-specific)
- [Installation Guide](/INSTALL.md#windows-wsl-setup) — Initial WSL2 setup
