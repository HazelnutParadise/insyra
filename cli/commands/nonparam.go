package commands

import (
	"fmt"
	"math"
	"strings"

	"github.com/HazelnutParadise/insyra/stats"
)

func init() {
	_ = Register(&CommandHandler{
		Name:        "wilcoxon",
		Args:        FormArgs(map[string]int{"single": 4, "paired": 4}),
		Usage:       "wilcoxon single|paired ...",
		Description: "Wilcoxon signed-rank tests",
		Forms: []string{
			"wilcoxon single <var> <mu> [two-sided|greater|less]        one sample against mu",
			"wilcoxon paired <var1> <var2> [two-sided|greater|less]     paired samples",
		},
		Examples: []string{"insyra wilcoxon single ratings 3", "insyra wilcoxon paired before after less"},
		Run:      runWilcoxonCommand,
	})
	_ = Register(&CommandHandler{
		Name:        "mannwhitney",
		Args:        MaxArgs(3),
		Usage:       "mannwhitney <var1> <var2> [two-sided|greater|less]",
		Description: "Mann-Whitney U test for two independent samples",
		Examples:    []string{"insyra mannwhitney a b", "insyra mannwhitney a b greater"},
		Run:         runMannWhitneyCommand,
	})
	_ = Register(&CommandHandler{
		Name:        "kruskal",
		Args:        OpenArgs(),
		Usage:       "kruskal <group1> <group2> [groupN]",
		Description: "Kruskal-Wallis test across groups",
		Examples:    []string{"insyra kruskal g1 g2 g3"},
		Run:         runKruskalCommand,
	})
}

func runWilcoxonCommand(ctx *ExecContext, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: wilcoxon single|paired")
	}
	mode := strings.ToLower(args[0])
	switch mode {
	case "single":
		if len(args) < 3 {
			return fmt.Errorf("usage: wilcoxon single <var> <mu> [two-sided|greater|less]")
		}
		dl, err := getDataListVar(ctx, args[1])
		if err != nil {
			return err
		}
		mu, err := parseFloatArg("wilcoxon", "mu", args[2])
		if err != nil {
			return err
		}
		alternative := stats.TwoSided
		if len(args) >= 4 {
			var parseErr error
			alternative, parseErr = parseAlternativeHypothesis(args[3])
			if parseErr != nil {
				return fmt.Errorf("%w", parseErr)
			}
		}
		result, err := stats.SingleSampleWilcoxon(dl, mu, stats.WilcoxonOptions{Alternative: alternative})
		if err != nil {
			return fmt.Errorf("wilcoxon failed: %w", err)
		}
		_, _ = fmt.Fprintf(ctx.Output, "W=%v p=%v\n", result.Statistic, result.PValue)
		return nil
	case "paired":
		if len(args) < 3 {
			return fmt.Errorf("usage: wilcoxon paired <var1> <var2> [two-sided|greater|less]")
		}
		dl1, err := getDataListVar(ctx, args[1])
		if err != nil {
			return err
		}
		dl2, err := getDataListVar(ctx, args[2])
		if err != nil {
			return err
		}
		alternative := stats.TwoSided
		if len(args) >= 4 {
			var parseErr error
			alternative, parseErr = parseAlternativeHypothesis(args[3])
			if parseErr != nil {
				return fmt.Errorf("%w", parseErr)
			}
		}
		result, err := stats.PairedWilcoxon(dl1, dl2, stats.WilcoxonOptions{Alternative: alternative})
		if err != nil {
			return fmt.Errorf("wilcoxon failed: %w", err)
		}
		_, _ = fmt.Fprintf(ctx.Output, "W=%v p=%v\n", result.Statistic, result.PValue)
		return nil
	default:
		return fmt.Errorf("unsupported wilcoxon mode: %s", mode)
	}
}

func runMannWhitneyCommand(ctx *ExecContext, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: mannwhitney <var1> <var2> [two-sided|greater|less]")
	}
	a, err := getDataListVar(ctx, args[0])
	if err != nil {
		return err
	}
	b, err := getDataListVar(ctx, args[1])
	if err != nil {
		return err
	}
	alternative := stats.TwoSided
	if len(args) >= 3 {
		var parseErr error
		alternative, parseErr = parseAlternativeHypothesis(args[2])
		if parseErr != nil {
			return fmt.Errorf("%w", parseErr)
		}
	}
	result, err := stats.MannWhitneyU(a, b, stats.MannWhitneyUOptions{Alternative: alternative})
	if err != nil {
		return fmt.Errorf("mannwhitney failed: %w", err)
	}
	_, _ = fmt.Fprintf(ctx.Output, "U=%v p=%v\n", result.Statistic, result.PValue)
	return nil
}

func runKruskalCommand(ctx *ExecContext, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: kruskal <group1> <group2> [groupN]")
	}
	groups, err := getDataListGroups(ctx, args)
	if err != nil {
		return err
	}
	result, err := stats.KruskalWallis(groups)
	if err != nil {
		return fmt.Errorf("kruskal failed: %w", err)
	}
	df := math.NaN()
	if result.DF != nil {
		df = *result.DF
	}
	_, _ = fmt.Fprintf(ctx.Output, "H=%v df=%v p=%v\n", result.Statistic, df, result.PValue)
	return nil
}
