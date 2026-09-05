package commands

import (
	"bytes"
	"math"
	"strconv"
	"strings"
	"testing"

	insyra "github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/quant"
)

// ===========================================================================
// quant command (add-cli-quant-commands)
// ===========================================================================

const quantTol = 1e-12

func quantReturns() *insyra.DataList {
	return insyra.NewDataList(0.012, -0.004, 0.007, 0.021, -0.011, 0.003, 0.009, -0.002)
}

func quantBenchmark() *insyra.DataList {
	return insyra.NewDataList(0.010, -0.002, 0.005, 0.018, -0.009, 0.001, 0.007, -0.003)
}

func quantEquity() *insyra.DataList {
	return insyra.NewDataList(100.0, 102.0, 99.0, 105.0, 103.0, 110.0, 108.0, 115.0)
}

func quantFactorTable() *insyra.DataTable {
	return insyra.NewDataTable(
		namedList("MKT", 0.010, -0.002, 0.005, 0.018, -0.009, 0.001, 0.007, -0.003),
		namedList("SMB", 0.003, 0.001, -0.002, 0.004, 0.002, -0.001, 0.005, 0.000),
		namedList("HML", -0.001, 0.002, 0.003, -0.004, 0.001, 0.002, -0.003, 0.004),
	)
}

// quantCtx builds the shared ExecContext for the quant tests.
func quantCtx(t *testing.T) *ExecContext {
	t.Helper()
	return newTimeSeriesContext(t, map[string]any{
		"r":     quantReturns(),
		"b":     quantBenchmark(),
		"eq":    quantEquity(),
		"a":     quantReturns(),
		"m":     quantBenchmark(),
		"f":     quantFactorTable(),
		"notdl": quantFactorTable(),
		"notdt": quantReturns(),
		"bad":   insyra.NewDataList(0.01, nil, 0.02, -0.01),
	})
}

func quantOutput(t *testing.T, ctx *ExecContext) string {
	t.Helper()
	buf, ok := ctx.Output.(*bytes.Buffer)
	if !ok {
		t.Fatalf("expected a *bytes.Buffer output, got %T", ctx.Output)
	}
	return buf.String()
}

// resultFloat retrieves a stored scalar result.
func resultFloat(t *testing.T, ctx *ExecContext, name string) float64 {
	t.Helper()
	value, exists := ctx.Vars[name]
	if !exists {
		t.Fatalf("expected variable %q to be set", name)
	}
	number, ok := value.(float64)
	if !ok {
		t.Fatalf("expected %q to be float64, got %T", name, value)
	}
	return number
}

// cellFloat reads one numeric cell out of a stored result table.
func cellFloat(t *testing.T, dt *insyra.DataTable, col string, row int) float64 {
	t.Helper()
	column := dt.GetColByName(col)
	if column == nil {
		t.Fatalf("column %q missing (have %v)", col, dt.ColNames())
	}
	data := column.Data()
	if row >= len(data) {
		t.Fatalf("column %q has %d rows, wanted row %d", col, len(data), row)
	}
	number, ok := insyra.ToFloat64Safe(data[row])
	if !ok {
		t.Fatalf("column %q row %d is not numeric: %v", col, row, data[row])
	}
	return number
}

func closeEnough(got, want, tol float64) bool {
	if math.IsNaN(got) && math.IsNaN(want) {
		return true
	}
	return math.Abs(got-want) <= tol
}

// --- scalar forms ----------------------------------------------------------

func TestRunQuantCommand_Sharpe(t *testing.T) {
	ctx := quantCtx(t)
	if err := runQuantCommand(ctx, []string{"sharpe", "r", "252", "rf", "0.0001", "as", "s"}); err != nil {
		t.Fatalf("quant sharpe failed: %v", err)
	}
	want, err := quant.SharpeRatio(quantReturns(), 0.0001, 252)
	if err != nil {
		t.Fatalf("quant.SharpeRatio failed: %v", err)
	}
	if got := resultFloat(t, ctx, "s"); !closeEnough(got, want, quantTol) {
		t.Errorf("s = %v want %v", got, want)
	}
	if out := quantOutput(t, ctx); !strings.Contains(out, "sharpe=") {
		t.Errorf("output %q should contain sharpe=", out)
	}
}

