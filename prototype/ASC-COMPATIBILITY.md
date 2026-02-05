# ASC Compatibility Guide

This is a migration / parallel use guide for ASC users. ASC is in no way, shape or form affiliated with Mehrhof, and ASC is not needed for Mehrhof to work.  

How to configure Mehrhof to match the asc file structure and conventions.

## Quick Setup

**Option 1: Use the `--asc` flag (easiest)**

```bash
mehr init --asc
```

This automatically configures all ASC-compatible settings. The flag is deliberately hidden from `--help` output.

**Option 2: Manual configuration**

Add this to your `.mehrhof/config.yaml`:

```yaml
git:
  branch_pattern: "asc/{key}"
  commit_prefix: "[{key}]"

storage:
  save_in_project: true
  project_dir: "tickets"

specification:
  filename_pattern: "SPEC-{n}.md"

review:
  filename_pattern: "CODERABBIT-{n}.txt"
```

## What This Does

| Feature        | ASC Pattern              | Mehrhof Config                                  |
|----------------|--------------------------|-------------------------------------------------|
| Branch naming  | `asc/A-123`              | `git.branch_pattern: "asc/{key}"`               |
| Commit prefix  | `[A-123] message`        | `git.commit_prefix: "[{key}]"`                  |
| Work directory | `tickets/A-123/`         | `storage.project_dir: "tickets"`                |
| Spec files     | `SPEC-1.md`, `SPEC-2.md` | `specification.filename_pattern: "SPEC-{n}.md"` |
| Review files   | `CODERABBIT-1.txt`       | `review.filename_pattern: "CODERABBIT-{n}.txt"` |

## File Structure

After configuration, your project will have:

```
your-repo/
├── .mehrhof/
│   └── config.yaml          # Your config
├── tickets/
│   └── A-123/               # Task directory (committed to repo)
│       ├── SPEC-1.md        # First specification
│       ├── SPEC-2.md        # Second specification (iterations)
│       ├── CODERABBIT-1.txt # First review output
│       └── CODERABBIT-2.txt # Second review output
└── ...
```

## Commands

```bash
# Initialize mehrhof (creates .mehrhof/config.yaml)
mehr init

# Edit config to add ASC settings
# (edit .mehrhof/config.yaml as shown above)

# Start a task - creates branch asc/A-123
mehr start A-123

# Plan - saves to tickets/A-123/SPEC-1.md
mehr plan

# Review with external tool - saves to tickets/A-123/CODERABBIT-1.txt
mehr review --tool coderabbit

# Commit changes - uses [A-123] prefix
mehr commit
```

## Key Differences from ASC

| Feature         | ASC                        | Mehrhof                              |
|-----------------|----------------------------|--------------------------------------|
| Sessions        | `.asc/sessions.json`       | Per-task YAML files (richer)         |
| Undo/Redo       | `.git/asc/undo_state.json` | Full git checkpoint system           |
| Config location | `~/.asc/settings.json`     | `.mehrhof/config.yaml` (per-project) |

## Notes

- The `{key}` placeholder resolves to the task's external key (e.g., `A-123` from Wrike, GitHub issue number)
- The `{n}` placeholder in filename patterns is replaced with the spec/review number
- When `storage.save_in_project: true`, all work files are stored in `tickets/<task-id>/` (or your configured `project_dir`)
- When `storage.save_in_project: false` (default), work is stored in `~/.valksor/mehrhof/workspaces/<project-id>/work/`
