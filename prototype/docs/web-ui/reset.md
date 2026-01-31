# Reset State (Web UI)

Reset the workflow state to idle without losing work.

## Using the Dashboard

When a task is stuck in planning, implementing, or reviewing state (e.g., after killing a hung agent), a **Reset State** button appears in the Actions panel.

1. Navigate to the dashboard
2. Find the **Actions** panel
3. Click **Reset State**
4. Confirm the action when prompted

The button appears with a warning style (yellow) to indicate it's a recovery action.

## API Endpoint

```
POST /api/v1/workflow/reset
```

### Response

```json
{
  "status": "reset",
  "state": "idle",
  "message": "workflow state reset to idle"
}
```

### cURL Example

```bash
curl -X POST http://localhost:8080/api/v1/workflow/reset
```

## When to Use

Use the reset function when:

- The agent process was killed externally
- The workflow state is stuck (planning/implementing/reviewing)
- You want to retry a step without losing specifications or code

## What Gets Preserved

- Specifications (all generated spec files)
- Notes (all task notes)
- Code changes (everything committed or staged)
- Task metadata

## What Gets Reset

- Workflow state (set to "idle")
- State machine position

## See Also

- [CLI: mehr reset](../cli/reset.md) - Command line equivalent
- [API Reference](../reference/api.md) - Full API documentation
