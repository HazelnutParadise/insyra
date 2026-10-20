package datafetch

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"golang.org/x/text/encoding/traditionalchinese"
)

func historyKey(path string, values ...string) string {
	v := url.Values{}
	for i := 0; i < len(values); i += 2 {
		v.Set(values[i], values[i+1])
	}
	return path + "?" + v.Encode()
}

func newHistoryClient(t *testing.T) (*TAIFEXClient, *fixtureTransport) {
	t.Helper()
	client, transport := newFixtureTAIFEX(t)
	client.websiteURL = "https://taifex-web.test"
	return client, transport
}

func TestTAIFEXPutCallRatioHistory(t *testing.T) {
	client, transport := newHistoryClient(t)
	addTAIFEXFixture(t, transport, historyKey("/cht/3/pcRatioDown", "queryStartDate", "2026/10/12", "queryEndDate", "2026/10/16"), "history_pcratio.csv.txt")
	dt, err := client.PutCallRatioHistory(dateUTC(2026, time.October, 12), dateUTC(2026, time.October, 16))
	if err != nil {
		t.Fatal(err)
	}
	if dt.NumRows() != 5 {
		t.Fatalf("rows = %d, want 5", dt.NumRows())
	}
	dates := dt.GetColByName("Date").Data()
	// The website lists newest first; the history is oldest first.
	if dates[0] != dateUTC(2026, time.October, 12) || dates[4] != dateUTC(2026, time.October, 16) {
		t.Fatalf("dates %v", dates)
	}
	last := func(name string) any { return dt.GetColByName(name).Data()[4] }
	if last("PutVolume") != int64(396499) || last("CallVolume") != int64(446741) || last("VolumeRatio") != 88.75 || last("OIRatio") != 67.6 {
		t.Errorf("row of 2026-10-16: %v %v %v %v", last("PutVolume"), last("CallVolume"), last("VolumeRatio"), last("OIRatio"))
	}
}

func TestTAIFEXInstitutionalHistory(t *testing.T) {
	client, transport := newHistoryClient(t)
	q := func(path string) string {
		return historyKey(path, "queryStartDate", "2026/10/15", "queryEndDate", "2026/10/16", "commodityId", "")
	}
	addTAIFEXFixture(t, transport, q("/cht/3/futContractsDateDown"), "history_inst_futures.csv.txt")
	addTAIFEXFixture(t, transport, q("/cht/3/callsAndPutsDateDown"), "history_inst_options.csv.txt")
	from, to := dateUTC(2026, time.October, 15), dateUTC(2026, time.October, 16)

	fut, err := client.InstitutionalFuturesHistory(from, to)
	if err != nil {
		t.Fatal(err)
	}
	if fut.NumRows() != 8 {
		t.Fatalf("futures rows = %d, want 8", fut.NumRows())
	}
	first := func(name string) any { return fut.GetColByName(name).Data()[0] }
	if first("Date") != from || first("Contract") != "臺股期貨" || first("Trader") != "自營商" || first("TradeLong") != int64(16318) || first("OILong") != int64(10619) {
		t.Errorf("first row: %v %v %v %v %v", first("Date"), first("Contract"), first("Trader"), first("TradeLong"), first("OILong"))
	}
	if fut.GetColByName("CallPut") != nil {
		t.Error("futures have no CallPut column")
	}

	opt, err := client.InstitutionalOptionsHistory(from, to)
	if err != nil {
		t.Fatal(err)
	}
	if opt.NumRows() != 8 || opt.GetColByName("CallPut").Data()[0] != "CALL" {
		t.Fatalf("options: %d rows, CallPut %v", opt.NumRows(), opt.GetColByName("CallPut").Data()[0])
	}
}

func TestTAIFEXLargeTradersHistory(t *testing.T) {
	client, transport := newHistoryClient(t)
	q := func(path string) string {
		return historyKey(path, "queryStartDate", "2026/10/16", "queryEndDate", "2026/10/16", "commodityId", "")
	}
	addTAIFEXFixture(t, transport, q("/cht/3/largeTraderFutDown"), "history_large_futures.csv.txt")
	addTAIFEXFixture(t, transport, q("/cht/3/largeTraderOptDown"), "history_large_options.csv.txt")
	day := dateUTC(2026, time.October, 16)

	fut, err := client.LargeTradersFuturesHistory(day, day)
	if err != nil {
		t.Fatal(err)
	}
	if fut.NumRows() != 10 {
		t.Fatalf("rows = %d", fut.NumRows())
	}
	row := func(name string) any { return fut.GetColByName(name).Data()[0] }
	// The padding the exchange puts around a contract code and a month is trimmed.
	if row("Contract") != "BRF" || row("ContractName") != "布蘭特原油期貨" || row("SettlementMonth") != "202612" || row("TraderType") != "0" {
		t.Errorf("first row: %q %q %q %q", row("Contract"), row("ContractName"), row("SettlementMonth"), row("TraderType"))
	}
	opt, err := client.LargeTradersOptionsHistory(day, day)
	if err != nil {
		t.Fatal(err)
	}
	if opt.GetColByName("CallPut").Data()[0] != "買權" || opt.GetColByName("Contract").Data()[0] != "CA" {
		t.Errorf("options first row: %v %v", opt.GetColByName("CallPut").Data()[0], opt.GetColByName("Contract").Data()[0])
	}
}

