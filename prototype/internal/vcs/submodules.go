package vcs

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

const (
	legacyDirtySubmodulesFile = ".asc-submodules.json"
	dirtySubmodulesFile       = ".mehrhof-submodules.json"
)

var allowedDirtyKeys = []string{
	"allow_dirty_submodules",
	"allow_dirty",
	"allow",
}

func (g *Git) filterAllowedDirtySubmodules(files []FileStatus) []FileStatus {
	allowed := g.loadAllowedDirtySubmodules()
	if len(allowed) == 0 || len(files) == 0 {
		return files
	}

	known := g.loadKnownSubmodules()
	if len(known) == 0 {
		return files
	}

	filtered := make([]FileStatus, 0, len(files))
	for _, fs := range files {
		path := normalizeStatusPath(fs.Path)
		path = normalizeRelativePath(path)
		if path == "" {
			filtered = append(filtered, fs)

			continue
		}

		if _, ok := allowed[path]; ok {
			if _, isSubmodule := known[path]; isSubmodule {
				continue
			}
		}

		filtered = append(filtered, fs)
	}

	return filtered
}

func (g *Git) loadAllowedDirtySubmodules() map[string]struct{} {
	result := make(map[string]struct{})

	// Prefer native mehrhof file, but also support legacy ASC file.
	for _, name := range []string{dirtySubmodulesFile, legacyDirtySubmodulesFile} {
		path := filepath.Join(g.repoRoot, name)
		paths := readAllowedDirtySubmodulesFile(path)
		for _, p := range paths {
			normalized := normalizeRelativePath(p)
			if normalized == "" {
				continue
			}
			result[normalized] = struct{}{}
		}
	}

	return result
}

func readAllowedDirtySubmodulesFile(path string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil
	}

	// List form: ["path/a", "path/b"]
	if list, ok := raw.([]any); ok {
		return toStringList(list)
	}

	// Object form: {"allow_dirty_submodules": [...]}
	obj, ok := raw.(map[string]any)
	if !ok {
		return nil
	}
	for _, key := range allowedDirtyKeys {
		value, ok := obj[key]
		if !ok {
			continue
		}
		if list, ok := value.([]any); ok {
			return toStringList(list)
		}
	}

	return nil
}

func toStringList(values []any) []string {
	out := make([]string, 0, len(values))
	for _, v := range values {
		s, ok := v.(string)
		if !ok {
			continue
		}
		out = append(out, s)
	}

	return out
}

func (g *Git) loadKnownSubmodules() map[string]struct{} {
	path := filepath.Join(g.repoRoot, ".gitmodules")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	known := make(map[string]struct{})
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || !strings.Contains(line, "=") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		if strings.TrimSpace(parts[0]) != "path" {
			continue
		}
		normalized := normalizeRelativePath(parts[1])
		if normalized == "" {
			continue
		}
		known[normalized] = struct{}{}
	}

	return known
}

func normalizeStatusPath(path string) string {
	p := strings.TrimSpace(path)
	if p == "" {
		return ""
	}
	if strings.Contains(p, " -> ") {
		parts := strings.SplitN(p, " -> ", 2)
		p = parts[1]
	}
	if strings.Contains(p, " (") {
		parts := strings.SplitN(p, " (", 2)
		p = parts[0]
	}
	p = strings.Trim(p, `"`)

	return strings.TrimSpace(p)
}

func normalizeRelativePath(path string) string {
	p := strings.TrimSpace(path)
	if p == "" {
		return ""
	}
	p = filepath.ToSlash(p)
	p = strings.TrimPrefix(p, "./")
	p = strings.TrimPrefix(p, "/")
	p = strings.TrimSuffix(p, "/")

	return p
}
