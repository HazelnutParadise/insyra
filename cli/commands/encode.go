package commands

import (
	"fmt"
	"strings"

	insyra "github.com/HazelnutParadise/insyra"
)

func init() {
	_ = Register(&CommandHandler{
		Name:        "encode",
		Usage:       "encode <var> onehot|label|ordinal ... [as <var>]",
		Description: "One-shot categorical encoding for DataTable variables (encoder state is not persisted)",
		Forms: []string{
			"encode <var> onehot <col1[,col2,...]> [dropfirst true|false] [keeporiginal true|false] [nan category|error|skip] [unknown ignore|error|new] [prefix <p>] [sep <s>] [sortcats true|false] [as <var>]",
			"encode <var> label <col> [newcol <name>] [sortby firstseen|lex|freq] [nan category|error|skip] [unknown ignore|error|new] [keeporiginal true|false] [as <var>]",
			"encode <var> ordinal <col> order <v1,v2,...> [newcol <name>] [unknown error|ignore] [nan category|error|skip] [keeporiginal true|false] [as <var>]",
		},
		Examples: []string{
			"insyra encode sales onehot region,channel dropfirst true as x",
			"insyra encode sales label segment newcol segment_id sortby freq keeporiginal true as labeled",
			"insyra encode survey ordinal satisfaction order low,medium,high unknown error as ranked",
		},
		Run: runEncodeCommand,
	})
}

func runEncodeCommand(ctx *ExecContext, args []string) error {
	coreArgs, alias := parseAlias(args)
	if len(coreArgs) < 3 {
		return fmt.Errorf("usage: encode <var> onehot|label|ordinal ... [as <var>]")
	}
	table, err := getDataTableVar(ctx, coreArgs[0])
	if err != nil {
		return err
	}
	mode := strings.ToLower(coreArgs[1])
	var result *insyra.DataTable
	switch mode {
	case "onehot":
		opts, err := parseOneHotEncodeOptions(coreArgs[2:])
		if err != nil {
			return err
		}
		result, _, err = table.OneHotEncode(opts)
		if err != nil {
			return err
		}
	case "label":
		opts, err := parseLabelEncodeOptions(coreArgs[2:])
		if err != nil {
			return err
		}
		result, _, err = table.LabelEncode(opts)
		if err != nil {
			return err
		}
	case "ordinal":
		opts, err := parseOrdinalEncodeOptions(coreArgs[2:])
		if err != nil {
			return err
		}
		result, _, err = table.OrdinalEncode(opts)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("encode: unknown mode %q (supported: onehot, label, ordinal)", coreArgs[1])
	}
	ctx.Vars[alias] = result
	_, _ = fmt.Fprintf(ctx.Output, "encoded into %s (%d rows, %d cols)\n", alias, result.NumRows(), result.NumCols())
	return nil
}

func parseOneHotEncodeOptions(args []string) (insyra.OneHotOptions, error) {
	var opts insyra.OneHotOptions
	if len(args) == 0 {
		return opts, fmt.Errorf("encode onehot: columns are required")
	}
	opts.Columns = parseCSVTokens(args[0])
	if len(opts.Columns) == 0 {
		return opts, fmt.Errorf("encode onehot: columns are required")
	}
	for i := 1; i < len(args); {
		key := strings.ToLower(args[i])
		next := func() (string, error) {
			if i+1 >= len(args) {
				return "", fmt.Errorf("encode onehot: option %q requires a value", args[i])
			}
			return args[i+1], nil
		}
		switch key {
		case "dropfirst":
			v, err := next()
			if err != nil {
				return opts, err
			}
			b, err := parseFlexBool(v)
			if err != nil {
				return opts, fmt.Errorf("encode onehot: invalid value for dropfirst: %w", err)
			}
			opts.DropFirst = b
			i += 2
		case "keeporiginal":
			v, err := next()
			if err != nil {
				return opts, err
			}
			b, err := parseFlexBool(v)
			if err != nil {
				return opts, fmt.Errorf("encode onehot: invalid value for keeporiginal: %w", err)
			}
			opts.KeepOriginal = b
			i += 2
		case "nan":
			v, err := next()
			if err != nil {
				return opts, err
			}
			policy, err := parseEncodeNaNPolicy(v, "encode onehot")
			if err != nil {
				return opts, err
			}
			opts.HandleNaN = policy
			i += 2
		case "unknown":
			v, err := next()
			if err != nil {
				return opts, err
			}
			policy, err := parseEncodeUnknownPolicy(v, "encode onehot", true)
			if err != nil {
				return opts, err
			}
			opts.Unknown = policy
			i += 2
		case "prefix":
			v, err := next()
			if err != nil {
				return opts, err
			}
			opts.Prefix = v
			i += 2
		case "sep":
			v, err := next()
			if err != nil {
				return opts, err
			}
			opts.Separator = v
			i += 2
		case "sortcats":
			v, err := next()
			if err != nil {
				return opts, err
			}
			b, err := parseFlexBool(v)
			if err != nil {
				return opts, fmt.Errorf("encode onehot: invalid value for sortcats: %w", err)
			}
			opts.SortCategories = b
			i += 2
		default:
			return opts, fmt.Errorf("encode onehot: unknown option %q (supported: dropfirst, keeporiginal, nan, unknown, prefix, sep, sortcats)", args[i])
		}
	}
	return opts, nil
}

