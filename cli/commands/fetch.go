package commands

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/HazelnutParadise/insyra/datafetch"
)

func init() {
	_ = Register(&CommandHandler{
		Name:        "fetch",
		Usage:       "fetch yahoo|tw ... [as <var>]",
		Description: "Fetch external data",
		Forms: append([]string{
			"fetch yahoo <ticker> quote                       latest quote",
			"fetch yahoo <ticker> info                        company info",
			"fetch yahoo <ticker> history                     OHLCV history",
			"fetch yahoo <ticker> dividends                   dividend history",
			"fetch yahoo <ticker> splits                      split history",
			"fetch yahoo <ticker> actions                     dividends + splits",
			"fetch yahoo <ticker> options                     options chain",
			"fetch yahoo <ticker> news [count]                latest news (default count=10)",
			"fetch yahoo <ticker> calendar                    earnings/event calendar",
			"fetch yahoo <ticker> fastinfo                    quick metrics snapshot",
		}, twFormLines()...),
		Examples: []string{
			"insyra fetch yahoo AAPL quote as q",
			"insyra fetch yahoo TSLA history as hist",
			"insyra fetch yahoo MSFT news 20 as news",
			"insyra fetch tw 2330 prices 2026-08-01 2026-08-31 twse as bars",
			"insyra fetch tw 2330 adjprices 2026-08-01 2026-08-31 twse as adj",
			"insyra fetch tw exrights 2026-08-01 2026-08-31 as xr",
			"insyra fetch tw institutional 2026-08-15 twse as inst",
			"insyra fetch tw margin 2026-08-15 as margin",
			"insyra fetch tw quotes twse as quotes",
		},
		Run: runFetchCommand,
	})
}

func runFetchCommand(ctx *ExecContext, args []string) error {
	coreArgs, alias := parseAlias(args)
	// The source is dispatched before anything else, so each source keeps its
	// own argument shape; `yahoo` below is unchanged.
	if len(coreArgs) >= 1 && strings.EqualFold(coreArgs[0], "tw") {
		return runFetchTW(ctx, coreArgs[1:], alias)
	}
	// yahoo <ticker> <method> is three tokens; `as <var>` has already been
	// stripped by parseAlias, so the documented `fetch yahoo AAPL quote as q`
	// arrives here with exactly three.
	if len(coreArgs) < 3 {
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
