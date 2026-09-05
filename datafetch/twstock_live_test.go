package datafetch

import (
	"os"
	"testing"
	"time"
)

func TestTWStockLive(t *testing.T) {
	if os.Getenv("INSYRA_RUN_LIVE_TWSTOCK") != "1" {
		t.Skip("set INSYRA_RUN_LIVE_TWSTOCK=1 to run live TWSE/TPEx requests")
	}

	stock, err := TWStock(TWStockConfig{Interval: 300 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	t.Run("DailyPrices", func(t *testing.T) {
		dt, err := stock.DailyPrices("2330", time.Now().AddDate(0, 0, -7), time.Now(), TWMarketTWSE)
		if err != nil || dt.NumRows() == 0 {
			t.Fatalf("DailyPrices rows=%d error=%v", dt.NumRows(), err)
		}
		if _, ok := dt.GetColByName("Close").Data()[0].(float64); !ok {
			t.Fatalf("Close type = %T", dt.GetColByName("Close").Data()[0])
		}
	})
	t.Run("InstitutionalTrades", func(t *testing.T) {
		dt, err := stock.InstitutionalTrades(time.Now().AddDate(0, 0, -1), TWMarketTWSE)
		if err != nil || dt.NumRows() == 0 {
			t.Fatalf("InstitutionalTrades rows=%d error=%v", dt.NumRows(), err)
		}
	})
	t.Run("MarginBalance", func(t *testing.T) {
		dt, err := stock.MarginBalance(time.Now().AddDate(0, 0, -1), TWMarketTWSE)
		if err != nil || dt.NumRows() == 0 {
			t.Fatalf("MarginBalance rows=%d error=%v", dt.NumRows(), err)
		}
	})
	t.Run("AllDailyQuotes", func(t *testing.T) {
		dt, err := stock.AllDailyQuotes(TWMarketTWSE)
		if err != nil || dt.NumRows() == 0 {
			t.Fatalf("AllDailyQuotes rows=%d error=%v", dt.NumRows(), err)
		}
	})
	t.Run("ExRights", func(t *testing.T) {
		dt, err := stock.ExRights(time.Now().AddDate(0, 0, -90), time.Now(), TWMarketTWSE)
		if err != nil || dt.NumRows() == 0 {
			t.Fatalf("ExRights rows=%d error=%v", dt.NumRows(), err)
		}
		if _, ok := dt.GetColByName("Date").Data()[0].(time.Time); !ok {
			t.Fatalf("Date type = %T", dt.GetColByName("Date").Data()[0])
		}
		if _, ok := dt.GetColByName("AdjFactor").Data()[0].(float64); !ok {
			t.Fatalf("AdjFactor type = %T", dt.GetColByName("AdjFactor").Data()[0])
		}
	})
	t.Run("DailyPricesAdjusted", func(t *testing.T) {
		dt, err := stock.DailyPricesAdjusted("2330", time.Now().AddDate(0, 0, -90), time.Now(), TWMarketTWSE)
		if err != nil || dt.NumRows() == 0 {
			t.Fatalf("DailyPricesAdjusted rows=%d error=%v", dt.NumRows(), err)
		}
		if _, ok := dt.GetColByName("AdjClose").Data()[0].(float64); !ok {
			t.Fatalf("AdjClose type = %T", dt.GetColByName("AdjClose").Data()[0])
		}
	})
}
