package commands

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	insyra "github.com/HazelnutParadise/insyra"
)

func newTestExecContext(t *testing.T) *ExecContext {
	t.Helper()
	return &ExecContext{
		Vars:   map[string]any{},
		Output: &bytes.Buffer{},
	}
}

func mustConnectSQLite(t *testing.T, ctx *ExecContext, name, dsn string) {
	t.Helper()
	if err := runDBCommand(ctx, []string{"connect", name, dsn}); err != nil {
		t.Fatalf("db connect failed: %v", err)
	}
	t.Cleanup(func() {
		_ = runDBCommand(ctx, []string{"disconnect", name})
	})
}

func TestDBConnect_DisconnectSQLite(t *testing.T) {
	ctx := newTestExecContext(t)
	mustConnectSQLite(t, ctx, "mem", "sqlite::memory:")

	if _, err := getDBConn(ctx, "mem"); err != nil {
		t.Fatalf("expected connection 'mem' to exist: %v", err)
	}

	out := &bytes.Buffer{}
	ctx.Output = out
	if err := runDBCommand(ctx, []string{"list"}); err != nil {
		t.Fatalf("db list failed: %v", err)
	}
	if !strings.Contains(out.String(), "mem\tsqlite") {
		t.Fatalf("expected list to show 'mem\\tsqlite', got %q", out.String())
	}
}

func TestDBConnect_RejectsDuplicate(t *testing.T) {
	ctx := newTestExecContext(t)
	mustConnectSQLite(t, ctx, "mem", "sqlite::memory:")
	err := runDBCommand(ctx, []string{"connect", "mem", "sqlite::memory:"})
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("expected duplicate-connect error, got %v", err)
	}
}

func TestDBConnect_RejectsUnknownDialect(t *testing.T) {
	ctx := newTestExecContext(t)
	err := runDBCommand(ctx, []string{"connect", "x", "oracle:foo"})
	if err == nil || !strings.Contains(err.Error(), "unsupported dialect") {
		t.Fatalf("expected unsupported-dialect error, got %v", err)
	}
}

func TestLoadSQL_TableRoundTrip(t *testing.T) {
	dir := t.TempDir()
	dsn := "sqlite:" + filepath.ToSlash(filepath.Join(dir, "round.db"))

	ctx := newTestExecContext(t)
	mustConnectSQLite(t, ctx, "rt", dsn)

	conn, _ := getDBConn(ctx, "rt")
	if err := conn.DB.Exec("CREATE TABLE t (a INTEGER, b TEXT);").Error; err != nil {
		t.Fatalf("create table failed: %v", err)
	}
	if err := conn.DB.Exec("INSERT INTO t (a, b) VALUES (1, 'one'), (2, 'two');").Error; err != nil {
		t.Fatalf("insert failed: %v", err)
	}

	if err := runLoadCommand(ctx, []string{"sql", "rt", "t", "as", "$dt"}); err != nil {
		t.Fatalf("load sql failed: %v", err)
	}
	dt, err := getDataTableVar(ctx, "$dt")
	if err != nil {
		t.Fatalf("$dt missing: %v", err)
	}
	if r, c := dt.Size(); r != 2 || c != 2 {
		t.Fatalf("expected 2x2 result, got %dx%d", r, c)
	}
}

func TestLoadSQL_QueryWithParams(t *testing.T) {
	ctx := newTestExecContext(t)
	mustConnectSQLite(t, ctx, "mem", "sqlite::memory:")

	conn, _ := getDBConn(ctx, "mem")
	if err := conn.DB.Exec("CREATE TABLE nums (n INTEGER);").Error; err != nil {
		t.Fatalf("create failed: %v", err)
	}
	for _, v := range []int{1, 2, 3, 4, 5} {
		if err := conn.DB.Exec("INSERT INTO nums (n) VALUES (?)", v).Error; err != nil {
			t.Fatalf("insert %d: %v", v, err)
		}
	}

	err := runLoadCommand(ctx, []string{
		"sql", "mem", "query",
		"SELECT n FROM nums WHERE n > ? AND n < ? ORDER BY n",
		"params", "1", "5",
		"as", "$dt",
	})
	if err != nil {
		t.Fatalf("load sql query failed: %v", err)
	}
	dt, err := getDataTableVar(ctx, "$dt")
	if err != nil {
		t.Fatalf("$dt missing: %v", err)
	}
	if r, _ := dt.Size(); r != 3 {
		t.Fatalf("expected 3 rows (n=2,3,4), got %d", r)
	}
}

