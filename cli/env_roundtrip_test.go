package cli

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	insyra "github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/cli/env"
	"github.com/HazelnutParadise/insyra/stats"
)

// One-shot commands restore their variables from the environment before they
// run and save them back afterwards, so every value in a `insyra <command>`
// session crosses state.json. The tests below drive whole invocations through
// NewRootCommand and read the environment back, so what is asserted is what a
// user gets on the second command, not what a single process happened to hold.

// runOneShot runs one `insyra ...` invocation the way the binary does: a new
// root command, so variables go through state.json between invocations.
func runOneShot(t *testing.T, args ...string) {
	t.Helper()
	cmd := NewRootCommand()
	cmd.SetArgs(args)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("insyra %s: %v", strings.Join(args, " "), err)
	}
}

// writeCSV puts content in a file of its own and hands back its absolute path.
func writeCSV(t *testing.T, filename, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), filename)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	return path
}

// restore reads the default environment's variables back, which is the same
// restore the next invocation would do.
func restore(t *testing.T) map[string]any {
	t.Helper()
	vars, err := env.Default().RestoreVariables("default")
	if err != nil {
		t.Fatalf("restore default: %v", err)
	}
	return vars
}

func tableVar(t *testing.T, vars map[string]any, name string) *insyra.DataTable {
	t.Helper()
	dt, ok := vars[name].(*insyra.DataTable)
	if !ok {
		t.Fatalf("%s is %T, want *insyra.DataTable", name, vars[name])
	}
	return dt
}

func columnNames(dt *insyra.DataTable) []string {
	names := make([]string, 0, dt.NumCols())
	for i := 0; i < dt.NumCols(); i++ {
		names = append(names, dt.GetColByNumber(i).GetName())
	}
	return names
}

func colIndex(t *testing.T, dt *insyra.DataTable, name string) int {
	t.Helper()
	for i := 0; i < dt.NumCols(); i++ {
		if dt.GetColByNumber(i).GetName() == name {
			return i
		}
	}
	t.Fatalf("column %q not found (columns: %s)", name, strings.Join(columnNames(dt), ", "))
	return -1
}

func cellAt(t *testing.T, dt *insyra.DataTable, column string, row int) any {
	t.Helper()
	return dt.GetElementByNumberIndex(row, colIndex(t, dt, column))
}

func requireFloat64(t *testing.T, dt *insyra.DataTable, column string, row int, want float64) {
	t.Helper()
	got := cellAt(t, dt, column, row)
	f, ok := got.(float64)
	if !ok {
		t.Fatalf("%s[%d] is %T (%v), want float64", column, row, got, got)
	}
	if math.Float64bits(f) != math.Float64bits(want) {
		t.Errorf("%s[%d] = %v, want %v", column, row, f, want)
	}
}

// requireInt64 checks the type before the value. A whole-integer column that
// comes back as float64 is a different column to a command that tests the cell
// type, and int is not int64 however wide it happens to be on the host.
func requireInt64(t *testing.T, dt *insyra.DataTable, column string, row int, want int64) {
	t.Helper()
	got := cellAt(t, dt, column, row)
	i, ok := got.(int64)
	if !ok {
		t.Fatalf("%s[%d] is %T (%v), want int64", column, row, got, got)
	}
	if i != want {
		t.Errorf("%s[%d] = %v, want %v", column, row, i, want)
	}
}

func requireString(t *testing.T, dt *insyra.DataTable, column string, row int, want string) {
	t.Helper()
	got := cellAt(t, dt, column, row)
	s, ok := got.(string)
	if !ok {
		t.Fatalf("%s[%d] is %T (%v), want string", column, row, got, got)
	}
	if s != want {
		t.Errorf("%s[%d] = %q, want %q", column, row, s, want)
	}
}

func requireTime(t *testing.T, dt *insyra.DataTable, column string, row int, want time.Time) {
	t.Helper()
	got := cellAt(t, dt, column, row)
	v, ok := got.(time.Time)
	if !ok {
		t.Fatalf("%s[%d] is %T (%v), want time.Time", column, row, got, got)
	}
	if !v.Equal(want) {
		t.Errorf("%s[%d] = %v, want %v", column, row, v, want)
	}
}

