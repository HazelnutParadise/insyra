package insyra

import (
	"math"
	"testing"
)

// TestDataTable_CCL_StdlibIntegration exercises a representative slice of the
// expanded CCL standard library through the real DataTable code path
// (AddColUsingCCL + ExecuteCCL). This is a smoke test for issue #154 — it
// confirms math, string, type-conversion, and statistical-aggregate
// functions are reachable from the public API, not just registered.
func TestDataTable_CCL_StdlibIntegration(t *testing.T) {
	dt := NewDataTable(
		NewDataList(-3, 4, -5, 0).SetName("X"),
		NewDataList("Foo", " Bar ", "BAZ", "qux").SetName("Name"),
		NewDataList(1.0, 2.0, 3.0, 4.0).SetName("Vals"),
	)

	// Math: ABS, ROUND, SQRT, SIGN
	dt.AddColUsingCCL("AbsX", "ABS(A)")
	dt.AddColUsingCCL("RoundedHalf", "ROUND(C / 3, 2)")
	dt.AddColUsingCCL("SqrtVals", "SQRT(C)")
	dt.AddColUsingCCL("SignX", "SIGN(A)")

	// String: TRIM + UPPER chained, LEN, CONTAINS, LEFT
	dt.AddColUsingCCL("CleanName", "UPPER(TRIM(B))")
	dt.AddColUsingCCL("NameLen", "LEN(B)")
	dt.AddColUsingCCL("HasA", "CONTAINS(LOWER(B), 'a')")
	dt.AddColUsingCCL("Prefix", "LEFT(B, 2)")

	// Type conversion / null handling
	dt.AddColUsingCCL("XStr", "TOSTR(A)")
	dt.AddColUsingCCL("XOrZero", "COALESCE(A, 0)")

	// Aggregates: STDEV / VAR / MEDIAN broadcast across rows
	dt.AddColUsingCCL("ValsStdev", "STDEV(C)")
	dt.AddColUsingCCL("ValsMedian", "MEDIAN(C)")

	if err := dt.Err(); err != nil {
		t.Fatalf("unexpected DataTable error: %s", err.Message)
	}

	getF := func(name string) []float64 {
		t.Helper()
		col := dt.GetColByName(name)
		if col == nil {
			t.Fatalf("column %q not found", name)
		}
		out := make([]float64, 0, len(col.Data()))
		for _, v := range col.Data() {
			f, ok := v.(float64)
			if !ok {
				t.Fatalf("column %q: expected float64 row, got %T (%v)", name, v, v)
			}
			out = append(out, f)
		}
		return out
	}
	getS := func(name string) []string {
		t.Helper()
		col := dt.GetColByName(name)
		if col == nil {
			t.Fatalf("column %q not found", name)
		}
		out := make([]string, 0, len(col.Data()))
		for _, v := range col.Data() {
			s, ok := v.(string)
			if !ok {
				t.Fatalf("column %q: expected string row, got %T (%v)", name, v, v)
			}
			out = append(out, s)
		}
		return out
	}

	checkF := func(name string, want []float64) {
		t.Helper()
		got := getF(name)
		if len(got) != len(want) {
			t.Fatalf("%s: length mismatch got=%d want=%d", name, len(got), len(want))
		}
		for i := range got {
			if math.Abs(got[i]-want[i]) > 1e-9 {
				t.Errorf("%s row %d: got %v want %v", name, i, got[i], want[i])
			}
		}
	}
	checkS := func(name string, want []string) {
		t.Helper()
		got := getS(name)
		if len(got) != len(want) {
			t.Fatalf("%s: length mismatch got=%d want=%d", name, len(got), len(want))
		}
		for i := range got {
			if got[i] != want[i] {
				t.Errorf("%s row %d: got %q want %q", name, i, got[i], want[i])
			}
		}
	}

	checkF("AbsX", []float64{3, 4, 5, 0})
	checkF("SqrtVals", []float64{1, math.Sqrt(2), math.Sqrt(3), 2})
	checkF("SignX", []float64{-1, 1, -1, 0})
	checkS("CleanName", []string{"FOO", "BAR", "BAZ", "QUX"})
	checkF("NameLen", []float64{3, 5, 3, 3})

	// HasA: "foo" no, " bar " yes, "baz" yes, "qux" no
	hasA := dt.GetColByName("HasA").Data()
	wantHas := []bool{false, true, true, false}
	for i, v := range hasA {
		b, ok := v.(bool)
		if !ok || b != wantHas[i] {
			t.Errorf("HasA row %d: got %v want %v", i, v, wantHas[i])
		}
	}

	checkS("Prefix", []string{"Fo", " B", "BA", "qu"})

	// Aggregate broadcast: STDEV/MEDIAN of {1,2,3,4} across 4 rows.
	wantStdev := math.Sqrt(((1-2.5)*(1-2.5) + (2-2.5)*(2-2.5) + (3-2.5)*(3-2.5) + (4-2.5)*(4-2.5)) / 3.0)
	for i, f := range getF("ValsStdev") {
		if math.Abs(f-wantStdev) > 1e-9 {
			t.Errorf("ValsStdev row %d: got %v want %v", i, f, wantStdev)
		}
	}
	for i, f := range getF("ValsMedian") {
		if f != 2.5 {
			t.Errorf("ValsMedian row %d: got %v want 2.5", i, f)
		}
	}
}

// TestDataTable_ExecuteCCL_StdlibPipeline runs a multi-statement pipeline
// that mixes new string and type helpers with assignment + NEW(), which is
// the path most likely to expose wiring bugs in Statement Mode.
func TestDataTable_ExecuteCCL_StdlibPipeline(t *testing.T) {
	dt := NewDataTable(
		NewDataList(" alice@Example.com ", "BOB@TEST.org", "carol@demo.io").SetName("Email"),
		NewDataList("19", "30", "not-a-number").SetName("AgeStr"),
	)

	dt.ExecuteCCL(`
		['Email'] = LOWER(TRIM(['Email']))
		NEW('Domain') = MID(['Email'], FIND('@', ['Email']) + 1, LEN(['Email']))
		NEW('AgeNum') = COALESCE(TONUM(['AgeStr']), 0)
	`)

	if err := dt.Err(); err != nil {
		t.Fatalf("unexpected DataTable error: %s", err.Message)
	}

	emails := dt.GetColByName("Email").Data()
	wantEmails := []string{"alice@example.com", "bob@test.org", "carol@demo.io"}
	for i, v := range emails {
		if v != wantEmails[i] {
			t.Errorf("Email row %d: got %v want %v", i, v, wantEmails[i])
		}
	}

	domains := dt.GetColByName("Domain").Data()
	wantDomains := []string{"example.com", "test.org", "demo.io"}
	for i, v := range domains {
		if v != wantDomains[i] {
			t.Errorf("Domain row %d: got %v want %v", i, v, wantDomains[i])
		}
	}

	ageNums := dt.GetColByName("AgeNum").Data()
	wantAges := []float64{19, 30, 0}
	for i, v := range ageNums {
		f, ok := v.(float64)
		if !ok || math.Abs(f-wantAges[i]) > 1e-9 {
			t.Errorf("AgeNum row %d: got %v want %v", i, v, wantAges[i])
		}
	}
}
