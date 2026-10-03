package commands

import (
	"fmt"
	"io"
	"os"
	"sort"
	"sync"

	insyra "github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/internal/dsl/env"
)

type ExecContext struct {
	Vars     map[string]any
	DBConns  map[string]*DBConn
	EnvName  string
	EnvPath  string
	Output   io.Writer
	InREPL   bool
	OpenREPL func(ctx *ExecContext) error
	// Env is the per-session environment Manager. Commands MUST go through
	// it (ctx.Env.SaveState etc.) so embedders that supply a per-workspace
	// Manager don't get their writes redirected to the default ~/.insyra.
	// Dispatch fills this with env.Default() if the caller left it nil.
	Env *env.Manager

	// scriptDepth counts nested `run` invocations (see maxScriptDepth).
	scriptDepth int

	// unsavedWarned maps each variable SaveEnvState has reported as not saved
	// to the Go type it held then, so a session reports it once.
	unsavedWarned map[string]string
}

type CommandHandler struct {
	Name        string
	Aliases     []string
	Usage       string
	Description string
	// Forms lists the major sub-shapes of a command (one entry per shape).
	// Each entry is rendered as-is under a "Forms:" header by `help <cmd>`.
	// Use for commands like `ttest single|two|paired` where the bare Usage
	// can't enumerate every form.
	Forms []string
	// Examples lists ready-to-run invocations rendered under "Examples:" by
	// `help <cmd>`. Each entry should be a complete `insyra ...` line that
	// the user can copy into a shell.
	Examples           []string
	DisableFlagParsing bool
	// Args declares how many arguments the command takes. Register wraps
	// Run so an argument beyond it is refused before the command runs,
	// instead of being silently ignored.
	Args ArgLimit
	// Flags lists the flags the one-shot form `insyra <command> ...` takes.
	// cli/commands.BuildCobraCommands registers them with the shell, and each
	// one that is set reaches Run as arguments, the way the REPL and scripts
	// pass them, so Run reads them in one place.
	Flags []CommandFlag
	Run   func(ctx *ExecContext, args []string) error
}

// CommandFlag is a flag a command's one-shot form takes, such as --force in
// `insyra env import backup.json --force`.
type CommandFlag struct {
	// Name is the flag without its leading dashes.
	Name string
	// Usage is the help text the shell prints for the flag.
	Usage string
	// TakesValue makes the flag take a value (--mode gpu), handed to Run as
	// the flag and its value when the value is not blank. Without it the flag
	// is a switch (--force), handed to Run when it is set.
	TakesValue bool
	// Form, when set, hands the flag to Run only when the command's first
	// argument is this word, in any letter case: --force is for `env import`
	// and is dropped from any other env form.
	Form string
}

// Registry holds every registered command by name. Access it through
// Register, Dispatch, LookupCommand and SnapshotRegistry, which take
// registryMu; reading the map directly is not safe while another goroutine
// registers.
var Registry = map[string]*CommandHandler{}

var registryMu sync.RWMutex

// LookupCommand returns the handler registered under name, taking the read
// lock. Reading Registry directly is not safe while another goroutine
// registers, which an embedder may do at any time.
func LookupCommand(name string) (*CommandHandler, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	h, ok := Registry[name]
	return h, ok
}

// SnapshotRegistry returns the registered names in sorted order and the
// handlers they point at, taken under the read lock.
func SnapshotRegistry() ([]string, map[string]*CommandHandler) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	names := make([]string, 0, len(Registry))
	handlers := make(map[string]*CommandHandler, len(Registry))
	for name, h := range Registry {
		names = append(names, name)
		handlers[name] = h
	}
	sort.Strings(names)
	return names, handlers
}

func Register(handler *CommandHandler) error {
	if handler == nil {
		return fmt.Errorf("handler is nil")
	}
	if handler.Name == "" {
		return fmt.Errorf("handler name is required")
	}
	if handler.Run == nil {
		return fmt.Errorf("handler run function is required")
	}
	seenFlags := make(map[string]bool, len(handler.Flags))
	for _, flag := range handler.Flags {
		if flag.Name == "" {
			return fmt.Errorf("command %s: a flag has no name", handler.Name)
		}
		if seenFlags[flag.Name] {
			return fmt.Errorf("command %s: flag --%s is declared twice", handler.Name, flag.Name)
		}
		seenFlags[flag.Name] = true
	}
	registryMu.Lock()
	defer registryMu.Unlock()
	if _, exists := Registry[handler.Name]; exists {
		return fmt.Errorf("command already registered: %s", handler.Name)
	}
	run, limit, name, usage := handler.Run, handler.Args, handler.Name, handler.Usage
	handler.Run = func(ctx *ExecContext, args []string) error {
		if err := limit.check(name, usage, args); err != nil {
			return err
		}
		clearStaleErrors(ctx)
		return run(ctx, args)
	}
	Registry[handler.Name] = handler
	return nil
}

func Dispatch(ctx *ExecContext, name string, args []string) error {
	registryMu.RLock()
	handler, ok := Registry[name]
	registryMu.RUnlock()
	if !ok {
		return fmt.Errorf("unknown command: %s", name)
	}
	if ctx == nil {
		ctx = &ExecContext{}
	}
	if ctx.Vars == nil {
		ctx.Vars = map[string]any{}
	}
	if ctx.Output == nil {
		ctx.Output = os.Stdout
	}
	if ctx.Env == nil {
		ctx.Env = env.Default()
	}
	return handler.Run(ctx, args)
}

// clearStaleErrors starts a command with no error recorded on any variable.
// A list or table keeps its first error until someone clears it, so a command
// that failed and reported its own message left the library's record behind,
// and the next command that checked for one reported it as its own: after
// `col t B` failed, a `sort t price` that had just succeeded failed with "no
// column is named B". Each command reports its own failures, so an error left
// from an earlier one has already been reported or was never this command's.
func clearStaleErrors(ctx *ExecContext) {
	if ctx == nil {
		return
	}
	for _, v := range ctx.Vars {
		switch typed := v.(type) {
		case *insyra.DataTable:
			typed.ClearErr()
		case *insyra.DataList:
			typed.ClearErr()
		}
	}
}
