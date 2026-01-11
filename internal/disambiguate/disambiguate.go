// Package disambiguate provides Symfony-style command prefix matching
// and interactive disambiguation for Cobra CLI applications.
package disambiguate

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// CommandMatch represents a command that matches a prefix.
type CommandMatch struct {
	Command     *cobra.Command
	MatchedName string   // The name that matched (command name)
	Path        []string // Full path for subcommands (e.g., ["config", "validate"])
}

// FindPrefixMatches finds all available commands matching the given prefix.
// Matching is case-insensitive.
func FindPrefixMatches(parent *cobra.Command, prefix string) []CommandMatch {
	var matches []CommandMatch
	prefix = strings.ToLower(prefix)

	for _, cmd := range parent.Commands() {
		if !cmd.IsAvailableCommand() || cmd.IsAdditionalHelpTopicCommand() {
			continue
		}

		name := strings.ToLower(cmd.Name())
		if strings.HasPrefix(name, prefix) {
			matches = append(matches, CommandMatch{
				Command:     cmd,
				MatchedName: cmd.Name(),
				Path:        []string{cmd.Name()},
			})
		}
	}

	return matches
}

// ResolveColonPath resolves a colon-separated command path like "c:v" to ["config", "validate"].
// Returns the resolved path segments, any ambiguous matches at the last level, and an error.
// If the path is unambiguous, the matches slice will have exactly one element or be empty.
func ResolveColonPath(root *cobra.Command, path string) ([]string, []CommandMatch, error) {
	if !strings.Contains(path, ":") {
		return nil, nil, errors.New("not a colon path")
	}

	segments := strings.Split(path, ":")
	resolved := make([]string, 0, len(segments))
	current := root

	for i, segment := range segments {
		if segment == "" {
			// Trailing colon - list subcommands of current
			if i == len(segments)-1 {
				matches := FindPrefixMatches(current, "")

				return resolved, matches, nil
			}

			continue
		}

		matches := FindPrefixMatches(current, segment)

		switch len(matches) {
		case 0:
			return resolved, nil, fmt.Errorf("no command matching %q in %s", segment, commandPath(resolved))
		case 1:
			resolved = append(resolved, matches[0].Command.Name())
			current = matches[0].Command
		default:
			// Ambiguous - return matches for user selection
			// Update paths to include full resolution so far
			for j := range matches {
				matches[j].Path = append(append([]string{}, resolved...), matches[j].Command.Name())
			}

			return resolved, matches, nil
		}
	}

	return resolved, nil, nil
}

// IsInteractive returns true if stdin is a terminal (TTY).
func IsInteractive() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

// SelectCommand prompts the user to select from matching commands.
// Returns an error if not in interactive mode or user cancels.
func SelectCommand(matches []CommandMatch, prefix string) (*CommandMatch, error) {
	if !IsInteractive() {
		return nil, fmt.Errorf("ambiguous command %q matches: %s (non-interactive mode)",
			prefix, formatMatchNames(matches))
	}

	// Build options with Cancel at the end
	options := make([]string, len(matches)+1)
	for i, m := range matches {
		options[i] = fmt.Sprintf("%s - %s", m.Command.Name(), m.Command.Short)
	}
	options[len(matches)] = "[Cancel]"

	var selected int
	prompt := &survey.Select{
		Message: fmt.Sprintf("Command %q is ambiguous. Select one:", prefix),
		Options: options,
	}

	if err := survey.AskOne(prompt, &selected); err != nil {
		return nil, errors.New("cancelled")
	}

	// Check if Cancel was selected
	if selected == len(matches) {
		return nil, errors.New("cancelled")
	}

	return &matches[selected], nil
}

// FormatAmbiguousError returns a formatted error message for ambiguous commands.
func FormatAmbiguousError(prefix string, matches []CommandMatch) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Command %q is ambiguous. Did you mean one of these?\n", prefix))
	for _, m := range matches {
		sb.WriteString(fmt.Sprintf("  %s - %s\n", m.Command.Name(), m.Command.Short))
	}

	return sb.String()
}

func formatMatchNames(matches []CommandMatch) string {
	names := make([]string, len(matches))
	for i, m := range matches {
		names[i] = m.Command.Name()
	}

	return strings.Join(names, ", ")
}

func commandPath(segments []string) string {
	if len(segments) == 0 {
		return "root"
	}

	return strings.Join(segments, " ")
}
