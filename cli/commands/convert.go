package commands

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/HazelnutParadise/insyra/csvxl"
)

const convertUsage = "convert <input> <output> | convert <file.xlsx> sheet <name> <file.csv> [sheet <name> <file.csv> ...] [allowformulas true|false]"

func init() {
	_ = Register(&CommandHandler{
		Name:        "convert",
		Usage:       convertUsage,
		Description: "Convert file formats (csv<->xlsx)",
		Forms: []string{
			"convert <file.csv> <file.xlsx>",
			"convert <file.xlsx> <file.csv>",
			"convert <file.xlsx> sheet <name> <file.csv> [sheet <name> <file.csv> ...] [allowformulas true|false]",
			"",
			"<file.xlsx> <file.csv> writes the first sheet to <file.csv> and every other sheet as <sheet>.csv beside it, as written.",
			"The sheet form writes only the sheets it names, each to its own file; `sheets <file.xlsx>` lists them. An unknown sheet, a sheet given twice, or two sheets for one file is an error, and nothing is written.",
			"The sheet form guards text a spreadsheet would run as a formula (=, +, -, @) with a leading quote; allowformulas true writes it as is.",
		},
		Examples: []string{
			"insyra convert sales.csv sales.xlsx",
			"insyra convert book.xlsx sheet Q1 q1.csv sheet \"2025 Q2\" q2.csv",
		},
		Run: runConvertCommand,
	})
	_ = Register(&CommandHandler{
		Name:        "sheets",
		Usage:       "sheets <file.xlsx>",
		Description: "List the sheets of an Excel workbook, one per line, in workbook order",
		Examples: []string{
			"insyra sheets book.xlsx",
		},
		Run: runSheetsCommand,
	})
}

// sheetFile is one `sheet <name> <file.csv>` pair of convert.
type sheetFile struct {
	sheet string
	file  string
}

func runConvertCommand(ctx *ExecContext, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: %s", convertUsage)
	}
	input := args[0]
	output := args[1]
	inExt := strings.ToLower(filepath.Ext(input))
	isExcel := inExt == ".xlsx" || inExt == ".xlsm"
	if isExcel && strings.EqualFold(output, "sheet") {
		return runConvertSheetPairs(ctx, input, args[1:])
	}
	outExt := strings.ToLower(filepath.Ext(output))

	switch {
	case inExt == ".csv" && (outExt == ".xlsx" || outExt == ".xlsm"):
		if err := csvxl.CSVToExcel([]csvxl.SheetFromCSV{{Path: input}}, output); err != nil {
			return err
		}
	case isExcel && outExt == ".csv":
		// convert keeps ExcelToCsv's output naming and unguarded text, which
		// ExcelToCSV does not share, until the old function is removed.
		if err := csvxl.ExcelToCsv(input, filepath.Dir(output), []string{filepath.Base(output)}); err != nil { //nolint:staticcheck
			return err
		}
	default:
		return fmt.Errorf("unsupported conversion: %s -> %s", inExt, outExt)
	}

	_, _ = fmt.Fprintf(ctx.Output, "converted %s to %s\n", input, output)
	return nil
}

// runConvertSheetPairs reads the `sheet <name> <file.csv>` pairs and the
// optional `allowformulas true|false` that follow the workbook.
func runConvertSheetPairs(ctx *ExecContext, input string, rest []string) error {
	var pairs []sheetFile
	for len(rest) > 0 && strings.EqualFold(rest[0], "sheet") {
		if len(rest) < 3 {
			return fmt.Errorf("convert: each `sheet` needs a sheet name and a CSV file: sheet <name> <file.csv>")
		}
		pairs = append(pairs, sheetFile{sheet: rest[1], file: rest[2]})
		rest = rest[3:]
	}
	allowFormulas := false
	for ; len(rest) > 0; rest = rest[2:] {
		if !strings.EqualFold(rest[0], "allowformulas") {
			return fmt.Errorf("convert: unknown argument %q (supported: allowformulas true|false)", rest[0])
		}
		if len(rest) < 2 {
			return fmt.Errorf("convert: 'allowformulas' requires true or false")
		}
		b, err := parseFlexBool(rest[1])
		if err != nil {
			return fmt.Errorf("convert: invalid value for allowformulas: %w", err)
		}
		allowFormulas = b
	}
	return convertSheetPairs(ctx, input, pairs, allowFormulas)
}

// convertSheetPairs writes each named sheet of the workbook to its CSV file.
// Every pair is checked before anything is written: the sheet must exist and
// be named once, and no two sheets may go to one file (compared without
// regard to case, as macOS and Windows compare file names). The files are
// written one output directory at a time.
func convertSheetPairs(ctx *ExecContext, input string, pairs []sheetFile, allowFormulas bool) error {
	sheets, err := csvxl.SheetNames(input)
	if err != nil {
		return err
	}
	type dirJob struct {
		dir   string
		opts  csvxl.ExcelToCSVOptions
		pairs []sheetFile
	}
	var jobs []*dirJob
	byDir := map[string]*dirJob{}
	sheetSeen := map[string]bool{}
	fileSeen := map[string]sheetFile{}
	for _, p := range pairs {
		if !slices.Contains(sheets, p.sheet) {
			return fmt.Errorf("convert: %s has no sheet %q (it has %s)", input, p.sheet, strings.Join(sheets, ", "))
		}
		if sheetSeen[p.sheet] {
			return fmt.Errorf("convert: sheet %q is given twice", p.sheet)
		}
		sheetSeen[p.sheet] = true
		if !strings.EqualFold(filepath.Ext(p.file), ".csv") {
			return fmt.Errorf("convert: %s is not a .csv file", p.file)
		}
		abs, err := filepath.Abs(p.file)
		if err != nil {
			return fmt.Errorf("convert: %s: %w", p.file, err)
		}
		key := strings.ToLower(abs)
		if prev, ok := fileSeen[key]; ok {
			if prev.file == p.file {
				return fmt.Errorf("convert: sheets %q and %q would both be written to %s", prev.sheet, p.sheet, p.file)
			}
			return fmt.Errorf("convert: sheets %q and %q would both be written to one file: %s and %s differ only in case", prev.sheet, p.sheet, prev.file, p.file)
		}
		fileSeen[key] = p

		dir := filepath.Dir(abs)
		job := byDir[dir]
		if job == nil {
			job = &dirJob{dir: dir, opts: csvxl.ExcelToCSVOptions{FileNames: map[string]string{}, AllowFormulas: allowFormulas}}
			byDir[dir] = job
			jobs = append(jobs, job)
		}
		job.opts.Sheets = append(job.opts.Sheets, p.sheet)
		job.opts.FileNames[p.sheet] = filepath.Base(abs)
		job.pairs = append(job.pairs, p)
	}
	for _, job := range jobs {
		if err := csvxl.ExcelToCSV(input, job.dir, job.opts); err != nil {
			return err
		}
		for _, p := range job.pairs {
			_, _ = fmt.Fprintf(ctx.Output, "converted %s sheet %s to %s\n", input, p.sheet, p.file)
		}
	}
	return nil
}

func runSheetsCommand(ctx *ExecContext, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: sheets <file.xlsx>")
	}
	sheets, err := csvxl.SheetNames(args[0])
	if err != nil {
		return err
	}
	for _, s := range sheets {
		_, _ = fmt.Fprintln(ctx.Output, s)
	}
	return nil
}
