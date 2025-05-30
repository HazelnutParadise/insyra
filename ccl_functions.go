package insyra

import (
	"fmt"
	"strings"
)

type Func = func(args ...any) (any, error)

var defaultFunctions = map[string]Func{}

func RegisterFunction(name string, fn Func) {
	defaultFunctions[strings.ToUpper(name)] = fn
}

func callFunction(name string, args []any) (any, error) {
	fn, ok := defaultFunctions[strings.ToUpper(name)]
	if !ok {
		return nil, fmt.Errorf("undefined function: %s", name)
	}
	return fn(args...)
}
