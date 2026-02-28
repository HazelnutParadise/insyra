package repl

import (
	"testing"

	"github.com/HazelnutParadise/insyra/cli/commands"
)

func TestPrompt(t *testing.T) {
	got := prompt("default")
	want := "insyra [default] > "
	if got != want {
		t.Fatalf("unexpected prompt: got %q want %q", got, want)
	}
}

func TestTokenize(t *testing.T) {
	input := `filter t1 "A > 10" as t2`
	tokens := tokenize(input)
	if len(tokens) != 5 {
		t.Fatalf("expected 5 tokens, got %d (%v)", len(tokens), tokens)
	}
	if tokens[0] != "filter" || tokens[1] != "t1" || tokens[2] != "A > 10" || tokens[3] != "as" || tokens[4] != "t2" {
		t.Fatalf("unexpected tokens: %v", tokens)
	}
}

func TestStartMissingEnvironment(t *testing.T) {
	ctx := &commands.ExecContext{EnvName: "__does_not_exist__", Vars: map[string]any{}}
	err := Start(ctx)
	if err == nil {
		t.Fatalf("expected error when environment does not exist")
	}
}
