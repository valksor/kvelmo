package security

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
)

// validateScanDir validates that the scan directory is safe to use.
func validateScanDir(dir string) error {
	// Convert to absolute path
	abs, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("invalid directory path: %w", err)
	}

	// Use filepath.Clean to normalize the path
	cleanPath := filepath.Clean(abs)

	// Check for parent directory references in a platform-agnostic way
	// Split the path and check for ".." components
	parts := strings.Split(cleanPath, string(filepath.Separator))
	for _, part := range parts {
		if part == ".." {
			return fmt.Errorf("directory traversal detected: %s", dir)
		}
	}

	// Check for backslash-based traversal patterns (malformed paths on Unix)
	if strings.Contains(abs, "..\\") {
		return fmt.Errorf("directory traversal detected: %s", dir)
	}

	return nil
}

// ScannerRegistry manages a collection of security scanners.
type ScannerRegistry struct {
	scanners map[string]Scanner
	mu       sync.RWMutex
	toolMgr  *ToolManager
}

// NewScannerRegistry creates a new scanner registry.
func NewScannerRegistry() *ScannerRegistry {
	return &ScannerRegistry{
		scanners: make(map[string]Scanner),
	}
}

// Register registers a scanner with the registry.
func (r *ScannerRegistry) Register(name string, scanner Scanner) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.scanners[name] = scanner
}

// Unregister removes a scanner from the registry.
func (r *ScannerRegistry) Unregister(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.scanners, name)
}

// Get retrieves a scanner by name.
func (r *ScannerRegistry) Get(name string) (Scanner, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	scanner, ok := r.scanners[name]

	return scanner, ok
}

// List returns all registered scanner names.
func (r *ScannerRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.scanners))
	for name := range r.scanners {
		names = append(names, name)
	}

	return names
}

// SetToolManager sets the tool manager for the registry.
func (r *ScannerRegistry) SetToolManager(tm *ToolManager) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.toolMgr = tm
}

// GetToolManager returns the tool manager for the registry.
func (r *ScannerRegistry) GetToolManager() *ToolManager {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.toolMgr
}

// RunAll runs all enabled scanners and returns their results.
func (r *ScannerRegistry) RunAll(ctx context.Context, dir string) ([]*ScanResult, error) {
	// Validate directory path before scanning
	if err := validateScanDir(dir); err != nil {
		return nil, fmt.Errorf("invalid scan directory: %w", err)
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	var results []*ScanResult
	var errors []error

	for _, scanner := range r.scanners {
		if !scanner.IsEnabled() {
			continue
		}

		result, err := scanner.Scan(ctx, dir)
		if err != nil {
			errors = append(errors, fmt.Errorf("%s: %w", scanner.Name(), err))
			// Still add the result with the error
			results = append(results, &ScanResult{
				Scanner: scanner.Name(),
				Error:   err,
			})

			continue
		}

		results = append(results, result)
	}

	if len(errors) > 0 && len(results) == 0 {
		return nil, fmt.Errorf("all scanners failed: %v", errors)
	}

	return results, nil
}

// RunEnabled runs only enabled scanners that match the given names.
// If names is empty, all enabled scanners are run.
func (r *ScannerRegistry) RunEnabled(ctx context.Context, dir string, names []string) ([]*ScanResult, error) {
	// Validate directory path before scanning
	if err := validateScanDir(dir); err != nil {
		return nil, fmt.Errorf("invalid scan directory: %w", err)
	}

	if len(names) == 0 {
		return r.RunAll(ctx, dir)
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	var results []*ScanResult
	var errors []error

	for _, name := range names {
		scanner, ok := r.scanners[name]
		if !ok {
			errors = append(errors, fmt.Errorf("scanner not found: %s", name))

			continue
		}

		if !scanner.IsEnabled() {
			continue
		}

		result, err := scanner.Scan(ctx, dir)
		if err != nil {
			errors = append(errors, fmt.Errorf("%s: %w", scanner.Name(), err))
			results = append(results, &ScanResult{
				Scanner: scanner.Name(),
				Error:   err,
			})

			continue
		}

		results = append(results, result)
	}

	if len(errors) > 0 && len(results) == 0 {
		return nil, fmt.Errorf("all scanners failed: %v", errors)
	}

	return results, nil
}

// Count returns the number of registered scanners.
func (r *ScannerRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.scanners)
}

// EnabledCount returns the number of enabled scanners.
func (r *ScannerRegistry) EnabledCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	count := 0
	for _, scanner := range r.scanners {
		if scanner.IsEnabled() {
			count++
		}
	}

	return count
}

// SummarizeFindings creates a summary from a list of findings.
func SummarizeFindings(findings []Finding) Summary {
	summary := Summary{
		Total:      len(findings),
		BySeverity: make(map[Severity]int),
	}

	for _, finding := range findings {
		summary.BySeverity[finding.Severity]++
	}

	return summary
}
