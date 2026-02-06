# Authentication

Manage user authentication for network-accessible Web UI servers.

## When Authentication is Required

| Host Setting               | Authentication |
|----------------------------|----------------|
| `localhost` (default)      | ❌ Not required |
| `127.0.0.1`                | ❌ Not required |
| `0.0.0.0` (all interfaces) | ✅ **Required** |
| Specific IP address        | ✅ **Required** |

Authentication is mandatory when using `--host 0.0.0.0` or any non-localhost address.

## Setting Up Authentication

User management is performed via the command line before starting the server. The CLI provides commands to:

- Add users with passwords and roles
- List configured users
- Change passwords
- Modify user roles
- Remove users

**Valid roles:** `user` (full access), `viewer` (read-only)

See [CLI: serve](/cli/serve.md) for all authentication commands and examples.

## Starting Server with Authentication

1. **Add users** using the CLI authentication commands
2. **Start the server** with network access enabled

The server then requires authentication for all non-public endpoints.

## Credential Storage

User credentials are stored securely in your home directory.

**Security notes:**
- Passwords are hashed using bcrypt
- Plain text passwords are never stored
- The auth file is automatically protected with restricted permissions

## Login Process

When accessing an authenticated server, the login page displays the Mehrhof logo, a welcome message, username and password fields, and a **Login** button.

### Session Management

- **Session duration:** 7 days by default
- **Session cookies:** HTTP-only and secure (when using HTTPS)
- **Auto-logout:** Sessions expire after inactivity

## CSRF Protection

When authentication is enabled, Mehrhof uses **CSRF (Cross-Site Request Forgery)** protection to prevent unauthorized actions from malicious websites.

### How It Works

CSRF protection uses the Synchronizer Token Pattern:

1. On login, the server generates a unique CSRF token per session
2. The token is returned in the login response and available via `GET /api/v1/auth/csrf`
3. All state-changing requests (POST, PUT, DELETE) must include the token in the `X-Csrf-Token` header
4. Requests without a valid token receive **HTTP 403 Forbidden**

### When CSRF Is Active

| Server Mode                | CSRF Enforced                             |
|----------------------------|-------------------------------------------|
| `localhost` (default)      | ❌ No — localhost mode skips CSRF          |
| `--host 0.0.0.0` with auth | ✅ Yes — all POST/PUT/DELETE require token |

CSRF is automatically disabled in localhost mode because cross-site attacks cannot target localhost.

### Getting a CSRF Token

The CSRF token is provided:
- **On login:** Returned in the login response
- **For existing sessions:** Available via the CSRF endpoint

For API integration details, see [REST API Reference](/reference/rest-api.md).

### Web UI Handling

The Web UI handles CSRF automatically:

- **API requests**: Token automatically included via the React API client layer
- **Token refresh**: Fetched on page load and cached for the session

No manual configuration is needed for Web UI users.

### IDE Plugin Handling

Both the VS Code extension and JetBrains plugin include CSRF infrastructure:

- Session cookies are automatically extracted from responses
- CSRF tokens are sent via `X-Csrf-Token` header on POST requests
- In localhost mode (default for IDE plugins), CSRF is not enforced

### Endpoints Exempt from CSRF

| Endpoint                 | Reason                                             |
|--------------------------|----------------------------------------------------|
| `GET`, `HEAD`, `OPTIONS` | Safe methods — no state changes                    |
| `/api/v1/auth/login`     | No session exists yet                              |
| `/api/v1/webhooks/*`     | Provider-specific authentication (webhook secrets) |

## Authentication Behavior

### Public Endpoints

These endpoints work without authentication:

| Endpoint  | Access                     |
|-----------|----------------------------|
| `/health` | Public                     |
| `/login`  | Public (login page itself) |

### Protected Endpoints

All other endpoints require authentication:

