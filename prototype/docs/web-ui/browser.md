# Browser Control

The Browser Control Panel provides a web interface for automating Chrome during testing and authentication flows.

## Accessing Browser Control

Navigate to `/browser` or click **"Browser"** in the navigation.

```
┌──────────────────────────────────────────────────────────────┐
│  Browser Automation                                          │
├──────────────────────────────────────────────────────────────┤
│  Chrome detected: chrome (port 9222)                         │
│                                                              │
│  Open Tabs:                                                  │
│  ┌────────────────────────────────────────────────────┐      │
│  │ 🌐 GitHub - valksor/go-mehrhof          [Active]   │      │
│  │ 🌐 Localhost:8080 - Health Endpoint                │      │
│  │ 🌐 Google - "How to implement OAuth"               │      │
│  └────────────────────────────────────────────────────┘      │
│                                                              │
│  Controls:                                                   │
│  URL: [____________________]  [Goto]  [Refresh]              │
│                                                              │
│  [Screenshot] [Console] [DOM Query] [Close Tab]              │
│                                                              │
│  Last action: Navigated to localhost:8080/health             │
└──────────────────────────────────────────────────────────────┘
```

## Starting Chrome

Before using browser control, start Chrome with remote debugging enabled:

```bash
# Start Chrome with remote debugging
google-chrome --remote-debugging-port=9222

# Or on macOS
/Applications/Google\ Chrome.app/Contents/MacOS/Google\ Chrome --remote-debugging-port=9222

# Or with Chrome binary
chrome --remote-debugging-port=9222
```

## Features

### Tab Management

View and manage all open Chrome tabs:

| Action          | Description                   |
|-----------------|-------------------------------|
| **List tabs**   | Shows all open tabs with URLs |
| **Switch tabs** | Click a tab to make it active |
| **Close tab**   | Close the selected tab        |
| **Refresh**     | Reload the tab list           |

### Navigation

Navigate to any URL:

```
┌──────────────────────────────────────────────────────────────┐
│  Navigate to URL                                             │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  URL: [http://localhost:8080/health___________]              │
│                                                              │
│  [Goto]  [Cancel]                                            │
└──────────────────────────────────────────────────────────────┘
```

### Screenshots

Capture screenshots of the current page:

```
┌──────────────────────────────────────────────────────────────┐
│  Screenshot Options                                          │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  [Full Page]  [Visible Only]                                 │
│                                                              │
│  Format: [PNG ▼]  [JPEG]                                     │
│                                                              │
│  Quality: [████████░░] 80%                                   │
│                                                              │
│  [Capture]  [Cancel]                                         │
└──────────────────────────────────────────────────────────────┘
```

Screenshots are saved to the configured screenshot directory.

### DOM Query

Inspect page elements using CSS selectors:

```
┌──────────────────────────────────────────────────────────────┐
│  DOM Query                                                   │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  Selector: [#submit-button________________]                  │
│                                                              │
│  Results (1):                                                │
│  • <button id="submit-button" class="btn primary">           │
│      Submit                                                  │
│    </button>                                                 │
│                                                              │
│  [Query]  [Cancel]                                           │
└──────────────────────────────────────────────────────────────┘
```

### JavaScript Console

View console output:

```
┌──────────────────────────────────────────────────────────────┐
│  Console Output                                              │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  Level: [All ▼]  [Info]  [Warn]  [Error]                     │
│                                                              │
│  10:23:45 [Info]   Page loaded                               │
│  10:23:46 [Info]   API request: GET /api/users               │
│  10:23:47 [Warn]   Deprecated API usage detected             │
│  10:23:48 [Error]  Failed to load resource: net::ERR_...     │
│                                                              │
│  [Clear]  [Close]                                            │
└──────────────────────────────────────────────────────────────┘
```

### Element Interaction

Click elements and type text:

```
┌──────────────────────────────────────────────────────────────┐
│  Interact with Element                                       │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  Action: [Click ▼]  [Type]                                   │
│                                                              │
│  Selector: [#email-input________________]                    │
│                                                              │
│  Text to type: [user@example.com____________]                │
│  (for Type action only)                                      │
│                                                              │
│  [Execute]  [Cancel]                                         │
└──────────────────────────────────────────────────────────────┘
```

## Use Cases

### Testing Endpoints

1. Start your server locally
2. Navigate to the endpoint
3. Capture screenshot to verify response

### Authentication Flows

1. Navigate to login page
2. Fill in credentials
3. Click submit button
4. Verify successful login

### Web Scraping

1. Navigate to target page
2. Use DOM query to find elements
3. Extract data from selected elements

## Troubleshooting

### Chrome Not Detected

```
⚠️ Chrome not detected

Make sure Chrome is running with:
  chrome --remote-debugging-port=9222
```

**Solutions:**
1. Start Chrome with remote debugging enabled
2. Verify the port matches (default: 9222)
3. Check if another process is using the port

### Connection Timeout

```
⚠️ Connection timeout

Could not connect to Chrome. Is it running?
```

**Solutions:**
1. Verify Chrome is running
2. Check the remote debugging port
3. Restart Chrome if needed

## Configuration

Configure browser settings in [Settings](settings.md) or `.mehrhof/config.yaml`:

```yaml
browser:
  enabled: true
  headless: false
  port: 9222
  timeout: 30
  screenshot_dir: "./screenshots"
```

## Next Steps

- [**Settings**](settings.md) - Configure browser options
- [**CLI: browser**](../cli/browser.md) - CLI browser commands

## CLI Equivalent

```bash
# Check browser status
mehr browser status

# Navigate to URL
mehr browser goto https://example.com

# Take screenshot
mehr browser screenshot --full-page

# Click element
mehr browser click #submit-button

# Type text
mehr browser type #email-input "user@example.com"

# Query DOM
mehr browser dom "a[href]"
```

See [CLI: browser](../cli/browser.md) for all options.
