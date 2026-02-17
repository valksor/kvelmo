package provider

import (
	"context"
	"errors"
	"testing"

	"github.com/valksor/go-toolkit/capability"
	"github.com/valksor/go-toolkit/providerconfig"
)

func TestProviderInfoStruct(t *testing.T) {
	info := ProviderInfo{
		Name:        "test",
		Description: "Test provider",
		Schemes:     []string{"test://", "t://"},
		Capabilities: capability.CapabilitySet{
			capability.CapRead: true,
			capability.CapList: true,
		},
		Priority: 10,
	}

	if info.Name != "test" {
		t.Errorf("ProviderInfo.Name = %q, want %q", info.Name, "test")
	}
	if len(info.Schemes) != 2 {
		t.Errorf("ProviderInfo.Schemes length = %d, want 2", len(info.Schemes))
	}
	if info.Priority != 10 {
		t.Errorf("ProviderInfo.Priority = %d, want 10", info.Priority)
	}
}

func TestNewRegistry(t *testing.T) {
	r := NewRegistry()

	if r.providers == nil {
		t.Error("providers map not initialized")
	}
	if r.schemes == nil {
		t.Error("schemes map not initialized")
	}
}

func TestRegistryRegister(t *testing.T) {
	r := NewRegistry()

	info := ProviderInfo{
		Name:    "test",
		Schemes: []string{"test://"},
	}
	factory := func(ctx context.Context, cfg providerconfig.Config) (any, error) {
		return nil, errors.New("test provider not implemented")
	}

	err := r.Register(info, factory)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Verify registration
	gotInfo, gotFactory, ok := r.Get("test")
	if !ok {
		t.Fatal("Get returned false for registered provider")
	}
	if gotInfo.Name != "test" {
		t.Errorf("got info.Name = %q, want %q", gotInfo.Name, "test")
	}
	if gotFactory == nil {
		t.Error("got factory is nil")
	}
}

func TestRegistryRegisterDuplicate(t *testing.T) {
	r := NewRegistry()

	info := ProviderInfo{Name: "test"}
	factory := func(ctx context.Context, cfg providerconfig.Config) (any, error) {
		return nil, errors.New("test provider not implemented")
	}

	if err := r.Register(info, factory); err != nil {
		t.Fatalf("Register first: %v", err)
	}
	err := r.Register(info, factory)

	if err == nil {
		t.Error("Register should fail for duplicate provider")
	}
}

func TestRegistryGet(t *testing.T) {
	r := NewRegistry()

	info := ProviderInfo{Name: "test"}
	factory := func(ctx context.Context, cfg providerconfig.Config) (any, error) {
		return nil, errors.New("test provider not implemented")
	}
	if err := r.Register(info, factory); err != nil {
		t.Fatalf("Register(test): %v", err)
	}

	gotInfo, _, ok := r.Get("test")
	if !ok {
		t.Fatal("Get returned false")
	}
	if gotInfo.Name != "test" {
		t.Errorf("got Name = %q, want %q", gotInfo.Name, "test")
	}
}

func TestRegistryGetNotFound(t *testing.T) {
	r := NewRegistry()

	_, _, ok := r.Get("nonexistent")
	if ok {
		t.Error("Get should return false for nonexistent provider")
	}
}

func TestRegistryGetByScheme(t *testing.T) {
	r := NewRegistry()

	info := ProviderInfo{
		Name:    "test",
		Schemes: []string{"test://", "t://"},
	}
	factory := func(ctx context.Context, cfg providerconfig.Config) (any, error) { return nil, nil } //nolint:nilnil // test mock
	if err := r.Register(info, factory); err != nil {
		t.Fatalf("Register(test): %v", err)
	}

	gotInfo, _, ok := r.GetByScheme("test://")
	if !ok {
		t.Fatal("GetByScheme returned false")
	}
	if gotInfo.Name != "test" {
		t.Errorf("got Name = %q, want %q", gotInfo.Name, "test")
	}

	// Test alternate scheme
	gotInfo, _, ok = r.GetByScheme("t://")
	if !ok {
		t.Fatal("GetByScheme returned false for alternate scheme")
	}
	if gotInfo.Name != "test" {
		t.Errorf("got Name = %q, want %q", gotInfo.Name, "test")
	}
}

