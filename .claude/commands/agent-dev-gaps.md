# Agent Developer Gap Analysis

Imagine — you are an agent developer building custom AI agents or extending kvelmo's agent system. You want to create specialized agents for your domain. You have:

- **Almost-right built-ins** — the built-in agents are good but not quite right for your use case
- **Domain-specific needs** — agents that understand your company's specific patterns, tools, and constraints
- **Black-box debugging** — agent behavior is opaque; when something goes wrong, you can't trace why
- **No production path** — no clear journey from "working prototype" to "production agent"
- **Permission friction** — systems are either too restrictive (blocking valid operations) or too permissive (allowing dangerous ones)
- **Expensive testing** — testing agents requires running real API calls; no mock/replay infrastructure
- **Manual state management** — agent state persistence and resumption is error-prone and hand-rolled

Now you find kvelmo, a tool with a well-defined Agent interface, WebSocket streaming, worker pools, and a plugin architecture — designed for exactly this kind of extensibility.

You are excited. You want to use it. **Can you?**

Critically — can you use kvelmo to achieve these goals:

---

## Phase 1: Core Goals (6)

For each goal, assess:
- **Status**: fully / partially / not at all
- **What exists**: current kvelmo features that help
- **Gap**: what's missing
- **Recommendation**: what to build (Fibonacci effort: 1, 2, 3, 5, 8, 13)

### Goal 1: Clear agent interface
Well-documented Agent interface. Know exactly what to implement and how.

### Goal 2: Event streaming
Emit and handle events correctly. Token streaming, tool calls, permissions, completion.

### Goal 3: Permission handling
Flexible permission system. Auto-approve safe operations, prompt for risky ones, custom policies.

### Goal 4: Testing harness
Test agents without API calls. Mock providers, replay conversations, snapshot testing.

### Goal 5: Debugging tools
Inspect agent state, trace tool calls, understand decision paths. Agent observability.

### Goal 6: Provider integration
Add new providers (local LLMs, different APIs). Clear extension points.

---

## Phase 2: Extended Goals (8)

### Goal 7: Agent composition
Combine agents—supervisor agents, agent pipelines, agent delegation.

### Goal 8: State persistence
Save and restore agent state. Resume long-running operations.

### Goal 9: Tool development
Create custom tools for agents. Tool discovery, parameter validation, documentation.

### Goal 10: Prompt management
Version prompts, A/B test them, track effectiveness. Prompt engineering infrastructure.

### Goal 11: Cost tracking
Track API costs per agent, per task, per user. Budget limits and alerts.

### Goal 12: Performance profiling
Measure agent latency, token usage, tool call frequency. Optimization guidance.

### Goal 13: Agent registry
Publish and discover community agents. Versioning, dependencies, compatibility.

### Goal 14: Sandboxing
Run agents with limited permissions. Containment for untrusted agents.

---

## Phase 2: Critical Audit

The 14 goals above are a starting point, not a ceiling. Investigate deeper across these dimensions:

1. **Real-world friction**: What makes agent developers abandon frameworks? Where does kvelmo create friction?
2. **Missing primitives**: What agent operations are awkward or impossible?
3. **Error & recovery gaps**: When agents fail, is debugging tractable?
4. **Scalability cliffs**: At what complexity does agent development become unmanageable?
5. **Observability blindspots**: Can developers understand what their agents are doing?
6. **Workflow completeness**: Is there a clear path from prototype to production?
7. **Integration gaps**: What agent infrastructure does kvelmo need to connect to?
8. **Data ownership & portability**: Can agent code be used outside kvelmo?

Report all gaps found—whether 3 or 30. Each gap should include severity and a recommended fix.

---

## Sibling Commands

This command is part of a family of 8 persona-specific gap analyses:

- `/solo-developer-gaps`
- `/team-lead-gaps`
- `/opensource-maintainer-gaps`
- `/devops-gaps`
- `/cli-poweruser-gaps`
- `/frontend-dev-gaps`
- `/agent-dev-gaps` (this command)
- `/enterprise-gaps`

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
