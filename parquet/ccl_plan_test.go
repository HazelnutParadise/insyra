package parquet

import (
	"context"
	"errors"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra"
)

// planRows is more than two batches, so a plan that holds only for one batch
// is caught.
const planRows = 2500

// writePlanFixture writes a file whose column A holds 0 to planRows-1 and
// column B holds A modulo 7.
func writePlanFixture(t *testing.T) string {
	t.Helper()
	a := make([]any, planRows)
	b := make([]any, planRows)
	for i := range a {
		a[i] = i
		b[i] = i % 7
	}
	path := filepath.Join(t.TempDir(), "plan.parquet")
	dt := insyra.NewDataTable(insyra.NewDataList(a...).SetName("A"), insyra.NewDataList(b...).SetName("B"))
	if err := Write(dt, path); err != nil {
		t.Fatalf("writing the fixture: %v", err)
	}
	return path
}

// passesOf runs fn and returns how many times it read the file's data.
func passesOf(t *testing.T, fn func() error) int {
	t.Helper()
	before := cclDataPasses.Load()
	if err := fn(); err != nil {
		t.Fatal(err)
	}
	return int(cclDataPasses.Load() - before)
}

func TestExplainFilterWithCCLMatchesTheCall(t *testing.T) {
	path := writePlanFixture(t)
	ctx := context.Background()
	cases := []struct {
		expr   string
		mode   CCLMode
		held   []string
		reason string // part of a reason, empty when there is none
	}{
		{expr: "A > 100", mode: CCLStreamed},
		{expr: "A > AVG(A)", mode: CCLStreamed},
		{expr: "A > STDEV(A)", mode: CCLStreamed},
		{expr: "A > AVG(A) + STDEV(B)", mode: CCLStreamed},
		{expr: "A > -1", mode: CCLStreamed},
		{expr: "A > A.0 + MAX(B)", mode: CCLStreamed},
		{expr: "IF(# > 0, A.(# - 1), 0) > 10", mode: CCLWholeColumns, held: []string{"A"}, reason: "row"},
		{expr: "LAG(A, 1)", mode: CCLStreamedSequence},
		{expr: "A > MEDIAN(A)", mode: CCLWholeColumns, held: []string{"A"}, reason: "MEDIAN"},
		{expr: "COUNT(@) > 0", mode: CCLStreamed},
	}
	for _, tt := range cases {
		t.Run(tt.expr, func(t *testing.T) {
			plan, err := ExplainFilterWithCCL(ctx, path, tt.expr)
			if err != nil {
				t.Fatalf("ExplainFilterWithCCL: %v", err)
			}
			if plan.Mode != tt.mode {
				t.Errorf("mode %v, want %v (reasons %q)", plan.Mode, tt.mode, plan.Reasons)
			}
			if !slices.Equal(plan.HeldColumns, tt.held) {
				t.Errorf("held columns %q, want %q", plan.HeldColumns, tt.held)
			}
			if tt.reason != "" && !strings.Contains(strings.Join(plan.Reasons, " "), tt.reason) {
				t.Errorf("reasons %q do not name %s", plan.Reasons, tt.reason)
			}
			if plan.NumRows != planRows || !plan.HoldsResult || plan.MayReadMore {
				t.Errorf("NumRows %d, HoldsResult %v, MayReadMore %v", plan.NumRows, plan.HoldsResult, plan.MayReadMore)
			}
			wantCells := int64(0)
			if tt.mode == CCLWholeColumns {
				wantCells = int64(planRows*len(tt.held) + planRows)
			}
			if plan.HeldCells != wantCells {
				t.Errorf("HeldCells %d, want %d", plan.HeldCells, wantCells)
			}
			if tt.mode == CCLWholeColumns && plan.EstimatedHeldBytes <= 0 {
				t.Errorf("EstimatedHeldBytes %d for held columns", plan.EstimatedHeldBytes)
			}
			got := passesOf(t, func() error {
				_, err := FilterWithCCL(ctx, path, tt.expr)
				return err
			})
			if got != plan.MinDataPasses {
				t.Errorf("FilterWithCCL read the data %d times, the plan says %d", got, plan.MinDataPasses)
			}
		})
	}
}

func TestExplainReadsNoData(t *testing.T) {
	path := writePlanFixture(t)
	got := passesOf(t, func() error {
		_, err := ExplainFilterWithCCL(context.Background(), path, "A > AVG(A) + MEDIAN(B)")
		return err
	})
	if got != 0 {
		t.Errorf("ExplainFilterWithCCL read the data %d times", got)
	}
}

