//go:build !testbinary
// +build !testbinary

package commands

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/valksor/go-mehrhof/internal/conductor"
	"github.com/valksor/go-mehrhof/internal/helper_test"
	"github.com/valksor/go-mehrhof/internal/storage"
)

func TestFinishCommand_Properties(t *testing.T) {
	if finishCmd.Use != "finish" {
		t.Errorf("Use = %q, want %q", finishCmd.Use, "finish")
	}

	if finishCmd.Short == "" {
		t.Error("Short description is empty")
	}

	if finishCmd.Long == "" {
		t.Error("Long description is empty")
	}

	if finishCmd.RunE == nil {
		t.Error("RunE not set")
	}
}

func TestFinishCommand_Flags(t *testing.T) {
	tests := []struct {
		name         string
		flagName     string
		shorthand    string
		defaultValue string
	}{
		{
			name:         "yes flag",
			flagName:     "yes",
			shorthand:    "y",
			defaultValue: "false",
		},
		{
			name:         "merge flag",
			flagName:     "merge",
			shorthand:    "",
			defaultValue: "false",
		},
		{
			name:         "delete flag",
			flagName:     "delete",
			shorthand:    "",
			defaultValue: "false",
		},
		{
			name:         "push flag",
			flagName:     "push",
			shorthand:    "",
			defaultValue: "false",
		},
		{
			name:         "squash flag",
			flagName:     "squash",
			shorthand:    "",
			defaultValue: "false",
		},
		{
			name:         "target flag",
			flagName:     "target",
			shorthand:    "t",
			defaultValue: "",
		},
		{
			name:         "no-quality flag",
			flagName:     "no-quality",
			shorthand:    "",
			defaultValue: "false",
		},
		{
			name:         "quality-target flag",
			flagName:     "quality-target",
			shorthand:    "",
			defaultValue: "quality",
		},
		{
			name:         "delete-work flag",
			flagName:     "delete-work",
			shorthand:    "",
			defaultValue: "false",
		},
		{
			name:         "draft flag",
			flagName:     "draft",
			shorthand:    "",
			defaultValue: "false",
		},
		{
			name:         "pr-title flag",
			flagName:     "pr-title",
			shorthand:    "",
			defaultValue: "",
		},
		{
			name:         "pr-body flag",
			flagName:     "pr-body",
			shorthand:    "",
			defaultValue: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := finishCmd.Flags().Lookup(tt.flagName)
			if flag == nil {
				t.Errorf("flag %q not found", tt.flagName)

				return
			}

			if flag.DefValue != tt.defaultValue {
				t.Errorf("flag %q default value = %q, want %q", tt.flagName, flag.DefValue, tt.defaultValue)
			}

			if tt.shorthand != "" {
				shorthand := finishCmd.Flags().ShorthandLookup(tt.shorthand)
				if shorthand == nil {
					t.Errorf("shorthand %q not found for flag %q", tt.shorthand, tt.flagName)
				}
			}
		})
	}
}

func TestFinishCommand_ShortDescription(t *testing.T) {
	expected := "Complete the task (creates PR by default for supported providers)"
	if finishCmd.Short != expected {
		t.Errorf("Short = %q, want %q", finishCmd.Short, expected)
	}
}

func TestFinishCommand_LongDescriptionContains(t *testing.T) {
	contains := []string{
		"Complete the current task",
		"pull request",
		"merge",
		"PROVIDER BEHAVIOR",
	}

	for _, substr := range contains {
		if !containsString(finishCmd.Long, substr) {
			t.Errorf("Long description does not contain %q", substr)
		}
	}
}

func TestFinishCommand_DocumentsProviderBehaviors(t *testing.T) {
	providers := []string{
		"github:",
		"gitlab:",
		"file:, dir:",
		"jira:",
	}

	for _, provider := range providers {
		if !containsString(finishCmd.Long, provider) {
			t.Errorf("Long description does not document provider %q", provider)
		}
	}
}

func TestFinishCommand_DocumentsFlagCombinations(t *testing.T) {
	if !containsString(finishCmd.Long, "FLAG COMBINATIONS") {
		t.Error("Long description does not document FLAG COMBINATIONS section")
	}

	if !containsString(finishCmd.Long, "PR mode") {
		t.Error("Long description does not mention PR mode")
	}

	if !containsString(finishCmd.Long, "Merge mode") {
		t.Error("Long description does not mention Merge mode")
	}
}

func TestFinishCommand_NoAliases(t *testing.T) {
	// Aliases removed in favor of prefix matching
	if len(finishCmd.Aliases) > 0 {
		t.Errorf("finish command should have no aliases, got %v", finishCmd.Aliases)
	}
}

