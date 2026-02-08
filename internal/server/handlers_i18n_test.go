package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valksor/go-toolkit/paths"

	"github.com/valksor/go-mehrhof/internal/conductor"
	"github.com/valksor/go-mehrhof/internal/helper_test"
	"github.com/valksor/go-mehrhof/internal/storage"
)

func TestHandleGetI18nOverrides_NoConductor(t *testing.T) {
	cfg := Config{
		Port:      0,
		Mode:      ModeGlobal,
		Conductor: nil,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	resp, err := doGet(ctx, client, srv.URL()+"/api/v1/i18n/overrides")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	// Should return OK with empty overrides when no conductor
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	// Should contain empty terminology and keys
	assert.Contains(t, string(body), `"terminology"`)
	assert.Contains(t, string(body), `"keys"`)
}

func TestHandleGetI18nOverrides_WithProjectContext(t *testing.T) {
	homeDir := t.TempDir()
	t.Cleanup(paths.SetHomeDirForTesting(homeDir))

	c := helper_test.NewTestConductor(t,
		conductor.WithHomeDir(homeDir),
	)

	// Save some global overrides
	globalOverrides := &storage.I18nOverrides{
		Terminology: map[string]string{"Task": "Ticket"},
		Keys:        map[string]map[string]string{"en": {"nav.dashboard": "Home"}},
	}
	err := storage.SaveI18nOverrides("", globalOverrides)
	require.NoError(t, err)

	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		Conductor: c,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	resp, err := doGet(ctx, client, srv.URL()+"/api/v1/i18n/overrides")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	bodyStr := string(body)
	assert.Contains(t, bodyStr, `"Task"`)
	assert.Contains(t, bodyStr, `"Ticket"`)
}

func TestHandleGetI18nOverridesGlobal_Success(t *testing.T) {
	homeDir := t.TempDir()
	t.Cleanup(paths.SetHomeDirForTesting(homeDir))

	// Save global overrides
	globalOverrides := &storage.I18nOverrides{
		Terminology: map[string]string{"Workflow": "Pipeline"},
		Keys:        map[string]map[string]string{},
	}
	err := storage.SaveI18nOverrides("", globalOverrides)
	require.NoError(t, err)

	cfg := Config{
		Port:      0,
		Mode:      ModeGlobal,
		Conductor: nil,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	resp, err := doGet(ctx, client, srv.URL()+"/api/v1/i18n/overrides/global")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	bodyStr := string(body)
	assert.Contains(t, bodyStr, `"Workflow"`)
	assert.Contains(t, bodyStr, `"Pipeline"`)
}

func TestHandleGetI18nOverridesProject_NoProject(t *testing.T) {
	cfg := Config{
		Port:      0,
		Mode:      ModeGlobal,
		Conductor: nil,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	resp, err := doGet(ctx, client, srv.URL()+"/api/v1/i18n/overrides/project")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	// Should return OK with empty overrides when no project
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	// Should contain empty terminology and keys
	var result storage.I18nOverrides
	err = json.Unmarshal(extractData(body), &result)
	require.NoError(t, err)

	assert.Empty(t, result.Terminology)
	assert.Empty(t, result.Keys)
}

func TestHandleSaveI18nOverridesGlobal_ValidJSON(t *testing.T) {
	homeDir := t.TempDir()
	t.Cleanup(paths.SetHomeDirForTesting(homeDir))

	cfg := Config{
		Port:      0,
		Mode:      ModeGlobal,
		Conductor: nil,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	payload := `{"terminology": {"Test": "Value"}, "keys": {}}`
	resp, err := doPost(ctx, client, srv.URL()+"/api/v1/i18n/overrides/global", bytes.NewBufferString(payload))
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), "saved")

	// Verify file was written
	loaded, err := storage.LoadI18nOverrides("")
	require.NoError(t, err)
	assert.Equal(t, "Value", loaded.Terminology["Test"])
}

func TestHandleSaveI18nOverridesGlobal_InvalidJSON(t *testing.T) {
	homeDir := t.TempDir()
	t.Cleanup(paths.SetHomeDirForTesting(homeDir))

	cfg := Config{
		Port:      0,
		Mode:      ModeGlobal,
		Conductor: nil,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	payload := `{invalid json}`
	resp, err := doPost(ctx, client, srv.URL()+"/api/v1/i18n/overrides/global", bytes.NewBufferString(payload))
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), "invalid")
}

func TestHandleSaveI18nOverridesProject_NoProjectContext(t *testing.T) {
	cfg := Config{
		Port:      0,
		Mode:      ModeGlobal,
		Conductor: nil,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	payload := `{"terminology": {"Test": "Value"}, "keys": {}}`
	resp, err := doPost(ctx, client, srv.URL()+"/api/v1/i18n/overrides/project", bytes.NewBufferString(payload))
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), "no project context")
}

