# Tools

The Tools page consolidates advanced debugging, memory search, security scanning, and stack management into one location.

## Accessing Tools

From the navigation bar, click **Tools**. The page has four tabs: **Browser**, **Memory**, **Security**, and **Stack**.

## Browser Tab

Control and inspect a Chrome browser for testing, verification, and debugging.

### Browser Status

Shows the current connection status:
- **Connected** — Browser is active with host and port displayed
- **Not connected** — No browser session (check configuration in [Settings](/web-ui/settings.md))

### Navigate

Enter a URL and click **Go** to navigate the browser.

### Open Tabs

When connected, displays all open browser tabs with:
- Tab title and URL
- **Close** button to close individual tabs
- **External link** to open in your browser

### Interactions

Expand the **Interactions** section to access:

**Screenshot:**
- Choose format (PNG or JPEG)
- Enable **Full page** for scrollable content
- Click **Capture** to take screenshot
- Download the captured image

**Click & Type:**
- Enter a CSS selector (e.g., `#submit`, `.btn-primary`, `[data-testid='login']`)
- Click **Click** to click the element
- Enter text and click **Type** to input text (optionally clear first)

**Evaluate JavaScript:**
- Enter JavaScript code
- Click **Run** to execute in the browser
- View the result below

**DOM Query:**
- Enter a CSS selector
- Enable **Query all** to find multiple elements
- Enable **Include HTML** to see element HTML
- Click **Query** to inspect elements

### DevTools

Expand the **DevTools** section for advanced monitoring:

**Network Tab:**
- Set duration (seconds) to monitor
- Enable **Capture bodies** for request/response content
- Click **Monitor** to start
- View method, URL, status, type, size, and timing

**Console Tab:**
- Set duration to monitor
- Filter by level (all, log, warning, error)
- Click **Monitor** to capture console output

**WebSocket Tab:**
- Monitor WebSocket frames
- See sent/received messages with direction indicators

**Source Tab:**
- Click **Get Page Source** to retrieve HTML
- Copy to clipboard for inspection

**Coverage Tab:**
- Track JavaScript and CSS usage
- See percentage of code actually used

## Memory Tab

Search semantic memory for similar past tasks, code changes, and solutions.

### Searching Memory

1. Enter your search query
2. Filter by document type:
   - **Code** — Code changes from past tasks
   - **Specifications** — Implementation plans
   - **Sessions** — Agent conversation logs
   - **Solutions** — Fixes and corrections
   - **Decisions** — Architectural decisions
   - **Errors** — Past errors and resolutions
3. Set the results limit
4. Click **Search**

### Understanding Results

Each result shows:

| Field | Description |
|-------|-------------|
| **Type** | Document category |
| **Task ID** | Source task (truncated) |
| **Match** | Similarity percentage |
| **Content** | Relevant excerpt |

Higher match percentages indicate stronger semantic similarity.

### Configuration

Memory settings are in [Settings → Advanced → Memory System](/web-ui/settings.md#memory-system).

## Security Tab

Run security scanners to detect vulnerabilities, exposed secrets, and dependency issues.

### Available Scanners

| Scanner | Purpose |
|---------|---------|
| **GoSec** | Go static analysis (SAST) |
| **GitLeaks** | Secret detection |
| **Go Vuln Check** | Go vulnerability database |
| **Semgrep** | Multi-language SAST |
| **NPM Audit** | JavaScript dependencies |
| **ESLint Security** | JavaScript security rules |
| **Bandit** | Python security analysis |
| **Pip Audit** | Python dependencies |

### Running a Scan

1. Select scanners to run (check the boxes)
2. Set the **Fail Level** — severity that indicates failure:
   - Critical only
   - High and above
   - Medium and above
   - Low and above
   - Any finding
3. Click **Run Scan**

### Scan Results

Results show:

**Summary:**
- Total findings count
- Breakdown by severity (Critical, High, Medium, Low)
- Pass/fail status based on your fail level

**Findings:**
Each finding displays:
- Severity badge
- Scanner name
- Rule ID (if applicable)
- Description message
- File location with line number

### Configuration

Security settings are in [Settings → Advanced → Security Scanning](/web-ui/settings.md#security-scanning).

## Stack Tab

Manage dependent task branches (stacked features) and keep them synchronized.

### Understanding Stacks

A stack is a chain of dependent tasks where each task's branch builds on the previous one. This enables:
- Breaking large features into reviewable chunks
- Maintaining dependencies between related changes
- Automatic rebasing when parent branches change

### Stack List

The stack list shows:
- Stack ID
- Number of tasks
- **Conflict** indicator if rebase conflicts exist
- **Needs Rebase** indicator if parent changed

### Stack Actions

**Sync All:**
Click **Sync All** to update all stacks with their upstream changes.

**Rebase:**
Click **Rebase** on individual stacks to rebase child branches onto updated parents.

### Stack Details

Each stack shows its tasks with:
- Order number (1, 2, 3...)
- Branch name
- Current state
- PR link (if created)
- Dependency indicator

### Configuration

Stack settings are in [Settings → Advanced → Stack Settings](/web-ui/settings.md#stack-settings).

---

## Also Available via CLI

Prefer working from the terminal? See:
- [CLI: browser](/cli/browser.md) — Browser automation commands
- [CLI: memory](/cli/memory.md) — Memory search and management
- [CLI: scan](/cli/scan.md) — Security scanning
- [CLI: stack](/cli/stack.md) — Stack management

## Next Steps

- [**Settings**](/web-ui/settings.md) — Configure Tools features
- [**Dashboard**](/web-ui/dashboard.md) — Return to main view