func TestRunQuantCommand_SharpeDefaultRiskFreeRateIsZero(t *testing.T) {
	ctx := quantCtx(t)
	if err := runQuantCommand(ctx, []string{"sharpe", "r", "252", "as", "s"}); err != nil {
		t.Fatalf("quant sharpe failed: %v", err)
	}
	want, err := quant.SharpeRatio(quantReturns(), 0, 252)
	if err != nil {
		t.Fatalf("quant.SharpeRatio failed: %v", err)
	}
	if got := resultFloat(t, ctx, "s"); !closeEnough(got, want, quantTol) {
		t.Errorf("s = %v want %v", got, want)
	}
}

func TestRunQuantCommand_Sortino(t *testing.T) {
	ctx := quantCtx(t)
	if err := runQuantCommand(ctx, []string{"sortino", "r", "12", "mar", "0.001", "as", "v"}); err != nil {
		t.Fatalf("quant sortino failed: %v", err)
	}
	want, err := quant.SortinoRatio(quantReturns(), 0.001, 12)
	if err != nil {
		t.Fatalf("quant.SortinoRatio failed: %v", err)
	}
	if got := resultFloat(t, ctx, "v"); !closeEnough(got, want, quantTol) {
		t.Errorf("v = %v want %v", got, want)
	}
	if out := quantOutput(t, ctx); !strings.Contains(out, "sortino=") {
		t.Errorf("output %q should contain sortino=", out)
	}
}

func TestRunQuantCommand_InformationRatio(t *testing.T) {
	ctx := quantCtx(t)
	if err := runQuantCommand(ctx, []string{"ir", "r", "b", "252", "as", "v"}); err != nil {
		t.Fatalf("quant ir failed: %v", err)
	}
	want, err := quant.InformationRatio(quantReturns(), quantBenchmark(), 252)
	if err != nil {
		t.Fatalf("quant.InformationRatio failed: %v", err)
	}
	if got := resultFloat(t, ctx, "v"); !closeEnough(got, want, quantTol) {
		t.Errorf("v = %v want %v", got, want)
	}
	if out := quantOutput(t, ctx); !strings.Contains(out, "ir=") {
		t.Errorf("output %q should contain ir=", out)
	}
}

func TestRunQuantCommand_MaxDrawdown(t *testing.T) {
	ctx := quantCtx(t)
	if err := runQuantCommand(ctx, []string{"maxdd", "eq", "as", "v"}); err != nil {
		t.Fatalf("quant maxdd failed: %v", err)
	}
	want, err := quant.MaxDrawdown(quantEquity())
	if err != nil {
		t.Fatalf("quant.MaxDrawdown failed: %v", err)
	}
	if got := resultFloat(t, ctx, "v"); !closeEnough(got, want, quantTol) {
		t.Errorf("v = %v want %v", got, want)
	}
	if out := quantOutput(t, ctx); !strings.Contains(out, "maxdd=") {
		t.Errorf("output %q should contain maxdd=", out)
	}
}

func TestRunQuantCommand_AnnualizedReturn(t *testing.T) {
	ctx := quantCtx(t)
	if err := runQuantCommand(ctx, []string{"annret", "eq", "365", "as", "v"}); err != nil {
		t.Fatalf("quant annret failed: %v", err)
	}
	want, err := quant.AnnualizedReturn(quantEquity(), 365)
	if err != nil {
		t.Fatalf("quant.AnnualizedReturn failed: %v", err)
	}
	if got := resultFloat(t, ctx, "v"); !closeEnough(got, want, quantTol) {
		t.Errorf("v = %v want %v", got, want)
	}
	if out := quantOutput(t, ctx); !strings.Contains(out, "annret=") {
		t.Errorf("output %q should contain annret=", out)
	}
}

