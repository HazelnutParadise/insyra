package repl

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra/cli/commands"
)

// completer.go was the whole of cli/repl's uncovered half: tab completion is
// what a user touches on every line of the REPL, and none of it had a test.
// Nothing here needs a terminal — Do takes a line and a position and returns
// what is left to type.

// complete runs Do over a line, with the cursor at the end.
func complete(t *testing.T, ctx *commands.ExecContext, line string) ([]string, int) {
	t.Helper()
	runes := []rune(line)
	got, length := NewAutoCompleter(ctx).(*simpleCompleter).Do(runes, len(runes))
	out := make([]string, 0, len(got))
	for _, r := range got {
		out = append(out, string(r))
	}
	return out, length
}

func contextWithVars(names ...string) *commands.ExecContext {
	ctx := &commands.ExecContext{Vars: map[string]any{}}
	for _, n := range names {
		ctx.Vars[n] = 1
	}
	return ctx
}

// The contract: the returned strings are the part still to type, and the int is
// how much of the target is already on the line. Getting either wrong leaves
// the user with a doubled or truncated word.
func TestDo_ReturnsSuffixesAndTheTypedLength(t *testing.T) {
	got, length := complete(t, contextWithVars(), "mea")

	if length != 3 {
		t.Errorf("typed length: got %d, want 3", length)
	}
	for _, s := range got {
		if strings.HasPrefix(s, "mea") {
			t.Errorf("completion %q repeats what is already typed", s)
		}
	}
	if !slicesContain(got, "n") {
		t.Errorf("completing \"mea\" did not offer \"n\" for \"mean\": %v", got)
	}
}

func TestDo_NothingBeforeTheCursor(t *testing.T) {
	got, length := NewAutoCompleter(contextWithVars()).(*simpleCompleter).Do([]rune("mean"), -1)
	if got != nil || length != 0 {
		t.Errorf("a negative position: got (%v, %d), want (nil, 0)", got, length)
	}
}

func TestDo_FirstWordCompletesCommands(t *testing.T) {
	ctx := contextWithVars("x")

	// A line of only whitespace offers every command.
	got, length := complete(t, ctx, "  ")
	if length != 0 {
		t.Errorf("whitespace-only line: typed length %d, want 0", length)
	}
	if len(got) == 0 {
		t.Fatal("a whitespace-only line offered no commands")
	}

	got, length = complete(t, ctx, "mea")
	if length != 3 {
		t.Errorf("typed length: got %d, want 3", length)
	}
	if !slicesContain(got, "n") {
		t.Errorf("completing \"mea\" did not offer \"n\": %v", got)
	}
}

// Commands come back in sorted order, so the list does not reshuffle between
// presses of the tab key.
func TestCommandCompletions_Sorted(t *testing.T) {
	got := toStrings(commandCompletions(""))

	if len(got) < 2 {
		t.Fatalf("the registry offered %d commands", len(got))
	}
	if !sort.StringsAreSorted(got) {
		t.Errorf("commands are not sorted: %v", got[:min(5, len(got))])
	}
}

func TestCommandCompletions_UnknownPrefix(t *testing.T) {
	if got := commandCompletions("zzzznotacommand"); len(got) != 0 {
		t.Errorf("an unknown prefix offered %d completions", len(got))
	}
}

