package commands

import (
	"fmt"
	"math"
	"strings"

	insyra "github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats"
)

// friedmanUsage is the friedman command's usage line, shared by its
// registration and its error for no arguments.
const friedmanUsage = "friedman <table> <value> <condition> <subject> | friedman <subject1> <subject2> [subjectN]"

func init() {
	_ = Register(&CommandHandler{
		Name:        "ttest",
		Args:        FormArgs(map[string]int{"single": 3, "two": 4, "paired": 3}),
		Usage:       "ttest single|two|paired ...",
		Description: "T-test commands",
		Forms: []string{
			"ttest single <var> <mu>                     one-sample, against population mean mu",
			"ttest two <var1> <var2> [equal|unequal]     two-sample (default: unequal, Welch's test)",
			"ttest paired <var1> <var2>                  paired on matched samples",
		},
		Examples: []string{
			"insyra ttest single weights 70",
			"insyra ttest two before after equal",
			"insyra ttest paired pre post",
		},
		Run: runTTestCommand,
	})
	_ = Register(&CommandHandler{
		Name:        "ztest",
		Args:        FormArgs(map[string]int{"single": 5, "two": 6}),
		Usage:       "ztest single|two ...",
		Description: "Z-test commands",
		Forms: []string{
			"ztest single <var> <mu> <sigma> [two-sided|greater|less]",
			"ztest two <var1> <var2> <sigma1> <sigma2> [two-sided|greater|less]",
		},
		Examples: []string{
			"insyra ztest single iq 100 15",
			"insyra ztest two a b 1.0 1.2 greater",
		},
		Run: runZTestCommand,
	})
	_ = Register(&CommandHandler{
		Name:        "anova",
		Args:        OpenArgs(),
		Usage:       "anova oneway|twoway|repeated ...",
		Description: "ANOVA commands",
		Forms: []string{
			"anova oneway <group1> <group2> [group3...]                  one-way ANOVA across groups",
			"anova twoway <aLevels> <bLevels> <cell1> <cell2> ...        two-way ANOVA; cell count must equal aLevels*bLevels",
			"anova twoway <table> <value> <factorA> <factorB>            two-way ANOVA on a table with one row per observation",
			"anova repeated <subject1> <subject2> [subjectN]             repeated-measures ANOVA",
			"anova repeated <table> <value> <condition> <subject>        repeated-measures ANOVA on a table with one row per measurement",
		},
		Examples: []string{
			"insyra anova oneway g1 g2 g3",
			"insyra anova twoway 2 3 c11 c12 c13 c21 c22 c23",
			"insyra anova twoway scores score drug dose",
			"insyra anova repeated s1 s2 s3",
			"insyra anova repeated trial value visit patient",
		},
		Run: runAnovaCommand,
	})
	_ = Register(&CommandHandler{
		Name:        "ftest",
		Args:        FormArgs(map[string]int{"var": 3, "levene": unlimited, "bartlett": unlimited}),
		Usage:       "ftest var|levene|bartlett ...",
		Description: "F-test commands",
		Forms: []string{
			"ftest var <var1> <var2>                          F-test for equality of two variances",
			"ftest levene <group1> <group2> [group3...]       Levene's test for variance homogeneity",
			"ftest bartlett <group1> <group2> [group3...]     Bartlett's test for variance homogeneity",
		},
		Examples: []string{
			"insyra ftest var a b",
			"insyra ftest levene g1 g2 g3",
			"insyra ftest bartlett g1 g2 g3",
		},
		Run: runFTestCommand,
	})
	_ = Register(&CommandHandler{
		Name:        "chisq",
		Args:        FormArgs(map[string]int{"gof": unlimited, "indep": 3}),
		Usage:       "chisq gof|indep ...",
		Description: "Chi-square test commands",
		Forms: []string{
			"chisq gof <var> [label=p ...]                goodness-of-fit; each proportion names its category, uniform when none are given",
			"chisq indep <rowVar> <colVar>                independence test on contingency table",
		},
		Examples: []string{
			"insyra chisq gof colors",
			"insyra chisq gof colors red=0.5 green=0.3 blue=0.2",
			"insyra chisq indep gender preference",
		},
		Run: runChiSqCommand,
	})
	_ = Register(&CommandHandler{
		Name:        "friedman",
		Args:        OpenArgs(),
		Usage:       friedmanUsage,
		Description: "Friedman rank test for repeated measures",
		Forms: []string{
			"friedman <table> <value> <condition> <subject>    one row per measurement",
			"friedman <subject1> <subject2> [subjectN]         one DataList per subject, its values in condition order",
		},
		Examples: []string{
			"insyra friedman trial value visit patient",
			"insyra friedman s1 s2 s3 s4",
		},
		Run: runFriedmanCommand,
	})
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
		mu, err := parseFloatArg("ttest", "mu", args[2])
		if err != nil {
			return err
		}
		result, err := stats.SingleSampleTTest(dl, mu)
		if err != nil {
			return fmt.Errorf("ttest failed: %w", err)
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
		equalVariance := false
		if len(args) >= 4 {
			var parseErr error
			equalVariance, parseErr = parseEqualVariance(args[3])
			if parseErr != nil {
				return fmt.Errorf("ttest: %w", parseErr)
			}
		}
		result, err := stats.TwoSampleTTest(a, b, stats.TTestOptions{EqualVariance: equalVariance})
		if err != nil {
			return fmt.Errorf("ttest failed: %w", err)
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
		result, err := stats.PairedTTest(a, b)
		if err != nil {
			return fmt.Errorf("ttest failed: %w", err)
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
		mu, err := parseFloatArg("ztest", "mu", args[2])
		if err != nil {
			return err
		}
		sigma, err := parseFloatArg("ztest", "sigma", args[3])
		if err != nil {
			return err
		}
		alternative := stats.TwoSided
		if len(args) >= 5 {
			var parseErr error
			alternative, parseErr = parseAlternativeHypothesis(args[4])
			if parseErr != nil {
				return fmt.Errorf("%w", parseErr)
			}
		}
		result, err := stats.SingleSampleZTest(dl, mu, sigma, stats.ZTestOptions{Alternative: alternative})
		if err != nil {
			return fmt.Errorf("ztest failed: %w", err)
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
		s1, err := parseFloatArg("ztest", "s1", args[3])
		if err != nil {
			return err
		}
		s2, err := parseFloatArg("ztest", "s2", args[4])
		if err != nil {
			return err
		}
		alternative := stats.TwoSided
		if len(args) >= 6 {
			var parseErr error
			alternative, parseErr = parseAlternativeHypothesis(args[5])
			if parseErr != nil {
				return fmt.Errorf("%w", parseErr)
			}
		}
		result, err := stats.TwoSampleZTest(a, b, s1, s2, stats.ZTestOptions{Alternative: alternative})
		if err != nil {
			return fmt.Errorf("ztest failed: %w", err)
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
		result, err := stats.OneWayANOVA(groups)
		if err != nil {
			return fmt.Errorf("anova failed: %w", err)
		}
		_, _ = fmt.Fprintf(ctx.Output, "F=%v p=%v\n", result.Factor.F, result.Factor.P)
		return nil
	case "twoway":
		if len(args) >= 2 {
			if table := tableVar(ctx, args[1]); table != nil {
				if len(args) != 5 {
					return fmt.Errorf("usage: anova twoway <table> <value> <factorA> <factorB>")
				}
				selectors, err := colSelectors("anova", table, args[2:5])
				if err != nil {
					return err
				}
				result, err := stats.TwoWayANOVAFromTable(table, selectors[0], selectors[1], selectors[2])
				if err != nil {
					return fmt.Errorf("anova failed: %w", err)
				}
				_, _ = fmt.Fprintf(ctx.Output, "FA=%v pA=%v FB=%v pB=%v\n", result.FactorA.F, result.FactorA.P, result.FactorB.F, result.FactorB.P)
				return nil
			}
		}
		if len(args) < 4 {
			return fmt.Errorf("usage: anova twoway <aLevels> <bLevels> <cell1> <cell2> [cellN]")
		}
		aLevels, err := parseIntArg("anova", "aLevels", args[1])
		if err != nil {
			return err
		}
		bLevels, err := parseIntArg("anova", "bLevels", args[2])
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
		result, err := stats.TwoWayANOVA(aLevels, bLevels, cells)
		if err != nil {
			return fmt.Errorf("anova failed: %w", err)
		}
		_, _ = fmt.Fprintf(ctx.Output, "FA=%v pA=%v FB=%v pB=%v\n", result.FactorA.F, result.FactorA.P, result.FactorB.F, result.FactorB.P)
		return nil
	case "repeated":
		if len(args) >= 2 {
			if table := tableVar(ctx, args[1]); table != nil {
				if len(args) != 5 {
					return fmt.Errorf("usage: anova repeated <table> <value> <condition> <subject>")
				}
				selectors, err := colSelectors("anova", table, args[2:5])
				if err != nil {
					return err
				}
				result, err := stats.RepeatedMeasuresANOVAFromTable(table, selectors[0], selectors[1], selectors[2])
				if err != nil {
					return fmt.Errorf("anova failed: %w", err)
				}
				_, _ = fmt.Fprintf(ctx.Output, "F=%v p=%v\n", result.Factor.F, result.Factor.P)
				return nil
			}
		}
		if len(args) < 3 {
			return fmt.Errorf("usage: anova repeated <subject1> <subject2> [subjectN]")
		}
		subjects, err := getDataListGroups(ctx, args[1:])
		if err != nil {
			return err
		}
		result, err := stats.RepeatedMeasuresANOVA(subjects)
		if err != nil {
			return fmt.Errorf("anova failed: %w", err)
		}
		_, _ = fmt.Fprintf(ctx.Output, "F=%v p=%v\n", result.Factor.F, result.Factor.P)
		return nil
	default:
		return fmt.Errorf("unsupported anova mode: %s", args[0])
	}
}

func runFTestCommand(ctx *ExecContext, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: ftest var|levene|bartlett")
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
		result, err := stats.FTestForVarianceEquality(a, b)
		if err != nil {
			return fmt.Errorf("ftest failed: %w", err)
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
		result, err := stats.LeveneTest(groups)
		if err != nil {
			return fmt.Errorf("levene test failed: %w", err)
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
		result, err := stats.BartlettTest(groups)
		if err != nil {
			return fmt.Errorf("bartlett test failed: %w", err)
		}
		_, _ = fmt.Fprintf(ctx.Output, "chi2=%v p=%v\n", result.Statistic, result.PValue)
		return nil
	default:
		return fmt.Errorf("unsupported ftest mode: %s", args[0])
	}
}

func runChiSqCommand(ctx *ExecContext, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: chisq gof|indep")
	}
	switch strings.ToLower(args[0]) {
	case "gof":
		if len(args) < 2 {
			return fmt.Errorf("usage: chisq gof <var> [label=p ...]")
		}
		dl, err := getDataListVar(ctx, args[1])
		if err != nil {
			return err
		}
		var probabilities map[string]float64
		if len(args) > 2 {
			probabilities = make(map[string]float64, len(args)-2)
			for _, raw := range args[2:] {
				eq := strings.LastIndex(raw, "=")
				if eq < 0 {
					return fmt.Errorf(`chisq gof: expected label=proportion, got %q`, raw)
				}
				label := raw[:eq]
				value, parseErr := parseFloatArg("chisq", "proportion", raw[eq+1:])
				if parseErr != nil {
					return parseErr
				}
				if _, exists := probabilities[label]; exists {
					return fmt.Errorf(`chisq gof: category %q is given twice`, label)
				}
				probabilities[label] = value
			}
		}
		result, err := stats.ChiSquareGoodnessOfFit(dl, probabilities, true)
		if err != nil {
			return fmt.Errorf("chi-square gof failed: %w", err)
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
		result, err := stats.ChiSquareIndependenceTest(row, col)
		if err != nil {
			return fmt.Errorf("chi-square independence test failed: %w", err)
		}
		_, _ = fmt.Fprintf(ctx.Output, "chi2=%v p=%v\n", result.Statistic, result.PValue)
		return nil
	default:
		return fmt.Errorf("unsupported chisq mode: %s", args[0])
	}
}

// parseAlternativeHypothesis accepts the documented spellings and nothing
// else: silently falling back to two-sided would report a p-value computed
// under an assumption the caller did not make.
func parseAlternativeHypothesis(raw string) (stats.AlternativeHypothesis, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "greater", ">":
		return stats.Greater, nil
	case "less", "<":
		return stats.Less, nil
	case "two-sided", "twosided", "two_sided", "!=", "":
		return stats.TwoSided, nil
	default:
		return stats.TwoSided, fmt.Errorf("invalid alternative %q (use two-sided, greater or less)", raw)
	}
}

// parseEqualVariance accepts the documented spellings for the two-sample
// t-test's variance assumption.
func parseEqualVariance(raw string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "equal", "pooled":
		return true, nil
	case "unequal", "welch":
		return false, nil
	default:
		return false, fmt.Errorf("invalid variance assumption %q (use equal or unequal)", raw)
	}
}

// tableVar returns the DataTable a variable holds, or nil when the variable
// does not exist or holds something else, so a command can pick its table
// form by the type of its first argument.
func tableVar(ctx *ExecContext, name string) *insyra.DataTable {
	table, _ := ctx.Vars[name].(*insyra.DataTable)
	return table
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

func runFriedmanCommand(ctx *ExecContext, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: %s", friedmanUsage)
	}
	if table := tableVar(ctx, args[0]); table != nil {
		if len(args) != 4 {
			return fmt.Errorf("usage: friedman <table> <value> <condition> <subject>")
		}
		selectors, err := colSelectors("friedman", table, args[1:4])
		if err != nil {
			return err
		}
		result, err := stats.FriedmanTestFromTable(table, selectors[0], selectors[1], selectors[2])
		if err != nil {
			return fmt.Errorf("friedman failed: %w", err)
		}
		printFriedman(ctx, result)
		return nil
	}
	if len(args) < 2 {
		return fmt.Errorf("usage: friedman <subject1> <subject2> [subjectN]")
	}
	subjects, err := getDataListGroups(ctx, args)
	if err != nil {
		return err
	}
	result, err := stats.FriedmanTest(subjects)
	if err != nil {
		return fmt.Errorf("friedman failed: %w", err)
	}
	printFriedman(ctx, result)
	return nil
}

// printFriedman writes a Friedman result the way the command prints it.
func printFriedman(ctx *ExecContext, result *stats.FriedmanTestResult) {
	df := math.NaN()
	if result.DF != nil {
		df = *result.DF
	}
	_, _ = fmt.Fprintf(ctx.Output, "Q=%v df=%v p=%v\n", result.Statistic, df, result.PValue)
}