func TestExplainFlagsAPeriodComputedFromTheData(t *testing.T) {
	plan, err := ExplainFilterWithCCL(context.Background(), writePlanFixture(t), "LAG(A, COUNT(B))")
	if err != nil {
		t.Fatal(err)
	}
	if !plan.MayReadMore || !strings.Contains(strings.Join(plan.Reasons, " "), "COUNT") {
		t.Errorf("MayReadMore %v, reasons %q", plan.MayReadMore, plan.Reasons)
	}
}

func TestExplainRefusesWhatTheCallRefuses(t *testing.T) {
	path := writePlanFixture(t)
	_, callErr := FilterWithCCL(context.Background(), path, "A >")
	_, planErr := ExplainFilterWithCCL(context.Background(), path, "A >")
	if callErr == nil || planErr == nil || callErr.Error() != planErr.Error() {
		t.Errorf("A >: the call said %v, Explain said %v", callErr, planErr)
	}
	// The call fails on a name the file does not have only when it evaluates
	// the first row, so Explain cannot refuse it, and says so instead.
	plan, err := ExplainFilterWithCCL(context.Background(), path, "['nope'] > 1")
	if err != nil || !strings.Contains(strings.Join(plan.Reasons, " "), "nope") {
		t.Errorf("['nope'] > 1: got %+v, %v; want a reason naming the column", plan, err)
	}
	if _, err := ExplainFilterWithCCL(context.Background(), filepath.Join(t.TempDir(), "missing.parquet"), "A > 1"); err == nil {
		t.Error("a missing file gave no error")
	}
}

func TestExplainApplyCCLPlansEachStatement(t *testing.T) {
	path := writePlanFixture(t)
	script := "NEW('C') = A * 2; A + 1; NEW('D') = A - MEDIAN(A); NEW('E') = B - AVG(B)"
	plans, err := ExplainApplyCCL(context.Background(), path, script)
	if err != nil {
		t.Fatal(err)
	}
	if len(plans) != 4 {
		t.Fatalf("%d plans for four statements", len(plans))
	}
	if plans[0].Mode != CCLStreamed || plans[0].MinDataPasses != 0 {
		t.Errorf("NEW('C') = A * 2: %+v", plans[0])
	}
	if plans[1].MinDataPasses != 0 || !strings.Contains(strings.Join(plans[1].Reasons, " "), "writes nothing") {
		t.Errorf("A + 1: %+v", plans[1])
	}
	if plans[2].Mode != CCLWholeColumns || !slices.Equal(plans[2].HeldColumns, []string{"A"}) || plans[2].MinDataPasses != 1 || plans[2].HeldCells != 2*planRows {
		t.Errorf("NEW('D') = A - MEDIAN(A): %+v", plans[2])
	}
	if plans[3].Mode != CCLStreamed || plans[3].MinDataPasses != 1 {
		t.Errorf("NEW('E') = B - AVG(B): %+v", plans[3])
	}
	for _, p := range plans {
		if p.HoldsResult {
			t.Errorf("an ApplyCCL plan holds a result: %+v", p)
		}
	}
	want := 1
	for _, p := range plans {
		want += p.MinDataPasses
	}
	if got := passesOf(t, func() error { return ApplyCCL(context.Background(), path, script) }); got != want {
		t.Errorf("ApplyCCL read the data %d times, the plans add up to %d", got, want)
	}
}

func TestFilterWithCCLMaxRows(t *testing.T) {
	path := writePlanFixture(t)
	ctx := context.Background()
	for _, expr := range []string{"A > -1", "A > MEDIAN(A) - 2000", "LAG(A, 1)"} {
		dt, err := FilterWithCCL(ctx, path, expr, FilterWithCCLOptions{MaxRows: 1000})
		if dt != nil || !errors.Is(err, ErrCCLRowLimit) || !strings.Contains(err.Error(), "1000") {
			t.Errorf("%s: got %v, %v; want nil and ErrCCLRowLimit naming 1000", expr, dt, err)
		}
	}
	dt, err := FilterWithCCL(ctx, path, "A < 999", FilterWithCCLOptions{MaxRows: 999})
	if err != nil {
		t.Fatal(err)
	}
	if rows, _ := dt.Size(); rows != 999 {
		t.Errorf("a filter keeping exactly MaxRows rows gave %d", rows)
	}
	if _, err := FilterWithCCL(ctx, path, "A > 0", FilterWithCCLOptions{MaxRows: -1}); err == nil {
		t.Error("a negative MaxRows gave no error")
	}
	if _, err := FilterWithCCL(ctx, path, "A > 0", FilterWithCCLOptions{}, FilterWithCCLOptions{}); err == nil {
		t.Error("two options values gave no error")
	}
}

