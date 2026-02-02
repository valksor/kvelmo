# Example Task Files

Ready-to-use task files for Mehrhof. Copy one to your project or start directly:

```bash
mehr start examples/add-feature.md
```

## Examples

| File | Type | Description |
|------|------|-------------|
| [add-feature.md](add-feature.md) | Feature | Add a new API endpoint with tests |
| [fix-bug.md](fix-bug.md) | Bug fix | Fix an existing issue with validation |
| [update-docs.md](update-docs.md) | Docs | Update project documentation |

## Usage

**Option 1** — Start directly from the examples directory:

```bash
mehr start examples/add-feature.md
```

**Option 2** — Copy to your project and customize:

```bash
cp examples/add-feature.md my-task.md
# Edit my-task.md to fit your needs
mehr start my-task.md
```

## Task File Format

Task files are Markdown with optional YAML frontmatter. See [Task File Format](https://valksor.com/docs/mehrhof/nightly/#/reference/task-format) for the full reference.