func TestRunQuantCommand_Calmar(t *testing.T) {
	ctx := quantCtx(t)
	if err := runQuantCommand(ctx, []string{"calmar", "eq", "365", "as", "v"}); err != nil {
		t.Fatalf("quant calmar failed: %v", err)
	}
	want, err := quant.CalmarRatio(quantEquity(), 365)
	if err != nil {
		t.Fatalf("quant.CalmarRatio failed: %v", err)
	}
	if got := resultFloat(t, ctx, "v"); !closeEnough(got, want, quantTol) {
		t.Errorf("v = %v want %v", got, want)
	}
	if out := quantOutput(t, ctx); !strings.Contains(out, "calmar=") {
		t.Errorf("output %q should contain calmar=", out)
	}
}

func TestRunQuantCommand_Beta(t *testing.T) {
	ctx := quantCtx(t)
	if err := runQuantCommand(ctx, []string{"beta", "a", "m", "as", "v"}); err != nil {
		t.Fatalf("quant beta failed: %v", err)
	}
	want, err := quant.Beta(quantReturns(), quantBenchmark())
	if err != nil {
		t.Fatalf("quant.Beta failed: %v", err)
	}
	if got := resultFloat(t, ctx, "v"); !closeEnough(got, want, quantTol) {
		t.Errorf("v = %v want %v", got, want)
	}
	if out := quantOutput(t, ctx); !strings.Contains(out, "beta=") {
		t.Errorf("output %q should contain beta=", out)
	}
}

// --- VaR / CVaR ------------------------------------------------------------

func TestRunQuantCommand_VaRMethodKeyword(t *testing.T) {
	historical, err := quant.ValueAtRisk(quantReturns(), 0.95, quant.VaRHistorical)
	if err != nil {
		t.Fatalf("quant.ValueAtRisk failed: %v", err)
	}
	parametric, err := quant.ValueAtRisk(quantReturns(), 0.95, quant.VaRParametric)
	if err != nil {
		t.Fatalf("quant.ValueAtRisk failed: %v", err)
	}
	if closeEnough(historical, parametric, 1e-9) {
		t.Fatalf("fixture cannot tell the two methods apart (%v vs %v)", historical, parametric)
	}

	cases := []struct {
		name string
		args []string
		want float64
	}{
		{"explicit parametric", []string{"var", "r", "0.95", "parametric", "as", "v"}, parametric},
		{"explicit historical", []string{"var", "r", "0.95", "historical", "as", "v"}, historical},
		{"default is historical", []string{"var", "r", "0.95", "as", "v"}, historical},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			ctx := quantCtx(t)
			if err := runQuantCommand(ctx, testCase.args); err != nil {
				t.Fatalf("quant var failed: %v", err)
			}
			if got := resultFloat(t, ctx, "v"); !closeEnough(got, testCase.want, quantTol) {
				t.Errorf("v = %v want %v", got, testCase.want)
			}
			if out := quantOutput(t, ctx); !strings.Contains(out, "var=") {
				t.Errorf("output %q should contain var=", out)
			}
		})
	}
}

func TestRunQuantCommand_CVaR(t *testing.T) {
	for _, testCase := range []struct {
		keyword string
		method  quant.VaRMethod
	}{
		{"historical", quant.VaRHistorical},
		{"parametric", quant.VaRParametric},
	} {
		t.Run(testCase.keyword, func(t *testing.T) {
			ctx := quantCtx(t)
			if err := runQuantCommand(ctx, []string{"cvar", "r", "0.9", testCase.keyword, "as", "v"}); err != nil {
				t.Fatalf("quant cvar failed: %v", err)
			}
			want, err := quant.ConditionalValueAtRisk(quantReturns(), 0.9, testCase.method)
			if err != nil {
				t.Fatalf("quant.ConditionalValueAtRisk failed: %v", err)
			}
			if got := resultFloat(t, ctx, "v"); !closeEnough(got, want, quantTol) {
				t.Errorf("v = %v want %v", got, want)
			}
			if out := quantOutput(t, ctx); !strings.Contains(out, "cvar=") {
				t.Errorf("output %q should contain cvar=", out)
			}
		})
	}
}

