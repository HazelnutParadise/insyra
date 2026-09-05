package commands

import (
	"strings"
	"testing"

	insyra "github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/quant"
)

// ===========================================================================
// quant portfolio / quant frontier (add-cli-quant-portfolio)
// ===========================================================================

// quantPortfolioReturns is a three-asset table of aligned per-period returns.
func quantPortfolioReturns() *insyra.DataTable {
	return insyra.NewDataTable(
		namedList("AAA", 0.012, -0.004, 0.007, 0.021, -0.011, 0.003, 0.009, -0.002),
		namedList("BBB", 0.005, 0.002, -0.003, 0.011, 0.001, -0.004, 0.006, -0.018),
		namedList("CCC", -0.002, 0.008, 0.004, -0.006, 0.010, 0.005, -0.001, -0.026),
	)
}

// quantPortfolioCollidingReturns names one asset after a fixed frontier column.
func quantPortfolioCollidingReturns() *insyra.DataTable {
	return insyra.NewDataTable(
		namedList("AAA", 0.012, -0.004, 0.007, 0.021, -0.011, 0.003, 0.009, -0.002),
		namedList("Variance", 0.005, 0.002, -0.003, 0.011, 0.001, -0.004, 0.006, -0.018),
		namedList("CCC", -0.002, 0.008, 0.004, -0.006, 0.010, 0.005, -0.001, -0.026),
	)
}

func quantPortfolioCtx(t *testing.T) *ExecContext {
	t.Helper()
	return newTimeSeriesContext(t, map[string]any{
		"dt":      quantPortfolioReturns(),
		"clash":   quantPortfolioCollidingReturns(),
		"notatbl": quantReturns(),
	})
}

// assertWeights compares a stored `Asset, Weight` table against a library result.
func assertWeights(t *testing.T, table *insyra.DataTable, want *quant.PortfolioResult) {
	t.Helper()
	gotCols := table.ColNames()
	if len(gotCols) != 2 || gotCols[0] != "Asset" || gotCols[1] != "Weight" {
		t.Fatalf("columns = %v want [Asset Weight]", gotCols)
	}
	if table.NumRows() != len(want.Weights) {
		t.Fatalf("rows = %d want %d", table.NumRows(), len(want.Weights))
	}
	assets := table.GetColByName("Asset").Data()
	for i, name := range want.AssetNames {
		if assets[i] != name {
			t.Errorf("Asset row %d = %v want %v", i, assets[i], name)
		}
		if got := cellFloat(t, table, "Weight", i); !closeEnough(got, want.Weights[i], quantTol) {
			t.Errorf("Weight row %d = %v want %v", i, got, want.Weights[i])
		}
	}
}

// assertStats compares a stored `<var>_stats` one-row table against a result.
func assertStats(t *testing.T, table *insyra.DataTable, want *quant.PortfolioResult) {
	t.Helper()
	if table.NumRows() != 1 {
		t.Fatalf("stats rows = %d want 1", table.NumRows())
	}
	for _, field := range []struct {
		col  string
		want float64
	}{
		{"ExpectedReturn", want.ExpectedReturn},
		{"Variance", want.Variance},
		{"Volatility", want.Volatility},
		{"SharpeRatio", want.SharpeRatio},
		{"Iterations", float64(want.Iterations)},
	} {
		if got := cellFloat(t, table, field.col, 0); !closeEnough(got, field.want, quantTol) {
			t.Errorf("%s = %v want %v", field.col, got, field.want)
		}
	}
	converged := table.GetColByName("Converged")
	if converged == nil {
		t.Fatalf("stats table has no Converged column (have %v)", table.ColNames())
	}
	if got := converged.Data()[0]; got != want.Converged {
		t.Errorf("Converged = %v want %v", got, want.Converged)
	}
}

// --- quant portfolio -------------------------------------------------------

