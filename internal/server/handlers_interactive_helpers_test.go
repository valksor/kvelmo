package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/valksor/go-mehrhof/internal/conductor"
	"github.com/valksor/go-mehrhof/internal/conductor/commands"
)

func TestParseChatRequest(t *testing.T) {
	t.Run("json", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"message":"hello"}`))
		req.Header.Set("Content-Type", "application/json")
		got, err := parseChatRequest(req)
		if err != nil {
			t.Fatalf("parseChatRequest error: %v", err)
		}
		if got.Message != "hello" {
			t.Fatalf("message = %q", got.Message)
		}
	})

	t.Run("form", func(t *testing.T) {
		form := url.Values{}
		form.Set("message", "hi form")
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		got, err := parseChatRequest(req)
		if err != nil {
			t.Fatalf("parseChatRequest error: %v", err)
		}
		if got.Message != "hi form" {
			t.Fatalf("message = %q", got.Message)
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{bad`))
		req.Header.Set("Content-Type", "application/json")
		_, err := parseChatRequest(req)
		if err == nil || !strings.Contains(err.Error(), "invalid JSON") {
			t.Fatalf("expected invalid JSON error, got %v", err)
		}
	})
}

func TestParseCommandRequest(t *testing.T) {
	t.Run("json", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"command":"review","args":["view","2"]}`))
		req.Header.Set("Content-Type", "application/json")
		got, err := parseCommandRequest(req)
		if err != nil {
			t.Fatalf("parseCommandRequest error: %v", err)
		}
		if got.Command != "review" || len(got.Args) != 2 {
			t.Fatalf("unexpected parsed command: %#v", got)
		}
	})

	t.Run("form comma args", func(t *testing.T) {
		form := url.Values{}
		form.Set("command", "label")
		form.Set("args", "add,bug,backend")
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		got, err := parseCommandRequest(req)
		if err != nil {
			t.Fatalf("parseCommandRequest error: %v", err)
		}
		if got.Command != "label" || len(got.Args) != 1 || got.Args[0] != "add,bug,backend" {
			t.Fatalf("unexpected parsed command: %#v", got)
		}
	})

	t.Run("form repeated args", func(t *testing.T) {
		form := url.Values{}
		form.Set("command", "label")
		form.Add("args", "add")
		form.Add("args", "bug")
		form.Add("args", "backend")
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		got, err := parseCommandRequest(req)
		if err != nil {
			t.Fatalf("parseCommandRequest error: %v", err)
		}
		if got.Command != "label" || len(got.Args) != 3 {
			t.Fatalf("unexpected parsed command: %#v", got)
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{bad`))
		req.Header.Set("Content-Type", "application/json")
		_, err := parseCommandRequest(req)
		if err == nil || !strings.Contains(err.Error(), "invalid JSON") {
			t.Fatalf("expected invalid JSON error, got %v", err)
		}
	})
}

func TestRouterResultToJSON(t *testing.T) {
	srv, err := New(Config{Mode: ModeProject})
	if err != nil {
		t.Fatalf("New server failed: %v", err)
	}

	cases := []*commands.Result{
		{
			Type:    commands.ResultStatus,
			Message: "ok",
			State:   "planning",
			TaskID:  "task-1",
			Data: commands.StatusData{
				TaskID:    "task-1",
				Title:     "Title",
				State:     "planning",
				Branch:    "feat/x",
				SpecCount: 2,
			},
		},
		{
			Type:    commands.ResultCost,
			Message: "cost",
			Data: commands.CostData{
				TotalTokens: 123,
			},
		},
		{
			Type:    commands.ResultBudget,
			Message: "budget",
			Data: commands.BudgetData{
				Type: "cost",
				Used: "$1.00",
				Max:  "$2.00",
			},
		},
		{
			Type:    commands.ResultList,
			Message: "list",
			Data:    []commands.TaskListItem{{ID: "t"}},
		},
		{
			Type:    commands.ResultHelp,
			Message: "help",
			Data:    []commands.CommandInfo{{Name: "status"}},
		},
		{
			Type:    commands.ResultMessage,
			Message: "message",
			Data:    map[string]string{"x": "y"},
		},
		{
			Type:    commands.ResultError,
			Message: "boom",
		},
	}

	for _, r := range cases {
		got := srv.routerResultToJSON(r)
		success, ok := got["success"].(bool)
		if !ok {
			t.Fatalf("success type missing for %s", r.Type)
		}
		if r.Type == commands.ResultError && success {
			t.Fatalf("error result should set success=false")
		}
		if r.Type != commands.ResultError && !success {
			t.Fatalf("non-error result should set success=true")
		}
	}
}

func TestBuildChatPromptAndInteractiveRoutes(t *testing.T) {
	cond, err := conductor.New()
	if err != nil {
		t.Fatalf("conductor.New failed: %v", err)
	}
	srv, err := New(Config{Mode: ModeProject, Conductor: cond})
	if err != nil {
		t.Fatalf("New server failed: %v", err)
	}

	prompt := srv.buildChatPrompt("hello")
	if !strings.Contains(prompt, "You are an AI assistant") || !strings.Contains(prompt, "User message: hello") {
		t.Fatalf("unexpected prompt: %q", prompt)
	}
}

func TestInteractiveCommandsEndpointDirect(t *testing.T) {
	srv, err := New(Config{Mode: ModeProject})
	if err != nil {
		t.Fatalf("New server failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/interactive/commands", nil)
	rr := httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}

	var body map[string]any
	if err := json.NewDecoder(bytes.NewReader(rr.Body.Bytes())).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if _, ok := body["commands"]; !ok {
		t.Fatalf("commands key missing in response")
	}
}

func TestNoInteractiveSwitchCaseFallback(t *testing.T) {
	interactiveCommands := []string{
		"reset", "auto", "find", "simplify", "label",
		"memory", "library", "links", "question",
		"delete", "export", "optimize", "submit", "sync",
	}

	for _, cmd := range interactiveCommands {
		if !commands.IsKnownCommand(cmd) {
			t.Fatalf("command %q must be registered in router", cmd)
		}
	}
}