func TestRunQuantCommand_UnknownVaRMethod(t *testing.T) {
	ctx := quantCtx(t)
	err := runQuantCommand(ctx, []string{"var", "r", "0.95", "montecarlo"})
	if err == nil {
		t.Fatalf("expected an error for an unknown VaR method")
	}
	message := err.Error()
	if !strings.HasPrefix(message, "quant var: ") {
		t.Errorf("error %q should carry the quant var: prefix", message)
	}
	for _, method := range []string{"historical", "parametric"} {
		if !strings.Contains(message, method) {
			t.Errorf("error %q should list %q", message, method)
		}
	}
}

// --- DataList result -------------------------------------------------------

func TestRunQuantCommand_DrawdownSeries(t *testing.T) {
	ctx := quantCtx(t)
	if err := runQuantCommand(ctx, []string{"drawdown", "eq", "as", "dd"}); err != nil {
		t.Fatalf("quant drawdown failed: %v", err)
	}
	want, err := quant.DrawdownSeries(quantEquity())
	if err != nil {
		t.Fatalf("quant.DrawdownSeries failed: %v", err)
	}
	got := resultDL(t, ctx, "dd")
	if !approxEqualAny(got.Data(), want.Data(), quantTol) {
		t.Errorf("drawdown = %v want %v", got.Data(), want.Data())
	}
}

// --- table results ---------------------------------------------------------

func TestRunQuantCommand_CAPMTable(t *testing.T) {
	ctx := quantCtx(t)
	if err := runQuantCommand(ctx, []string{"capm", "a", "m", "rf", "0.0002", "as", "c"}); err != nil {
		t.Fatalf("quant capm failed: %v", err)
	}
	want, err := quant.CAPM(quantReturns(), quantBenchmark(), 0.0002)
	if err != nil {
		t.Fatalf("quant.CAPM failed: %v", err)
	}
	table := resultDT(t, ctx, "c")
	if table.NumRows() != 1 {
		t.Fatalf("rows = %d want 1", table.NumRows())
	}
	for _, field := range []struct {
		col  string
		want float64
	}{
		{"beta", want.Beta},
		{"alpha", want.Alpha},
		{"r2", want.RSquared},
		{"beta_se", want.BetaStdErr},
		{"alpha_se", want.AlphaStdErr},
		{"n", float64(want.N)},
	} {
		if got := cellFloat(t, table, field.col, 0); !closeEnough(got, field.want, quantTol) {
			t.Errorf("%s = %v want %v", field.col, got, field.want)
		}
	}
	out := quantOutput(t, ctx)
	for _, name := range []string{"beta=", "alpha=", "r2=", "beta_se=", "alpha_se=", "n="} {
		if !strings.Contains(out, name) {
			t.Errorf("output %q should contain %q", out, name)
		}
	}
}

