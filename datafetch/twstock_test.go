package datafetch

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/HazelnutParadise/insyra"
)

type fixtureReply struct {
	status int
	body   []byte
}

type fixtureRequest struct {
	key  string
	at   time.Time
	user string
}

type fixtureTransport struct {
	mu       sync.Mutex
	replies  map[string][]fixtureReply
	requests []fixtureRequest
}

func newFixtureTransport() *fixtureTransport {
	return &fixtureTransport{replies: make(map[string][]fixtureReply)}
}

func (f *fixtureTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	key := req.URL.EscapedPath()
	if req.URL.RawQuery != "" {
		key += "?" + req.URL.Query().Encode()
	}
	f.mu.Lock()
	f.requests = append(f.requests, fixtureRequest{key: key, at: time.Now(), user: req.Header.Get("User-Agent")})
	replies := f.replies[key]
	if len(replies) == 0 {
		f.mu.Unlock()
		return nil, fmt.Errorf("fixture has no response for %s", key)
	}
	reply := replies[0]
	f.replies[key] = replies[1:]
	f.mu.Unlock()

	return &http.Response{
		StatusCode: reply.status,
		Status:     fmt.Sprintf("%d", reply.status),
		Header:     make(http.Header),
		Body:       io.NopCloser(bytes.NewReader(reply.body)),
		Request:    req,
	}, nil
}

func (f *fixtureTransport) addFixture(t *testing.T, key, name string) {
	t.Helper()
	body, err := os.ReadFile(filepath.Join("testdata", "twstock", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	f.replies[key] = append(f.replies[key], fixtureReply{status: http.StatusOK, body: body})
}

func (f *fixtureTransport) addStatus(key string, status int) {
	f.replies[key] = append(f.replies[key], fixtureReply{status: status, body: []byte(`{"stat":"temporary failure"}`)})
}

func (f *fixtureTransport) requestTimes() []time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	result := make([]time.Time, len(f.requests))
	for i, request := range f.requests {
		result[i] = request.at
	}
	return result
}

func (f *fixtureTransport) requestCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.requests)
}

func (f *fixtureTransport) lastUserAgent() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.requests) == 0 {
		return ""
	}
	return f.requests[len(f.requests)-1].user
}

func fixtureKey(path string, query url.Values) string {
	if query == nil || query.Encode() == "" {
		return path
	}
	return path + "?" + query.Encode()
}

func newFixtureTWStock(t *testing.T, cfg TWStockConfig) (*TWStockClient, *fixtureTransport) {
	t.Helper()
	stock, err := TWStock(cfg)
	if err != nil {
		t.Fatalf("TWStock returned error: %v", err)
	}
	transport := newFixtureTransport()
	stock.client.Transport = transport
	stock.twseBaseURL = "https://twse.test"
	stock.tpexBaseURL = "https://tpex.test"
	stock.twseOpenAPIBaseURL = "https://twse-openapi.test"
	stock.tpexOpenAPIBaseURL = "https://tpex-openapi.test"
	stock.tdccBaseURL = "https://tdcc.test"
	stock.mopsBaseURL = "https://mops.test"
	return stock, transport
}

func query(values ...string) url.Values {
	result := url.Values{}
	for i := 0; i < len(values); i += 2 {
		result.Set(values[i], values[i+1])
	}
	return result
}

// slowFirstRequest holds the first request it forwards back by delay, the way a
// first request that takes longer than the second to reach the transport does.
type slowFirstRequest struct {
	next  http.RoundTripper
	delay time.Duration
	once  sync.Once
}

func (s *slowFirstRequest) RoundTrip(req *http.Request) (*http.Response, error) {
	s.once.Do(func() { time.Sleep(s.delay) })
	return s.next.RoundTrip(req)
}

func TestTWStockThrottle(t *testing.T) {
	const interval = 20 * time.Millisecond
	stock, transport := newFixtureTWStock(t, TWStockConfig{Interval: interval})
	key := fixtureKey("/v1/exchangeReport/STOCK_DAY_ALL", nil)
	transport.addFixture(t, key, "twse_stock_day_all.json")
	transport.addFixture(t, key, "twse_stock_day_all.json")
	// Hold the first request back on its way to the transport. Two transport
	// timestamps taken this way normally come out less than interval apart, the
	// condition the old assertion flaked on, so the assertion below has to hold
	// under it. The delay stays well short of interval, so the second request
	// still has to wait for its slot and a missing throttle still fails.
	stock.client.Transport = &slowFirstRequest{next: transport, delay: 5 * time.Millisecond}
	// The limiter spaces the scheduled starts of requests, and no request starts
	// before its slot. The first slot is no earlier than start, so the second
	// request cannot reach the transport before start+interval. Measuring from
	// the first request's transport timestamp instead would count that request's
	// own time between its slot and the transport, which the limiter does not
	// control, and could come out a few microseconds short.
	start := time.Now()
	if _, err := stock.AllDailyQuotes(TWMarketTWSE); err != nil {
		t.Fatalf("first AllDailyQuotes error: %v", err)
	}
	if _, err := stock.AllDailyQuotes(TWMarketTWSE); err != nil {
		t.Fatalf("second AllDailyQuotes error: %v", err)
	}
	times := transport.requestTimes()
	if len(times) != 2 {
		t.Fatalf("the transport saw %d requests, want 2", len(times))
	}
	if gap := times[1].Sub(start); gap < interval {
		t.Fatalf("the second request reached the transport %v after the first was scheduled, want at least %v", gap, interval)
	}
}

func TestTWStockRetryThenFail(t *testing.T) {
	stock, transport := newFixtureTWStock(t, TWStockConfig{Retries: 2, RetryBackoff: time.Nanosecond})
	key := fixtureKey("/v1/exchangeReport/STOCK_DAY_ALL", nil)
	transport.addStatus(key, http.StatusInternalServerError)
	transport.addStatus(key, http.StatusInternalServerError)
	transport.addStatus(key, http.StatusInternalServerError)
	_, err := stock.AllDailyQuotes(TWMarketTWSE)
	if err == nil || !strings.Contains(err.Error(), "500") {
		t.Fatalf("AllDailyQuotes error = %v, want status 500", err)
	}
	if got := transport.requestCount(); got != 3 {
		t.Fatalf("request count = %d, want 3", got)
	}
}

func TestTWStockAutoFallsThroughToTPEx(t *testing.T) {
	stock, transport := newFixtureTWStock(t, TWStockConfig{})
	twseKey := fixtureKey("/rwd/zh/afterTrading/STOCK_DAY", query("date", "20260801", "response", "json", "stockNo", "6488"))
	tpexKey := fixtureKey("/www/zh-tw/afterTrading/tradingStock", query("code", "6488", "date", "2026/08/01", "response", "json"))
	transport.addFixture(t, twseKey, "twse_stock_day_nodata.json")
	transport.addFixture(t, tpexKey, "tpex_trading_stock_6488_202608.json")
	dt, err := stock.DailyPrices("6488", dateUTC(2026, time.August, 3), dateUTC(2026, time.August, 4), TWMarketAuto)
	if err != nil {
		t.Fatalf("DailyPrices error: %v", err)
	}
	if dt.NumRows() != 2 || dt.GetColByName("Market").Data()[0] != "TPEx" {
		t.Fatalf("auto result rows/market = %d/%v", dt.NumRows(), dt.GetColByName("Market").Data())
	}
}

