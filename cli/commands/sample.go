package commands

import (
	"fmt"
	"strconv"
	"strings"

	insyra "github.com/HazelnutParadise/insyra"
)

func init() {
	_ = Register(&CommandHandler{
		Name:        "sample",
		Usage:       "sample <var> <n>|frac <frac>|shuffle [replace true|false] [seed N] [as <var>]",
		Description: "Randomly sample or shuffle a DataList/DataTable",
		Forms: []string{
			"sample <var> <n> [replace true|false] [seed N] [as <var>]",
			"sample <var> frac <frac> [replace true|false] [seed N] [as <var>]",
			"sample <var> shuffle [seed N] [as <var>]",
		},
		Examples: []string{
			"insyra sample t 100 as t_sample",
			"insyra sample t frac 0.1 seed 42 as preview",
			"insyra sample x shuffle seed 42 as x_shuffled",
		},
		Run: runSampleCommand,
	})
}

type sampleCommandOptions struct {
	Replace bool
	Seed    uint64
	UseSeed bool
}

func runSampleCommand(ctx *ExecContext, args []string) error {
	coreArgs, alias := parseAlias(args)
	if len(coreArgs) < 2 {
		return fmt.Errorf("usage: sample <var> <n>|frac <frac>|shuffle [replace true|false] [seed N] [as <var>]")
	}

	varName := coreArgs[0]
	mode := strings.ToLower(coreArgs[1])
	var result any
	var err error

	switch mode {
	case "frac":
		if len(coreArgs) < 3 {
			return fmt.Errorf("sample: frac requires a value")
		}
		frac, parseErr := strconv.ParseFloat(coreArgs[2], 64)
		if parseErr != nil {
			return fmt.Errorf("invalid sample fraction: %s", coreArgs[2])
		}
		opts, parseErr := parseSampleCommandOptions(coreArgs[3:], true)
		if parseErr != nil {
			return parseErr
		}
		result, err = applySampleFrac(ctx, varName, frac, opts)
	case "shuffle":
		opts, parseErr := parseSampleCommandOptions(coreArgs[2:], false)
		if parseErr != nil {
			return parseErr
		}
		result, err = applyShuffle(ctx, varName, opts)
	default:
		n, parseErr := strconv.Atoi(coreArgs[1])
		if parseErr != nil {
			return fmt.Errorf("invalid sample size: %s", coreArgs[1])
		}
		opts, parseErr := parseSampleCommandOptions(coreArgs[2:], true)
		if parseErr != nil {
			return parseErr
		}
		result, err = applySampleN(ctx, varName, n, opts)
	}
	if err != nil {
		return err
	}
	ctx.Vars[alias] = result
	_, _ = fmt.Fprintf(ctx.Output, "saved sample as %s\n", alias)
	return nil
}

func parseSampleCommandOptions(args []string, allowReplace bool) (sampleCommandOptions, error) {
	var opts sampleCommandOptions
	for i := 0; i < len(args); {
		key := strings.ToLower(args[i])
		if i+1 >= len(args) {
			return opts, fmt.Errorf("sample: option %q requires a value", args[i])
		}
		value := args[i+1]
		switch key {
		case "replace":
			if !allowReplace {
				return opts, fmt.Errorf("sample: replace is not supported for shuffle")
			}
			parsed, err := parseFlexBool(value)
			if err != nil {
				return opts, fmt.Errorf("sample: invalid value for replace: %w", err)
			}
			opts.Replace = parsed
		case "seed":
			seed, err := strconv.ParseUint(value, 10, 64)
			if err != nil {
				return opts, fmt.Errorf("sample: invalid seed %q", value)
			}
			opts.Seed = seed
			opts.UseSeed = true
		default:
			return opts, fmt.Errorf("sample: unknown option %q (supported: replace, seed)", args[i])
		}
		i += 2
	}
	return opts, nil
}

func samplingOptionsFromCLI(opts sampleCommandOptions) []insyra.SamplingOptions {
	if !opts.UseSeed {
		return nil
	}
	return []insyra.SamplingOptions{{Seed: opts.Seed, UseSeed: true}}
}

func applySampleN(ctx *ExecContext, varName string, n int, opts sampleCommandOptions) (any, error) {
	value, exists := ctx.Vars[varName]
	if !exists {
		return nil, fmt.Errorf("variable not found: %s", varName)
	}
	samplingOpts := samplingOptionsFromCLI(opts)
	switch v := value.(type) {
	case *insyra.DataTable:
		if !opts.Replace && !opts.UseSeed {
			return v.SimpleRandomSample(n), nil
		}
		return v.Sample(n, opts.Replace, samplingOpts...), nil
	case *insyra.DataList:
		return v.Sample(n, opts.Replace, samplingOpts...), nil
	default:
		return nil, fmt.Errorf("variable %s is not a DataList or DataTable", varName)
	}
}

func applySampleFrac(ctx *ExecContext, varName string, frac float64, opts sampleCommandOptions) (any, error) {
	value, exists := ctx.Vars[varName]
	if !exists {
		return nil, fmt.Errorf("variable not found: %s", varName)
	}
	samplingOpts := samplingOptionsFromCLI(opts)
	switch v := value.(type) {
	case *insyra.DataTable:
		return v.SampleFrac(frac, opts.Replace, samplingOpts...), nil
	case *insyra.DataList:
		return v.SampleFrac(frac, opts.Replace, samplingOpts...), nil
	default:
		return nil, fmt.Errorf("variable %s is not a DataList or DataTable", varName)
	}
}

func applyShuffle(ctx *ExecContext, varName string, opts sampleCommandOptions) (any, error) {
	value, exists := ctx.Vars[varName]
	if !exists {
		return nil, fmt.Errorf("variable not found: %s", varName)
	}
	samplingOpts := samplingOptionsFromCLI(opts)
	switch v := value.(type) {
	case *insyra.DataTable:
		return v.Shuffle(samplingOpts...), nil
	case *insyra.DataList:
		return v.Shuffle(samplingOpts...), nil
	default:
		return nil, fmt.Errorf("variable %s is not a DataList or DataTable", varName)
	}
}
