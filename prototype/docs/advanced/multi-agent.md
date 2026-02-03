# Multi-Agent Orchestration

Chain multiple AI agents together for complex workflows using sequential, parallel, or consensus-based execution.

## Overview

Multi-agent orchestration enables you to:
- **Chain agents sequentially** - Pass outputs from one agent to the next
- **Run agents in parallel** - Execute multiple agents simultaneously
- **Build consensus** - Combine outputs from multiple agents using voting or synthesis

## Configuration

Enable orchestration in `.mehrhof/config.yaml`:

```yaml
orchestration:
  enabled: true
  steps:
    planning:
      mode: sequential  # sequential, parallel, or consensus
      agents:
        - name: architect
          agent: claude
          model: claude-opus-4
          role: Design system architecture and identify components
          output: architecture.md

        - name: security-analyst
          agent: claude
          model: claude-sonnet-4
          role: Review architecture for security concerns
          input: [architecture.md]
          output: security-review.md

        - name: technical-lead
          agent: claude
          model: claude-opus-4
          role: Synthesize inputs into final specification
          input: [architecture.md, security-review.md]
          output: spec.md

    implementing:
      mode: single      # Use single agent for implementation
      agent: claude

    reviewing:
      mode: parallel
      agents:
        - name: code-reviewer
          agent: claude
          role: Review code quality and correctness

        - name: security-reviewer
          agent: claude
          role: Review for security vulnerabilities

        - name: test-reviewer
          agent: claude
          role: Review test coverage and quality
      consensus:
        mode: majority   # majority, unanimous, or any
        min_votes: 2
        synthesizer: technical-lead
```

## Orchestration Modes

### Sequential

Execute agents one after another, passing outputs as inputs to the next agent:

```yaml
steps:
  planning:
    mode: sequential
    agents:
      - name: researcher
        agent: claude
        role: Research best practices
        output: research.md

      - name: architect
        agent: claude
        role: Design based on research
        input: [research.md]
        output: design.md

      - name: implementer
        agent: claude
        role: Create implementation plan
        input: [research.md, design.md]
```

**Data Flow**: `researcher → architect → implementer`

### Parallel

Execute multiple agents simultaneously, then collect all results:

```yaml
steps:
  reviewing:
    mode: parallel
    agents:
      - name: code-reviewer
        agent: claude
        role: Review code quality

      - name: security-reviewer
        agent: claude
        role: Review security

      - name: performance-reviewer
        agent: claude
        role: Review performance
```

**Data Flow**: All agents execute independently, results collected together.

### Consensus

Execute agents in parallel and build consensus from their outputs:

```yaml
steps:
  decision:
    mode: consensus
    agents:
      - name: agent-a
        agent: claude
        role: Provide analysis

      - name: agent-b
        agent: claude
        role: Provide analysis

      - name: agent-c
        agent: claude
        role: Provide analysis
    consensus:
      mode: majority   # or unanimous, any
      min_votes: 2
      synthesizer: lead-agent
```

**Consensus Modes**:
- `majority` - Most common output wins
- `unanimous` - All agents must agree
- `any` - First valid output

## Agent Step Configuration

Each agent in the pipeline supports:

| Field     | Description                                   |
|-----------|-----------------------------------------------|
| `name`    | Unique step identifier                        |
| `agent`   | Agent to use (from registry)                  |
| `model`   | Model override (optional)                     |
| `role`    | Instructions for the agent                    |
| `input`   | List of input artifacts to consume            |
| `output`  | Output artifact name                          |
| `depends` | List of step dependencies (for DAG execution) |
| `env`     | Environment variables for the agent           |
| `args`    | Additional arguments for the agent            |
| `timeout` | Per-step timeout in seconds                   |

## Artifact Pipeline

Artifacts are named outputs that can be passed between steps:

```yaml
steps:
  planning:
    mode: sequential
    agents:
      - name: architect
        output: architecture.md

      - name: security-review
        input: [architecture.md]
        output: secure-architecture.md

      - name: implementation-plan
        input: [architecture.md, secure-architecture.md]
        # No output - final step
```

## Examples

### Three-Stage Planning

1. **Research** - Gather requirements and best practices
2. **Design** - Create system architecture
3. **Review** - Validate and refine

```yaml
steps:
  planning:
    mode: sequential
    agents:
      - name: researcher
        agent: claude
        role: Research requirements and best practices
        output: research.md

      - name: architect
        agent: claude
        role: Design system architecture
        input: [research.md]
        output: design.md

      - name: reviewer
        agent: claude
        role: Validate and refine design
        input: [research.md, design.md]
```

### Parallel Code Review

Review code from multiple perspectives simultaneously:

```yaml
steps:
  reviewing:
    mode: parallel
    agents:
      - name: functional-review
        agent: claude
        role: Review functionality and correctness

      - name: security-review
        agent: claude
        role: Review for security vulnerabilities

      - name: performance-review
        agent: claude
        role: Review for performance issues

      - name: style-review
        agent: claude
        role: Review code style and conventions
    consensus:
      mode: majority
      synthesizer: lead-reviewer
```

### Consensus-Based Decision

Get agreement from multiple agents before proceeding:

```yaml
steps:
  critical-decision:
    mode: consensus
    agents:
      - name: senior-dev
        agent: claude
        role: Evaluate from senior developer perspective

      - name: security-expert
        agent: claude
        role: Evaluate from security perspective

      - name: performance-expert
        agent: claude
        role: Evaluate from performance perspective
    consensus:
      mode: unanimous
      synthesizer: tie-breaker
```

## Advanced Features

### Dependency Management

Execute steps with complex dependencies:

```yaml
steps:
  complex:
    mode: sequential
    agents:
      - name: step-a
        output: artifact-a

      - name: step-b
        output: artifact-b

      - name: step-c
        depends: [step-a, step-b]  # Wait for both
        input: [artifact-a, artifact-b]

      - name: step-d
        depends: [step-a]  # Only needs A
        input: [artifact-a]
```

### Custom Environment

Pass environment variables to agents:

```yaml
steps:
  planning:
    mode: sequential
    agents:
      - name: planner
        agent: claude
        env:
          API_KEY: "${SECRET_API_KEY}"
          LOG_LEVEL: debug
        args:
          - --verbose
          - --debug
```

### Per-Step Timeouts

Set different timeouts for each step:

```yaml
steps:
  planning:
    mode: sequential
    agents:
      - name: quick-review
        timeout: 60  # 1 minute

      - name: deep-analysis
        timeout: 600  # 10 minutes
```

## Performance Considerations

- **Sequential** - Slower overall, but outputs inform next step
- **Parallel** - Faster, independent execution
- **Consensus** - Parallel + consensus overhead, but more reliable

## Troubleshooting

### Agent Not Found

Ensure the agent is registered in your config:

```yaml
agents:
  aliases:
    custom-agent: claude  # Register alias
```

### Dependency Not Met

Check that all `depends` names match actual step `name` values.

### Consensus Timeout

Increase timeout or reduce number of agents in consensus.

## See Also

- [Configuration Guide](/configuration/index.md) - Orchestration settings
- [Agent Documentation](/agents/index.md) - Available agents
- [Workflow Concepts](/concepts/workflow.md) - How orchestration fits in workflow
