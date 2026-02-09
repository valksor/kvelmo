# Language & Translations

Customize the Mehrhof Web UI language and terminology to match your workflow and preferences.

## Language Selection

Mehrhof currently supports English as the primary language, with more languages planned for future releases.

To change your language preference:

1. Navigate to **Settings**
2. Open the **Work** section
3. Expand **Appearance**
4. Select your preferred language from the dropdown

Your language preference is saved automatically and persists across sessions.

## Translation Customization

Power users can customize translations to match their organization's terminology. This feature is available in **Advanced** settings mode.

### Accessing Translation Settings

1. Navigate to **Settings**
2. Click the **Simple/Advanced** toggle in the top-right corner of the page
3. Select the **System** section from the sidebar
4. Expand **Translation Customization**

### Terminology Replacements

Replace terms throughout the entire interface. Useful for aligning Mehrhof with your team's vocabulary.

| Example Use Case | Find      | Replace With |
|------------------|-----------|--------------|
| Jira integration | Task      | Ticket       |
| CI/CD workflow   | Workflow  | Pipeline     |
| Internal naming  | Dashboard | Home         |

**To add a terminology replacement:**

1. Enter the term to find
2. Enter the replacement text
3. Choose the scope:
   - **Global** — applies to all projects
   - **This Project** — applies only to the current project
4. Click **Add**

Terminology replacements are case-insensitive and apply across all translations.

#### Terminology Matching Behavior

Terminology replacements use **word boundary matching**, meaning:

- "Task" will match "Task" but **not** "TaskBar" or "Subtask"
- Matching is case-insensitive: "task", "TASK", "Task" all match
- Replacements preserve the original casing pattern when possible

| Find | Replace | Input           | Output                |
|------|---------|-----------------|-----------------------|
| Task | Ticket  | "Create a Task" | "Create a Ticket"     |
| Task | Ticket  | "TaskBar"       | "TaskBar" (no change) |
| Task | Ticket  | "Subtask"       | "Subtask" (no change) |

### Key Overrides

Override specific translation keys for fine-grained control. This is useful when you want to change individual labels without affecting related terms.

**To add a key override:**

1. Start typing the key name (autocomplete suggestions appear)
2. Enter the custom value
3. Choose the scope (Global or This Project)
4. Click **Add**

Common override keys include:

| Key                            | Default Value |
|--------------------------------|---------------|
| `nav.dashboard`                | Dashboard     |
| `nav.project`                  | Project       |
| `workflow:states.planning`     | Planning      |
| `workflow:states.implementing` | Implementing  |
| `workflow:task.title`          | Task          |

### Override Precedence

When both global and project overrides exist for the same term or key:

1. **Project overrides** take precedence within that project
2. **Global overrides** apply to all other projects
3. **Bundled defaults** apply when no override exists

This allows you to set organization-wide terminology globally while customizing for specific projects.

### Saving and Applying Changes

1. Make your changes in the editors
2. Click **Save Overrides**
3. The page will reload to apply the new translations

Changes take effect immediately after the page reloads.

## Scope Indicators

In the translation editors, scope badges show where each override originates:

| Badge       | Meaning                         |
|-------------|---------------------------------|
| **Global**  | Applies to all projects         |
| **Project** | Applies only to current project |

## Removing Overrides

To remove an override:

1. Locate the override in the list
2. Click the trash icon on the right
3. Click **Save Overrides** to apply

### Resetting All Overrides

To clear all translation customizations at once:

1. Click **Reset All** in the Translation Customization section
2. Confirm the reset when prompted
3. The page will reload with default translations

This clears both global and project-specific overrides.

## Storage Location

Translation overrides are stored in your Mehrhof home directory:

```
~/.valksor/mehrhof/i18n/
├── overrides.json                    # Global overrides
└── projects/
    └── {project-name}/
        └── overrides.json            # Project-specific overrides
```

## Adding New Languages

When new languages become available, they will appear automatically in the language selector. To contribute translations, see the developer documentation.

---

## Related

- [**Settings**](/web-ui/settings.md) — Full settings reference
- [**Configuration Guide**](/configuration/index.md) — Advanced configuration options