func TestRunQuantCommand_PortfolioMinimumVariance(t *testing.T) {
	ctx := quantPortfolioCtx(t)
	if err := runQuantCommand(ctx, []string{"portfolio", "dt", "minvar", "as", "w"}); err != nil {
		t.Fatalf("quant portfolio minvar failed: %v", err)
	}
	want, err := quant.OptimizePortfolio(quantPortfolioReturns(), quant.PortfolioConfig{})
	if err != nil {
		t.Fatalf("quant.OptimizePortfolio failed: %v", err)
	}
	assertWeights(t, resultDT(t, ctx, "w"), want)
	assertStats(t, resultDT(t, ctx, "w_stats"), want)

	out := quantOutput(t, ctx)
	for _, fragment := range []string{"AAA=", "BBB=", "CCC=", "return=", "vol=", "sharpe=", "iterations=", "converged="} {
		if !strings.Contains(out, fragment) {
			t.Errorf("output %q should contain %q", out, fragment)
		}
	}
}

func TestRunQuantCommand_PortfolioTargetReturnAndBounds(t *testing.T) {
	ctx := quantPortfolioCtx(t)
	args := []string{"portfolio", "dt", "target", "0.001", "rf", "0.0001",
		"min", "0.1,0,0", "max", "0.5,0.5,0.5", "as", "w"}
	if err := runQuantCommand(ctx, args); err != nil {
		t.Fatalf("quant portfolio target failed: %v", err)
	}
	want, err := quant.OptimizePortfolio(quantPortfolioReturns(), quant.PortfolioConfig{
		Objective:    quant.TargetReturn,
		TargetReturn: 0.001,
		RiskFreeRate: 0.0001,
		MinWeight:    []float64{0.1, 0, 0},
		MaxWeight:    []float64{0.5, 0.5, 0.5},
	})
	if err != nil {
		t.Fatalf("quant.OptimizePortfolio failed: %v", err)
	}
	assertWeights(t, resultDT(t, ctx, "w"), want)
	assertStats(t, resultDT(t, ctx, "w_stats"), want)
}

func TestRunQuantCommand_PortfolioMaximumSharpe(t *testing.T) {
	ctx := quantPortfolioCtx(t)
	if err := runQuantCommand(ctx, []string{"portfolio", "dt", "maxsharpe", "rf", "0.0001", "as", "w"}); err != nil {
		t.Fatalf("quant portfolio maxsharpe failed: %v", err)
	}
	want, err := quant.OptimizePortfolio(quantPortfolioReturns(), quant.PortfolioConfig{
		Objective:    quant.MaximumSharpe,
		RiskFreeRate: 0.0001,
	})
	if err != nil {
		t.Fatalf("quant.OptimizePortfolio failed: %v", err)
	}
	assertWeights(t, resultDT(t, ctx, "w"), want)
	assertStats(t, resultDT(t, ctx, "w_stats"), want)
}

func TestRunQuantCommand_PortfolioDefaultsToResultVar(t *testing.T) {
	ctx := quantPortfolioCtx(t)
	if err := runQuantCommand(ctx, []string{"portfolio", "dt", "minvar"}); err != nil {
		t.Fatalf("quant portfolio minvar failed: %v", err)
	}
	want, err := quant.OptimizePortfolio(quantPortfolioReturns(), quant.PortfolioConfig{})
	if err != nil {
		t.Fatalf("quant.OptimizePortfolio failed: %v", err)
	}
	assertWeights(t, resultDT(t, ctx, "$result"), want)
	assertStats(t, resultDT(t, ctx, "$result_stats"), want)
}

