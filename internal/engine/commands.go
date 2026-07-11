package engine

import (
	"strings"

	"github.com/crimsab/oneday/internal/game/contracts"
)

// Command represents a parsed chat command.
type Command struct {
	Name string   // canonical name e.g. "inventory", "stats", "save", "load", "help", "quit"
	Args []string // arguments after the command name (may be empty)
}

// CommandRegistry maps command names (including aliases) to canonical names.
var CommandRegistry = contracts.CommandAliasRegistry()

// IsCommand returns true if the input starts with "/".
func IsCommand(input string) bool {
	return strings.HasPrefix(strings.TrimSpace(input), "/")
}

// ParseCommand checks if input is a command (starts with "/").
// Returns nil if not a command.
func ParseCommand(input string) *Command {
	input = strings.TrimSpace(input)
	if !strings.HasPrefix(input, "/") {
		return nil
	}

	// Strip leading slash.
	input = input[1:]

	// Split by whitespace.
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return &Command{Name: "unknown", Args: []string{"/"}}
	}

	name := strings.ToLower(parts[0])
	args := parts[1:]

	canonical, ok := CommandRegistry[name]
	if !ok {
		return &Command{Name: "unknown", Args: []string{name}}
	}

	return &Command{Name: canonical, Args: args}
}
