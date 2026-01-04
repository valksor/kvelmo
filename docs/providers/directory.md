# Directory Provider

> **⚠️ Experimental**: This provider is not fully tested beyond unit tests. Edge cases may exist. Manual validation recommended before production use.

**Schemes:** `dir:`

**Capabilities:** `read`, `list`, `snapshot`

Reads tasks from markdown files in a local directory. Can list all available task files. All directory files are copied to the task's `source/` directory as read-only snapshots.

## Usage

```bash
mehr start dir:./tasks
mehr plan dir:./docs
```

## Listing Files

The directory provider can enumerate all markdown files in a directory, allowing you to browse available tasks before starting one.
