# kvelmo config

Configuration management.

## Usage

```bash
kvelmo config <subcommand>
```

## Subcommands

| Command | Description |
|---------|-------------|
| `show` | Show current configuration |
| `init` | Initialize default configuration |
| `set <key> <value>` | Set a configuration value |

## Examples

```bash
# Show all settings
kvelmo config show

# Initialize defaults
kvelmo config init

# Set a value
kvelmo config set default_agent claude
kvelmo config set max_workers 8
```

## Configuration Files

| Scope | Location |
|-------|----------|
| Global | `~/.valksor/kvelmo/kvelmo.yaml` |
| Project | `.valksor/kvelmo.yaml` |

## Common Settings

| Key | Description |
|-----|-------------|
| `default_agent` | Default AI agent |
| `max_workers` | Maximum concurrent workers |
| `web_port` | Web UI port |

## Related

- [Configuration](/configuration/index.md) — Full configuration guide
- [Settings](/configuration/settings.md) — All settings
- [Environment](/configuration/environment.md) — Environment variables