func TestFinishCommand_Examples(t *testing.T) {
	examples := []string{
		"mehr finish",
		"--yes",
		"--merge",
		"--delete",
		"--push",
		"--squash",
		"--target",
		"--no-quality",
		"--draft",
		"--pr-title",
		"--delete-work",
	}

	for _, example := range examples {
		if !containsString(finishCmd.Long, example) {
			t.Errorf("Long description does not contain example %q", example)
		}
	}
}

func TestFinishCommand_RegisteredInRoot(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "finish" {
			found = true

			break
		}
	}
	if !found {
		t.Error("finish command not registered in root command")
	}
}

func TestFinishCommand_YesFlagHasShorthand(t *testing.T) {
	flag := finishCmd.Flags().Lookup("yes")
	if flag == nil {
		t.Fatal("yes flag not found")

		return
	}
	if flag.Shorthand != "y" {
		t.Errorf("yes flag shorthand = %q, want 'y'", flag.Shorthand)
	}
}

func TestFinishCommand_TargetFlagHasShorthand(t *testing.T) {
	flag := finishCmd.Flags().Lookup("target")
	if flag == nil {
		t.Fatal("target flag not found")

		return
	}
	if flag.Shorthand != "t" {
		t.Errorf("target flag shorthand = %q, want 't'", flag.Shorthand)
	}
}

