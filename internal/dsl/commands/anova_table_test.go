package commands

import (
	"fmt"
	"testing"

	insyra "github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats"
)

func twoWayTable() *insyra.DataTable {
	return insyra.NewDataTable(
		insyra.NewDataList(5.1, 6.2, 4.8, 6.9, 5.5, 6.0, 4.2, 5.3, 4.9, 5.8, 4.4, 5.0).SetName("score"),
		insyra.NewDataList("placebo", "placebo", "placebo", "placebo", "placebo", "placebo", "new", "new", "new", "new", "new", "new").SetName("drug"),
		insyra.NewDataList("low", "high", "low", "high", "low", "high", "low", "high", "low", "high", "low", "high").SetName("dose"),
	)
}

func repeatedTable() *insyra.DataTable {
	return insyra.NewDataTable(
		insyra.NewDataList(10.0, 12.0, 15.0, 11.0, 13.0, 13.0, 9.0, 12.0, 14.0, 12.0, 15.0, 17.0).SetName("value"),
		insyra.NewDataList("t1", "t2", "t3", "t1", "t2", "t3", "t1", "t2", "t3", "t1", "t2", "t3").SetName("cond"),
		insyra.NewDataList("s1", "s1", "s1", "s2", "s2", "s2", "s3", "s3", "s3", "s4", "s4", "s4").SetName("subj"),
	)
}

func repeatedTableMissingRow() *insyra.DataTable {
	return insyra.NewDataTable(
		insyra.NewDataList(10.0, 12.0, 15.0, 11.0, 13.0, 13.0, 9.0, 12.0, 14.0, 12.0, 15.0).SetName("value"),
		insyra.NewDataList("t1", "t2", "t3", "t1", "t2", "t3", "t1", "t2", "t3", "t1", "t2").SetName("cond"),
		insyra.NewDataList("s1", "s1", "s1", "s2", "s2", "s2", "s3", "s3", "s3", "s4", "s4").SetName("subj"),
	)
}

func cellLists() []*insyra.DataList {
	c11 := insyra.NewDataList(5.1, 4.8, 5.5)
	c12 := insyra.NewDataList(6.2, 6.9, 6.0)
	c21 := insyra.NewDataList(4.2, 4.9, 4.4)
	c22 := insyra.NewDataList(5.3, 5.8, 5.0)
	return []*insyra.DataList{c11, c12, c21, c22}
}

func subjectLists() []*insyra.DataList {
	s1 := insyra.NewDataList(10.0, 12.0, 15.0)
	s2 := insyra.NewDataList(11.0, 13.0, 13.0)
	s3 := insyra.NewDataList(9.0, 12.0, 14.0)
	s4 := insyra.NewDataList(12.0, 15.0, 17.0)
	return []*insyra.DataList{s1, s2, s3, s4}
}

func runAnova(ctx *ExecContext, args ...string) (string, error) {
	err := Dispatch(ctx, "anova", args)
	return outputOf(ctx), err
}

func TestAnovaTableTwoWayMatchesLibraryAndLists(t *testing.T) {
	twoWay := twoWayTable()
	libResult, err := stats.TwoWayANOVAFromTable(twoWay, insyra.Name("score"), insyra.Name("drug"), insyra.Name("dose"))
	if err != nil {
		t.Fatalf("library TwoWayANOVAFromTable failed: %v", err)
	}
	expected := fmt.Sprintf("FA=%v pA=%v FB=%v pB=%v\n", libResult.FactorA.F, libResult.FactorA.P, libResult.FactorB.F, libResult.FactorB.P)

	ctx := newTestExecContext(t)
	ctx.Vars["t"] = twoWay
	out, err := runAnova(ctx, "twoway", "t", "score", "drug", "dose")
	if err != nil {
		t.Fatalf("anova twoway table form failed: %v", err)
	}
	if out != expected {
		t.Errorf("table form output %q != library output %q", out, expected)
	}

	cells := cellLists()
	ctx2 := newTestExecContext(t)
	ctx2.Vars["c11"] = cells[0]
	ctx2.Vars["c12"] = cells[1]
	ctx2.Vars["c21"] = cells[2]
	ctx2.Vars["c22"] = cells[3]
	listOut, err := runAnova(ctx2, "twoway", "2", "2", "c11", "c12", "c21", "c22")
	if err != nil {
		t.Fatalf("anova twoway list form failed: %v", err)
	}
	if listOut != expected {
		t.Errorf("list form output %q != expected %q", listOut, expected)
	}
}