func TestRunQuantCommand_PortfolioErrors(t *testing.T) {
	cases := []struct {
		name     string
		args     []string
		contains []string
	}{
		{
			name:     "bounds length mismatch",
			args:     []string{"portfolio", "dt", "minvar", "min", "0.1,0"},
			contains: []string{"quant portfolio:", "min", "2", "3"},
		},
		{
			name:     "non-numeric bound",
			args:     []string{"portfolio", "dt", "minvar", "max", "0.5,half,0.5"},
			contains: []string{"quant portfolio:", "max", "half"},
		},
		{
			name:     "target without a value",
			args:     []string{"portfolio", "dt", "target"},
			contains: []string{"quant portfolio:", "target"},
		},
		{
			name:     "unknown objective",
			args:     []string{"portfolio", "dt", "maxreturn"},
			contains: []string{"quant portfolio:", "maxreturn", "minvar", "maxsharpe"},
		},
		{
			name:     "unknown option",
			args:     []string{"portfolio", "dt", "minvar", "mar", "0.1"},
			contains: []string{"quant portfolio:", "mar", "rf"},
		},
		{
			name:     "not a DataTable",
			args:     []string{"portfolio", "notatbl", "minvar"},
			contains: []string{"quant portfolio:", "not a DataTable"},
		},
		{
			name:     "usage",
			args:     []string{"portfolio", "dt"},
			contains: []string{"quant portfolio <returns_dt>"},
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			ctx := quantPortfolioCtx(t)
			err := runQuantCommand(ctx, testCase.args)
			if err == nil {
				t.Fatalf("expected an error")
			}
			for _, fragment := range testCase.contains {
				if !strings.Contains(err.Error(), fragment) {
					t.Errorf("error %q should contain %q", err, fragment)
				}
			}
			if _, exists := ctx.Vars["$result"]; exists {
				t.Errorf("no variable should be stored on error")
			}
		})
	}
}

func TestRunQuantCommand_PortfolioLibraryErrorIsSurfaced(t *testing.T) {
	ctx := quantPortfolioCtx(t)
	// Upper bounds summing to 0.9 cannot reach sum(w) = 1.
	err := runQuantCommand(ctx, []string{"portfolio", "dt", "minvar", "max", "0.3,0.3,0.3"})
	if err == nil {
		t.Fatalf("expected the library to refuse infeasible bounds")
	}
	if !strings.HasPrefix(err.Error(), "quant portfolio: ") {
		t.Errorf("error %q should carry the quant portfolio: prefix", err)
	}
	_, libErr := quant.OptimizePortfolio(quantPortfolioReturns(), quant.PortfolioConfig{
		MaxWeight: []float64{0.3, 0.3, 0.3},
	})
	if libErr == nil {
		t.Fatalf("quant.OptimizePortfolio unexpectedly accepted infeasible bounds")
	}
	if !strings.Contains(err.Error(), libErr.Error()) {
		t.Errorf("error %q should carry the library message %q verbatim", err, libErr)
	}
	if _, exists := ctx.Vars["$result"]; exists {
		t.Errorf("no variable should be stored on error")
	}
}

// --- quant frontier --------------------------------------------------------

func TestRunQuantCommand_Frontier(t *testing.T) {
	ctx := quantPortfolioCtx(t)
	if err := runQuantCommand(ctx, []string{"frontier", "dt", "10", "as", "f"}); err != nil {
		t.Fatalf("quant frontier failed: %v", err)
	}
	want, err := quant.EfficientFrontier(quantPortfolioReturns(), 10, quant.PortfolioConfig{})
	if err != nil {
		t.Fatalf("quant.EfficientFrontier failed: %v", err)
	}

	table := resultDT(t, ctx, "f")
	wantCols := []string{"ExpectedReturn", "Variance", "Volatility", "SharpeRatio", "Converged", "AAA", "BBB", "CCC"}
	gotCols := table.ColNames()
	if len(gotCols) != len(wantCols) {
		t.Fatalf("columns = %v want %v", gotCols, wantCols)
	}
	for i, name := range wantCols {
		if gotCols[i] != name {
			t.Fatalf("columns = %v want %v", gotCols, wantCols)
		}
	}
	if table.NumRows() != 10 {
		t.Fatalf("rows = %d want 10", table.NumRows())
	}
	convergedCol := table.GetColByName("Converged").Data()
	for i := range want {
		for _, field := range []struct {
			col  string
			want float64
		}{
			{"ExpectedReturn", want[i].ExpectedReturn},
			{"Variance", want[i].Variance},
			{"Volatility", want[i].Volatility},
			{"SharpeRatio", want[i].SharpeRatio},
		} {
			if got := cellFloat(t, table, field.col, i); !closeEnough(got, field.want, quantTol) {
				t.Errorf("%s row %d = %v want %v", field.col, i, got, field.want)
			}
		}
		if convergedCol[i] != want[i].Converged {
			t.Errorf("Converged row %d = %v want %v", i, convergedCol[i], want[i].Converged)
		}
		for j, asset := range want[i].AssetNames {
			if got := cellFloat(t, table, asset, i); !closeEnough(got, want[i].Weights[j], quantTol) {
				t.Errorf("%s row %d = %v want %v", asset, i, got, want[i].Weights[j])
			}
		}
	}
}

