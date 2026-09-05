package commands

import (
	"fmt"
	"strconv"
	"strings"

	insyra "github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/quant"
)

// quantForm describes one `quant <form>` shape. The table below is the single
// source for dispatch, for the `Forms:` block that `help quant` prints, and
// for the usage line an argument error carries, so the three cannot drift.
type quantForm struct {
	name  string
	usage string
	desc  string
	run   func(ctx *ExecContext, form *quantForm, args []string, alias string) error
}

// quantOption is one `key value` option; the parsed number is written to
// target. Every quant option is a rate, so they are all float64.
type quantOption struct {
	key    string
	target *float64
}

var quantForms = []quantForm{
	{
		name:  "sharpe",
		usage: "quant sharpe <returns> <periods> [rf <r>] [as <var>]",
		desc:  "SharpeRatio: annualized Sharpe; rf is per period (default 0)",
		run:   runQuantSharpe,
	},
	{
		name:  "sortino",
		usage: "quant sortino <returns> <periods> [mar <r>] [as <var>]",
		desc:  "SortinoRatio: downside-deviation Sharpe; mar is per period (default 0)",
		run:   runQuantSortino,
	},
	{
		name:  "ir",
		usage: "quant ir <returns> <benchmark> <periods> [as <var>]",
		desc:  "InformationRatio: annualized active return over tracking error",
		run:   runQuantInformationRatio,
	},
	{
		name:  "maxdd",
		usage: "quant maxdd <equity> [as <var>]",
		desc:  "MaxDrawdown: worst fall below a prior peak, as a fraction",
		run:   runQuantMaxDrawdown,
	},
	{
		name:  "annret",
		usage: "quant annret <equity> <days> [as <var>]",
		desc:  "AnnualizedReturn: CAGR over a calendar-day span",
		run:   runQuantAnnualizedReturn,
	},
	{
		name:  "calmar",
		usage: "quant calmar <equity> <days> [as <var>]",
		desc:  "CalmarRatio: annualized return over maximum drawdown",
		run:   runQuantCalmar,
	},
	{
		name:  "drawdown",
		usage: "quant drawdown <equity> [as <var>]",
		desc:  "DrawdownSeries: per-period drawdown, stored as a DataList",
		run:   runQuantDrawdownSeries,
	},
	{
		name:  "var",
		usage: "quant var <returns> <confidence> [historical|parametric] [as <var>]",
		desc:  "ValueAtRisk: tail loss as a positive fraction (default historical)",
		run:   runQuantValueAtRisk,
	},
	{
		name:  "cvar",
		usage: "quant cvar <returns> <confidence> [historical|parametric] [as <var>]",
		desc:  "ConditionalValueAtRisk: mean loss inside the tail (default historical)",
		run:   runQuantConditionalValueAtRisk,
	},
	{
		name:  "beta",
		usage: "quant beta <asset> <market> [as <var>]",
		desc:  "Beta: market exposure of aligned per-period returns",
		run:   runQuantBeta,
	},
	{
		name:  "capm",
		usage: "quant capm <asset> <market> [rf <r>] [as <var>]",
		desc:  "CAPM: beta, alpha, R2, standard errors and N as a one-row DataTable",
		run:   runQuantCAPM,
	},
	{
		name:  "factor",
		usage: "quant factor <asset> <factors> [rf <r>] [as <var>]",
		desc:  "FactorModel: one DataTable row per factor, alpha in <var>_alpha",
		run:   runQuantFactorModel,
	},
	{
		name:  "bs",
		usage: "quant bs call|put <spot> <strike> <rate> <vol> <years> [q <yield>] [as <var>]",
		desc:  "BlackScholes: price and greeks as a one-row DataTable",
		run:   runQuantBlackScholes,
	},
	{
		name:  "iv",
		usage: "quant iv call|put <price> <spot> <strike> <rate> <years> [q <yield>] [as <var>]",
		desc:  "ImpliedVolatility: the volatility that reproduces the given price",
		run:   runQuantImpliedVolatility,
	},
}