func TestHandleSaveI18nOverridesProject_ValidJSON(t *testing.T) {
	homeDir := t.TempDir()
	t.Cleanup(paths.SetHomeDirForTesting(homeDir))

	c := helper_test.NewTestConductor(t,
		conductor.WithHomeDir(homeDir),
	)

	cfg := Config{
		Port:      0,
		Mode:      ModeProject,
		Conductor: c,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	payload := `{"terminology": {"Project": "ProjectValue"}, "keys": {"en": {"nav.test": "Test"}}}`
	resp, err := doPost(ctx, client, srv.URL()+"/api/v1/i18n/overrides/project", bytes.NewBufferString(payload))
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), "saved")
}

func TestHandleSaveI18nOverridesGlobal_EmptyFindKey(t *testing.T) {
	homeDir := t.TempDir()
	t.Cleanup(paths.SetHomeDirForTesting(homeDir))

	cfg := Config{
		Port:      0,
		Mode:      ModeGlobal,
		Conductor: nil,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	// Empty find key in terminology should fail validation
	payload := `{"terminology": {"": "Value"}, "keys": {}}`
	resp, err := doPost(ctx, client, srv.URL()+"/api/v1/i18n/overrides/global", bytes.NewBufferString(payload))
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), "find")
}

func TestHandleSaveI18nOverridesGlobal_EmptyReplaceValue(t *testing.T) {
	homeDir := t.TempDir()
	t.Cleanup(paths.SetHomeDirForTesting(homeDir))

	cfg := Config{
		Port:      0,
		Mode:      ModeGlobal,
		Conductor: nil,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	// Empty replace value in terminology should fail validation
	payload := `{"terminology": {"Task": ""}, "keys": {}}`
	resp, err := doPost(ctx, client, srv.URL()+"/api/v1/i18n/overrides/global", bytes.NewBufferString(payload))
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), "replace")
}

func TestHandleSaveI18nOverridesGlobal_InvalidLanguageCode(t *testing.T) {
	homeDir := t.TempDir()
	t.Cleanup(paths.SetHomeDirForTesting(homeDir))

	cfg := Config{
		Port:      0,
		Mode:      ModeGlobal,
		Conductor: nil,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	// Uppercase language code should fail validation (must be lowercase)
	payload := `{"terminology": {}, "keys": {"EN": {"nav.test": "Test"}}}`
	resp, err := doPost(ctx, client, srv.URL()+"/api/v1/i18n/overrides/global", bytes.NewBufferString(payload))
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), "invalid language code")
}

func TestHandleSaveI18nOverridesGlobal_EmptyTranslationKey(t *testing.T) {
	homeDir := t.TempDir()
	t.Cleanup(paths.SetHomeDirForTesting(homeDir))

	cfg := Config{
		Port:      0,
		Mode:      ModeGlobal,
		Conductor: nil,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	// Empty translation key should fail validation
	payload := `{"terminology": {}, "keys": {"en": {"": "Test"}}}`
	resp, err := doPost(ctx, client, srv.URL()+"/api/v1/i18n/overrides/global", bytes.NewBufferString(payload))
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), "translation key")
}