func TestRunQuantCommand_FrontierPassesOptions(t *testing.T) {
	ctx := quantPortfolioCtx(t)
	args := []string{"frontier", "dt", "4", "rf", "0.0001", "min", "0.1,0,0", "max", "0.6,0.6,0.6", "as", "f"}
	if err := runQuantCommand(ctx, args); err != nil {
		t.Fatalf("quant frontier failed: %v", err)
	}
	want, err := quant.EfficientFrontier(quantPortfolioReturns(), 4, quant.PortfolioConfig{
		RiskFreeRate: 0.0001,
		MinWeight:    []float64{0.1, 0, 0},
		MaxWeight:    []float64{0.6, 0.6, 0.6},
	})
	if err != nil {
		t.Fatalf("quant.EfficientFrontier failed: %v", err)
	}
	table := resultDT(t, ctx, "f")
	if table.NumRows() != 4 {
		t.Fatalf("rows = %d want 4", table.NumRows())
	}
	for i := range want {
		if got := cellFloat(t, table, "SharpeRatio", i); !closeEnough(got, want[i].SharpeRatio, quantTol) {
			t.Errorf("SharpeRatio row %d = %v want %v", i, got, want[i].SharpeRatio)
		}
		if got := cellFloat(t, table, "AAA", i); !closeEnough(got, want[i].Weights[0], quantTol) {
			t.Errorf("AAA row %d = %v want %v", i, got, want[i].Weights[0])
		}
	}
}

func TestRunQuantCommand_FrontierErrors(t *testing.T) {
	cases := []struct {
		name     string
		args     []string
		contains []string
	}{
		{
			name:     "points below two",
			args:     []string{"frontier", "dt", "1"},
			contains: []string{"quant frontier:", "points", "2"},
		},
		{
			name:     "points not an integer",
			args:     []string{"frontier", "dt", "ten"},
			contains: []string{"quant frontier:", "points", "ten"},
		},
		{
			name:     "asset name collides with a fixed column",
			args:     []string{"frontier", "clash", "5"},
			contains: []string{"quant frontier:", "Variance"},
		},
		{
			name:     "bounds length mismatch",
			args:     []string{"frontier", "dt", "5", "min", "0.1,0"},
			contains: []string{"quant frontier:", "min", "3"},
		},
		{
			name:     "not a DataTable",
			args:     []string{"frontier", "notatbl", "5"},
			contains: []string{"quant frontier:", "not a DataTable"},
		},
		{
			name:     "usage",
			args:     []string{"frontier", "dt"},
			contains: []string{"quant frontier <returns_dt> <points>"},
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			ctx := quantPortfolioCtx(t)
			err := runQuantCommand(ctx, testCase.args)
			if err == nil {
				t.Fatalf("expected an error")
			}
			for _, fragment := range testCase.contains {
				if !strings.Contains(err.Error(), fragment) {
					t.Errorf("error %q should contain %q", err, fragment)
				}
			}
			if _, exists := ctx.Vars["$result"]; exists {
				t.Errorf("no variable should be stored on error")
			}
		})
	}
}
