package commands

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"

	insyra "github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/cli/env"
	"github.com/HazelnutParadise/insyra/datafetch"
)

// ===========================================================================
// fetch tw (add-cli-fetch-tw)
// ===========================================================================

// fakeTWCall records one call made on the fake client.
type fakeTWCall struct {
	method string
	code   string
	from   time.Time
	to     time.Time
	date   time.Time
	market datafetch.TWMarket
}

// fakeTWClient is the injected stand-in for datafetch.TWStock's client. It
// records every call and returns a canned table, so no test touches the
// network.
type fakeTWClient struct {
	calls []fakeTWCall
	table *insyra.DataTable
	err   error
}

func (f *fakeTWClient) result() (*insyra.DataTable, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.table, nil
}

func (f *fakeTWClient) DailyPrices(code string, from, to time.Time, market datafetch.TWMarket) (*insyra.DataTable, error) {
	f.calls = append(f.calls, fakeTWCall{method: "DailyPrices", code: code, from: from, to: to, market: market})
	return f.result()
}

func (f *fakeTWClient) DailyPricesAdjusted(code string, from, to time.Time, market datafetch.TWMarket) (*insyra.DataTable, error) {
	f.calls = append(f.calls, fakeTWCall{method: "DailyPricesAdjusted", code: code, from: from, to: to, market: market})
	return f.result()
}

func (f *fakeTWClient) ExRights(from, to time.Time, market datafetch.TWMarket) (*insyra.DataTable, error) {
	f.calls = append(f.calls, fakeTWCall{method: "ExRights", from: from, to: to, market: market})
	return f.result()
}

func (f *fakeTWClient) InstitutionalTrades(date time.Time, market datafetch.TWMarket) (*insyra.DataTable, error) {
	f.calls = append(f.calls, fakeTWCall{method: "InstitutionalTrades", date: date, market: market})
	return f.result()
}

func (f *fakeTWClient) MarginBalance(date time.Time, market datafetch.TWMarket) (*insyra.DataTable, error) {
	f.calls = append(f.calls, fakeTWCall{method: "MarginBalance", date: date, market: market})
	return f.result()
}

func (f *fakeTWClient) AllDailyQuotes(market datafetch.TWMarket) (*insyra.DataTable, error) {
	f.calls = append(f.calls, fakeTWCall{method: "AllDailyQuotes", market: market})
	return f.result()
}

// fetchTWContext builds an ExecContext whose config lives in a temp directory,
// so a `config fetch.tw.interval_ms` write never touches the real ~/.insyra.
func fetchTWContext(t *testing.T) *ExecContext {
	t.Helper()
	ctx := newTestExecContext(t)
	ctx.Env = env.NewManager(t.TempDir(), "")
	return ctx
}

// installFakeTWClient swaps the package-level factory for the run of one test
// and returns the fake plus the config it was handed.
func installFakeTWClient(t *testing.T, fake *fakeTWClient) *datafetch.TWStockConfig {
	t.Helper()
	seen := &datafetch.TWStockConfig{}
	original := newTWStockClient
	newTWStockClient = func(cfg datafetch.TWStockConfig) (twStockClient, error) {
		*seen = cfg
		return fake, nil
	}
	t.Cleanup(func() { newTWStockClient = original })
	return seen
}

func twTestTable() *insyra.DataTable {
	return insyra.NewDataTable(namedList("AdjClose", 100.0, 101.0))
}

func mustDate(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		t.Fatalf("bad test date %q: %v", value, err)
	}
	return parsed
}

func twOutput(t *testing.T, ctx *ExecContext) string {
	t.Helper()
	buf, ok := ctx.Output.(*bytes.Buffer)
	if !ok {
		t.Fatalf("expected a *bytes.Buffer output, got %T", ctx.Output)
	}
	return buf.String()
}

// --- 1.1 the six forms -----------------------------------------------------