func parseLabelEncodeOptions(args []string) (insyra.LabelEncodeOptions, error) {
	var opts insyra.LabelEncodeOptions
	if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
		return opts, fmt.Errorf("encode label: column is required")
	}
	opts.Column = args[0]
	for i := 1; i < len(args); {
		key := strings.ToLower(args[i])
		next := func() (string, error) {
			if i+1 >= len(args) {
				return "", fmt.Errorf("encode label: option %q requires a value", args[i])
			}
			return args[i+1], nil
		}
		switch key {
		case "newcol":
			v, err := next()
			if err != nil {
				return opts, err
			}
			opts.NewColumn = v
			i += 2
		case "sortby":
			v, err := next()
			if err != nil {
				return opts, err
			}
			sortBy, err := parseEncodeLabelSort(v)
			if err != nil {
				return opts, err
			}
			opts.SortBy = sortBy
			i += 2
		case "nan":
			v, err := next()
			if err != nil {
				return opts, err
			}
			policy, err := parseEncodeNaNPolicy(v, "encode label")
			if err != nil {
				return opts, err
			}
			opts.HandleNaN = policy
			i += 2
		case "unknown":
			v, err := next()
			if err != nil {
				return opts, err
			}
			policy, err := parseEncodeUnknownPolicy(v, "encode label", true)
			if err != nil {
				return opts, err
			}
			opts.Unknown = policy
			i += 2
		case "keeporiginal":
			v, err := next()
			if err != nil {
				return opts, err
			}
			b, err := parseFlexBool(v)
			if err != nil {
				return opts, fmt.Errorf("encode label: invalid value for keeporiginal: %w", err)
			}
			opts.KeepOriginal = b
			i += 2
		default:
			return opts, fmt.Errorf("encode label: unknown option %q (supported: newcol, sortby, nan, unknown, keeporiginal)", args[i])
		}
	}
	return opts, nil
}

func parseOrdinalEncodeOptions(args []string) (insyra.OrdinalEncodeOptions, error) {
	var opts insyra.OrdinalEncodeOptions
	if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
		return opts, fmt.Errorf("encode ordinal: column is required")
	}
	opts.Column = args[0]
	for i := 1; i < len(args); {
		key := strings.ToLower(args[i])
		next := func() (string, error) {
			if i+1 >= len(args) {
				return "", fmt.Errorf("encode ordinal: option %q requires a value", args[i])
			}
			return args[i+1], nil
		}
		switch key {
		case "order":
			v, err := next()
			if err != nil {
				return opts, err
			}
			tokens := parseCSVTokens(v)
			if len(tokens) == 0 {
				return opts, fmt.Errorf("encode ordinal: order requires at least one value")
			}
			opts.Order = make([]any, 0, len(tokens))
			for _, token := range tokens {
				opts.Order = append(opts.Order, parseLiteral(token))
			}
			i += 2
		case "newcol":
			v, err := next()
			if err != nil {
				return opts, err
			}
			opts.NewColumn = v
			i += 2
		case "unknown":
			v, err := next()
			if err != nil {
				return opts, err
			}
			policy, err := parseEncodeUnknownPolicy(v, "encode ordinal", false)
			if err != nil {
				return opts, err
			}
			opts.Unknown = policy
			i += 2
		case "nan":
			v, err := next()
			if err != nil {
				return opts, err
			}
			policy, err := parseEncodeNaNPolicy(v, "encode ordinal")
			if err != nil {
				return opts, err
			}
			opts.HandleNaN = policy
			i += 2
		case "keeporiginal":
			v, err := next()
			if err != nil {
				return opts, err
			}
			b, err := parseFlexBool(v)
			if err != nil {
				return opts, fmt.Errorf("encode ordinal: invalid value for keeporiginal: %w", err)
			}
			opts.KeepOriginal = b
			i += 2
		default:
			return opts, fmt.Errorf("encode ordinal: unknown option %q (supported: order, newcol, unknown, nan, keeporiginal)", args[i])
		}
	}
	if len(opts.Order) == 0 {
		return opts, fmt.Errorf("encode ordinal: order is required (use 'order <v1,v2,...>')")
	}
	return opts, nil
}

func parseEncodeNaNPolicy(raw, context string) (insyra.NaNPolicy, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "category":
		return insyra.NaNAsCategory, nil
	case "error":
		return insyra.NaNError, nil
	case "skip":
		return insyra.NaNSkip, nil
	}
	return insyra.NaNAsCategory, fmt.Errorf("%s: invalid value for nan %q (supported: category, error, skip)", context, raw)
}

func parseEncodeUnknownPolicy(raw, context string, allowNew bool) (insyra.UnknownPolicy, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "ignore":
		return insyra.UnknownIgnore, nil
	case "error":
		return insyra.UnknownError, nil
	case "new":
		if allowNew {
			return insyra.UnknownAsNew, nil
		}
	}
	supported := "ignore, error"
	if allowNew {
		supported = "ignore, error, new"
	}
	return insyra.UnknownIgnore, fmt.Errorf("%s: invalid value for unknown %q (supported: %s)", context, raw, supported)
}

func parseEncodeLabelSort(raw string) (insyra.LabelSort, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "firstseen":
		return insyra.LabelSortFirstSeen, nil
	case "lex":
		return insyra.LabelSortLexicographic, nil
	case "freq":
		return insyra.LabelSortByFrequency, nil
	}
	return insyra.LabelSortFirstSeen, fmt.Errorf("encode label: invalid value for sortby %q (supported: firstseen, lex, freq)", raw)
}
