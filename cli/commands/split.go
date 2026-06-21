package commands

import (
	"fmt"
	"strconv"
	"strings"

	insyra "github.com/HazelnutParadise/insyra"
)

func init() {
	_ = Register(&CommandHandler{
		Name:        "split",
		Usage:       "split <var> train <frac> [shuffle true|false] [seed N] as <trainVar> <testVar>",
		Description: "Split a DataTable into train and test tables",
		Forms: []string{
			"split <var> train <frac> [shuffle true|false] [seed N] as <trainVar> <testVar>",
		},
		Examples: []string{
			"insyra split t train 0.8 seed 42 as train test",
			"insyra split t train 0.8 shuffle false as train test",
		},
		Run: runSplitCommand,
	})
}

type splitCommandOptions struct {
	Shuffle bool
	Seed    uint64
	UseSeed bool
}

func runSplitCommand(ctx *ExecContext, args []string) error {
	coreArgs, trainAlias, testAlias, err := parseSplitAliases(args)
	if err != nil {
		return err
	}
	if len(coreArgs) < 3 {
		return fmt.Errorf("usage: split <var> train <frac> [shuffle true|false] [seed N] as <trainVar> <testVar>")
	}
	if !strings.EqualFold(coreArgs[1], "train") {
		return fmt.Errorf("split: expected 'train <frac>'")
	}
	trainFrac, err := strconv.ParseFloat(coreArgs[2], 64)
	if err != nil {
		return fmt.Errorf("split: invalid train fraction %q", coreArgs[2])
	}
	if trainFrac <= 0 || trainFrac >= 1 {
		return fmt.Errorf("split: train fraction must be in (0, 1), got %v", trainFrac)
	}
	opts, err := parseSplitCommandOptions(coreArgs[3:])
	if err != nil {
		return err
	}

	table, err := getDataTableVar(ctx, coreArgs[0])
	if err != nil {
		return err
	}
	samplingOpts := insyra.SamplingOptions{
		Seed:          opts.Seed,
		UseSeed:       opts.UseSeed,
		PreserveOrder: !opts.Shuffle,
	}
	train, test := table.TrainTestSplit(trainFrac, samplingOpts)
	ctx.Vars[trainAlias] = train
	ctx.Vars[testAlias] = test
	_, _ = fmt.Fprintf(ctx.Output, "split into %s (%d rows) and %s (%d rows)\n", trainAlias, train.NumRows(), testAlias, test.NumRows())
	return nil
}

func parseSplitAliases(args []string) (coreArgs []string, trainAlias string, testAlias string, err error) {
	if len(args) < 4 {
		return nil, "", "", fmt.Errorf("usage: split <var> train <frac> [shuffle true|false] [seed N] as <trainVar> <testVar>")
	}
	asIndex := -1
	for i := len(args) - 1; i >= 0; i-- {
		if strings.EqualFold(args[i], "as") {
			asIndex = i
			break
		}
	}
	if asIndex < 0 || len(args)-asIndex != 3 {
		return nil, "", "", fmt.Errorf("split: expected 'as <trainVar> <testVar>'")
	}
	return args[:asIndex], args[asIndex+1], args[asIndex+2], nil
}

func parseSplitCommandOptions(args []string) (splitCommandOptions, error) {
	opts := splitCommandOptions{Shuffle: true}
	for i := 0; i < len(args); {
		key := strings.ToLower(args[i])
		if i+1 >= len(args) {
			return opts, fmt.Errorf("split: option %q requires a value", args[i])
		}
		value := args[i+1]
		switch key {
		case "shuffle":
			parsed, err := parseFlexBool(value)
			if err != nil {
				return opts, fmt.Errorf("split: invalid value for shuffle: %w", err)
			}
			opts.Shuffle = parsed
		case "seed":
			seed, err := strconv.ParseUint(value, 10, 64)
			if err != nil {
				return opts, fmt.Errorf("split: invalid seed %q", value)
			}
			opts.Seed = seed
			opts.UseSeed = true
		default:
			return opts, fmt.Errorf("split: unknown option %q (supported: shuffle, seed)", args[i])
		}
		i += 2
	}
	return opts, nil
}
