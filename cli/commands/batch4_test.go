package commands

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/cli/env"
)

// CLI-1: a selector that matches nothing is an error and stores nothing.
func TestColRowMissingSelectorIsError(t *testing.T) {
	ctx := newTestExecContext(t)
	ctx.Vars["dt"] = insyra.NewDataTable(insyra.NewDataList(1, 2).SetName("price"))
	if err := runColCommand(ctx, []string{"dt", "nope", "as", "c"}); err == nil {
		t.Fatal("col with unknown name returned nil")
	}
	if _, ok := ctx.Vars["c"]; ok {
		t.Fatal("col stored a value for a missing column")
	}
	if err := runRowCommand(ctx, []string{"dt", "nope", "as", "r"}); err == nil {
		t.Fatal("row with unknown name returned nil")
	}
	if _, ok := ctx.Vars["r"]; ok {
		t.Fatal("row stored a value for a missing row")
	}
}

// CLI-1: transforms that fail inside the library must not store a nil.
func TestTransformsRejectNilResults(t *testing.T) {
	ctx := newTestExecContext(t)
	ctx.Vars["x"] = insyra.NewDataList(1.0, 2.0, 3.0)
	ctx.Vars["empty"] = insyra.NewDataList()
	cases := [][]string{
		{"movavg", "x", "0", "as", "r"},
		{"movavg", "x", "99", "as", "r"},
		{"expsmooth", "x", "5", "as", "r"},
		{"diff", "empty", "as", "r"},
	}
	for _, c := range cases {
		if err := Dispatch(ctx, c[0], c[1:]); err == nil {
			t.Errorf("%v returned nil error", c)
		}
		if v, ok := ctx.Vars["r"]; ok {
			t.Errorf("%v stored %v", c, v)
		}
	}
}

