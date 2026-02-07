# Accessibility

Mehrhof Web UI includes built-in accessibility patterns so teams can use workflow features with keyboard navigation and assistive technologies.

## What Is Included

| Area                   | Accessibility Support                                                                          |
|------------------------|------------------------------------------------------------------------------------------------|
| **Page Navigation**    | A skip link moves focus directly to main content                                               |
| **Dialogs and Modals** | Dialogs expose proper ARIA roles, trap focus while open, and close with `Esc`                  |
| **Live Updates**       | Workflow notifications are announced to screen readers with polite or assertive priority       |
| **Tabs and Panels**    | Tab interfaces include ARIA tab semantics and keyboard-friendly focus behavior                 |
| **Controls and Icons** | Interactive controls provide accessible names; decorative icons are hidden from assistive tech |

## Keyboard Navigation

You can operate key UI flows without a mouse:

- Use `Tab` and `Shift+Tab` to move through controls
- Use the **Skip to main content** link at the top of the page
- Use `Esc` to close open dialog windows
- In tabbed interfaces, move focus between tabs and open a panel from the keyboard

## Screen Reader Behavior

The UI includes live regions for status and alert announcements. This helps users hear important workflow events as they happen, including:

- Task completion
- Workflow errors
- Pending questions that require user input

## Notification Accessibility

The notification center is keyboard-operable and announces new items automatically. Users can:

- Open notifications from the header
- Mark items as read
- Dismiss individual notifications
- Clear all notifications

## Accessibility Scope

Accessibility support applies to core Web UI workflows such as dashboard actions, task dialogs, settings sections, and tools panels.

---

## Also Available via CLI

Prefer terminal workflows? CLI status output includes text state prefixes (for example, `[P]` for Planning) to avoid relying on color alone.

See [CLI: status](/cli/status.md) for details.
