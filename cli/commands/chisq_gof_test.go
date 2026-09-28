package commands

import (
	"fmt"
	"strings"
	"testing"

	insyra "github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats"
)

func TestChiSqGof_LabeledProportions(t *testing.T) {
	dl := insyra.NewDataList("green", "red", "blue", "red", "red", "green", "blue", "red", "blue", "red")
	ctx := newTestExecContext(t)
	ctx.Vars["colors"] = dl

	// Expected result from the library
	want, err := stats.ChiSquareGoodnessOfFit(dl, map[string]float64{"red": 0.5, "green": 0.3, "blue": 0.2}, true)
	if err != nil {
		t.Fatalf("library call failed: %v", err)
	}
	wantOut := fmt.Sprintf("chi2=%v p=%v\n", want.Statistic, want.PValue)

	err = Dispatch(ctx, "chisq", []string{"gof", "colors", "red=0.5", "green=0.3", "blue=0.2"})
	if err != nil {
		t.Fatalf("chisq gof with labels failed: %v", err)
	}
	got := outputOf(ctx)
	if got != wantOut {
		t.Errorf("output mismatch\ngot:  %q\nwant: %q", got, wantOut)
	}
}

func TestChiSqGof_RejectsBareProportions(t *testing.T) {
	dl := insyra.NewDataList("green", "red", "blue", "red", "red", "green", "blue", "red", "blue", "red")
	ctx := newTestExecContext(t)
	ctx.Vars["colors"] = dl

	err := Dispatch(ctx, "chisq", []string{"gof", "colors", "0.5", "0.3", "0.2"})
	if err == nil {
		t.Fatal("chisq gof with bare proportions was accepted")
	}
	if !strings.Contains(err.Error(), `chisq gof: expected label=proportion, got "0.5"`) {
		t.Errorf("wrong error: %v", err)
	}
}

func TestChiSqGof_RejectsDuplicateLabel(t *testing.T) {
	dl := insyra.NewDataList("green", "red", "blue", "red", "red", "green", "blue", "red", "blue", "red")
	ctx := newTestExecContext(t)
	ctx.Vars["colors"] = dl

	err := Dispatch(ctx, "chisq", []string{"gof", "colors", "red=0.5", "red=0.5"})
	if err == nil {
		t.Fatal("chisq gof with duplicate label was accepted")
	}
	if !strings.Contains(err.Error(), `category "red" is given twice`) {
		t.Errorf("wrong error: %v", err)
	}
}

func TestChiSqGof_RejectsInvalidNumber(t *testing.T) {
	dl := insyra.NewDataList("green", "red", "blue", "red", "red", "green", "blue", "red", "blue", "red")
	ctx := newTestExecContext(t)
	ctx.Vars["colors"] = dl

	err := Dispatch(ctx, "chisq", []string{"gof", "colors", "red=abc"})
	if err == nil {
		t.Fatal("chisq gof with invalid number was accepted")
	}
	if !strings.Contains(err.Error(), "proportion") {
		t.Errorf("error should mention proportion; got: %v", err)
	}
}

func TestChiSqGof_UniformWhenNoProportions(t *testing.T) {
	dl := insyra.NewDataList("green", "red", "blue", "red", "red", "green", "blue", "red", "blue", "red")
	ctx := newTestExecContext(t)
	ctx.Vars["colors"] = dl

	want, err := stats.ChiSquareGoodnessOfFit(dl, nil, true)
	if err != nil {
		t.Fatalf("library call failed: %v", err)
	}
	wantOut := fmt.Sprintf("chi2=%v p=%v\n", want.Statistic, want.PValue)

	err = Dispatch(ctx, "chisq", []string{"gof", "colors"})
	if err != nil {
		t.Fatalf("chisq gof with no proportions failed: %v", err)
	}
	got := outputOf(ctx)
	if got != wantOut {
		t.Errorf("output mismatch\ngot:  %q\nwant: %q", got, wantOut)
	}
}

func TestChiSqGof_LabelContainingEquals(t *testing.T) {
	dl := insyra.NewDataList("a=b", "a=b", "c", "c", "c")
	ctx := newTestExecContext(t)
	ctx.Vars["x"] = dl

	// Label "a=b" is split at the LAST '=' so the p key is "a=b" with value 0.4
	want, err := stats.ChiSquareGoodnessOfFit(dl, map[string]float64{"a=b": 0.4, "c": 0.6}, true)
	if err != nil {
		t.Fatalf("library call failed: %v", err)
	}
	wantOut := fmt.Sprintf("chi2=%v p=%v\n", want.Statistic, want.PValue)

	err = Dispatch(ctx, "chisq", []string{"gof", "x", "a=b=0.4", "c=0.6"})
	if err != nil {
		t.Fatalf("chisq gof with label containing '=' failed: %v", err)
	}
	got := outputOf(ctx)
	if got != wantOut {
		t.Errorf("output mismatch\ngot:  %q\nwant: %q", got, wantOut)
	}
}

func TestChiSqGof_EmptyLabel(t *testing.T) {
	dl := insyra.NewDataList("", "", "c")
	ctx := newTestExecContext(t)
	ctx.Vars["x"] = dl

	want, err := stats.ChiSquareGoodnessOfFit(dl, map[string]float64{"": 0.5, "c": 0.5}, true)
	if err != nil {
		t.Fatalf("library call failed: %v", err)
	}
	wantOut := fmt.Sprintf("chi2=%v p=%v\n", want.Statistic, want.PValue)

	err = Dispatch(ctx, "chisq", []string{"gof", "x", "=0.5", "c=0.5"})
	if err != nil {
		t.Fatalf("chisq gof with empty label failed: %v", err)
	}
	got := outputOf(ctx)
	if got != wantOut {
		t.Errorf("output mismatch\ngot:  %q\nwant: %q", got, wantOut)
	}
}

func TestChiSqGof_UnknownLabelIsReported(t *testing.T) {
	dl := insyra.NewDataList("green", "red", "blue", "red", "red", "green", "blue", "red", "blue", "red")
	ctx := newTestExecContext(t)
	ctx.Vars["colors"] = dl

	// "Blue" (capital B) does not match "blue" in the data
	err := Dispatch(ctx, "chisq", []string{"gof", "colors", "red=0.5", "green=0.3", "Blue=0.2"})
	if err == nil {
		t.Fatal("chisq gof with unknown label was accepted")
	}
	if !strings.Contains(err.Error(), `p names category "Blue", which does not occur in input`) {
		t.Errorf("wrong error: %v", err)
	}
}
