package commands

import (
	"fmt"
	"strconv"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats"
)

func init() {
	_ = Register(&CommandHandler{Name: "kmeans", Usage: "kmeans <var> <k> [nstart <n>] [itermax <n>] [seed <n>] [as <var>]", Description: "K-means clustering", Run: runKMeansCommand})
	_ = Register(&CommandHandler{Name: "hclust", Usage: "hclust <var> <method> [as <var>]", Description: "Hierarchical agglomerative clustering", Run: runHClustCommand})
	_ = Register(&CommandHandler{Name: "cutree", Usage: "cutree <tree_var> k <n>|h <value> [as <var>]", Description: "Cut a hierarchical clustering tree", Run: runCutTreeCommand})
	_ = Register(&CommandHandler{Name: "dbscan", Usage: "dbscan <var> <eps> <minpts> [as <var>]", Description: "Density-based clustering", Run: runDBSCANCommand})
	_ = Register(&CommandHandler{Name: "silhouette", Usage: "silhouette <var> <labels_var> [as <var>]", Description: "Silhouette analysis", Run: runSilhouetteCommand})
}

func runKMeansCommand(ctx *ExecContext, args []string) error {
	coreArgs, alias := parseAlias(args)
	if len(coreArgs) < 2 {
		return fmt.Errorf("usage: kmeans <var> <k> [nstart <n>] [itermax <n>] [seed <n>]")
	}
	dt, err := getDataTableVar(ctx, coreArgs[0])
	if err != nil {
		return err
	}
	k, err := strconv.Atoi(coreArgs[1])
	if err != nil || k < 1 {
		return fmt.Errorf("invalid cluster count: %s", coreArgs[1])
	}
	opts, err := parseKMeansOptions(coreArgs[2:])
	if err != nil {
		return err
	}
	result, err := stats.KMeans(dt, k, opts)
	if err != nil {
		return fmt.Errorf("kmeans failed: %w", err)
	}
	ctx.Vars[alias] = intSliceToDataList(result.Cluster)
	ctx.Vars[alias+"_centers"] = result.Centers
	ctx.Vars[alias+"_size"] = result.Size
	ctx.Vars[alias+"_withinss"] = result.WithinSS
	ctx.Vars[alias+"_totss"] = result.TotSS
	ctx.Vars[alias+"_totwithinss"] = result.TotWithinSS
	ctx.Vars[alias+"_betweenss"] = result.BetweenSS
	ctx.Vars[alias+"_iter"] = result.Iter
	ctx.Vars[alias+"_ifault"] = result.IFault
	_, _ = fmt.Fprintf(ctx.Output, "stored %s (labels)\n", alias)
	return nil
}

func runHClustCommand(ctx *ExecContext, args []string) error {
	coreArgs, alias := parseAlias(args)
	if len(coreArgs) < 2 {
		return fmt.Errorf("usage: hclust <var> <method>")
	}
	dt, err := getDataTableVar(ctx, coreArgs[0])
	if err != nil {
		return err
	}
	result, err := stats.HierarchicalAgglomerative(dt, stats.AgglomerativeMethod(coreArgs[1]))
	if err != nil {
		return fmt.Errorf("hclust failed: %w", err)
	}
	ctx.Vars[alias] = result
	_, _ = fmt.Fprintf(ctx.Output, "stored %s (tree)\n", alias)
	return nil
}