func TestFinishCommand_QualityTargetDefault(t *testing.T) {
	flag := finishCmd.Flags().Lookup("quality-target")
	if flag == nil {
		t.Fatal("quality-target flag not found")

		return
	}
	if flag.DefValue != "quality" {
		t.Errorf("quality-target default = %q, want 'quality'", flag.DefValue)
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// runFinishLogic behavioral tests
// ──────────────────────────────────────────────────────────────────────────────

func TestRunFinishLogic_NoActiveTask(t *testing.T) {
	mock := helper_test.NewMockConductor()
	// No active task set

	opts := finishOptions{skipQuality: true}
	err := runFinishLogic(context.Background(), mock, opts, nil)

	if err == nil {
		t.Error("expected error for no active task")
	}
	if err != nil && err.Error() != "no active task" {
		t.Errorf("error = %q, want %q", err.Error(), "no active task")
	}
}

func TestRunFinishLogic_CallsRunQuality(t *testing.T) {
	mock := helper_test.NewMockConductor().
		WithActiveTask(&storage.ActiveTask{ID: "test", State: "implementing"}).
		WithRunQualityResult(&conductor.QualityResult{Ran: true, Passed: true})

	var stdout bytes.Buffer
	opts := finishOptions{skipQuality: false, qualityTarget: "quality"}
	err := runFinishLogic(context.Background(), mock, opts, &stdout)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(mock.RunQualityCalls) != 1 {
		t.Errorf("RunQuality called %d times, want 1", len(mock.RunQualityCalls))
	}
	if mock.RunQualityCalls[0].Target != "quality" {
		t.Errorf("RunQuality target = %q, want %q", mock.RunQualityCalls[0].Target, "quality")
	}
}

func TestRunFinishLogic_SkipsQuality(t *testing.T) {
	mock := helper_test.NewMockConductor().
		WithActiveTask(&storage.ActiveTask{ID: "test", State: "implementing"})

	opts := finishOptions{skipQuality: true}
	err := runFinishLogic(context.Background(), mock, opts, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(mock.RunQualityCalls) != 0 {
		t.Errorf("RunQuality called %d times, want 0", len(mock.RunQualityCalls))
	}
}

func TestRunFinishLogic_CallsFinish(t *testing.T) {
	mock := helper_test.NewMockConductor().
		WithActiveTask(&storage.ActiveTask{ID: "test", State: "implementing"})

	opts := finishOptions{
		skipQuality:  true,
		targetBranch: "main",
		delete:       true,
	}
	err := runFinishLogic(context.Background(), mock, opts, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(mock.FinishCalls) != 1 {
		t.Fatalf("Finish called %d times, want 1", len(mock.FinishCalls))
	}
	if mock.FinishCalls[0].TargetBranch != "main" {
		t.Errorf("TargetBranch = %q, want %q", mock.FinishCalls[0].TargetBranch, "main")
	}
	if !mock.FinishCalls[0].DeleteBranch {
		t.Error("DeleteBranch = false, want true")
	}
}

func TestRunFinishLogic_MergeMode(t *testing.T) {
	mock := helper_test.NewMockConductor().
		WithActiveTask(&storage.ActiveTask{ID: "test", State: "implementing"})

	opts := finishOptions{
		skipQuality: true,
		merge:       true,
		push:        true,
		squash:      true,
	}
	err := runFinishLogic(context.Background(), mock, opts, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(mock.FinishCalls) != 1 {
		t.Fatalf("Finish called %d times, want 1", len(mock.FinishCalls))
	}
	if !mock.FinishCalls[0].ForceMerge {
		t.Error("ForceMerge = false, want true")
	}
	if !mock.FinishCalls[0].PushAfter {
		t.Error("PushAfter = false, want true")
	}
	if !mock.FinishCalls[0].SquashMerge {
		t.Error("SquashMerge = false, want true")
	}
}

func TestRunFinishLogic_PRMode(t *testing.T) {
	mock := helper_test.NewMockConductor().
		WithActiveTask(&storage.ActiveTask{ID: "test", State: "implementing"})

	opts := finishOptions{
		skipQuality: true,
		draftPR:     true,
		prTitle:     "My PR Title",
		prBody:      "PR description",
	}
	err := runFinishLogic(context.Background(), mock, opts, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(mock.FinishCalls) != 1 {
		t.Fatalf("Finish called %d times, want 1", len(mock.FinishCalls))
	}
	if !mock.FinishCalls[0].DraftPR {
		t.Error("DraftPR = false, want true")
	}
	if mock.FinishCalls[0].PRTitle != "My PR Title" {
		t.Errorf("PRTitle = %q, want %q", mock.FinishCalls[0].PRTitle, "My PR Title")
	}
	if mock.FinishCalls[0].PRBody != "PR description" {
		t.Errorf("PRBody = %q, want %q", mock.FinishCalls[0].PRBody, "PR description")
	}
}

func TestRunFinishLogic_PropagatesQualityError(t *testing.T) {
	qualityErr := errors.New("lint failed")
	mock := helper_test.NewMockConductor().
		WithActiveTask(&storage.ActiveTask{ID: "test", State: "implementing"}).
		WithRunQualityError(qualityErr)

	opts := finishOptions{skipQuality: false}
	err := runFinishLogic(context.Background(), mock, opts, nil)

	if err == nil {
		t.Error("expected error")
	}
	if !errors.Is(err, qualityErr) {
		t.Errorf("error = %v, want wrapped %v", err, qualityErr)
	}
}

func TestRunFinishLogic_PropagatesFinishError(t *testing.T) {
	finishErr := errors.New("merge conflict")
	mock := helper_test.NewMockConductor().
		WithActiveTask(&storage.ActiveTask{ID: "test", State: "implementing"}).
		WithFinishError(finishErr)

	opts := finishOptions{skipQuality: true}
	err := runFinishLogic(context.Background(), mock, opts, nil)

	if err == nil {
		t.Error("expected error")
	}
	if !errors.Is(err, finishErr) {
		t.Errorf("error = %v, want wrapped %v", err, finishErr)
	}
}

func TestRunFinishLogic_QualityUserAborted(t *testing.T) {
	mock := helper_test.NewMockConductor().
		WithActiveTask(&storage.ActiveTask{ID: "test", State: "implementing"}).
		WithRunQualityResult(&conductor.QualityResult{Ran: true, UserAborted: true})

	opts := finishOptions{skipQuality: false}
	err := runFinishLogic(context.Background(), mock, opts, nil)
	// User abort should return nil (not an error)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	// Finish should NOT have been called (user cancelled)
	if len(mock.FinishCalls) != 0 {
		t.Errorf("Finish called %d times, want 0 (user aborted)", len(mock.FinishCalls))
	}
}

func TestRunFinishLogic_SquashGeneratesCommitMessage(t *testing.T) {
	mock := helper_test.NewMockConductor().
		WithActiveTask(&storage.ActiveTask{ID: "test", State: "implementing"}).
		WithCommitMessagePreview("feat: add feature\n\nDetails here")

	var stdout bytes.Buffer
	opts := finishOptions{skipQuality: true, squash: true}
	err := runFinishLogic(context.Background(), mock, opts, &stdout)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if mock.CommitMessagePreviewCalls != 1 {
		t.Errorf("GenerateCommitMessagePreview called %d times, want 1", mock.CommitMessagePreviewCalls)
	}
	if len(mock.FinishCalls) != 1 {
		t.Fatalf("Finish called %d times, want 1", len(mock.FinishCalls))
	}
	if mock.FinishCalls[0].CommitMessage != "feat: add feature\n\nDetails here" {
		t.Errorf("CommitMessage = %q, want %q", mock.FinishCalls[0].CommitMessage, "feat: add feature\n\nDetails here")
	}
}
