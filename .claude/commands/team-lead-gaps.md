# Team Lead Gap Analysis

Imagine you are a **team lead** managing 4-8 engineers on a product team. You've adopted AI-assisted development and now face new coordination challenges:

- Engineers using different AI tools with inconsistent outputs
- No visibility into what AI agents are actually doing across the team
- Code reviews bottlenecked because AI-generated code needs extra scrutiny
- Sprint planning doesn't account for AI's impact on velocity
- Onboarding new engineers to AI-assisted workflows takes longer than expected
- Knowledge silos forming around which prompts work best for your codebase
- Hard to tell if AI is helping or creating tech debt faster

You want **kvelmo** to give you visibility, consistency, and control over AI-assisted development across your team.

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

1. **Socket layer** (global + worktree sockets)
2. **State machine** (11 states, guards, transitions)
3. **Agent interface** (WebSocket + CLI modes)
4. **Worker pool** (job queue, event streaming)
5. **Providers** (GitHub, GitLab, Wrike, file)
6. **Storage** (tasks, chats, plans, reviews)
7. **Git integration** (checkpoints, undo/redo)
8. **Web UI** (real-time updates, stores)
9. **CLI commands** (50+ commands)
10. **Memory system** (embeddings, semantic search)

### Critical Rules

1. **Extend, don't replace**: New features add alongside existing ones. Never remove capabilities.
2. **Domain-agnostic core**: Features benefiting one persona should work for all unless explicitly domain-specific.
3. **Check the siblings**: Before recommending a feature, verify it doesn't break other personas' workflows.