func TestTWStockDailyPricesTwoMonthRange(t *testing.T) {
	stock, transport := newFixtureTWStock(t, TWStockConfig{})
	transport.addFixture(t, fixtureKey("/rwd/zh/afterTrading/STOCK_DAY", query("date", "20260801", "response", "json", "stockNo", "2330")), "twse_stock_day_2330_202608.json")
	transport.addFixture(t, fixtureKey("/rwd/zh/afterTrading/STOCK_DAY", query("date", "20260901", "response", "json", "stockNo", "2330")), "twse_stock_day_2330_202609.json")
	dt, err := stock.DailyPrices("2330", dateUTC(2026, time.August, 15), dateUTC(2026, time.September, 3), TWMarketTWSE)
	if err != nil {
		t.Fatalf("DailyPrices error: %v", err)
	}
	if got := transport.requestCount(); got != 2 {
		t.Fatalf("request count = %d, want 2", got)
	}
	if dt.NumRows() != 14 {
		t.Fatalf("rows = %d, want 14", dt.NumRows())
	}
	dateValues := dt.GetColByName("Date").Data()
	if _, ok := dateValues[0].(time.Time); !ok {
		t.Fatalf("Date type = %T, want time.Time", dateValues[0])
	}
	if !dateValues[0].(time.Time).Equal(dateUTC(2026, time.August, 17)) || !dateValues[len(dateValues)-1].(time.Time).Equal(dateUTC(2026, time.September, 3)) {
		t.Fatalf("date range = %v to %v", dateValues[0], dateValues[len(dateValues)-1])
	}
	if _, ok := dt.GetColByName("Close").Data()[0].(float64); !ok {
		t.Fatalf("Close type = %T, want float64", dt.GetColByName("Close").Data()[0])
	}
	if _, ok := dt.GetColByName("Volume").Data()[0].(int64); !ok {
		t.Fatalf("Volume type = %T, want int64", dt.GetColByName("Volume").Data()[0])
	}
}

func TestTWStockTPExDailyPricesSameSchema(t *testing.T) {
	stock, transport := newFixtureTWStock(t, TWStockConfig{})
	transport.addFixture(t, fixtureKey("/www/zh-tw/afterTrading/tradingStock", query("code", "6488", "date", "2026/08/01", "response", "json")), "tpex_trading_stock_6488_202608.json")
	dt, err := stock.DailyPrices("6488", dateUTC(2026, time.August, 3), dateUTC(2026, time.August, 4), TWMarketTPEx)
	if err != nil {
		t.Fatalf("DailyPrices error: %v", err)
	}
	for _, name := range []string{"Date", "Code", "Volume", "Turnover", "Open", "High", "Low", "Close", "Change", "Transactions", "Market"} {
		if dt.GetColByName(name) == nil {
			t.Errorf("missing column %q", name)
		}
	}
	if _, ok := dt.GetColByName("Close").Data()[0].(float64); !ok {
		t.Errorf("Close type = %T, want float64", dt.GetColByName("Close").Data()[0])
	}
	if _, ok := dt.GetColByName("Volume").Data()[0].(int64); !ok {
		t.Errorf("Volume type = %T, want int64", dt.GetColByName("Volume").Data()[0])
	}
}

func TestTWStockNonNumericCellsBecomeNil(t *testing.T) {
	stock, transport := newFixtureTWStock(t, TWStockConfig{})
	transport.addFixture(t, fixtureKey("/rwd/zh/afterTrading/STOCK_DAY", query("date", "20260801", "response", "json", "stockNo", "2330")), "twse_stock_day_2330_202608.json")
	dt, err := stock.DailyPrices("2330", dateUTC(2026, time.August, 3), dateUTC(2026, time.August, 3), TWMarketTWSE)
	if err != nil {
		t.Fatalf("DailyPrices error: %v", err)
	}
	if got := dt.GetColByName("Change").Data()[0]; got == nil {
		t.Fatalf("fixture's first Change is numeric; got nil")
	}

	custom := []byte(`{"stat":"OK","fields":["日期","成交股數","成交金額","開盤價","最高價","最低價","收盤價","漲跌價差","成交筆數","註記"],"data":[["115/08/03","1,000","2,000","--","10.00","9.00","X","X","5",""]]}`)
	key := fixtureKey("/rwd/zh/afterTrading/STOCK_DAY", query("date", "20260801", "response", "json", "stockNo", "9999"))
	transport.replies[key] = []fixtureReply{{status: http.StatusOK, body: custom}}
	dt, err = stock.DailyPrices("9999", dateUTC(2026, time.August, 3), dateUTC(2026, time.August, 3), TWMarketTWSE)
	if err != nil {
		t.Fatalf("custom DailyPrices error: %v", err)
	}
	if dt.GetColByName("Open").Data()[0] != nil || dt.GetColByName("Close").Data()[0] != nil || dt.GetColByName("Change").Data()[0] != nil {
		t.Fatalf("non-numeric cells were not nil: open=%v close=%v change=%v", dt.GetColByName("Open").Data()[0], dt.GetColByName("Close").Data()[0], dt.GetColByName("Change").Data()[0])
	}
}

func TestTWStockInstitutionalTrades(t *testing.T) {
	stock, transport := newFixtureTWStock(t, TWStockConfig{})
	transport.addFixture(t, fixtureKey("/rwd/zh/fund/T86", query("date", "20260903", "response", "json", "selectType", "ALL")), "twse_t86_20260903.json")
	dt, err := stock.InstitutionalTrades(dateUTC(2026, time.September, 3), TWMarketTWSE)
	if err != nil {
		t.Fatalf("InstitutionalTrades error: %v", err)
	}
	for _, name := range []string{"Date", "Code", "Name", "ForeignNet", "TrustNet", "DealerNet", "TotalNet"} {
		if dt.GetColByName(name) == nil {
			t.Errorf("missing column %q", name)
		}
	}
	if got := dt.GetColByName("Date").Data()[0].(time.Time); !got.Equal(dateUTC(2026, time.September, 3)) {
		t.Errorf("Date = %v", got)
	}
	if _, ok := dt.GetColByName("TotalNet").Data()[0].(int64); !ok {
		t.Errorf("TotalNet type = %T, want int64", dt.GetColByName("TotalNet").Data()[0])
	}
}

func TestTWStockMarginBalance(t *testing.T) {
	stock, transport := newFixtureTWStock(t, TWStockConfig{})
	transport.addFixture(t, fixtureKey("/rwd/zh/marginTrading/MI_MARGN", query("date", "20260903", "response", "json", "selectType", "ALL")), "twse_mi_margn_20260903.json")
	dt, err := stock.MarginBalance(dateUTC(2026, time.September, 3), TWMarketTWSE)
	if err != nil {
		t.Fatalf("MarginBalance error: %v", err)
	}
	for _, name := range []string{"Date", "Code", "Name", "MarginBalance", "ShortBalance"} {
		if dt.GetColByName(name) == nil {
			t.Errorf("missing column %q", name)
		}
	}
	if _, ok := dt.GetColByName("MarginBalance").Data()[0].(int64); !ok {
		t.Errorf("MarginBalance type = %T, want int64", dt.GetColByName("MarginBalance").Data()[0])
	}
}

func TestTWStockAllDailyQuotes(t *testing.T) {
	stock, transport := newFixtureTWStock(t, TWStockConfig{})
	transport.addFixture(t, "/v1/exchangeReport/STOCK_DAY_ALL", "twse_stock_day_all.json")
	dt, err := stock.AllDailyQuotes(TWMarketTWSE)
	if err != nil {
		t.Fatalf("AllDailyQuotes error: %v", err)
	}
	if dt.NumRows() != 25 {
		t.Fatalf("rows = %d, want fixture row count 25", dt.NumRows())
	}
	if got := dt.GetColByName("Date").Data()[0].(time.Time); !got.Equal(dateUTC(2026, time.September, 3)) {
		t.Errorf("Date = %v, want 2026-09-03", got)
	}
	if _, ok := dt.GetColByName("Open").Data()[0].(float64); !ok {
		t.Errorf("Open type = %T, want float64", dt.GetColByName("Open").Data()[0])
	}
}

