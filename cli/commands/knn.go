package commands

import (
	"fmt"
	"strconv"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats"
)

func init() {
	_ = Register(&CommandHandler{Name: "knn_classify", Usage: "knn_classify <train_var> <labels_var> <test_var> <k> [weighting <uniform|distance>] [algorithm <auto|brute|kd_tree|ball_tree>] [leafsize <n>] [as <var>]", Description: "K-nearest neighbors classification", Run: runKNNClassifyCommand})
	_ = Register(&CommandHandler{Name: "knn_regress", Usage: "knn_regress <train_var> <targets_var> <test_var> <k> [weighting <uniform|distance>] [algorithm <auto|brute|kd_tree|ball_tree>] [leafsize <n>] [as <var>]", Description: "K-nearest neighbors regression", Run: runKNNRegressCommand})
	_ = Register(&CommandHandler{Name: "knn_neighbors", Usage: "knn_neighbors <train_var> <test_var> <k> [algorithm <auto|brute|kd_tree|ball_tree>] [leafsize <n>] [as <var>]", Description: "K-nearest neighbors search", Run: runKNNNeighborsCommand})
}

func runKNNClassifyCommand(ctx *ExecContext, args []string) error {
	coreArgs, alias := parseAlias(args)
	if len(coreArgs) < 4 {
		return fmt.Errorf("usage: knn_classify <train_var> <labels_var> <test_var> <k> [weighting <uniform|distance>] [algorithm <auto|brute|kd_tree|ball_tree>] [leafsize <n>]")
	}
	train, err := getDataTableVar(ctx, coreArgs[0])
	if err != nil {
		return err
	}
	labels, err := getDataListVar(ctx, coreArgs[1])
	if err != nil {
		return err
	}
	test, err := getDataTableVar(ctx, coreArgs[2])
	if err != nil {
		return err
	}
	k, err := strconv.Atoi(coreArgs[3])
	if err != nil {
		return fmt.Errorf("invalid k: %s", coreArgs[3])
	}
	opts, err := parseKNNCommandOptions(coreArgs[4:])
	if err != nil {
		return err
	}
	result, err := stats.KNNClassify(train, labels, test, k, opts)
	if err != nil {
		return fmt.Errorf("knn_classify failed: %w", err)
	}
	ctx.Vars[alias] = result.Predictions
	ctx.Vars[alias+"_classes"] = result.Classes
	ctx.Vars[alias+"_probs"] = result.Probabilities
	_, _ = fmt.Fprintf(ctx.Output, "stored %s (predictions)\n", alias)
	return nil
}

func runKNNRegressCommand(ctx *ExecContext, args []string) error {
	coreArgs, alias := parseAlias(args)
	if len(coreArgs) < 4 {
		return fmt.Errorf("usage: knn_regress <train_var> <targets_var> <test_var> <k> [weighting <uniform|distance>] [algorithm <auto|brute|kd_tree|ball_tree>] [leafsize <n>]")
	}
	train, err := getDataTableVar(ctx, coreArgs[0])
	if err != nil {
		return err
	}
	targets, err := getDataListVar(ctx, coreArgs[1])
	if err != nil {
		return err
	}
	test, err := getDataTableVar(ctx, coreArgs[2])
	if err != nil {
		return err
	}
	k, err := strconv.Atoi(coreArgs[3])
	if err != nil {
		return fmt.Errorf("invalid k: %s", coreArgs[3])
	}
	opts, err := parseKNNCommandOptions(coreArgs[4:])
	if err != nil {
		return err
	}
	result, err := stats.KNNRegress(train, targets, test, k, opts)
	if err != nil {
		return fmt.Errorf("knn_regress failed: %w", err)
	}
	ctx.Vars[alias] = floatSliceToDataList(result.Predictions)
	_, _ = fmt.Fprintf(ctx.Output, "stored %s (predictions)\n", alias)
	return nil
}

func runKNNNeighborsCommand(ctx *ExecContext, args []string) error {
	coreArgs, alias := parseAlias(args)
	if len(coreArgs) < 3 {
		return fmt.Errorf("usage: knn_neighbors <train_var> <test_var> <k> [algorithm <auto|brute|kd_tree|ball_tree>] [leafsize <n>]")
	}
	train, err := getDataTableVar(ctx, coreArgs[0])
	if err != nil {
		return err
	}
	test, err := getDataTableVar(ctx, coreArgs[1])
	if err != nil {
		return err
	}
	k, err := strconv.Atoi(coreArgs[2])
	if err != nil {
		return fmt.Errorf("invalid k: %s", coreArgs[2])
	}
	opts, err := parseKNNCommandOptions(coreArgs[3:])
	if err != nil {
		return err
	}
	result, err := stats.KNearestNeighbors(train, test, k, opts)
	if err != nil {
		return fmt.Errorf("knn_neighbors failed: %w", err)
	}
	ctx.Vars[alias] = intMatrixToDataTable(result.Indices, "neighbor")
	ctx.Vars[alias+"_distances"] = floatMatrixToDataTable(result.Distances, "distance")
	_, _ = fmt.Fprintf(ctx.Output, "stored %s (indices)\n", alias)
	return nil
}

func parseKNNCommandOptions(args []string) (stats.KNNOptions, error) {
	opts := stats.KNNOptions{}
	for i := 0; i < len(args); i += 2 {
		if i+1 >= len(args) {
			return opts, fmt.Errorf("missing value for option %s", args[i])
		}
		switch args[i] {
		case "weighting":
			opts.Weighting = stats.KNNWeighting(args[i+1])
		case "algorithm":
			opts.Algorithm = stats.KNNAlgorithm(args[i+1])
		case "leafsize":
			v, err := strconv.Atoi(args[i+1])
			if err != nil {
				return opts, fmt.Errorf("invalid leafsize: %s", args[i+1])
			}
			opts.LeafSize = v
		default:
			return opts, fmt.Errorf("unknown option: %s", args[i])
		}
	}
	return opts, nil
}

func floatSliceToDataList(values []float64) *insyra.DataList {
	out := insyra.NewDataList()
	for _, v := range values {
		out.Append(v)
	}
	return out
}

func intMatrixToDataTable(rows [][]int, prefix string) *insyra.DataTable {
	if len(rows) == 0 {
		return insyra.NewDataTable()
	}
	dt := insyra.NewDataTable()
	for c := range len(rows[0]) {
		col := insyra.NewDataList().SetName(fmt.Sprintf("%s%d", prefix, c+1))
		for r := range len(rows) {
			col.Append(rows[r][c])
		}
		dt.AppendCols(col)
	}
	return dt
}

func floatMatrixToDataTable(rows [][]float64, prefix string) *insyra.DataTable {
	if len(rows) == 0 {
		return insyra.NewDataTable()
	}
	dt := insyra.NewDataTable()
	for c := range len(rows[0]) {
		col := insyra.NewDataList().SetName(fmt.Sprintf("%s%d", prefix, c+1))
		for r := range len(rows) {
			col.Append(rows[r][c])
		}
		dt.AppendCols(col)
	}
	return dt
}
