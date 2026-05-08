package commands

import (
	"fmt"
	"strings"

	"github.com/HazelnutParadise/insyra/parquet"
)

func init() {
	_ = Register(&CommandHandler{
		Name:        "save",
		Usage:       "save <var> <file> [headers true|false] [rownames true|false] [bom true|false] | save <var> sql <conn> <table> [if-exists fail|replace|append] [batch N] [schema <s>] [rownames [true|false]]",
		Description: "Save a DataTable variable to a file or SQL connection",
		Run:         runSaveCommand,
	})
}

func runSaveCommand(ctx *ExecContext, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: save <var> <file> [headers true|false] [rownames true|false] [bom true|false] | save <var> sql <conn> <table> [...]")
	}
	table, err := getDataTableVar(ctx, args[0])
	if err != nil {
		return err
	}

	if args[1] == "sql" {
		return runSaveSQL(ctx, args[0], table, args[2:])
	}

	path := args[1]
	opts, err := parseFileSaveOptions(args[2:])
	if err != nil {
		return err
	}
	switch detectFileKind(path) {
	case "csv":
		err = table.ToCSV(path, opts.RowNames, opts.Headers, opts.BOM)
	case "json":
		if opts.RowNamesSet || opts.BOMSet {
			return fmt.Errorf("save json: only 'headers' is supported (controls whether values use column names as keys)")
		}
		err = table.ToJSON(path, opts.Headers)
	case "parquet":
		if opts.HeadersSet || opts.RowNamesSet || opts.BOMSet {
			return fmt.Errorf("save parquet: headers/rownames/bom options are not supported")
		}
		err = parquet.Write(table, path)
	default:
		return fmt.Errorf("unsupported output file type: %s", path)
	}
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(ctx.Output, "saved %s to %s\n", args[0], path)
	return nil
}

// fileSaveOptions captures the file-output toggles. The *Set flags let
// format-specific code reject options that don't apply.
type fileSaveOptions struct {
	Headers     bool
	HeadersSet  bool
	RowNames    bool
	RowNamesSet bool
	BOM         bool
	BOMSet      bool
}

func parseFileSaveOptions(args []string) (fileSaveOptions, error) {
	opts := fileSaveOptions{Headers: true, RowNames: false, BOM: false}
	for i := 0; i < len(args); {
		key := strings.ToLower(args[i])
		next := func() (string, error) {
			if i+1 >= len(args) {
				return "", fmt.Errorf("save: option %q requires a value", args[i])
			}
			return args[i+1], nil
		}
		switch key {
		case "headers", "header":
			v, err := next()
			if err != nil {
				return opts, err
			}
			b, err := parseFlexBool(v)
			if err != nil {
				return opts, fmt.Errorf("save: invalid value for headers: %w", err)
			}
			opts.Headers = b
			opts.HeadersSet = true
			i += 2
		case "rownames", "rowname":
			v, err := next()
			if err != nil {
				return opts, err
			}
			b, err := parseFlexBool(v)
			if err != nil {
				return opts, fmt.Errorf("save: invalid value for rownames: %w", err)
			}
			opts.RowNames = b
			opts.RowNamesSet = true
			i += 2
		case "bom":
			v, err := next()
			if err != nil {
				return opts, err
			}
			b, err := parseFlexBool(v)
			if err != nil {
				return opts, fmt.Errorf("save: invalid value for bom: %w", err)
			}
			opts.BOM = b
			opts.BOMSet = true
			i += 2
		default:
			return opts, fmt.Errorf("save: unknown option %q (supported: headers, rownames, bom)", args[i])
		}
	}
	return opts, nil
}