func TestTWStockTPExAllDailyQuotes(t *testing.T) {
	stock, transport := newFixtureTWStock(t, TWStockConfig{})
	transport.addFixture(t, "/openapi/v1/tpex_mainboard_daily_close_quotes", "tpex_daily_close_quotes.json")
	dt, err := stock.AllDailyQuotes(TWMarketTPEx)
	if err != nil {
		t.Fatalf("TPEx AllDailyQuotes error: %v", err)
	}
	if dt.NumRows() != 25 {
		t.Fatalf("rows = %d, want fixture row count 25", dt.NumRows())
	}
	if got := dt.GetColByName("Date").Data()[0].(time.Time); !got.Equal(dateUTC(2026, time.September, 4)) {
		t.Errorf("Date = %v, want 2026-09-04", got)
	}
}

func TestTWStockRejectsInvalidDailyPriceRange(t *testing.T) {
	stock, _ := newFixtureTWStock(t, TWStockConfig{})
	if _, err := stock.DailyPrices("2330", dateUTC(2026, time.September, 3), dateUTC(2026, time.August, 3), TWMarketTWSE); err == nil {
		t.Error("DailyPrices accepted from > to")
	}
	if _, err := stock.DailyPrices("", dateUTC(2026, time.August, 3), dateUTC(2026, time.August, 3), TWMarketTWSE); err == nil {
		t.Error("DailyPrices accepted empty code")
	}
}

func TestTWStockConfigDefaultsAndUserAgent(t *testing.T) {
	stock, _ := newFixtureTWStock(t, TWStockConfig{})
	if stock.cfg.Timeout != defaultTWStockTimeout || stock.cfg.RetryBackoff != defaultTWStockBackoff || stock.cfg.Concurrency != defaultTWStockConcurrency {
		t.Fatalf("config defaults = %+v", stock.cfg)
	}
	key := "/v1/exchangeReport/STOCK_DAY_ALL"
	transport := stock.client.Transport.(*fixtureTransport)
	transport.addFixture(t, key, "twse_stock_day_all.json")
	if _, err := stock.AllDailyQuotes(TWMarketTWSE); err != nil {
		t.Fatalf("AllDailyQuotes error: %v", err)
	}
	// Derived from insyra.Version, not hard-coded: a literal here breaks on
	// every release, which is exactly what happened cutting v0.3.2.
	wantUA := "insyra-datafetch/" + insyra.Version
	if got := transport.lastUserAgent(); got != wantUA {
		t.Errorf("User-Agent = %q, want %q", got, wantUA)
	}
}

func TestTWStockTPExInstitutionalAndMarginMappings(t *testing.T) {
	stock, transport := newFixtureTWStock(t, TWStockConfig{})
	transport.addFixture(t, fixtureKey("/www/zh-tw/insti/dailyTrade", query("date", "2026/09/03", "response", "json", "type", "Daily")), "tpex_insti_daily_20260903.json")
	transport.addFixture(t, fixtureKey("/www/zh-tw/margin/balance", query("date", "2026/09/03", "response", "json")), "tpex_margin_balance_20260903.json")
	inst, err := stock.InstitutionalTrades(dateUTC(2026, time.September, 3), TWMarketTPEx)
	if err != nil {
		t.Fatalf("TPEx InstitutionalTrades error: %v", err)
	}
	margin, err := stock.MarginBalance(dateUTC(2026, time.September, 3), TWMarketTPEx)
	if err != nil {
		t.Fatalf("TPEx MarginBalance error: %v", err)
	}
	if _, ok := inst.GetColByName("ForeignNet").Data()[0].(int64); !ok {
		t.Errorf("TPEx ForeignNet type = %T, want int64", inst.GetColByName("ForeignNet").Data()[0])
	}
	// Fixture row 0 (00411A): foreign ex-dealer -689,550, trust 0, dealer total
	// -139,119, grand total -828,669. Pins the triplet column positions.
	for name, want := range map[string]int64{"ForeignNet": -689550, "TrustNet": 0, "DealerNet": -139119, "TotalNet": -828669} {
		if got := inst.GetColByName(name).Data()[0]; got != want {
			t.Errorf("TPEx %s = %v, want %d", name, got, want)
		}
	}
	if _, ok := margin.GetColByName("ShortBalance").Data()[0].(int64); !ok {
		t.Errorf("TPEx ShortBalance type = %T, want int64", margin.GetColByName("ShortBalance").Data()[0])
	}
}

