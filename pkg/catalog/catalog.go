// Package catalog manages task templates that can be reused across projects.
// Templates are stored as YAML files in a local directory.
package catalog

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/valksor/kvelmo/pkg/paths"

	"gopkg.in/yaml.v3"
)

// Catalog manages templates from a local directory.
type Catalog struct {
	dir string
}

// New creates a Catalog that reads templates from dir.
// If dir is empty, defaults to <BaseDir>/templates.
func New(dir string) *Catalog {
	if dir == "" {
		dir = filepath.Join(paths.BaseDir(), "templates")
	}

	return &Catalog{dir: dir}
}

// List reads all .yaml files in the catalog directory and returns them
// as parsed Templates.
func (c *Catalog) List() ([]Template, error) {
	if err := os.MkdirAll(c.dir, 0o755); err != nil {
		return nil, fmt.Errorf("ensure catalog dir: %w", err)
	}

	entries, err := os.ReadDir(c.dir)
	if err != nil {
		return nil, fmt.Errorf("read catalog dir: %w", err)
	}

	var templates []Template

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}

		tmpl, err := c.loadTemplate(filepath.Join(c.dir, entry.Name()))
		if err != nil {
			slog.Warn("skipping invalid template", "file", entry.Name(), "error", err)

			continue
		}

		templates = append(templates, *tmpl)
	}

	return templates, nil
}

// Get reads a specific template by name. The name should not include
// the .yaml extension.
func (c *Catalog) Get(name string) (*Template, error) {
	path := filepath.Join(c.dir, name+".yaml")

	tmpl, err := c.loadTemplate(path)
	if err != nil {
		return nil, fmt.Errorf("get template %q: %w", name, err)
	}

	return tmpl, nil
}

// Import copies a .yaml file into the catalog directory.
func (c *Catalog) Import(path string) error {
	if err := os.MkdirAll(c.dir, 0o755); err != nil {
		return fmt.Errorf("ensure catalog dir: %w", err)
	}

	// Validate that the source is valid YAML with a template.
	if _, err := c.loadTemplate(path); err != nil {
		return fmt.Errorf("validate template: %w", err)
	}

	src, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open source: %w", err)
	}
	defer func() { _ = src.Close() }()

	destPath := filepath.Join(c.dir, filepath.Base(path))

	dst, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("create destination: %w", err)
	}
	defer func() { _ = dst.Close() }()

	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("copy template: %w", err)
	}

	return nil
}

func (c *Catalog) loadTemplate(path string) (*Template, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	var tmpl Template
	if err := yaml.Unmarshal(data, &tmpl); err != nil {
		return nil, fmt.Errorf("parse yaml: %w", err)
	}

	if tmpl.Name == "" {
		return nil, errors.New("template missing required field 'name'")
	}

	return &tmpl, nil
}