func runCutTreeCommand(ctx *ExecContext, args []string) error {
	coreArgs, alias := parseAlias(args)
	if len(coreArgs) != 3 {
		return fmt.Errorf("usage: cutree <tree_var> k <n>|h <value>")
	}
	tree, ok := ctx.Vars[coreArgs[0]].(*stats.HierarchicalResult)
	if !ok {
		return fmt.Errorf("%s is not a hierarchical result", coreArgs[0])
	}
	var labels []int
	var err error
	switch coreArgs[1] {
	case "k":
		k, convErr := strconv.Atoi(coreArgs[2])
		if convErr != nil {
			return fmt.Errorf("invalid k: %s", coreArgs[2])
		}
		labels, err = stats.CutTreeByK(tree, k)
	case "h":
		h, convErr := strconv.ParseFloat(coreArgs[2], 64)
		if convErr != nil {
			return fmt.Errorf("invalid height: %s", coreArgs[2])
		}
		labels, err = stats.CutTreeByHeight(tree, h)
	default:
		return fmt.Errorf("cut mode must be k or h")
	}
	if err != nil {
		return fmt.Errorf("cutree failed: %w", err)
	}
	ctx.Vars[alias] = intSliceToDataList(labels)
	_, _ = fmt.Fprintf(ctx.Output, "stored %s (labels)\n", alias)
	return nil
}

func runDBSCANCommand(ctx *ExecContext, args []string) error {
	coreArgs, alias := parseAlias(args)
	if len(coreArgs) < 3 {
		return fmt.Errorf("usage: dbscan <var> <eps> <minpts>")
	}
	dt, err := getDataTableVar(ctx, coreArgs[0])
	if err != nil {
		return err
	}
	eps, err := strconv.ParseFloat(coreArgs[1], 64)
	if err != nil {
		return fmt.Errorf("invalid eps: %s", coreArgs[1])
	}
	minPts, err := strconv.Atoi(coreArgs[2])
	if err != nil {
		return fmt.Errorf("invalid minpts: %s", coreArgs[2])
	}
	result, err := stats.DBSCAN(dt, eps, minPts)
	if err != nil {
		return fmt.Errorf("dbscan failed: %w", err)
	}
	ctx.Vars[alias] = intSliceToDataList(result.Cluster)
	ctx.Vars[alias+"_isseed"] = result.IsSeed
	_, _ = fmt.Fprintf(ctx.Output, "stored %s (labels)\n", alias)
	return nil
}

func runSilhouetteCommand(ctx *ExecContext, args []string) error {
	coreArgs, alias := parseAlias(args)
	if len(coreArgs) < 2 {
		return fmt.Errorf("usage: silhouette <var> <labels_var>")
	}
	dt, err := getDataTableVar(ctx, coreArgs[0])
	if err != nil {
		return err
	}
	labels, err := getDataListVar(ctx, coreArgs[1])
	if err != nil {
		return err
	}
	result, err := stats.Silhouette(dt, labels)
	if err != nil {
		return fmt.Errorf("silhouette failed: %w", err)
	}
	widths := insyra.NewDataList()
	for _, pt := range result.Points {
		widths.Append(pt.SilWidth)
	}
	ctx.Vars[alias] = widths
	ctx.Vars[alias+"_avg"] = result.AverageSilhouette
	_, _ = fmt.Fprintf(ctx.Output, "stored %s (widths)\n", alias)
	return nil
}

func parseKMeansOptions(args []string) (stats.KMeansOptions, error) {
	opts := stats.KMeansOptions{}
	for i := 0; i < len(args); i += 2 {
		if i+1 >= len(args) {
			return opts, fmt.Errorf("missing value for option %s", args[i])
		}
		switch args[i] {
		case "nstart":
			v, err := strconv.Atoi(args[i+1])
			if err != nil {
				return opts, fmt.Errorf("invalid nstart: %s", args[i+1])
			}
			opts.NStart = v
		case "itermax":
			v, err := strconv.Atoi(args[i+1])
			if err != nil {
				return opts, fmt.Errorf("invalid itermax: %s", args[i+1])
			}
			opts.IterMax = v
		case "seed":
			v, err := strconv.ParseInt(args[i+1], 10, 64)
			if err != nil {
				return opts, fmt.Errorf("invalid seed: %s", args[i+1])
			}
			opts.Seed = &v
		default:
			return opts, fmt.Errorf("unknown option: %s", args[i])
		}
	}
	return opts, nil
}

func intSliceToDataList(values []int) *insyra.DataList {
	out := insyra.NewDataList()
	for _, v := range values {
		out.Append(v)
	}
	return out
}