func TestRegistryGetBySchemeNotFound(t *testing.T) {
	r := NewRegistry()

	_, _, ok := r.GetByScheme("unknown://")
	if ok {
		t.Error("GetByScheme should return false for unknown scheme")
	}
}

func TestRegistryList(t *testing.T) {
	r := NewRegistry()

	factory := func(ctx context.Context, cfg providerconfig.Config) (any, error) { return nil, nil } //nolint:nilnil // test mock

	if err := r.Register(ProviderInfo{Name: "low", Priority: 1}, factory); err != nil {
		t.Fatalf("Register(low): %v", err)
	}
	if err := r.Register(ProviderInfo{Name: "high", Priority: 10}, factory); err != nil {
		t.Fatalf("Register(high): %v", err)
	}
	if err := r.Register(ProviderInfo{Name: "medium", Priority: 5}, factory); err != nil {
		t.Fatalf("Register(medium): %v", err)
	}

	list := r.List()
	if len(list) != 3 {
		t.Errorf("List returned %d providers, want 3", len(list))
	}

	// Should be sorted by priority descending
	if list[0].Name != "high" {
		t.Errorf("list[0].Name = %q, want %q (highest priority)", list[0].Name, "high")
	}
	if list[1].Name != "medium" {
		t.Errorf("list[1].Name = %q, want %q", list[1].Name, "medium")
	}
	if list[2].Name != "low" {
		t.Errorf("list[2].Name = %q, want %q (lowest priority)", list[2].Name, "low")
	}
}

func TestRegistryCreate(t *testing.T) {
	r := NewRegistry()

	created := false
	factory := func(ctx context.Context, cfg providerconfig.Config) (any, error) {
		created = true

		return "instance", nil
	}

	if err := r.Register(ProviderInfo{Name: "test"}, factory); err != nil {
		t.Fatalf("Register(test): %v", err)
	}

	ctx := context.Background()
	instance, err := r.Create(ctx, "test", providerconfig.NewConfig())
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if !created {
		t.Error("factory was not called")
	}
	if instance != "instance" {
		t.Errorf("instance = %v, want %q", instance, "instance")
	}
}

func TestRegistryCreateNotFound(t *testing.T) {
	r := NewRegistry()

	ctx := context.Background()
	_, err := r.Create(ctx, "nonexistent", providerconfig.NewConfig())
	if err == nil {
		t.Error("Create should fail for nonexistent provider")
	}
}

// mockIdentifier implements workunit.Identifier interface for testing.
type mockIdentifier struct {
	parseErr    error
	parseResult string
	matchResult bool
}

func (m *mockIdentifier) Match(input string) bool {
	return m.matchResult
}

func (m *mockIdentifier) Parse(input string) (string, error) {
	return m.parseResult, m.parseErr
}

func TestParseScheme(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantScheme string
		wantIdent  string
	}{
		{
			name:       "file scheme",
			input:      "file:task.md",
			wantScheme: "file",
			wantIdent:  "task.md",
		},
		{
			name:       "github scheme",
			input:      "github:owner/repo#123",
			wantScheme: "github",
			wantIdent:  "owner/repo#123",
		},
		{
			name:       "dir scheme",
			input:      "dir:tasks/",
			wantScheme: "dir",
			wantIdent:  "tasks/",
		},
		{
			name:       "no scheme",
			input:      "task.md",
			wantScheme: "",
			wantIdent:  "task.md",
		},
		{
			name:       "windows path C drive",
			input:      "C:\\Users\\test\\task.md",
			wantScheme: "",
			wantIdent:  "C:\\Users\\test\\task.md",
		},
		{
			name:       "windows path C drive forward slash",
			input:      "C:/Users/test/task.md",
			wantScheme: "",
			wantIdent:  "C:/Users/test/task.md",
		},
		{
			name:       "empty input",
			input:      "",
			wantScheme: "",
			wantIdent:  "",
		},
		{
			name:       "colon at end",
			input:      "file:",
			wantScheme: "file",
			wantIdent:  "",
		},
		{
			name:       "multiple colons",
			input:      "http:example.com:8080",
			wantScheme: "http",
			wantIdent:  "example.com:8080",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme, ident := parseScheme(tt.input)
			if scheme != tt.wantScheme {
				t.Errorf("parseScheme(%q) scheme = %q, want %q", tt.input, scheme, tt.wantScheme)
			}
			if ident != tt.wantIdent {
				t.Errorf("parseScheme(%q) ident = %q, want %q", tt.input, ident, tt.wantIdent)
			}
		})
	}
}

