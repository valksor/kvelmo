# `mehr commit`

Create logically grouped commits from uncommitted changes using AI.

## Overview

The `mehr commit` command analyzes your uncommitted git changes and groups them into logical commits based on semantic relationships (same feature, bugfix, refactor, etc.). Commit messages are generated to match the style of existing commits in your repository.

## Usage

```bash
mehr commit [flags]
```

## Flags

| Flag             | Short | Description                                            |
|------------------|-------|--------------------------------------------------------|
| `--agent-commit` |       | Agent to use for commit message generation             |
| `--all`          | `-a`  | Include unstaged changes in addition to staged changes |
| `--dry-run`      | `-n`  | Show what would be committed without creating commits  |
| `--note`         | `-m`  | Hint to guide AI grouping when re-running              |
| `--push`         | `-p`  | Push commits to remote after creating                  |

## Examples

### Preview commits before creating

```bash
mehr commit --dry-run
```

### Create commits from staged changes

```bash
mehr commit
```

### Include unstaged changes

```bash
mehr commit --all
```

### Create and push

```bash
mehr commit --push
```

### Guide AI grouping with a hint

```bash
mehr commit --note "group 1 and 3 are the same feature"
```

## How it Works

1. **Analysis**: The command analyzes all changed files (staged, or staged + unstaged with `--all`)
2. **AI Grouping**: Files are grouped into logical commits based on semantic relationships
3. **Style Matching**: Commit messages are generated to match the style of existing commits in your repository
4. **Preview**: You see the proposed commits with their messages
5. **Confirmation**: Confirm before creating (unless `--push` is used)
6. **Execution**: Commits are created, optionally pushed to remote

## Dry-Run Optimization

When you run `mehr commit --dry-run` followed by `mehr commit` (without any file changes), the grouping is reused - no AI call is made. This saves time when you're just previewing before committing.

## Steering with `--note`

If the AI's grouping doesn't make sense, you can guide it with a note:

```bash
# First attempt
$ mehr commit --dry-run
[1] Add authentication feature
[2] Update styling
[3] Fix navigation bug  # You want this merged with authentication

# Re-run with guidance
$ mehr commit --dry-run --note "merge groups 1 and 3, they're both about auth"
[1] Add authentication feature with navigation fix
[2] Update styling
```

## Commit Message Style Matching

The AI reads your existing commit history (`git log -20 --format=%B`) to match:

- **Format**: Emoji prefixes, conventional commits, Co-Authored-By tags
- **Length**: Short one-liners vs detailed multi-line messages
- **Tone**: Casual vs formal language

This ensures generated commits blend seamlessly with your repository's style.
