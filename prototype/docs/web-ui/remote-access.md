# Remote Access

Access the Mehrhof Web UI from anywhere using SSH tunnels, Cloudflare tunnels, or direct network access.

## Access Options

| Method | Use Case | Difficulty |
|--------|----------|------------|
| **SSH Tunnel** | Secure access without open ports | Easy |
| **Tailscale** | Mesh VPN access | Medium |
| **Cloudflare Tunnel** | Public URL without port forwarding | Easy |
| **Direct Binding** | LAN/exposed server | Easy (requires auth) |

## Option 1: SSH Tunnel

Securely access the Web UI from your local machine through an SSH tunnel.

### From Local to Remote

Access a remote server's Web UI from your local machine:

```bash
# Create tunnel
ssh -L 8080:localhost:3000 user@remote-server

# Open browser to
http://localhost:8080
```

Your local port 8080 forwards to the remote server's localhost:3000.

### From Remote to Local

Expose your local Web UI to a remote server:

```bash
# Create reverse tunnel
ssh -R 8080:localhost:3000 user@remote-server

# Access from remote server at
http://localhost:8080
```

### Quick Tunnel Instructions

Use the built-in helper:

```bash
mehr serve --tunnel-info
```

Output:
```
SSH Tunnel Instructions:
  Access remote serve from your local machine (-L flag):
    ssh -L 8080:localhost:3000 user@remote-server
    Then open: http://localhost:8080 on YOUR local machine

  Access local serve from remote server (-R flag):
    ssh -R 8080:localhost:3000 user@remote-server
    Then open: http://localhost:8080 on THE REMOTE server
```

**Note:** This flag exits after showing instructions—it doesn't start the server.

## Option 2: Tailscale

Use Tailscale mesh VPN for secure access to your Web UI.

### Setup

1. Install Tailscale on both machines
2. Log in to each machine
3. Start the server with network binding:

```bash
mehr serve --host 0.0.0.0 --port 3000
```

4. Access via Tailscale IP:
```
http://100.x.x.x:3000
```

### Benefits

- **No port forwarding** - Works through NAT
- **Encrypted** - All traffic is secure
- **Built-in auth** - Tailscale handles authentication
- **Easy** - Just install and log in

## Option 3: Cloudflare Tunnel

Use Cloudflare Tunnel to expose your Web UI publicly without port forwarding.

### Setup

1. Start the Mehrhof server:
```bash
mehr serve --port 3000
```

2. In another terminal, start Cloudflare tunnel:
```bash
cloudflared tunnel --url http://localhost:3000
```

3. Cloudflare provides a public URL:
```
https://random-name.trycloudflare.com
```

### Benefits

- **Free** - No cost for basic usage
- **No port forwarding** - Works through NAT
- **HTTPS** - Automatic SSL certificate
- **DDoS protection** - Cloudflare shields your server

## Option 4: Direct Network Binding

Bind to all network interfaces for LAN or internet access.

### Basic Setup

```bash
# Bind to all interfaces
mehr serve --host 0.0.0.0 --port 3000
```

### ⚠️ Authentication Required

When using `--host 0.0.0.0` or any non-localhost address, authentication is **required**.

First, set up authentication:

```bash
# Add users
mehr serve auth add admin mypassword
mehr serve auth add developer devpass123

# List users
mehr serve auth list
```

Then start the server:

```bash
mehr serve --host 0.0.0.0 --port 3000
```

Access from any device on your network:
```
http://192.168.1.100:3000
```

You'll be prompted to log in with the credentials you created.

See [Authentication](authentication.md) for details on user management.

## Access Comparison

| Method | Public URL | Auth Needed | Port Forwarding |
|--------|------------|-------------|-----------------|
| **Default** | ❌ No | ❌ No | ❌ No |
| **SSH Tunnel** | ❌ No | ❌ No | ❌ No |
| **Tailscale** | ❌ No | ✅ Yes (Tailscale) | ❌ No |
| **Cloudflare** | ✅ Yes | ❌ No (optional) | ❌ No |
| **Direct (0.0.0.0)** | ⚠️ LAN | ✅ Yes | ❌ No |

## Security Best Practices

### Always Use Authentication

When exposing the Web UI beyond localhost:
- ✅ **Use authentication** - Required for non-localhost
- ✅ **Use SSH tunnels** - Secure without extra auth
- ✅ **Use VPN** - Tailscale or similar
- ❌ **Open ports without auth** - Never do this

### HTTPS Considerations

For production use:
- Use a reverse proxy (nginx, Caddy) for SSL termination
- Or use Cloudflare Tunnel for automatic HTTPS
- Configure strong passwords

### Firewall Rules

If binding to 0.0.0.0:
- Use firewall rules to restrict access
- Allow only trusted IPs or networks
- Consider using a VPN

## Starting Server with Remote Access

```bash
# SSH tunnel (no auth needed)
mehr serve --port 3000 &
ssh -L 8080:localhost:3000 user@remote-server

# Tailscale (VPN auth)
mehr serve --host 0.0.0.0 --port 3000

# Direct binding with auth
mehr serve auth add admin secretpassword
mehr serve --host 0.0.0.0 --port 3000

# Cloudflare tunnel
mehr serve --port 3000 &
cloudflared tunnel --url http://localhost:3000
```

## Troubleshooting

### "Connection Refused"

- Verify the server is running
- Check the port is correct
- Ensure firewall allows the connection

### "Authentication Required"

- Set up users with `mehr serve auth add`
- Use `--host localhost` to skip auth
- Use SSH tunnel to bypass auth requirement

### "Port Already in Use"

- Choose a different port with `--port`
- Find what's using the port:
  ```bash
  lsof -i :3000
  ```

## Next Steps

- [**Authentication**](authentication.md) - Set up users and security
- [**CLI: serve**](../cli/serve.md) - Server command options
- [**Settings**](settings.md) - Configure workspace

## CLI Equivalent

```bash
# Show tunnel info
mehr serve --tunnel-info

# Add user for remote access
mehr serve auth add admin password

# Start with network binding
mehr serve --host 0.0.0.0 --port 3000
```

See [CLI: serve](../cli/serve.md) for all options.