func TestValidateI18nOverrides_ValidData(t *testing.T) {
	// Valid data should pass validation
	o := &storage.I18nOverrides{
		Terminology: map[string]string{"Task": "Ticket", "Workflow": "Pipeline"},
		Keys:        map[string]map[string]string{"en": {"nav.test": "Test"}, "de": {"nav.test": "Prüfung"}},
	}

	err := validateI18nOverrides(o)
	assert.NoError(t, err)
}

func TestValidateI18nOverrides_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		o       *storage.I18nOverrides
		wantErr bool
		errMsg  string
	}{
		{
			name:    "empty overrides",
			o:       storage.NewI18nOverrides(),
			wantErr: false,
		},
		{
			name: "whitespace-only find key",
			o: &storage.I18nOverrides{
				Terminology: map[string]string{"   ": "Value"},
				Keys:        map[string]map[string]string{},
			},
			wantErr: true,
			errMsg:  "find",
		},
		{
			name: "whitespace-only replace value",
			o: &storage.I18nOverrides{
				Terminology: map[string]string{"Task": "   "},
				Keys:        map[string]map[string]string{},
			},
			wantErr: true,
			errMsg:  "replace",
		},
		{
			name: "single-letter language code (invalid)",
			o: &storage.I18nOverrides{
				Terminology: map[string]string{},
				Keys:        map[string]map[string]string{"e": {"nav.test": "Test"}},
			},
			wantErr: true,
			errMsg:  "invalid language code",
		},
		{
			name: "four-letter language code (invalid)",
			o: &storage.I18nOverrides{
				Terminology: map[string]string{},
				Keys:        map[string]map[string]string{"engl": {"nav.test": "Test"}},
			},
			wantErr: true,
			errMsg:  "invalid language code",
		},
		{
			name: "three-letter language code (valid)",
			o: &storage.I18nOverrides{
				Terminology: map[string]string{},
				Keys:        map[string]map[string]string{"deu": {"nav.test": "Test"}},
			},
			wantErr: false,
		},
		{
			name: "language code with numbers (invalid)",
			o: &storage.I18nOverrides{
				Terminology: map[string]string{},
				Keys:        map[string]map[string]string{"en1": {"nav.test": "Test"}},
			},
			wantErr: true,
			errMsg:  "invalid language code",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateI18nOverrides(tt.o)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestIsValidLanguageCode(t *testing.T) {
	tests := []struct {
		code  string
		valid bool
	}{
		{"en", true},
		{"de", true},
		{"fra", true},
		{"deu", true},
		{"EN", false},    // uppercase
		{"De", false},    // mixed case
		{"e", false},     // too short
		{"engl", false},  // too long
		{"en1", false},   // contains number
		{"e-n", false},   // contains hyphen
		{"", false},      // empty
		{"en_US", false}, // contains underscore
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			result := isValidLanguageCode(tt.code)
			assert.Equal(t, tt.valid, result, "isValidLanguageCode(%q)", tt.code)
		})
	}
}

func TestHandleGetI18nKeys_ReturnsKeys(t *testing.T) {
	cfg := Config{
		Port:      0,
		Mode:      ModeGlobal,
		Conductor: nil,
	}

	srv, cleanup := startTestServer(t, cfg)
	defer cleanup()

	ctx := context.Background()
	client := testHTTPClient()

	resp, err := doGet(ctx, client, srv.URL()+"/api/v1/i18n/keys")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	bodyStr := string(body)
	// Check some expected keys are present
	assert.Contains(t, bodyStr, `"nav.dashboard"`)
	assert.Contains(t, bodyStr, `"nav.settings"`)
	assert.Contains(t, bodyStr, `"workflow:states.idle"`)
	assert.Contains(t, bodyStr, `"settings:title"`)
}

// extractData extracts the "data" field from a JSON response.
// The server wraps responses in {"success": true, "data": ...}.
func extractData(body []byte) []byte {
	var resp struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return body // Return original if not wrapped
	}

	return resp.Data
}
