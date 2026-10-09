// Package commands is the registry of the insyra command language, shared by
// the insyra command, its REPL, .isr scripts and engine/dsl, and the shell
// commands BuildCobraCommands makes from it. A program that adds its own
// command does so through engine/dsl.
package commands

import "github.com/HazelnutParadise/insyra/internal/dsl/commands"

// ExecContext is the state one command runs against: the session's variables,
// database connections, environment and output.
//
// Deprecated: use ExecContext from engine/dsl instead, the same type.
// Removed in the release after the one that deprecated it.
type ExecContext = commands.ExecContext

// CommandHandler is a registered command: its name, help text, the arguments
// and flags it takes, and its Run function.
//
// Deprecated: use CommandHandler from engine/dsl instead, the same type.
// Removed in the release after the one that deprecated it.
type CommandHandler = commands.CommandHandler

// CommandFlag is a flag a command's one-shot form takes, such as --force in
// `insyra env import backup.json --force`.
//
// Deprecated: use CommandFlag from engine/dsl instead, the same type.
// Removed in the release after the one that deprecated it.
type CommandFlag = commands.CommandFlag

// ArgLimit says how many arguments a command takes, so an argument the command
// would silently ignore is refused before it runs.
//
// Deprecated: use ArgLimit from engine/dsl instead, the same type.
// Removed in the release after the one that deprecated it.
type ArgLimit = commands.ArgLimit

// DBConn represents a named database connection registered in an ExecContext.
type DBConn = commands.DBConn

// Registry holds every registered command by name. It is the map the command
// language reads, so a command registered through Register is in it. Access it
// through Register, Dispatch, LookupCommand and SnapshotRegistry, which take
// the registry's lock; reading the map directly is not safe while another
// goroutine registers, and assigning another map to this variable changes
// nothing the command language reads.
var Registry = commands.Registry

// Register adds a command to the registry. It refuses a handler without a name
// or a Run function, a name already registered, and a flag with no name or one
// declared twice.
//
// Deprecated: use Register from engine/dsl instead, the same function.
// Removed in the release after the one that deprecated it.
func Register(handler *CommandHandler) error { return commands.Register(handler) }

// Dispatch runs the command registered under name with args against ctx. A nil
// ctx, or one without variables, output or an environment manager, gets an
// empty variable map, standard output and Default() from cli/env.
func Dispatch(ctx *ExecContext, name string, args []string) error {
	return commands.Dispatch(ctx, name, args)
}

// LookupCommand returns the handler registered under name or listing it among
// its aliases.
func LookupCommand(name string) (*CommandHandler, bool) { return commands.LookupCommand(name) }

// SnapshotRegistry returns the registered names in sorted order and the
// handlers they point at.
func SnapshotRegistry() ([]string, map[string]*CommandHandler) { return commands.SnapshotRegistry() }

// MaxArgs declares that a command takes at most n arguments.
//
// Deprecated: use MaxArgs from engine/dsl instead, the same function.
// Removed in the release after the one that deprecated it.
func MaxArgs(n int) ArgLimit { return commands.MaxArgs(n) }

// FormArgs declares a limit for each form a first argument selects. An unknown
// form is left for the command to report.
//
// Deprecated: use FormArgs from engine/dsl instead, the same function.
// Removed in the release after the one that deprecated it.
func FormArgs(forms map[string]int) ArgLimit { return commands.FormArgs(forms) }

// FormArgsAt is FormArgs for a form chosen by the argument at position at, such
// as `clean <var> nan|outliers`.
//
// Deprecated: use FormArgsAt from engine/dsl instead, the same function.
// Removed in the release after the one that deprecated it.
func FormArgsAt(at int, forms map[string]int) ArgLimit { return commands.FormArgsAt(at, forms) }

// OpenArgs declares that the command validates every argument itself.
//
// Deprecated: use OpenArgs from engine/dsl instead, the same function.
// Removed in the release after the one that deprecated it.
func OpenArgs() ArgLimit { return commands.OpenArgs() }

// SanitizeHistoryLine returns line with any database password masked, so a `db
// connect <name> <dsn>` never lands in history.txt or an exported environment
// in clear text. Other lines are returned unchanged.
func SanitizeHistoryLine(line string) string { return commands.SanitizeHistoryLine(line) }

// CloseAllDBConns closes every connection registered on ctx and clears the map.
func CloseAllDBConns(ctx *ExecContext) { commands.CloseAllDBConns(ctx) }

// SaveEnvState saves ctx.Vars to the environment ctx.EnvName. A variable the
// environment cannot store is left out and reported on ctx.Output, once per
// variable and type for the life of ctx, and does not make the save fail. The
// error is for a state that could not be written.
func SaveEnvState(ctx *ExecContext) error { return commands.SaveEnvState(ctx) }
