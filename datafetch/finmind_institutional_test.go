package datafetch

import (
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/HazelnutParadise/insyra"
)

func finMindFixture(t *testing.T, name string) []byte {
	t.Helper()
	body, err := os.ReadFile(filepath.Join("testdata", "finmind", name))
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func finMindNamesKey() string { return finMindKey("dataset", "TaiwanFutOptDailyInfo") }

// taifexReference builds the table TAIFEXClient builds from the exchange's own
// CSV for the same days, which FinMind's must equal.
func taifexReference(t *testing.T, name string, options bool) *insyra.DataTable {
	t.Helper()
	body, err := os.ReadFile(filepath.Join("testdata", "taifex", name))
	if err != nil {
		t.Fatal(err)
	}
	records, err := parseTAIFEXRecords(body)
	if err != nil {
		t.Fatal(err)
	}
	dt, err := taifexTable(name, records, institutionalColumns(options))
	if err != nil {
		t.Fatal(err)
	}
	return dt
}

func assertSameTable(t *testing.T, got, want *insyra.DataTable) {
	t.Helper()
	if !reflect.DeepEqual(got.ColNames(), want.ColNames()) {
		t.Fatalf("columns = %v, want %v", got.ColNames(), want.ColNames())
	}
	if got.NumRows() != want.NumRows() {
		t.Fatalf("rows = %d, want %d", got.NumRows(), want.NumRows())
	}
	for _, name := range want.ColNames() {
		g, w := got.GetColByName(name).Data(), want.GetColByName(name).Data()
		for r := range w {
			// The exchange subtracts before rounding to thousands, FinMind sends
			// the rounded sides, so a net value may differ by one thousand.
			if strings.HasSuffix(name, "NetValue") {
				if gi, wi := g[r].(int64), w[r].(int64); gi-wi > 1 || wi-gi > 1 {
					t.Errorf("%s row %d = %v, want %v within 1", name, r, g[r], w[r])
				}
				continue
			}
			if !reflect.DeepEqual(g[r], w[r]) {
				t.Errorf("%s row %d = %v (%T), want %v (%T)", name, r, g[r], g[r], w[r], w[r])
			}
		}
	}
}

func TestFinMindInstitutionalFuturesMatchesTAIFEX(t *testing.T) {
	client, transport, _ := newFixtureFinMind(t, FinMindConfig{})
	transport.replies[finMindNamesKey()] = []fixtureReply{{status: http.StatusOK, body: finMindFixture(t, "fut_opt_info.json")}}
	transport.replies[finMindKey("dataset", "TaiwanFuturesInstitutionalInvestors", "data_id", "TX", "start_date", "2025-10-01", "end_date", "2025-10-02")] =
		[]fixtureReply{{status: http.StatusOK, body: finMindFixture(t, "inst_futures_tx.json")}}

	got, err := client.InstitutionalFutures("TX", dateUTC(2025, time.October, 1), dateUTC(2025, time.October, 2))
	if err != nil {
		t.Fatalf("InstitutionalFutures error: %v", err)
	}
	assertSameTable(t, got, taifexReference(t, "inst_futures_tx_utf8.csv", false))
}

func TestFinMindInstitutionalOptionsMatchesTAIFEX(t *testing.T) {
	client, transport, _ := newFixtureFinMind(t, FinMindConfig{})
	transport.replies[finMindNamesKey()] = []fixtureReply{{status: http.StatusOK, body: finMindFixture(t, "fut_opt_info.json")}}
	transport.replies[finMindKey("dataset", "TaiwanOptionInstitutionalInvestors", "data_id", "TXO", "start_date", "2025-10-01", "end_date", "2025-10-01")] =
		[]fixtureReply{{status: http.StatusOK, body: finMindFixture(t, "inst_options_txo.json")}}

	got, err := client.InstitutionalOptions("TXO", dateUTC(2025, time.October, 1), dateUTC(2025, time.October, 1))
	if err != nil {
		t.Fatalf("InstitutionalOptions error: %v", err)
	}
	assertSameTable(t, got, taifexReference(t, "inst_options_txo_utf8.csv", true))
}

func TestFinMindInstitutionalNamesAreReadOncePerClient(t *testing.T) {
	client, transport, _ := newFixtureFinMind(t, FinMindConfig{})
	transport.replies[finMindNamesKey()] = []fixtureReply{{status: http.StatusOK, body: finMindFixture(t, "fut_opt_info.json")}}
	data := finMindKey("dataset", "TaiwanFuturesInstitutionalInvestors", "data_id", "TX", "start_date", "2025-10-01", "end_date", "2025-10-02")
	body := finMindFixture(t, "inst_futures_tx.json")
	transport.replies[data] = []fixtureReply{{status: http.StatusOK, body: body}, {status: http.StatusOK, body: body}}
	for i := 0; i < 2; i++ {
		if _, err := client.InstitutionalFutures("TX", dateUTC(2025, time.October, 1), dateUTC(2025, time.October, 2)); err != nil {
			t.Fatalf("call %d: %v", i, err)
		}
	}
	names := 0
	for _, request := range transport.requests {
		if request.key == finMindNamesKey() {
			names++
		}
	}
	if names != 1 {
		t.Fatalf("contract names requested %d times, want 1", names)
	}
}

func TestFinMindInstitutionalTakesTheNameOfTheRightKind(t *testing.T) {
	// CJ is both a futures and an options code; the futures call must use the
	// futures name.
	client, transport, _ := newFixtureFinMind(t, FinMindConfig{})
	transport.replies[finMindNamesKey()] = []fixtureReply{{status: http.StatusOK, body: finMindFixture(t, "fut_opt_info.json")}}
	transport.replies[finMindKey("dataset", "TaiwanFuturesInstitutionalInvestors", "data_id", "CJ")] = []fixtureReply{{status: http.StatusOK, body: []byte(
		`{"msg":"success","status":200,"data":[{"futures_id":"CJ","date":"2025-10-01","institutional_investors":"外資","long_deal_volume":1,"long_deal_amount":2,"short_deal_volume":3,"short_deal_amount":4,"long_open_interest_balance_volume":5,"long_open_interest_balance_amount":6,"short_open_interest_balance_volume":7,"short_open_interest_balance_amount":8}]}`)}}
	got, err := client.InstitutionalFutures("CJ", time.Time{}, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if name := got.GetColByName("Contract").Data()[0]; name != "華南金(股票期貨)" {
		t.Errorf("Contract = %v", name)
	}
	if v := got.GetColByName("TradeNet").Data()[0]; v != int64(-2) {
		t.Errorf("TradeNet = %v", v)
	}
	if v := got.GetColByName("OINetValue").Data()[0]; v != int64(-2) {
		t.Errorf("OINetValue = %v", v)
	}
	if v := got.GetColByName("Trader").Data()[0]; v != "外資及陸資" {
		t.Errorf("Trader = %v", v)
	}
}

func TestFinMindInstitutionalRefusals(t *testing.T) {
	client, transport, _ := newFixtureFinMind(t, FinMindConfig{})
	transport.replies[finMindNamesKey()] = []fixtureReply{{status: http.StatusOK, body: finMindFixture(t, "fut_opt_info.json")}}

	if _, err := client.InstitutionalFutures(" ", time.Time{}, time.Time{}); err == nil || !strings.Contains(err.Error(), "contract code") {
		t.Errorf("empty contract: err = %v", err)
	}
	if _, err := client.InstitutionalFutures("TX", dateUTC(2025, time.October, 2), dateUTC(2025, time.October, 1)); err == nil {
		t.Error("to before from: no error")
	}
	if _, err := client.InstitutionalFutures("TXO", time.Time{}, time.Time{}); err == nil || !strings.Contains(err.Error(), `"TXO"`) {
		t.Errorf("options code given to futures: err = %v", err)
	}
	if len(transport.requests) != 1 {
		t.Errorf("requests = %d, want only the names request", len(transport.requests))
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := client.InstitutionalOptionsContext(ctx, "TXO", time.Time{}, time.Time{}); !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled context: err = %v", err)
	}
	var zero FinMindClient
	if _, err := zero.InstitutionalFutures("TX", time.Time{}, time.Time{}); !errors.Is(err, errFinMindNotCreated) {
		t.Errorf("zero client: err = %v", err)
	}
}

func TestFinMindInstitutionalNamesFailureFailsTheCall(t *testing.T) {
	client, transport, _ := newFixtureFinMind(t, FinMindConfig{})
	transport.replies[finMindNamesKey()] = []fixtureReply{{status: http.StatusPaymentRequired, body: []byte(`{}`)}}
	if _, err := client.InstitutionalFutures("TX", time.Time{}, time.Time{}); !errors.Is(err, ErrFinMindLimit) {
		t.Fatalf("err = %v, want ErrFinMindLimit", err)
	}
	if client.names != nil {
		t.Error("a failed names request was cached")
	}
}
