package datafetch

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
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

func newFixtureTWStock(t *testing.T, cfg TWStockConfig) (*twStock, *fixtureTransport) {
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
	return stock, transport
}

func query(values ...string) url.Values {
	result := url.Values{}
	for i := 0; i < len(values); i += 2 {
		result.Set(values[i], values[i+1])
	}
	return result
}

func TestTWStockThrottle(t *testing.T) {
	stock, transport := newFixtureTWStock(t, TWStockConfig{Interval: 20 * time.Millisecond})
	key := fixtureKey("/v1/exchangeReport/STOCK_DAY_ALL", nil)
	transport.addFixture(t, key, "twse_stock_day_all.json")
	transport.addFixture(t, key, "twse_stock_day_all.json")
	if _, err := stock.AllDailyQuotes(TWMarketTWSE); err != nil {
		t.Fatalf("first AllDailyQuotes error: %v", err)
	}
	if _, err := stock.AllDailyQuotes(TWMarketTWSE); err != nil {
		t.Fatalf("second AllDailyQuotes error: %v", err)
	}
	times := transport.requestTimes()
	if len(times) != 2 || times[1].Sub(times[0]) < 20*time.Millisecond {
		t.Fatalf("request times = %v, want at least 20ms apart", times)
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
	if got := transport.lastUserAgent(); got != "insyra-datafetch/0.3.1" {
		t.Errorf("User-Agent = %q", got)
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
