# Authentication

Manage user authentication for network-accessible Web UI servers.

## When Authentication is Required

| Host Setting | Authentication |
|--------------|----------------|
| `localhost` (default) | ❌ Not required |
| `127.0.0.1` | ❌ Not required |
| `0.0.0.0` (all interfaces) | ✅ **Required** |
| Specific IP address | ✅ **Required** |

Authentication is mandatory when using `--host 0.0.0.0` or any non-localhost address.

## Setting Up Authentication

### Add Users

Before starting a network-accessible server, add users:

```bash
# Add a user
mehr serve auth add admin mypassword

# Add multiple users
mehr serve auth add admin secretpassword
mehr serve auth add developer devpass123
mehr serve auth add viewer viewpass
```

### List Users

View all configured users:

```bash
mehr serve auth list
```

Output:
```
Configured users:
  • admin (created: 2025-01-15 10:30:00)
  • developer (created: 2025-01-15 10:31:00)
  • viewer (created: 2025-01-15 10:32:00)
```

### Change Passwords

Update a user's password:

```bash
mehr serve auth passwd admin newpassword
```

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

## Authentication Behavior

### Public Endpoints

These endpoints work without authentication:

| Endpoint | Access |
|----------|--------|
| `/health` | Public |
| `/login` | Public (login page itself) |

### Protected Endpoints

All other endpoints require authentication:

| Endpoint | Auth Required |
|----------|---------------|
| `/` (dashboard) | ✅ Yes |
| `/api/v1/*` | ✅ Yes |
| `/settings` | ✅ Yes |
| `/browser` | ✅ Yes |
| `/history` | ✅ Yes |

## Managing Auth via CLI

### All Auth Commands

```bash
# Add a user
mehr serve auth add <username> <password>

# List all users
mehr serve auth list

# Change password
mehr serve auth passwd <username> <new-password>

# Remove a user
mehr serve auth remove <username>
```

### Examples

```bash
# Add admin user
mehr serve auth add admin securepassword123

# Change admin password
mehr serve auth passwd admin newsecurepassword

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

### HTTPS Considerations

When using authentication over the internet:
- Use HTTPS (reverse proxy or Cloudflare Tunnel)
- Strong passwords become even more important
- Consider shorter session duration

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
# Add user
mehr serve auth add admin password

# List users
mehr serve auth list

# Change password
mehr serve auth passwd admin newpass

# Remove user
mehr serve auth remove admin
```

See [CLI: serve](../cli/serve.md) for all options.
