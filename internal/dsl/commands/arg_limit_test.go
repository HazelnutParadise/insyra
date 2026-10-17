package commands

import (
	"strings"
	"testing"
)

// Every command says how many arguments it takes, so none can silently
// ignore one (owner's ruling of 2026-09-26; AGENTS.md follow-up of
// 2026-09-11).
func TestEveryCommandDeclaresItsArguments(t *testing.T) {
	names, handlers := SnapshotRegistry()
	for _, name := range names {
		if !handlers[name].Args.declared {
			t.Errorf("command %q does not declare Args", name)
		}
	}
}

func TestArgLimitRefusesAnIgnoredArgument(t *testing.T) {
	cases := []struct {
		limit ArgLimit
		args  []string
		bad   string
	}{
		{MaxArgs(1), []string{"x", "junk"}, "junk"},
		{MaxArgs(1).WithAlias(), []string{"x", "junk", "as", "y"}, "junk"},
		{FormArgs(map[string]int{"single": 4}), []string{"single", "x", "0", "0.95", "junk"}, "junk"},
		{FormArgsAt(1, map[string]int{"nan": 2}), []string{"x", "nan", "3"}, "3"},
	}
	for _, tc := range cases {
		err := tc.limit.check("cmd", "cmd ...", tc.args)
		if err == nil || !strings.Contains(err.Error(), tc.bad) {
			t.Errorf("%v: got %v, want an error naming %q", tc.args, err, tc.bad)
		}
	}
	accepted := []struct {
		limit ArgLimit
		args  []string
	}{
		{MaxArgs(1), []string{"x"}},
		{MaxArgs(1).WithAlias(), []string{"x", "as", "y"}},
		{FormArgs(map[string]int{"single": 3}), []string{"single", "x", "0"}},
		{FormArgs(map[string]int{"single": 3}), []string{"unknownform", "a", "b", "c", "d"}},
		{OpenArgs(), []string{"a", "b", "c", "d", "e"}},
	}
	for _, tc := range accepted {
		if err := tc.limit.check("cmd", "cmd ...", tc.args); err != nil {
			t.Errorf("%v: refused with %v", tc.args, err)
		}
	}
}

func TestRegisterRejectsNegativeArgumentLimits(t *testing.T) {
	for _, tc := range []struct {
		name  string
		limit ArgLimit
	}{
		{"negative max", MaxArgs(-1)},
		{"negative position", FormArgsAt(-1, map[string]int{"form": 1})},
		{"negative form limit", FormArgs(map[string]int{"form": -1})},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := Register(&CommandHandler{Name: "invalid_arg_limit_" + strings.ReplaceAll(tc.name, " ", "_"), Args: tc.limit, Run: func(*ExecContext, []string) error { return nil }})
			if err == nil {
				t.Fatal("Register accepted invalid ArgLimit")
			}
		})
	}
}
