# Browser Configuration

Browser automation configuration for web-based testing, scraping, and authentication flows.

## Settings

```yaml
browser:
  enabled: true                    # Enable browser automation
  headless: false                  # Show browser window (false = visible, true = background)
  port: 0                          # 0 = random isolated browser, 9222 = existing Chrome
  timeout: 30                      # Operation timeout in seconds
  screenshot_dir: ".mehrhof/screenshots"
  cookie_profile: "default"        # Cookie profile name (default: "default")
  cookie_auto_load: true           # Auto-load cookies on connect (default: true)
  cookie_auto_save: true           # Auto-save cookies on disconnect (default: true)
  cookie_dir: ""                   # Custom cookie directory (default: ~/.valksor/mehrhof/)
```

| Setting | Default | Description |
|---------|---------|-------------|
| `enabled` | `false` | Enable browser automation |
| `headless` | `false` | Run browser in headless mode |
| `port` | `0` | CDP port (0 = random isolated, 9222 = existing Chrome) |
| `timeout` | `30` | Operation timeout in seconds |
| `screenshot_dir` | `.mehrhof/screenshots` | Directory for screenshots |
| `cookie_profile` | `"default"` | Cookie profile name for session persistence |
| `cookie_auto_load` | `true` | Auto-load cookies on browser connect |
| `cookie_auto_save` | `true` | Auto-save cookies on browser disconnect |
| `cookie_dir` | `""` | Custom cookie storage directory (default: `~/.valksor/mehrhof/`) |

## Session Management

Browser sessions are tracked in `.mehrhof/browser.json`. Key behaviors:

- **Automatic reuse**: If a browser session is still running and responsive, it will be reused
- **Stale session recovery**: If a browser process is alive but unresponsive (hung/zombie), it is automatically terminated and a fresh browser is launched
- **Cleanup on finish**: Sessions are cleaned up when the workflow completes

## Cookie Profiles

Browser sessions can be persisted using named cookie profiles, enabling:

- **Session persistence**: Stay logged in across browser sessions
- **Multiple accounts**: Use different profiles for personal vs work accounts
- **Cross-project usage**: Cookies stored globally in `~/.valksor/mehrhof/`

Example profiles:
```bash
# Use default profile
mehr browser goto https://github.com

# Use work profile
mehr browser --cookie-profile work-github goto https://github.com
```

Cookies are stored as:
```
~/.valksor/mehrhof/
  ├── cookies-default.json        # Default profile
  ├── cookies-work-github.json    # Work GitHub account
  └── cookies-client-a.json       # Client-specific profile
```

## See Also

- [Browser Commands](../cli/browser.md) - Complete browser automation documentation
- [Configuration Overview](index.md) - All configuration options