func TestRunFetchCommand_TWPrices(t *testing.T) {
	ctx := fetchTWContext(t)
	fake := &fakeTWClient{table: twTestTable()}
	installFakeTWClient(t, fake)

	if err := runFetchCommand(ctx, []string{"tw", "2330", "prices", "2026-08-01", "2026-08-31", "as", "p"}); err != nil {
		t.Fatalf("fetch tw prices failed: %v", err)
	}
	if len(fake.calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(fake.calls))
	}
	call := fake.calls[0]
	if call.method != "DailyPrices" || call.code != "2330" {
		t.Errorf("call = %+v, want DailyPrices for 2330", call)
	}
	if !call.from.Equal(mustDate(t, "2026-08-01")) || !call.to.Equal(mustDate(t, "2026-08-31")) {
		t.Errorf("dates = %v..%v", call.from, call.to)
	}
	if call.market != datafetch.TWMarketAuto {
		t.Errorf("market = %q, want Auto by default", call.market)
	}
	if ctx.Vars["p"] != any(fake.table) {
		t.Errorf("p = %v, want the fetched table", ctx.Vars["p"])
	}
	if out := twOutput(t, ctx); !strings.Contains(out, "fetched into p") {
		t.Errorf("output %q should report the alias", out)
	}
}

// Scenario: Adjusted prices into a variable
func TestRunFetchCommand_TWAdjPricesIntoVariable(t *testing.T) {
	ctx := fetchTWContext(t)
	fake := &fakeTWClient{table: twTestTable()}
	installFakeTWClient(t, fake)

	if err := runFetchCommand(ctx, []string{"tw", "2330", "adjprices", "2026-08-01", "2026-09-03", "as", "p"}); err != nil {
		t.Fatalf("fetch tw adjprices failed: %v", err)
	}
	want := fakeTWCall{
		method: "DailyPricesAdjusted",
		code:   "2330",
		from:   mustDate(t, "2026-08-01"),
		to:     mustDate(t, "2026-09-03"),
		market: datafetch.TWMarketAuto,
	}
	if len(fake.calls) != 1 || fake.calls[0] != want {
		t.Fatalf("calls = %+v, want exactly %+v", fake.calls, want)
	}
	table, ok := ctx.Vars["p"].(*insyra.DataTable)
	if !ok || table != fake.table {
		t.Fatalf("p = %v, want the DataTable the client returned", ctx.Vars["p"])
	}
}

// Scenario: Market keyword maps to TWMarket
func TestRunFetchCommand_TWMarketKeyword(t *testing.T) {
	ctx := fetchTWContext(t)
	fake := &fakeTWClient{table: twTestTable()}
	installFakeTWClient(t, fake)

	if err := runFetchCommand(ctx, []string{"tw", "6488", "prices", "2026-08-01", "2026-08-31", "tpex"}); err != nil {
		t.Fatalf("fetch tw prices tpex failed: %v", err)
	}
	if len(fake.calls) != 1 || fake.calls[0].market != datafetch.TWMarketTPEx {
		t.Fatalf("calls = %+v, want TWMarketTPEx", fake.calls)
	}
	if fake.calls[0].code != "6488" {
		t.Errorf("code = %q, want 6488", fake.calls[0].code)
	}
	if ctx.Vars["$result"] != any(fake.table) {
		t.Errorf("$result = %v, want the fetched table", ctx.Vars["$result"])
	}
}

func TestRunFetchCommand_TWMarketKeywordsAllMap(t *testing.T) {
	cases := map[string]datafetch.TWMarket{
		"twse": datafetch.TWMarketTWSE,
		"TWSE": datafetch.TWMarketTWSE,
		"tpex": datafetch.TWMarketTPEx,
		"TPEx": datafetch.TWMarketTPEx,
		"auto": datafetch.TWMarketAuto,
		"Auto": datafetch.TWMarketAuto,
	}
	for keyword, want := range cases {
		t.Run(keyword, func(t *testing.T) {
			ctx := fetchTWContext(t)
			fake := &fakeTWClient{table: twTestTable()}
			installFakeTWClient(t, fake)
			if err := runFetchCommand(ctx, []string{"tw", "quotes", keyword}); err != nil {
				t.Fatalf("fetch tw quotes %s failed: %v", keyword, err)
			}
			if len(fake.calls) != 1 || fake.calls[0].market != want {
				t.Fatalf("calls = %+v, want market %q", fake.calls, want)
			}
		})
	}
}