func TestAnovaTableTokens(t *testing.T) {
	twoWay := twoWayTable()
	libResult, err := stats.TwoWayANOVAFromTable(twoWay, insyra.Name("score"), insyra.Name("drug"), insyra.Name("dose"))
	if err != nil {
		t.Fatalf("library TwoWayANOVAFromTable failed: %v", err)
	}
	expected := fmt.Sprintf("FA=%v pA=%v FB=%v pB=%v\n", libResult.FactorA.F, libResult.FactorA.P, libResult.FactorB.F, libResult.FactorB.P)

	tests := [][]string{
		{"twoway", "t", "A", "B", "C"},
		{"twoway", "t", "0", "1", "2"},
		{"twoway", "t", "name:score", "name:drug", "name:dose"},
	}
	for _, args := range tests {
		ctx := newTestExecContext(t)
		ctx.Vars["t"] = twoWay
		out, err := runAnova(ctx, args...)
		if err != nil {
			t.Errorf("anova %v failed: %v", args, err)
			continue
		}
		if out != expected {
			t.Errorf("anova %v output %q != expected %q", args, out, expected)
		}
	}
}

func TestAnovaTableRepeatedMatchesLibraryAndLists(t *testing.T) {
	rep := repeatedTable()
	libResult, err := stats.RepeatedMeasuresANOVAFromTable(rep, insyra.Name("value"), insyra.Name("cond"), insyra.Name("subj"))
	if err != nil {
		t.Fatalf("library RepeatedMeasuresANOVAFromTable failed: %v", err)
	}
	expected := fmt.Sprintf("F=%v p=%v\n", libResult.Factor.F, libResult.Factor.P)

	ctx := newTestExecContext(t)
	ctx.Vars["t"] = rep
	out, err := runAnova(ctx, "repeated", "t", "value", "cond", "subj")
	if err != nil {
		t.Fatalf("anova repeated table form failed: %v", err)
	}
	if out != expected {
		t.Errorf("table form output %q != library output %q", out, expected)
	}

	subjs := subjectLists()
	ctx2 := newTestExecContext(t)
	ctx2.Vars["s1"] = subjs[0]
	ctx2.Vars["s2"] = subjs[1]
	ctx2.Vars["s3"] = subjs[2]
	ctx2.Vars["s4"] = subjs[3]
	listOut, err := runAnova(ctx2, "repeated", "s1", "s2", "s3", "s4")
	if err != nil {
		t.Fatalf("anova repeated list form failed: %v", err)
	}
	if listOut != expected {
		t.Errorf("list form output %q != expected %q", listOut, expected)
	}
}

func TestAnovaTableErrors(t *testing.T) {
	twoWay := twoWayTable()

	ctx := newTestExecContext(t)
	ctx.Vars["t"] = twoWay
	_, err := runAnova(ctx, "twoway", "t", "score", "drug")
	if err == nil {
		t.Error("anova twoway t score drug should have failed with usage error")
	} else if err.Error() != "usage: anova twoway <table> <value> <factorA> <factorB>" {
		t.Errorf("wrong error text: got %q, want %q", err.Error(), "usage: anova twoway <table> <value> <factorA> <factorB>")
	}

	ctx2 := newTestExecContext(t)
	ctx2.Vars["t"] = twoWay
	_, err = runAnova(ctx2, "twoway", "t", "price", "drug", "dose")
	if err == nil {
		t.Error("anova twoway t price drug dose should have failed (column does not exist)")
	}

	repMissing := repeatedTableMissingRow()
	ctx3 := newTestExecContext(t)
	ctx3.Vars["t"] = repMissing
	_, err = runAnova(ctx3, "repeated", "t", "value", "cond", "subj")
	if err == nil {
		t.Error("anova repeated on table with missing row should have failed")
	} else if !contains(err.Error(), "subject s4 has no observation for condition t3") {
		t.Errorf("error does not contain expected text: got %q", err.Error())
	}
}

func TestAnovaListFormsUnchanged(t *testing.T) {
	cells := cellLists()
	ctx := newTestExecContext(t)
	ctx.Vars["c11"] = cells[0]
	ctx.Vars["c12"] = cells[1]
	ctx.Vars["c21"] = cells[2]
	ctx.Vars["c22"] = cells[3]
	out, err := runAnova(ctx, "twoway", "2", "2", "c11", "c12", "c21", "c22")
	if err != nil {
		t.Fatalf("anova twoway list form failed: %v", err)
	}
	if !startsWith(out, "FA=") {
		t.Errorf("anova twoway list output does not start with FA=: %q", out)
	}

	subjs := subjectLists()
	ctx2 := newTestExecContext(t)
	ctx2.Vars["s1"] = subjs[0]
	ctx2.Vars["s2"] = subjs[1]
	ctx2.Vars["s3"] = subjs[2]
	ctx2.Vars["s4"] = subjs[3]
	out2, err := runAnova(ctx2, "repeated", "s1", "s2", "s3", "s4")
	if err != nil {
		t.Fatalf("anova repeated list form failed: %v", err)
	}
	if !startsWith(out2, "F=") {
		t.Errorf("anova repeated list output does not start with F=: %q", out2)
	}
}

func startsWith(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
