package commands

// CommandArg describes a command argument for discovery.
type CommandArg struct {
	Name        string `json:"name"`
	Required    bool   `json:"required"`
	Description string `json:"description,omitempty"`
}

// CommandInfo contains metadata about a command for discovery.
// This is returned by the /api/v1/interactive/commands endpoint
// so IDE plugins can auto-discover available commands.
type CommandInfo struct {
	Name         string       `json:"name"`
	Aliases      []string     `json:"aliases,omitempty"`
	Description  string       `json:"description"`
	Category     string       `json:"category"`
	Args         []CommandArg `json:"args,omitempty"`
	RequiresTask bool         `json:"requires_task"`
	MutatesState bool         `json:"mutates_state"`
	Subcommands  []string     `json:"subcommands,omitempty"`
}

// Metadata returns all registered command info for discovery.
func Metadata() []CommandInfo {
	var cmds []CommandInfo
	for _, cmd := range registry {
		cmds = append(cmds, cmd.Info)
	}

	return cmds
}

// Categories groups commands by their category.
func Categories() map[string][]CommandInfo {
	categories := make(map[string][]CommandInfo)
	for _, cmd := range registry {
		categories[cmd.Info.Category] = append(categories[cmd.Info.Category], cmd.Info)
	}

	return categories
}

// GetCommandInfo returns info for a specific command by name or alias.
func GetCommandInfo(name string) (CommandInfo, bool) {
	if cmd, ok := registry[name]; ok {
		return cmd.Info, true
	}
	if canonical, ok := aliases[name]; ok {
		if cmd, ok := registry[canonical]; ok {
			return cmd.Info, true
		}
	}

	return CommandInfo{}, false
}
