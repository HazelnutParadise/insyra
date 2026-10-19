package datafetch

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func newFixtureTAIFEX(t *testing.T) (*TAIFEXClient, *fixtureTransport) {
	t.Helper()
	client, err := TAIFEX(TWStockConfig{})
	if err != nil {
		t.Fatalf("TAIFEX returned error: %v", err)
	}
	transport := newFixtureTransport()
	client.base.client.Transport = transport
	client.baseURL = "https://taifex.test"
	return client, transport
}

func addTAIFEXFixture(t *testing.T, transport *fixtureTransport, key, name string) {
	t.Helper()
	body, err := os.ReadFile(filepath.Join("testdata", "taifex", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	transport.replies[key] = append(transport.replies[key], fixtureReply{status: http.StatusOK, body: body})
}

func TestTAIFEXInstitutional(t *testing.T) {
	client, transport := newFixtureTAIFEX(t)
	addTAIFEXFixture(t, transport, "/v1/MarketDataOfMajorInstitutionalTradersDetailsOfFuturesContractsBytheDate", "institutional_futures.json")
	addTAIFEXFixture(t, transport, "/v1/MarketDataOfMajorInstitutionalTradersDetailsOfCallsAndPutsBytheDate", "institutional_options.json")

	futures, err := client.InstitutionalFutures()
	if err != nil {
		t.Fatalf("InstitutionalFutures error: %v", err)
	}
	if futures.NumRows() != 6 || futures.GetColByName("CallPut") != nil {
		t.Fatalf("futures: %d rows", futures.NumRows())
	}
	// Row 0: 臺股期貨, 自營商 on 2026-10-16.
	want := map[string]any{
		"Date": time.Date(2026, 10, 16, 0, 0, 0, 0, time.UTC), "Contract": "臺股期貨", "Trader": "自營商",
		"TradeLong": int64(7101), "TradeShort": int64(10778), "TradeNet": int64(-3677), "TradeNetValue": int64(-34795850),
		"OILong": int64(7760), "OIShort": int64(4563), "OINet": int64(3197), "OINetValue": int64(30302618),
	}
	for name, w := range want {
		if got := futures.GetColByName(name).Data()[0]; got != w {
			t.Errorf("futures %s = %v (%T), want %v", name, got, got, w)
		}
	}

	options, err := client.InstitutionalOptions()
	if err != nil {
		t.Fatalf("InstitutionalOptions error: %v", err)
	}
	if got := options.GetColByName("CallPut").Data()[0]; got != "CALL" {
		t.Errorf("options CallPut = %v", got)
	}
	if got := options.GetColByName("TradeLong").Data()[0]; got != int64(110650) {
		t.Errorf("options TradeLong = %v", got)
	}
}

func TestTAIFEXLargeTraders(t *testing.T) {
	client, transport := newFixtureTAIFEX(t)
	addTAIFEXFixture(t, transport, "/v1/OpenInterestOfLargeTradersFutures", "large_traders_futures.csv.txt")
	addTAIFEXFixture(t, transport, "/v1/OpenInterestOfLargeTradersOptions", "large_traders_options.json")

	// The futures table comes back as CSV with Chinese headers and a byte order mark.
	futures, err := client.LargeTradersFutures()
	if err != nil {
		t.Fatalf("LargeTradersFutures error: %v", err)
	}
	for name, w := range map[string]any{"Contract": "TX", "SettlementMonth": "202610", "TraderType": "0", "Top5Buy": int64(68157), "Top5Sell": int64(53675), "Top10Buy": int64(75382), "Top10Sell": int64(72888), "MarketOI": int64(106162)} {
		if got := futures.GetColByName(name).Data()[0]; got != w {
			t.Errorf("futures %s = %v (%T), want %v", name, got, got, w)
		}
	}
	if got := futures.GetColByName("Date").Data()[0].(time.Time); !got.Equal(time.Date(2026, 10, 16, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("Date = %v", got)
	}

	options, err := client.LargeTradersOptions()
	if err != nil {
		t.Fatalf("LargeTradersOptions error: %v", err)
	}
	if got := options.GetColByName("Contract").Data()[0]; got != "TXO" {
		t.Errorf("options Contract = %v", got)
	}
	if v := options.GetColByName("CallPut").Data()[0]; v != "買權" && v != "賣權" {
		t.Errorf("options CallPut = %v", v)
	}
	if _, ok := options.GetColByName("Top5Buy").Data()[0].(int64); !ok {
		t.Errorf("Top5Buy type = %T", options.GetColByName("Top5Buy").Data()[0])
	}
}

func TestTAIFEXPutCallRatio(t *testing.T) {
	client, transport := newFixtureTAIFEX(t)
	addTAIFEXFixture(t, transport, "/v1/PutCallRatio", "put_call_ratio.json")
	dt, err := client.PutCallRatio()
	if err != nil {
		t.Fatalf("PutCallRatio error: %v", err)
	}
	for name, w := range map[string]any{"PutVolume": int64(396499), "CallVolume": int64(446741), "VolumeRatio": 88.75, "PutOI": int64(55093), "CallOI": int64(81497), "OIRatio": 67.6} {
		if got := dt.GetColByName(name).Data()[0]; got != w {
			t.Errorf("%s = %v (%T), want %v", name, got, got, w)
		}
	}
	if dt.NumRows() != 3 {
		t.Errorf("rows = %d", dt.NumRows())
	}
}

func TestTAIFEXRefusesWhatItCannotRead(t *testing.T) {
	client, transport := newFixtureTAIFEX(t)
	transport.replies["/v1/PutCallRatio"] = []fixtureReply{{status: http.StatusOK, body: []byte(`[{"Date":"2026-10-16","PutVolume":"1"}]`)}}
	if _, err := client.PutCallRatio(); err == nil {
		t.Error("a record without the expected fields should be an error")
	}
	transport.replies["/v1/PutCallRatio"] = []fixtureReply{{status: http.StatusOK, body: []byte(`[{"Date":"20261016","PutVolume":"1","CallVolume":"1","PutCallVolumeRatio%":"1","PutOI":"1","CallOI":"1","PutCallOIRatio%":"1"}`)}}
	if _, err := client.PutCallRatio(); err == nil {
		t.Error("truncated JSON should be an error")
	}
	transport.addStatus("/v1/PutCallRatio", http.StatusInternalServerError)
	if _, err := client.PutCallRatio(); err == nil {
		t.Error("a server error should be an error")
	}
	transport.replies["/v1/PutCallRatio"] = []fixtureReply{{status: http.StatusOK, body: []byte(`[{"Date":"20261016","PutVolume":"1","CallVolume":"1","PutCallVolumeRatio%":"1","PutOI":"1","CallOI":"1","PutCallOIRatio%":"1"}]`)}}
	one, err := client.PutCallRatio()
	if err != nil || one.NumRows() != 1 {
		t.Errorf("a valid single record: %v, %v", one, err)
	}
	transport.replies["/v1/PutCallRatio"] = []fixtureReply{{status: http.StatusOK, body: nil}}
	empty, err := client.PutCallRatio()
	if err != nil || empty.NumRows() != 0 || empty.GetColByName("PutVolume") == nil {
		t.Errorf("an empty reply should be an empty table: %v, %v", empty, err)
	}
	var zero TAIFEXClient
	if _, err := zero.PutCallRatio(); err == nil {
		t.Error("a zero client should be an error")
	}
	var nilClient *TAIFEXClient
	if _, err := nilClient.InstitutionalFutures(); err == nil {
		t.Error("a nil client should be an error")
	}
}
