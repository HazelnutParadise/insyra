package commands

import (
	"fmt"
	"strings"

	"github.com/HazelnutParadise/insyra/stats"
)

func init() {
	_ = Register(&CommandHandler{Name: "corr", Usage: "corr <x> <y> [pearson|kendall|spearman]", Description: "Correlation between two DataLists", Run: runCorrCommand})
	_ = Register(&CommandHandler{Name: "corrmatrix", Usage: "corrmatrix <datatable> [pearson|kendall|spearman] [as <var>]", Description: "Correlation matrix for a DataTable", Run: runCorrMatrixCommand})
	_ = Register(&CommandHandler{Name: "cov", Usage: "cov <x> <y>", Description: "Covariance between two DataLists", Run: runCovCommand})
	_ = Register(&CommandHandler{Name: "skewness", Usage: "skewness <var>", Description: "Skewness of a DataList", Run: runSkewnessCommand})
	_ = Register(&CommandHandler{Name: "kurtosis", Usage: "kurtosis <var>", Description: "Kurtosis of a DataList", Run: runKurtosisCommand})
}

func runCorrCommand(ctx *ExecContext, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: corr <x> <y> [pearson|kendall|spearman]")
	}
	x, err := getDataListVar(ctx, args[0])
	if err != nil {
		return err
	}
	y, err := getDataListVar(ctx, args[1])
	if err != nil {
		return err
	}
	method, err := parseCorrelationMethod("pearson")
	if err != nil {
		return err
	}
	if len(args) >= 3 {
		method, err = parseCorrelationMethod(args[2])
		if err != nil {
			return err
		}
	}

	result := stats.Correlation(x, y, method)
	if result == nil {
		return fmt.Errorf("failed to calculate correlation")
	}
	_, _ = fmt.Fprintf(ctx.Output, "correlation=%v p=%v\n", result.Statistic, result.PValue)
	return nil
}

func runCorrMatrixCommand(ctx *ExecContext, args []string) error {
	coreArgs, alias := parseAlias(args)
	if len(coreArgs) < 1 {
		return fmt.Errorf("usage: corrmatrix <datatable> [pearson|kendall|spearman] [as <var>]")
	}
	dt, err := getDataTableVar(ctx, coreArgs[0])
	if err != nil {
		return err
	}
	method := stats.PearsonCorrelation
	if len(coreArgs) >= 2 {
		method, err = parseCorrelationMethod(coreArgs[1])
		if err != nil {
			return err
		}
	}

	corr, p := stats.CorrelationMatrix(dt, method)
	if corr == nil {
		return fmt.Errorf("failed to compute correlation matrix")
	}
	ctx.Vars[alias] = corr
	if p != nil {
		ctx.Vars[alias+"_p"] = p
	}
	_, _ = fmt.Fprintf(ctx.Output, "stored %s (and %s_p)\n", alias, alias)
	return nil
}

func runCovCommand(ctx *ExecContext, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: cov <x> <y>")
	}
	x, err := getDataListVar(ctx, args[0])
	if err != nil {
		return err
	}
	y, err := getDataListVar(ctx, args[1])
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(ctx.Output, "%v\n", stats.Covariance(x, y))
	return nil
}

func runSkewnessCommand(ctx *ExecContext, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: skewness <var>")
	}
	dl, err := getDataListVar(ctx, args[0])
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(ctx.Output, "%v\n", stats.Skewness(dl))
	return nil
}

func runKurtosisCommand(ctx *ExecContext, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: kurtosis <var>")
	}
	dl, err := getDataListVar(ctx, args[0])
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(ctx.Output, "%v\n", stats.Kurtosis(dl))
	return nil
}

func parseCorrelationMethod(raw string) (stats.CorrelationMethod, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "pearson", "p":
		return stats.PearsonCorrelation, nil
	case "kendall", "k":
		return stats.KendallCorrelation, nil
	case "spearman", "s":
		return stats.SpearmanCorrelation, nil
	default:
		return stats.PearsonCorrelation, fmt.Errorf("invalid correlation method: %s", raw)
	}
}
