package commands

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/parquet"
)

func init() {
	_ = Register(&CommandHandler{
		Name:        "save",
		Usage:       "save <var> <file> [headers true|false] [rownames true|false] [bom true|false] [allowformulas true|false] [sheet <name>] [if-exists fail|replace] | save <var> sql <conn> <table> [if-exists fail|replace|append] [batch N] [schema <s>] [rownames [true|false]]",
		Description: "Save a DataTable variable to a file or SQL connection",
		Forms: []string{
			"save <var> <file.csv> [headers true|false] [rownames true|false] [bom true|false] [allowformulas true|false]",
			"save <var> <file.json> [headers true|false]",
			"save <var> <file.xlsx> [sheet <name>] [if-exists fail|replace] [headers true|false] [rownames true|false]",
			"save <var> <file.parquet>",
			"save <var> sql <conn> <table> [if-exists fail|replace|append] [batch N] [schema <s>] [rownames [true|false]]",
			"",
			"File option defaults: headers=true, rownames=false, bom=false, allowformulas=false.",
			"CSV guards text a spreadsheet would run as a formula (=, +, -, @) with a leading quote; allowformulas true writes it as is.",
			"Excel writes one sheet (default Sheet1) and leaves the workbook's other sheets alone.",
			"Excel if-exists default: fail, so an existing sheet is only overwritten with 'if-exists replace'.",
			"Booleans accept true|false|yes|no|on|off|1|0.",
			"SQL if-exists default: fail.",
		},
		Examples: []string{
			"insyra save report data.csv",
			"insyra save matrix data.csv headers false",
			"insyra save gdp out.csv rownames true",
			"insyra save report data.csv bom true",
			"insyra save sales2025 report.xlsx sheet 2025",
			"insyra save sales2025 report.xlsx sheet 2025 if-exists replace",
			"insyra save report sql main report_table if-exists replace batch 1000",
		},
		Run: runSaveCommand,
	})
}

func runSaveCommand(ctx *ExecContext, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: save <var> <file> [headers true|false] [rownames true|false] [bom true|false] [allowformulas true|false] [sheet <name>] [if-exists fail|replace] | save <var> sql <conn> <table> [...]")
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
	kind := detectFileKind(path)
	if kind != "excel" && (opts.SheetSet || opts.IfExistsSet) {
		return fmt.Errorf("save: 'sheet' and 'if-exists' only apply to Excel files (.xlsx)")
	}
	if kind != "csv" && opts.AllowFormulasSet {
		return fmt.Errorf("save: 'allowformulas' only applies to CSV files")
	}
	switch kind {
	case "csv":
		err = table.ToCSV(path, insyra.CSVWriteOptions{NoHeaderRow: !opts.Headers, HasRowNames: opts.RowNames, IncludeBOM: opts.BOM, AllowFormulas: opts.AllowFormulas})
	case "json":
		if opts.RowNamesSet || opts.BOMSet {
			return fmt.Errorf("save json: only 'headers' is supported (controls whether values use column names as keys)")
		}
		err = table.ToJSON(path, opts.Headers)
	case "excel":
		err = saveExcel(table, path, opts)
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

// saveExcel writes the table as one sheet of the workbook at path. Only the
// named sheet is touched, and an existing one is overwritten only when the
// caller said if-exists replace.
func saveExcel(table *insyra.DataTable, path string, opts fileSaveOptions) error {
	if strings.EqualFold(filepath.Ext(path), ".xls") {
		return fmt.Errorf("save excel: .xls is the legacy binary format and cannot be written; save as .xlsx instead")
	}
	if opts.BOMSet {
		return fmt.Errorf("save excel: 'bom' only applies to CSV files")
	}
	err := table.ToExcel(path, insyra.ExcelWriteOptions{
		Sheet:         opts.Sheet,
		NoHeaderRow:   !opts.Headers,
		HasRowNames:   opts.RowNames,
		IfSheetExists: opts.IfExists,
	})
	if errors.Is(err, insyra.ErrSheetExists) {
		if !opts.SheetSet {
			// No name was given, so a new sheet may be what was meant.
			return fmt.Errorf("save excel: %w; add \"sheet <name>\" to save it as a new sheet, or \"if-exists replace\" to overwrite that sheet (the other sheets are kept)", err)
		}
		return fmt.Errorf("save excel: %w; add \"if-exists replace\" to overwrite that sheet (the other sheets are kept)", err)
	}
	return err
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
	// AllowFormulas turns off the CSV formula guard, for a file read back by
	// a program rather than opened in a spreadsheet.
	AllowFormulas    bool
	AllowFormulasSet bool
	Sheet            string
	SheetSet         bool
	IfExists         insyra.SheetExistsPolicy
	IfExistsSet      bool
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
		case "allowformulas":
			v, err := next()
			if err != nil {
				return opts, err
			}
			b, err := parseFlexBool(v)
			if err != nil {
				return opts, fmt.Errorf("save: invalid value for allowformulas: %w", err)
			}
			opts.AllowFormulas = b
			opts.AllowFormulasSet = true
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
		case "sheet":
			v, err := next()
			if err != nil {
				return opts, err
			}
			opts.Sheet = v
			opts.SheetSet = true
			i += 2
		case "if-exists", "ifexists":
			v, err := next()
			if err != nil {
				return opts, err
			}
			switch strings.ToLower(v) {
			case "fail":
				opts.IfExists = insyra.SheetExistsFail
			case "replace":
				opts.IfExists = insyra.SheetExistsReplace
			default:
				return opts, fmt.Errorf("save: invalid if-exists %q (expected fail|replace)", v)
			}
			opts.IfExistsSet = true
			i += 2
		default:
			return opts, fmt.Errorf("save: unknown option %q (supported: headers, rownames, bom, allowformulas, sheet, if-exists)", args[i])
		}
	}
	return opts, nil
}