func TestRunFetchCommand_TWExRights(t *testing.T) {
	ctx := fetchTWContext(t)
	fake := &fakeTWClient{table: twTestTable()}
	installFakeTWClient(t, fake)

	if err := runFetchCommand(ctx, []string{"tw", "exrights", "2026-08-01", "2026-08-31", "twse", "as", "x"}); err != nil {
		t.Fatalf("fetch tw exrights failed: %v", err)
	}
	want := fakeTWCall{
		method: "ExRights",
		from:   mustDate(t, "2026-08-01"),
		to:     mustDate(t, "2026-08-31"),
		market: datafetch.TWMarketTWSE,
	}
	if len(fake.calls) != 1 || fake.calls[0] != want {
		t.Fatalf("calls = %+v, want %+v", fake.calls, want)
	}
	if ctx.Vars["x"] != any(fake.table) {
		t.Errorf("x = %v, want the fetched table", ctx.Vars["x"])
	}
}

func TestRunFetchCommand_TWInstitutional(t *testing.T) {
	ctx := fetchTWContext(t)
	fake := &fakeTWClient{table: twTestTable()}
	installFakeTWClient(t, fake)

	if err := runFetchCommand(ctx, []string{"tw", "institutional", "2026-08-15", "twse", "as", "i"}); err != nil {
		t.Fatalf("fetch tw institutional failed: %v", err)
	}
	want := fakeTWCall{
		method: "InstitutionalTrades",
		date:   mustDate(t, "2026-08-15"),
		market: datafetch.TWMarketTWSE,
	}
	if len(fake.calls) != 1 || fake.calls[0] != want {
		t.Fatalf("calls = %+v, want %+v", fake.calls, want)
	}
	if ctx.Vars["i"] != any(fake.table) {
		t.Errorf("i = %v, want the fetched table", ctx.Vars["i"])
	}
}

func TestRunFetchCommand_TWMargin(t *testing.T) {
	ctx := fetchTWContext(t)
	fake := &fakeTWClient{table: twTestTable()}
	installFakeTWClient(t, fake)

	if err := runFetchCommand(ctx, []string{"tw", "margin", "2026-08-15", "as", "m"}); err != nil {
		t.Fatalf("fetch tw margin failed: %v", err)
	}
	want := fakeTWCall{
		method: "MarginBalance",
		date:   mustDate(t, "2026-08-15"),
		market: datafetch.TWMarketAuto,
	}
	if len(fake.calls) != 1 || fake.calls[0] != want {
		t.Fatalf("calls = %+v, want %+v", fake.calls, want)
	}
	if ctx.Vars["m"] != any(fake.table) {
		t.Errorf("m = %v, want the fetched table", ctx.Vars["m"])
	}
}

func TestRunFetchCommand_TWQuotes(t *testing.T) {
	ctx := fetchTWContext(t)
	fake := &fakeTWClient{table: twTestTable()}
	installFakeTWClient(t, fake)

	if err := runFetchCommand(ctx, []string{"tw", "quotes", "as", "q"}); err != nil {
		t.Fatalf("fetch tw quotes failed: %v", err)
	}
	want := fakeTWCall{method: "AllDailyQuotes", market: datafetch.TWMarketAuto}
	if len(fake.calls) != 1 || fake.calls[0] != want {
		t.Fatalf("calls = %+v, want %+v", fake.calls, want)
	}
	if ctx.Vars["q"] != any(fake.table) {
		t.Errorf("q = %v, want the fetched table", ctx.Vars["q"])
	}
}

// --- 1.2 errors before any request ----------------------------------------

// twErrorCase is one argument list that must be rejected before a request.
type twErrorCase struct {
	name     string
	args     []string
	contains string
}

