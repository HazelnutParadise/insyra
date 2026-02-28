package repl

import (
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
	tokens := strings.Fields(prefix)

	candidates := []string{}
	target := ""
	if len(tokens) <= 1 {
		target = strings.TrimSpace(prefix)
		for name := range commands.Registry {
			candidates = append(candidates, name)
		}
	} else {
		target = tokens[len(tokens)-1]
		for name := range c.ctx.Vars {
			candidates = append(candidates, name)
		}
	}

	sort.Strings(candidates)
	results := [][]rune{}
	for _, candidate := range candidates {
		if strings.HasPrefix(candidate, target) {
			suffix := strings.TrimPrefix(candidate, target)
			results = append(results, []rune(suffix))
		}
	}
	return results, len([]rune(target))
}
