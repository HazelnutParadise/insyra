package datafetch

import (
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"
)

// stockDayFloorBody is what STOCK_DAY answered for 2009-12 on 2026-11-07.
const stockDayFloorBody = `{"stat":"查詢日期小於99年1月4日，請重新查詢!","total":0}`

func tpexDailyKey(date, code string) string {
	return fixtureKey("/www/zh-tw/afterTrading/tradingStock", query("code", code, "date", date, "response", "json"))
}

func TestTWStockDailyPricesBeforeTheFloorIsRefused(t *testing.T) {
	stock, transport := newFixtureTWStock(t, TWStockConfig{Retries: 3, RetryBackoff: time.Millisecond})
	transport.replies[stockDayKey("20091201", "2330")] = []fixtureReply{
		{status: http.StatusOK, body: []byte(stockDayFloorBody)},
		{status: http.StatusOK, body: stockDayBody()},
	}
	_, err := stock.DailyPrices("2330", dateUTC(2009, time.December, 1), dateUTC(2009, time.December, 31), TWMarketTWSE)
	if !errors.Is(err, ErrTWStockRefused) {
		t.Fatalf("a month before 2010 should match ErrTWStockRefused: %v", err)
	}
	for _, want := range []string{"2330", "2009-12-01", "2009-12-31", "查詢日期小於99年1月4日", "2010-01-04"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("message should contain %q: %v", want, err)
		}
	}
	if got := transport.requestCount(); got != 1 {
		t.Fatalf("a refusal is final and must not be retried: %d requests", got)
	}
}

func TestTWStockRangeAcrossTheFloorIsRefusedNotShortened(t *testing.T) {
	stock, transport := newFixtureTWStock(t, TWStockConfig{})
	transport.replies[stockDayKey("20091201", "2330")] = []fixtureReply{{status: http.StatusOK, body: []byte(stockDayFloorBody)}}
	transport.replies[stockDayKey("20100101", "2330")] = []fixtureReply{{status: http.StatusOK, body: stockDayBody(`["99/01/04","1,000","64,900","65.00","65.00","64.80","64.90","0.40","10"]`)}}
	dt, err := stock.DailyPrices("2330", dateUTC(2009, time.December, 15), dateUTC(2010, time.January, 10), TWMarketTWSE)
	if !errors.Is(err, ErrTWStockRefused) || dt != nil {
		t.Fatalf("a range reaching before 2010 should fail, not return the later rows: dt=%v err=%v", dt, err)
	}
}

func TestTWStockDailyPricesFromTheFloorIsUnchanged(t *testing.T) {
	stock, transport := newFixtureTWStock(t, TWStockConfig{})
	transport.replies[stockDayKey("20100101", "2330")] = []fixtureReply{{status: http.StatusOK, body: stockDayBody(`["99/01/04","1,000","64,900","65.00","65.00","64.80","64.90","0.40","10"]`)}}
	dt, err := stock.DailyPrices("2330", dateUTC(2010, time.January, 4), dateUTC(2010, time.January, 31), TWMarketTWSE)
	if err != nil {
		t.Fatalf("2010-01-04 is served: %v", err)
	}
	if dt.NumRows() != 1 || !dt.GetColByName("Date").Data()[0].(time.Time).Equal(dateUTC(2010, time.January, 4)) {
		t.Fatalf("want the one 2010-01-04 row, got %d rows", dt.NumRows())
	}
}

func TestTWStockDailyPricesAdjustedBeforeTheFloorIsRefused(t *testing.T) {
	stock, transport := newFixtureTWStock(t, TWStockConfig{})
	transport.replies[stockDayKey("20091201", "2330")] = []fixtureReply{{status: http.StatusOK, body: []byte(stockDayFloorBody)}}
	if _, err := stock.DailyPricesAdjusted("2330", dateUTC(2009, time.December, 1), dateUTC(2009, time.December, 31), TWMarketTWSE); !errors.Is(err, ErrTWStockRefused) {
		t.Fatalf("DailyPricesAdjusted before 2010 should match ErrTWStockRefused: %v", err)
	}
}

func TestTWStockAutoAsksTPExForARefusedMonth(t *testing.T) {
	stock, transport := newFixtureTWStock(t, TWStockConfig{})
	transport.replies[stockDayKey("20260801", "6488")] = []fixtureReply{{status: http.StatusOK, body: []byte(stockDayFloorBody)}}
	transport.addFixture(t, tpexDailyKey("2026/08/01", "6488"), "tpex_trading_stock_6488_202608.json")
	dt, err := stock.DailyPrices("6488", dateUTC(2026, time.August, 3), dateUTC(2026, time.August, 4), TWMarketAuto)
	if err != nil {
		t.Fatalf("Auto should take TPEx's rows for a month TWSE refuses: %v", err)
	}
	if dt.NumRows() == 0 || dt.GetColByName("Market").Data()[0] != "TPEx" {
		t.Fatalf("want TPEx rows, got %d rows", dt.NumRows())
	}
}

