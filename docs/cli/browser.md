# mehr browser

Browser automation commands for web-based testing, scraping, and authentication flows.

## Synopsis

```bash
mehr browser <subcommand> [flags]
```

## Description

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
  cookie_profile: "default"      # Cookie profile name (default: "default")
  cookie_auto_load: true         # Auto-load cookies on connect (default: true)
  cookie_auto_save: true         # Auto-save cookies on disconnect (default: true)
  cookie_dir: ""                 # Custom cookie directory (default: ~/.valksor/mehrhof/)
```

### Session Isolation

When `port: 0`, Mehrhof launches an isolated Chrome instance on a random port:
- Won't interfere with your personal Chrome browser
- Session tracked in `.mehrhof/browser.json`
- Automatically cleaned up when workflow completes

**Automatic Stale Session Recovery**: If a browser process becomes unresponsive (hung or zombie), Mehrhof automatically detects this by checking the Chrome DevTools endpoint. Unresponsive sessions are terminated and cleaned up, then a fresh browser is launched. This eliminates the need to manually delete session files.

To connect to an existing Chrome instance, use `port: 9222` and launch Chrome with:
```bash
google-chrome --remote-debugging-port=9222
```

### Cookie Persistence

Browser sessions can be persisted across runs using **named cookie profiles**. This enables:

- **Session persistence**: Stay logged in across browser sessions
- **Multiple accounts**: Use different profiles for personal vs work accounts (e.g., `default`, `work-github`, `client-a`)
- **Cross-project usage**: Cookie profiles are stored globally in `~/.valksor/mehrhof/`, not per-workspace

Cookie storage location:
```
~/.valksor/mehrhof/
  ├── cookies-default.json        # Default profile
  ├── cookies-work-github.json    # Work GitHub account
  └── cookies-client-a.json       # Client-specific profile
```

When `cookie_auto_load` is enabled, cookies are automatically restored when the browser connects. When `cookie_auto_save` is enabled, cookies are saved when the browser disconnects.

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

### browser cookies export

Export current browser cookies to a JSON file:

```bash
mehr browser cookies export [--profile=<name>] [--output=<path>]
```

Flags:
- `--profile` - Cookie profile to export (default: "default", or from `--cookie-profile` flag)
- `--output` - Output file path (default: `~/.valksor/mehrhof/cookies-<profile>.json`)

Examples:
```bash
# Export default profile to default location
mehr browser cookies export

# Export work profile to custom path
mehr browser cookies export --profile work-github --output /tmp/work-cookies.json

# Export using profile flag set at command level
mehr browser --cookie-profile client-a cookies export
```

### browser cookies import

Import cookies from a JSON file to the browser:

```bash
mehr browser cookies import [--profile=<name>] [--file=<path>]
```

Flags:
- `--profile` - Cookie profile to import to (default: "default", or from `--cookie-profile` flag)
- `--file` - Input file path (default: `~/.valksor/mehrhof/cookies-<profile>.json`)

Examples:
```bash
# Import default profile from default location
mehr browser cookies import

# Import cookies from file to work profile
mehr browser cookies import --profile work-github --file /tmp/work-cookies.json

# Import using profile flag set at command level
mehr browser --cookie-profile client-a cookies import
```

## Using Cookie Profiles

Use the `--cookie-profile` flag to specify which cookie profile to use for a session:

```bash
# Use default profile
mehr browser goto https://github.com

# Use work profile
mehr browser --cookie-profile work-github goto https://github.com

# Use client profile
mehr browser --cookie-profile client-a goto https://github.com
```

This enables maintaining separate sessions for different accounts on the same domain.

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
    - Manage browser cookies (get, set, export, import)

    When implementing web features, include testing steps such as:
    - "Navigate to http://localhost:8080 and verify the page loads"
    - "Check that the form submission works correctly"
    - "Verify the error message displays for invalid input"

    Cookie profiles are available for session persistence. Use cookies to:
    - Stay logged in across browser sessions
    - Test with different user accounts
    - Maintain authentication state between test runs

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

### Managing Multiple Accounts with Cookie Profiles

```bash
# Login with personal GitHub account
mehr browser --cookie-profile personal goto https://github.com
# (complete login flow in browser)
# Cookies are auto-saved to ~/.valksor/mehrhof/cookies-personal.json

# Login with work GitHub account
mehr browser --cookie-profile work-github goto https://github.com
# (complete login flow with work credentials)
# Cookies are auto-saved to ~/.valksor/mehrhof/cookies-work-github.json

# Verify personal account
mehr browser --cookie-profile personal goto https://github.com
mehr browser screenshot --full-page

# Verify work account
mehr browser --cookie-profile work-github goto https://github.com
mehr browser screenshot --full-page

# Export cookies for backup
mehr browser cookies export --profile work-github --output /tmp/backup-cookies.json

# Import cookies to another machine
mehr browser cookies import --profile work-github --file /tmp/backup-cookies.json
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

Mehrhof automatically detects and cleans up stale browser sessions. When a browser process is alive but unresponsive (hung or zombie state), the next browser command will:

1. Detect the unresponsive endpoint
2. Terminate the stuck process
3. Clean up the session file
4. Launch a fresh Chrome instance

You can see this in the logs:
```
WARN browser process unresponsive, cleaning up pid=12345 port=9222
INFO launching isolated browser...
```

If automatic cleanup fails, you can manually reset:
```bash
# Kill any stuck Chrome processes
pkill -f 'chrome.*remote-debugging-port'

# Remove session file
rm .mehrhof/browser.json
```

The next browser command will launch a fresh Chrome instance.

## See Also

- [Configuration Guide](../configuration/index.md) - Browser settings in config.yaml
- [Agents Overview](../agents/index.md) - Agent integration with browser tools
- [Storage Structure](../reference/storage.md) - Session file location
