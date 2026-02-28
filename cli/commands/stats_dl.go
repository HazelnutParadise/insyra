package commands

import (
	"fmt"
)

func init() {
	registerScalarDLStat("sum", "sum <var>", "Sum of DataList", func(name string, dlAccessor func(string) (float64, error)) {})

	_ = Register(&CommandHandler{Name: "sum", Usage: "sum <var>", Description: "DataList sum", Run: makeDLNumberPrinter(func(dlName string, dl *floatStatProxy) (any, error) { return dl.sum(dlName) })})
	_ = Register(&CommandHandler{Name: "mean", Usage: "mean <var>", Description: "DataList mean", Run: makeDLNumberPrinter(func(dlName string, dl *floatStatProxy) (any, error) { return dl.mean(dlName) })})
	_ = Register(&CommandHandler{Name: "median", Usage: "median <var>", Description: "DataList median", Run: makeDLNumberPrinter(func(dlName string, dl *floatStatProxy) (any, error) { return dl.median(dlName) })})
	_ = Register(&CommandHandler{Name: "mode", Usage: "mode <var>", Description: "DataList mode", Run: makeDLNumberPrinter(func(dlName string, dl *floatStatProxy) (any, error) { return dl.mode(dlName) })})
	_ = Register(&CommandHandler{Name: "stdev", Usage: "stdev <var>", Description: "DataList standard deviation", Run: makeDLNumberPrinter(func(dlName string, dl *floatStatProxy) (any, error) { return dl.stdev(dlName) })})
	_ = Register(&CommandHandler{Name: "var", Usage: "var <var>", Description: "DataList variance", Run: makeDLNumberPrinter(func(dlName string, dl *floatStatProxy) (any, error) { return dl.variance(dlName) })})
	_ = Register(&CommandHandler{Name: "min", Usage: "min <var>", Description: "DataList minimum", Run: makeDLNumberPrinter(func(dlName string, dl *floatStatProxy) (any, error) { return dl.min(dlName) })})
	_ = Register(&CommandHandler{Name: "max", Usage: "max <var>", Description: "DataList maximum", Run: makeDLNumberPrinter(func(dlName string, dl *floatStatProxy) (any, error) { return dl.max(dlName) })})
	_ = Register(&CommandHandler{Name: "range", Usage: "range <var>", Description: "DataList range", Run: makeDLNumberPrinter(func(dlName string, dl *floatStatProxy) (any, error) { return dl.rangeVal(dlName) })})
}

type floatStatProxy struct {
	ctx *ExecContext
}

func makeDLNumberPrinter(fn func(dlName string, proxy *floatStatProxy) (any, error)) func(*ExecContext, []string) error {
	return func(ctx *ExecContext, args []string) error {
		if len(args) < 1 {
			return fmt.Errorf("usage: <command> <var>")
		}
		result, err := fn(args[0], &floatStatProxy{ctx: ctx})
		if err != nil {
			return err
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
