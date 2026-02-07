# Frequently Asked Questions

## General

### What is Mehrhof?

Mehrhof is a structured creation environment that wraps your existing AI agent CLI (Claude, Codex, or custom) and adds a repeatable workflow: plan, create, review, finish. It gives you human checkpoints at every step — nothing ships without your approval.

### Is Mehrhof free?

Yes. Mehrhof is BSD-3 licensed, completely free, and requires no Mehrhof account or usage fees. You only need your existing agent CLI (e.g., Claude) — Mehrhof adds zero extra cost on top of what you already pay for your AI provider.

### What does "structured creation environment" mean?

Instead of giving an AI agent an open-ended prompt and hoping for the best, Mehrhof breaks the work into clear phases — planning, implementation, review — with human approval gates between each step. You stay in control of what gets built and what gets shipped.

---

## Cost and Privacy

### Does Mehrhof send my code anywhere?

Mehrhof itself does not transmit your code. It runs locally and calls your locally-installed agent CLI (e.g., Claude CLI). Your code goes wherever your agent CLI sends it — typically to the AI provider's API. Mehrhof adds no additional data transmission.

### What data does Mehrhof store?

Mehrhof stores project-level data in `.mehrhof/` within your project directory (configuration, specifications), and workspace data in `~/.valksor/mehrhof/` (session logs, task history, workspaces). All data stays on your machine.

### Does Mehrhof require an internet connection?

Only for the AI agent calls. Mehrhof itself works offline for task management, configuration, undo/redo, and other local operations. The agent CLI needs internet access to reach the AI provider's API.

---

## Prerequisites

### What agent CLI do I need?

Mehrhof works with any supported agent CLI installed on your machine. **Claude** (from Anthropic) is the primary and best-tested agent. **Codex** (from OpenAI) is also supported but less thoroughly tested. You can also write custom agent plugins.

### Can I use a different AI model?

Yes. You can configure agent aliases to use different models for different workflow steps. For example, use Claude Opus for planning and Claude Sonnet for implementation:

```yaml
# .mehrhof/config.yaml
agents:
  opus:
    extends: claude
    args: ["--model", "claude-opus-4"]
  sonnet:
    extends: claude
    args: ["--model", "claude-sonnet-4"]

agent:
  steps:
    planning: { name: opus }
    implementing: { name: sonnet }
```

### What if I don't have Claude installed yet?

Install the Claude CLI first: visit [claude.ai/code](https://claude.ai/code) for setup instructions. Then install Mehrhof and run `mehr init` to configure your workspace.

---

## Compatibility

### What programming languages does Mehrhof support?

Mehrhof is language-agnostic. It works with any codebase — Go, Python, JavaScript, TypeScript, PHP, Ruby, Rust, Java, or any other language. The AI agent handles the language-specific work; Mehrhof handles the workflow.

### Does it work on Windows?

Yes, via [WSL2](https://learn.microsoft.com/en-us/windows/wsl/) (Windows Subsystem for Linux). Mehrhof runs as a native Linux binary inside WSL2, giving you full feature parity with Linux and macOS. Performance inside WSL2 is near-native — the key is storing projects on the Linux filesystem (`~/projects/`) rather than the mounted Windows drive.

Both VS Code (via the WSL extension) and JetBrains IDEs (2025.1+) integrate seamlessly with WSL, so your IDE runs on Windows while Mehrhof operates inside Linux.

See the [Windows & WSL2 Guide](/guides/windows-wsl.md) for setup instructions, best practices, and IDE integration.

### Can I use it on an existing project?

Yes. Navigate to your project directory and run `mehr init`. This creates a `.mehrhof/` configuration directory. Your existing code is not modified. Mehrhof works with any Git repository.

### Does it work with monorepos?

Yes. Mehrhof operates within whichever directory you run it from. For monorepos, navigate to the relevant subdirectory or use the full repo root — the AI agent will scope its work to the task description you provide.

---

## Comparison

### How is this different from using Claude Code directly?

Claude Code is an AI coding assistant — you give it prompts and it writes code. Mehrhof wraps Claude Code (or other agents) and adds:

- **Structured workflow** — Enforced plan → implement → review → finish phases
- **Human checkpoints** — Nothing ships without explicit approval at each step
- **Task source integration** — Pull tasks from GitHub Issues, Jira, Linear, and 13+ other tools
- **Undo/redo** — Git checkpoints at every step with full rollback
- **Web UI** — Non-technical users can create and manage tasks in a browser
- **Budget controls** — Set cost and token limits per task
- **Audit trail** — Full history of what was planned, implemented, and reviewed

### How is this different from Cursor or Aider?

Similar to the Claude Code comparison: Cursor and Aider are AI coding tools. Mehrhof is a workflow layer that can wrap any agent CLI. It adds structure, safety, repeatability, and team features (Web UI, task sources, budgets) on top of whatever AI tool you already use.

### Why would I add this layer instead of just using the AI tool directly?

You might not need to — for quick, personal coding tasks, using Claude Code or Cursor directly is perfectly fine. Mehrhof is valuable when:

- Multiple people (including non-technical) need to create changes
- Human review is mandatory before anything ships
- Tasks come from project management tools, not ad-hoc prompts
- You want audit trails and repeatable processes
- You need budget controls on AI spending

---

## Troubleshooting

For common errors and their solutions, see [Troubleshooting](troubleshooting/index.md).

For bugs or feature requests, visit [GitHub Issues](https://github.com/valksor/go-mehrhof/issues).
