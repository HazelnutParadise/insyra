package commands

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	insyra "github.com/HazelnutParadise/insyra"
)

func init() {
	_ = Register(&CommandHandler{
		Name:        "fillna",
		Args:        OpenArgs(),
		Usage:       "fillna <var> mean|median|mode|ffill|bfill|interpolate [cols A,B,C] [limit N] [extrapolate yes|no] [missing nan|nil|both] [as <var>]",
		Description: "Fill missing DataList/DataTable values",
		Forms: []string{
			"mean|median|interpolate  apply to numeric data only",
			"mode|ffill|bfill         work with any data type",
			"cols <A,B,C>             DataTable only; comma-separated column names or indices",
			"limit <n>                cap consecutive ffill/bfill replacements (0 = unlimited)",
			"extrapolate yes|no       interpolation only; fill leading/trailing gaps (default no)",
			"missing nan|nil|both     which kind of missing to fill (default both)",
		},
		Examples: []string{
			"insyra fillna price median as price_filled",
			"insyra fillna price ffill limit 2 missing nan as price_ffill",
			"insyra fillna price interpolate extrapolate yes as price_interp",
			"insyra fillna sales median cols revenue,cost as cleaned",
		},
		Run: runFillNACommand,
	})
	_ = Register(&CommandHandler{
		Name:        "fillnan",
		Args:        MaxArgs(2).WithAlias(),
		Usage:       "fillnan <var> mean [as <var>]",
		Description: "Fill NaN with mean (deprecated alias; prefer 'fillna ... missing nan')",
		Run:         runFillNaNCommand,
	})
}

// runFillNaNCommand handles the legacy `fillnan <var> mean [as <var>]` shape.
// It only fills NaN (not nil) and only supports the mean strategy. Use `fillna`
// for the full surface.
func runFillNaNCommand(ctx *ExecContext, args []string) error {
	coreArgs, alias := parseAlias(args)
	if len(coreArgs) < 2 || strings.ToLower(coreArgs[1]) != "mean" {
		return fmt.Errorf("usage: fillnan <var> mean [as <var>] (use 'fillna' for other strategies)")
	}
	dl, err := getDataListVar(ctx, coreArgs[0])
	if err != nil {
		return err
	}
	if len(coreArgs) > 2 {
		return fmt.Errorf("fillnan: unexpected extra args %v (use 'fillna' for options)", coreArgs[2:])
	}
	// The deprecated `fillnan` command intentionally keeps NaN-only mean fill.
	result := dl.Clone().FillNaNWithMean() // nolint:staticcheck
	ctx.Vars[alias] = result
	_, _ = fmt.Fprintf(ctx.Output, "warning: 'fillnan' is deprecated; use 'fillna %s mean missing nan%s' instead\n", coreArgs[0], aliasSuffix(alias))
	_, _ = fmt.Fprintf(ctx.Output, "saved as %s\n", alias)
	return nil
}

func aliasSuffix(alias string) string {
	if alias == "" || alias == "$result" {
		return ""
	}
	return " as " + alias
}

type fillNAOptions struct {
	Cols        []string
	Limit       int
	Extrapolate bool
	Missing     string // "nan", "nil", or "both" (default)
}

func runFillNACommand(ctx *ExecContext, args []string) error {
	coreArgs, alias := parseAlias(args)
	if len(coreArgs) < 2 {
		return fmt.Errorf("usage: fillna <var> mean|median|mode|ffill|bfill|interpolate [cols A,B,C] [limit N] [extrapolate yes|no] [missing nan|nil|both] [as <var>]")
	}
	strategy := strings.ToLower(coreArgs[1])
	opts, err := parseFillNAOptions(coreArgs[2:])
	if err != nil {
		return err
	}

	if dt, err := getDataTableVar(ctx, coreArgs[0]); err == nil {
		result, err := applyFillNAToTable(dt, strategy, opts)
		if err != nil {
			return err
		}
		ctx.Vars[alias] = result
		_, _ = fmt.Fprintf(ctx.Output, "saved as %s\n", alias)
		return nil
	}
	if dl, err := getDataListVar(ctx, coreArgs[0]); err == nil {
		if len(opts.Cols) > 0 {
			return fmt.Errorf("fillna: 'cols' only applies to DataTable variables")
		}
		result, err := applyFillNAToList(dl, strategy, opts)
		if err != nil {
			return err
		}
		ctx.Vars[alias] = result
		_, _ = fmt.Fprintf(ctx.Output, "saved as %s\n", alias)
		return nil
	}
	return varTypeError(ctx, "fillna", coreArgs[0])
}