// A range is requested month by month, because the website limits how long a
// range may be, and the months are joined in date order.
func TestTAIFEXHistoryIsRequestedMonthByMonth(t *testing.T) {
	client, transport := newHistoryClient(t)
	header := "日期,賣權成交量,買權成交量,買賣權成交量比率%,賣權未平倉量,買權未平倉量,買賣權未平倉量比率%\n"
	body := func(lines ...string) []fixtureReply {
		return []fixtureReply{{status: http.StatusOK, body: toBig5(t, header+strings.Join(lines, "\n")+"\n")}}
	}
	transport.replies[historyKey("/cht/3/pcRatioDown", "queryStartDate", "2026/09/29", "queryEndDate", "2026/09/30")] = body("2026/09/30,10,20,50.00,30,40,75.00,")
	transport.replies[historyKey("/cht/3/pcRatioDown", "queryStartDate", "2026/10/01", "queryEndDate", "2026/10/02")] = body("2026/10/02,11,22,50.00,33,44,75.00,", "2026/10/01,12,24,50.00,36,48,75.00,")
	dt, err := client.PutCallRatioHistory(dateUTC(2026, time.September, 29), dateUTC(2026, time.October, 2))
	if err != nil {
		t.Fatal(err)
	}
	if transport.requestCount() != 2 {
		t.Fatalf("requests = %d, want one per month", transport.requestCount())
	}
	dates := dt.GetColByName("Date").Data()
	if len(dates) != 3 || dates[0] != dateUTC(2026, time.September, 30) || dates[1] != dateUTC(2026, time.October, 1) || dates[2] != dateUTC(2026, time.October, 2) {
		t.Fatalf("dates %v", dates)
	}
	// A range with no trading day is an empty table, not an error.
	transport.replies[historyKey("/cht/3/pcRatioDown", "queryStartDate", "2026/10/03", "queryEndDate", "2026/10/04")] = body()
	empty, err := client.PutCallRatioHistory(dateUTC(2026, time.October, 3), dateUTC(2026, time.October, 4))
	if err != nil || empty.NumRows() != 0 || empty.GetColByName("Date") == nil {
		t.Errorf("a weekend: %v rows, %v", empty.NumRows(), err)
	}
}

func TestTAIFEXHistoryRefusesWhatTheWebsiteRefuses(t *testing.T) {
	client, transport := newHistoryClient(t)
	addTAIFEXFixture(t, transport, historyKey("/cht/3/futContractsDateDown", "queryStartDate", "2023/06/03", "queryEndDate", "2023/06/07", "commodityId", ""), "history_refused.html")
	_, err := client.InstitutionalFuturesHistory(dateUTC(2023, time.June, 3), dateUTC(2023, time.June, 7))
	if !errors.Is(err, ErrTAIFEXRefused) {
		t.Fatalf("a refusal should match ErrTAIFEXRefused: %v", err)
	}
	if !strings.Contains(err.Error(), "DateTime error") || !strings.Contains(err.Error(), "2023-06-03") {
		t.Fatalf("the website's alert should come back with the range: %v", err)
	}
	if _, err := client.PutCallRatioHistory(time.Time{}, dateUTC(2026, time.October, 12)); err == nil || errors.Is(err, ErrTAIFEXRefused) {
		t.Errorf("a zero from should be refused before any request: %v", err)
	}
	if _, err := client.PutCallRatioHistory(dateUTC(2026, time.October, 16), dateUTC(2026, time.October, 12)); err == nil {
		t.Error("from after to should be an error")
	}
	var zero TAIFEXClient
	if _, err := zero.PutCallRatioHistory(dateUTC(2026, time.October, 12), dateUTC(2026, time.October, 16)); err == nil {
		t.Error("a zero client should be an error")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := client.PutCallRatioHistoryContext(ctx, dateUTC(2026, time.October, 12), dateUTC(2026, time.October, 16)); err == nil {
		t.Error("a cancelled context should be an error")
	}
}

func toBig5(t *testing.T, text string) []byte {
	t.Helper()
	out, err := traditionalchinese.Big5.NewEncoder().Bytes([]byte(text))
	if err != nil {
		t.Fatal(err)
	}
	return out
}