// Scenario: Bad date is rejected before any request (plus the sibling
// pre-request rejections: from > to, unknown market, unknown form).
func TestRunFetchCommand_TWRejectsBeforeRequest(t *testing.T) {
	cases := []twErrorCase{
		{
			name:     "bad from date",
			args:     []string{"tw", "2330", "prices", "2026/08/01", "2026-09-03"},
			contains: `invalid date "2026/08/01"`,
		},
		{
			name:     "bad to date",
			args:     []string{"tw", "2330", "prices", "2026-08-01", "20260903"},
			contains: `invalid date "20260903"`,
		},
		{
			name:     "bad single date",
			args:     []string{"tw", "institutional", "15/08/2026"},
			contains: `invalid date "15/08/2026"`,
		},
		{
			name:     "from after to",
			args:     []string{"tw", "2330", "prices", "2026-09-03", "2026-08-01"},
			contains: "from must not be after to",
		},
		{
			name:     "unknown market",
			args:     []string{"tw", "2330", "prices", "2026-08-01", "2026-08-31", "nyse"},
			contains: `unknown market "nyse"`,
		},
		{
			name:     "unknown form",
			args:     []string{"tw", "2330", "candles", "2026-08-01", "2026-08-31"},
			contains: `unknown form "candles"`,
		},
		{
			name:     "unknown bare form",
			args:     []string{"tw", "shorts"},
			contains: "unknown form",
		},
		{
			name:     "missing arguments",
			args:     []string{"tw", "2330", "prices", "2026-08-01"},
			contains: "usage:",
		},
		{
			name:     "trailing argument",
			args:     []string{"tw", "quotes", "twse", "extra"},
			contains: "usage:",
		},
		{
			name:     "no form",
			args:     []string{"tw"},
			contains: "usage:",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			ctx := fetchTWContext(t)
			fake := &fakeTWClient{table: twTestTable()}
			installFakeTWClient(t, fake)

			err := runFetchCommand(ctx, testCase.args)
			if err == nil {
				t.Fatalf("expected an error for %v", testCase.args)
			}
			if !strings.Contains(err.Error(), testCase.contains) {
				t.Errorf("error %q should contain %q", err, testCase.contains)
			}
			if !strings.Contains(err.Error(), "usage:") {
				t.Errorf("error %q should carry a usage line", err)
			}
			if len(fake.calls) != 0 {
				t.Errorf("client was called %d time(s); it must not be reached", len(fake.calls))
			}
			if len(ctx.Vars) != 0 {
				t.Errorf("vars = %v, want nothing stored", ctx.Vars)
			}
		})
	}
}

// Scenario: Library error is surfaced
func TestRunFetchCommand_TWLibraryErrorSurfaced(t *testing.T) {
	ctx := fetchTWContext(t)
	libraryErr := errors.New("datafetch: TPEx ex-rights not supported: TPEx publishes no dated ex-rights history endpoint")
	fake := &fakeTWClient{err: libraryErr}
	installFakeTWClient(t, fake)

	err := runFetchCommand(ctx, []string{"tw", "exrights", "2026-08-01", "2026-08-31", "tpex", "as", "x"})
	if err == nil {
		t.Fatal("expected the library error to surface")
	}
	if !strings.HasPrefix(err.Error(), "fetch tw:") {
		t.Errorf("error %q should start with \"fetch tw:\"", err)
	}
	if !strings.Contains(err.Error(), libraryErr.Error()) {
		t.Errorf("error %q should carry the library message verbatim", err)
	}
	if !errors.Is(err, libraryErr) {
		t.Errorf("error %q should wrap the library error", err)
	}
	if _, stored := ctx.Vars["x"]; stored {
		t.Error("a failed fetch must not store a variable")
	}
}

func TestRunFetchCommand_TWFactoryErrorSurfaced(t *testing.T) {
	ctx := fetchTWContext(t)
	factoryErr := errors.New("datafetch: TWStock Interval must be >= 0")
	original := newTWStockClient
	newTWStockClient = func(datafetch.TWStockConfig) (twStockClient, error) { return nil, factoryErr }
	t.Cleanup(func() { newTWStockClient = original })

	err := runFetchCommand(ctx, []string{"tw", "quotes"})
	if err == nil || !strings.HasPrefix(err.Error(), "fetch tw:") {
		t.Fatalf("error = %v, want a \"fetch tw:\" prefixed factory error", err)
	}
	if !errors.Is(err, factoryErr) {
		t.Errorf("error %q should wrap the factory error", err)
	}
}

// --- 1.3 factory configuration --------------------------------------------

// Scenario: Default interval
func TestRunFetchCommand_TWDefaultFactoryConfig(t *testing.T) {
	ctx := fetchTWContext(t)
	fake := &fakeTWClient{table: twTestTable()}
	seen := installFakeTWClient(t, fake)

	if err := runFetchCommand(ctx, []string{"tw", "quotes"}); err != nil {
		t.Fatalf("fetch tw quotes failed: %v", err)
	}
	if seen.Interval != 300*time.Millisecond {
		t.Errorf("Interval = %v, want 300ms", seen.Interval)
	}
	if seen.Retries != 2 {
		t.Errorf("Retries = %d, want 2", seen.Retries)
	}
}

