package commands

import (
	"fmt"
	"strconv"
	"strings"

	insyra "github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats"
)

func init() {
	_ = Register(&CommandHandler{Name: "ttest", Usage: "ttest single|two|paired ...", Description: "T-test commands", Run: runTTestCommand})
	_ = Register(&CommandHandler{Name: "ztest", Usage: "ztest single|two ...", Description: "Z-test commands", Run: runZTestCommand})
	_ = Register(&CommandHandler{Name: "anova", Usage: "anova oneway|twoway|repeated ...", Description: "ANOVA commands", Run: runAnovaCommand})
	_ = Register(&CommandHandler{Name: "ftest", Usage: "ftest var|levene|bartlett ...", Description: "F-test commands", Run: runFTestCommand})
	_ = Register(&CommandHandler{Name: "chisq", Usage: "chisq gof|indep ...", Description: "Chi-square test commands", Run: runChiSqCommand})
}

func runTTestCommand(ctx *ExecContext, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: ttest single|two|paired")
	}
	switch strings.ToLower(args[0]) {
	case "single":
		if len(args) < 3 {
			return fmt.Errorf("usage: ttest single <var> <mu>")
		}
		dl, err := getDataListVar(ctx, args[1])
		if err != nil {
			return err
		}
		mu, err := strconv.ParseFloat(args[2], 64)
		if err != nil {
			return err
		}
		result := stats.SingleSampleTTest(dl, mu)
		if result == nil {
			return fmt.Errorf("ttest failed")
		}
		_, _ = fmt.Fprintf(ctx.Output, "t=%v p=%v\n", result.Statistic, result.PValue)
		return nil
	case "two":
		if len(args) < 3 {
			return fmt.Errorf("usage: ttest two <var1> <var2> [equal|unequal]")
		}
		a, err := getDataListVar(ctx, args[1])
		if err != nil {
			return err
		}
		b, err := getDataListVar(ctx, args[2])
		if err != nil {
			return err
		}
		equalVariance := true
		if len(args) >= 4 {
			equalVariance = strings.EqualFold(args[3], "equal")
		}
		result := stats.TwoSampleTTest(a, b, equalVariance)
		if result == nil {
			return fmt.Errorf("ttest failed")
		}
		_, _ = fmt.Fprintf(ctx.Output, "t=%v p=%v\n", result.Statistic, result.PValue)
		return nil
	case "paired":
		if len(args) < 3 {
			return fmt.Errorf("usage: ttest paired <var1> <var2>")
		}
		a, err := getDataListVar(ctx, args[1])
		if err != nil {
			return err
		}
		b, err := getDataListVar(ctx, args[2])
		if err != nil {
			return err
		}
		result := stats.PairedTTest(a, b)
		if result == nil {
			return fmt.Errorf("ttest failed")
		}
		_, _ = fmt.Fprintf(ctx.Output, "t=%v p=%v\n", result.Statistic, result.PValue)
		return nil
	default:
		return fmt.Errorf("unsupported ttest mode: %s", args[0])
	}
}

func runZTestCommand(ctx *ExecContext, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: ztest single|two")
	}
	switch strings.ToLower(args[0]) {
	case "single":
		if len(args) < 4 {
			return fmt.Errorf("usage: ztest single <var> <mu> <sigma> [two-sided|greater|less]")
		}
		dl, err := getDataListVar(ctx, args[1])
		if err != nil {
			return err
		}
		mu, err := strconv.ParseFloat(args[2], 64)
		if err != nil {
			return err
		}
		sigma, err := strconv.ParseFloat(args[3], 64)
		if err != nil {
			return err
		}
		alternative := stats.TwoSided
		if len(args) >= 5 {
			alternative = parseAlternativeHypothesis(args[4])
		}
		result := stats.SingleSampleZTest(dl, mu, sigma, alternative, 0.95)
		if result == nil {
			return fmt.Errorf("ztest failed")
		}
		_, _ = fmt.Fprintf(ctx.Output, "z=%v p=%v\n", result.Statistic, result.PValue)
		return nil
	case "two":
		if len(args) < 5 {
			return fmt.Errorf("usage: ztest two <var1> <var2> <sigma1> <sigma2> [two-sided|greater|less]")
		}
		a, err := getDataListVar(ctx, args[1])
		if err != nil {
			return err
		}
		b, err := getDataListVar(ctx, args[2])
		if err != nil {
			return err
		}
		s1, err := strconv.ParseFloat(args[3], 64)
		if err != nil {
			return err
		}
		s2, err := strconv.ParseFloat(args[4], 64)
		if err != nil {
			return err
		}
		alternative := stats.TwoSided
		if len(args) >= 6 {
			alternative = parseAlternativeHypothesis(args[5])
		}
		result := stats.TwoSampleZTest(a, b, s1, s2, alternative, 0.95)
		if result == nil {
			return fmt.Errorf("ztest failed")
		}
		_, _ = fmt.Fprintf(ctx.Output, "z=%v p=%v\n", result.Statistic, result.PValue)
		return nil
	default:
		return fmt.Errorf("unsupported ztest mode: %s", args[0])
	}
}

