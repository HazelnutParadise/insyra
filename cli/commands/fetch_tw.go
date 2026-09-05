package commands

import (
	"fmt"
	"strings"
	"time"

	insyra "github.com/HazelnutParadise/insyra"
	clienv "github.com/HazelnutParadise/insyra/cli/env"
	"github.com/HazelnutParadise/insyra/datafetch"
)

const (
	// fetchTWDateLayout is the only date form `fetch tw` accepts. Dates are
	// parsed in UTC before a client is built, so a typo never costs a request.
	fetchTWDateLayout = "2006-01-02"
	// fetchTWRetries is the backfill-safe retry count the datafetch docs
	// recommend for TWSE/TPEx. Its companion, the 300ms request interval, is
	// the default of the global `fetch.tw.interval_ms` config key, because the
	// CLI is the surface most likely to run a ten-year loop and a script that
	// needs to go faster overrides it there.
	fetchTWRetries = 2
)

// twStockClient is the part of datafetch.TWStock's client that `fetch tw`
// uses. It exists so tests can inject a fake through newTWStockClient without
// reaching into datafetch internals.
type twStockClient interface {
	DailyPrices(code string, from, to time.Time, market datafetch.TWMarket) (*insyra.DataTable, error)
	DailyPricesAdjusted(code string, from, to time.Time, market datafetch.TWMarket) (*insyra.DataTable, error)
	ExRights(from, to time.Time, market datafetch.TWMarket) (*insyra.DataTable, error)
	InstitutionalTrades(date time.Time, market datafetch.TWMarket) (*insyra.DataTable, error)
	MarginBalance(date time.Time, market datafetch.TWMarket) (*insyra.DataTable, error)
	AllDailyQuotes(market datafetch.TWMarket) (*insyra.DataTable, error)
}

// newTWStockClient is the package-level factory `fetch tw` builds its client
// with. Tests swap it for one returning a fake client.
var newTWStockClient = func(cfg datafetch.TWStockConfig) (twStockClient, error) {
	return datafetch.TWStock(cfg)
}

// twRequest carries the parsed arguments of one `fetch tw` invocation. Which
// fields are filled depends on the form's date arity.
type twRequest struct {
	code   string
	from   time.Time
	to     time.Time
	date   time.Time
	market datafetch.TWMarket
}

// twForm describes one `fetch tw` shape. The table below is the single source
// for dispatch, for the `Forms:` block `help fetch` prints, and for the usage
// line an argument error carries, so the three cannot drift.
type twForm struct {
	name string
	// code reports whether a stock code precedes the form name.
	code bool
	// dates is how many YYYY-MM-DD arguments follow the form name.
	dates int
	usage string
	desc  string
	run   func(client twStockClient, req twRequest) (*insyra.DataTable, error)
}

var twForms = []twForm{
	{
		name: "prices", code: true, dates: 2,
		usage: "fetch tw <code> prices <from> <to> [twse|tpex|auto] [as <var>]",
		desc:  "DailyPrices: daily OHLCV bars for one code",
		run: func(client twStockClient, req twRequest) (*insyra.DataTable, error) {
			return client.DailyPrices(req.code, req.from, req.to, req.market)
		},
	},
	{
		name: "adjprices", code: true, dates: 2,
		usage: "fetch tw <code> adjprices <from> <to> [twse|auto] [as <var>]",
		desc:  "DailyPricesAdjusted: daily bars plus AdjFactor and adjusted OHLC (TWSE only)",
		run: func(client twStockClient, req twRequest) (*insyra.DataTable, error) {
			return client.DailyPricesAdjusted(req.code, req.from, req.to, req.market)
		},
	},
	{
		name: "exrights", code: false, dates: 2,
		usage: "fetch tw exrights <from> <to> [twse|auto] [as <var>]",
		desc:  "ExRights: the exchange's ex-rights/ex-dividend reference table (TWSE only)",
		run: func(client twStockClient, req twRequest) (*insyra.DataTable, error) {
			return client.ExRights(req.from, req.to, req.market)
		},
	},
	{
		name: "institutional", code: false, dates: 1,
		usage: "fetch tw institutional <date> [twse|tpex|auto] [as <var>]",
		desc:  "InstitutionalTrades: one day of institutional buy/sell totals",
		run: func(client twStockClient, req twRequest) (*insyra.DataTable, error) {
			return client.InstitutionalTrades(req.date, req.market)
		},
	},
	{
		name: "margin", code: false, dates: 1,
		usage: "fetch tw margin <date> [twse|tpex|auto] [as <var>]",
		desc:  "MarginBalance: one day of margin-purchase and short-sale balances",
		run: func(client twStockClient, req twRequest) (*insyra.DataTable, error) {
			return client.MarginBalance(req.date, req.market)
		},
	},
	{
		name: "quotes", code: false, dates: 0,
		usage: "fetch tw quotes [twse|tpex|auto] [as <var>]",
		desc:  "AllDailyQuotes: the latest quote table for every listed code",
		run: func(client twStockClient, req twRequest) (*insyra.DataTable, error) {
			return client.AllDailyQuotes(req.market)
		},
	},
}

// runFetchTW handles `fetch tw ...`; args excludes the `tw` source keyword and
// the trailing `as <var>`, which the caller has already stripped.
func runFetchTW(ctx *ExecContext, args []string, alias string) error {
	form, code, rest, err := resolveTWForm(args)
	if err != nil {
		return err
	}
	req, err := parseTWRequest(form, code, rest)
	if err != nil {
		return err
	}

	cfg, err := fetchTWStockConfig(ctx)
	if err != nil {
		return err
	}
	client, err := newTWStockClient(cfg)
	if err != nil {
		return fmt.Errorf("fetch tw: %w", err)
	}
	table, err := form.run(client, req)
	if err != nil {
		return fmt.Errorf("fetch tw: %w", err)
	}

	ctx.Vars[alias] = table
	_, _ = fmt.Fprintf(ctx.Output, "fetched into %s\n", alias)
	return nil
}

