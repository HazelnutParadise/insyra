package commands

import (
	"fmt"
	"strconv"
)

func init() {
	_ = Register(&CommandHandler{Name: "quartile", Usage: "quartile <var> <q>", Description: "DataList quartile", Run: runQuartileCommand})
	_ = Register(&CommandHandler{Name: "iqr", Usage: "iqr <var>", Description: "DataList IQR", Run: runIQRCommand})
	_ = Register(&CommandHandler{Name: "percentile", Usage: "percentile <var> <p>", Description: "DataList percentile", Run: runPercentileCommand})
	_ = Register(&CommandHandler{Name: "count", Usage: "count <var> [value]", Description: "Count occurrences", Run: runCountCommand})
	_ = Register(&CommandHandler{Name: "counter", Usage: "counter <var>", Description: "DataList frequency map", Run: runCounterCommand})
}

func runQuartileCommand(ctx *ExecContext, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: quartile <var> <q>")
	}
	dl, err := getDataListVar(ctx, args[0])
	if err != nil {
		return err
	}
	q, err := strconv.Atoi(args[1])
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(ctx.Output, "%v\n", dl.Quartile(q))
	return nil
}

func runIQRCommand(ctx *ExecContext, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: iqr <var>")
	}
	dl, err := getDataListVar(ctx, args[0])
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(ctx.Output, "%v\n", dl.IQR())
	return nil
}

func runPercentileCommand(ctx *ExecContext, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: percentile <var> <p>")
	}
	dl, err := getDataListVar(ctx, args[0])
	if err != nil {
		return err
	}
	p, err := strconv.ParseFloat(args[1], 64)
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(ctx.Output, "%v\n", dl.Percentile(p))
	return nil
}

func runCountCommand(ctx *ExecContext, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: count <var> [value]")
	}
	if dl, err := getDataListVar(ctx, args[0]); err == nil {
		if len(args) < 2 {
			return fmt.Errorf("usage for datalist: count <var> <value>")
		}
		_, _ = fmt.Fprintf(ctx.Output, "%d\n", dl.Count(parseLiteral(args[1])))
		return nil
	}
	if dt, err := getDataTableVar(ctx, args[0]); err == nil {
		if len(args) < 2 {
			return fmt.Errorf("usage for datatable: count <var> <value>")
		}
		_, _ = fmt.Fprintf(ctx.Output, "%d\n", dt.Count(parseLiteral(args[1])))
		return nil
	}
	return fmt.Errorf("variable not found: %s", args[0])
}

func runCounterCommand(ctx *ExecContext, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: counter <var>")
	}
	dl, err := getDataListVar(ctx, args[0])
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(ctx.Output, "%v\n", dl.Counter())
	return nil
}
