# Remote Access

Access the Mehrhof Web UI from anywhere using SSH tunnels, Cloudflare tunnels, or direct network access.

## Access Options

| Method                | Use Case                           | Difficulty           |
|-----------------------|------------------------------------|----------------------|
| **SSH Tunnel**        | Secure access without open ports   | Easy                 |
| **Tailscale**         | Mesh VPN access                    | Medium               |
| **Cloudflare Tunnel** | Public URL without port forwarding | Easy                 |
| **Direct Binding**    | LAN/exposed server                 | Easy (requires auth) |

## Option 1: SSH Tunnel

Securely access the Web UI from your local machine through an SSH tunnel.

### How It Works

- **Local to Remote:** Forward a local port to the remote server's Web UI
- **Remote to Local:** Expose your local Web UI to a remote server

Once the tunnel is established, open your browser to the local forwarded port (e.g., `http://localhost:8080`).

### Setup

See [CLI: serve](/cli/serve.md) for SSH tunnel commands and the built-in tunnel helper.

## Option 2: Tailscale

Use Tailscale mesh VPN for secure access to your Web UI.

### Setup

1. Install Tailscale on both machines
2. Log in to each machine
3. Start the server with network binding (see [CLI: serve](/cli/serve.md))
4. Access via your Tailscale IP in the browser (e.g., `http://100.x.x.x:3000`)

### Benefits

- **No port forwarding** - Works through NAT
- **Encrypted** - All traffic is secure
- **Built-in auth** - Tailscale handles authentication
- **Easy** - Just install and log in

## Option 3: Cloudflare Tunnel

Use Cloudflare Tunnel to expose your Web UI publicly without port forwarding.

### Setup

1. Start the Mehrhof server (see [CLI: serve](/cli/serve.md))
2. Start Cloudflare tunnel pointing to your local server
3. Cloudflare provides a public HTTPS URL

### Benefits

- **Free** - No cost for basic usage
- **No port forwarding** - Works through NAT
- **HTTPS** - Automatic SSL certificate
- **DDoS protection** - Cloudflare shields your server

## Option 4: Direct Network Binding

Bind to all network interfaces for LAN or internet access.

### ⚠️ Authentication Required

When binding to network interfaces, authentication is **required**. You must set up users before starting the server.

### Setup

1. Add users via the CLI (see [CLI: serve](/cli/serve.md))
2. Start the server with network binding
3. Access from any device on your network (e.g., `http://192.168.1.100:3000`)
4. Log in with the credentials you created

See [Authentication](authentication.md) for details on user management and the login experience.

## Access Comparison

| Method               | Public URL | Auth Needed       | Port Forwarding |
|----------------------|------------|-------------------|-----------------|
| **Default**          | ❌ No       | ❌ No              | ❌ No            |
| **SSH Tunnel**       | ❌ No       | ❌ No              | ❌ No            |
| **Tailscale**        | ❌ No       | ✅ Yes (Tailscale) | ❌ No            |
| **Cloudflare**       | ✅ Yes      | ❌ No (optional)   | ❌ No            |
| **Direct (0.0.0.0)** | ⚠️ LAN     | ✅ Yes             | ❌ No            |

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

## Troubleshooting

### "Connection Refused"

- Verify the server is running
- Check the port is correct
- Ensure firewall allows the connection

### "Authentication Required"

- Set up users via the CLI before starting
- Use localhost mode to skip auth requirement
- Use SSH tunnel to bypass auth requirement

### "Port Already in Use"

- Choose a different port when starting the server
- Check what process is using the port and stop it

## Next Steps

- [**Authentication**](authentication.md) - Set up users and security
- [**CLI: serve**](/cli/serve.md) - Server command options
- [**Settings**](settings.md) - Configure workspace

---

## Also Available via CLI

All server configuration and remote access setup is performed via the command line.

See [CLI: serve](/cli/serve.md) for all server options, tunnel instructions, and authentication commands.
