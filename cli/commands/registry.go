package commands

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"sync"

	"github.com/HazelnutParadise/insyra/cli/env"
	"github.com/spf13/cobra"
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
	Args               ArgLimit
	Flags              []CommandFlag
	Run                func(ctx *ExecContext, args []string) error
}

// CommandFlag describes a one-shot CLI flag.
type CommandFlag struct {
	Name       string
	Usage      string
	TakesValue bool
	Form       string
}

func (f CommandFlag) define(cmd *cobra.Command) {
	if f.TakesValue {
		cmd.Flags().String(f.Name, "", f.Usage)
		return
	}
	cmd.Flags().Bool(f.Name, false, f.Usage)
}

func (f CommandFlag) appendTo(cmd *cobra.Command, args, runArgs []string) []string {
	if f.Form != "" {
		matches := false
		if len(args) > 0 {
			for _, form := range strings.Split(f.Form, "|") {
				if strings.EqualFold(args[0], strings.TrimSpace(form)) {
					matches = true
					break
				}
			}
		}
		if !matches {
			return runArgs
		}
	}
	if f.TakesValue {
		value, err := cmd.Flags().GetString(f.Name)
		if err == nil && strings.TrimSpace(value) != "" {
			return append(runArgs, "--"+f.Name, value)
		}
		return runArgs
	}
	if set, err := cmd.Flags().GetBool(f.Name); err == nil && set {
		return append(runArgs, "--"+f.Name)
	}
	return runArgs
}

// Registry holds every registered command by name. Access it through
// Register, Dispatch, and BuildCobraCommands, which take registryMu; reading
// the map directly is not safe while another goroutine registers.
var Registry = map[string]*CommandHandler{}

var registryMu sync.RWMutex

func alsoStored(alias string, suffixes ...string) string {
	names := make([]string, len(suffixes))
	for i, suffix := range suffixes {
		names[i] = alias + "_" + suffix
	}
	return strings.Join(names, ", ")
}

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
	run, limit, name, usage := handler.Run, handler.Args, handler.Name, handler.Usage
	handler.Run = func(ctx *ExecContext, args []string) error {
		if err := limit.check(name, usage, args); err != nil {
			return err
		}
		return run(ctx, args)
	}
	registryMu.Lock()
	defer registryMu.Unlock()
	if _, exists := Registry[handler.Name]; exists {
		return fmt.Errorf("command already registered: %s", handler.Name)
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

func BuildCobraCommands(ctx *ExecContext) []*cobra.Command {
	registryMu.RLock()
	keys := make([]string, 0, len(Registry))
	handlers := make(map[string]*CommandHandler, len(Registry))
	for name, h := range Registry {
		keys = append(keys, name)
		handlers[name] = h
	}
	registryMu.RUnlock()
	sort.Strings(keys)

	commands := make([]*cobra.Command, 0, len(keys))
	for _, name := range keys {
		handler := handlers[name]
		localHandler := handler

		use := localHandler.Usage
		if use == "" {
			use = localHandler.Name
		}

		commands = append(commands, &cobra.Command{
			Use:                use,
			Aliases:            localHandler.Aliases,
			Short:              localHandler.Description,
			DisableFlagParsing: localHandler.DisableFlagParsing,
			RunE: func(cmd *cobra.Command, args []string) error {
				runArgs := args
				for _, flag := range localHandler.Flags {
					runArgs = flag.appendTo(cmd, args, runArgs)
				}
				err := Dispatch(ctx, localHandler.Name, runArgs)
				envName := ctx.EnvName
				if envName == "" {
					envName = "default"
				}
				if localHandler.Name != "history" {
					line := localHandler.Name
					if len(runArgs) > 0 {
						line += " " + strings.Join(runArgs, " ")
					}
					mgr := ctx.Env
					if mgr == nil {
						mgr = env.Default()
					}
					_ = mgr.AppendHistory(envName, SanitizeHistoryLine(line))
				}
				return err
			},
		})

		created := commands[len(commands)-1]
		for _, flag := range localHandler.Flags {
			flag.define(created)
		}
	}

	return commands
}
