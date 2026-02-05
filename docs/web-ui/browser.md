# Browser Control

The Browser Control Panel provides a web interface for automating Chrome during testing and authentication flows.

## Accessing Browser Control

From the navigation bar, click **Tools**, then select the **Browser** tab.

The Browser panel shows the Chrome connection status and lists all open tabs. You can navigate to URLs, take screenshots, click elements, type text, evaluate JavaScript, query the DOM, and access DevTools features (network, console, WebSocket, source, coverage).

For a complete overview of the Tools page, see [Tools](/web-ui/tools.md).

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

Enter the URL in the address field and click **Goto** to navigate, or **Cancel** to close the dialog.

### Screenshots

Capture screenshots of the current page:

The screenshot dialog lets you choose between **Full Page** or **Visible Only** capture, select the format (PNG or JPEG), and adjust the quality slider. Click **Capture** to take the screenshot.

Screenshots are saved to the configured screenshot directory.

### DOM Query

Inspect page elements using CSS selectors:

Enter a CSS selector in the text field and click **Query** to find matching elements. Results display the matched HTML elements below the input field.

### JavaScript Console

View console output:

The console panel displays browser console messages with timestamps and severity levels. Use the level filter to show All, Info, Warn, or Error messages. Click **Clear** to reset the log.

### Element Interaction

Click elements and type text:

Select an action (Click or Type), enter the element selector, and optionally provide text to type. Click **Execute** to perform the interaction.

## DevTools Monitoring

The Browser Control Panel includes deep DevTools inspection panels for monitoring network traffic, console logs, WebSocket frames, page source, CSS styles, and code coverage.

### Network Monitor

Capture HTTP requests and responses over a configurable duration:

Set the monitoring duration, optionally enable **Capture Body** to include request/response bodies, and click **Monitor** to start. Captured requests display with their methods, paths, and status codes.

- **Duration**: How long to monitor (1–30 seconds)
- **Capture Body**: Include request/response bodies (up to 1MB default)

### Console Logs

Listen for browser console messages with optional level filtering:

Set the listening duration, choose a level filter (All, Error, Warning, Info, or Log), and click **Listen** to capture console messages. Messages are color-coded by severity.

- **Level filter**: All, Error, Warning, Info, or Log
- Messages are color-coded by severity

### WebSocket Monitor

Monitor WebSocket frames sent and received:

Set the monitoring duration and click **Monitor** to capture WebSocket frames. Sent frames are marked with → and received frames with ←.

- **→** indicates sent frames, **←** indicates received frames

### Page Source

View the full HTML source or loaded JavaScript files:

Click **View Source** to see the full HTML of the current page, or **Scripts** to list all loaded JavaScript files with their source code.

- **View Source**: Full HTML of the current page
- **Scripts**: Lists all loaded JavaScript files with their source code

### CSS Inspector

Inspect computed and matched CSS styles for any element:

Enter a CSS selector, choose whether to show **Computed** styles (final resolved values) or **Matched** styles (CSS rules that applied), and click **Inspect** to view the results.

- **Computed**: Final resolved CSS property values
- **Matched**: CSS rules that matched the element (with selectors and origins)

### Code Coverage

Measure JavaScript and CSS code coverage:

Set the measurement duration, toggle **JS** and/or **CSS** coverage tracking, and click **Measure**. The results show used vs total bytes and list the files analyzed.

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

See [CLI: browser](/cli/browser.md) for all options and DevTools monitoring commands.
