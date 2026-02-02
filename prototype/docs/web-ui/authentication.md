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

### Add Users

Before starting a network-accessible server, add users:

```bash
# Add a full-access user
mehr serve auth add admin mypassword

# Add a read-only viewer
mehr serve auth add stakeholder viewpass --role viewer

# Add multiple users
mehr serve auth add admin secretpassword
mehr serve auth add developer devpass123
```

### List Users

View all configured users:

```bash
mehr serve auth list
```

Output:
```
Configured users:
USERNAME    ROLE    CREATED
admin       user    2025-01-15 10:30
developer   user    2025-01-15 10:31
stakeholder viewer   2025-01-15 10:32
```

### Change Passwords

Update a user's password:

```bash
mehr serve auth passwd admin newpassword
```

### Change User Roles

Modify a user's role after creation:

```bash
# Promote viewer to full user
mehr serve auth role stakeholder user

# Demote user to viewer
mehr serve auth role contractor viewer
```

**Valid roles:** `user`, `viewer`

### Remove Users

Delete a user:

```bash
mehr serve auth remove developer
```

## Starting Server with Authentication

### Step 1: Add Users

```bash
mehr serve auth add admin mypassword
```

### Step 2: Start Server

```bash
mehr serve --host 0.0.0.0 --port 3000
```

The server now requires authentication for all non-public endpoints.

## Credential Storage

User credentials are stored at `~/.valksor/mehrhof/auth.yaml`:

```yaml
version: "1"
users:
  admin:
    username: admin
    password_hash: "$2a$10$..."  # bcrypt hash
    created_at: "2025-01-15T10:30:00Z"
  developer:
    username: developer
    password_hash: "$2a$10$..."
    created_at: "2025-01-15T10:31:00Z"
```

**Security notes:**
- Passwords are hashed using bcrypt
- Plain text passwords are never stored
- The auth file should be protected (chmod 600)

## Login Process

When accessing an authenticated server:

```
┌──────────────────────────────────────────────────────────────┐
│  Mehrhof Login                                              │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│                   [Mehrhof Logo]                             │
│                                                              │
│  Welcome to Mehrhof                                          │
│  Please log in to continue                                  │
│                                                              │
│  ┌────────────────────────────────────────────┐             │
│  │ Username: [____________________________]  │             │
│  │                                          │             │
│  │ Password: [____________________________]  │             │
│  │                                          │             │
│  │            [Login]                      │             │
│  └────────────────────────────────────────────┘             │
│                                                              │
└──────────────────────────────────────────────────────────────┘
```

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

| Server Mode | CSRF Enforced |
|-------------|---------------|
| `localhost` (default) | ❌ No — localhost mode skips CSRF |
| `--host 0.0.0.0` with auth | ✅ Yes — all POST/PUT/DELETE require token |

CSRF is automatically disabled in localhost mode because cross-site attacks cannot target localhost.

### Getting a CSRF Token

#### From Login Response

The login endpoint returns the CSRF token in the JSON response:

```bash
curl -X POST http://your-server/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username": "admin", "password": "secret"}'
```

Response:
```json
{
  "status": "ok",
  "csrf_token": "abc123..."
}
```

#### From CSRF Endpoint

For existing sessions, fetch a fresh token:

```bash
curl http://your-server/api/v1/auth/csrf \
  -H "Cookie: mehr_session=your-session-cookie"
```

Response:
```json
{
  "csrf_token": "abc123..."
}
```

### Using the Token

Include the token in the `X-Csrf-Token` header on all state-changing requests:

```bash
curl -X POST http://your-server/api/v1/workflow/plan \
  -H "Cookie: mehr_session=your-session-cookie" \
  -H "X-Csrf-Token: abc123..." \
  -H "Content-Type: application/json"
```

### Web UI Handling

The Web UI handles CSRF automatically:

- **HTMX requests**: Token injected via `htmx:configRequest` event
- **JavaScript fetch()**: Uses `csrfFetch()` wrapper that adds the header
- **Token refresh**: Fetched on page load and cached for the session

No manual configuration is needed for Web UI users.

### IDE Plugin Handling

Both the VS Code extension and JetBrains plugin include CSRF infrastructure:

- Session cookies are automatically extracted from responses
- CSRF tokens are sent via `X-Csrf-Token` header on POST requests
- In localhost mode (default for IDE plugins), CSRF is not enforced

### Endpoints Exempt from CSRF

| Endpoint | Reason |
|----------|--------|
| `GET`, `HEAD`, `OPTIONS` | Safe methods — no state changes |
| `/api/v1/auth/login` | No session exists yet |
| `/api/v1/webhooks/*` | Provider-specific authentication (webhook secrets) |

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
| `/browser`      | ✅ Yes         |
| `/history`      | ✅ Yes         |

## Managing Auth via CLI

### All Auth Commands

```bash
# Add a user
mehr serve auth add <username> <password> [--role <role>]

# List all users
mehr serve auth list

# Change password
mehr serve auth passwd <username> <new-password>

# Change user role
mehr serve auth role <username> <role>

# Remove a user
mehr serve auth remove <username>
```

### Examples

```bash
# Add admin user
mehr serve auth add admin securepassword123

# Add read-only viewer
mehr serve auth add stakeholder viewpass123 --role viewer

# Change admin password
mehr serve auth passwd admin newsecurepassword

# Promote viewer to user
mehr serve auth role stakeholder user

# List users to verify
mehr serve auth list

# Remove a user
mehr serve auth remove olduser
```

## Web UI Auth Management

In the Web UI, authentication is managed through the login page. There is no settings panel for user management—use the CLI for that.

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

Protect the auth file:
```bash
chmod 600 ~/.valksor/mehrhof/auth.yaml
```

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

### Creating a Viewer

```bash
# Add a new viewer
mehr serve auth add stakeholder viewpass123 --role viewer
```

### Modifying User Roles

Change an existing user's role:

```bash
# Promote viewer to full user
mehr serve auth role stakeholder user

# Demote user to viewer
mehr serve auth role contractor viewer
```

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

| Endpoint Type | Limit | Window |
|---------------|-------|--------|
| General API (`/api/v1/*`) | 120 requests | Per minute |
| Auth endpoints (`/api/v1/auth/login`) | 10 requests | Per minute |

When rate limited, the server returns **HTTP 429 Too Many Requests**. Wait and retry.

Rate limiting is automatically disabled in localhost mode.

## Troubleshooting

### "Authentication Required" Error

You see this when accessing `0.0.0.0` without users configured:

```
Error: Authentication required for non-localhost access
Configure users with: mehr serve auth add <username> <password>
```

**Solution:** Add users before starting the server

### Login Not Working

If you can't log in:
1. Verify username is correct
2. Reset password with `mehr serve auth passwd`
3. Check auth file exists at `~/.valksor/mehrhof/auth.yaml`

### Lost Password

If you forget your password:
```bash
# Reset it with passwd command
mehr serve auth pass admin newpassword
```

## Next Steps

- [**Remote Access**](remote-access.md) - Set up remote access
- [**CLI: serve**](../cli/serve.md) - Server command options

## CLI Equivalent

```bash
# Add user (default: user role)
mehr serve auth add admin password

# Add viewer
mehr serve auth add stakeholder viewpass --role viewer

# List users
mehr serve auth list

# Change password
mehr serve auth passwd admin newpass

# Change role
mehr serve auth role stakeholder user

# Remove user
mehr serve auth remove admin
```

See [CLI: serve](../cli/serve.md) for all options.
