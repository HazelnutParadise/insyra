package commands

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	insyra "github.com/HazelnutParadise/insyra"
)

func init() {
	_ = Register(&CommandHandler{
		Name:        "scale",
		Usage:       "scale fit std|minmax|robust|maxabs <scalerVar> <tableVar> [range <min> <max>] cols <c1,c2,...> | scale transform|inverse <scalerVar> <tableVar> as <outVar>",
		Description: "Fit a reusable feature scaler and transform tables with it",
		Forms: []string{
			"scale fit std <scalerVar> <tableVar> cols <c1,c2,...>",
			"scale fit minmax <scalerVar> <tableVar> range <min> <max> cols <c1,c2,...>",
			"scale fit robust <scalerVar> <tableVar> cols <c1,c2,...>",
			"scale fit maxabs <scalerVar> <tableVar> cols <c1,c2,...>",
			"",
			"scale transform <scalerVar> <tableVar> as <outVar>",
			"scale inverse <scalerVar> <tableVar> as <outVar>",
		},
		Examples: []string{
			"insyra split t train 0.8 as train test",
			"insyra scale fit std sc train cols Age,Income",
			"insyra scale transform sc train as train_scaled",
			"insyra scale transform sc test as test_scaled",
			"insyra scale inverse sc pred as pred_original",
		},
		Run: runScaleCommand,
	})
}

func runScaleCommand(ctx *ExecContext, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: scale fit|transform|inverse ...")
	}
	switch strings.ToLower(args[0]) {
	case "fit":
		return runScaleFit(ctx, args[1:])
	case "transform":
		return runScaleApply(ctx, args[1:], false)
	case "inverse":
		return runScaleApply(ctx, args[1:], true)
	default:
		return fmt.Errorf("scale: unknown subcommand %q (supported: fit, transform, inverse)", args[0])
	}
}

func runScaleFit(ctx *ExecContext, args []string) error {
	if len(args) < 3 {
		return fmt.Errorf("usage: scale fit <method> <scalerVar> <tableVar> [range <min> <max>] cols <c1,c2,...>")
	}
	method := strings.ToLower(args[0])
	scalerVar := args[1]
	tableVar := args[2]

	cols, featureMin, featureMax, hasRange, err := parseScaleFitOptions(args[3:])
	if err != nil {
		return err
	}
	if len(cols) == 0 {
		return fmt.Errorf("scale fit: cols is required (e.g. cols Age,Income)")
	}

	table, err := getDataTableVar(ctx, tableVar)
	if err != nil {
		return err
	}

	var scaler insyra.Scaler
	switch method {
	case "std", "standard":
		scaler = insyra.NewStandardScaler()
	case "minmax":
		if !hasRange {
			featureMin, featureMax = 0, 1
		}
		scaler = insyra.NewMinMaxScaler(featureMin, featureMax)
	case "robust":
		scaler = insyra.NewRobustScaler()
	case "maxabs":
		scaler = insyra.NewMaxAbsScaler()
	default:
		return fmt.Errorf("scale fit: unknown method %q (supported: std, minmax, robust, maxabs)", method)
	}
	if hasRange && method != "minmax" {
		return fmt.Errorf("scale fit: range is only valid for minmax")
	}

	if err := scaler.Fit(table, cols...); err != nil {
		return err
	}
	ctx.Vars[scalerVar] = scaler
	_, _ = fmt.Fprintf(ctx.Output, "fitted %s scaler %s on %s (cols: %s)\n", scaler.Kind(), scalerVar, tableVar, strings.Join(scaledColumnNames(scaler), ", "))
	return nil
}

func runScaleApply(ctx *ExecContext, args []string, inverse bool) error {
	coreArgs, alias := parseAlias(args)
	if len(coreArgs) != 2 {
		op := "transform"
		if inverse {
			op = "inverse"
		}
		return fmt.Errorf("usage: scale %s <scalerVar> <tableVar> as <outVar>", op)
	}
	scaler, err := getScalerVar(ctx, coreArgs[0])
	if err != nil {
		return err
	}
	table, err := getDataTableVar(ctx, coreArgs[1])
	if err != nil {
		return err
	}

	var result *insyra.DataTable
	if inverse {
		result, err = scaler.InverseTransform(table)
	} else {
		result, err = scaler.Transform(table)
	}
	if err != nil {
		return err
	}
	ctx.Vars[alias] = result
	verb := "scaled"
	if inverse {
		verb = "inverse-scaled"
	}
	_, _ = fmt.Fprintf(ctx.Output, "%s %s into %s\n", verb, coreArgs[1], alias)
	return nil
}

func parseScaleFitOptions(args []string) (cols []string, featureMin, featureMax float64, hasRange bool, err error) {
	for i := 0; i < len(args); {
		key := strings.ToLower(args[i])
		switch key {
		case "cols":
			if i+1 >= len(args) {
				return nil, 0, 0, false, fmt.Errorf("scale fit: option %q requires a value", args[i])
			}
			cols = parseCSVTokens(args[i+1])
			i += 2
		case "range":
			if i+2 >= len(args) {
				return nil, 0, 0, false, fmt.Errorf("scale fit: range requires <min> <max>")
			}
			featureMin, err = strconv.ParseFloat(args[i+1], 64)
			if err != nil {
				return nil, 0, 0, false, fmt.Errorf("scale fit: invalid range min %q", args[i+1])
			}
			featureMax, err = strconv.ParseFloat(args[i+2], 64)
			if err != nil {
				return nil, 0, 0, false, fmt.Errorf("scale fit: invalid range max %q", args[i+2])
			}
			hasRange = true
			i += 3
		default:
			return nil, 0, 0, false, fmt.Errorf("scale fit: unknown option %q (supported: cols, range)", args[i])
		}
	}
	return cols, featureMin, featureMax, hasRange, nil
}

func getScalerVar(ctx *ExecContext, name string) (insyra.Scaler, error) {
	value, exists := ctx.Vars[name]
	if !exists {
		return nil, fmt.Errorf("variable not found: %s", name)
	}
	scaler, ok := value.(insyra.Scaler)
	if !ok {
		return nil, fmt.Errorf("variable %s is not a scaler (fit one with 'scale fit ...')", name)
	}
	return scaler, nil
}

func scaledColumnNames(scaler insyra.Scaler) []string {
	params := scaler.Params()
	names := make([]string, 0, len(params))
	for name := range params {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
