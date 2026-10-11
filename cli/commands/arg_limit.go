package commands

import (
	"fmt"
	"strings"
)

// ArgLimit declares how many arguments a command accepts. Its zero value
// leaves validation to the command handler.
type ArgLimit struct {
	declared bool
	open     bool
	alias    bool
	max      int
	forms    map[string]int
	formAt   int
}

func MaxArgs(n int) ArgLimit { return ArgLimit{declared: true, max: n} }

func FormArgs(forms map[string]int) ArgLimit { return FormArgsAt(0, forms) }

func FormArgsAt(at int, forms map[string]int) ArgLimit {
	lower := make(map[string]int, len(forms))
	for form, n := range forms {
		lower[strings.ToLower(form)] = n
	}
	return ArgLimit{declared: true, forms: lower, formAt: at}
}

func OpenArgs() ArgLimit { return ArgLimit{declared: true, open: true} }

func (l ArgLimit) WithAlias() ArgLimit { l.alias = true; return l }

func (l ArgLimit) check(name, usage string, args []string) error {
	if !l.declared || l.open {
		return nil
	}
	core := args
	if l.alias {
		core, _ = parseAlias(args)
	}
	limit := l.max
	if l.forms != nil {
		if len(core) <= l.formAt {
			return nil
		}
		n, ok := l.forms[strings.ToLower(core[l.formAt])]
		if !ok {
			return nil
		}
		limit = n
	}
	if len(core) <= limit {
		return nil
	}
	return fmt.Errorf("%s: unexpected argument %q; usage: %s", name, core[limit], usage)
}

func (l ArgLimit) validate() error {
	if l.max < 0 || l.formAt < 0 {
		return fmt.Errorf("argument limits and form positions must be non-negative")
	}
	for form, n := range l.forms {
		if n < 0 {
			return fmt.Errorf("argument limit for form %q must be non-negative", form)
		}
	}
	return nil
}
