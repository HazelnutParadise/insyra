package dsl

import "github.com/HazelnutParadise/insyra/cli/commands"

// ExecContext is the state a registered command reads and writes.
type ExecContext = commands.ExecContext

// CommandHandler describes a command available to DSL sessions and the CLI.
type CommandHandler = commands.CommandHandler

// CommandFlag declares a flag accepted by a one-shot CLI command.
type CommandFlag = commands.CommandFlag

// ArgLimit declares the maximum number of arguments accepted by a command.
type ArgLimit = commands.ArgLimit

// Register adds a command to the shared registry used by DSL sessions and the
// CLI. Register is safe to call concurrently and rejects duplicate names.
func Register(handler *CommandHandler) error { return commands.Register(handler) }

func MaxArgs(n int) ArgLimit { return commands.MaxArgs(n) }

func FormArgs(forms map[string]int) ArgLimit { return commands.FormArgs(forms) }

func FormArgsAt(at int, forms map[string]int) ArgLimit { return commands.FormArgsAt(at, forms) }

func OpenArgs() ArgLimit { return commands.OpenArgs() }
