//go:build !testbinary
// +build !testbinary

package commands

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/valksor/go-mehrhof/internal/conductor"
	routercommands "github.com/valksor/go-mehrhof/internal/conductor/commands"
	"github.com/valksor/go-mehrhof/internal/workflow"
)

func mustNewConductorForInteractive(t *testing.T) *conductor.Conductor {
	t.Helper()

	cond, err := conductor.New()
	if err != nil {
		t.Fatalf("conductor.New failed: %v", err)
	}

	return cond
}

func TestInteractiveSessionUtilityMethods(t *testing.T) {
	s := newInteractiveSession(mustNewConductorForInteractive(t))
	if s == nil || s.sessionID == "" {
		t.Fatalf("session not initialized correctly: %#v", s)
	}
	if s.state != workflow.StateIdle {
		t.Fatalf("initial state = %q, want idle", s.state)
	}

	prompt := s.getPrompt()
	if !strings.Contains(prompt, "mehrhof (") {
		t.Fatalf("prompt = %q", prompt)
	}
	if s.getCompleter() == nil {
		t.Fatalf("expected non-nil completer")
	}

	chatPrompt := s.buildChatPrompt("hello")
	if !strings.Contains(chatPrompt, "You are an AI assistant") || !strings.Contains(chatPrompt, "User message: hello") {
		t.Fatalf("unexpected chat prompt: %q", chatPrompt)
	}

	called := false
	s.cancelFunc = func() { called = true }
	s.cancelCurrentOperation()
	if !called || s.cancelFunc != nil {
		t.Fatalf("cancelCurrentOperation did not clear cancel function")
	}
}

func TestInteractiveSessionValidationPaths(t *testing.T) {
	s := newInteractiveSession(mustNewConductorForInteractive(t))
	ctx := context.Background()

	if err := s.handleChat(ctx, ""); err == nil || !strings.Contains(err.Error(), "message cannot be empty") {
		t.Fatalf("expected empty-message error, got %v", err)
	}
	if err := s.handleFind(ctx, nil); err == nil || !strings.Contains(err.Error(), "usage: find") {
		t.Fatalf("expected usage error, got %v", err)
	}
	if err := s.handleSimplify(ctx, nil); err == nil || !strings.Contains(err.Error(), "no active task") {
		t.Fatalf("expected no active task error, got %v", err)
	}
	if err := s.handleLabelAdd(ctx, "", []string{"bug"}); err == nil || !strings.Contains(err.Error(), "no active task") {
		t.Fatalf("expected no active task error, got %v", err)
	}
	if err := s.handleLabelRemove(ctx, "task-1", nil); err == nil || !strings.Contains(err.Error(), "usage: label remove") {
		t.Fatalf("expected usage error, got %v", err)
	}
	if err := s.handleLabelSet(ctx, "", []string{"x"}); err == nil || !strings.Contains(err.Error(), "no active task") {
		t.Fatalf("expected no active task error, got %v", err)
	}
	if err := s.handleMemory(ctx, nil); err == nil || !strings.Contains(err.Error(), "usage: memory") {
		t.Fatalf("expected usage error, got %v", err)
	}
	if err := s.handleLibrary(ctx, nil); err == nil || !strings.Contains(err.Error(), "library system is not enabled") {
		t.Fatalf("expected library-disabled error, got %v", err)
	}
}

func TestInteractiveSessionExecuteAndRender(t *testing.T) {
	s := newInteractiveSession(mustNewConductorForInteractive(t))
	ctx := context.Background()

	// Unknown commands fall back to chat path (no agent available in this setup).
	err := s.executeCommand(ctx, "unknown-cmd", nil, "hello")
	if err == nil || !strings.Contains(err.Error(), "no agent available") {
		t.Fatalf("expected chat fallback error, got %v", err)
	}

	// Router command that fails without active task gets mapped in handleCommand.
	err = s.handleCommand(ctx, "cost")
	if err == nil || !errors.Is(err, errors.New("x")) && !strings.Contains(err.Error(), "no active task - use 'start <reference>' first") {
		t.Fatalf("expected mapped no-active-task error, got %v", err)
	}

	// Exercise render branches for coverage and regression safety.
	s.renderResult(&routercommands.Result{Type: routercommands.ResultMessage, Message: "ok"})
	s.renderResult(&routercommands.Result{Type: routercommands.ResultError, Message: "boom"})
	s.renderResult(&routercommands.Result{Type: routercommands.ResultChat, Message: "chat"})
	s.renderResult(&routercommands.Result{Type: routercommands.ResultQuestion, Message: "q?"})
	s.renderResult(&routercommands.Result{
		Type:    routercommands.ResultStatus,
		Message: "status",
		Data: routercommands.StatusData{
			TaskID: "t1", State: "planning", SpecCount: 1,
		},
	})
	s.renderResult(&routercommands.Result{
		Type:    routercommands.ResultCost,
		Message: "cost",
		Data:    routercommands.CostData{InputTokens: 1, OutputTokens: 2, TotalTokens: 3},
	})
	s.renderResult(&routercommands.Result{
		Type:    routercommands.ResultBudget,
		Message: "budget",
		Data:    routercommands.BudgetData{Type: "cost", Used: "$1", Max: "$2", Percentage: 50},
	})
	s.renderResult(&routercommands.Result{
		Type:    routercommands.ResultList,
		Message: "tasks",
		Data:    []routercommands.TaskListItem{{ID: "t1", Title: "Task", State: "idle"}},
	})
	s.renderResult(&routercommands.Result{
		Type:    routercommands.ResultSpecifications,
		Message: "specs",
		Data:    []routercommands.SpecificationItem{{Number: 1, Title: "Spec", Status: "open"}},
	})
	s.renderResult(&routercommands.Result{
		Type:    routercommands.ResultHelp,
		Message: "help",
		Data:    []routercommands.CommandInfo{{Name: "status", Category: "info", Description: "Show status"}},
	})
}

func TestInteractiveStandaloneHelpers(t *testing.T) {
	if got := formatSize(512); got != "512 B" {
		t.Fatalf("formatSize(512) = %q", got)
	}
	if got := capitalizeFirst("hello"); got != "Hello" {
		t.Fatalf("capitalizeFirst = %q", got)
	}
	if got := capitalizeFirst(""); got != "" {
		t.Fatalf("capitalizeFirst empty = %q", got)
	}
}
