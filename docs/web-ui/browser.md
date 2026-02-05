# Browser Control

The Browser Control Panel provides a web interface for automating Chrome during testing and authentication flows.

## Accessing Browser Control

From the navigation bar, open the **More** dropdown and click **Tools**. The Browser Control panel is located on the Tools page.

```
┌──────────────────────────────────────────────────────────────┐
│  Browser Automation                                          │
├──────────────────────────────────────────────────────────────┤
│  Chrome detected: chrome (port 9222)                         │
│                                                              │
│  Open Tabs:                                                  │
│  ┌────────────────────────────────────────────────────┐      │
│  │ 🌐 GitHub - valksor/go-mehrhof          [Active]    │      │
│  │ 🌐 Localhost:8080 - Health Endpoint                 │      │
│  │ 🌐 Google - "How to implement OAuth"                │      │
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

Before using browser control, Chrome must be running with remote debugging enabled on port 9222 (the default).

See [CLI: browser](/cli/browser.md) for platform-specific instructions on launching Chrome with remote debugging.

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

## DevTools Monitoring

The Browser Control Panel includes deep DevTools inspection panels for monitoring network traffic, console logs, WebSocket frames, page source, CSS styles, and code coverage.

### Network Monitor

Capture HTTP requests and responses over a configurable duration:

```
┌──────────────────────────────────────────────────────────────┐
│  Network Monitor                                             │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  Duration: [5__]s  ☑ Capture Body  [Monitor]                 │
│                                                              │
│  3 request(s) captured                                       │
│  GET /api/users                                  200         │
│  POST /api/login                                 401         │
│  GET /static/logo.png                            304         │
└──────────────────────────────────────────────────────────────┘
```

- **Duration**: How long to monitor (1–30 seconds)
- **Capture Body**: Include request/response bodies (up to 1MB default)

### Console Logs

Listen for browser console messages with optional level filtering:

```
┌──────────────────────────────────────────────────────────────┐
│  Console Logs                                                │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  Duration: [5__]s  Level: [All ▼]  [Listen]                  │
│                                                              │
│  [error]   Uncaught TypeError: Cannot read property          │
│  [warning] Deprecated API usage detected                     │
│  [log]     Page loaded successfully                          │
└──────────────────────────────────────────────────────────────┘
```

- **Level filter**: All, Error, Warning, Info, or Log
- Messages are color-coded by severity

### WebSocket Monitor

Monitor WebSocket frames sent and received:

```
┌──────────────────────────────────────────────────────────────┐
│  WebSocket Monitor                                           │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  Duration: [5__]s  [Monitor]                                 │
│                                                              │
│  → {"type":"subscribe","channel":"updates"}                  │
│  ← {"type":"ack","status":"subscribed"}                      │
│  ← {"type":"data","payload":{...}}                           │
└──────────────────────────────────────────────────────────────┘
```

- **→** indicates sent frames, **←** indicates received frames

### Page Source

View the full HTML source or loaded JavaScript files:

```
┌──────────────────────────────────────────────────────────────┐
│  Page Source                                                 │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  [View Source]  [Scripts]                                    │
│                                                              │
│  <!DOCTYPE html>                                             │
│  <html lang="en">                                            │
│    <head>...</head>                                          │
│    <body>...</body>                                          │
│  </html>                                                     │
└──────────────────────────────────────────────────────────────┘
```

- **View Source**: Full HTML of the current page
- **Scripts**: Lists all loaded JavaScript files with their source code

### CSS Inspector

Inspect computed and matched CSS styles for any element:

```
┌──────────────────────────────────────────────────────────────┐
│  CSS Inspector                                               │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  Selector: [h1.title_____________]                           │
│  ☑ Computed  ☐ Matched  [Inspect]                            │
│                                                              │
│  Computed Styles                                             │
│  color: rgb(0, 0, 0)                                         │
│  font-size: 24px                                             │
│  font-weight: 700                                            │
│  margin-bottom: 16px                                         │
└──────────────────────────────────────────────────────────────┘
```

- **Computed**: Final resolved CSS property values
- **Matched**: CSS rules that matched the element (with selectors and origins)

### Code Coverage

Measure JavaScript and CSS code coverage:

```
┌──────────────────────────────────────────────────────────────┐
│  Code Coverage                                               │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  Duration: [5__]s  ☑ JS  ☑ CSS  [Measure]                    │
│                                                              │
│  Coverage Summary                                            │
│  ┌──────────┐  ┌──────────┐                                  │
│  │ JS Used  │  │ JS Total │                                  │
│  │  45 KB   │  │  120 KB  │                                  │
│  └──────────┘  └──────────┘                                  │
│  JS Files (3): main.js, vendor.js, analytics.js              │
│  CSS Files (2): styles.css, theme.css                        │
└──────────────────────────────────────────────────────────────┘
```

- **Duration**: How long to collect coverage data
- **JS/CSS**: Toggle which types to track

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

If you see a "Chrome not detected" warning:

1. Start Chrome with remote debugging enabled (see [CLI: browser](/cli/browser.md) for instructions)
2. Verify the port matches (default: 9222)
3. Check if another process is using the port

### Connection Timeout

If you see a "Connection timeout" error:

1. Verify Chrome is running with remote debugging enabled
2. Check the remote debugging port matches your configuration
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
- [**CLI: browser**](/cli/browser.md) - CLI browser commands

---

## Also Available via CLI

Control Chrome automation from the command line for scripting or terminal workflows.

| Command | What It Does |
|---------|--------------|
| `mehr browser status` | Check browser connection status |
| `mehr browser goto <url>` | Navigate to a URL |
| `mehr browser screenshot` | Capture page screenshot |
| `mehr browser click <selector>` | Click an element |
| `mehr browser type <selector> <text>` | Type text into an element |
| `mehr browser dom <selector>` | Query DOM elements |
| `mehr browser network` | Monitor network traffic |
| `mehr browser console` | Listen for console logs |
| `mehr browser websocket` | Monitor WebSocket frames |
| `mehr browser source` | Get page HTML source |
| `mehr browser styles <selector>` | Inspect CSS styles |
| `mehr browser coverage` | Measure code coverage |

See [CLI: browser](/cli/browser.md) for all options and DevTools monitoring commands.