// fillSkippingOtherKind runs ffill or bfill (missing nan or nil) on one kind of
// missing cell alone. Cells of the other kind are taken out, the rest is
// filled, and they are put back where they were, so they are not filled, give
// no value to copy and do not count toward limit. Filling them and restoring
// them afterwards, as the other strategies do, would still let them use up
// limit.
func fillSkippingOtherKind(dl *insyra.DataList, strategy string, opts fillNAOptions) (*insyra.DataList, error) {
	skip := snapshotPreservedDL(dl, opts.Missing)
	data := dl.Data()

	kept := make([]any, 0, len(data)-len(skip))
	next := 0
	for i, v := range data {
		if next < len(skip) && skip[next] == i {
			next++
			continue
		}
		kept = append(kept, v)
	}
	// Append rather than NewDataList(kept...), which would flatten a slice cell.
	compact := insyra.NewDataList().Append(kept...)
	switch strategy {
	case "ffill":
		compact.FillForward(opts.Limit)
	case "bfill":
		compact.FillBackward(opts.Limit)
	}
	if err := compact.PopErr(); err != nil {
		return nil, fmt.Errorf("fillna: %w", err)
	}

	filled := compact.Data()
	rebuilt := make([]any, len(data))
	next = 0
	taken := 0
	for i, v := range data {
		if next < len(skip) && skip[next] == i {
			next++
			rebuilt[i] = v
			continue
		}
		rebuilt[i] = filled[taken]
		taken++
	}
	return insyra.NewDataList().Append(rebuilt...).SetName(dl.GetName()), nil
}

func applyFillNAToList(dl *insyra.DataList, strategy string, opts fillNAOptions) (*insyra.DataList, error) {
	if (strategy == "ffill" || strategy == "bfill") && (opts.Missing == "nan" || opts.Missing == "nil") {
		return fillSkippingOtherKind(dl.Clone(), strategy, opts)
	}
	result := dl.Clone()
	preserved := snapshotPreservedDL(result, opts.Missing)
	if err := runListStrategy(result, strategy, opts); err != nil {
		return nil, err
	}
	// A fill that could not do what it was asked records why on the list;
	// saving the result anyway would hand back a variable that looks filled.
	if err := result.PopErr(); err != nil {
		return nil, fmt.Errorf("fillna: %w", err)
	}
	restorePreservedDL(result, opts.Missing, preserved)
	return result, nil
}

func applyFillNAToTable(dt *insyra.DataTable, strategy string, opts fillNAOptions) (*insyra.DataTable, error) {
	result := dt.Clone()
	positions, err := resolveColumnTokens("fillna", result, opts.Cols)
	if err != nil {
		return nil, err
	}
	if (strategy == "ffill" || strategy == "bfill") && (opts.Missing == "nan" || opts.Missing == "nil") {
		targets := positions
		if len(targets) == 0 {
			for i := range result.NumCols() {
				targets = append(targets, i)
			}
		}
		for _, pos := range targets {
			// GetColByNumber returns a copy, so the filled column is written back.
			filled, err := fillSkippingOtherKind(result.GetColByNumber(pos), strategy, opts)
			if err != nil {
				return nil, err
			}
			result.UpdateCol(pos, filled)
		}
		if err := checkTableErr("fillna", result); err != nil {
			return nil, err
		}
		return result, nil
	}
	preserved := snapshotPreservedDT(result, positions, opts.Missing)
	if err := runTableStrategy(result, strategy, opts, positions); err != nil {
		return nil, err
	}
	if err := checkTableErr("fillna", result); err != nil {
		return nil, err
	}
	restorePreservedDT(result, opts.Missing, preserved)
	return result, nil
}

func runListStrategy(dl *insyra.DataList, strategy string, opts fillNAOptions) error {
	switch strategy {
	case "mean":
		dl.FillWithMean()
	case "median":
		dl.FillWithMedian()
	case "mode":
		dl.FillWithMode()
	case "ffill":
		dl.FillForward(opts.Limit)
	case "bfill":
		dl.FillBackward(opts.Limit)
	case "interpolate":
		dl.FillByInterpolation(opts.Extrapolate)
	default:
		return fmt.Errorf("fillna: unknown strategy %q (supported: mean, median, mode, ffill, bfill, interpolate)", strategy)
	}
	return nil
}