func TestRunQuantCommand_FactorTable(t *testing.T) {
	ctx := quantCtx(t)
	if err := runQuantCommand(ctx, []string{"factor", "a", "f", "as", "fm"}); err != nil {
		t.Fatalf("quant factor failed: %v", err)
	}
	want, err := quant.FactorModel(quantReturns(), quantFactorTable(), 0)
	if err != nil {
		t.Fatalf("quant.FactorModel failed: %v", err)
	}

	table := resultDT(t, ctx, "fm")
	wantCols := []string{"Factor", "Exposure", "StdErr", "TValue", "PValue"}
	gotCols := table.ColNames()
	if len(gotCols) != len(wantCols) {
		t.Fatalf("columns = %v want %v", gotCols, wantCols)
	}
	for i, name := range wantCols {
		if gotCols[i] != name {
			t.Fatalf("columns = %v want %v", gotCols, wantCols)
		}
	}
	if table.NumRows() != len(want.FactorNames) {
		t.Fatalf("rows = %d want %d", table.NumRows(), len(want.FactorNames))
	}
	factorCol := table.GetColByName("Factor").Data()
	for i, name := range want.FactorNames {
		if factorCol[i] != name {
			t.Errorf("Factor row %d = %v want %v", i, factorCol[i], name)
		}
		if got := cellFloat(t, table, "Exposure", i); !closeEnough(got, want.Exposures[i], quantTol) {
			t.Errorf("Exposure row %d = %v want %v", i, got, want.Exposures[i])
		}
		if got := cellFloat(t, table, "StdErr", i); !closeEnough(got, want.StdErrs[i], quantTol) {
			t.Errorf("StdErr row %d = %v want %v", i, got, want.StdErrs[i])
		}
		if got := cellFloat(t, table, "TValue", i); !closeEnough(got, want.TValues[i], quantTol) {
			t.Errorf("TValue row %d = %v want %v", i, got, want.TValues[i])
		}
		if got := cellFloat(t, table, "PValue", i); !closeEnough(got, want.PValues[i], quantTol) {
			t.Errorf("PValue row %d = %v want %v", i, got, want.PValues[i])
		}
	}

	alpha := resultDT(t, ctx, "fm_alpha")
	if alpha.NumRows() != 1 {
		t.Fatalf("fm_alpha rows = %d want 1", alpha.NumRows())
	}
	if name := alpha.GetColByName("Factor").Data()[0]; name != "alpha" {
		t.Errorf("fm_alpha Factor = %v want alpha", name)
	}
	for _, field := range []struct {
		col  string
		want float64
	}{
		{"Exposure", want.Alpha},
		{"StdErr", want.AlphaStdErr},
		{"TValue", want.AlphaTValue},
		{"PValue", want.AlphaPValue},
	} {
		if got := cellFloat(t, alpha, field.col, 0); !closeEnough(got, field.want, quantTol) {
			t.Errorf("fm_alpha %s = %v want %v", field.col, got, field.want)
		}
	}

	out := quantOutput(t, ctx)
	if !strings.Contains(out, "alpha=") {
		t.Errorf("output %q should contain alpha=", out)
	}
	for _, name := range want.FactorNames {
		if !strings.Contains(out, name+" exposure=") {
			t.Errorf("output %q should carry a line for factor %q", out, name)
		}
	}
	if !strings.Contains(out, "t=") || !strings.Contains(out, "p=") {
		t.Errorf("output %q should carry t= and p= per factor", out)
	}
}

func TestRunQuantCommand_FactorRiskFreeRate(t *testing.T) {
	ctx := quantCtx(t)
	if err := runQuantCommand(ctx, []string{"factor", "a", "f", "rf", "0.0002", "as", "fm"}); err != nil {
		t.Fatalf("quant factor failed: %v", err)
	}
	want, err := quant.FactorModel(quantReturns(), quantFactorTable(), 0.0002)
	if err != nil {
		t.Fatalf("quant.FactorModel failed: %v", err)
	}
	alpha := resultDT(t, ctx, "fm_alpha")
	if got := cellFloat(t, alpha, "Exposure", 0); !closeEnough(got, want.Alpha, quantTol) {
		t.Errorf("alpha = %v want %v", got, want.Alpha)
	}
}

// --- options ---------------------------------------------------------------

// TestRunQuantCommand_BlackScholesHullExample uses Hull's worked European call
// (S=42, K=40, r=10%, sigma=20%, T=0.5), whose price is 4.759.
func TestRunQuantCommand_BlackScholesHullExample(t *testing.T) {
	ctx := quantCtx(t)
	if err := runQuantCommand(ctx, []string{"bs", "call", "42", "40", "0.10", "0.20", "0.5", "as", "o"}); err != nil {
		t.Fatalf("quant bs failed: %v", err)
	}
	want, err := quant.BlackScholes(quant.BSInput{
		Spot: 42, Strike: 40, Rate: 0.10, Volatility: 0.20, TimeToExpiry: 0.5, Type: quant.OptionCall,
	})
	if err != nil {
		t.Fatalf("quant.BlackScholes failed: %v", err)
	}
	if math.Abs(want.Price-4.759) > 5e-4 {
		t.Fatalf("library price %v is not Hull's 4.759", want.Price)
	}
	table := resultDT(t, ctx, "o")
	if table.NumRows() != 1 {
		t.Fatalf("rows = %d want 1", table.NumRows())
	}
	for _, field := range []struct {
		col  string
		want float64
	}{
		{"price", want.Price},
		{"delta", want.Delta},
		{"gamma", want.Gamma},
		{"vega", want.Vega},
		{"theta", want.Theta},
		{"rho", want.Rho},
	} {
		if got := cellFloat(t, table, field.col, 0); !closeEnough(got, field.want, quantTol) {
			t.Errorf("%s = %v want %v", field.col, got, field.want)
		}
	}
	out := quantOutput(t, ctx)
	if !strings.Contains(out, "price=4.759") {
		t.Errorf("output %q should print price=4.759...", out)
	}
	for _, name := range []string{"delta=", "gamma=", "vega=", "theta=", "rho="} {
		if !strings.Contains(out, name) {
			t.Errorf("output %q should contain %q", out, name)
		}
	}
}

