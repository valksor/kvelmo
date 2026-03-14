# Team Lead Gap Analysis

Imagine — you are a team lead managing 4-8 engineers on a product team. You've adopted AI-assisted development and now face new coordination challenges. You have:

- **Inconsistent outputs** — engineers using different AI tools with no standard approach
- **No visibility** — what AI agents are actually doing across the team is a black box
- **Review bottlenecks** — AI-generated code needs extra scrutiny, slowing your pipeline
- **Sprint chaos** — planning doesn't account for AI's impact on velocity
- **Onboarding friction** — new engineers take longer than expected to learn AI-assisted workflows
- **Knowledge silos** — which prompts work best for your codebase lives in individual heads
- **Quality uncertainty** — hard to tell if AI is helping or creating tech debt faster

Now you find kvelmo, a tool that promises to orchestrate AI-assisted development with full lifecycle visibility, consistent workflows, and checkpoint-based safety.

You are excited. You want to use it. **Can you?**

Critically — can you use kvelmo to achieve these goals:

---

## Phase 1: Core Goals (6)

For each goal, assess:
- **Status**: fully / partially / not at all
- **What exists**: current kvelmo features that help
- **Gap**: what's missing
- **Recommendation**: what to build (Fibonacci effort: 1, 2, 3, 5, 8, 13)

### Goal 1: Team-wide visibility
See all active kvelmo tasks across the team. Know who's working on what, what state they're in, what agents are running.

### Goal 2: Consistent workflows
Enforce team standards—same planning depth, same review requirements, same commit practices. Configurable guardrails.

### Goal 3: Code review integration
Surface AI-generated changes with appropriate context. Help reviewers understand what the AI did and why.

### Goal 4: Progress tracking
Track task completion rates, agent success rates, and common failure modes. Data for retrospectives.

### Goal 5: Knowledge sharing
Share effective prompts, task templates, and patterns across the team. Collective intelligence.

### Goal 6: Risk visibility
Flag high-risk AI operations before they happen. Security-sensitive code, critical paths, production systems.

---

## Phase 2: Extended Goals (8)

### Goal 7: Onboarding workflows
Guided paths for new team members learning kvelmo. Progressive disclosure of capabilities.

### Goal 8: Workload distribution
See which engineers are overloaded with active tasks. Balance AI-assisted work.

### Goal 9: Quality metrics
Track code quality trends—are AI-assisted PRs getting approved faster? Creating more bugs? Better test coverage?

### Goal 10: Audit trail
Complete history of who ran what agents with what prompts. Compliance and debugging.

### Goal 11: Permission management
Control who can run which agents, access which providers, modify which configurations.

### Goal 12: Integration with project management
Sync with Jira, Linear, Asana. Tasks flow in, status flows out.

### Goal 13: Team templates
Shared task templates, prompt libraries, and workflow presets. Standardization without rigidity.

### Goal 14: Cross-project coordination
When the team works on multiple repos, maintain coherent views and shared context.

---

## Phase 2: Critical Audit

The 14 goals above are a starting point, not a ceiling. Investigate deeper across these dimensions:

1. **Real-world friction**: What makes team leads abandon tools? Where does kvelmo create management overhead?
2. **Missing primitives**: What coordination operations are awkward or impossible?
3. **Error & recovery gaps**: When one engineer's kvelmo fails, how does it affect the team?
4. **Scalability cliffs**: At what team size does kvelmo's coordination break down?
5. **Observability blindspots**: Can leads understand team-wide AI activity patterns?
6. **Workflow completeness**: Are there gaps between kvelmo and existing team tools (CI, PM, chat)?
7. **Integration gaps**: What team infrastructure does kvelmo need to connect to?
8. **Data ownership & portability**: Can the team migrate away from kvelmo without losing history?

Report all gaps found—whether 3 or 30. Each gap should include severity and a recommended fix.

---

## Sibling Commands

This command is part of a family of 8 persona-specific gap analyses:

- `/solo-developer-gaps`
- `/team-lead-gaps` (this command)
- `/opensource-maintainer-gaps`
- `/devops-gaps`
- `/cli-poweruser-gaps`
- `/frontend-dev-gaps`
- `/agent-dev-gaps`
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
