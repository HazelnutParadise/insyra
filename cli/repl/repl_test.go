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

// History keeps every non-empty line entered, as readline's automatic saving
// did: a comment, exit and a line of only empty quotes included. Saving by
// hand, after those checks, dropped them.
func TestHistoryEntryKeepsEveryNonEmptyLine(t *testing.T) {
	for _, line := range []string{"# a note", "exit", "quit", `""`, "  newdl 1 2 3  "} {
		if _, ok := historyEntry(line); !ok {
			t.Errorf("historyEntry(%q) was not saved", line)
		}
	}
	for _, line := range []string{"", "   ", "\t"} {
		if entry, ok := historyEntry(line); ok {
			t.Errorf("historyEntry(%q) saved %q for an empty line", line, entry)
		}
	}
	if entry, _ := historyEntry(`db connect pg "host=h password=a b"`); entry != `db connect pg "host=h password=***"` {
		t.Errorf("a db connect line was saved as %q", entry)
	}
}