// Scenario: Interval from config
func TestRunFetchCommand_TWIntervalFromConfig(t *testing.T) {
	ctx := fetchTWContext(t)
	if err := runConfigCommand(ctx, []string{"fetch.tw.interval_ms", "1000"}); err != nil {
		t.Fatalf("config fetch.tw.interval_ms failed: %v", err)
	}
	fake := &fakeTWClient{table: twTestTable()}
	seen := installFakeTWClient(t, fake)

	if err := runFetchCommand(ctx, []string{"tw", "quotes"}); err != nil {
		t.Fatalf("fetch tw quotes failed: %v", err)
	}
	if seen.Interval != time.Second {
		t.Errorf("Interval = %v, want 1s", seen.Interval)
	}
	if seen.Retries != 2 {
		t.Errorf("Retries = %d, want 2", seen.Retries)
	}
}

func TestRunFetchCommand_TWIntervalConfigRejectsNonInteger(t *testing.T) {
	ctx := fetchTWContext(t)
	err := runConfigCommand(ctx, []string{"fetch.tw.interval_ms", "fast"})
	if err == nil || !strings.Contains(err.Error(), "fetch.tw.interval_ms") {
		t.Fatalf("error = %v, want a rejection naming the key", err)
	}
}

func TestRunFetchCommand_TWNegativeIntervalRejected(t *testing.T) {
	ctx := fetchTWContext(t)
	if err := runConfigCommand(ctx, []string{"fetch.tw.interval_ms", "-5"}); err == nil {
		t.Fatal("expected a negative interval to be rejected")
	}
}

// --- 1.4 the yahoo source is untouched -------------------------------------

// Scenario: Yahoo forms unchanged. A real `fetch yahoo AAPL quote` needs the
// network, so the dispatch boundary is asserted instead: `yahoo` still reaches
// the yahoo branch (its own method error proves it) and never the tw client.
func TestRunFetchCommand_YahooDispatchUnchanged(t *testing.T) {
	ctx := fetchTWContext(t)
	fake := &fakeTWClient{table: twTestTable()}
	installFakeTWClient(t, fake)

	err := runFetchCommand(ctx, []string{"yahoo", "AAPL", "nosuchmethod", "extra", "as", "q"})
	if err == nil || !strings.Contains(err.Error(), "unsupported yahoo method: nosuchmethod") {
		t.Fatalf("error = %v, want the unchanged yahoo method error", err)
	}
	if len(fake.calls) != 0 {
		t.Errorf("the tw client must not be reached by a yahoo command")
	}

	if err := runFetchCommand(ctx, []string{"yahoo", "AAPL"}); err == nil ||
		!strings.Contains(err.Error(), "usage: fetch yahoo <ticker> <method> [params...] [as <var>]") {
		t.Fatalf("error = %v, want the unchanged yahoo usage line", err)
	}

	// The documented three-token form (`fetch yahoo AAPL quote as q` after
	// parseAlias strips the alias) must reach the method branch, not the
	// usage line — this was broken before the off-by-one fix in fetch.go.
	if err := runFetchCommand(ctx, []string{"yahoo", "AAPL", "nosuchmethod", "as", "q"}); err == nil ||
		!strings.Contains(err.Error(), "unsupported yahoo method: nosuchmethod") {
		t.Fatalf("error = %v, want the method error for the three-token form", err)
	}
}

func TestRunFetchCommand_UnsupportedProvider(t *testing.T) {
	ctx := fetchTWContext(t)
	fake := &fakeTWClient{table: twTestTable()}
	installFakeTWClient(t, fake)

	err := runFetchCommand(ctx, []string{"bloomberg", "AAPL", "quote", "extra"})
	if err == nil || !strings.Contains(err.Error(), "unsupported provider: bloomberg") {
		t.Fatalf("error = %v, want an unsupported-provider error", err)
	}
	if len(fake.calls) != 0 {
		t.Errorf("the tw client must not be reached by an unknown provider")
	}
}
