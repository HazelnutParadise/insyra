package insyra

import (
	"bytes"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// withColoredOutput turns coloured output on, captures the standard log in a
// buffer and restores both when the test ends.
func withColoredOutput(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	previousWriter, previousFlags := log.Writer(), log.Flags()
	previousColored := Config.GetUseColoredOutput()
	previousLogger := Config.GetLogger()
	previousLevel := Config.GetLogLevel()
	Config.SetLogLevel(LogLevelInfo)
	log.SetOutput(&buf)
	log.SetFlags(0)
	Config.SetUseColoredOutput(true)
	Config.SetLogger(nil)
	t.Setenv("NO_COLOR", "")
	t.Setenv("TERM", "xterm-256color")
	t.Cleanup(func() {
		log.SetOutput(previousWriter)
		log.SetFlags(previousFlags)
		Config.SetUseColoredOutput(previousColored)
		Config.SetLogger(previousLogger)
		Config.SetLogLevel(previousLevel)
	})
	return &buf
}

func TestErrorLinesAreNotColouredOutsideATerminal(t *testing.T) {
	buf := withColoredOutput(t)

	LogError("pkg", "Fn", "broken")
	LogFatal("pkg", "Fn", "worse")

	if strings.Contains(buf.String(), "\x1b") {
		t.Fatalf("a log line written to a buffer carries colour codes: %q", buf.String())
	}
	want := "[insyra - Error] pkg.Fn: Broken\n<{[insyra - FATAL!]}> pkg.Fn: Worse\n"
	if buf.String() != want {
		t.Fatalf("got %q, want %q", buf.String(), want)
	}
}

func TestShowAndSummaryAreNotColouredOutsideATerminal(t *testing.T) {
	withColoredOutput(t)
	dl := NewDataList(1, 2, 3)
	dt := NewDataTable(NewDataList(1, 2), NewDataList("a", "b"))

	for name, write := range map[string]func(*bytes.Buffer){
		"DataList.ShowTo":       func(b *bytes.Buffer) { dl.ShowTo(b) },
		"DataList.SummaryTo":    func(b *bytes.Buffer) { dl.SummaryTo(b) },
		"DataTable.ShowTo":      func(b *bytes.Buffer) { dt.ShowTo(b) },
		"DataTable.ShowTypesTo": func(b *bytes.Buffer) { dt.ShowTypesTo(b) },
		"DataTable.SummaryTo":   func(b *bytes.Buffer) { dt.SummaryTo(b) },
	} {
		var b bytes.Buffer
		write(&b)
		if b.Len() == 0 {
			t.Errorf("%s wrote nothing", name)
		}
		if strings.Contains(b.String(), "\x1b") {
			t.Errorf("%s to a buffer carries colour codes: %q", name, b.String())
		}
	}
}

// pretendTerminal makes every *os.File count as a terminal for the test, so
// the coloured path can be checked without a real terminal.
func pretendTerminal(t *testing.T) *os.File {
	t.Helper()
	previous := isTerminal
	isTerminal = func(*os.File) bool { return true }
	t.Cleanup(func() { isTerminal = previous })
	f, err := os.Create(filepath.Join(t.TempDir(), "tty"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = f.Close() })
	return f
}

func TestErrorLinesAreColouredOnATerminal(t *testing.T) {
	withColoredOutput(t)
	f := pretendTerminal(t)
	log.SetOutput(f)

	LogError("pkg", "Fn", "broken")

	got, err := os.ReadFile(f.Name())
	if err != nil {
		t.Fatal(err)
	}
	// The reset comes before the newline, so the next line starts uncoloured.
	if want := "\x1b[31m[insyra - Error] pkg.Fn: Broken\x1b[0m\n"; string(got) != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestShowIsColouredOnATerminal(t *testing.T) {
	withColoredOutput(t)
	f := pretendTerminal(t)

	NewDataList(1, 2, 3).ShowTo(f)

	got, err := os.ReadFile(f.Name())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "\x1b[") {
		t.Fatalf("ShowTo on a terminal is not coloured: %q", got)
	}
}

func TestColourSwitchesStillTurnColourOff(t *testing.T) {
	withColoredOutput(t)
	f := pretendTerminal(t)

	Config.SetUseColoredOutput(false)
	if got := colorText(f, "31", "x"); got != "x" {
		t.Errorf("SetUseColoredOutput(false): got %q", got)
	}
	Config.SetUseColoredOutput(true)
	t.Setenv("NO_COLOR", "1")
	if got := colorText(f, "31", "x"); got != "x" {
		t.Errorf("NO_COLOR=1: got %q", got)
	}
}