func TestRunQuantCommand_BlackScholesPutWithDividendYield(t *testing.T) {
	ctx := quantCtx(t)
	if err := runQuantCommand(ctx, []string{"bs", "put", "42", "40", "0.10", "0.20", "0.5", "q", "0.03", "as", "o"}); err != nil {
		t.Fatalf("quant bs failed: %v", err)
	}
	want, err := quant.BlackScholes(quant.BSInput{
		Spot: 42, Strike: 40, Rate: 0.10, DividendYield: 0.03,
		Volatility: 0.20, TimeToExpiry: 0.5, Type: quant.OptionPut,
	})
	if err != nil {
		t.Fatalf("quant.BlackScholes failed: %v", err)
	}
	if got := cellFloat(t, resultDT(t, ctx, "o"), "price", 0); !closeEnough(got, want.Price, quantTol) {
		t.Errorf("price = %v want %v", got, want.Price)
	}
}

func TestRunQuantCommand_ImpliedVolatilityRoundTrip(t *testing.T) {
	priced, err := quant.BlackScholes(quant.BSInput{
		Spot: 42, Strike: 40, Rate: 0.10, Volatility: 0.20, TimeToExpiry: 0.5, Type: quant.OptionCall,
	})
	if err != nil {
		t.Fatalf("quant.BlackScholes failed: %v", err)
	}
	price := strconv64(priced.Price)

	ctx := quantCtx(t)
	if err := runQuantCommand(ctx, []string{"iv", "call", price, "42", "40", "0.10", "0.5", "as", "v"}); err != nil {
		t.Fatalf("quant iv failed: %v", err)
	}
	if got := resultFloat(t, ctx, "v"); math.Abs(got-0.20) > 1e-8 {
		t.Errorf("implied volatility = %v want 0.20 within 1e-8", got)
	}
	if out := quantOutput(t, ctx); !strings.Contains(out, "iv=") {
		t.Errorf("output %q should contain iv=", out)
	}
}

// --- storage defaults ------------------------------------------------------

func TestRunQuantCommand_DefaultsToResultVar(t *testing.T) {
	ctx := quantCtx(t)
	if err := runQuantCommand(ctx, []string{"sharpe", "r", "252"}); err != nil {
		t.Fatalf("quant sharpe failed: %v", err)
	}
	want, err := quant.SharpeRatio(quantReturns(), 0, 252)
	if err != nil {
		t.Fatalf("quant.SharpeRatio failed: %v", err)
	}
	if got := resultFloat(t, ctx, "$result"); !closeEnough(got, want, quantTol) {
		t.Errorf("$result = %v want %v", got, want)
	}

	ctx = quantCtx(t)
	if err := runQuantCommand(ctx, []string{"capm", "a", "m"}); err != nil {
		t.Fatalf("quant capm failed: %v", err)
	}
	if resultDT(t, ctx, "$result").NumRows() != 1 {
		t.Errorf("capm without `as` should store a one-row table in $result")
	}
}

// --- errors ----------------------------------------------------------------

