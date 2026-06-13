package commands

import (
	"bytes"
	"strings"
	"testing"

	insyra "github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats"
)

func TestRunRegressionCommand_Logistic(t *testing.T) {
	ctx := newTestExecContext(t)
	ctx.Vars["y"] = insyra.NewDataList(0, 0, 1, 0, 1, 1, 0, 1, 1, 0)
	ctx.Vars["x"] = insyra.NewDataList(-2.1, -1.5, -0.4, 0.1, 0.6, 1.2, 1.7, 2.1, 2.8, 3.3)

	if err := runRegressionCommand(ctx, []string{"logistic", "y", "x", "as", "fit"}); err != nil {
		t.Fatalf("logistic regression failed: %v", err)
	}

	if _, ok := ctx.Vars["fit"].(*stats.LogisticRegressionResult); !ok {
		t.Fatalf("expected fit to be *stats.LogisticRegressionResult, got %T", ctx.Vars["fit"])
	}
	got := ctx.Output.(*bytes.Buffer).String()
	for _, want := range []string{
		"logistic regression stored in fit",
		"McFaddenR2=",
		"AIC=",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected output to contain %q, got %q", want, got)
		}
	}
}

func TestRunRegressionCommand_Poisson(t *testing.T) {
	ctx := newTestExecContext(t)
	ctx.Vars["y"] = insyra.NewDataList(1, 2, 3, 4, 6, 8, 9, 12)
	ctx.Vars["x"] = insyra.NewDataList(0.1, 0.4, 0.8, 1.2, 1.7, 2.0, 2.4, 2.9)

	if err := runRegressionCommand(ctx, []string{"poisson", "y", "x", "as", "fit"}); err != nil {
		t.Fatalf("poisson regression failed: %v", err)
	}

	if _, ok := ctx.Vars["fit"].(*stats.PoissonRegressionResult); !ok {
		t.Fatalf("expected fit to be *stats.PoissonRegressionResult, got %T", ctx.Vars["fit"])
	}
	got := ctx.Output.(*bytes.Buffer).String()
	for _, want := range []string{
		"poisson regression stored in fit",
		"AIC=",
		"dispersion=",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected output to contain %q, got %q", want, got)
		}
	}
}
