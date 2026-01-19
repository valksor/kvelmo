#!/bin/bash
set -euo pipefail

# Check unnecessary aliased imports where no conflict exists
# Flags alias != basename and where basename is not already imported
# Exception: when the actual package name equals the current package name, alias is NOT flagged
# (since using the same name would conflict with the current package)

# Pre-build a cache of import path -> actual package name for common stdlib packages
# This speeds up the check significantly

find . -type f -name '*.go' \
  -not -name '*.qtpl.go' \
  -not -path '*/vendor/*' \
  | while read -r file; do
    gawk '
      BEGIN {
        inblock = 0
        line_count = 0
        current_package = ""
      }

      # Store all lines of the file and detect current package
      {
        lines[line_count++] = $0
        # Detect package declaration
        if (match($0, /^package\s+([a-zA-Z_][a-zA-Z0-9_]*)/, pkg)) {
          current_package = pkg[1]
        }
      }

      # First pass: collect all imports with aliases and their paths
      /^\s*import\s*\(/ { inblock = 1; next }
      inblock && /^\s*\)/ { inblock = 0; next }

      inblock && match($0, /^\s*"([^"]+)"\s*$/, m) {
        split(m[1], parts, "/")
        base = parts[length(parts)]
        used[base] = 1
        next
      }

      inblock && match($0, /^\s*([a-zA-Z_][a-zA-Z0-9_]*)\s+"([^"]+)"/, m) {
        alias = m[1]
        path = m[2]
        imports[path] = alias
        used[alias] = 1
        next
      }

      match($0, /^\s*import\s+"([^"]+)"\s*$/, m) {
        split(m[1], parts, "/")
        base = parts[length(parts)]
        used[base] = 1
        next
      }

      match($0, /^\s*import\s+([a-zA-Z_][a-zA-Z0-9_]*)\s+"([^"]+)"\s*$/, m) {
        alias = m[1]
        path = m[2]
        imports[path] = alias
        used[alias] = 1
        next
      }

      END {
        inblock = 0
        for (i = 0; i < line_count; i++) {
          line = lines[i]

          if (line ~ /^\s*import\s*\(/) { inblock = 1; continue }
          if (inblock && line ~ /^\s*\)/) { inblock = 0; continue }

          if (inblock && match(line, /^\s*([a-zA-Z_][a-zA-Z0-9_]*)\s+"([^"]+)"/, m)) {
            alias = m[1]
            path = m[2]
            split(path, parts, "/")
            base = parts[length(parts)]

            # Determine actual package name
            pkg_name = base
            # Only check external packages, not stdlib or local paths
            if (path !~ /^\./ && index(path, ".") > 0) {
              cmd = "go list -f \"{{.Name}}\" \"" path "\" 2>/dev/null"
              if ((cmd | getline pname) > 0) {
                pkg_name = pname
              }
              close(cmd)
            }

            # Skip if alias is underscore, dot, equals base, or base is already used
            # Also skip if the actual package name equals current package (alias is necessary to avoid conflict)
            if (alias != "_" && alias != "." && alias != base && !(base in used) && pkg_name != current_package && base != current_package) {
              print FILENAME ":" i+1 ":" line
            }
            continue
          }

          if (!inblock && match(line, /^\s*import\s+([a-zA-Z_][a-zA-Z0-9_]*)\s+"([^"]+)"\s*$/, m)) {
            alias = m[1]
            path = m[2]
            split(path, parts, "/")
            base = parts[length(parts)]

            # Determine actual package name
            pkg_name = base
            if (path !~ /^\./ && index(path, ".") > 0) {
              cmd = "go list -f \"{{.Name}}\" \"" path "\" 2>/dev/null"
              if ((cmd | getline pname) > 0) {
                pkg_name = pname
              }
              close(cmd)
            }

            # Same check for single-line import
            if (alias != "_" && alias != "." && alias != base && !(base in used) && pkg_name != current_package && base != current_package) {
              print FILENAME ":" i+1 ":" line
            }
          }
        }
      }
    ' "$file"
  done
