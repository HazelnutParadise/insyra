package commands

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/HazelnutParadise/insyra/datafetch"
)

func init() {
	_ = Register(&CommandHandler{Name: "fetch", Usage: "fetch yahoo <ticker> <method> [params...] [as <var>]", Description: "Fetch external data", Run: runFetchCommand})
}

func runFetchCommand(ctx *ExecContext, args []string) error {
	coreArgs, alias := parseAlias(args)
	if len(coreArgs) < 4 {
		return fmt.Errorf("usage: fetch yahoo <ticker> <method> [params...] [as <var>]")
	}
	provider := strings.ToLower(coreArgs[0])
	if provider != "yahoo" {
		return fmt.Errorf("unsupported provider: %s", coreArgs[0])
	}

	ticker := coreArgs[1]
	method := strings.ToLower(coreArgs[2])
	params := coreArgs[3:]

	yf, err := datafetch.YFinance(datafetch.YFinanceConfig{})
	if err != nil {
		return err
	}
	t := yf.Ticker(ticker)

	switch method {
	case "quote":
		dt, getErr := t.Quote()
		if getErr != nil {
			return getErr
		}
		ctx.Vars[alias] = dt
	case "info":
		dt, getErr := t.Info()
		if getErr != nil {
			return getErr
		}
		ctx.Vars[alias] = dt
	case "history":
		dt, getErr := t.History(datafetch.YFHistoryParams{})
		if getErr != nil {
			return getErr
		}
		ctx.Vars[alias] = dt
	case "dividends":
		dt, getErr := t.Dividends()
		if getErr != nil {
			return getErr
		}
		ctx.Vars[alias] = dt
	case "splits":
		dt, getErr := t.Splits()
		if getErr != nil {
			return getErr
		}
		ctx.Vars[alias] = dt
	case "actions":
		dt, getErr := t.Actions()
		if getErr != nil {
			return getErr
		}
		ctx.Vars[alias] = dt
	case "options":
		dt, getErr := t.Options()
		if getErr != nil {
			return getErr
		}
		ctx.Vars[alias] = dt
	case "news":
		count := 10
		if len(params) >= 1 {
			parsed, parseErr := strconv.Atoi(params[0])
			if parseErr != nil {
				return parseErr
			}
			count = parsed
		}
		dt, getErr := t.News(count, "")
		if getErr != nil {
			return getErr
		}
		ctx.Vars[alias] = dt
	case "calendar":
		dt, getErr := t.Calendar()
		if getErr != nil {
			return getErr
		}
		ctx.Vars[alias] = dt
	case "fastinfo":
		dt, getErr := t.FastInfo()
		if getErr != nil {
			return getErr
		}
		ctx.Vars[alias] = dt
	default:
		return fmt.Errorf("unsupported yahoo method: %s", method)
	}

	_, _ = fmt.Fprintf(ctx.Output, "fetched into %s\n", alias)
	return nil
}
