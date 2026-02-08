package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/valksor/go-mehrhof/internal/conductor"
	"github.com/valksor/go-mehrhof/internal/conductor/commands"
)

func TestParseStartInvocation_JSON(t *testing.T) {
	srv, err := New(Config{
		Mode:          ModeProject,
		WorkspaceRoot: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("New server failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/workflow/start", strings.NewReader(`{
		"content":"# Task",
		"template":"bugfix",
		"no_branch":true
	}`))
	req.Header.Set("Content-Type", "application/json")

	inv, err := srv.parseStartInvocation(req)
	if err != nil {
		t.Fatalf("parseStartInvocation returned error: %v", err)
	}
	if len(inv.Args) != 1 || !strings.HasPrefix(inv.Args[0], "file:") {
		t.Fatalf("unexpected args: %#v", inv.Args)
	}
	if inv.Source != "api" {
		t.Fatalf("unexpected source: %q", inv.Source)
	}
}

func TestParseStartInvocation_Form(t *testing.T) {
	srv, err := New(Config{
		Mode:          ModeProject,
		WorkspaceRoot: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("New server failed: %v", err)
	}

	form := url.Values{}
	form.Set("content", "# Task from form")
	form.Set("template", "bugfix")
	form.Set("no_branch", "true")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workflow/start", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	inv, err := srv.parseStartInvocation(req)
	if err != nil {
		t.Fatalf("parseStartInvocation returned error: %v", err)
	}
	if len(inv.Args) != 1 || !strings.HasPrefix(inv.Args[0], "file:") {
		t.Fatalf("unexpected args: %#v", inv.Args)
	}
}

func TestParseStartInvocation_Multipart(t *testing.T) {
	srv, err := New(Config{
		Mode:          ModeProject,
		WorkspaceRoot: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("New server failed: %v", err)
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "task.md")
	if err != nil {
		t.Fatalf("CreateFormFile failed: %v", err)
	}
	if _, err := part.Write([]byte("# Multipart task")); err != nil {
		t.Fatalf("write form file failed: %v", err)
	}
	_ = writer.WriteField("template", "feature")
	_ = writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/workflow/start", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	inv, err := srv.parseStartInvocation(req)
	if err != nil {
		t.Fatalf("parseStartInvocation returned error: %v", err)
	}
	if len(inv.Args) != 1 || !strings.HasPrefix(inv.Args[0], "file:") {
		t.Fatalf("unexpected args: %#v", inv.Args)
	}
}

func TestParseAnswerInvocation(t *testing.T) {
	t.Run("json", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/workflow/answer", strings.NewReader(`{"answer":"yes"}`))
		req.Header.Set("Content-Type", "application/json")

		inv, err := parseAnswerInvocation(req)
		if err != nil {
			t.Fatalf("parseAnswerInvocation returned error: %v", err)
		}
		if len(inv.Args) != 1 || inv.Args[0] != "yes" {
			t.Fatalf("unexpected args: %#v", inv.Args)
		}
	})

	t.Run("form", func(t *testing.T) {
		form := url.Values{}
		form.Set("answer", "form-answer")
		req := httptest.NewRequest(http.MethodPost, "/api/v1/workflow/answer", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		inv, err := parseAnswerInvocation(req)
		if err != nil {
			t.Fatalf("parseAnswerInvocation returned error: %v", err)
		}
		if len(inv.Args) != 1 || inv.Args[0] != "form-answer" {
			t.Fatalf("unexpected args: %#v", inv.Args)
		}
	})
}

func TestParseContinueInvocation(t *testing.T) {
	t.Run("empty body defaults", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/workflow/continue", http.NoBody)
		req.Header.Set("Content-Type", "application/json")

		inv, err := parseContinueInvocation(req)
		if err != nil {
			t.Fatalf("parseContinueInvocation returned error: %v", err)
		}
		if inv.Source != commands.SourceAPI {
			t.Fatalf("source=%q", inv.Source)
		}
		if commands.GetBool(inv.Options, "auto") {
			t.Fatalf("expected auto=false, got true")
		}
	})

	t.Run("auto true", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/workflow/continue", strings.NewReader(`{"auto":true}`))
		req.Header.Set("Content-Type", "application/json")

		inv, err := parseContinueInvocation(req)
		if err != nil {
			t.Fatalf("parseContinueInvocation returned error: %v", err)
		}
		if !commands.GetBool(inv.Options, "auto") {
			t.Fatalf("expected auto=true, got false")
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/workflow/continue", strings.NewReader("{"))
		req.Header.Set("Content-Type", "application/json")

		_, err := parseContinueInvocation(req)
		if err == nil || !strings.Contains(err.Error(), "invalid request body") {
			t.Fatalf("expected invalid request body error, got %v", err)
		}
	})
}

func TestParseAutoInvocation(t *testing.T) {
	t.Run("valid with defaults", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/workflow/auto", strings.NewReader(`{"ref":"file:task.md"}`))
		req.Header.Set("Content-Type", "application/json")

		inv, err := parseAutoInvocation(req)
		if err != nil {
			t.Fatalf("parseAutoInvocation returned error: %v", err)
		}
		if len(inv.Args) != 1 || inv.Args[0] != "file:task.md" {
			t.Fatalf("unexpected args: %#v", inv.Args)
		}
		if commands.GetInt(inv.Options, "max_retries") != 3 {
			t.Fatalf("max_retries=%d", commands.GetInt(inv.Options, "max_retries"))
		}
		if commands.GetString(inv.Options, "quality_target") != "quality" {
			t.Fatalf("quality_target=%q", commands.GetString(inv.Options, "quality_target"))
		}
	})

	t.Run("no quality overrides retries", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/workflow/auto", strings.NewReader(`{
			"ref":"file:task.md",
			"max_retries":5,
			"quality_target":"strict",
			"no_quality":true
		}`))
		req.Header.Set("Content-Type", "application/json")

		inv, err := parseAutoInvocation(req)
		if err != nil {
			t.Fatalf("parseAutoInvocation returned error: %v", err)
		}
		if commands.GetInt(inv.Options, "max_retries") != 0 {
			t.Fatalf("max_retries=%d", commands.GetInt(inv.Options, "max_retries"))
		}
		if !commands.GetBool(inv.Options, "no_quality") {
			t.Fatalf("expected no_quality=true")
		}
	})

	t.Run("missing ref", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/workflow/auto", strings.NewReader(`{}`))
		req.Header.Set("Content-Type", "application/json")

		_, err := parseAutoInvocation(req)
		if err == nil || !strings.Contains(err.Error(), "ref is required") {
			t.Fatalf("expected ref is required error, got %v", err)
		}
	})
}

func TestWriteCommandResultShapes(t *testing.T) {
	srv, err := New(Config{
		Mode:          ModeProject,
		WorkspaceRoot: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("New server failed: %v", err)
	}

	tests := []struct {
		name       string
		result     *commands.Result
		wantStatus int
		assertFn   func(t *testing.T, payload map[string]any)
	}{
		{
			name: "message with state and data",
			result: (&commands.Result{
				Type:    commands.ResultMessage,
				Message: "ok",
			}).WithState("planning").WithTaskID("task-1").WithData(map[string]any{
				"extra": "value",
			}),
			wantStatus: http.StatusOK,
			assertFn: func(t *testing.T, payload map[string]any) {
				t.Helper()
				if payload["success"] != true {
					t.Fatalf("success=%v", payload["success"])
				}
				if payload["message"] != "ok" {
					t.Fatalf("message=%v", payload["message"])
				}
				if payload["state"] != "planning" {
					t.Fatalf("state=%v", payload["state"])
				}
				if payload["task_id"] != "task-1" {
					t.Fatalf("task_id=%v", payload["task_id"])
				}
				if payload["extra"] != "value" {
					t.Fatalf("extra=%v", payload["extra"])
				}
			},
		},
		{
			name: "waiting",
			result: &commands.Result{
				Type:    commands.ResultWaiting,
				Message: "waiting",
				Data: commands.WaitingData{
					Question: "continue?",
					Options: []commands.QuestionOption{
						{Label: "Yes", Value: "yes", Description: "proceed"},
					},
					Phase: "planning",
				},
			},
			wantStatus: http.StatusOK,
			assertFn: func(t *testing.T, payload map[string]any) {
				t.Helper()
				if payload["success"] != true {
					t.Fatalf("success=%v", payload["success"])
				}
				if payload["status"] != "waiting" {
					t.Fatalf("status=%v", payload["status"])
				}
				if payload["question"] != "continue?" {
					t.Fatalf("question=%v", payload["question"])
				}
				if payload["phase"] != "planning" {
					t.Fatalf("phase=%v", payload["phase"])
				}
				if _, ok := payload["options"]; !ok {
					t.Fatalf("options missing in payload: %#v", payload)
				}
			},
		},
		{
			name: "paused",
			result: &commands.Result{
				Type:    commands.ResultPaused,
				Message: "paused",
			},
			wantStatus: http.StatusOK,
			assertFn: func(t *testing.T, payload map[string]any) {
				t.Helper()
				if payload["success"] != true {
					t.Fatalf("success=%v", payload["success"])
				}
				if payload["status"] != "paused" {
					t.Fatalf("status=%v", payload["status"])
				}
			},
		},
		{
			name: "stopped",
			result: &commands.Result{
				Type:    commands.ResultStopped,
				Message: "stopped",
			},
			wantStatus: http.StatusOK,
			assertFn: func(t *testing.T, payload map[string]any) {
				t.Helper()
				if payload["success"] != false {
					t.Fatalf("success=%v", payload["success"])
				}
				if payload["status"] != "stopped" {
					t.Fatalf("status=%v", payload["status"])
				}
			},
		},
		{
			name: "conflict",
			result: &commands.Result{
				Type:    commands.ResultConflict,
				Message: "conflict",
				Data: map[string]any{
					"conflict_type": "active_task",
				},
			},
			wantStatus: http.StatusConflict,
			assertFn: func(t *testing.T, payload map[string]any) {
				t.Helper()
				if payload["success"] != false {
					t.Fatalf("success=%v", payload["success"])
				}
				if payload["status"] != "conflict" {
					t.Fatalf("status=%v", payload["status"])
				}
				if payload["conflict_type"] != "active_task" {
					t.Fatalf("conflict_type=%v", payload["conflict_type"])
				}
			},
		},
		{
			name: "error",
			result: &commands.Result{
				Type:    commands.ResultError,
				Message: "boom",
			},
			wantStatus: http.StatusInternalServerError,
			assertFn: func(t *testing.T, payload map[string]any) {
				t.Helper()
				if payload["success"] != false {
					t.Fatalf("success=%v", payload["success"])
				}
				if payload["message"] != "boom" {
					t.Fatalf("message=%v", payload["message"])
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			srv.writeCommandResult(rr, tc.result)

			if rr.Code != tc.wantStatus {
				t.Fatalf("status=%d want=%d body=%s", rr.Code, tc.wantStatus, rr.Body.String())
			}

			var payload map[string]any
			if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			tc.assertFn(t, payload)
		})
	}
}

func TestHandleViaRouterMapsErrors(t *testing.T) {
	cond, err := conductor.New()
	if err != nil {
		t.Fatalf("conductor.New failed: %v", err)
	}

	srv, err := New(Config{
		Mode:          ModeProject,
		Conductor:     cond,
		WorkspaceRoot: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("New server failed: %v", err)
	}

	cmdNoTask := "adapter-test-no-active-task"
	commands.Register(commands.Command{
		Info: commands.CommandInfo{
			Name:        cmdNoTask,
			Description: "test error mapping no active task",
			Category:    "test",
		},
		Handler: func(context.Context, *conductor.Conductor, commands.Invocation) (*commands.Result, error) {
			return nil, commands.ErrNoActiveTask
		},
	})

	cmdGenericErr := "adapter-test-generic-error"
	commands.Register(commands.Command{
		Info: commands.CommandInfo{
			Name:        cmdGenericErr,
			Description: "test error mapping generic error",
			Category:    "test",
		},
		Handler: func(context.Context, *conductor.Conductor, commands.Invocation) (*commands.Result, error) {
			return nil, errors.New("boom")
		},
	})

	tests := []struct {
		name       string
		command    string
		wantStatus int
		wantErr    string
	}{
		{
			name:       "no active task maps to 400",
			command:    cmdNoTask,
			wantStatus: http.StatusBadRequest,
			wantErr:    "no active task",
		},
		{
			name:       "generic error maps to 500",
			command:    cmdGenericErr,
			wantStatus: http.StatusInternalServerError,
			wantErr:    "boom",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/test", nil)
			rr := httptest.NewRecorder()

			srv.handleViaRouter(CommandRoute{Command: tc.command})(rr, req)

			if rr.Code != tc.wantStatus {
				t.Fatalf("status=%d want=%d body=%s", rr.Code, tc.wantStatus, rr.Body.String())
			}
			if !strings.Contains(rr.Body.String(), tc.wantErr) {
				t.Fatalf("expected body to contain %q, got %s", tc.wantErr, rr.Body.String())
			}
		})
	}
}

func TestHandleViaRouterClassifiesExecutorErrors(t *testing.T) {
	cond, err := conductor.New()
	if err != nil {
		t.Fatalf("conductor.New failed: %v", err)
	}

	srv, err := New(Config{
		Mode:          ModeProject,
		Conductor:     cond,
		WorkspaceRoot: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("New server failed: %v", err)
	}

	cmdWithExecutor := "adapter-test-executor-error"
	commands.Register(commands.Command{
		Info: commands.CommandInfo{
			Name:         cmdWithExecutor,
			Description:  "test executor error classification",
			Category:     "test",
			MutatesState: true,
		},
		Handler: func(ctx context.Context, cond *conductor.Conductor, inv commands.Invocation) (*commands.Result, error) {
			result := commands.NewResult("started")
			result.Executor = func(ctx context.Context) error {
				return conductor.ErrBudgetPaused
			}

			return result, nil
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/test", nil)
	rr := httptest.NewRecorder()

	srv.handleViaRouter(CommandRoute{Command: cmdWithExecutor})(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d want=200 body=%s", rr.Code, rr.Body.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload["status"] != "paused" {
		t.Fatalf("expected status=paused, got %v", payload["status"])
	}
}