func TestRunQuantCommand_UnknownFormListsEveryForm(t *testing.T) {
	ctx := quantCtx(t)
	err := runQuantCommand(ctx, []string{"omega", "r"})
	if err == nil {
		t.Fatalf("expected an error for an unknown form")
	}
	message := err.Error()
	for _, form := range []string{
		"sharpe", "sortino", "ir", "maxdd", "annret", "calmar", "drawdown",
		"var", "cvar", "beta", "capm", "factor", "bs", "iv",
	} {
		if !strings.Contains(message, form) {
			t.Errorf("error %q should list the form %q", message, form)
		}
	}
}

func TestRunQuantCommand_NoArgs(t *testing.T) {
	ctx := quantCtx(t)
	err := runQuantCommand(ctx, nil)
	if err == nil || !strings.Contains(err.Error(), "usage: quant") {
		t.Fatalf("expected a usage error, got %v", err)
	}
}

func TestRunQuantCommand_MissingRequiredPositional(t *testing.T) {
	cases := []struct {
		name      string
		args      []string
		wantUsage string
	}{
		{"sharpe", []string{"sharpe", "r"}, "quant sharpe <returns> <periods>"},
		{"sortino", []string{"sortino", "r"}, "quant sortino <returns> <periods>"},
		{"ir", []string{"ir", "r", "b"}, "quant ir <returns> <benchmark> <periods>"},
		{"maxdd", []string{"maxdd"}, "quant maxdd <equity>"},
		{"annret", []string{"annret", "eq"}, "quant annret <equity> <days>"},
		{"calmar", []string{"calmar", "eq"}, "quant calmar <equity> <days>"},
		{"drawdown", []string{"drawdown"}, "quant drawdown <equity>"},
		{"var", []string{"var", "r"}, "quant var <returns> <confidence>"},
		{"cvar", []string{"cvar", "r"}, "quant cvar <returns> <confidence>"},
		{"beta", []string{"beta", "a"}, "quant beta <asset> <market>"},
		{"capm", []string{"capm", "a"}, "quant capm <asset> <market>"},
		{"factor", []string{"factor", "a"}, "quant factor <asset> <factors>"},
		{"bs", []string{"bs", "call", "42", "40", "0.10", "0.20"}, "quant bs call|put <spot> <strike> <rate> <vol> <years>"},
		{"iv", []string{"iv", "call", "4.76", "42", "40", "0.10"}, "quant iv call|put <price> <spot> <strike> <rate> <years>"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			ctx := quantCtx(t)
			err := runQuantCommand(ctx, testCase.args)
			if err == nil {
				t.Fatalf("expected a usage error")
			}
			if !strings.Contains(err.Error(), testCase.wantUsage) {
				t.Errorf("error %q should show the usage %q", err, testCase.wantUsage)
			}
			if _, exists := ctx.Vars["$result"]; exists {
				t.Errorf("no variable should be stored on error")
			}
		})
	}
}

func TestRunQuantCommand_NotADataList(t *testing.T) {
	ctx := quantCtx(t)
	err := runQuantCommand(ctx, []string{"sharpe", "notdl", "252"})
	if err == nil || !strings.HasPrefix(err.Error(), "quant sharpe: ") {
		t.Fatalf("expected a quant sharpe: prefixed error, got %v", err)
	}
	if !strings.Contains(err.Error(), "not a DataList") {
		t.Errorf("error %q should say the variable is not a DataList", err)
	}
}

func TestRunQuantCommand_NotADataTable(t *testing.T) {
	ctx := quantCtx(t)
	err := runQuantCommand(ctx, []string{"factor", "a", "notdt"})
	if err == nil || !strings.HasPrefix(err.Error(), "quant factor: ") {
		t.Fatalf("expected a quant factor: prefixed error, got %v", err)
	}
	if !strings.Contains(err.Error(), "not a DataTable") {
		t.Errorf("error %q should say the variable is not a DataTable", err)
	}
}

func TestRunQuantCommand_VariableNotFound(t *testing.T) {
	ctx := quantCtx(t)
	err := runQuantCommand(ctx, []string{"maxdd", "missing"})
	if err == nil || !strings.HasPrefix(err.Error(), "quant maxdd: ") {
		t.Fatalf("expected a quant maxdd: prefixed error, got %v", err)
	}
	if !strings.Contains(err.Error(), "variable not found") {
		t.Errorf("error %q should say the variable was not found", err)
	}
}