func runAnovaCommand(ctx *ExecContext, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: anova oneway|twoway|repeated")
	}
	switch strings.ToLower(args[0]) {
	case "oneway":
		if len(args) < 3 {
			return fmt.Errorf("usage: anova oneway <group1> <group2> [group3...]")
		}
		groups, err := getDataListGroups(ctx, args[1:])
		if err != nil {
			return err
		}
		result := stats.OneWayANOVA(groups...)
		if result == nil {
			return fmt.Errorf("anova failed")
		}
		_, _ = fmt.Fprintf(ctx.Output, "F=%v p=%v\n", result.Factor.F, result.Factor.P)
		return nil
	case "twoway":
		if len(args) < 4 {
			return fmt.Errorf("usage: anova twoway <aLevels> <bLevels> <cell1> <cell2> ...")
		}
		aLevels, err := strconv.Atoi(args[1])
		if err != nil {
			return err
		}
		bLevels, err := strconv.Atoi(args[2])
		if err != nil {
			return err
		}
		cells, err := getDataListGroups(ctx, args[3:])
		if err != nil {
			return err
		}
		if len(cells) != aLevels*bLevels {
			return fmt.Errorf("twoway requires exactly %d cells", aLevels*bLevels)
		}
		result := stats.TwoWayANOVA(aLevels, bLevels, cells...)
		if result == nil {
			return fmt.Errorf("anova failed")
		}
		_, _ = fmt.Fprintf(ctx.Output, "FA=%v pA=%v FB=%v pB=%v\n", result.FactorA.F, result.FactorA.P, result.FactorB.F, result.FactorB.P)
		return nil
	case "repeated":
		if len(args) < 3 {
			return fmt.Errorf("usage: anova repeated <subject1> <subject2> ...")
		}
		subjects, err := getDataListGroups(ctx, args[1:])
		if err != nil {
			return err
		}
		result := stats.RepeatedMeasuresANOVA(subjects...)
		if result == nil {
			return fmt.Errorf("anova failed")
		}
		_, _ = fmt.Fprintf(ctx.Output, "F=%v p=%v\n", result.Factor.F, result.Factor.P)
		return nil
	default:
		return fmt.Errorf("unsupported anova mode: %s", args[0])
	}
}

func runFTestCommand(ctx *ExecContext, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: ftest var|levene|bartlett ...")
	}
	switch strings.ToLower(args[0]) {
	case "var":
		if len(args) < 3 {
			return fmt.Errorf("usage: ftest var <var1> <var2>")
		}
		a, err := getDataListVar(ctx, args[1])
		if err != nil {
			return err
		}
		b, err := getDataListVar(ctx, args[2])
		if err != nil {
			return err
		}
		result := stats.FTestForVarianceEquality(a, b)
		if result == nil {
			return fmt.Errorf("ftest failed")
		}
		_, _ = fmt.Fprintf(ctx.Output, "F=%v p=%v\n", result.Statistic, result.PValue)
		return nil
	case "levene":
		if len(args) < 3 {
			return fmt.Errorf("usage: ftest levene <group1> <group2> [group3...]")
		}
		groups, err := getDataListGroups(ctx, args[1:])
		if err != nil {
			return err
		}
		result := stats.LeveneTest(groups)
		if result == nil {
			return fmt.Errorf("levene test failed")
		}
		_, _ = fmt.Fprintf(ctx.Output, "F=%v p=%v\n", result.Statistic, result.PValue)
		return nil
	case "bartlett":
		if len(args) < 3 {
			return fmt.Errorf("usage: ftest bartlett <group1> <group2> [group3...]")
		}
		groups, err := getDataListGroups(ctx, args[1:])
		if err != nil {
			return err
		}
		result := stats.BartlettTest(groups)
		if result == nil {
			return fmt.Errorf("bartlett test failed")
		}
		_, _ = fmt.Fprintf(ctx.Output, "chi2=%v p=%v\n", result.Statistic, result.PValue)
		return nil
	default:
		return fmt.Errorf("unsupported ftest mode: %s", args[0])
	}
}

func runChiSqCommand(ctx *ExecContext, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: chisq gof|indep ...")
	}
	switch strings.ToLower(args[0]) {
	case "gof":
		if len(args) < 2 {
			return fmt.Errorf("usage: chisq gof <var> [p1 p2 ...]")
		}
		dl, err := getDataListVar(ctx, args[1])
		if err != nil {
			return err
		}
		probabilities := make([]float64, 0, len(args)-2)
		for _, raw := range args[2:] {
			value, parseErr := strconv.ParseFloat(raw, 64)
			if parseErr != nil {
				return parseErr
			}
			probabilities = append(probabilities, value)
		}
		result := stats.ChiSquareGoodnessOfFit(dl, probabilities, true)
		if result == nil {
			return fmt.Errorf("chi-square gof failed")
		}
		_, _ = fmt.Fprintf(ctx.Output, "chi2=%v p=%v\n", result.Statistic, result.PValue)
		return nil
	case "indep":
		if len(args) < 3 {
			return fmt.Errorf("usage: chisq indep <rowVar> <colVar>")
		}
		row, err := getDataListVar(ctx, args[1])
		if err != nil {
			return err
		}
		col, err := getDataListVar(ctx, args[2])
		if err != nil {
			return err
		}
		result := stats.ChiSquareIndependenceTest(row, col)
		if result == nil {
			return fmt.Errorf("chi-square independence test failed")
		}
		_, _ = fmt.Fprintf(ctx.Output, "chi2=%v p=%v\n", result.Statistic, result.PValue)
		return nil
	default:
		return fmt.Errorf("unsupported chisq mode: %s", args[0])
	}
}

func parseAlternativeHypothesis(raw string) stats.AlternativeHypothesis {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "greater", ">":
		return stats.Greater
	case "less", "<":
		return stats.Less
	default:
		return stats.TwoSided
	}
}

func getDataListGroups(ctx *ExecContext, names []string) ([]insyra.IDataList, error) {
	groups := make([]insyra.IDataList, 0, len(names))
	for _, name := range names {
		dl, err := getDataListVar(ctx, name)
		if err != nil {
			return nil, err
		}
		groups = append(groups, dl)
	}
	return groups, nil
}
