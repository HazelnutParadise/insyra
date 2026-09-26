package commands

import (
	"fmt"
	"math"
	"strings"
)

// unlimited is a form limit for a form whose arguments are a list the
// command checks item by item.
const unlimited = math.MaxInt

// ArgLimit says how many arguments a command takes, so a token the command
// would silently ignore is refused in one place instead of by a hand-written
// check in every command. Its zero value means "not declared", which
// TestEveryCommandDeclaresItsArguments rejects, so a new command cannot
// forget to say.
type ArgLimit struct {
	declared bool
	// open means the command checks every argument itself: a list of
	// values, or keyword options whose parser rejects anything it does not
	// know.
	open bool
	// alias means a trailing `as <var>` is accepted and not counted.
	alias bool
	max   int
	// forms holds the limit per form when an argument selects one
	// (`ttest single|two|paired`); the count includes the form word.
	forms map[string]int
	// formAt is the position of the argument that selects the form: 0 for
	// `ttest single ...`, 1 for `clean <var> nan`.
	formAt int
}

// MaxArgs declares that a command takes at most n arguments.
func MaxArgs(n int) ArgLimit { return ArgLimit{declared: true, max: n} }

// FormArgs declares a limit for each form a first argument selects. An
// unknown form is left for the command to report.
func FormArgs(forms map[string]int) ArgLimit { return FormArgsAt(0, forms) }

// FormArgsAt is FormArgs for a form chosen by the argument at position at,
// such as `clean <var> nan|outliers`.
func FormArgsAt(at int, forms map[string]int) ArgLimit {
	lower := make(map[string]int, len(forms))
	for form, n := range forms {
		lower[strings.ToLower(form)] = n
	}
	return ArgLimit{declared: true, forms: lower, formAt: at}
}

// OpenArgs declares that the command validates every argument itself.
func OpenArgs() ArgLimit { return ArgLimit{declared: true, open: true} }

// WithAlias allows a trailing `as <var>` on top of the limit.
func (l ArgLimit) WithAlias() ArgLimit {
	l.alias = true
	return l
}

// check refuses an argument beyond the declared limit, naming it.
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
