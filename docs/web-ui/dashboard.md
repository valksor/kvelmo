# Dashboard

The dashboard is your central view for managing tasks in kvelmo.

## Opening the Dashboard

Start the server and open your browser:

```bash
kvelmo serve --open
```

Or navigate to http://localhost:6337.

## Dashboard Layout

The dashboard consists of several panels:

### Project Selector

At the top, select which project to work on. kvelmo can manage multiple projects simultaneously.

### Task Status

Shows the current task state and progress:
- Task title and description
- Current state (loaded, planning, implemented, etc.)
- Time elapsed
- Agent activity

### Actions Panel

Workflow buttons for the current state:
- **Plan** — Generate specification
- **Implement** — Execute specification
- **Simplify** — Optional code cleanup
- **Optimize** — Optional optimization
- **Review** — Start review phase
- **Submit** — Create PR

Buttons are enabled/disabled based on the current state.

### Output Panel

Real-time output from the agent:
- Agent thoughts and reasoning
- Tool calls and results
- Errors and warnings

### Sidebar

Access to additional panels:
- Files — Browse project files
- Changes — View file diffs
- Checkpoints — Navigate undo/redo history
- Workers — Monitor worker pool
- Memory — Semantic memory
- Screenshots — Screenshot gallery
- Browser — Browser automation
- Settings — Configuration

## Task States

The dashboard shows the current state with visual indicators:

| State          | Indicator      | Description                |
|----------------|----------------|----------------------------|
| `none`         | Gray           | No active task             |
| `loaded`       | Blue           | Task loaded, ready to plan |
| `planning`     | Yellow spinner | Planning in progress       |
| `planned`      | Green          | Ready to implement         |
| `implementing` | Yellow spinner | Implementation in progress |
| `implemented`  | Green          | Ready to review            |
| `reviewing`    | Yellow         | Review in progress         |
| `submitted`    | Green check    | Task complete              |
| `failed`       | Red            | Error occurred             |

## Keyboard Shortcuts

| Shortcut | Action    |
|----------|-----------|
| `Ctrl+P` | Plan      |
| `Ctrl+I` | Implement |
| `Ctrl+R` | Review    |
| `Ctrl+Z` | Undo      |
| `Ctrl+Y` | Redo      |

## Refreshing

The dashboard updates automatically via WebSocket. No manual refresh needed.

Prefer the command line? See [CLI Reference](/cli/index.md).
