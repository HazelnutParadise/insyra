package repl

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/HazelnutParadise/insyra/cli/commands"
	"github.com/ergochat/readline"
)

type simpleCompleter struct {
	ctx *commands.ExecContext
}

func NewAutoCompleter(ctx *commands.ExecContext) readline.AutoCompleter {
	return &simpleCompleter{ctx: ctx}
}

func (c *simpleCompleter) Do(line []rune, pos int) ([][]rune, int) {
	if pos <= 0 {
		return nil, 0
	}
	prefix := string(line[:pos])
	trimmedLeft := strings.TrimLeft(prefix, " \t")
	tokens := strings.Fields(prefix)

	if len(tokens) == 0 {
		return commandCompletions(""), 0
	}

	if len(tokens) == 1 && !strings.Contains(prefix, " ") {
		target := trimmedLeft
		return commandCompletions(target), len([]rune(target))
	}

	commandName := tokens[0]
	target := ""
	if len(prefix) > 0 && prefix[len(prefix)-1] != ' ' && prefix[len(prefix)-1] != '\t' {
		target = tokens[len(tokens)-1]
	}

	if shouldCompletePath(commandName, len(tokens), target) {
		return fileCompletions(target), len([]rune(target))
	}

	return variableCompletions(c.ctx, target), len([]rune(target))
}

func commandCompletions(target string) [][]rune {
	commandsList := make([]string, 0, len(commands.Registry))
	for name := range commands.Registry {
		commandsList = append(commandsList, name)
	}
	sort.Strings(commandsList)
	return toSuffixes(commandsList, target)
}

func variableCompletions(ctx *commands.ExecContext, target string) [][]rune {
	if ctx == nil {
		return nil
	}
	vars := make([]string, 0, len(ctx.Vars))
	for name := range ctx.Vars {
		vars = append(vars, name)
	}
	sort.Strings(vars)
	return toSuffixes(vars, target)
}

func fileCompletions(target string) [][]rune {
	dir := "."
	base := target
	if target != "" {
		dir = filepath.Dir(target)
		if dir == "" {
			dir = "."
		}
		base = filepath.Base(target)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	candidates := make([]string, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasPrefix(name, base) {
			continue
		}
		candidate := name
		if dir != "." {
			candidate = filepath.Join(dir, name)
		}
		if entry.IsDir() {
			candidate += string(os.PathSeparator)
		}
		candidates = append(candidates, candidate)
	}

	sort.Strings(candidates)
	return toSuffixes(candidates, target)
}

func toSuffixes(candidates []string, target string) [][]rune {
	results := make([][]rune, 0, len(candidates))
	for _, candidate := range candidates {
		if strings.HasPrefix(candidate, target) {
			results = append(results, []rune(strings.TrimPrefix(candidate, target)))
		}
	}
	return results
}

func shouldCompletePath(commandName string, tokenCount int, target string) bool {
	if target == "" {
		if commandName == "load" || commandName == "save" || commandName == "read" || commandName == "run" || commandName == "convert" {
			return true
		}
	}

	switch commandName {
	case "load", "read", "run":
		return tokenCount <= 2
	case "save":
		return tokenCount == 3
	case "convert":
		return tokenCount == 2 || tokenCount == 3
	default:
		return false
	}
}