func init() {
	_ = Register(&CommandHandler{
		Name:        "quant",
		Usage:       "quant " + quantFormNames("|") + " ...",
		Description: "Quantitative finance: performance, risk, exposure, factor and option analytics",
		Forms:       quantFormLines(),
		Examples: []string{
			"insyra quant sharpe returns 252 rf 0.0001 as sharpe",
			"insyra quant sortino returns 252 mar 0.0002",
			"insyra quant ir returns benchmark 252",
			"insyra quant maxdd equity",
			"insyra quant calmar equity 365",
			"insyra quant drawdown equity as dd",
			"insyra quant var returns 0.95 parametric as var95",
			"insyra quant cvar returns 0.95 as cvar95",
			"insyra quant beta asset market",
			"insyra quant capm asset market rf 0.0002 as capm",
			"insyra quant factor asset factors rf 0.0002 as fm",
			"insyra quant bs call 42 40 0.10 0.20 0.5 as opt",
			"insyra quant iv call 4.759 42 40 0.10 0.5 as vol",
		},
		Run: runQuantCommand,
	})
}

func runQuantCommand(ctx *ExecContext, args []string) error {
	coreArgs, alias := parseAlias(args)
	if len(coreArgs) < 1 {
		return fmt.Errorf("usage: quant <form> <args> (forms: %s)", quantFormNames(", "))
	}
	name := strings.ToLower(coreArgs[0])
	for i := range quantForms {
		form := &quantForms[i]
		if form.name == name {
			return form.run(ctx, form, coreArgs[1:], alias)
		}
	}
	return fmt.Errorf("quant: unknown form %q (forms: %s)", coreArgs[0], quantFormNames(", "))
}

// ---------------------------------------------------------------------------
// forms
// ---------------------------------------------------------------------------

func runQuantSharpe(ctx *ExecContext, form *quantForm, args []string, alias string) error {
	if len(args) < 2 {
		return quantUsageError(form)
	}
	returns, err := quantList(ctx, form, args[0])
	if err != nil {
		return err
	}
	periods, err := quantFloat(form, "periods", args[1])
	if err != nil {
		return err
	}
	riskFreeRate := 0.0
	if err := parseQuantOptions(form, args[2:], quantOption{key: "rf", target: &riskFreeRate}); err != nil {
		return err
	}
	value, err := quant.SharpeRatio(returns, riskFreeRate, periods)
	if err != nil {
		return quantLibraryError(form, err)
	}
	quantStoreScalar(ctx, alias, "sharpe", value)
	return nil
}

func runQuantSortino(ctx *ExecContext, form *quantForm, args []string, alias string) error {
	if len(args) < 2 {
		return quantUsageError(form)
	}
	returns, err := quantList(ctx, form, args[0])
	if err != nil {
		return err
	}
	periods, err := quantFloat(form, "periods", args[1])
	if err != nil {
		return err
	}
	minimumAcceptableReturn := 0.0
	if err := parseQuantOptions(form, args[2:], quantOption{key: "mar", target: &minimumAcceptableReturn}); err != nil {
		return err
	}
	value, err := quant.SortinoRatio(returns, minimumAcceptableReturn, periods)
	if err != nil {
		return quantLibraryError(form, err)
	}
	quantStoreScalar(ctx, alias, "sortino", value)
	return nil
}

func runQuantInformationRatio(ctx *ExecContext, form *quantForm, args []string, alias string) error {
	if len(args) < 3 {
		return quantUsageError(form)
	}
	returns, err := quantList(ctx, form, args[0])
	if err != nil {
		return err
	}
	benchmark, err := quantList(ctx, form, args[1])
	if err != nil {
		return err
	}
	periods, err := quantFloat(form, "periods", args[2])
	if err != nil {
		return err
	}
	if err := parseQuantOptions(form, args[3:]); err != nil {
		return err
	}
	value, err := quant.InformationRatio(returns, benchmark, periods)
	if err != nil {
		return quantLibraryError(form, err)
	}
	quantStoreScalar(ctx, alias, "ir", value)
	return nil
}

