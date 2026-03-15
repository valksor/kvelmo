# Pragmatic Developer Gap Analysis

Imagine — you are an experienced developer who's been shipping software for years. You're equally comfortable in the terminal and the browser, switching between CLI and web UI without thinking. You have:

- **No patience for ceremony** — if a tool makes you click through 5 dialogs to start work, you'll close it and use the terminal
- **Multiple projects in flight** — you context-switch daily and need to know instantly where each project stands
- **Strong opinions, loosely held** — you'll skip planning when the task is obvious, but want it available when it's not
- **CLI and web interchangeably** — whichever is faster in the moment wins; you expect them to show the same state
- **Speed over polish** — you'd rather ship a rough PR fast and iterate than perfect it before submitting
- **Zero tolerance for friction** — extra prompts, confirmation dialogs, and mandatory steps that add no value make you abandon tools

Now you find kvelmo, a tool that orchestrates your development workflow — task loading, planning, implementation, review, and PR submission.

You are excited. You want to use it. **Can you?**

Critically — can you use kvelmo to achieve these goals:

---

## Phase 1: Core Goals (6)

For each goal, assess:
- **Status**: fully / partially / not at all
- **What exists**: current kvelmo features that help
- **Gap**: what's missing
- **Recommendation**: what to build (Fibonacci effort: 1, 2, 3, 5, 8, 13)

### Goal 1: Zero-friction task start
Load a task from a GitHub issue URL, a file path, or inline text in under 5 seconds. One command, no wizards, no follow-up prompts.

### Goal 2: Switch freely between CLI and web
Start a task in the terminal, check progress in the browser, intervene from either. Same state, no drift, no sync issues.

### Goal 3: Skip optional steps
Jump straight to implement when planning is unnecessary. Skip simplify and optimize when the code is already clean. The workflow should be flexible, not a forced march.

### Goal 4: Instant status at a glance
One command (`kvelmo status`) or one screen (web dashboard) tells you exactly where every active task stands — state, progress, blockers. No digging.

### Goal 5: Fast undo without thinking
Undo the last action immediately. No "are you sure?" dialogs. No selecting from a list of checkpoints. Just undo. If you want more control, checkpoints are there, but the default is fast.

### Goal 6: Ship PR in one command
`kvelmo submit` generates the PR description from task context, fills in what was planned vs. implemented, and creates the PR. Zero manual editing needed for routine PRs.

---

## Phase 2: Extended Goals (8)

### Goal 7: Batch operations
Act on multiple tasks at once — submit all reviewed tasks, pause everything, reset failed tasks. Bulk actions for bulk workflows.

### Goal 8: Keyboard shortcuts in web UI
Navigate the web dashboard without touching the mouse. Vim-style or customizable shortcuts for common actions — next task, approve, submit, undo.

### Goal 9: Customizable defaults
Set preferred agent, default skip steps, auto-approve patterns. Per-project or global. `kvelmo config set default-agent claude` and never think about it again.

### Goal 10: Quick context dump
Export the current task state — plan, changes, chat history — as a shareable artifact. Useful for debugging, handoffs, or "what did I do yesterday?"

### Goal 11: Aliases and shortcuts
`kvelmo i` = `kvelmo implement`. `kvelmo s` = `kvelmo status`. Custom user-defined aliases that match your muscle memory.

### Goal 12: Notification preferences
Only alert on failures and blockers, not routine progress. Configurable per-project. Silent mode for when you're in flow.

### Goal 13: Template tasks
Reuse common task shapes — "bug fix", "feature", "refactor" — with pre-filled fields and default settings. Skip repetitive setup.

### Goal 14: History search
Find past tasks by keyword, file touched, or date range. Not by ID. "What did I work on in the auth module last week?"

---

## Phase 2: Critical Audit

The 14 goals above are a starting point, not a ceiling. Investigate deeper across these dimensions:

1. **Real-world friction**: Where does kvelmo add steps that a pragmatic dev would skip? What makes power users abandon the tool?
2. **Missing primitives**: What basic operations require multiple commands when one would do?
3. **Error & recovery gaps**: When something breaks mid-workflow, can you recover without losing progress?
4. **Scalability cliffs**: At what point does managing many concurrent tasks become unwieldy?
5. **Observability blindspots**: Can you tell what kvelmo is doing without reading logs?
6. **Workflow completeness**: Are there "last mile" gaps between kvelmo and actually shipping code?
7. **Integration gaps**: Does kvelmo work with the tools pragmatic devs already use (editors, terminals, browsers)?
8. **Data ownership & portability**: Can you get your task history out if you stop using kvelmo?

Report all gaps found—whether 3 or 30. Each gap should include severity and a recommended fix.

---

## Sibling Commands

This command is part of a family of 10 persona-specific gap analyses:

- `/solo-developer-gaps`
- `/team-lead-gaps`
- `/opensource-maintainer-gaps`
- `/devops-gaps`
- `/cli-poweruser-gaps`
- `/frontend-dev-gaps`
- `/agent-dev-gaps`
- `/enterprise-gaps`
- `/pragmatic-dev-gaps` (this command)
- `/regulated-dev-gaps`

---

## Shared Infrastructure

All personas share these 10 core systems. When recommending features, these must not be removed, narrowed, or domain-specific without cross-persona review:

1. **Socket layer** (`pkg/socket/` — global + worktree sockets)
2. **State machine** (`pkg/conductor/` — lifecycle states, guards, transitions)
3. **Agent interface** (`pkg/agent/` — WebSocket + CLI modes)
4. **Worker pool** (`pkg/worker/` — job queue, event streaming)
5. **Providers** (`pkg/provider/` — GitHub, GitLab, Wrike, Linear, file)
6. **Storage** (`pkg/storage/` — tasks, chats, plans, reviews)
7. **Git integration** (`pkg/git/` — checkpoints, undo/redo)
8. **Web UI** (`web/` — React 19, stores, real-time updates)
9. **CLI commands** (`cmd/kvelmo/commands/`)
10. **Memory system** (`pkg/memory/` — embeddings, semantic search)

### Critical Rules

1. **Extend, don't replace**: New features add alongside existing ones. Never remove capabilities.
2. **Domain-agnostic core**: Features benefiting one persona should work for all unless explicitly domain-specific.
3. **Check the siblings**: Before recommending a feature, verify it doesn't break other personas' workflows.
4. **Full-stack implementation** — every recommended feature MUST be wired end-to-end. For each new feature, specify:
   - **Go package** (`pkg/<feature>/`) + handler wiring
   - **Socket RPC method** registered in socket server
   - **Web UI store** update in `web/src/stores/`
   - **Web UI component** (widget, panel, or page)
   - **Route + navigation** wiring in web frontend
   - **CLI command** in `cmd/kvelmo/commands/` (if user-facing)
   - A feature without both CLI and web UI is not complete (per CLAUDE.md parity rule). If a feature is backend-only by nature, explicitly note why.
5. **Name by function, not domain** — packages, RPC methods, CLI commands, and frontend components must be named for what they DO, not which persona inspired them. Litmus test: "Would a user from a DIFFERENT persona find this name sensible?" Domain-specific terminology belongs in help text and documentation, NOT in code identifiers.
