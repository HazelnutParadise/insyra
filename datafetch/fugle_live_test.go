package datafetch

import (
	"os"
	"testing"
	"time"
)

// TestFugleLive calls the real API. It needs FUGLE_API_KEY, the whole key.
func TestFugleLive(t *testing.T) {
	key := os.Getenv("FUGLE_API_KEY")
	if key == "" {
		t.Skip("set FUGLE_API_KEY to run live Fugle requests")
	}
	c, err := Fugle(FugleConfig{APIKey: key, Retries: 2})
	if err != nil {
		t.Fatal(err)
	}
	t.Run("daily candles over a year are cut and joined", func(t *testing.T) {
		to := time.Now().UTC()
		dt, err := c.Stock.HistoricalCandles("2330", FugleHistoricalCandlesOptions{From: to.AddDate(-2, 0, 0), To: to})
		if err != nil || dt.NumRows() < 300 {
			t.Fatalf("rows=%d err=%v", dt.NumRows(), err)
		}
		dates := dt.GetColByName("date").Data()
		for i := 1; i < len(dates); i++ {
			if !dates[i].(time.Time).After(dates[i-1].(time.Time)) {
				t.Fatalf("dates not strictly ascending at %d", i)
			}
		}
	})
	t.Run("technical", func(t *testing.T) {
		dt, err := c.Stock.TechnicalKDJ("2330", FugleTimeframeDay, 9, 3, 3, FugleTechnicalOptions{From: time.Date(2024, 8, 1, 0, 0, 0, 0, time.UTC), To: time.Date(2024, 8, 10, 0, 0, 0, 0, time.UTC)})
		if err != nil || dt.NumRows() == 0 || dt.GetColByName("j") == nil {
			t.Fatalf("err=%v", err)
		}
	})
	t.Run("ticker", func(t *testing.T) {
		dt, err := c.Stock.IntradayTicker("2330")
		if err != nil || dt.NumRows() != 1 || dt.GetColByName("referencePrice").Data()[0] == nil {
			t.Fatalf("err=%v", err)
		}
	})
	t.Run("quote", func(t *testing.T) {
		dt, err := c.Stock.IntradayQuote("2330")
		if err != nil || dt.NumRows() != 1 || dt.GetColByName("symbol") == nil {
			t.Fatalf("rows=%d err=%v", dt.NumRows(), err)
		}
	})
}