func TestRegistryListSchemes(t *testing.T) {
	r := NewRegistry()

	// Initially empty
	schemes := r.listSchemes()
	if len(schemes) != 0 {
		t.Errorf("listSchemes() on empty registry = %v, want empty", schemes)
	}

	// Register providers with schemes
	factory := func(ctx context.Context, cfg providerconfig.Config) (any, error) {
		return &mockIdentifier{}, nil
	}

	if err := r.Register(ProviderInfo{Name: "file", Schemes: []string{"file", "f"}}, factory); err != nil {
		t.Fatalf("Register(file): %v", err)
	}
	if err := r.Register(ProviderInfo{Name: "github", Schemes: []string{"github", "gh"}}, factory); err != nil {
		t.Fatalf("Register(github): %v", err)
	}

	schemes = r.listSchemes()
	if len(schemes) != 4 {
		t.Errorf("listSchemes() = %v, want 4 schemes", schemes)
	}

	// Should be sorted alphabetically
	expected := []string{"f", "file", "gh", "github"}
	for i, s := range schemes {
		if s != expected[i] {
			t.Errorf("listSchemes()[%d] = %q, want %q", i, s, expected[i])
		}
	}
}

func TestRegistryResolve(t *testing.T) {
	r := NewRegistry()

	// Register a file provider
	factory := func(ctx context.Context, cfg providerconfig.Config) (any, error) {
		return &mockIdentifier{matchResult: true, parseResult: "resolved-id"}, nil
	}
	if err := r.Register(ProviderInfo{Name: "file", Schemes: []string{"file"}}, factory); err != nil {
		t.Fatalf("Register(file): %v", err)
	}

	ctx := context.Background()

	// Test with explicit scheme
	instance, id, err := r.Resolve(ctx, "file:task.md", providerconfig.NewConfig(), ResolveOptions{})
	if err != nil {
		t.Fatalf("Resolve with scheme failed: %v", err)
	}
	if id != "resolved-id" {
		t.Errorf("id = %q, want %q", id, "resolved-id")
	}
	if instance == nil {
		t.Error("instance is nil")
	}

	// Test with default provider
	_, id, err = r.Resolve(ctx, "task.md", providerconfig.NewConfig(), ResolveOptions{DefaultProvider: "file"})
	if err != nil {
		t.Fatalf("Resolve with default provider failed: %v", err)
	}
	if id != "resolved-id" {
		t.Errorf("id = %q, want %q", id, "resolved-id")
	}

	// Test without scheme and without default - should error
	_, _, err = r.Resolve(ctx, "task.md", providerconfig.NewConfig(), ResolveOptions{})
	if err == nil {
		t.Error("Resolve without scheme or default should fail")
	}
}

func TestRegistryResolveUnknownScheme(t *testing.T) {
	r := NewRegistry()

	ctx := context.Background()
	_, _, err := r.Resolve(ctx, "unknown:task.md", providerconfig.NewConfig(), ResolveOptions{})
	if err == nil {
		t.Error("Resolve with unknown scheme should fail")
	}
}

func TestRegistryNoSchemeError(t *testing.T) {
	r := NewRegistry()

	// Register a provider to have schemes in error message
	factory := func(ctx context.Context, cfg providerconfig.Config) (any, error) {
		return &mockIdentifier{}, nil
	}
	if err := r.Register(ProviderInfo{Name: "file", Schemes: []string{"file"}}, factory); err != nil {
		t.Fatalf("Register(file): %v", err)
	}

	err := r.noSchemeError("task.md")
	if err == nil {
		t.Fatal("noSchemeError should return an error")
	}

	errStr := err.Error()
	if !contains(errStr, "task.md") {
		t.Error("error should contain the input")
	}
	if !contains(errStr, "file") {
		t.Error("error should list available schemes")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}

	return false
}