func runQuantMaxDrawdown(ctx *ExecContext, form *quantForm, args []string, alias string) error {
	if len(args) < 1 {
		return quantUsageError(form)
	}
	equity, err := quantList(ctx, form, args[0])
	if err != nil {
		return err
	}
	if err := parseQuantOptions(form, args[1:]); err != nil {
		return err
	}
	value, err := quant.MaxDrawdown(equity)
	if err != nil {
		return quantLibraryError(form, err)
	}
	quantStoreScalar(ctx, alias, "maxdd", value)
	return nil
}

func runQuantAnnualizedReturn(ctx *ExecContext, form *quantForm, args []string, alias string) error {
	if len(args) < 2 {
		return quantUsageError(form)
	}
	equity, err := quantList(ctx, form, args[0])
	if err != nil {
		return err
	}
	days, err := quantInt(form, "days", args[1])
	if err != nil {
		return err
	}
	if err := parseQuantOptions(form, args[2:]); err != nil {
		return err
	}
	value, err := quant.AnnualizedReturn(equity, days)
	if err != nil {
		return quantLibraryError(form, err)
	}
	quantStoreScalar(ctx, alias, "annret", value)
	return nil
}

func runQuantCalmar(ctx *ExecContext, form *quantForm, args []string, alias string) error {
	if len(args) < 2 {
		return quantUsageError(form)
	}
	equity, err := quantList(ctx, form, args[0])
	if err != nil {
		return err
	}
	days, err := quantInt(form, "days", args[1])
	if err != nil {
		return err
	}
	if err := parseQuantOptions(form, args[2:]); err != nil {
		return err
	}
	value, err := quant.CalmarRatio(equity, days)
	if err != nil {
		return quantLibraryError(form, err)
	}
	quantStoreScalar(ctx, alias, "calmar", value)
	return nil
}

func runQuantDrawdownSeries(ctx *ExecContext, form *quantForm, args []string, alias string) error {
	if len(args) < 1 {
		return quantUsageError(form)
	}
	equity, err := quantList(ctx, form, args[0])
	if err != nil {
		return err
	}
	if err := parseQuantOptions(form, args[1:]); err != nil {
		return err
	}
	series, err := quant.DrawdownSeries(equity)
	if err != nil {
		return quantLibraryError(form, err)
	}
	ctx.Vars[alias] = series
	_, _ = fmt.Fprintf(ctx.Output, "saved as %s\n", alias)
	return nil
}

func runQuantValueAtRisk(ctx *ExecContext, form *quantForm, args []string, alias string) error {
	returns, confidence, method, err := parseQuantVaRArgs(ctx, form, args)
	if err != nil {
		return err
	}
	value, err := quant.ValueAtRisk(returns, confidence, method)
	if err != nil {
		return quantLibraryError(form, err)
	}
	quantStoreScalar(ctx, alias, "var", value)
	return nil
}

func runQuantConditionalValueAtRisk(ctx *ExecContext, form *quantForm, args []string, alias string) error {
	returns, confidence, method, err := parseQuantVaRArgs(ctx, form, args)
	if err != nil {
		return err
	}
	value, err := quant.ConditionalValueAtRisk(returns, confidence, method)
	if err != nil {
		return quantLibraryError(form, err)
	}
	quantStoreScalar(ctx, alias, "cvar", value)
	return nil
}

