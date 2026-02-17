// Package helper_test provides shared testing utilities for go-mehrhof tests.
package helper_test

import (
	"time"

	"github.com/valksor/go-mehrhof/internal/browser"
	"github.com/valksor/go-mehrhof/internal/storage"
	"github.com/valksor/go-toolkit/workunit"
)

// SampleTaskContent returns a sample task markdown content.
func SampleTaskContent(title string) string {
	return `---` + "\n" + `title: ` + title + "\n" + `---` + "\n\n" +
		`This is a sample task description for testing purposes.`
}

// SampleWorkUnit returns a sample WorkUnit for testing.
func SampleWorkUnit() *workunit.WorkUnit {
	return &workunit.WorkUnit{
		ID:          "sample-task-123",
		Title:       "Sample Task",
		Description: "This is a sample task for testing",
		ExternalKey: "SAMPLE-123",
		Provider:    "file",
		Status:      workunit.StatusOpen,
		Priority:    workunit.PriorityNormal,
		Source: workunit.SourceInfo{
			Type:      "file",
			Reference: "task.md",
			SyncedAt:  time.Now(),
		},
		ExternalID: "sample-task-123",
		TaskType:   "feature",
		Slug:       "sample-task",
		Metadata: map[string]any{
			"title": "Sample Task",
		},
		AgentConfig: nil,
	}
}

// SampleWorkUnitWithOptions returns a configurable WorkUnit for testing.
func SampleWorkUnitWithOptions(opts func(*workunit.WorkUnit)) *workunit.WorkUnit {
	wu := SampleWorkUnit()
	if opts != nil {
		opts(wu)
	}

	return wu
}

// SampleWorkspaceConfig returns a sample workspace configuration.
func SampleWorkspaceConfig() *storage.WorkspaceConfig {
	return &storage.WorkspaceConfig{
		Git: storage.GitSettings{
			BranchPattern: "feature/{key}--{slug}",
			CommitPrefix:  "[{key}]",
		},
		Agent: storage.AgentSettings{
			Default: "claude",
		},
		Providers: storage.ProvidersSettings{
			Default: "file",
		},
		Plugins: storage.PluginsConfig{
			Enabled: []string{},
			Config:  map[string]map[string]any{},
		},
		Agents: map[string]storage.AgentAliasConfig{},
		Env:    map[string]string{},
	}
}

// SampleAgentConfig returns a sample agent configuration.
func SampleAgentConfig() *workunit.AgentConfig {
	return &workunit.AgentConfig{
		Name: "test-agent",
		Env: map[string]string{
			"TEST_VAR": "test-value",
		},
		Args: []string{"--test-arg"},
		Steps: map[string]workunit.StepAgentConfig{
			"planning": {
				Name: "planning-agent",
				Env:  map[string]string{"PLANNING_VAR": "planning-value"},
				Args: []string{"--planning-turns", "5"},
			},
		},
	}
}

// SampleSourceInfo returns a sample source info for testing.
func SampleSourceInfo() storage.SourceInfo {
	return storage.SourceInfo{
		Type:   "file",
		Ref:    "task.md",
		ReadAt: time.Now(),
	}
}

// SampleActiveTask returns a sample active task for testing.
func SampleActiveTask(taskID, title string) *storage.ActiveTask {
	return &storage.ActiveTask{
		ID:      taskID,
		Ref:     "file:task.md",
		WorkDir: ".mehrhof/work/" + taskID,
		State:   "idle",
		Branch:  "",
		UseGit:  false,
		Started: time.Now(),
	}
}