func dateUTC(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func exRightsBody(rows ...string) []byte {
	return []byte(`{"stat":"OK","fields":["資料日期","股票代號","股票名稱","除權息前收盤價","除權息參考價","權值+息值","權/息"],"data":[` + strings.Join(rows, ",") + `]}`)
}

func exRightsKey(start, end string) string {
	return fixtureKey("/rwd/zh/exRight/TWT49U", query("startDate", start, "endDate", end, "response", "json"))
}

func TestTWStockExRightsFixture(t *testing.T) {
	stock, transport := newFixtureTWStock(t, TWStockConfig{})
	transport.addFixture(t, exRightsKey("20260601", "20260903"), "twse_twt49u_20260601_20260903.json")
	dt, err := stock.ExRights(dateUTC(2026, time.June, 1), dateUTC(2026, time.September, 3), TWMarketTWSE)
	if err != nil {
		t.Fatalf("ExRights error: %v", err)
	}
	for _, name := range []string{"Date", "Code", "Name", "PrevClose", "RefPrice", "Distribution", "Kind", "AdjFactor"} {
		if dt.GetColByName(name) == nil {
			t.Fatalf("missing column %q", name)
		}
	}
	if dt.NumRows() != 25 {
		t.Fatalf("rows = %d, want fixture row count 25", dt.NumRows())
	}
	if got := dt.GetColByName("Code").Data()[0]; got != "2612" {
		t.Errorf("Code = %v, want 2612", got)
	}
	if got := dt.GetColByName("Date").Data()[0].(time.Time); !got.Equal(dateUTC(2026, time.June, 1)) {
		t.Errorf("Date = %v, want 2026-06-01", got)
	}
	if got := dt.GetColByName("PrevClose").Data()[0]; got != 57.7 {
		t.Errorf("PrevClose = %v, want 57.7", got)
	}
	if got := dt.GetColByName("RefPrice").Data()[0]; got != 55.5 {
		t.Errorf("RefPrice = %v, want 55.5", got)
	}
	if got := dt.GetColByName("Distribution").Data()[0]; got != 2.2 {
		t.Errorf("Distribution = %v, want 2.2", got)
	}
	if got := dt.GetColByName("Kind").Data()[0]; got != "dividend" {
		t.Errorf("Kind = %v, want dividend", got)
	}
	factor, ok := dt.GetColByName("AdjFactor").Data()[0].(float64)
	if !ok {
		t.Fatalf("AdjFactor type = %T, want float64", dt.GetColByName("AdjFactor").Data()[0])
	}
	if math.Abs(factor-55.5/57.7) > 1e-12 {
		t.Errorf("AdjFactor = %v, want %v", factor, 55.5/57.7)
	}
}

func TestTWStockExRightsKinds(t *testing.T) {
	stock, transport := newFixtureTWStock(t, TWStockConfig{})
	transport.replies[exRightsKey("20260601", "20260603")] = []fixtureReply{{status: http.StatusOK, body: exRightsBody(
		`["115年06月01日","1101","甲","100.00","98.00","2.000000","息"]`,
		`["115年06月02日","1102","乙","100.00","95.00","5.000000","權"]`,
		`["115年06月03日","1103","丙","100.00","90.00","10.000000","權息"]`,
	)}}
	dt, err := stock.ExRights(dateUTC(2026, time.June, 1), dateUTC(2026, time.June, 3), TWMarketTWSE)
	if err != nil {
		t.Fatalf("ExRights error: %v", err)
	}
	want := []any{"dividend", "rights", "both"}
	for i, kind := range dt.GetColByName("Kind").Data() {
		if kind != want[i] {
			t.Errorf("Kind[%d] = %v, want %v", i, kind, want[i])
		}
	}
}

func TestTWStockExRightsPagesByYear(t *testing.T) {
	stock, transport := newFixtureTWStock(t, TWStockConfig{})
	transport.replies[exRightsKey("20240101", "20241231")] = []fixtureReply{{status: http.StatusOK, body: exRightsBody(
		`["113年07月01日","1101","甲","100.00","98.00","2.000000","息"]`)}}
	transport.replies[exRightsKey("20250101", "20251231")] = []fixtureReply{{status: http.StatusOK, body: exRightsBody(
		`["114年07月01日","1101","甲","100.00","98.00","2.000000","息"]`)}}
	transport.replies[exRightsKey("20260101", "20260903")] = []fixtureReply{{status: http.StatusOK, body: exRightsBody(
		`["115年07月01日","1101","甲","100.00","98.00","2.000000","息"]`)}}
	dt, err := stock.ExRights(dateUTC(2024, time.January, 1), dateUTC(2026, time.September, 3), TWMarketTWSE)
	if err != nil {
		t.Fatalf("ExRights error: %v", err)
	}
	if got := transport.requestCount(); got != 3 {
		t.Fatalf("request count = %d, want 3", got)
	}
	transport.mu.Lock()
	keys := make([]string, len(transport.requests))
	for i, request := range transport.requests {
		keys[i] = request.key
	}
	transport.mu.Unlock()
	wantKeys := []string{exRightsKey("20240101", "20241231"), exRightsKey("20250101", "20251231"), exRightsKey("20260101", "20260903")}
	for i, want := range wantKeys {
		if keys[i] != want {
			t.Fatalf("request %d = %s, want %s", i, keys[i], want)
		}
	}
	dates := dt.GetColByName("Date").Data()
	if len(dates) != 3 {
		t.Fatalf("rows = %d, want 3", len(dates))
	}
	for i := 1; i < len(dates); i++ {
		if !dates[i-1].(time.Time).Before(dates[i].(time.Time)) {
			t.Fatalf("dates not ascending: %v", dates)
		}
	}
}

func TestTWStockExRightsMissingHeader(t *testing.T) {
	stock, transport := newFixtureTWStock(t, TWStockConfig{})
	body := []byte(`{"stat":"OK","fields":["資料日期","股票代號","股票名稱","除權息前收盤價","權值+息值","權/息"],"data":[["115年06月01日","1101","甲","100.00","2.000000","息"]]}`)
	transport.replies[exRightsKey("20260601", "20260601")] = []fixtureReply{{status: http.StatusOK, body: body}}
	_, err := stock.ExRights(dateUTC(2026, time.June, 1), dateUTC(2026, time.June, 1), TWMarketTWSE)
	if err == nil || !strings.Contains(err.Error(), "除權息參考價") {
		t.Fatalf("ExRights error = %v, want missing 除權息參考價", err)
	}
}

func TestTWStockExRightsRejectsInvalidRangeAndTPEx(t *testing.T) {
	stock, _ := newFixtureTWStock(t, TWStockConfig{})
	if _, err := stock.ExRights(dateUTC(2026, time.September, 3), dateUTC(2026, time.June, 1), TWMarketTWSE); err == nil {
		t.Error("ExRights accepted from > to")
	}
	_, err := stock.ExRights(dateUTC(2026, time.June, 1), dateUTC(2026, time.September, 3), TWMarketTPEx)
	if err == nil || !strings.Contains(err.Error(), "not supported") {
		t.Fatalf("TPEx ExRights error = %v, want \"not supported\"", err)
	}
	if _, err := stock.ExRights(dateUTC(2026, time.June, 1), dateUTC(2026, time.September, 3), TWMarket("NYSE")); err == nil {
		t.Error("ExRights accepted an unsupported market")
	}
	_, err = stock.DailyPricesAdjusted("6488", dateUTC(2026, time.August, 1), dateUTC(2026, time.August, 31), TWMarketTPEx)
	if err == nil || !strings.Contains(err.Error(), "not supported") {
		t.Fatalf("TPEx DailyPricesAdjusted error = %v, want \"not supported\"", err)
	}
	if _, err := stock.DailyPricesAdjusted("2330", dateUTC(2026, time.September, 3), dateUTC(2026, time.August, 3), TWMarketTWSE); err == nil {
		t.Error("DailyPricesAdjusted accepted from > to")
	}
}

// Auto may resolve an OTC code to TPEx prices; those bars cannot be adjusted
// from the TWSE ex-rights table, so the call must refuse instead of returning
// silently unadjusted prices.
func TestTWStockDailyPricesAdjustedRefusesAutoResolvedTPEx(t *testing.T) {
	stock, transport := newFixtureTWStock(t, TWStockConfig{})
	transport.addFixture(t, stockDayKey("20260801", "6488"), "twse_stock_day_nodata.json")
	transport.addFixture(t, fixtureKey("/www/zh-tw/afterTrading/tradingStock", query("code", "6488", "date", "2026/08/01", "response", "json")), "tpex_trading_stock_6488_202608.json")
	_, err := stock.DailyPricesAdjusted("6488", dateUTC(2026, time.August, 3), dateUTC(2026, time.August, 4), TWMarketAuto)
	if err == nil || !strings.Contains(err.Error(), "not supported") {
		t.Fatalf("Auto-resolved TPEx DailyPricesAdjusted error = %v, want \"not supported\"", err)
	}
}

func stockDayBody(rows ...string) []byte {
	return []byte(`{"stat":"OK","fields":["日期","成交股數","成交金額","開盤價","最高價","最低價","收盤價","漲跌價差","成交筆數"],"data":[` + strings.Join(rows, ",") + `]}`)
}

func stockDayKey(date, code string) string {
	return fixtureKey("/rwd/zh/afterTrading/STOCK_DAY", query("date", date, "response", "json", "stockNo", code))
}

func adjustedFixture(t *testing.T, prices, exRights []byte) *insyra.DataTable {
	t.Helper()
	stock, transport := newFixtureTWStock(t, TWStockConfig{})
	transport.replies[stockDayKey("20260801", "2330")] = []fixtureReply{{status: http.StatusOK, body: prices}}
	transport.replies[exRightsKey("20260801", "20260831")] = []fixtureReply{{status: http.StatusOK, body: exRights}}
	dt, err := stock.DailyPricesAdjusted("2330", dateUTC(2026, time.August, 1), dateUTC(2026, time.August, 31), TWMarketTWSE)
	if err != nil {
		t.Fatalf("DailyPricesAdjusted error: %v", err)
	}
	return dt
}

func TestTWStockDailyPricesAdjustedOneExDate(t *testing.T) {
	dt := adjustedFixture(t,
		stockDayBody(
			`["115/08/14","1,000","100,000","100.00","100.00","100.00","100.00","0.00","10"]`,
			`["115/08/15","1,000","98,000","98.00","98.00","98.00","98.00","-2.00","10"]`,
		),
		exRightsBody(`["115年08月15日","2330","台積電","100.00","98.00","2.000000","息"]`),
	)
	for _, name := range []string{"Date", "Code", "Volume", "Turnover", "Open", "High", "Low", "Close", "Change", "Transactions", "Market", "AdjFactor", "AdjOpen", "AdjHigh", "AdjLow", "AdjClose"} {
		if dt.GetColByName(name) == nil {
			t.Fatalf("missing column %q", name)
		}
	}
	factors := dt.GetColByName("AdjFactor").Data()
	adjClose := dt.GetColByName("AdjClose").Data()
	closes := dt.GetColByName("Close").Data()
	if dt.NumRows() != 2 {
		t.Fatalf("rows = %d, want 2", dt.NumRows())
	}
	if math.Abs(factors[0].(float64)-0.98) > 1e-12 || factors[1].(float64) != 1 {
		t.Fatalf("AdjFactor = %v", factors)
	}
	if math.Abs(adjClose[0].(float64)-98) > 1e-12 || adjClose[1].(float64) != 98 {
		t.Fatalf("AdjClose = %v", adjClose)
	}
	// Ex-dividend day: the raw close falls 2% while the adjusted return is 0.
	rawReturn := closes[1].(float64)/closes[0].(float64) - 1
	adjReturn := adjClose[1].(float64)/adjClose[0].(float64) - 1
	if math.Abs(rawReturn+0.02) > 1e-12 {
		t.Errorf("raw return = %v, want -0.02", rawReturn)
	}
	if math.Abs(adjReturn) > 1e-12 {
		t.Errorf("adjusted return = %v, want 0", adjReturn)
	}
}

func TestTWStockDailyPricesAdjustedCompoundsTwoExDates(t *testing.T) {
	dt := adjustedFixture(t,
		stockDayBody(
			`["115/08/10","1,000","100,000","100.00","100.00","100.00","100.00","0.00","10"]`,
			`["115/08/15","1,000","100,000","100.00","100.00","100.00","100.00","0.00","10"]`,
			`["115/08/20","1,000","100,000","100.00","100.00","100.00","100.00","0.00","10"]`,
		),
		exRightsBody(
			`["115年08月15日","2330","台積電","100.00","98.00","2.000000","息"]`,
			`["115年08月20日","2330","台積電","100.00","95.00","5.000000","息"]`,
		),
	)
	factors := dt.GetColByName("AdjFactor").Data()
	if math.Abs(factors[0].(float64)-0.98*0.95) > 1e-12 {
		t.Errorf("AdjFactor[0] = %v, want %v", factors[0], 0.98*0.95)
	}
	if math.Abs(factors[1].(float64)-0.95) > 1e-12 {
		t.Errorf("AdjFactor[1] = %v, want 0.95", factors[1])
	}
	if factors[2].(float64) != 1 {
		t.Errorf("AdjFactor[2] = %v, want 1", factors[2])
	}
}

func TestTWStockDailyPricesAdjustedWithoutExDate(t *testing.T) {
	dt := adjustedFixture(t,
		stockDayBody(
			`["115/08/14","1,000","100,000","100.00","100.00","100.00","100.00","0.00","10"]`,
			`["115/08/15","1,000","98,000","98.00","98.00","98.00","98.00","-2.00","10"]`,
		),
		// Ex-dividend for a different code must not touch 2330.
		exRightsBody(`["115年08月15日","1101","甲","100.00","98.00","2.000000","息"]`),
	)
	for i, factor := range dt.GetColByName("AdjFactor").Data() {
		if factor.(float64) != 1 {
			t.Errorf("AdjFactor[%d] = %v, want 1", i, factor)
		}
	}
	closes := dt.GetColByName("Close").Data()
	for i, adjusted := range dt.GetColByName("AdjClose").Data() {
		if adjusted != closes[i] {
			t.Errorf("AdjClose[%d] = %v, want %v", i, adjusted, closes[i])
		}
	}
}

func TestTWStockDailyPricesAdjustedKeepsNils(t *testing.T) {
	dt := adjustedFixture(t,
		stockDayBody(
			`["115/08/14","1,000","100,000","--","100.00","100.00","100.00","0.00","10"]`,
			`["115/08/15","1,000","98,000","98.00","98.00","98.00","98.00","-2.00","10"]`,
		),
		exRightsBody(`["115年08月15日","2330","台積電","100.00","98.00","2.000000","息"]`),
	)
	if dt.GetColByName("AdjOpen").Data()[0] != nil {
		t.Errorf("AdjOpen[0] = %v, want nil", dt.GetColByName("AdjOpen").Data()[0])
	}
	if got := dt.GetColByName("AdjHigh").Data()[0]; math.Abs(got.(float64)-98) > 1e-12 {
		t.Errorf("AdjHigh[0] = %v, want 98", got)
	}
	if got := dt.GetColByName("AdjClose").Data()[0]; math.Abs(got.(float64)-98) > 1e-12 {
		t.Errorf("AdjClose[0] = %v, want 98", got)
	}
}

func TestTWStockOddLotQuotes(t *testing.T) {
	stock, transport := newFixtureTWStock(t, TWStockConfig{})
	transport.addFixture(t, fixtureKey("/rwd/zh/afterTrading/TWTC7U", query("date", "20260102", "response", "json", "selectType", "ALL")), "twse_twtc7u_20260102.json")
	transport.addFixture(t, fixtureKey("/rwd/zh/afterTrading/TWT53U", query("date", "20260102", "response", "json", "selectType", "ALL")), "twse_twt53u_20260102.json")

	intraday, err := stock.OddLotQuotes(dateUTC(2026, time.January, 2), TWOddLotIntraday)
	if err != nil {
		t.Fatalf("intraday error: %v", err)
	}
	// Fixture: 0050, 0051, 0052 and the 合計 totals row, which is dropped.
	if got := intraday.GetColByName("Code").Data(); len(got) != 3 || got[0] != "0050" {
		t.Fatalf("intraday codes = %v", got)
	}
	for name, want := range map[string]any{"Volume": int64(3257147), "Transactions": int64(36193), "Turnover": int64(216356465), "Open": 66.0, "Close": 67.0, "High": 67.0, "Low": 65.95, "BidPrice": 66.95, "BidVolume": int64(9148), "AskPrice": 67.0, "AskVolume": int64(147266)} {
		if got := intraday.GetColByName(name).Data()[0]; got != want {
			t.Errorf("intraday %s = %v (%T), want %v", name, got, got, want)
		}
	}

	after, err := stock.OddLotQuotes(dateUTC(2026, time.January, 2), TWOddLotAfterHours)
	if err != nil {
		t.Fatalf("after-hours error: %v", err)
	}
	// 0057 traded nothing, so its price is "--" and nil.
	codes := after.GetColByName("Code").Data()
	if len(codes) != 4 || codes[3] != "0057" {
		t.Fatalf("after-hours codes = %v", codes)
	}
	if got := after.GetColByName("Price").Data()[0]; got != 67.0 {
		t.Errorf("after-hours Price = %v, want 67", got)
	}
	if got := after.GetColByName("Price").Data()[3]; got != nil {
		t.Errorf("a price of -- should be nil, got %v", got)
	}
	if after.GetColByName("Open") != nil {
		t.Error("the after-hours table has no Open column")
	}
}

func TestTWStockOddLotQuotesNoDataAndBadSession(t *testing.T) {
	stock, transport := newFixtureTWStock(t, TWStockConfig{})
	transport.replies[fixtureKey("/rwd/zh/afterTrading/TWTC7U", query("date", "20260103", "response", "json", "selectType", "ALL"))] = []fixtureReply{{status: http.StatusOK, body: []byte(`{"stat":"很抱歉，沒有符合條件的資料!","total":0}`)}}
	transport.replies[fixtureKey("/rwd/zh/afterTrading/TWTC7U", query("date", "20190102", "response", "json", "selectType", "ALL"))] = []fixtureReply{{status: http.StatusOK, body: []byte(`{"stat":"查詢日期小於109年10月26日，請重新查詢!","total":0}`)}}
	for _, day := range []time.Time{dateUTC(2026, time.January, 3), dateUTC(2019, time.January, 2)} {
		dt, err := stock.OddLotQuotes(day, TWOddLotIntraday)
		if err != nil {
			t.Fatalf("%v: %v", day, err)
		}
		if dt.NumRows() != 0 || dt.GetColByName("Code") == nil {
			t.Errorf("%v: want an empty table with columns, got %d rows", day, dt.NumRows())
		}
	}
	if _, err := stock.OddLotQuotes(dateUTC(2026, time.January, 2), TWOddLotSession("Night")); err == nil {
		t.Error("an unknown session should be an error")
	}
	var zero TWStockClient
	if _, err := zero.OddLotQuotes(dateUTC(2026, time.January, 2), TWOddLotIntraday); err == nil {
		t.Error("a zero client should be an error")
	}
}

func TestTWStockTPExOddLotQuotes(t *testing.T) {
	stock, transport := newFixtureTWStock(t, TWStockConfig{})
	transport.addFixture(t, fixtureKey("/www/zh-tw/afterTrading/oddQuote", query("date", "2026/10/16", "type", "Daily", "response", "json")), "tpex_oddquote_20261016.json")
	transport.addFixture(t, fixtureKey("/www/zh-tw/afterTrading/odd", query("date", "2026/10/16", "type", "Daily", "response", "json")), "tpex_odd_20261016.json")
	day := dateUTC(2026, time.October, 16)

	intraday, err := stock.TPExOddLotQuotes(day, TWOddLotIntraday)
	if err != nil {
		t.Fatalf("intraday error: %v", err)
	}
	// Fixture: 00411A, 006201, 00679B and 00687C, which did not trade.
	if got := intraday.GetColByName("Code").Data(); len(got) != 4 || got[0] != "00411A" || got[3] != "00687C" {
		t.Fatalf("intraday codes = %v", got)
	}
	for name, want := range map[string]any{"Volume": int64(46322), "Transactions": int64(426), "Turnover": int64(471215), "Open": 10.12, "Close": 10.17, "High": 10.44, "Low": 10.12, "BidPrice": 10.17, "BidVolume": int64(130), "AskPrice": 10.2, "AskVolume": int64(560)} {
		if got := intraday.GetColByName(name).Data()[0]; got != want {
			t.Errorf("intraday %s = %v (%T), want %v", name, got, got, want)
		}
	}
	for _, name := range []string{"Open", "Close", "High", "Low", "AskPrice"} {
		if got := intraday.GetColByName(name).Data()[3]; got != nil {
			t.Errorf("a stock that did not trade has %s = %v, want nil", name, got)
		}
	}
	if intraday.GetColByName("Change") != nil || intraday.GetColByName("漲跌") != nil {
		t.Error("the change column is left out")
	}
	if got := intraday.GetColByName("Date").Data()[0]; got != day {
		t.Errorf("Date = %v, want %v", got, day)
	}

	after, err := stock.TPExOddLotQuotes(day, TWOddLotAfterHours)
	if err != nil {
		t.Fatalf("after-hours error: %v", err)
	}
	for name, want := range map[string]any{"Volume": int64(620), "Transactions": int64(5), "Turnover": int64(6322), "Price": 10.2, "BidPrice": 10.2, "BidVolume": int64(6854), "AskVolume": int64(0)} {
		if got := after.GetColByName(name).Data()[0]; got != want {
			t.Errorf("after-hours %s = %v (%T), want %v", name, got, got, want)
		}
	}
	if got := after.GetColByName("AskPrice").Data()[0]; got != nil {
		t.Errorf("an ask of 0.00 means no quote and should be nil, got %v", got)
	}
	// Same columns as the TWSE's table for the session, so the two stack.
	twse := oddLotTable(oddLotColumns[TWOddLotAfterHours], nil)
	if !reflect.DeepEqual(after.ColNames(), twse.ColNames()) {
		t.Errorf("columns %v differ from the TWSE's %v", after.ColNames(), twse.ColNames())
	}
}

func TestTWStockTPExOddLotQuotesNoDataAndBadSession(t *testing.T) {
	stock, transport := newFixtureTWStock(t, TWStockConfig{})
	empty := []byte(`{"tables":[{"title":"盤中零股每日收盤行情","date":"20261018","fields":["代號","名稱"],"data":[]}],"stat":"ok"}`)
	transport.replies[fixtureKey("/www/zh-tw/afterTrading/oddQuote", query("date", "2026/10/18", "type", "Daily", "response", "json"))] = []fixtureReply{{status: http.StatusOK, body: empty}}
	dt, err := stock.TPExOddLotQuotes(dateUTC(2026, time.October, 18), TWOddLotIntraday)
	if err != nil {
		t.Fatal(err)
	}
	if dt.NumRows() != 0 || dt.GetColByName("Code") == nil {
		t.Errorf("a day with no data should be an empty table with columns, got %d rows", dt.NumRows())
	}
	if _, err := stock.TPExOddLotQuotes(dateUTC(2026, time.October, 16), TWOddLotSession("Night")); err == nil {
		t.Error("an unknown session should be an error")
	}
	var zero TWStockClient
	if _, err := zero.TPExOddLotQuotes(dateUTC(2026, time.October, 16), TWOddLotIntraday); err == nil {
		t.Error("a zero client should be an error")
	}
	// A header the code needs but the report lacks is refused, not guessed.
	bad := []byte(`{"tables":[{"fields":["代號","名稱"],"data":[["00411A","x"]]}],"stat":"ok"}`)
	transport.replies[fixtureKey("/www/zh-tw/afterTrading/odd", query("date", "2026/10/16", "type", "Daily", "response", "json"))] = []fixtureReply{{status: http.StatusOK, body: bad}}
	if _, err := stock.TPExOddLotQuotes(dateUTC(2026, time.October, 16), TWOddLotAfterHours); err == nil {
		t.Error("a report without the required headers should be an error")
	}
}

func TestTWStockShareholdingDistribution(t *testing.T) {
	stock, transport := newFixtureTWStock(t, TWStockConfig{})
	transport.addFixture(t, "/getOD.ashx?id=1-5", "tdcc_shareholding_20261016.csv.txt")
	dt, err := stock.ShareholdingDistribution()
	if err != nil {
		t.Fatalf("ShareholdingDistribution error: %v", err)
	}
	// Fixture: securities 000218 and 0050, 17 levels each.
	if dt.NumRows() != 34 {
		t.Fatalf("rows = %d, want 34", dt.NumRows())
	}
	row := func(code string, level int64) int {
		codes, levels := dt.GetColByName("Code").Data(), dt.GetColByName("Level").Data()
		for i := range codes {
			if codes[i] == code && levels[i] == level {
				return i
			}
		}
		t.Fatalf("no row for %s level %d", code, level)
		return -1
	}
	i := row("000218", 15)
	if got := dt.GetColByName("Date").Data()[i].(time.Time); !got.Equal(dateUTC(2026, time.October, 16)) {
		t.Errorf("Date = %v", got)
	}
	for name, want := range map[string]any{"Holders": int64(1), "Shares": int64(422278902), "Percent": 100.0, "LevelLabel": "1,000,001 and above"} {
		if got := dt.GetColByName(name).Data()[i]; got != want {
			t.Errorf("%s = %v (%T), want %v", name, got, got, want)
		}
	}
	if label := dt.GetColByName("LevelLabel").Data()[row("0050", 17)]; label != "total" {
		t.Errorf("level 17 label = %v", label)
	}
	if label := dt.GetColByName("LevelLabel").Data()[row("0050", 16)]; label != "difference adjustment" {
		t.Errorf("level 16 label = %v", label)
	}
}

func TestTWStockShareholdingDistributionRefusesBadFiles(t *testing.T) {
	for name, body := range map[string]string{
		"header":    "\xef\xbb\xbf日期,代號\r\n20261016,1\r\n",
		"short row": "資料日期,證券代號,持股分級,人數,股數,占集保庫存數比例%\r\n20261016,0050,1,2\r\n",
		"bad date":  "資料日期,證券代號,持股分級,人數,股數,占集保庫存數比例%\r\n2026-10-16,0050,1,2,3,0.1\r\n",
		"bad level": "資料日期,證券代號,持股分級,人數,股數,占集保庫存數比例%\r\n20261016,0050,x,2,3,0.1\r\n",
		"empty":     "",
	} {
		stock, transport := newFixtureTransportTWStock(t)
		transport.replies["/getOD.ashx?id=1-5"] = []fixtureReply{{status: http.StatusOK, body: []byte(body)}}
		if _, err := stock.ShareholdingDistribution(); err == nil {
			t.Errorf("%s: want an error", name)
		}
	}
	stock, transport := newFixtureTransportTWStock(t)
	transport.addStatus("/getOD.ashx?id=1-5", http.StatusInternalServerError)
	if _, err := stock.ShareholdingDistribution(); err == nil {
		t.Error("a server error should be an error")
	}
	var zero TWStockClient
	if _, err := zero.ShareholdingDistribution(); err == nil {
		t.Error("a zero client should be an error")
	}
}

func newFixtureTransportTWStock(t *testing.T) (*TWStockClient, *fixtureTransport) {
	t.Helper()
	return newFixtureTWStock(t, TWStockConfig{})
}

func miIndexKey(date string) string {
	return fixtureKey("/rwd/zh/afterTrading/MI_INDEX", query("date", date, "type", "ALLBUT0999", "response", "json"))
}

func TestTWStockDailyQuotesOn(t *testing.T) {
	stock, transport := newFixtureTWStock(t, TWStockConfig{})
	transport.addFixture(t, miIndexKey("20261016"), "twse_mi_index_20261016.json")
	dt, err := stock.DailyQuotesOn(dateUTC(2026, time.October, 16), TWMarketTWSE)
	if err != nil {
		t.Fatalf("DailyQuotesOn error: %v", err)
	}
	if dt.NumRows() != 9 {
		t.Fatalf("rows = %d, want 9", dt.NumRows())
	}
	row := func(code string) int {
		for i, c := range dt.GetColByName("Code").Data() {
			if c == code {
				return i
			}
		}
		t.Fatalf("no row for %s", code)
		return -1
	}
	check := func(code string, want map[string]any) {
		for name, w := range want {
			if got := dt.GetColByName(name).Data()[row(code)]; got != w {
				t.Errorf("%s %s = %v (%T), want %v", code, name, got, got, w)
			}
		}
	}
	// 台泥 rose 0.07-style marks: the sign comes from the markup, the size from 漲跌價差.
	check("1101", map[string]any{"Sign": "+", "Close": 27.3, "Volume": int64(57437884), "Transactions": int64(16647), "Turnover": int64(1574086796), "Open": 27.1})
	check("1203", map[string]any{"Sign": "-"})
	check("1108", map[string]any{"Sign": "=", "Change": 0.0})
	check("1470", map[string]any{"Sign": "=", "Close": nil, "Volume": int64(0)})
	check("8908", map[string]any{"Sign": "X", "Change": nil})
	if got := dt.GetColByName("Change").Data()[row("1203")].(float64); got >= 0 {
		t.Errorf("a falling stock has Change %v, want it negative", got)
	}
	if got := dt.GetColByName("Date").Data()[0].(time.Time); !got.Equal(dateUTC(2026, time.October, 16)) {
		t.Errorf("Date = %v", got)
	}
}

func TestTWStockDailyQuotesOnTPEx(t *testing.T) {
	stock, transport := newFixtureTWStock(t, TWStockConfig{})
	transport.addFixture(t, fixtureKey("/www/zh-tw/afterTrading/otc", query("date", "2026/10/16", "type", "EW", "response", "json")), "tpex_daily_otc_20261016.json")
	dt, err := stock.DailyQuotesOn(dateUTC(2026, time.October, 16), TWMarketTPEx)
	if err != nil {
		t.Fatalf("TPEx DailyQuotesOn error: %v", err)
	}
	if dt.NumRows() != 6 {
		t.Fatalf("rows = %d, want 6", dt.NumRows())
	}
	signs := dt.GetColByName("Sign").Data()
	changes := dt.GetColByName("Change").Data()
	if signs[0] != "+" || changes[0] != 0.05 || signs[2] != "-" || changes[2] != -0.25 || signs[4] != "=" || signs[5] != "X" || changes[5] != nil {
		t.Errorf("signs %v changes %v", signs, changes)
	}
	if got := dt.GetColByName("Volume").Data()[0]; got != int64(13452000) {
		t.Errorf("Volume = %v, want 13452000 shares", got)
	}
}

func TestTWStockDailyQuotesOnEmptyDay(t *testing.T) {
	stock, transport := newFixtureTWStock(t, TWStockConfig{})
	transport.replies[miIndexKey("20261017")] = []fixtureReply{{status: http.StatusOK, body: []byte(`{"stat":"很抱歉，沒有符合條件的資料!","type":"ALLBUT0999"}`)}}
	dt, err := stock.DailyQuotesOn(dateUTC(2026, time.October, 17), TWMarketTWSE)
	if err != nil || dt.NumRows() != 0 || dt.GetColByName("Close") == nil {
		t.Errorf("empty day: %v rows, err %v", dt.NumRows(), err)
	}
	if _, err := stock.DailyQuotesOn(dateUTC(2026, time.October, 17), TWMarket("NYSE")); err == nil {
		t.Error("an unknown market should be an error")
	}
	var zero TWStockClient
	if _, err := zero.DailyQuotesOn(dateUTC(2026, time.October, 16), TWMarketTWSE); err == nil {
		t.Error("a zero client should be an error")
	}
}

func TestTWStockAdvanceDecline(t *testing.T) {
	stock, transport := newFixtureTWStock(t, TWStockConfig{})
	transport.addFixture(t, miIndexKey("20261016"), "twse_mi_index_20261016.json")
	transport.addFixture(t, miIndexKey("20261016"), "twse_mi_index_20261016.json")
	transport.addFixture(t, miIndexKey("20161004"), "twse_mi_index_no_breadth.json")
	transport.addFixture(t, miIndexKey("20161004"), "twse_mi_index_no_breadth.json")
	day := dateUTC(2026, time.October, 16)
	value := func(dt *insyra.DataTable, name string) any { return dt.GetColByName(name).Data()[0] }

	// The official table: the stock column when commonStockOnly, the whole market otherwise.
	stocks, err := stock.AdvanceDecline(day, TWMarketTWSE, true)
	if err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]any{"Advancing": int64(684), "AdvancingLimit": int64(19), "Declining": int64(286), "DecliningLimit": int64(0), "Unchanged": int64(106), "NoTrade": int64(5), "NoReference": int64(1), "Source": "official"} {
		if got := value(stocks, name); got != want {
			t.Errorf("stock %s = %v, want %v", name, got, want)
		}
	}
	market, err := stock.AdvanceDecline(day, TWMarketTWSE, false)
	if err != nil {
		t.Fatal(err)
	}
	if value(market, "Advancing") != int64(9016) || value(market, "AdvancingLimit") != int64(68) || value(market, "NoReference") != int64(3528) {
		t.Errorf("market counts: %v %v %v", value(market, "Advancing"), value(market, "AdvancingLimit"), value(market, "NoReference"))
	}

	// A day without the official table is counted from the closes.
	oldDay := dateUTC(2016, time.October, 4)
	common, err := stock.AdvanceDecline(oldDay, TWMarketTWSE, true)
	if err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]any{"Advancing": int64(2), "Declining": int64(2), "Unchanged": int64(1), "NoTrade": int64(1), "NoReference": int64(1), "AdvancingLimit": nil, "Source": "computed"} {
		if got := value(common, name); got != want {
			t.Errorf("computed common %s = %v, want %v", name, got, want)
		}
	}
	all, err := stock.AdvanceDecline(oldDay, TWMarketTWSE, false)
	if err != nil {
		t.Fatal(err)
	}
	if value(all, "Advancing") != int64(3) || value(all, "Declining") != int64(3) {
		t.Errorf("computed all: %v up, %v down", value(all, "Advancing"), value(all, "Declining"))
	}
}