func TestSaveSQL_WritesAndReplaces(t *testing.T) {
	dir := t.TempDir()
	dsn := "sqlite:" + filepath.ToSlash(filepath.Join(dir, "ws.db"))

	ctx := newTestExecContext(t)
	mustConnectSQLite(t, ctx, "ws", dsn)

	dt := insyra.NewDataTable(
		insyra.NewDataList(10, 20, 30).SetName("v"),
	)
	ctx.Vars["$dt"] = dt

	if err := runSaveCommand(ctx, []string{"$dt", "sql", "ws", "out"}); err != nil {
		t.Fatalf("save sql failed: %v", err)
	}

	// Re-saving should fail in default (fail) mode.
	if err := runSaveCommand(ctx, []string{"$dt", "sql", "ws", "out"}); err == nil {
		t.Fatalf("expected failure when target table already exists")
	}

	// Replace mode should overwrite.
	if err := runSaveCommand(ctx, []string{"$dt", "sql", "ws", "out", "if-exists", "replace"}); err != nil {
		t.Fatalf("save sql replace failed: %v", err)
	}

	// Verify count via load.
	if err := runLoadCommand(ctx, []string{"sql", "ws", "out", "as", "$check"}); err != nil {
		t.Fatalf("load back failed: %v", err)
	}
	got, err := getDataTableVar(ctx, "$check")
	if err != nil {
		t.Fatalf("missing $check: %v", err)
	}
	if r, _ := got.Size(); r != 3 {
		t.Fatalf("expected 3 rows after round-trip, got %d", r)
	}
}

func TestSaveSQL_AppendBatchSize(t *testing.T) {
	ctx := newTestExecContext(t)
	mustConnectSQLite(t, ctx, "mem", "sqlite::memory:")

	values := make([]any, 1500)
	for i := range values {
		values[i] = i
	}
	dt := insyra.NewDataTable(insyra.NewDataList(values...).SetName("n"))
	ctx.Vars["$dt"] = dt

	if err := runSaveCommand(ctx, []string{"$dt", "sql", "mem", "big", "batch", "400"}); err != nil {
		t.Fatalf("save sql with batch failed: %v", err)
	}

	conn, _ := getDBConn(ctx, "mem")
	var cnt int64
	if err := conn.DB.Raw("SELECT COUNT(*) FROM big").Scan(&cnt).Error; err != nil {
		t.Fatalf("count failed: %v", err)
	}
	if cnt != 1500 {
		t.Fatalf("expected 1500 rows, got %d", cnt)
	}
}

func TestDBTables_ListsSQLiteTables(t *testing.T) {
	ctx := newTestExecContext(t)
	mustConnectSQLite(t, ctx, "mem", "sqlite::memory:")

	conn, _ := getDBConn(ctx, "mem")
	if err := conn.DB.Exec("CREATE TABLE alpha (n INTEGER);").Error; err != nil {
		t.Fatalf("create alpha: %v", err)
	}
	if err := conn.DB.Exec("CREATE TABLE beta (n INTEGER);").Error; err != nil {
		t.Fatalf("create beta: %v", err)
	}

	out := &bytes.Buffer{}
	ctx.Output = out
	if err := runDBCommand(ctx, []string{"tables", "mem"}); err != nil {
		t.Fatalf("db tables failed: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "alpha") || !strings.Contains(got, "beta") {
		t.Fatalf("expected alpha and beta in output, got %q", got)
	}
}

func TestDBTables_NoTables(t *testing.T) {
	ctx := newTestExecContext(t)
	mustConnectSQLite(t, ctx, "mem", "sqlite::memory:")

	out := &bytes.Buffer{}
	ctx.Output = out
	if err := runDBCommand(ctx, []string{"tables", "mem"}); err != nil {
		t.Fatalf("db tables failed: %v", err)
	}
	if !strings.Contains(out.String(), "(no tables)") {
		t.Fatalf("expected '(no tables)', got %q", out.String())
	}
}

func TestMaskDSNPassword(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"sqlite:./foo.db", "sqlite:./foo.db"},
		{"postgres://user:secret@host:5432/db", "postgres://user:***@host:5432/db"},
		{"mysql://root:hunter2@host:3306/db?p=v", "mysql://root:***@host:3306/db?p=v"},
		{"mysql:root:hunter2@tcp(host:3306)/db", "mysql:root:***@tcp(host:3306)/db"},
	}
	for _, c := range cases {
		got := maskDSNPassword(c.in)
		if got != c.want {
			t.Errorf("maskDSNPassword(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
