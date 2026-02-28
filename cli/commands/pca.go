package commands

import (
	"fmt"
	"strconv"

	"github.com/HazelnutParadise/insyra/stats"
)

func init() {
	_ = Register(&CommandHandler{Name: "pca", Usage: "pca <var> <n>", Description: "Principal component analysis", Run: runPCACommand})
}

func runPCACommand(ctx *ExecContext, args []string) error {
	coreArgs, alias := parseAlias(args)
	if len(coreArgs) < 2 {
		return fmt.Errorf("usage: pca <var> <n>")
	}
	dt, err := getDataTableVar(ctx, coreArgs[0])
	if err != nil {
		return err
	}
	n, err := strconv.Atoi(coreArgs[1])
	if err != nil || n < 1 {
		return fmt.Errorf("invalid number of components: %s", coreArgs[1])
	}
	result := stats.PCA(dt, n)
	if result == nil {
		return fmt.Errorf("pca failed")
	}
	ctx.Vars[alias] = result.Components
	ctx.Vars[alias+"_eigenvalues"] = result.Eigenvalues
	ctx.Vars[alias+"_explained_variance"] = result.ExplainedVariance
	_, _ = fmt.Fprintf(ctx.Output, "stored %s (components)\n", alias)
	return nil
}
