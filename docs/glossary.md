# Glossary

Terms and definitions used in kvelmo.

## A

### Agent
An AI model that executes kvelmo's workflow phases. Examples: Claude, Codex.

### Abandon
Complete cleanup of a task, removing the branch and work directory.

### Abort
Stop the current task execution and transition to failed state.

## B

### Branch
A git branch created for each task. Pattern: `feature/<slug>`.

## C

### Checkpoint
A git commit created at each workflow phase, enabling undo/redo.

### CLI
Command Line Interface. Text-based interface for kvelmo.

### Conductor
The core orchestrator component that drives the workflow.

## E

### Event
A message emitted during agent execution (token, tool_call, etc.).

## G

### Global Socket
The main kvelmo socket at `~/.valksor/kvelmo/global.sock`.

### Guard
A condition that must be true for a state transition to occur.

## I

### Implement
The workflow phase where the agent writes code based on the specification.

## J

### Job
A unit of work in the worker pool.

### JSON-RPC
The protocol used for socket communication.

## M

### Memory
Semantic memory system for codebase understanding.

## P

### Phase
A stage in the workflow: start, plan, implement, review, submit.

### Plan
The workflow phase where a specification is generated.

### Provider
A source for tasks: file, GitHub, GitLab, Wrike.

## R

### Redo
Restore a checkpoint that was undone.

### Reset
Recover from a stuck or failed state.

### Review
The workflow phase for human approval of changes.

## S

### Socket
Unix domain socket for inter-process communication.

### Specification
A document describing how to implement a task, generated during planning.

### State
The current position in the workflow (loaded, planning, planned, etc.).

### State Machine
The system that manages workflow states and transitions.

### Submit
The workflow phase where a PR is created.

## T

### Task
A unit of work to be completed (feature, bug fix, etc.).

### Tool
A capability an agent can use (Read, Write, Bash, etc.).

### Transition
Moving from one state to another in the workflow.

## U

### Undo
Revert to a previous checkpoint.

## W

### Web UI
Browser-based graphical interface for kvelmo.

### Worker
A process that executes agent jobs.

### Worker Pool
The system that manages multiple concurrent workers.

### Worktree Socket
Per-project socket at `<project>/.kvelmo/worktree.sock`.

### Workflow
The sequence of phases: start → plan → implement → review → submit.