func runQuantBeta(ctx *ExecContext, form *quantForm, args []string, alias string) error {
	if len(args) < 2 {
		return quantUsageError(form)
	}
	asset, err := quantList(ctx, form, args[0])
	if err != nil {
		return err
	}
	market, err := quantList(ctx, form, args[1])
	if err != nil {
		return err
	}
	if err := parseQuantOptions(form, args[2:]); err != nil {
		return err
	}
	value, err := quant.Beta(asset, market)
	if err != nil {
		return quantLibraryError(form, err)
	}
	quantStoreScalar(ctx, alias, "beta", value)
	return nil
}

func runQuantCAPM(ctx *ExecContext, form *quantForm, args []string, alias string) error {
	if len(args) < 2 {
		return quantUsageError(form)
	}
	asset, err := quantList(ctx, form, args[0])
	if err != nil {
		return err
	}
	market, err := quantList(ctx, form, args[1])
	if err != nil {
		return err
	}
	riskFreeRate := 0.0
	if err := parseQuantOptions(form, args[2:], quantOption{key: "rf", target: &riskFreeRate}); err != nil {
		return err
	}
	result, err := quant.CAPM(asset, market, riskFreeRate)
	if err != nil {
		return quantLibraryError(form, err)
	}
	ctx.Vars[alias] = quantOneRowTable(
		quantColumn("beta", result.Beta),
		quantColumn("alpha", result.Alpha),
		quantColumn("r2", result.RSquared),
		quantColumn("beta_se", result.BetaStdErr),
		quantColumn("alpha_se", result.AlphaStdErr),
		quantColumn("n", result.N),
	)
	_, _ = fmt.Fprintf(ctx.Output, "beta=%v alpha=%v r2=%v beta_se=%v alpha_se=%v n=%d\n",
		result.Beta, result.Alpha, result.RSquared, result.BetaStdErr, result.AlphaStdErr, result.N)
	return nil
}

func runQuantFactorModel(ctx *ExecContext, form *quantForm, args []string, alias string) error {
	if len(args) < 2 {
		return quantUsageError(form)
	}
	asset, err := quantList(ctx, form, args[0])
	if err != nil {
		return err
	}
	factors, err := quantTable(ctx, form, args[1])
	if err != nil {
		return err
	}
	riskFreeRate := 0.0
	if err := parseQuantOptions(form, args[2:], quantOption{key: "rf", target: &riskFreeRate}); err != nil {
		return err
	}
	result, err := quant.FactorModel(asset, factors, riskFreeRate)
	if err != nil {
		return quantLibraryError(form, err)
	}

	names := make([]any, len(result.FactorNames))
	exposures := make([]any, len(result.FactorNames))
	standardErrors := make([]any, len(result.FactorNames))
	tValues := make([]any, len(result.FactorNames))
	pValues := make([]any, len(result.FactorNames))
	for i, name := range result.FactorNames {
		names[i] = name
		exposures[i] = result.Exposures[i]
		standardErrors[i] = result.StdErrs[i]
		tValues[i] = result.TValues[i]
		pValues[i] = result.PValues[i]
	}
	ctx.Vars[alias] = insyra.NewDataTable(
		quantColumn("Factor", names...),
		quantColumn("Exposure", exposures...),
		quantColumn("StdErr", standardErrors...),
		quantColumn("TValue", tValues...),
		quantColumn("PValue", pValues...),
	)
	ctx.Vars[alias+"_alpha"] = insyra.NewDataTable(
		quantColumn("Factor", "alpha"),
		quantColumn("Exposure", result.Alpha),
		quantColumn("StdErr", result.AlphaStdErr),
		quantColumn("TValue", result.AlphaTValue),
		quantColumn("PValue", result.AlphaPValue),
	)

	_, _ = fmt.Fprintf(ctx.Output, "alpha=%v t=%v p=%v\n", result.Alpha, result.AlphaTValue, result.AlphaPValue)
	for i, name := range result.FactorNames {
		_, _ = fmt.Fprintf(ctx.Output, "%s exposure=%v t=%v p=%v\n", name, result.Exposures[i], result.TValues[i], result.PValues[i])
	}
	return nil
}

