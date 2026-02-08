package commands

import (
	"context"
	"errors"
	"runtime"

	"github.com/valksor/go-mehrhof/internal/conductor"
	"github.com/valksor/go-mehrhof/internal/events"
	"github.com/valksor/go-mehrhof/internal/sandbox"
	"github.com/valksor/go-mehrhof/internal/storage"
)

func init() {
	Register(Command{
		Info: CommandInfo{
			Name:         "sandbox-status",
			Description:  "Get current sandbox status",
			Category:     "info",
			RequiresTask: false,
		},
		Handler: handleSandboxStatus,
	})

	Register(Command{
		Info: CommandInfo{
			Name:         "sandbox-enable",
			Description:  "Enable sandbox for the workspace",
			Category:     "control",
			RequiresTask: false,
			MutatesState: true,
		},
		Handler: handleSandboxEnable,
	})

	Register(Command{
		Info: CommandInfo{
			Name:         "sandbox-disable",
			Description:  "Disable sandbox for the workspace",
			Category:     "control",
			RequiresTask: false,
			MutatesState: true,
		},
		Handler: handleSandboxDisable,
	})
}

func handleSandboxStatus(_ context.Context, cond *conductor.Conductor, _ Invocation) (*Result, error) {
	cfg := loadWorkspaceConfigOrDefault(cond)

	status := sandbox.Status{
		Enabled:   cfg.Sandbox != nil && cfg.Sandbox.Enabled,
		Platform:  runtime.GOOS,
		Active:    isSandboxActive(cond),
		Network:   cfg.Sandbox != nil && cfg.Sandbox.Network,
		Supported: sandbox.Supported(),
	}

	return NewResult("Sandbox status loaded").WithData(status), nil
}

func handleSandboxEnable(_ context.Context, cond *conductor.Conductor, _ Invocation) (*Result, error) {
	ws := cond.GetWorkspace()
	if ws == nil {
		return nil, errors.New("workspace not available")
	}

	cfg, err := ws.LoadConfig()
	if err != nil {
		cfg = storage.NewDefaultWorkspaceConfig()
	}

	if cfg.Sandbox == nil {
		cfg.Sandbox = &storage.SandboxSettings{}
	}
	cfg.Sandbox.Enabled = true

	if err := ws.SaveConfig(cfg); err != nil {
		return nil, errors.New("failed to save config: " + err.Error())
	}

	// Publish SSE event
	if bus := cond.GetEventBus(); bus != nil {
		bus.Publish(events.SandboxStatusChangedEvent{
			Enabled:  true,
			Active:   false,
			Platform: runtime.GOOS,
		})
	}

	status := sandbox.Status{
		Enabled:  true,
		Platform: runtime.GOOS,
		Active:   false,
		Network:  cfg.Sandbox.Network,
	}

	return NewResult("Sandbox enabled").WithData(status), nil
}

func handleSandboxDisable(_ context.Context, cond *conductor.Conductor, _ Invocation) (*Result, error) {
	ws := cond.GetWorkspace()
	if ws == nil {
		return nil, errors.New("workspace not available")
	}

	cfg, err := ws.LoadConfig()
	if err != nil {
		cfg = storage.NewDefaultWorkspaceConfig()
	}

	if cfg.Sandbox != nil {
		cfg.Sandbox.Enabled = false
	}

	if err := ws.SaveConfig(cfg); err != nil {
		return nil, errors.New("failed to save config: " + err.Error())
	}

	// Publish SSE event
	if bus := cond.GetEventBus(); bus != nil {
		bus.Publish(events.SandboxStatusChangedEvent{
			Enabled:  false,
			Active:   false,
			Platform: runtime.GOOS,
		})
	}

	status := sandbox.Status{
		Enabled:  false,
		Platform: runtime.GOOS,
		Active:   false,
		Network:  true,
	}

	return NewResult("Sandbox disabled").WithData(status), nil
}

// loadWorkspaceConfigOrDefault loads workspace config, falling back to defaults.
func loadWorkspaceConfigOrDefault(cond *conductor.Conductor) *storage.WorkspaceConfig {
	if cond != nil {
		if ws := cond.GetWorkspace(); ws != nil {
			if cfg, err := ws.LoadConfig(); err == nil {
				return cfg
			}
		}
	}

	return storage.NewDefaultWorkspaceConfig()
}

// isSandboxActive checks if sandbox is currently active for a running task.
func isSandboxActive(cond *conductor.Conductor) bool {
	if cond == nil {
		return false
	}

	activeTask := cond.GetActiveTask()
	if activeTask == nil || activeTask.State != "implementing" {
		return false
	}

	ws := cond.GetWorkspace()
	if ws == nil {
		return false
	}

	cfg, err := ws.LoadConfig()
	if err != nil {
		return false
	}

	return cfg.Sandbox != nil && cfg.Sandbox.Enabled
}
