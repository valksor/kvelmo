Audit kvelmo's feature surface for completeness and parity across CLI, web UI, and socket RPC.

---

## Phase 1: Package-to-Surface Mapping

For each package in `pkg/`:

1. Read the package's exported API (service structs, public methods)
2. Check if it has:
   - **CLI command(s)** in `cmd/kvelmo/commands/`
   - **Socket RPC method(s)** registered in `pkg/socket/`
   - **Web UI coverage** in `web/src/`
3. Assess whether the package SHOULD have user-facing surface (some packages like `paths/` are internal-only)

Report any packages that have CLI but no web UI, or web UI but no CLI (violates parity rule in CLAUDE.md).

---

## Phase 2: Socket RPC Coverage

1. List all registered RPC methods in the socket server
2. For each method, verify:
   - A CLI command invokes it (or explain why not)
   - The web UI calls it via WebSocket (or explain why not)
3. Flag any "dead" RPC methods with no callers

---

## Phase 3: CLI/Web Parity Check

Per CLAUDE.md: "CLI and web UI must maintain feature parity; never ship one without the other."

1. List all CLI commands in `cmd/kvelmo/commands/`
2. For each command, check if the equivalent action exists in the web UI
3. List all web UI actions/buttons/workflows
4. For each action, check if the equivalent CLI command exists
5. Report gaps in either direction

---

## Phase 4: Persona Alignment

Cross-reference the 8 persona gap analyses against actual features:

### Solo Developer
Should have: task loading from multiple sources, planning, implementation, review, undo/redo, PR submission
- Cross-check with `/solo-developer-gaps` goals

### Team Lead
Should have: multi-project visibility, worker pool monitoring, metrics, audit trail
- Cross-check with `/team-lead-gaps` goals

### Open Source Maintainer
Should have: GitHub provider, PR workflows, issue integration
- Cross-check with `/opensource-maintainer-gaps` goals

### DevOps/SRE
Should have: metrics, security scanning, configuration management, deployment configs
- Cross-check with `/devops-gaps` goals

### CLI Power User
Should have: composable commands, JSON output, shell completion, streaming
- Cross-check with `/cli-poweruser-gaps` goals

### Frontend Developer
Should have: full web UI coverage, real-time updates, visual diff, dashboard
- Cross-check with `/frontend-dev-gaps` goals

### Agent Developer
Should have: agent interface docs, event streaming, permission system, testing
- Cross-check with `/agent-dev-gaps` goals

### Enterprise
Should have: configuration management, access control, audit logging, backup
- Cross-check with `/enterprise-gaps` goals

---

## Output Format

For each gap found:

```
## [Category] Issue Title
- **Type**: Parity gap / Missing surface / Dead code / Persona gap
- **Evidence**: [file:line or observation]
- **Impact**: [what breaks or is missing for which persona]
- **Recommendation**: [specific action]
- **Effort**: [Fibonacci 1-13]
```

---

## Summary Checklist

- [ ] All `pkg/` packages assessed for user-facing surface
- [ ] All socket RPC methods have CLI and/or web UI callers
- [ ] No CLI-only features (must have web UI equivalent)
- [ ] No web-UI-only features (must have CLI equivalent)
- [ ] Each persona's core goals have corresponding features
- [ ] No dead RPC methods or unreachable code paths