func TestTWStockAutoReportsTheRefusalWhenTPExHasNothing(t *testing.T) {
	stock, transport := newFixtureTWStock(t, TWStockConfig{})
	transport.replies[stockDayKey("20091201", "2330")] = []fixtureReply{{status: http.StatusOK, body: []byte(stockDayFloorBody)}}
	transport.replies[tpexDailyKey("2009/12/01", "2330")] = []fixtureReply{{status: http.StatusOK, body: []byte(`{"tables":[{"title":"個股日成交資訊","data":[],"fields":["日 期","成交仟股","成交仟元","開盤","最高","最低","收盤","漲跌","筆數"],"totalCount":0}],"stat":"ok"}`)}}
	if _, err := stock.DailyPrices("2330", dateUTC(2009, time.December, 1), dateUTC(2009, time.December, 31), TWMarketAuto); !errors.Is(err, ErrTWStockRefused) {
		t.Fatalf("Auto with no TPEx data should report TWSE's refusal: %v", err)
	}
}

func TestTWStockMonthAfterTodayIsRefused(t *testing.T) {
	stock, transport := newFixtureTWStock(t, TWStockConfig{})
	transport.replies[stockDayKey("20300101", "2330")] = []fixtureReply{{status: http.StatusOK, body: []byte(`{"stat":"查詢日期大於今日，請重新查詢!","total":0}`)}}
	_, err := stock.DailyPrices("2330", dateUTC(2030, time.January, 1), dateUTC(2030, time.January, 31), TWMarketTWSE)
	if !errors.Is(err, ErrTWStockRefused) || strings.Contains(err.Error(), "2010-01-04") {
		t.Fatalf("a future month should match ErrTWStockRefused without the 2010 hint: %v", err)
	}
}

func TestTWStockOtherPayloadStatusIsNotARefusal(t *testing.T) {
	stock, transport := newFixtureTWStock(t, TWStockConfig{})
	transport.replies[stockDayKey("20260801", "2330")] = []fixtureReply{{status: http.StatusOK, body: []byte(`{"stat":"系統忙碌中"}`)}}
	_, err := stock.DailyPrices("2330", dateUTC(2026, time.August, 1), dateUTC(2026, time.August, 31), TWMarketTWSE)
	if err == nil || errors.Is(err, ErrTWStockRefused) {
		t.Fatalf("another payload status is an error but not a refusal: %v", err)
	}
}

// tpexThousandSharesBody is TPEx's answer for 8069 in 2009-12 as measured on
// 2026-11-07, trimmed to two days: older months head volume 成交仟股.
const tpexThousandSharesBody = `{"tables":[{"title":"個股日成交資訊","date":"20091201","data":[["98/12/01","15,450","1,165,786","73.20","76.00","73.20","76.00","4.90","5,818"],["98/12/02","31,510","2,462,136","80.20","81.30","75.00","77.30","1.30","13,082"]],"fields":["日 期","成交仟股","成交仟元","開盤","最高","最低","收盤","漲跌","筆數"],"totalCount":2}],"stat":"ok"}`

func TestTWStockAutoReachesTPExHistoryBeforeTheFloor(t *testing.T) {
	stock, transport := newFixtureTWStock(t, TWStockConfig{})
	transport.replies[stockDayKey("20091201", "8069")] = []fixtureReply{{status: http.StatusOK, body: []byte(stockDayFloorBody)}}
	transport.replies[tpexDailyKey("2009/12/01", "8069")] = []fixtureReply{{status: http.StatusOK, body: []byte(tpexThousandSharesBody)}}
	dt, err := stock.DailyPrices("8069", dateUTC(2009, time.December, 1), dateUTC(2009, time.December, 2), TWMarketAuto)
	if err != nil {
		t.Fatalf("a TPEx stock in 2009 should come from TPEx: %v", err)
	}
	if dt.NumRows() != 2 {
		t.Fatalf("rows = %d, want 2", dt.NumRows())
	}
	if got := dt.GetColByName("Volume").Data()[0]; got != int64(15450000) {
		t.Fatalf("15,450 thousand shares should be 15450000 shares, got %v", got)
	}
}
