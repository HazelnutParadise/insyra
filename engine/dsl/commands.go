package dsl

import "github.com/HazelnutParadise/insyra/internal/dsl/commands"

// ExecContext is what a command runs against: the session's variables (Vars),
// where it prints (Output), the environment's name and folder, and the
// environment's Manager (Env). Session.Context returns the session's own.
type ExecContext = commands.ExecContext

// CommandHandler describes a command: its Name and Aliases, the Usage,
// Description, Forms and Examples that `help` prints, the arguments it takes
// (Args), the flags its one-shot form takes in the insyra command (Flags), and
// Run, which receives the arguments after the command's name.
type CommandHandler = commands.CommandHandler

// CommandFlag is a flag a command's one-shot form takes in the insyra command,
// such as --force in `insyra env import backup.json --force`. A session hands a
// command its flags as ordinary arguments, so a command used only from a
// program has no need of one.
type CommandFlag = commands.CommandFlag

// ArgLimit says how many arguments a command takes. An argument past it is
// refused before Run is called, with an error naming it and the command's Usage.
type ArgLimit = commands.ArgLimit

// Register adds a command. It is then available in every session in the
// process, and in the insyra command's shell when that runs in the same
// process: there is one registry, and Register is safe to call from any
// goroutine. Register refuses a handler without a Name or a Run function, a
// name already taken, a built-in command's included, and a flag with no name or
// one declared twice. There is no way to remove a command.
//
// Run receives the session's ExecContext and the arguments after the command's
// name. It reads and writes variables through ctx.Vars and prints to
// ctx.Output; the session saves the variables after Run returns nil.
func Register(handler *CommandHandler) error { return commands.Register(handler) }

// MaxArgs declares that a command takes at most n arguments. Add
// .WithAlias() when it also takes a trailing `as <var>`.
func MaxArgs(n int) ArgLimit { return commands.MaxArgs(n) }

// FormArgs declares a limit for each form a first argument selects, counting
// the form word: `ttest single <var> <mu>` is 3. An unknown form is left for
// the command to report.
func FormArgs(forms map[string]int) ArgLimit { return commands.FormArgs(forms) }

// FormArgsAt is FormArgs for a form chosen by the argument at position at, such
// as `clean <var> nan|outliers`.
func FormArgsAt(at int, forms map[string]int) ArgLimit { return commands.FormArgsAt(at, forms) }

// OpenArgs declares that the command checks every argument itself.
func OpenArgs() ArgLimit { return commands.OpenArgs() }