func runQuantBlackScholes(ctx *ExecContext, form *quantForm, args []string, alias string) error {
	if len(args) < 6 {
		return quantUsageError(form)
	}
	optionType, err := parseQuantOptionType(form, args[0])
	if err != nil {
		return err
	}
	input := quant.BSInput{Type: optionType}
	for _, field := range []struct {
		label  string
		raw    string
		target *float64
	}{
		{"spot", args[1], &input.Spot},
		{"strike", args[2], &input.Strike},
		{"rate", args[3], &input.Rate},
		{"vol", args[4], &input.Volatility},
		{"years", args[5], &input.TimeToExpiry},
	} {
		value, err := quantFloat(form, field.label, field.raw)
		if err != nil {
			return err
		}
		*field.target = value
	}
	if err := parseQuantOptions(form, args[6:], quantOption{key: "q", target: &input.DividendYield}); err != nil {
		return err
	}

	result, err := quant.BlackScholes(input)
	if err != nil {
		return quantLibraryError(form, err)
	}
	ctx.Vars[alias] = quantOneRowTable(
		quantColumn("price", result.Price),
		quantColumn("delta", result.Delta),
		quantColumn("gamma", result.Gamma),
		quantColumn("vega", result.Vega),
		quantColumn("theta", result.Theta),
		quantColumn("rho", result.Rho),
	)
	_, _ = fmt.Fprintf(ctx.Output, "price=%v delta=%v gamma=%v vega=%v theta=%v rho=%v\n",
		result.Price, result.Delta, result.Gamma, result.Vega, result.Theta, result.Rho)
	return nil
}

func runQuantImpliedVolatility(ctx *ExecContext, form *quantForm, args []string, alias string) error {
	if len(args) < 6 {
		return quantUsageError(form)
	}
	optionType, err := parseQuantOptionType(form, args[0])
	if err != nil {
		return err
	}
	var price float64
	input := quant.BSInput{Type: optionType}
	for _, field := range []struct {
		label  string
		raw    string
		target *float64
	}{
		{"price", args[1], &price},
		{"spot", args[2], &input.Spot},
		{"strike", args[3], &input.Strike},
		{"rate", args[4], &input.Rate},
		{"years", args[5], &input.TimeToExpiry},
	} {
		value, err := quantFloat(form, field.label, field.raw)
		if err != nil {
			return err
		}
		*field.target = value
	}
	if err := parseQuantOptions(form, args[6:], quantOption{key: "q", target: &input.DividendYield}); err != nil {
		return err
	}

	value, err := quant.ImpliedVolatility(price, input)
	if err != nil {
		return quantLibraryError(form, err)
	}
	quantStoreScalar(ctx, alias, "iv", value)
	return nil
}

// ---------------------------------------------------------------------------
// shared parsing and output
// ---------------------------------------------------------------------------

// parseQuantVaRArgs reads the shape `<returns> <confidence> [method]` shared
// by `quant var` and `quant cvar`.
func parseQuantVaRArgs(ctx *ExecContext, form *quantForm, args []string) (*insyra.DataList, float64, quant.VaRMethod, error) {
	if len(args) < 2 {
		return nil, 0, quant.VaRHistorical, quantUsageError(form)
	}
	returns, err := quantList(ctx, form, args[0])
	if err != nil {
		return nil, 0, quant.VaRHistorical, err
	}
	confidence, err := quantFloat(form, "confidence", args[1])
	if err != nil {
		return nil, 0, quant.VaRHistorical, err
	}
	method := quant.VaRHistorical
	if len(args) >= 3 {
		method, err = parseQuantVaRMethod(form, args[2])
		if err != nil {
			return nil, 0, quant.VaRHistorical, err
		}
	}
	if len(args) > 3 {
		return nil, 0, quant.VaRHistorical, fmt.Errorf("quant %s: unexpected argument %q (usage: %s)", form.name, args[3], form.usage)
	}
	return returns, confidence, method, nil
}

