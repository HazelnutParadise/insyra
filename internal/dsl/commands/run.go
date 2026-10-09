package commands

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/HazelnutParadise/insyra/internal/dsl/style"
)

func init() {
	_ = Register(&CommandHandler{Name: "run", Args: MaxArgs(1), Usage: "run <script.isr>", Description: "Run DSL script file", Run: runScriptCommand})
}

// errScriptExited is what a nested run returns after an exit line, so the
// script that ran it stops too without reporting the exit again.
var errScriptExited = fmt.Errorf("script ended by %w", ErrExit)

// EnterScript counts the caller as one level of running script, as run does
// for each file, until the returned function is called. Session.ExecuteFile
// uses it so that exit inside the file, or inside a script the file runs,
// ends the whole file the way it ends `insyra run`.
func EnterScript(ctx *ExecContext) (leave func()) {
	ctx.scriptDepth++
	return func() { ctx.scriptDepth-- }
}

// maxScriptDepth bounds nested `run` calls so a script that runs itself
// (directly or through another script) stops instead of recursing forever.
const maxScriptDepth = 16

func runScriptCommand(ctx *ExecContext, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: run <script.isr>")
	}
	if ctx.scriptDepth >= maxScriptDepth {
		return fmt.Errorf("run: script nesting depth exceeds %d (is %s running itself?)", maxScriptDepth, args[0])
	}
	file, err := os.Open(args[0])
	if err != nil {
		return err
	}
	defer func() {
		_ = file.Close()
	}()

	// A script is non-interactive: `env open` inside it must only switch the
	// environment, never start the REPL and wait on stdin.
	ctx.scriptDepth++
	savedOpenREPL := ctx.OpenREPL
	ctx.OpenREPL = nil
	defer func() {
		ctx.scriptDepth--
		ctx.OpenREPL = savedOpenREPL
	}()

	scanner := bufio.NewScanner(file)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		tokens := splitScriptTokens(line)
		if len(tokens) == 0 {
			continue
		}
		if err := Dispatch(ctx, tokens[0], tokens[1:]); err != nil {
			if errors.Is(err, ErrExit) {
				// Only the script whose own line was exit reports it; a script
				// that ran it stops without a second message.
				if !errors.Is(err, errScriptExited) {
					_, _ = fmt.Fprintf(ctx.Output, "script ended by exit at line %d\n", lineNumber)
				}
				// exit ends every script in a chain of run lines, and the
				// outermost run succeeds, so the REPL or the shell goes on.
				if ctx.scriptDepth > 1 {
					return errScriptExited
				}
				return nil
			}
			_, _ = fmt.Fprintf(ctx.Output, "%s\n", style.ErrorText(fmt.Sprintf("line %d: %v", lineNumber, err)))
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	_, _ = fmt.Fprintln(ctx.Output, "script complete")
	return nil
}

func splitScriptTokens(line string) []string {
	result := []string{}
	var builder strings.Builder
	quote := rune(0)
	escaped := false
	flush := func() {
		if builder.Len() == 0 {
			return
		}
		result = append(result, builder.String())
		builder.Reset()
	}

	runes := []rune(line)
	for idx, ch := range runes {
		if escaped {
			builder.WriteRune(ch)
			escaped = false
			continue
		}
		// A backslash escapes only a quote or another backslash. Treating it
		// as a universal escape ate every separator of a Windows path, so
		// `load C:\Users\me\bars.csv` opened `C:Usersmebars.csv`.
		if ch == '\\' && idx+1 < len(runes) {
			switch runes[idx+1] {
			case '"', '\'', '\\':
				escaped = true
				continue
			}
		}
		if quote != 0 {
			if ch == quote {
				quote = 0
				continue
			}
			builder.WriteRune(ch)
			continue
		}
		if ch == '"' || ch == '\'' {
			quote = ch
			continue
		}
		if ch == ' ' || ch == '\t' {
			flush()
			continue
		}
		builder.WriteRune(ch)
	}
	flush()
	return result
}
