# mehr license

Display license information for Mehrhof and its dependencies.

## Synopsis

```bash
# Show project license
mehr license

# List all dependency licenses
mehr license info [flags]
```

## Description

The `license` command provides license information:

- **`mehr license`** - Displays the project's own license (BSD 3-Clause)
- **`mehr license info`** - Lists all dependency licenses with SPDX identifiers

This is useful for:
- Compliance checking
- Attributing dependencies
- Understanding licensing obligations
- Generating attribution notices

## Commands

### `license`

Display the full text of the project's license (BSD 3-Clause).

```bash
mehr license
```

Output:
```
BSD 3-Clause License

Copyright (c) 2025+, Dāvis Zālītis (k0d3r1s)
Copyright (c) 2025+, SIA Valksor

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are met:

[full license text...]
```

### `license info`

List all Go module dependencies with their detected licenses.

```bash
mehr license info
```

Output:
```
Dependency Licenses:
github.com/spf13/cobra                              MIT
github.com/spf13/pflag                             BSD-3-Clause
github.com/stretchr/testify                         MIT
github.com/valksor/go-toolkit                       BSD-3-Clause
google.golang.org/protobuf                          BSD-3-Clause
golang.org/x/crypto                                  BSD-style
golang.org/x/net                                    BSD-style
golang.org/x/oauth2                                 BSD-style
golang.org/x/sys                                    BSD-style
golang.org/x/term                                   BSD-style
golang.org/x/text                                   BSD-style
gopkg.in/yaml.v3                                    MIT
...
```

## Flags

### `license info` Flags

| Flag           | Description                                      |
| -------------- | ------------------------------------------------ |
| `--json`       | Output as JSON                                   |
| `--unknown-only` | Only show packages with unknown licenses       |

## Examples

### Check Dependency Licenses

```bash
mehr license info
```

### Filter Unknown Licenses

Find dependencies that couldn't be automatically classified:

```bash
mehr license info --unknown-only
```

### JSON Output for Scripts

```bash
mehr license info --json | jq '.licenses[] | select(.unknown)'
```

### Count Dependencies by License

```bash
mehr license info --json | jq '.licenses | group_by(.license) | map({license: .[0].license, count: length})'
```

## SPDX Detection

Mehrhof uses [Google's go-licenses](https://github.com/google/go-licenses) library to detect SPDX license identifiers. This provides:

- **Standardized identifiers** - MIT, Apache-2.0, BSD-3-Clause, etc.
- **License types** - Permissive, Reciprocal, Restricted, etc.
- **High accuracy** - Based on license text analysis

### License Types

The detector classifies licenses into these types:

| Type          | Examples                                     |
| ------------- | -------------------------------------------- |
| Permissive    | MIT, BSD, Apache-2.0                         |
| Reciprocal    | GPL-2.0, MPL-2.0                              |
| Restricted    | AGPL-3.0, GPL-3.0                            |
| Unknown       | Custom or undetectable license text          |

## Web UI

License information is also available in the web UI:

- Navigate to **Settings → License**
- View project license and all dependency licenses
- Export license information for attribution

## API Endpoints

When using `mehr serve`, license information is available via API:

```bash
# Get project license
curl http://localhost:8080/api/v1/license

# Get dependency licenses
curl http://localhost:8080/api/v1/license/info
```

## Attribution

When distributing software that includes Mehrhof, ensure you:

1. **Preserve license notices** - Keep all copyright and license text
2. **Acknowledge dependencies** - Include attribution for third-party packages
3. **Comply with terms** - Follow conditions of reciprocal licenses (GPL, MPL, etc.)

Use `mehr license info --json` to generate machine-readable attribution data.

## See Also

- [Legal Information](../legal/license.md) - Full BSD 3-Clause license text
- [Contributing](../legal/contributing.md) - Contribution licensing terms
- [CLI Overview](index.md) - All available commands