func parseQuantVaRMethod(form *quantForm, raw string) (quant.VaRMethod, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "historical":
		return quant.VaRHistorical, nil
	case "parametric":
		return quant.VaRParametric, nil
	}
	return quant.VaRHistorical, fmt.Errorf("quant %s: unknown method %q (supported: historical, parametric)", form.name, raw)
}

func parseQuantOptionType(form *quantForm, raw string) (quant.OptionType, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "call":
		return quant.OptionCall, nil
	case "put":
		return quant.OptionPut, nil
	}
	return quant.OptionCall, fmt.Errorf("quant %s: unknown option type %q (supported: call, put)", form.name, raw)
}

// parseQuantOptions consumes the trailing `key value` pairs a form accepts.
// A form with no options rejects every remaining token, so a stray argument
// is reported instead of ignored.
func parseQuantOptions(form *quantForm, args []string, options ...quantOption) error {
	for i := 0; i < len(args); {
		matched := false
		for _, option := range options {
			if !strings.EqualFold(args[i], option.key) {
				continue
			}
			if i+1 >= len(args) {
				return fmt.Errorf("quant %s: option %q requires a value", form.name, args[i])
			}
			value, err := quantFloat(form, option.key, args[i+1])
			if err != nil {
				return err
			}
			*option.target = value
			matched = true
			i += 2
			break
		}
		if !matched {
			return fmt.Errorf("quant %s: unknown option %q (supported: %s)", form.name, args[i], quantOptionKeys(options))
		}
	}
	return nil
}

func quantOptionKeys(options []quantOption) string {
	if len(options) == 0 {
		return "none"
	}
	keys := make([]string, 0, len(options))
	for _, option := range options {
		keys = append(keys, option.key)
	}
	return strings.Join(keys, ", ")
}

func quantList(ctx *ExecContext, form *quantForm, name string) (*insyra.DataList, error) {
	dl, err := getDataListVar(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("quant %s: %w", form.name, err)
	}
	return dl, nil
}

func quantTable(ctx *ExecContext, form *quantForm, name string) (*insyra.DataTable, error) {
	dt, err := getDataTableVar(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("quant %s: %w", form.name, err)
	}
	return dt, nil
}

func quantFloat(form *quantForm, label, raw string) (float64, error) {
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, fmt.Errorf("quant %s: invalid %s %q", form.name, label, raw)
	}
	return value, nil
}

func quantInt(form *quantForm, label, raw string) (int, error) {
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("quant %s: invalid %s %q", form.name, label, raw)
	}
	return value, nil
}

func quantUsageError(form *quantForm) error {
	return fmt.Errorf("usage: %s", form.usage)
}

// quantLibraryError wraps a quant package error, keeping its text verbatim
// behind a `quant <form>:` prefix.
func quantLibraryError(form *quantForm, err error) error {
	return fmt.Errorf("quant %s: %w", form.name, err)
}

func quantStoreScalar(ctx *ExecContext, alias, name string, value float64) {
	ctx.Vars[alias] = value
	_, _ = fmt.Fprintf(ctx.Output, "%s=%v\n", name, value)
}

// quantColumn builds a named result column.
func quantColumn(name string, values ...any) *insyra.DataList {
	column := insyra.NewDataList(values...)
	column.SetName(name)
	return column
}

func quantOneRowTable(columns ...*insyra.DataList) *insyra.DataTable {
	return insyra.NewDataTable(columns...)
}

func quantFormNames(separator string) string {
	names := make([]string, 0, len(quantForms))
	for i := range quantForms {
		names = append(names, quantForms[i].name)
	}
	return strings.Join(names, separator)
}

func quantFormLines() []string {
	lines := make([]string, 0, len(quantForms))
	for i := range quantForms {
		lines = append(lines, fmt.Sprintf("%-79s %s", quantForms[i].usage, quantForms[i].desc))
	}
	return lines
}
