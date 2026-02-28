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
		Usage:       "convert <input> <output>",
		Description: "Convert file formats (csv<->xlsx)",
		Run:         runConvertCommand,
	})
}

func runConvertCommand(ctx *ExecContext, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: convert <input> <output>")
	}
	input := args[0]
	output := args[1]
	inExt := strings.ToLower(filepath.Ext(input))
	outExt := strings.ToLower(filepath.Ext(output))

	switch {
	case inExt == ".csv" && (outExt == ".xlsx" || outExt == ".xlsm"):
		if err := csvxl.CsvToExcel([]string{input}, nil, output); err != nil {
			return err
		}
	case (inExt == ".xlsx" || inExt == ".xlsm") && outExt == ".csv":
		if err := csvxl.ExcelToCsv(input, filepath.Dir(output), []string{filepath.Base(output)}); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported conversion: %s -> %s", inExt, outExt)
	}

	_, _ = fmt.Fprintf(ctx.Output, "converted %s to %s\n", input, output)
	return nil
}
