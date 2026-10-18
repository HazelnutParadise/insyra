package commands

import (
	"fmt"
)

func init() {
	registerScalarDLStat("sum", "sum <var>", "Sum of DataList", func(name string, dlAccessor func(string) (float64, error)) {})

	_ = Register(&CommandHandler{Name: "sum", Args: MaxArgs(1).WithAlias(), Usage: "sum <var> [as <var>]", Description: "DataList sum", Run: makeDLNumberPrinter("sum", func(dlName string, dl *floatStatProxy) (any, error) { return dl.sum(dlName) })})
	_ = Register(&CommandHandler{Name: "mean", Args: MaxArgs(1).WithAlias(), Usage: "mean <var> [as <var>]", Description: "DataList mean", Run: makeDLNumberPrinter("mean", func(dlName string, dl *floatStatProxy) (any, error) { return dl.mean(dlName) })})
	_ = Register(&CommandHandler{Name: "median", Args: MaxArgs(1).WithAlias(), Usage: "median <var> [as <var>]", Description: "DataList median", Run: makeDLNumberPrinter("median", func(dlName string, dl *floatStatProxy) (any, error) { return dl.median(dlName) })})
	_ = Register(&CommandHandler{Name: "mode", Args: MaxArgs(1).WithAlias(), Usage: "mode <var> [as <var>]", Description: "DataList mode", Run: makeDLNumberPrinter("mode", func(dlName string, dl *floatStatProxy) (any, error) { return dl.mode(dlName) })})
	_ = Register(&CommandHandler{Name: "stdev", Args: MaxArgs(1).WithAlias(), Usage: "stdev <var> [as <var>]", Description: "DataList standard deviation", Run: makeDLNumberPrinter("stdev", func(dlName string, dl *floatStatProxy) (any, error) { return dl.stdev(dlName) })})
	_ = Register(&CommandHandler{Name: "var", Args: MaxArgs(1).WithAlias(), Usage: "var <var> [as <var>]", Description: "DataList variance", Run: makeDLNumberPrinter("var", func(dlName string, dl *floatStatProxy) (any, error) { return dl.variance(dlName) })})
	_ = Register(&CommandHandler{Name: "min", Args: MaxArgs(1).WithAlias(), Usage: "min <var> [as <var>]", Description: "DataList minimum", Run: makeDLNumberPrinter("min", func(dlName string, dl *floatStatProxy) (any, error) { return dl.min(dlName) })})
	_ = Register(&CommandHandler{Name: "max", Args: MaxArgs(1).WithAlias(), Usage: "max <var> [as <var>]", Description: "DataList maximum", Run: makeDLNumberPrinter("max", func(dlName string, dl *floatStatProxy) (any, error) { return dl.max(dlName) })})
	_ = Register(&CommandHandler{Name: "range", Args: MaxArgs(1).WithAlias(), Usage: "range <var> [as <var>]", Description: "DataList range", Run: makeDLNumberPrinter("range", func(dlName string, dl *floatStatProxy) (any, error) { return dl.rangeVal(dlName) })})
}

type floatStatProxy struct {
	ctx *ExecContext
}

func makeDLNumberPrinter(name string, fn func(dlName string, proxy *floatStatProxy) (any, error)) func(*ExecContext, []string) error {
	return func(ctx *ExecContext, args []string) error {
		coreArgs, alias := parseAlias(args)
		if len(coreArgs) < 1 {
			return fmt.Errorf("usage: %s <var> [as <var>]", name)
		}
		if len(coreArgs) > 1 {
			return fmt.Errorf("%s: unexpected argument %q (usage: %s <var> [as <var>])", name, coreArgs[1], name)
		}
		result, err := fn(coreArgs[0], &floatStatProxy{ctx: ctx})
		if err != nil {
			return err
		}
		// `as` keeps the number so later commands can take it by name.
		if len(coreArgs) != len(args) {
			ctx.Vars[alias] = result
		}
		_, _ = fmt.Fprintf(ctx.Output, "%v\n", result)
		return nil
	}
}

func (p *floatStatProxy) sum(name string) (float64, error) {
	dl, err := getDataListVar(p.ctx, name)
	if err != nil {
		return 0, err
	}
	return dl.Sum(), nil
}
func (p *floatStatProxy) mean(name string) (float64, error) {
	dl, err := getDataListVar(p.ctx, name)
	if err != nil {
		return 0, err
	}
	return dl.Mean(), nil
}
func (p *floatStatProxy) median(name string) (float64, error) {
	dl, err := getDataListVar(p.ctx, name)
	if err != nil {
		return 0, err
	}
	return dl.Median(), nil
}
func (p *floatStatProxy) mode(name string) (any, error) {
	dl, err := getDataListVar(p.ctx, name)
	if err != nil {
		return nil, err
	}
	return dl.Mode(), nil
}
func (p *floatStatProxy) stdev(name string) (float64, error) {
	dl, err := getDataListVar(p.ctx, name)
	if err != nil {
		return 0, err
	}
	return dl.Stdev(), nil
}
func (p *floatStatProxy) variance(name string) (float64, error) {
	dl, err := getDataListVar(p.ctx, name)
	if err != nil {
		return 0, err
	}
	return dl.Var(), nil
}
func (p *floatStatProxy) min(name string) (float64, error) {
	dl, err := getDataListVar(p.ctx, name)
	if err != nil {
		return 0, err
	}
	return dl.Min(), nil
}
func (p *floatStatProxy) max(name string) (float64, error) {
	dl, err := getDataListVar(p.ctx, name)
	if err != nil {
		return 0, err
	}
	return dl.Max(), nil
}
func (p *floatStatProxy) rangeVal(name string) (float64, error) {
	dl, err := getDataListVar(p.ctx, name)
	if err != nil {
		return 0, err
	}
	return dl.Range(), nil
}

func registerScalarDLStat(name, usage, description string, _ func(string, func(string) (float64, error))) {
}
