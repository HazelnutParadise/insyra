package commands

import (
	"bytes"
	"fmt"
	"math"
	"strings"
	"testing"

	insyra "github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats"
)

func friedmanTable() *insyra.DataTable {
	return insyra.NewDataTable(
		insyra.NewDataList(10.0, 12.0, 15.0, 11.0, 13.0, 13.0, 9.0, 12.0, 14.0, 12.0, 15.0, 17.0).SetName("value"),
		insyra.NewDataList("t1", "t2", "t3", "t1", "t2", "t3", "t1", "t2", "t3", "t1", "t2", "t3").SetName("cond"),
		insyra.NewDataList("s1", "s1", "s1", "s2", "s2", "s2", "s3", "s3", "s3", "s4", "s4", "s4").SetName("subj"),
	)
}

func friedmanTableMissingRow() *insyra.DataTable {
	return insyra.NewDataTable(
		insyra.NewDataList(10.0, 12.0, 15.0, 11.0, 13.0, 13.0, 9.0, 12.0, 14.0, 12.0, 15.0).SetName("value"),
		insyra.NewDataList("t1", "t2", "t3", "t1", "t2", "t3", "t1", "t2", "t3", "t1", "t2").SetName("cond"),
		insyra.NewDataList("s1", "s1", "s1", "s2", "s2", "s2", "s3", "s3", "s3", "s4", "s4").SetName("subj"),
	)
}

func friedmanSubjectLists() []*insyra.DataList {
	s1 := insyra.NewDataList(10.0, 12.0, 15.0)
	s2 := insyra.NewDataList(11.0, 13.0, 13.0)
	s3 := insyra.NewDataList(9.0, 12.0, 14.0)
	s4 := insyra.NewDataList(12.0, 15.0, 17.0)
	return []*insyra.DataList{s1, s2, s3, s4}
}

func runFriedman(ctx *ExecContext, args ...string) (string, error) {
	err := Dispatch(ctx, "friedman", args)
	return outputOf(ctx), err
}

func TestFriedmanTableMatchesLibraryAndLists(t *testing.T) {
	libResult, err := stats.FriedmanTestFromTable(friedmanTable(), insyra.Name("value"), insyra.Name("cond"), insyra.Name("subj"))
	if err != nil {
		t.Fatalf("library FriedmanTestFromTable failed: %v", err)
	}
	df := math.NaN()
	if libResult.DF != nil {
		df = *libResult.DF
	}
	expected := fmt.Sprintf("Q=%v df=%v p=%v\n", libResult.Statistic, df, libResult.PValue)

	ctx := newTestExecContext(t)
	ctx.Vars["t"] = friedmanTable()
	out, err := runFriedman(ctx, "t", "value", "cond", "subj")
	if err != nil {
		t.Fatalf("friedman table form failed: %v", err)
	}
	if out != expected {
		t.Errorf("table form output %q != library output %q", out, expected)
	}

	subjs := friedmanSubjectLists()
	ctx2 := newTestExecContext(t)
	ctx2.Vars["s1"] = subjs[0]
	ctx2.Vars["s2"] = subjs[1]
	ctx2.Vars["s3"] = subjs[2]
	ctx2.Vars["s4"] = subjs[3]
	listOut, err := runFriedman(ctx2, "s1", "s2", "s3", "s4")
	if err != nil {
		t.Fatalf("friedman list form failed: %v", err)
	}
	if listOut != expected {
		t.Errorf("list form output %q != expected %q", listOut, expected)
	}
}

func TestFriedmanErrors(t *testing.T) {
	ctx := newTestExecContext(t)
	_, err := runFriedman(ctx)
	if err == nil {
		t.Error("friedman with no args should have failed with usage error")
	} else {
		if err.Error() != "usage: "+friedmanUsage {
			t.Errorf("wrong error text: got %q, want %q", err.Error(), "usage: "+friedmanUsage)
		}
	}

	ctx2 := newTestExecContext(t)
	ctx2.Vars["t"] = friedmanTable()
	_, err = runFriedman(ctx2, "t", "value", "cond")
	if err == nil {
		t.Error("friedman t value cond should have failed with usage error")
	} else if err.Error() != "usage: friedman <table> <value> <condition> <subject>" {
		t.Errorf("wrong error text: got %q, want %q", err.Error(), "usage: friedman <table> <value> <condition> <subject>")
	}

	ctx3 := newTestExecContext(t)
	ctx3.Vars["s1"] = insyra.NewDataList(1.0, 2.0, 3.0)
	_, err = runFriedman(ctx3, "s1")
	if err == nil {
		t.Error("friedman s1 (one DataList) should have failed with usage error")
	} else if err.Error() != "usage: friedman <subject1> <subject2> [subjectN]" {
		t.Errorf("wrong error text: got %q, want %q", err.Error(), "usage: friedman <subject1> <subject2> [subjectN]")
	}

	repMissing := friedmanTableMissingRow()
	ctx4 := newTestExecContext(t)
	ctx4.Vars["t"] = repMissing
	_, err = runFriedman(ctx4, "t", "value", "cond", "subj")
	if err == nil {
		t.Error("friedman on table with missing row should have failed")
	} else if !strings.Contains(err.Error(), "subject s4 has no observation for condition t3") {
		t.Errorf("error does not contain expected text: got %q", err.Error())
	}
}

func TestFriedmanHelpListsBothForms(t *testing.T) {
	ctx := newTestExecContext(t)
	if err := runHelpCommand(ctx, []string{"friedman"}); err != nil {
		t.Fatalf("help friedman failed: %v", err)
	}
	got := ctx.Output.(*bytes.Buffer).String()
	for _, want := range []string{
		"friedman <table> <value> <condition> <subject>",
		"friedman <subject1> <subject2> [subjectN]",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in help output, got:\n%s", want, got)
		}
	}
}
