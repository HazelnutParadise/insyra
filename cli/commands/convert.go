package commands

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/HazelnutParadise/insyra/csvxl"
)

func init() {
	_ = Register(&CommandHandler{
		Name:        "convert",
		Args:        OpenArgs(),
		Usage:       "convert <input> <output> [allowformulas true|false]",
		Description: "Convert file formats (csv<->xlsx)",
		Forms: []string{
			"convert <file.csv> <file.xlsx>",
			"convert <file.xlsx> <file.csv> [allowformulas true|false]",
			"",
			"xlsx->csv guards text a spreadsheet would run as a formula (=, +, -, @) with a leading quote; allowformulas true writes it as is.",
		},
		Run: runConvertCommand,
	})
}

func runConvertCommand(ctx *ExecContext, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: convert <input> <output> [allowformulas true|false]")
	}
	input := args[0]
	output := args[1]
	allowFormulas := false
	allowFormulasSet := false
	for rest := args[2:]; len(rest) > 0; rest = rest[2:] {
		if !strings.EqualFold(rest[0], "allowformulas") {
			return fmt.Errorf("convert: unknown argument %q (supported: allowformulas true|false)", rest[0])
		}
		if len(rest) < 2 {
			return fmt.Errorf("convert: option %q requires a value", rest[0])
		}
		b, err := parseFlexBool(rest[1])
		if err != nil {
			return fmt.Errorf("convert: invalid value for allowformulas: %w", err)
		}
		allowFormulas, allowFormulasSet = b, true
	}
	inExt := strings.ToLower(filepath.Ext(input))
	outExt := strings.ToLower(filepath.Ext(output))

	switch {
	case inExt == ".csv" && (outExt == ".xlsx" || outExt == ".xlsm"):
		if allowFormulasSet {
			return fmt.Errorf("convert: 'allowformulas' only applies to xlsx->csv")
		}
		if err := csvxl.CsvToExcel([]string{input}, nil, output); err != nil {
			return err
		}
	case (inExt == ".xlsx" || inExt == ".xlsm") && outExt == ".csv":
		if err := csvxl.ExcelToCsv(input, filepath.Dir(output), []string{filepath.Base(output)}, csvxl.ExcelToCsvOptions{AllowFormulas: allowFormulas}); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported conversion: %s -> %s", inExt, outExt)
	}

	_, _ = fmt.Fprintf(ctx.Output, "converted %s to %s\n", input, output)
	return nil
}