// CLI-5: `env open` inside a script switches environment without opening
// the REPL, and a script that runs itself stops at the depth limit.
func TestRunScriptDoesNotOpenREPLAndLimitsRecursion(t *testing.T) {
	base := t.TempDir()
	mgr := env.NewManager(base, "envs")
	ctx := newTestExecContext(t)
	ctx.Env = mgr
	ctx.EnvName = "default"
	opened := false
	ctx.OpenREPL = func(*ExecContext) error { opened = true; return nil }

	script := filepath.Join(base, "s.isr")
	if err := os.WriteFile(script, []byte("env create scr\nenv open scr\nnewdl 1 2 as x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := runScriptCommand(ctx, []string{script}); err != nil {
		t.Fatal(err)
	}
	if opened {
		t.Fatal("run opened the interactive REPL from a script")
	}
	if ctx.EnvName != "scr" {
		t.Fatalf("env open inside a script did not switch environment, got %q", ctx.EnvName)
	}

	loop := filepath.Join(base, "loop.isr")
	if err := os.WriteFile(loop, []byte("run "+loop+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- runScriptCommand(ctx, []string{loop}) }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("self-recursive script never returned")
	}
	if out := ctx.Output.(*bytes.Buffer).String(); !strings.Contains(out, "depth") {
		t.Fatalf("expected a recursion-depth error in output, got %q", out)
	}
}

// SEC-11 / CLI-4: history never records a database password.
func TestSanitizeHistoryLineMasksPasswords(t *testing.T) {
	cases := map[string]string{
		"db connect a mysql://alice:S3cretPW@localhost/db":              "S3cretPW",
		"db connect b postgres:host=x user=u password=hunter2 dbname=d": "hunter2",
		"db connect c alice:S3cretPW@tcp(localhost:3306)/db":            "S3cretPW",
	}
	for line, secret := range cases {
		got := SanitizeHistoryLine(line)
		if strings.Contains(got, secret) {
			t.Errorf("%q still contains the password: %q", line, got)
		}
		if !strings.HasPrefix(got, "db connect") {
			t.Errorf("%q lost its command prefix: %q", line, got)
		}
	}
	if got := SanitizeHistoryLine("newdl 1 2 3"); got != "newdl 1 2 3" {
		t.Fatalf("plain line changed: %q", got)
	}
}

// A quoted DSN or a quoted password value may contain spaces; masking stopped
// at the first space and left the rest of the password in history.
func TestSanitizeHistoryLineMasksAPasswordWithSpaces(t *testing.T) {
	cases := map[string]string{
		`db connect pg "host=h password=a b"`:                           `db connect pg "host=h password=***"`,
		`db connect pg "host=h password='a b' port=5"`:                  `db connect pg "host=h password=*** port=5"`,
		`db connect pg "host=h password=a b sslmode=disable"`:           `db connect pg "host=h password=*** sslmode=disable"`,
		`db connect ms "Server=s;Pwd=a b;Database=d"`:                   `db connect ms "Server=s;Pwd=***;Database=d"`,
		"db connect b postgres:host=x user=u password=hunter2 dbname=d": "db connect b postgres:host=x user=u password=*** dbname=d",
	}
	for line, want := range cases {
		if got := SanitizeHistoryLine(line); got != want {
			t.Errorf("SanitizeHistoryLine(%q)\n got %q\nwant %q", line, got, want)
		}
	}
}

// libpq and pgx accept spaces around '=', a single-quoted libpq value escapes a
// quote or a backslash with a backslash, and an ODBC braced value writes '}' as
// '}}'. Masking missed the first form and stopped early inside the other two.
func TestSanitizeHistoryLineMasksEscapedAndSpacedPasswords(t *testing.T) {
	cases := map[string]string{
		`db connect pg "host=h password = s3cr3tA port=5432"`:    `db connect pg "host=h password = *** port=5432"`,
		`db connect pg "host=h PASSWORD=  s3cr3tA"`:              `db connect pg "host=h PASSWORD=  ***"`,
		`db connect pg "host=h password='it\'s s3cr3tB' port=5"`: `db connect pg "host=h password=*** port=5"`,
		`db connect pg "password='a\\' port=5"`:                  `db connect pg "password=*** port=5"`,
		`db connect ms "Server=s;PWD={ab}}s3cr3tC};Database=d"`:  `db connect ms "Server=s;PWD=***;Database=d"`,
		`db connect ms "Server=s;Pwd={}}}}s3cr3tC}"`:             `db connect ms "Server=s;Pwd=***"`,
	}
	for line, want := range cases {
		if got := SanitizeHistoryLine(line); got != want {
			t.Errorf("SanitizeHistoryLine(%q)\n got %q\nwant %q", line, got, want)
		}
	}
	for _, line := range []string{
		`set password = s3cr3tA`,
		`echo "password='it\'s s3cr3tB'"`,
	} {
		if got := SanitizeHistoryLine(line); got != line {
			t.Errorf("a line that is not db connect changed: %q -> %q", line, got)
		}
	}
}

// CLI-4: the one-shot dispatcher path writes the sanitized line.
func TestDispatchHistoryIsSanitized(t *testing.T) {
	base := t.TempDir()
	mgr := env.NewManager(base, "envs")
	if err := mgr.EnsureDefaultEnvironment(); err != nil {
		t.Fatal(err)
	}
	ctx := newTestExecContext(t)
	ctx.Env = mgr
	ctx.EnvName = "default"
	cmds := BuildCobraCommands(ctx)
	for _, c := range cmds {
		if c.Name() == "db" {
			_ = c.RunE(c, []string{"connect", "bad", "mysql://alice:S3cretPW@127.0.0.1:1/db"})
		}
	}
	lines, err := mgr.ReadHistory("default")
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(lines, "\n")
	if strings.Contains(joined, "S3cretPW") {
		t.Fatalf("history contains the password: %q", joined)
	}
	if !strings.Contains(joined, "db connect bad") {
		t.Fatalf("history lost the command: %q", joined)
	}
	info, err := os.Stat(filepath.Join(base, "envs", "default", "history.txt"))
	if err != nil {
		t.Fatal(err)
	}
	// Windows has no POSIX permission bits: Go reports a writable file there
	// as 0666 whatever mode it was created with, so the check means nothing.
	if perm := info.Mode().Perm(); runtime.GOOS != "windows" && perm&0o077 != 0 {
		t.Fatalf("history.txt is group/world readable: %o", perm)
	}
}
