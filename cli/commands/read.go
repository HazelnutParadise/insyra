package commands

import "fmt"

func init() {
	_ = Register(&CommandHandler{
		Name:        "read",
		Usage:       "read <file> [headers true|false] [rownames true|false] [encoding <enc>] [infer true|false] [ragged true|false] [trimspace true|false] [sheet <name>]",
		Description: "Quick preview a file without saving variable",
		Forms: []string{
			"read <file>                       preview with the format taken from the extension",
			"",
			"headers true|false                treat the first row as column names",
			"rownames true|false               treat the first column as row names",
			"encoding <enc>                    override the detected CSV encoding",
			"infer true|false                  infer column types (CSV only)",
			"ragged true|false                 allow rows of differing length (CSV only)",
			"trimspace true|false              ignore spaces before a CSV field",
			"sheet <name>                      which sheet to read (Excel only)",
		},
		Examples: []string{
			"insyra read sales.csv",
			"insyra read sales.csv headers true rownames true",
			"insyra read book.xlsx sheet Q1",
		},
		Run: runReadCommand,
	})
}

func runReadCommand(ctx *ExecContext, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: read <file> [headers true|false] [rownames true|false] [encoding <enc>] [infer true|false] [ragged true|false] [trimspace true|false] [sheet <name>]")
	}
	fakeArgs := append([]string(nil), args...)
	fakeArgs = append(fakeArgs, "as", "$preview")
	if err := runLoadCommand(ctx, fakeArgs); err != nil {
		return err
	}
	table, err := getDataTableVar(ctx, "$preview")
	if err != nil {
		return err
	}
	table.ShowRangeTo(ctx.Output, 10)
	delete(ctx.Vars, "$preview")
	return nil
}
