# Agent Sandboxing

Isolate AI agent execution using platform-specific sandboxing for enhanced security.

## Overview

Mehrhof supports sandboxing AI agents to prevent unauthorized access to your system while still allowing them to perform useful work.

**Supported Platforms:**
- **Linux** - User namespaces (unprivileged chroot via `unshare` + `pivot_root`)
- **macOS** - `sandbox-exec` with SBPL profiles

## What the Sandbox Allows

The sandbox is designed to be permissive enough for agents to do real work while restricting access to sensitive areas:

| Resource | Access | Notes |
|----------|--------|-------|
| Project directory | ✅ Read/Write | Primary workspace |
| `/tmp` | ✅ Read/Write | tmpfs mount |
| `$HOME/.claude` | ✅ Read/Write | For Claude Code plans |
| `/dev/null`, `/dev/zero`, `/dev/random`, `/dev/urandom` | ✅ Read | Device access |
| `/proc` | ✅ Read | Process information |
| Network | ✅ Full access | Required for LLM APIs |
| System binaries | ✅ Execute | git, node, python, go, etc. |
| Shared libraries | ✅ Read | `.so`/`.dylib` files |

**Explicitly Denied:**
- Access to other directories in `$HOME`
- Access to `/etc`, `/var`, `/sys` (except /proc)
- Access to other users' files
- Access to system configuration

## Why Keep Network Access?

Unlike traditional sandboxes, Mehrhof's sandbox **keeps network access enabled** because:

1. **LLM APIs require HTTPS** - Agents must communicate with Anthropic, OpenAI, etc.
2. **Package downloads** - `go mod download`, `npm install`, `pip install` need network
3. **Git operations** - Clone, fetch, push require network
4. **DNS resolution** - Required for most network operations

The isolation comes from **filesystem restrictions**, not network isolation.

## Configuration

Enable sandboxing in `.mehrhof/config.yaml`:

```yaml
sandbox:
  enabled: true      # Enable sandboxing
  network: true       # Allow network access (default: true)
  tmp_dir: ""         # Custom tmpdir path (default: auto)
  tools: []           # Additional binary paths to allow
```

### CLI Flag

Enable sandbox per-command:

```bash
mehr --sandbox start task.md
```

### Agent-Specific Flags

Different agents may add sandbox-specific flags:

| Agent | Flag | Purpose |
|-------|------|---------|
| Codex | `--yolo` | Skip confirmations (can't answer in sandbox) |

## Platform Details

### Linux (User Namespaces)

Linux sandboxes use **unprivileged user namespaces** - no root required!

**How it works:**
```
┌─────────────────────────────────────────────────────┐
│                    Host System                      │
├─────────────────────────────────────────────────────┤
│  unshare --user --mount --pid --fork               │
│  ┌─────────────────────────────────────────────┐   │
│  │          User Namespace (mapped to root)    │   │
│  │  ┌───────────────────────────────────────┐  │   │
│  │  │       New Root (pivot_root)          │  │   │
│  │  │  /tmp        (tmpfs)                 │  │   │
│  │  │  /workspace   (bind mount → project)  │  │   │
│  │  │  /dev/*      (device nodes)          │  │   │
│  │  │  /proc       (procfs)                │  │   │
│  │  │  /bin, /usr/bin (bind mounts)        │  │   │
│  │  │  /lib, /usr/lib (bind mounts)        │  │   │
│  │  └───────────────────────────────────────┘  │   │
│  └─────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────┘
```

**Key points:**
- `CLONE_NEWUSER` creates new user namespace (unprivileged!)
- `--map-root-user` maps current user to root inside namespace
- `pivot_root` changes the root filesystem
- **NO `CLONE_NEWNET`** - network namespace is shared with host

### macOS (sandbox-exec)

macOS uses Apple's built-in `sandbox-exec` with SBPL profiles.

**How it works:**
```bash
sandbox-exec -p <profile> /path/to/agent
```

**Generated profile allows:**
- File read/write to project directory, `/tmp`, `$HOME/.claude`
- Network access (outbound TCP/UDP, DNS)
- Process execution for allowed binaries
- Device file access (`/dev/null`, `/dev/urandom`, etc.)

## Web UI

The Web UI provides sandbox controls:

### Settings Page

Enable/disable sandboxing via the settings interface at `/settings`:

```
┌─────────────────────────────────────────────┐
│  Sandbox Configuration                      │
├─────────────────────────────────────────────┤
│  Enable Sandbox      [✓]                    │
│  Network Access      [✓]                    │
│                                             │
│  Platform: linux                            │
│  Status: Supported                          │
└─────────────────────────────────────────────┘
```

### Dashboard Integration

- Sandbox toggle appears in workflow actions
- "Sandboxed" badge shows on active tasks
- Real-time status updates via SSE

## API Endpoints

```bash
# Get sandbox status
GET /api/v1/sandbox/status

# Enable sandbox
POST /api/v1/sandbox/enable

# Disable sandbox
POST /api/v1/sandbox/disable
```

**Response format:**
```json
{
  "enabled": true,
  "platform": "linux",
  "supported": true,
  "active": false,
  "network": true,
  "profile": ""
}
```

## Limitations

### Linux

- Requires kernel support for user namespaces (kernel 3.8+)
- May not work in some containerized environments (Docker, LXC)
- `pivot_root` requires CAP_SYS_ADMIN in the parent namespace (but user namespaces handle this)

### macOS

- `sandbox-exec` is deprecated by Apple but still works
- Some operations may be restricted more than on Linux
- Profile generation is more restrictive than Linux approach

## Troubleshooting

### "operation not permitted" on Linux

This occurs in containerized environments where user namespaces are restricted. **Solutions:**

1. **Run outside container** - Use bare metal or VM
2. **Enable user namespaces** in container runtime
3. **Disable sandbox** - Not recommended for untrusted agents

### Agent can't access required files

**Symptoms:** Agent reports file not found

**Solutions:**
1. Check file is in project directory (workspace)
2. For Claude Code plans, verify `$HOME/.claude` is mounted
3. Add custom tool paths to `sandbox.tools` config

### Network not working inside sandbox

**Symptoms:** Agent can't reach LLM APIs

**Solutions:**
1. Verify `sandbox.network: true` in config
2. Check DNS resolution: `meer --sandbox run "ping -c 1 api.anthropic.com"`
3. Firewall rules may affect network namespace

## Security Considerations

### What Sandbox Protects Against

- ✅ Accidental file access outside project
- ✅ Agents reading sensitive files (`~/.ssh`, `~/.gnupg`)
- ✅ Agents modifying system configuration
- ✅ Credential theft from other users

### What Sandbox Does NOT Protect Against

- ❌ Malicious agent with network access (can exfiltrate data)
- ❌ Exploits in LLM API endpoints
- ❌ Side-channel attacks (timing, cache)
- ❌ Compromised shared libraries

### Best Practices

1. **Review agent output** - Check what files agents access
2. **Use separate workspaces** - Isolate different projects
3. **Rotate API keys** - If agent is compromised, rotate credentials
4. **Monitor network traffic** - Check for suspicious connections
5. **Keep dependencies updated** - Patch vulnerabilities in shared libraries

## See Also

- [Security & Compliance Scanning](security.md) - SAST, secrets, vulnerability scanning
- [Configuration Guide](../configuration/index.md) - Workspace configuration
- [CLI Reference](../cli/index.md) - Command-line options