// resolveTWForm reads the form name, which sits either first (`exrights`,
// `institutional`, `margin`, `quotes`) or after a stock code (`prices`,
// `adjprices`). It returns the form, the code it was given, and the arguments
// still to be parsed.
func resolveTWForm(args []string) (*twForm, string, []string, error) {
	if len(args) == 0 {
		return nil, "", nil, fmt.Errorf("fetch tw: missing form (%s)", twFormsUsage())
	}
	if form := lookupTWForm(args[0]); form != nil {
		if form.code {
			return nil, "", nil, fmt.Errorf("fetch tw %s: a stock code must come before the form (usage: %s)", form.name, form.usage)
		}
		return form, "", args[1:], nil
	}
	if len(args) < 2 {
		return nil, "", nil, fmt.Errorf("fetch tw: unknown form %q (%s)", args[0], twFormsUsage())
	}
	form := lookupTWForm(args[1])
	if form == nil {
		return nil, "", nil, fmt.Errorf("fetch tw: unknown form %q (%s)", args[1], twFormsUsage())
	}
	if !form.code {
		return nil, "", nil, fmt.Errorf("fetch tw %s: this form takes no stock code (usage: %s)", form.name, form.usage)
	}
	return form, args[0], args[2:], nil
}

// lookupTWForm finds a form by name, case-insensitively.
func lookupTWForm(name string) *twForm {
	for i := range twForms {
		if strings.EqualFold(twForms[i].name, name) {
			return &twForms[i]
		}
	}
	return nil
}

// parseTWRequest consumes the form's dates and its optional market keyword.
// Every rejection here happens before a client is built, so a malformed
// command never reaches the exchange.
func parseTWRequest(form *twForm, code string, rest []string) (twRequest, error) {
	if len(rest) < form.dates || len(rest) > form.dates+1 {
		return twRequest{}, fmt.Errorf("fetch tw %s: usage: %s", form.name, form.usage)
	}
	req := twRequest{code: code}

	switch form.dates {
	case 1:
		date, err := parseTWDate(form, rest[0])
		if err != nil {
			return twRequest{}, err
		}
		req.date = date
	case 2:
		from, err := parseTWDate(form, rest[0])
		if err != nil {
			return twRequest{}, err
		}
		to, err := parseTWDate(form, rest[1])
		if err != nil {
			return twRequest{}, err
		}
		if from.After(to) {
			return twRequest{}, fmt.Errorf("fetch tw %s: from must not be after to (usage: %s)", form.name, form.usage)
		}
		req.from, req.to = from, to
	}

	req.market = datafetch.TWMarketAuto
	if len(rest) == form.dates+1 {
		market, err := parseTWMarket(form, rest[form.dates])
		if err != nil {
			return twRequest{}, err
		}
		req.market = market
	}
	return req, nil
}

func parseTWDate(form *twForm, raw string) (time.Time, error) {
	parsed, err := time.ParseInLocation(fetchTWDateLayout, strings.TrimSpace(raw), time.UTC)
	if err != nil {
		return time.Time{}, fmt.Errorf("fetch tw %s: invalid date %q: expected YYYY-MM-DD (usage: %s)", form.name, raw, form.usage)
	}
	return parsed, nil
}

func parseTWMarket(form *twForm, raw string) (datafetch.TWMarket, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "twse":
		return datafetch.TWMarketTWSE, nil
	case "tpex":
		return datafetch.TWMarketTPEx, nil
	case "auto":
		return datafetch.TWMarketAuto, nil
	}
	return "", fmt.Errorf("fetch tw %s: unknown market %q (supported: twse, tpex, auto) (usage: %s)", form.name, raw, form.usage)
}

// fetchTWStockConfig builds the client configuration: the recommended
// backfill throttle, with the interval overridable through the global
// `fetch.tw.interval_ms` config key.
func fetchTWStockConfig(ctx *ExecContext) (datafetch.TWStockConfig, error) {
	manager := ctx.Env
	if manager == nil {
		manager = clienv.Default()
	}
	cfg, err := manager.LoadGlobalConfig()
	if err != nil {
		return datafetch.TWStockConfig{}, fmt.Errorf("fetch tw: %w", err)
	}
	if cfg.FetchTWIntervalMS < 0 {
		return datafetch.TWStockConfig{}, fmt.Errorf("fetch tw: invalid config fetch.tw.interval_ms %d: must be >= 0", cfg.FetchTWIntervalMS)
	}
	return datafetch.TWStockConfig{
		Interval: time.Duration(cfg.FetchTWIntervalMS) * time.Millisecond,
		Retries:  fetchTWRetries,
	}, nil
}

// twFormLines renders the `fetch tw` entries of the `Forms:` help block.
func twFormLines() []string {
	lines := make([]string, 0, len(twForms))
	for i := range twForms {
		lines = append(lines, fmt.Sprintf("%-63s %s", twForms[i].usage, twForms[i].desc))
	}
	return lines
}

func twFormNames(separator string) string {
	names := make([]string, 0, len(twForms))
	for i := range twForms {
		names = append(names, twForms[i].name)
	}
	return strings.Join(names, separator)
}

func twFormsUsage() string {
	return fmt.Sprintf("usage: fetch tw [<code>] %s ... (forms: %s)", twFormNames("|"), twFormNames(", "))
}