func TestFilterWithCCLRefuseWholeColumns(t *testing.T) {
	path := writePlanFixture(t)
	ctx := context.Background()
	var dt *insyra.DataTable
	got := passesOf(t, func() error {
		var err error
		dt, err = FilterWithCCL(ctx, path, "A > MEDIAN(A)", FilterWithCCLOptions{RefuseWholeColumns: true})
		if dt != nil || !errors.Is(err, ErrCCLWholeColumnsRefused) || !strings.Contains(err.Error(), "MEDIAN") {
			t.Errorf("got %v, %v; want nil and ErrCCLWholeColumnsRefused naming MEDIAN", dt, err)
		}
		return nil
	})
	if got != 0 {
		t.Errorf("the refused filter read the data %d times", got)
	}
	if _, err := FilterWithCCL(ctx, path, "A > AVG(A)", FilterWithCCLOptions{RefuseWholeColumns: true}); err != nil {
		t.Errorf("a streamed filter under RefuseWholeColumns: %v", err)
	}
}

func TestFilterWithCCLStopsOnCancel(t *testing.T) {
	path := writePlanFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := FilterWithCCL(ctx, path, "A > -1", FilterWithCCLOptions{MaxRows: 10}); !errors.Is(err, context.Canceled) {
		t.Errorf("a cancelled context gave %v", err)
	}
	if _, err := ExplainFilterWithCCL(ctx, path, "A > -1"); !errors.Is(err, context.Canceled) {
		t.Errorf("Explain with a cancelled context gave %v", err)
	}
}

func TestCCLPlanString(t *testing.T) {
	plan, err := ExplainFilterWithCCL(context.Background(), writePlanFixture(t), "A > MEDIAN(A)")
	if err != nil {
		t.Fatal(err)
	}
	s := plan.String()
	for _, want := range []string{"whole columns", "MEDIAN", "Reads:", "A", "2,500 rows", "at least 2", "every kept row"} {
		if !strings.Contains(s, want) {
			t.Errorf("String() lacks %q:\n%s", want, s)
		}
	}
	if CCLMode(9).String() != "CCLMode(9)" || CCLStreamed.String() != "CCLStreamed" {
		t.Errorf("CCLMode names: %v, %v", CCLMode(9), CCLStreamed)
	}
}

// The review measured A > A.(MAX(B)) reading the file three times where the
// plan, read from no rows, said two and promised the count was exact.
func TestExplainFlagsARowComputedFromTheData(t *testing.T) {
	path := writePlanFixture(t)
	ctx := context.Background()
	for _, expr := range []string{"A > A.(MAX(B))", "A > A.(A.0)", "CUMSUM(A + A.(A.0))"} {
		plan, err := ExplainFilterWithCCL(ctx, path, expr)
		if err != nil {
			t.Fatal(err)
		}
		got := passesOf(t, func() error {
			_, err := FilterWithCCL(ctx, path, expr)
			return err
		})
		if !plan.MayReadMore || got < plan.MinDataPasses {
			t.Errorf("%s: MayReadMore %v, plan %d passes, call %d", expr, plan.MayReadMore, plan.MinDataPasses, got)
		}
	}
}

// The review measured ExplainApplyCCL accepting Z = A + 1 on a file without
// Z, which ApplyCCL refuses before it writes.
func TestExplainApplyCCLRefusesAMissingTarget(t *testing.T) {
	path := writePlanFixture(t)
	callErr := ApplyCCL(context.Background(), path, "Z = A + 1")
	_, planErr := ExplainApplyCCL(context.Background(), path, "Z = A + 1")
	if callErr == nil || planErr == nil || callErr.Error() != planErr.Error() {
		t.Errorf("the call said %v, Explain said %v", callErr, planErr)
	}
}

func TestApplyCCLRefusesANilContext(t *testing.T) {
	//lint:ignore SA1012 the nil context is what is being tested
	if err := ApplyCCL(nil, writePlanFixture(t), "NEW('C') = A"); !errors.Is(err, errNilContext) { //nolint:staticcheck
		t.Errorf("got %v", err)
	}
}
