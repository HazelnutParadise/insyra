package commands

import (
	"fmt"
	"strconv"
	"strings"

	insyra "github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats"
)

func init() {
	_ = Register(&CommandHandler{Name: "regression", Usage: "regression <type> <y> <x...>", Description: "Regression analysis: linear/poly/exp/log", Run: runRegressionCommand})
}

func runRegressionCommand(ctx *ExecContext, args []string) error {
	coreArgs, alias := parseAlias(args)
	if len(coreArgs) < 3 {
		return fmt.Errorf("usage: regression <type> <y> <x...>")
	}

	regressionType := strings.ToLower(coreArgs[0])
	y, err := getDataListVar(ctx, coreArgs[1])
	if err != nil {
		return err
	}

	switch regressionType {
	case "linear":
		xs := make([]insyra.IDataList, 0, len(coreArgs)-2)
		for _, name := range coreArgs[2:] {
			x, getErr := getDataListVar(ctx, name)
			if getErr != nil {
				return getErr
			}
			xs = append(xs, x)
		}
		regressionResult := stats.LinearRegression(y, xs...)
		if regressionResult == nil {
			return fmt.Errorf("failed to run linear regression")
		}
		ctx.Vars[alias] = regressionResult
		_, _ = fmt.Fprintf(ctx.Output, "linear regression stored in %s (R2=%v)\n", alias, regressionResult.RSquared)
		return nil
	case "poly", "polynomial":
		if len(coreArgs) < 4 {
			return fmt.Errorf("usage: regression poly <y> <x> <degree> [as <var>]")
		}
		x, err := getDataListVar(ctx, coreArgs[2])
		if err != nil {
			return err
		}
		degree, err := strconv.Atoi(coreArgs[3])
		if err != nil || degree < 1 {
			return fmt.Errorf("invalid polynomial degree: %s", coreArgs[3])
		}
		result := stats.PolynomialRegression(y, x, degree)
		if result == nil {
			return fmt.Errorf("failed to run polynomial regression")
		}
		ctx.Vars[alias] = result
		_, _ = fmt.Fprintf(ctx.Output, "polynomial regression stored in %s (R2=%v)\n", alias, result.RSquared)
		return nil
	case "exp", "exponential":
		x, err := getDataListVar(ctx, coreArgs[2])
		if err != nil {
			return err
		}
		result := stats.ExponentialRegression(y, x)
		if result == nil {
			return fmt.Errorf("failed to run exponential regression")
		}
		ctx.Vars[alias] = result
		_, _ = fmt.Fprintf(ctx.Output, "exponential regression stored in %s (R2=%v)\n", alias, result.RSquared)
		return nil
	case "log", "logarithmic":
		x, err := getDataListVar(ctx, coreArgs[2])
		if err != nil {
			return err
		}
		result := stats.LogarithmicRegression(y, x)
		if result == nil {
			return fmt.Errorf("failed to run logarithmic regression")
		}
		ctx.Vars[alias] = result
		_, _ = fmt.Fprintf(ctx.Output, "logarithmic regression stored in %s (R2=%v)\n", alias, result.RSquared)
		return nil
	default:
		return fmt.Errorf("unsupported regression type: %s", regressionType)
	}
}
