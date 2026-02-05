# Creating Tasks

Create tasks from scratch, upload files, or pull from external providers like GitHub, Jira, and Linear.

## The Create Task Dialog

Click **"Create Task"** on the dashboard to open the creation dialog. You'll see four tabs:

```text
              ┌─────────────┐
              │ Create Task │
              └──────┬──────┘
                     │
    ┌────────┬───────┼───────┬───────────────┐
    ▼        ▼       ▼       ▼               ▼
┌────────┐ ┌────────┐ ┌──────┐ ┌───────────────┐
│ Write  │ │ Upload │ │ From │ │ From Provider │
│ Task   │ │ File   │ │ URL  │ │               │
└────────┘ └────────┘ └──────┘ └───────────────┘
```

## Option 1: Write Task

Write your task description directly in the browser.

**Steps:**
1. Click **"Create Task"**
2. Select the **"Write Task"** tab
3. Enter a title and description
4. Click **"Create Task"**

The **Create Task** dialog has four tabs at the top: **Write Task**, **Upload File**, **From URL**, and **From Provider**. The Write Task tab shows a title field and a description text area that supports markdown formatting. Click **Create Task** to save.

**Example task:**

````markdown
# Add User Authentication

Implement OAuth2 login with Google as the provider.

## Requirements
- Use the existing OAuth2 library
- Store user sessions in PostgreSQL
- Add logout functionality

## Endpoints
- GET /auth/login - Redirect to Google OAuth
- GET /auth/callback - Handle OAuth return
- POST /auth/logout - Clear session cookie
````

## Option 2: Upload File

Drag and drop an existing task file.

**Supported formats:**
- `.md` - Markdown with frontmatter
- `.txt` - Plain text description

**Steps:**
1. Click **"Create Task"**
2. Select the **"Upload File"** tab
3. Drag and drop your file (or click to browse)
4. Click **"Create Task"**

The **Upload File** tab shows a drop zone where you can drag and drop a task file, or click to browse. Supported formats are `.md` (markdown with frontmatter) and `.txt` (plain text), up to 5MB.

## Option 3: From URL

Fetch task description from a URL.

**Steps:**
1. Click **"Create Task"**
2. Select the **"From URL"** tab
3. Paste the URL
4. Click **"Create Task"**

## Option 4: From Provider

Pull tasks directly from external project management tools.

**Steps:**
1. Click **"Create Task"**
2. Select the **"From Provider"** tab
3. Choose your provider (GitHub, GitLab, Jira, etc.)
4. Enter the issue/ticket number
5. Click **"Create Task"**

The **From Provider** tab lets you select a provider from the dropdown (GitHub, GitLab, Jira, etc.) and enter the issue or ticket number. The dialog shows which repository it will fetch from based on your project's git remote.

### Supported Providers

Providers must be configured before use. Supported providers include:

- GitHub
- GitLab
- Jira
- Linear
- Notion
- Trello
- Asana
- ClickUp

See [Providers](/providers/index.md) for setup instructions.

### GitHub Example

1. Configure GitHub access (see [Providers: GitHub](/providers/github.md))
2. In the Web UI:
   - Select **"GitHub"** from the provider dropdown
   - Enter the issue number (e.g., `42`)
   - Click **"Create Task"**

Mehrhof automatically fetches:
- Issue title and description
- Labels (to infer task type)
- Comments
- Linked issues (`#123` references)

### Using Shorthands

Some providers support shorthand notation:

| Provider | Shorthand                   | Example    |
|----------|-----------------------------|------------|
| GitHub   | `gh`                        | `gh:42`    |
| GitLab   | `gl`                        | `gl:123`   |
| Jira     | None (requires project key) | `PROJ-123` |

## What Happens After Creating

Once you create a task:

1. **Task ID generated** - Unique 8-character identifier
2. **Git branch created** - Based on your branch pattern (e.g., `feature/add-user-auth`)
3. **Work directory initialized** - At `~/.valksor/mehrhof/workspaces/<project-id>/work/<id>/`
4. **Task set as active** - Appears in the Active Task card

Once created, the task appears in the **Active Task** card showing its state, git branch, and progress. Action buttons let you proceed with the workflow: **Plan**, **Implement**, **Review**, and **Finish**.

### Local Metadata Enrichment

When starting a task from an external provider (GitHub, Jira, Wrike, etc.), Mehrhof automatically searches local queues for a task with a matching external ID. If found, local metadata is merged into the work context:

- **Description**: Local description replaces provider description when it contains richer content
- **Metadata**: Custom frontmatter fields from local task files fill gaps in provider data (local never overwrites provider)
- **Source files**: If the local task has a source path, those files are included in the agent's context

This lets you enrich external tasks with local code examples, file references, or implementation notes. See [CLI: start](/cli/start.md#local-metadata-enrichment) for details.

## Next Steps

After creating your task:

- [**Planning**](planning.md) - Generate implementation specifications
- [**Notes**](notes.md) - Add context before planning
- [**Dashboard**](dashboard.md) - Understand the full interface

---

## Also Available via CLI

Create tasks from the command line for terminal-based workflows or scripting.

See [CLI: start](/cli/start.md) for all providers and options.
