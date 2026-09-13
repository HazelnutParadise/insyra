package style

import (
	"strings"
	"testing"

	insyra "github.com/HazelnutParadise/insyra"
)

// This package had no test. It decides whether the CLI writes ANSI escapes,
// which is the difference between a readable message and a line of gibberish
// in a log file or a pipe.

// withColoredOutput sets the global switch for one test and puts it back.
func withColoredOutput(t *testing.T, colored bool) {
	t.Helper()
	prev := insyra.Config.GetDoesUseColoredOutput()
	insyra.Config.SetUseColoredOutput(colored)
	t.Cleanup(func() { insyra.Config.SetUseColoredOutput(prev) })
}

func TestErrorAndWarningText_Uncolored(t *testing.T) {
	withColoredOutput(t, false)

	if got, want := ErrorText("boom"), "error: boom"; got != want {
		t.Errorf("ErrorText: got %q, want %q", got, want)
	}
	if got, want := WarningText("careful"), "warn: careful"; got != want {
		t.Errorf("WarningText: got %q, want %q", got, want)
	}
}

func TestErrorAndWarningText_Colored(t *testing.T) {
	withColoredOutput(t, true)

	got := ErrorText("boom")
	if !strings.Contains(got, "error: boom") {
		t.Errorf("ErrorText lost its message: %q", got)
	}
	if !strings.HasPrefix(got, "\x1b[31m") || !strings.HasSuffix(got, "\x1b[0m") {
		t.Errorf("ErrorText = %q, want it wrapped in red and a reset", got)
	}

	got = WarningText("careful")
	if !strings.Contains(got, "warn: careful") {
		t.Errorf("WarningText lost its message: %q", got)
	}
	if !strings.HasPrefix(got, "\x1b[33m") || !strings.HasSuffix(got, "\x1b[0m") {
		t.Errorf("WarningText = %q, want it wrapped in yellow and a reset", got)
	}
	if strings.HasPrefix(got, "\x1b[31m") {
		t.Error("WarningText used the error colour")
	}
}

// An empty message still gets its prefix, and colouring still closes what it
// opens — a half-emitted escape would swallow the rest of the terminal line.
func TestText_EmptyMessage(t *testing.T) {
	withColoredOutput(t, false)
	if got, want := ErrorText(""), "error: "; got != want {
		t.Errorf("ErrorText(\"\"): got %q, want %q", got, want)
	}

	withColoredOutput(t, true)
	if got, want := ErrorText(""), "\x1b[31merror: \x1b[0m"; got != want {
		t.Errorf("ErrorText(\"\") coloured: got %q, want %q", got, want)
	}
}
