package datafetch

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"
)

const (
	twseListedBody   = `[{"出表日期":"1151105","公司代號":"1101","公司簡稱":"台泥","上市日期":"19620209"},{"出表日期":"1151105","公司代號":"6214","公司簡稱":"精誠","上市日期":"20101230"}]`
	twseSuspendBody  = `{"status":"ok","data":[["115/09/01","三商壽","2867"],["090/01/20","彥武企業","2011"]],"title":"終止上市公司","fields":["終止上市日期","公司名稱","上市編號"],"total":2}`
	tpexListedBody   = `[{"Date":"1151106","SecuritiesCompanyCode":"1240","CompanyName":"茂生農經股份有限公司","CompanyAbbreviation":"茂生農經","DateOfListing":"20180808"}]`
	tpexDelistedNone = `{"tables":[{"fields":["股票代號","公司名稱","終止上櫃日期","終止上櫃原因","公司資料網址"],"data":[],"totalCount":0}],"stat":"ok"}`
	// The 1995 rows are the TPEx's; the reason on 5303 and its move are made up
	// so that one row carries both.
	tpexDelisted1995 = `{"tables":[{"fields":["股票代號","公司名稱","終止上櫃日期","終止上櫃原因","公司資料網址"],"data":[["4501","利奇機械","84-11-14","","u"],["5303","精業","84-09-23","依本中心業務規則第12條之2第1項第1款","u"]],"totalCount":2}],"date":"1995","stat":"ok"}`
	tpexMoved1995    = `{"tables":[{"fields":["股票代號","公司名稱","終止上櫃日期","終止上櫃原因","公司資料網址"],"data":[["5303","精業","84-09-23","依本中心業務規則第12條之2第1項第1款","u"]],"totalCount":1}],"date":"1995","stat":"ok"}`
)

func tpexDelistedKey(year, reason string) string {
	return fixtureKey("/www/zh-tw/company/deListed", query("date", year, "reason", reason, "response", "json"))
}

func listingsFixture(t *testing.T) (*TWStockClient, *fixtureTransport) {
	t.Helper()
	old := listingsNow
	listingsNow = func() time.Time { return time.Date(1995, time.June, 1, 0, 0, 0, 0, time.UTC) }
	t.Cleanup(func() { listingsNow = old })
	client, transport := newFixtureTransportTWStock(t)
	ok := func(body string) []fixtureReply { return []fixtureReply{{status: http.StatusOK, body: []byte(body)}} }
	transport.replies[fixtureKey("/v1/opendata/t187ap03_L", nil)] = ok(twseListedBody)
	transport.replies[fixtureKey("/rwd/zh/company/suspendListing", query("response", "json"))] = ok(twseSuspendBody)
	transport.replies[fixtureKey("/openapi/v1/mopsfin_t187ap03_O", nil)] = ok(tpexListedBody)
	transport.replies[tpexDelistedKey("1994", "-1")] = ok(tpexDelistedNone)
	transport.replies[tpexDelistedKey("1994", "2")] = ok(tpexDelistedNone)
	transport.replies[tpexDelistedKey("1995", "-1")] = ok(tpexDelisted1995)
	transport.replies[tpexDelistedKey("1995", "2")] = ok(tpexMoved1995)
	return client, transport
}

func TestListingsBothExchanges(t *testing.T) {
	client, _ := listingsFixture(t)
	dt, err := client.Listings(TWMarketAuto)
	if err != nil {
		t.Fatalf("Listings error: %v", err)
	}
	if got := dt.ColNames(); !reflect.DeepEqual(got, listingColumns) {
		t.Fatalf("columns = %v", got)
	}
	want := [][]any{
		{"1101", "台泥", "TWSE", dateUTC(1962, time.February, 9), nil, nil, nil},
		{"2011", "彥武企業", "TWSE", nil, dateUTC(2001, time.January, 20), nil, nil},
		{"2867", "三商壽", "TWSE", nil, dateUTC(2026, time.September, 1), nil, nil},
		{"6214", "精誠", "TWSE", dateUTC(2010, time.December, 30), nil, nil, nil},
		{"1240", "茂生農經", "TPEx", dateUTC(2018, time.August, 8), nil, nil, nil},
		{"4501", "利奇機械", "TPEx", nil, dateUTC(1995, time.November, 14), nil, nil},
		{"5303", "精業", "TPEx", nil, dateUTC(1995, time.September, 23), "依本中心業務規則第12條之2第1項第1款", "TWSE"},
	}
	if dt.NumRows() != len(want) {
		t.Fatalf("rows = %d, want %d", dt.NumRows(), len(want))
	}
	for r, row := range want {
		for c, name := range listingColumns {
			if got := dt.GetColByName(name).Data()[r]; !reflect.DeepEqual(got, row[c]) {
				t.Errorf("row %d %s = %v (%T), want %v", r, name, got, got, row[c])
			}
		}
	}
}

func TestListingsOneExchangeAsksOnlyIt(t *testing.T) {
	client, transport := listingsFixture(t)
	dt, err := client.Listings(TWMarketTWSE)
	if err != nil {
		t.Fatal(err)
	}
	if dt.NumRows() != 4 {
		t.Errorf("rows = %d, want 4", dt.NumRows())
	}
	for _, request := range transport.requests {
		if strings.Contains(request.key, "deListed") || strings.Contains(request.key, "mopsfin") {
			t.Errorf("TWSE call asked the TPEx: %s", request.key)
		}
	}
	tpex, err := client.Listings(TWMarketTPEx)
	if err != nil {
		t.Fatal(err)
	}
	if tpex.NumRows() != 3 {
		t.Errorf("TPEx rows = %d, want 3", tpex.NumRows())
	}
}

func TestListingsFailures(t *testing.T) {
	client, transport := listingsFixture(t)
	if _, err := client.Listings("NYSE"); err == nil {
		t.Error("unknown market: no error")
	}
	transport.replies[tpexDelistedKey("1995", "2")] = []fixtureReply{{status: http.StatusOK, body: []byte(`{"tables":[{"fields":["代號"],"data":[["1"]]}],"stat":"ok"}`)}}
	if _, err := client.Listings(TWMarketTPEx); err == nil || !strings.Contains(err.Error(), "1995") {
		t.Errorf("changed headers: err = %v", err)
	}
	transport.replies[fixtureKey("/v1/opendata/t187ap03_L", nil)] = []fixtureReply{{status: http.StatusOK, body: []byte(`[]`)}}
	if _, err := client.Listings(TWMarketTWSE); err == nil || !strings.Contains(err.Error(), "empty") {
		t.Errorf("empty company list: err = %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := client.ListingsContext(ctx, TWMarketAuto); !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled: err = %v", err)
	}
	var zero TWStockClient
	if _, err := zero.Listings(TWMarketAuto); !errors.Is(err, errTWStockNotCreated) {
		t.Errorf("zero client: err = %v", err)
	}
}