| Endpoint        | Auth Required |
|-----------------|---------------|
| `/` (dashboard) | ✅ Yes         |
| `/api/v1/*`     | ✅ Yes         |
| `/settings`     | ✅ Yes         |
| `/tools`        | ✅ Yes         |
| `/history`      | ✅ Yes         |

## Managing Users

User management is performed via the command line. There is no settings panel for user management in the Web UI.

See [CLI: serve](/cli/serve.md) for all authentication commands.

## Security Best Practices

### Password Security

1. **Use strong passwords** - At least 12 characters with mixed types
2. **Don't reuse passwords** - Unique passwords for each installation
3. **Change regularly** - Update passwords periodically

### User Management

1. **Principle of least privilege** - Only create necessary users
2. **Remove unused accounts** - Delete users who no longer need access
3. **Audit access** - Review user list regularly

### File Permissions

The auth file is automatically created with restricted permissions to protect credentials.

## Read-Only Users

Mehrhof supports a **viewer** role for users who need read-only access to the Web UI. This is useful for stakeholders, managers, or anyone who needs visibility into tasks and workflow progress without the ability to make changes.

### What Viewers Can Access

Viewers have **full read access** to all information:

| Access         | Description                                  |
|----------------|----------------------------------------------|
| Dashboard      | View task status, workflow state, statistics |
| Specifications | Read all task specifications and progress    |
| History        | Browse session history and conversation logs |
| Logs           | View agent output and execution logs         |
| Settings       | View configuration values (read-only)        |
| Projects       | View project plans and task breakdowns       |

### What Viewers Cannot Do

Viewers are **blocked from all write operations**:

| Operation           | Blocked                                 |
|---------------------|-----------------------------------------|
| Starting workflows  | ❌ Plan, Implement, Review commands      |
| Modifying workflows | ❌ Continue, Answer, Abandon actions     |
| Submitting tasks    | ❌ Quick tasks and project submissions   |
| Changing settings   | ❌ Workspace configuration modifications |
| Running scans       | ❌ Security and quality scans            |
| Clearing memory     | ❌ Memory cache operations               |

### Creating and Managing Viewers

Viewers are created and managed via the CLI. See [CLI: serve](/cli/serve.md) for commands to add viewers and modify user roles.

### Viewer Experience

When a viewer logs in:
- The dashboard displays normally with all information visible
- Action buttons (Plan, Implement, Review, etc.) are **hidden**
- Forms are shown as **read-only** (no submit buttons)
- API write endpoints return **403 Forbidden**

### HTTPS Considerations

When using authentication over the internet:
- Use HTTPS (reverse proxy or Cloudflare Tunnel)
- Strong passwords become even more important
- Consider shorter session duration

## Rate Limiting

When authentication is enabled, the server enforces per-IP rate limiting to protect against abuse:

| Endpoint Type                         | Limit        | Window     |
|---------------------------------------|--------------|------------|
| General API (`/api/v1/*`)             | 120 requests | Per minute |
| Auth endpoints (`/api/v1/auth/login`) | 10 requests  | Per minute |

When rate limited, the server returns **HTTP 429 Too Many Requests**. Wait and retry.

Rate limiting is automatically disabled in localhost mode.

## Troubleshooting

### "Authentication Required" Error

This appears when accessing a network server without users configured.

**Solution:** Add users via the CLI before starting the server. See [CLI: serve](/cli/serve.md).

### Login Not Working

If you can't log in:
1. Verify username is correct
2. Reset password via the CLI
3. Check that the auth file exists in your home directory

### Lost Password

Passwords can be reset via the CLI. See [CLI: serve](/cli/serve.md) for the password reset command.

## Next Steps

- [**Remote Access**](remote-access.md) - Set up remote access
- [**CLI: serve**](/cli/serve.md) - Server command options

---

## Also Available via CLI

All authentication management commands are covered above since user setup requires the command line. For the complete CLI reference including additional flags and options, see [CLI: serve](/cli/serve.md).