func TestIsCommonStockCode(t *testing.T) {
	for code, want := range map[string]bool{"2330": true, "1101": true, "9958": true, "9103": false, "00400A": false, "0050": false, "23": false, "A123": false, "": false} {
		if got := isCommonStockCode(code); got != want {
			t.Errorf("isCommonStockCode(%q) = %v, want %v", code, got, want)
		}
	}
}

func mopsFixture(t *testing.T, transport *fixtureTransport, key, name string) {
	t.Helper()
	body, err := os.ReadFile(filepath.Join("testdata", "mops", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	transport.replies[key] = append(transport.replies[key], fixtureReply{status: http.StatusOK, body: body})
}

func TestTWStockMonthlyRevenue(t *testing.T) {
	stock, transport := newFixtureTWStock(t, TWStockConfig{})
	mopsFixture(t, transport, "/nas/t21/sii/t21sc03_114_9_0.html", "sii_114_9_0.html")
	mopsFixture(t, transport, "/nas/t21/sii/t21sc03_114_9_1.html", "sii_114_9_1.html")
	dt, err := stock.MonthlyRevenue(2025, 9, TWMarketTWSE)
	if err != nil {
		t.Fatalf("MonthlyRevenue error: %v", err)
	}
	// Fixture: six domestic companies in two industries and one foreign company.
	if dt.NumRows() != 7 {
		t.Fatalf("rows = %d, want 7", dt.NumRows())
	}
	want := map[string]any{
		"Month": dateUTC(2025, time.September, 1), "Code": "1101", "Name": "台泥", "Industry": "水泥工業", "Market": "TWSE",
		"Revenue": int64(13306676), "PrevMonthRevenue": int64(12214776), "LastYearRevenue": int64(13325249),
		"MoMChange": 8.93, "YoYChange": -0.13, "Cumulative": int64(109438297), "LastYearCumulative": int64(105591346),
		"CumulativeChange": 3.64, "Remark": "", "PageDate": dateUTC(2026, time.October, 16),
	}
	for name, w := range want {
		if got := dt.GetColByName(name).Data()[0]; got != w {
			t.Errorf("%s = %v (%T), want %v", name, got, got, w)
		}
	}
	if got := dt.GetColByName("Industry").Data()[3]; got != "食品工業" {
		t.Errorf("fourth company's industry = %v, want 食品工業", got)
	}
}

func TestTWStockMonthlyRevenueBothMarketsAndOldPages(t *testing.T) {
	stock, transport := newFixtureTWStock(t, TWStockConfig{})
	mopsFixture(t, transport, "/nas/t21/sii/t21sc03_114_9_0.html", "sii_114_9_0.html")
	transport.replies["/nas/t21/sii/t21sc03_114_9_1.html"] = []fixtureReply{{status: http.StatusNotFound, body: []byte("Not Found")}}
	mopsFixture(t, transport, "/nas/t21/otc/t21sc03_114_9_0.html", "otc_114_9_0.html")
	transport.replies["/nas/t21/otc/t21sc03_114_9_1.html"] = []fixtureReply{{status: http.StatusNotFound, body: []byte("Not Found")}}
	both, err := stock.MonthlyRevenue(2025, 9, TWMarketAuto)
	if err != nil {
		t.Fatalf("both markets: %v", err)
	}
	markets := map[any]int{}
	for _, m := range both.GetColByName("Market").Data() {
		markets[m]++
	}
	if markets["TWSE"] != 6 || markets["TPEx"] != 4 {
		t.Errorf("markets = %v, want 6 TWSE and 4 TPEx", markets)
	}

	// January 2003 has one page with no suffix, ten cells to a row and no remark.
	transport.replies["/nas/t21/sii/t21sc03_92_1_0.html"] = []fixtureReply{{status: http.StatusNotFound, body: []byte("Not Found")}}
	mopsFixture(t, transport, "/nas/t21/sii/t21sc03_92_1.html", "sii_92_1.html")
	old, err := stock.MonthlyRevenue(2003, 1, TWMarketTWSE)
	if err != nil {
		t.Fatalf("2003: %v", err)
	}
	if old.NumRows() != 4 || old.GetColByName("Revenue").Data()[0] != int64(2273373) || old.GetColByName("Remark").Data()[0] != "" {
		t.Errorf("2003: %d rows, first revenue %v", old.NumRows(), old.GetColByName("Revenue").Data()[0])
	}
	if got := old.GetColByName("Month").Data()[0]; got != dateUTC(2003, time.January, 1) {
		t.Errorf("2003 Month = %v", got)
	}

	// A month with no page at all is an empty table.
	transport.replies["/nas/t21/sii/t21sc03_115_12_0.html"] = []fixtureReply{{status: http.StatusNotFound, body: []byte("Not Found")}}
	transport.replies["/nas/t21/sii/t21sc03_115_12.html"] = []fixtureReply{{status: http.StatusNotFound, body: []byte("Not Found")}}
	none, err := stock.MonthlyRevenue(2026, 12, TWMarketTWSE)
	if err != nil || none.NumRows() != 0 || none.GetColByName("Revenue") == nil {
		t.Errorf("missing month: %v, %v", none, err)
	}
}

func TestTWStockMonthlyRevenueRefusesBadCalls(t *testing.T) {
	stock, _ := newFixtureTWStock(t, TWStockConfig{})
	for name, f := range map[string]func() error{
		"early year": func() error { _, e := stock.MonthlyRevenue(2002, 12, TWMarketTWSE); return e },
		"month 13":   func() error { _, e := stock.MonthlyRevenue(2025, 13, TWMarketTWSE); return e },
		"month 0":    func() error { _, e := stock.MonthlyRevenue(2025, 0, TWMarketTWSE); return e },
		"market":     func() error { _, e := stock.MonthlyRevenue(2025, 9, TWMarket("NYSE")); return e },
	} {
		if f() == nil {
			t.Errorf("%s: want an error", name)
		}
	}
	var zero TWStockClient
	if _, err := zero.MonthlyRevenue(2025, 9, TWMarketTWSE); err == nil {
		t.Error("a zero client should be an error")
	}
}