// A table saved and restored keeps the column order the user loaded and the
// cell types they loaded: a whole-integer column comes back as int64, and a
// table with parsed dates keeps them as time.Time rather than going back to
// text.
func TestOneShotKeepsColumnOrderAndCellTypes(t *testing.T) {
	env.SetBasePath(t.TempDir())
	t.Cleanup(func() { env.SetBasePath("") })

	// This table intentionally has no NaN: the old code path only reordered
	// columns to alphabetical on tables without NaN, so a clean table is what
	// exposes that bug.
	csvPath := writeCSV(t, "mixed.csv", "zeta,id,score,alpha,when\n3,7,0.5,x,2024-01-02\n1.5,8,2.25,y,2024-03-04\n")

	runOneShot(t, "load", csvPath, "as", "t")
	runOneShot(t, "parsedates", "t", "cols", "when", "as", "t2")

	vars := restore(t)
	loaded := tableVar(t, vars, "t")
	parsed := tableVar(t, vars, "t2")

	for label, dt := range map[string]*insyra.DataTable{"t": loaded, "t2": parsed} {
		if dt.NumCols() != 5 {
			t.Errorf("%s has %d columns, want 5: %s", label, dt.NumCols(), strings.Join(columnNames(dt), ", "))
			continue
		}
		// Load order, not alphabetical: alpha would come first if the
		// environment had rebuilt the table from a map of columns.
		for i, want := range []string{"zeta", "id", "score", "alpha", "when"} {
			if got := dt.GetColByNumber(i).GetName(); got != want {
				t.Errorf("%s column %d is %q, want %q (order: %s)", label, i, got, want, strings.Join(columnNames(dt), ", "))
			}
		}
		if dt.NumRows() != 2 {
			t.Errorf("%s has %d rows, want 2", label, dt.NumRows())
		}
		// 3 and 1.5 in one column make it a float column; a round trip that
		// narrowed per cell would hand back int64(3).
		requireFloat64(t, dt, "zeta", 0, 3)
		requireFloat64(t, dt, "zeta", 1, 1.5)
		// 7 and 8 in one column make it an integer column, and an integer
		// column is int64 everywhere else in the library — coming back as
		// float64 or as int would be a different answer on each platform.
		requireInt64(t, dt, "id", 0, 7)
		requireInt64(t, dt, "id", 1, 8)
		// score column is float64 with both values present.
		requireFloat64(t, dt, "score", 0, 0.5)
		requireFloat64(t, dt, "score", 1, 2.25)
		requireString(t, dt, "alpha", 0, "x")
		requireString(t, dt, "alpha", 1, "y")
	}

	// The unparsed table still holds the date as text.
	requireString(t, loaded, "when", 0, "2024-01-02")
	requireString(t, loaded, "when", 1, "2024-03-04")

	// The parsed one holds time.Time, not the string it came from.
	requireTime(t, parsed, "when", 0, time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC))
	requireTime(t, parsed, "when", 1, time.Date(2024, 3, 4, 0, 0, 0, 0, time.UTC))

	// Column letter A is the first column, so it is still zeta.
	first := loaded.GetElementByNumberIndex(0, 0)
	if f, ok := first.(float64); !ok || f != 3 {
		t.Errorf("A1 is %T(%v), want float64(3)", first, first)
	}

	// And the restored table is a working input, not just a printable one: a
	// resample on it needs the dates as time.Time and the value as a number.
	runOneShot(t, "resample", "t2", "when", "monthly", "zeta:sum", "as", "r")
	tableVar(t, restore(t), "r")
}

