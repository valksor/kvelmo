# Process Supervision

Example configurations for running kvelmo as a managed service.

## systemd (Linux)

```bash
# Install the service file
sudo cp contrib/systemd/kvelmo.service /etc/systemd/system/

# Create the kvelmo user (if needed)
sudo useradd -r -s /bin/false -m kvelmo

# Reload systemd and enable
sudo systemctl daemon-reload
sudo systemctl enable kvelmo
sudo systemctl start kvelmo

# Check status
sudo systemctl status kvelmo
journalctl -u kvelmo -f
```

## launchd (macOS)

```bash
# Install the plist
cp contrib/launchd/com.valksor.kvelmo.plist ~/Library/LaunchAgents/

# Create log directory
mkdir -p /usr/local/var/log/kvelmo

# Load the service
launchctl load ~/Library/LaunchAgents/com.valksor.kvelmo.plist

# Check status
launchctl list | grep kvelmo
```

## Configuration

Both configurations use `--log-format json` for structured logging and set
`KVELMO_ENVIRONMENT=prod` for production guardrails.

Adjust paths to match your kvelmo binary location.