func TestDo_LaterWordsCompleteVariables(t *testing.T) {
	ctx := contextWithVars("alpha", "beta", "gamma")

	got, length := complete(t, ctx, "mean al")
	if length != 2 {
		t.Errorf("typed length: got %d, want 2", length)
	}
	if want := []string{"pha"}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	// A trailing space means nothing is typed yet, so every variable is offered.
	got, length = complete(t, ctx, "mean ")
	if length != 0 {
		t.Errorf("after a trailing space: typed length %d, want 0", length)
	}
	if want := []string{"alpha", "beta", "gamma"}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestVariableCompletions_NoContext(t *testing.T) {
	if got := variableCompletions(nil, ""); got != nil {
		t.Errorf("a nil context offered %d completions", len(got))
	}
	if got := variableCompletions(&commands.ExecContext{}, ""); len(got) != 0 {
		t.Errorf("a context with no variables offered %d completions", len(got))
	}
}

// The commands that take a path get file completions where the others get
// variable completions. This table is the whole rule.
func TestShouldCompletePath(t *testing.T) {
	tests := []struct {
		command    string
		tokenCount int
		target     string
		want       bool
	}{
		// Nothing typed yet after a path command: offer paths.
		{command: "load", tokenCount: 2, target: "", want: true},
		{command: "save", tokenCount: 2, target: "", want: true},
		{command: "read", tokenCount: 2, target: "", want: true},
		{command: "run", tokenCount: 2, target: "", want: true},
		{command: "convert", tokenCount: 2, target: "", want: true},
		{command: "mean", tokenCount: 2, target: "", want: false},
		// Partway through a word, the position in the line decides.
		{command: "load", tokenCount: 2, target: "da", want: true},
		{command: "load", tokenCount: 3, target: "da", want: false},
		{command: "read", tokenCount: 2, target: "da", want: true},
		{command: "run", tokenCount: 2, target: "da", want: true},
		{command: "save", tokenCount: 2, target: "da", want: false},
		{command: "save", tokenCount: 3, target: "da", want: true},
		{command: "convert", tokenCount: 2, target: "da", want: true},
		{command: "convert", tokenCount: 3, target: "da", want: true},
		{command: "convert", tokenCount: 4, target: "da", want: false},
		{command: "mean", tokenCount: 2, target: "da", want: false},
	}
	for _, tt := range tests {
		if got := shouldCompletePath(tt.command, tt.tokenCount, tt.target); got != tt.want {
			t.Errorf("shouldCompletePath(%q, %d, %q) = %v, want %v",
				tt.command, tt.tokenCount, tt.target, got, tt.want)
		}
	}
}

func TestFileCompletions(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"data.csv", "data.json", "other.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o600); err != nil {
			t.Fatalf("writing the fixture: %v", err)
		}
	}
	if err := os.Mkdir(filepath.Join(dir, "nested"), 0o755); err != nil {
		t.Fatalf("making the directory: %v", err)
	}
	t.Chdir(dir)

	t.Run("everything in the working directory", func(t *testing.T) {
		got := toStrings(fileCompletions(""))
		want := []string{"data.csv", "data.json", "nested" + string(os.PathSeparator), "other.txt"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("filtered by prefix, returned as suffixes", func(t *testing.T) {
		got := toStrings(fileCompletions("data."))
		if want := []string{"csv", "json"}; !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	// A directory is marked with a separator so the user can keep typing into it.
	t.Run("directories carry a separator", func(t *testing.T) {
		got := toStrings(fileCompletions("nes"))
		if want := []string{"ted" + string(os.PathSeparator)}; !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("inside a subdirectory", func(t *testing.T) {
		if err := os.WriteFile(filepath.Join(dir, "nested", "deep.csv"), []byte("x"), 0o600); err != nil {
			t.Fatalf("writing the fixture: %v", err)
		}
		target := "nested" + string(os.PathSeparator)
		got := toStrings(fileCompletions(target + "de"))
		if want := []string{"ep.csv"}; !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("a directory that is not there", func(t *testing.T) {
		if got := fileCompletions("nosuchdir/x"); got != nil {
			t.Errorf("an unreadable directory offered %d completions", len(got))
		}
	})
}

// The path commands reach fileCompletions through Do, which is the path a user
// actually takes.
func TestDo_PathCommandsCompleteFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "sales.csv"), []byte("x"), 0o600); err != nil {
		t.Fatalf("writing the fixture: %v", err)
	}
	t.Chdir(dir)

	ctx := contextWithVars("sales_variable")

	got, length := complete(t, ctx, "load sa")
	if length != 2 {
		t.Errorf("typed length: got %d, want 2", length)
	}
	if want := []string{"les.csv"}; !reflect.DeepEqual(got, want) {
		t.Errorf("load offered %v, want the file", got)
	}

	// The same position after a command that does not take a path offers the
	// variable instead.
	got, _ = complete(t, ctx, "mean sa")
	if want := []string{"les_variable"}; !reflect.DeepEqual(got, want) {
		t.Errorf("mean offered %v, want the variable", got)
	}
}

func TestToSuffixes(t *testing.T) {
	candidates := []string{"apple", "apricot", "banana"}

	if got := toStrings(toSuffixes(candidates, "ap")); !reflect.DeepEqual(got, []string{"ple", "ricot"}) {
		t.Errorf("toSuffixes with \"ap\": got %v", got)
	}
	if got := toStrings(toSuffixes(candidates, "")); !reflect.DeepEqual(got, candidates) {
		t.Errorf("toSuffixes with no target: got %v", got)
	}
	if got := toSuffixes(candidates, "zz"); len(got) != 0 {
		t.Errorf("toSuffixes with a prefix nothing matches: got %v", got)
	}
	// An exact match leaves nothing to type, which is still a completion.
	if got := toStrings(toSuffixes(candidates, "apple")); !reflect.DeepEqual(got, []string{""}) {
		t.Errorf("toSuffixes with an exact match: got %v", got)
	}
}

func toStrings(rs [][]rune) []string {
	out := make([]string, 0, len(rs))
	for _, r := range rs {
		out = append(out, string(r))
	}
	return out
}

func slicesContain(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}