// SampleTaskWork returns sample task work for testing.
func SampleTaskWork(taskID, title string) *storage.TaskWork {
	now := time.Now()

	return &storage.TaskWork{
		Version: "1",
		Metadata: storage.WorkMetadata{
			ID:          taskID,
			Title:       title,
			ExternalKey: "SAMPLE-123",
			TaskType:    "feature",
			Slug:        "sample-task",
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		Source: storage.SourceInfo{
			Type:   "file",
			Ref:    "task.md",
			ReadAt: now,
		},
		Git: storage.GitInfo{
			Branch:        "feature/sample-123--sample-task",
			BaseBranch:    "main",
			CommitPrefix:  "[SAMPLE-123]",
			BranchPattern: "feature/{key}--{slug}",
		},
		Agent: storage.AgentInfo{
			Name:   "claude",
			Source: "workspace",
		},
	}
}

// SampleSpecification returns sample specification content.
func SampleSpecification(num int) string {
	return `# Specification ` + string(rune('0'+num)) + `

## Summary
This is a sample specification for testing.

## Details
- Implement feature X
- Write tests for X
- Document the changes
`
}

// SamplePendingQuestion returns a sample pending question.
func SamplePendingQuestion() *storage.PendingQuestion {
	return &storage.PendingQuestion{
		Question:       "Should we implement option A or B?",
		Phase:          "planning",
		AskedAt:        time.Now(),
		ContextSummary: "Context summary here",
		FullContext:    "Full context here",
		Options: []storage.QuestionOption{
			{Label: "Option A", Description: "Implement A"},
			{Label: "Option B", Description: "Implement B"},
		},
	}
}

// SampleSession returns a sample session for testing.
func SampleSession() *storage.Session {
	now := time.Now()

	return &storage.Session{
		Version: "1",
		Kind:    "Session",
		Metadata: storage.SessionMetadata{
			StartedAt: now,
			EndedAt:   now.Add(5 * time.Minute),
			Type:      "planning",
			Agent:     "claude",
			State:     "idle",
		},
		Exchanges: []storage.Exchange{
			{
				Role:      "system",
				Timestamp: now,
				Content:   "You are a helpful assistant.",
			},
			{
				Role:      "user",
				Timestamp: now,
				Content:   "Plan this task.",
			},
		},
	}
}

// WithTitle sets a custom title on a WorkUnit.
func WithTitle(title string) func(*workunit.WorkUnit) {
	return func(wu *workunit.WorkUnit) {
		wu.Title = title
	}
}

// WithExternalKey sets a custom external key on a WorkUnit.
func WithExternalKey(key string) func(*workunit.WorkUnit) {
	return func(wu *workunit.WorkUnit) {
		wu.ExternalKey = key
	}
}

// WithAgentConfig sets agent configuration on a WorkUnit.
func WithAgentConfig(cfg *workunit.AgentConfig) func(*workunit.WorkUnit) {
	return func(wu *workunit.WorkUnit) {
		wu.AgentConfig = cfg
	}
}

// WithTaskType sets the task type on a WorkUnit.
func WithTaskType(taskType string) func(*workunit.WorkUnit) {
	return func(wu *workunit.WorkUnit) {
		wu.TaskType = taskType
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Browser Fixtures
// ──────────────────────────────────────────────────────────────────────────────

// SampleTab returns a sample browser tab for testing.
func SampleTab(id, title, url string) browser.Tab {
	return browser.Tab{
		ID:    id,
		Title: title,
		URL:   url,
	}
}

// SampleConsoleMessage returns a sample console message for testing.
func SampleConsoleMessage(level, text string) browser.ConsoleMessage {
	return browser.ConsoleMessage{
		Level:     level,
		Text:      text,
		URL:       "https://example.com/script.js",
		Line:      42,
		Column:    10,
		Timestamp: time.Now(),
	}
}

// SampleNetworkRequest returns a sample network request for testing.
func SampleNetworkRequest(url, method string, status int) browser.NetworkRequest {
	return browser.NetworkRequest{
		ID:           "req-1",
		URL:          url,
		Method:       method,
		Status:       status,
		StatusText:   "OK",
		Headers:      map[string]string{"Content-Type": "application/json"},
		ResourceType: "XHR",
		MimeType:     "application/json",
		Timestamp:    time.Now(),
	}
}

// SampleDOMElement returns a sample DOM element for testing.
func SampleDOMElement(tag, text string) *browser.DOMElement {
	return &browser.DOMElement{
		NodeID:      1,
		BackendID:   100,
		TagName:     tag,
		TextContent: text,
		OuterHTML:   "<" + tag + ">" + text + "</" + tag + ">",
		Attributes:  map[string]string{"class": "test-class"},
		ChildCount:  0,
		Visible:     true,
		Interactive: true,
		X:           100,
		Y:           200,
	}
}

// SampleWebSocketFrame returns a sample WebSocket frame for testing.
func SampleWebSocketFrame(direction, data string) browser.WebSocketFrame {
	return browser.WebSocketFrame{
		ConnectionID: "ws-1",
		Direction:    direction,
		Data:         data,
		Opcode:       1, // Text frame
		Timestamp:    time.Now(),
	}
}

// SampleScriptSource returns a sample script source for testing.
func SampleScriptSource(scriptID, url string) browser.ScriptSource {
	return browser.ScriptSource{
		ScriptID: scriptID,
		URL:      url,
		Source:   "console.log('Hello, World!');",
		Length:   30,
	}
}

// SampleComputedStyle returns a sample computed style for testing.
func SampleComputedStyle(name, value string) browser.ComputedStyle {
	return browser.ComputedStyle{
		Name:  name,
		Value: value,
	}
}

// SampleCookie returns a sample browser cookie for testing.
func SampleCookie(name, value, domain string) browser.Cookie {
	return browser.Cookie{
		Name:     name,
		Value:    value,
		Domain:   domain,
		Path:     "/",
		Secure:   true,
		HTTPOnly: true,
		SameSite: "Lax",
		Expires:  time.Now().Add(24 * time.Hour).Unix(),
	}
}

// SampleCoverageSummary returns a sample coverage summary for testing.
func SampleCoverageSummary(jsUsed, jsTotal, cssUsed, cssTotal int) *browser.CoverageSummary {
	totalUsed := jsUsed + cssUsed
	totalSize := jsTotal + cssTotal
	jsPct := 0.0
	if jsTotal > 0 {
		jsPct = float64(jsUsed) / float64(jsTotal) * 100
	}
	cssPct := 0.0
	if cssTotal > 0 {
		cssPct = float64(cssUsed) / float64(cssTotal) * 100
	}
	totalPct := 0.0
	if totalSize > 0 {
		totalPct = float64(totalUsed) / float64(totalSize) * 100
	}

	return &browser.CoverageSummary{
		JSUsedBytes:     jsUsed,
		JSTotalBytes:    jsTotal,
		JSPercentage:    jsPct,
		CSSUsedBytes:    cssUsed,
		CSSTotalBytes:   cssTotal,
		CSSPercentage:   cssPct,
		TotalUsed:       totalUsed,
		TotalSize:       totalSize,
		TotalPercentage: totalPct,
	}
}

// SampleJSCoverageEntry returns a sample JS coverage entry for testing.
func SampleJSCoverageEntry(url string, usedBytes, totalBytes int) browser.JSCoverageEntry {
	return browser.JSCoverageEntry{
		ScriptID:   "script-1",
		URL:        url,
		UsedBytes:  usedBytes,
		TotalBytes: totalBytes,
		Ranges: []browser.CoverageRange{
			{StartOffset: 0, EndOffset: usedBytes, Count: 1},
		},
	}
}

// SampleCSSCoverageEntry returns a sample CSS coverage entry for testing.
func SampleCSSCoverageEntry(url string, usedBytes, totalBytes int) browser.CSSCoverageEntry {
	return browser.CSSCoverageEntry{
		StyleSheetID: "stylesheet-1",
		URL:          url,
		UsedBytes:    usedBytes,
		TotalBytes:   totalBytes,
		Ranges: []browser.CoverageRange{
			{StartOffset: 0, EndOffset: usedBytes},
		},
	}
}

// SampleMatchedStyles returns a sample matched styles structure for testing.
func SampleMatchedStyles() *browser.MatchedStyles {
	return &browser.MatchedStyles{
		InlineStyles: []browser.CSSProperty{
			{Name: "color", Value: "red", Important: false},
		},
		MatchedRules: []browser.MatchedRule{
			{
				Selector:  ".test-class",
				Origin:    "author",
				SourceURL: "styles.css",
				Properties: []browser.CSSProperty{
					{Name: "font-size", Value: "16px", Important: false},
				},
			},
		},
		InheritedStyles: []browser.InheritedStyleEntry{
			{
				MatchedRules: []browser.MatchedRule{
					{
						Selector:  "body",
						Origin:    "author",
						SourceURL: "styles.css",
						Properties: []browser.CSSProperty{
							{Name: "font-family", Value: "Arial", Important: false},
						},
					},
				},
			},
		},
	}
}