// A DataList the user typed by hand is the one place every cell type arrives
// on purpose rather than by inference: an int, two floats, a bool, a nil, a
// string, and a NaN in a single list. The second command has to see the same
// seven cells, so the round trip may not flatten them into text or into one
// common type.
func TestOneShotKeepsListCellTypes(t *testing.T) {
	env.SetBasePath(t.TempDir())
	t.Cleanup(func() { env.SetBasePath("") })

	runOneShot(t, "newdl", "1", "2.5", "3.0", "true", "nil", "x", "NaN", "as", "l")

	vars := restore(t)
	list, ok := vars["l"].(*insyra.DataList)
	if !ok {
		t.Fatalf("l is %T, want *insyra.DataList", vars["l"])
	}
	if list.Len() != 7 {
		t.Fatalf("l has %d cells, want 7: %v", list.Len(), list.Data())
	}

	// Type first, then value: 3.0 has to stay float64 rather than being read
	// back as the int 3, nil has to stay nil rather than becoming "", and NaN
	// has to stay a float64 NaN.
	want := []struct {
		typ   string
		value any
	}{
		{"int", 1},
		{"float64", 2.5},
		{"float64", 3.0},
		{"bool", true},
		{"<nil>", nil},
		{"string", "x"},
		{"float64", float64(0)}, // NaN, value checked separately
	}
	for i, w := range want {
		got := list.Get(i)
		if typ := fmt.Sprintf("%T", got); typ != w.typ {
			t.Errorf("l[%d] is %s (%v), want %s", i, typ, got, w.typ)
			continue
		}
		if i == 6 {
			f, ok := got.(float64)
			if !ok || !math.IsNaN(f) {
				t.Errorf("l[6] = %v, want float64 NaN", got)
			}
			continue
		}
		if got != w.value {
			t.Errorf("l[%d] = %v, want %v", i, got, w.value)
		}
	}
}

// A fitted scaler is reusable across invocations only if it survives the
// environment, and if it does, it transforms a later table the way the scaler
// fitted in the earlier invocation would.
func TestOneShotKeepsFittedScaler(t *testing.T) {
	env.SetBasePath(t.TempDir())
	t.Cleanup(func() { env.SetBasePath("") })

	csvPath := writeCSV(t, "mixed.csv", "zeta,alpha,when\n3,x,2024-01-02\n1.5,y,2024-03-04\n")

	runOneShot(t, "load", csvPath, "as", "t")
	runOneShot(t, "scale", "fit", "std", "sc", "t", "cols", "zeta")
	runOneShot(t, "scale", "transform", "sc", "t", "as", "t3")

	vars := restore(t)
	if _, ok := vars["sc"].(*insyra.StandardScaler); !ok {
		t.Fatalf("sc is %T, want *insyra.StandardScaler", vars["sc"])
	}
	scaled := tableVar(t, vars, "t3")
	restoredTable := tableVar(t, vars, "t")

	// Fitting here in this process, from the table the environment handed
	// back, gives the answer the saved scaler has to reproduce.
	want, err := insyra.NewStandardScaler().FitTransform(restoredTable, insyra.Name("zeta"))
	if err != nil {
		t.Fatalf("FitTransform: %v", err)
	}
	got := scaled.GetColByNumber(colIndex(t, scaled, "zeta"))
	expected := want.GetColByNumber(colIndex(t, want, "zeta"))
	if got.Len() != expected.Len() {
		t.Fatalf("scaled zeta has %d cells, want %d", got.Len(), expected.Len())
	}
	for i := 0; i < got.Len(); i++ {
		gotF, gotOK := got.Get(i).(float64)
		wantF, wantOK := expected.Get(i).(float64)
		if !gotOK || !wantOK {
			t.Fatalf("zeta[%d]: t3 holds %T, want %T", i, got.Get(i), expected.Get(i))
		}
		// Bits, not a tolerance: the saved scaler has to scale identically,
		// so a mean or stdev that lost precision shows up here.
		if math.Float64bits(gotF) != math.Float64bits(wantF) {
			t.Errorf("zeta[%d]: t3 = %v (%#016x), want %v (%#016x)",
				i, gotF, math.Float64bits(gotF), wantF, math.Float64bits(wantF))
		}
	}
}

