# Syncing

The Sync action refreshes the current task from its external provider and prepares a delta specification when upstream content changes.

## Where to Use It

1. Open an active task in the Web UI (Task Detail view).
2. In the **Actions** card, click **Sync**.
3. Wait for the sync result banner to appear.

## What Happens During Sync

When you run Sync, Mehrhof:

1. Fetches the latest task content from the provider.
2. Compares it with the local source snapshot.
3. Generates a new delta specification if differences are found.
4. Updates local source content for the active task.
5. Shows generated artifact paths directly in the result banner.

## Sync Result Banner

After Sync completes, the Actions card shows:

- **Message** indicating whether changes were detected.
- **Delta specification path** when a new sync specification is generated.
- **Change summary** when differences are detected.
- **Source update status** (`yes` or `no`).
- **Previous snapshot path** and **diff path** when provider-specific artifacts are created.
- **Warnings** if non-blocking file operations fail.

## Provider-Specific Artifacts

For Wrike tasks, Sync can create additional files in the task source directory:

- `wrike_previous.txt` - Snapshot of content before sync.
- `wrike_diff.txt` - Human-readable diff summary.

## When to Sync

Use Sync when:

- An external task was edited after you started implementation.
- New provider comments or metadata should be reflected before further planning.
- You need an auditable delta specification before continuing implementation.

---

## Also Available via CLI

Need terminal-based sync or scripting support? See [CLI: sync](/cli/sync.md) for all command options.
