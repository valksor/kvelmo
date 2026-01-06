# browser

Browser automation commands for web-based testing, scraping, and authentication flows.

## Overview

Mehrhof includes Chrome browser automation capabilities using the [Rod](https://github.com/go-rod/rod) library. This enables:
- Web-based testing and verification
- Screen captures for documentation
- DOM interaction (clicking, typing, JavaScript evaluation)
- Network and console monitoring
- Authentication flow handling

## Configuration

Browser automation is disabled by default. Enable it in `.mehrhof/config.yaml`:

```yaml
browser:
  enabled: true                  # Enable browser automation
  headless: false                # Show browser window (false = visible, true = background)
  port: 0                        # 0 = random isolated browser, 9222 = existing Chrome
  timeout: 30                    # Operation timeout in seconds
  screenshot_dir: ".mehrhof/screenshots"
```

### Session Isolation

When `port: 0`, Mehrhof launches an isolated Chrome instance on a random port:
- Won't interfere with your personal Chrome browser
- Session tracked in `.mehrhof/browser.json`
- Automatically cleaned up when workflow completes

To connect to an existing Chrome instance, use `port: 9222` and launch Chrome with:
```bash
google-chrome --remote-debugging-port=9222
```

## Commands

### browser status

Check browser connection status:

```bash
mehr browser status
```

Example output:
```
Browser: Connected
Session: PID 12345 on port 9223
Headless: No
```

### browser tabs

List all open browser tabs:

```bash
mehr browser tabs
```

Example output:
```
Open tabs:
  ABC123  Example Domain  https://example.com
  DEF456  Google         https://google.com
```

### browser goto

Open a URL in a new tab:

```bash
mehr browser goto <url>
```

Example:
```bash
mehr browser goto https://example.com
```

### browser screenshot

Capture a screenshot of the current tab:

```bash
mehr browser screenshot [--full-page] [--format=png|jpeg] [--quality=80]
```

Flags:
- `--full-page` - Capture entire scrollable page (default: viewport only)
- `--format` - Image format: `png` (default) or `jpeg`
- `--quality` - JPEG quality 1-100 (default: 80)

Examples:
```bash
# Capture viewport
mehr browser screenshot

# Capture full page
mehr browser screenshot --full-page

# Capture as JPEG with quality 90
mehr browser screenshot --format=jpeg --quality=90
```

Screenshots are saved to `.mehrhof/screenshots/`.

### browser click

Click an element using CSS selector:

```bash
mehr browser click <selector>
```

Examples:
```bash
# Click button with ID
mehr browser click "#submit-btn"

# Click first submit button
mehr browser click "button[type='submit']"

# Click element with class
mehr browser click ".cta-button"
```

### browser type

Type text into an input field:

```bash
mehr browser type <selector> <text> [--clear]
```

Flags:
- `--clear` - Clear existing text before typing (default: append)

Examples:
```bash
# Type into username field
mehr browser type "#username" "john@example.com"

# Clear and type password
mehr browser type "#password" "secret123" --clear
```

### browser eval

Evaluate JavaScript in the page context:

```bash
mehr browser eval <expression>
```

Examples:
```bash
# Get page title
mehr browser eval "document.title"

# Scroll to bottom
mehr browser eval "window.scrollTo(0, document.body.scrollHeight)"

# Check if element exists
mehr browser eval "document.querySelector('#my-element') !== null"
```

### browser console

Monitor console logs for a duration:

```bash
mehr browser console [--duration=5]
```

Flags:
- `--duration` - Listen duration in seconds (default: 5)

Example output:
```
Listening to console logs for 5 seconds...
[log] Page loaded
[warn] Deprecated API used
[error] Failed to load resource
```

### browser network

Monitor network requests for a duration:

```bash
mehr browser network [--duration=5]
```

Flags:
- `--duration` - Listen duration in seconds (default: 5)

Example output:
```
Listening to network requests for 5 seconds...
[GET] https://example.com/api/users - 200 OK
[POST] https://example.com/api/login - 201 Created
[GET] https://example.com/favicon.ico - 404 Not Found
```

## Agent Integration

To enable AI agents to use browser automation, add instructions to your config:

```yaml
agent:
  instructions: |
    Browser automation is available for web-based tasks:
    - Navigate to URLs and take screenshots
    - Interact with DOM elements (click, type, evaluate JavaScript)
    - Monitor network requests and console logs
    - Handle authentication flows

    When implementing web features, include testing steps such as:
    - "Navigate to http://localhost:8080 and verify the page loads"
    - "Check that the form submission works correctly"
    - "Verify the error message displays for invalid input"

  steps:
    implementing:
      instructions: |
        After implementing web features, provide manual testing steps.
        Include specific URLs to visit and what to verify.
```

## Examples

### Test a Web Application

```bash
# Enable browser in config
cat >> .mehrhof/config.yaml <<EOF
browser:
  enabled: true
  headless: false
EOF

# Start your web server
npm run dev &

# Open the application
mehr browser goto http://localhost:3000

# Take a screenshot
mehr browser screenshot --full-page

# Fill out a form
mehr browser type "#email" "test@example.com" --clear
mehr browser type "#password" "secret123"
mehr browser click "button[type='submit']"

# Monitor console for errors
mehr browser console --duration=10
```

### Authentication Testing

```bash
# Open login page
mehr browser goto https://example.com/login

# Fill credentials
mehr browser type "#username" "$TEST_USER" --clear
mehr browser type "#password" "$TEST_PASS" --clear

# Submit and monitor network
mehr browser click "button[type='submit']"
mehr browser network --duration=5
```

### Screenshot Documentation

```bash
# Navigate through application and capture screenshots
# Screenshots are saved to .mehrhof/screenshots/ with timestamps

mehr browser goto https://example.com
mehr browser screenshot --full-page

mehr browser goto https://example.com/features
mehr browser screenshot --full-page

mehr browser goto https://example.com/pricing
mehr browser screenshot --full-page
```

## Troubleshooting

### Chrome Not Found

If Chrome is not installed, install it:

**Linux:**
```bash
sudo apt-get update
sudo apt-get install -y wget gnupg
wget -q -O - https://dl-ssl.google.com/linux/linux_signing_key.pub | sudo apt-key add -
sudo sh -c 'echo "deb [arch=amd64] http://dl.google.com/linux/chrome/deb/ stable main" >> /etc/apt/sources.list.d/google.list'
sudo apt-get update
sudo apt-get install -y google-chrome-stable
```

**macOS:**
```bash
brew install --cask google-chrome
```

### Port Already in Use

If you see "port already in use", either:
1. Use a different port in config: `port: 9224`
2. Or set `port: 0` to use a random available port

### Session Stale

If the browser session becomes stale, delete the session file:
```bash
rm .mehrhof/browser.json
```

The next browser command will launch a fresh Chrome instance.

## See Also

- [Configuration Guide](../configuration/index.md) - Browser settings in config.yaml
- [Agents Overview](../agents/index.md) - Agent integration with browser tools
- [Storage Structure](../reference/storage.md) - Session file location
