package commands

import "fmt"

func init() {
	_ = Register(&CommandHandler{Name: "rank", Usage: "rank <var> [as <var>]", Description: "Rank DataList", Run: runRankCommand})
	_ = Register(&CommandHandler{Name: "normalize", Usage: "normalize <var> [as <var>]", Description: "Normalize DataList", Run: runNormalizeCommand})
	_ = Register(&CommandHandler{Name: "standardize", Usage: "standardize <var> [as <var>]", Description: "Standardize DataList", Run: runStandardizeCommand})
	_ = Register(&CommandHandler{Name: "reverse", Usage: "reverse <var> [as <var>]", Description: "Reverse DataList", Run: runReverseCommand})
	_ = Register(&CommandHandler{Name: "upper", Usage: "upper <var> [as <var>]", Description: "Uppercase DataList strings", Run: runUpperCommand})
	_ = Register(&CommandHandler{Name: "lower", Usage: "lower <var> [as <var>]", Description: "Lowercase DataList strings", Run: runLowerCommand})
	_ = Register(&CommandHandler{Name: "capitalize", Usage: "capitalize <var> [as <var>]", Description: "Capitalize DataList strings", Run: runCapitalizeCommand})
	_ = Register(&CommandHandler{Name: "parsenums", Usage: "parsenums <var> [as <var>]", Description: "Parse DataList strings to numbers", Run: runParseNumsCommand})
	_ = Register(&CommandHandler{Name: "parsestrings", Usage: "parsestrings <var> [as <var>]", Description: "Parse DataList numbers to strings", Run: runParseStringsCommand})
}

func runRankCommand(ctx *ExecContext, args []string) error {
	return runDLTransform(ctx, args, func(dlName string, dlOps *dlTransformProxy) (any, error) { return dlOps.rank(dlName) })
}
func runNormalizeCommand(ctx *ExecContext, args []string) error {
	return runDLTransform(ctx, args, func(dlName string, dlOps *dlTransformProxy) (any, error) { return dlOps.normalize(dlName) })
}
func runStandardizeCommand(ctx *ExecContext, args []string) error {
	return runDLTransform(ctx, args, func(dlName string, dlOps *dlTransformProxy) (any, error) { return dlOps.standardize(dlName) })
}
func runReverseCommand(ctx *ExecContext, args []string) error {
	return runDLTransform(ctx, args, func(dlName string, dlOps *dlTransformProxy) (any, error) { return dlOps.reverse(dlName) })
}
func runUpperCommand(ctx *ExecContext, args []string) error {
	return runDLTransform(ctx, args, func(dlName string, dlOps *dlTransformProxy) (any, error) { return dlOps.upper(dlName) })
}
func runLowerCommand(ctx *ExecContext, args []string) error {
	return runDLTransform(ctx, args, func(dlName string, dlOps *dlTransformProxy) (any, error) { return dlOps.lower(dlName) })
}
func runCapitalizeCommand(ctx *ExecContext, args []string) error {
	return runDLTransform(ctx, args, func(dlName string, dlOps *dlTransformProxy) (any, error) { return dlOps.capitalize(dlName) })
}
func runParseNumsCommand(ctx *ExecContext, args []string) error {
	return runDLTransform(ctx, args, func(dlName string, dlOps *dlTransformProxy) (any, error) { return dlOps.parseNums(dlName) })
}
func runParseStringsCommand(ctx *ExecContext, args []string) error {
	return runDLTransform(ctx, args, func(dlName string, dlOps *dlTransformProxy) (any, error) { return dlOps.parseStrings(dlName) })
}

type dlTransformProxy struct{ ctx *ExecContext }

func runDLTransform(ctx *ExecContext, args []string, fn func(dlName string, ops *dlTransformProxy) (any, error)) error {
	coreArgs, alias := parseAlias(args)
	if len(coreArgs) < 1 {
		return fmt.Errorf("usage: <command> <var> [as <var>]")
	}
	result, err := fn(coreArgs[0], &dlTransformProxy{ctx: ctx})
	if err != nil {
		return err
	}
	ctx.Vars[alias] = result
	_, _ = fmt.Fprintf(ctx.Output, "saved as %s\n", alias)
	return nil
}

func (p *dlTransformProxy) rank(name string) (any, error) {
	dl, err := getDataListVar(p.ctx, name)
	if err != nil {
		return nil, err
	}
	return dl.Clone().Rank(), nil
}
func (p *dlTransformProxy) normalize(name string) (any, error) {
	dl, err := getDataListVar(p.ctx, name)
	if err != nil {
		return nil, err
	}
	return dl.Clone().Normalize(), nil
}
func (p *dlTransformProxy) standardize(name string) (any, error) {
	dl, err := getDataListVar(p.ctx, name)
	if err != nil {
		return nil, err
	}
	return dl.Clone().Standardize(), nil
}
func (p *dlTransformProxy) reverse(name string) (any, error) {
	dl, err := getDataListVar(p.ctx, name)
	if err != nil {
		return nil, err
	}
	return dl.Clone().Reverse(), nil
}
func (p *dlTransformProxy) upper(name string) (any, error) {
	dl, err := getDataListVar(p.ctx, name)
	if err != nil {
		return nil, err
	}
	return dl.Clone().Upper(), nil
}
func (p *dlTransformProxy) lower(name string) (any, error) {
	dl, err := getDataListVar(p.ctx, name)
	if err != nil {
		return nil, err
	}
	return dl.Clone().Lower(), nil
}
func (p *dlTransformProxy) capitalize(name string) (any, error) {
	dl, err := getDataListVar(p.ctx, name)
	if err != nil {
		return nil, err
	}
	return dl.Clone().Capitalize(), nil
}
func (p *dlTransformProxy) parseNums(name string) (any, error) {
	dl, err := getDataListVar(p.ctx, name)
	if err != nil {
		return nil, err
	}
	return dl.Clone().ParseNumbers(), nil
}
func (p *dlTransformProxy) parseStrings(name string) (any, error) {
	dl, err := getDataListVar(p.ctx, name)
	if err != nil {
		return nil, err
	}
	return dl.Clone().ParseStrings(), nil
}