func runTableStrategy(dt *insyra.DataTable, strategy string, opts fillNAOptions, positions []int) error {
	cols := make([]any, len(positions))
	for i, pos := range positions {
		cols[i] = selectorAt(dt, pos)
	}
	switch strategy {
	case "mean":
		dt.FillWithMean(cols...)
	case "median":
		dt.FillWithMedian(cols...)
	case "mode":
		dt.FillWithMode(cols...)
	case "ffill":
		dt.FillForward(opts.Limit, cols...)
	case "bfill":
		dt.FillBackward(opts.Limit, cols...)
	case "interpolate":
		dt.FillByInterpolation(opts.Extrapolate, cols...)
	default:
		return fmt.Errorf("fillna: unknown strategy %q (supported: mean, median, mode, ffill, bfill, interpolate)", strategy)
	}
	return nil
}

// snapshotPreservedDL records positions whose original value must be restored
// after the fill runs. With missing=="nan", nil positions are preserved (so the
// fill only sticks at NaN positions). With missing=="nil", NaN positions are
// preserved. With "both" (or unset), nothing is preserved.
func snapshotPreservedDL(dl *insyra.DataList, missing string) []int {
	if missing != "nan" && missing != "nil" {
		return nil
	}
	var positions []int
	for i, v := range dl.Data() {
		if missing == "nan" && v == nil {
			positions = append(positions, i)
		} else if missing == "nil" {
			if f, ok := v.(float64); ok && math.IsNaN(f) {
				positions = append(positions, i)
			}
		}
	}
	return positions
}

func restorePreservedDL(dl *insyra.DataList, missing string, positions []int) {
	if len(positions) == 0 {
		return
	}
	var restoreVal any
	if missing == "nan" {
		restoreVal = nil
	} else {
		restoreVal = math.NaN()
	}
	for _, i := range positions {
		dl.Update(i, restoreVal)
	}
}

// snapshotPreservedDT records, per column position, the cells a fill must
// leave alone. Columns are addressed by position: a name can be empty or
// shared, and looking one up by name records an error on the table when it
// misses.
func snapshotPreservedDT(dt *insyra.DataTable, cols []int, missing string) map[int][]int {
	if missing != "nan" && missing != "nil" {
		return nil
	}
	targets := cols
	if len(targets) == 0 {
		for i := range dt.NumCols() {
			targets = append(targets, i)
		}
	}
	preserved := map[int][]int{}
	for _, col := range targets {
		dl := dt.GetColByNumber(col)
		if dl == nil {
			continue
		}
		positions := snapshotPreservedDL(dl, missing)
		if len(positions) > 0 {
			preserved[col] = positions
		}
	}
	return preserved
}

func restorePreservedDT(dt *insyra.DataTable, missing string, preserved map[int][]int) {
	for col, positions := range preserved {
		dl := dt.GetColByNumber(col)
		if dl == nil {
			continue
		}
		// GetColByNumber returns a CLONE, so mutating it in place is lost.
		// Restore on the clone, then write it back into the live table.
		restorePreservedDL(dl, missing, positions)
		dt.UpdateCol(col, dl)
	}
}

func parseFillNAOptions(args []string) (fillNAOptions, error) {
	opts := fillNAOptions{Missing: "both"}
	for i := 0; i < len(args); {
		key := strings.ToLower(args[i])
		next := func() (string, error) {
			if i+1 >= len(args) {
				return "", fmt.Errorf("fillna: option %q requires a value", args[i])
			}
			return args[i+1], nil
		}
		switch key {
		case "cols":
			v, err := next()
			if err != nil {
				return opts, err
			}
			opts.Cols = parseCSVTokens(v)
			i += 2
		case "limit":
			v, err := next()
			if err != nil {
				return opts, err
			}
			n, err := strconv.Atoi(v)
			if err != nil || n < 0 {
				return opts, fmt.Errorf("fillna: invalid limit %q", v)
			}
			opts.Limit = n
			i += 2
		case "extrapolate":
			v, err := next()
			if err != nil {
				return opts, err
			}
			b, err := parseFlexBool(v)
			if err != nil {
				return opts, fmt.Errorf("fillna: invalid value for extrapolate: %w", err)
			}
			opts.Extrapolate = b
			i += 2
		case "missing":
			v, err := next()
			if err != nil {
				return opts, err
			}
			switch strings.ToLower(v) {
			case "nan", "nil", "both":
				opts.Missing = strings.ToLower(v)
			default:
				return opts, fmt.Errorf("fillna: invalid value for missing %q (supported: nan, nil, both)", v)
			}
			i += 2
		default:
			return opts, fmt.Errorf("fillna: unknown option %q (supported: cols, limit, extrapolate, missing)", args[i])
		}
	}
	return opts, nil
}