func TestRunQuantCommand_LibraryErrorIsSurfaced(t *testing.T) {
	ctx := quantCtx(t)
	err := runQuantCommand(ctx, []string{"sortino", "bad", "252"})
	if err == nil {
		t.Fatalf("expected the library to refuse a nil observation")
	}
	message := err.Error()
	if !strings.HasPrefix(message, "quant sortino: ") {
		t.Errorf("error %q should carry the quant sortino: prefix", message)
	}
	want, libErr := quant.SortinoRatio(insyra.NewDataList(0.01, nil, 0.02, -0.01), 0, 252)
	if libErr == nil {
		t.Fatalf("quant.SortinoRatio unexpectedly succeeded with %v", want)
	}
	if !strings.Contains(message, libErr.Error()) {
		t.Errorf("error %q should carry the library message %q verbatim", message, libErr)
	}
	if !strings.Contains(message, "row 2") {
		t.Errorf("error %q should keep the library's row number", message)
	}
	if _, exists := ctx.Vars["$result"]; exists {
		t.Errorf("no variable should be stored on error")
	}
}

func TestRunQuantCommand_UnknownOption(t *testing.T) {
	ctx := quantCtx(t)
	err := runQuantCommand(ctx, []string{"sharpe", "r", "252", "mar", "0.1"})
	if err == nil {
		t.Fatalf("expected an error for an option the form does not take")
	}
	message := err.Error()
	if !strings.HasPrefix(message, "quant sharpe: ") {
		t.Errorf("error %q should carry the quant sharpe: prefix", message)
	}
	if !strings.Contains(message, "rf") {
		t.Errorf("error %q should list the supported option rf", message)
	}
}

func TestRunQuantCommand_OptionRequiresValue(t *testing.T) {
	ctx := quantCtx(t)
	err := runQuantCommand(ctx, []string{"sharpe", "r", "252", "rf"})
	if err == nil || !strings.Contains(err.Error(), "requires a value") {
		t.Fatalf("expected a missing-value error, got %v", err)
	}
}

func TestRunQuantCommand_InvalidNumericArgument(t *testing.T) {
	ctx := quantCtx(t)
	err := runQuantCommand(ctx, []string{"sharpe", "r", "yearly"})
	if err == nil || !strings.HasPrefix(err.Error(), "quant sharpe: ") {
		t.Fatalf("expected a quant sharpe: prefixed error, got %v", err)
	}
	if !strings.Contains(err.Error(), "periods") {
		t.Errorf("error %q should name the offending argument", err)
	}
}

func TestRunQuantCommand_UnknownOptionType(t *testing.T) {
	ctx := quantCtx(t)
	err := runQuantCommand(ctx, []string{"bs", "straddle", "42", "40", "0.10", "0.20", "0.5"})
	if err == nil {
		t.Fatalf("expected an error for an unknown option type")
	}
	if !strings.Contains(err.Error(), "call") || !strings.Contains(err.Error(), "put") {
		t.Errorf("error %q should list call and put", err)
	}
}

// --- registration ----------------------------------------------------------

func TestQuantCommandIsRegisteredWithEveryForm(t *testing.T) {
	handler, ok := Registry["quant"]
	if !ok {
		t.Fatalf("quant is not registered")
	}
	if len(handler.Forms) != len(quantForms) {
		t.Fatalf("Forms lists %d entries, want %d", len(handler.Forms), len(quantForms))
	}
	for _, form := range quantForms {
		found := false
		for _, line := range handler.Forms {
			if strings.HasPrefix(line, form.usage) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("help quant does not show the form %q", form.name)
		}
	}
	if len(handler.Examples) == 0 {
		t.Errorf("quant should carry Examples")
	}
}

// strconv64 renders a float64 the way a user would type it on the command
// line, at full precision.
func strconv64(value float64) string {
	return strconv.FormatFloat(value, 'g', -1, 64)
}