// A hierarchical tree is only reusable if it survives the environment as a
// tree: cutree in the next invocation has to be able to cut it, and the
// labels have to come back out of the environment as a list.
func TestOneShotKeepsHierarchicalTree(t *testing.T) {
	env.SetBasePath(t.TempDir())
	t.Cleanup(func() { env.SetBasePath("") })

	csvPath := writeCSV(t, "points.csv", "x,y\n1,2\n2,3\n8,9\n9,9\n")

	runOneShot(t, "load", csvPath, "as", "h")
	runOneShot(t, "hclust", "h", "complete", "as", "tr")
	// cutree is a separate invocation from hclust, so reaching here means the
	// tree was read back out of the environment first.
	runOneShot(t, "cutree", "tr", "k", "2", "as", "lab")

	vars := restore(t)
	if _, ok := vars["tr"].(*stats.HierarchicalResult); !ok {
		t.Fatalf("tr is %T, want *stats.HierarchicalResult", vars["tr"])
	}
	labels, ok := vars["lab"].(*insyra.DataList)
	if !ok {
		t.Fatalf("lab is %T, want *insyra.DataList", vars["lab"])
	}
	if labels.Len() != 4 {
		t.Fatalf("lab has %d cells, want 4: %v", labels.Len(), labels.Data())
	}
	// (1,2) and (2,3) are 1.41 apart, (8,9) and (9,9) are 1 apart, and the two
	// pairs are 9.9 apart, so k=2 splits them down the middle.
	if labels.Get(0) != labels.Get(1) {
		t.Errorf("lab[0]=%v and lab[1]=%v differ, but (1,2) and (2,3) are the closest pair", labels.Get(0), labels.Get(1))
	}
	if labels.Get(2) != labels.Get(3) {
		t.Errorf("lab[2]=%v and lab[3]=%v differ, but (8,9) and (9,9) belong together", labels.Get(2), labels.Get(3))
	}
	if labels.Get(0) == labels.Get(2) {
		t.Errorf("lab[0] and lab[2] are both %v, so the two pairs were not split", labels.Get(0))
	}
}

// CCL 的日期相減會產生 time.Duration，環境以前存不了它，會把整張表丟掉。
func TestOneShotKeepsDurationColumn(t *testing.T) {
	env.SetBasePath(t.TempDir())
	t.Cleanup(func() { env.SetBasePath("") })

	csvPath := writeCSV(t, "dates.csv", "start,end\n2024-01-02,2024-01-05\n2024-03-04,2024-03-04\n")

	runOneShot(t, "load", csvPath, "as", "d")
	runOneShot(t, "parsedates", "d", "cols", "start,end", "as", "d2")
	runOneShot(t, "addcolccl", "d2", "gap", "B - A")

	vars := restore(t)
	d2 := tableVar(t, vars, "d2")

	if d2.NumCols() != 3 {
		t.Fatalf("d2 has %d columns, want 3: %s", d2.NumCols(), strings.Join(columnNames(d2), ", "))
	}
	wantNames := []string{"start", "end", "gap"}
	for i, want := range wantNames {
		if got := d2.GetColByNumber(i).GetName(); got != want {
			t.Errorf("d2 column %d is %q, want %q", i, got, want)
		}
	}
	if d2.NumRows() != 2 {
		t.Fatalf("d2 has %d rows, want 2", d2.NumRows())
	}

	// 第 0 格：2024-01-05 - 2024-01-02 = 3 天 = 72 小時
	gap0 := cellAt(t, d2, "gap", 0)
	if typ := fmt.Sprintf("%T", gap0); typ != "time.Duration" {
		t.Fatalf("gap[0] is %s (%v), want time.Duration", typ, gap0)
	}
	if gap0 != time.Duration(72*time.Hour) {
		t.Errorf("gap[0] = %v, want %v", gap0, time.Duration(72*time.Hour))
	}

	// 第 1 格：同一天相減 = 0
	gap1 := cellAt(t, d2, "gap", 1)
	if typ := fmt.Sprintf("%T", gap1); typ != "time.Duration" {
		t.Fatalf("gap[1] is %s (%v), want time.Duration", typ, gap1)
	}
	if gap1 != time.Duration(0) {
		t.Errorf("gap[1] = %v, want %v", gap1, time.Duration(0))
	}
}
