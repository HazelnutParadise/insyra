package datafetch

import (
	"strings"
	"testing"
	"time"

	"github.com/HazelnutParadise/insyra"
)

// #251: the client types are exported, so a caller can hold a zero value or a
// nil pointer that never went through its constructor. Every method must say
// which constructor to use instead of dereferencing the missing HTTP client.

func TestClientsNotBuiltByTheirConstructorFailWithoutPanicking(t *testing.T) {
	day := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)

	t.Run("TWStockClient", func(t *testing.T) {
		receivers := []*TWStockClient{{}, nil}
		methods := []struct {
			name string
			f    func(*TWStockClient) (any, error)
		}{
			{"DailyPrices", func(c *TWStockClient) (any, error) { return c.DailyPrices("2330", day, day, TWMarketTWSE) }},
			{"DailyPricesAdjusted", func(c *TWStockClient) (any, error) { return c.DailyPricesAdjusted("2330", day, day, TWMarketTWSE) }},
			{"ExRights", func(c *TWStockClient) (any, error) { return c.ExRights(day, day, TWMarketTWSE) }},
			{"InstitutionalTrades", func(c *TWStockClient) (any, error) { return c.InstitutionalTrades(day, TWMarketTWSE) }},
			{"MarginBalance", func(c *TWStockClient) (any, error) { return c.MarginBalance(day, TWMarketTWSE) }},
			{"AllDailyQuotes", func(c *TWStockClient) (any, error) { return c.AllDailyQuotes(TWMarketTWSE) }},
		}
		for _, recv := range receivers {
			for _, m := range methods {
				_, err := m.f(recv)
				if err == nil {
					t.Errorf("%v.%s: expected non-nil error, got nil", recv, m.name)
				} else if !strings.Contains(err.Error(), "TWStock") {
					t.Errorf("%v.%s: error %q does not contain \"TWStock\"", recv, m.name, err.Error())
				}
			}
		}
	})

	t.Run("TWGeocodingClient", func(t *testing.T) {
		lat := insyra.NewDataList(24.98)
		lat.SetName("lat")
		lng := insyra.NewDataList(121.45)
		lng.SetName("lng")
		src := insyra.NewDataTable(lat, lng)

		receivers := []*TWGeocodingClient{{}, nil}
		methods := []struct {
			name string
			f    func(*TWGeocodingClient) (any, error)
		}{
			{"Reverse", func(c *TWGeocodingClient) (any, error) { return c.Reverse(24.98, 121.45) }},
			{"ReverseCols", func(c *TWGeocodingClient) (any, error) { return c.ReverseCols(lat, lng) }},
			{"ReverseTable", func(c *TWGeocodingClient) (any, error) { return c.ReverseTable(src, "A", "B") }},
			{"ReverseTableByColName", func(c *TWGeocodingClient) (any, error) { return c.ReverseTableByColName(src, "lat", "lng") }},
		}
		for _, recv := range receivers {
			for _, m := range methods {
				_, err := m.f(recv)
				if err == nil {
					t.Errorf("%v.%s: expected non-nil error, got nil", recv, m.name)
				} else if !strings.Contains(err.Error(), "TWGeocoding") {
					t.Errorf("%v.%s: error %q does not contain \"TWGeocoding\"", recv, m.name, err.Error())
				}
			}
		}
	})

	t.Run("YFinanceClient", func(t *testing.T) {
		// var y YFinanceClient -> y.Ticker("AAPL").Info()
		var y YFinanceClient
		_, err := y.Ticker("AAPL").Info()
		if err == nil || !strings.Contains(err.Error(), "YFinance") {
			t.Errorf("var YFinanceClient -> Ticker(\"AAPL\").Info(): expected error containing \"YFinance\", got %v", err)
		}

		// var tk YFTicker -> tk.Info()
		var tk YFTicker
		_, err = tk.Info()
		if err == nil || !strings.Contains(err.Error(), "YFinance") {
			t.Errorf("var YFTicker -> Info(): expected error containing \"YFinance\", got %v", err)
		}

		// (*YFinanceClient)(nil).Ticker("AAPL").Info()
		_, err = (*YFinanceClient)(nil).Ticker("AAPL").Info()
		if err == nil || !strings.Contains(err.Error(), "YFinance") {
			t.Errorf("(*YFinanceClient)(nil).Ticker(\"AAPL\").Info(): expected error containing \"YFinance\", got %v", err)
		}
	})

	t.Run("GoogleMapsStoresClient", func(t *testing.T) {
		receivers := []*GoogleMapsStoresClient{{}, nil}
		for _, recv := range receivers {
			logged := captureGmapsWarnings(t)
			if got := recv.Search("鼎泰豐"); got != nil {
				t.Errorf("%v.Search: expected nil return, got %v", recv, got)
			}
			if !strings.Contains(logged.String(), "GoogleMapsStores") {
				t.Errorf("%v.Search: logged warning %q does not contain \"GoogleMapsStores\"", recv, logged.String())
			}

			logged = captureGmapsWarnings(t)
			if got := recv.GetReviews("0x1:0x2", 1); got != nil {
				t.Errorf("%v.GetReviews: expected nil return, got %v", recv, got)
			}
			if !strings.Contains(logged.String(), "GoogleMapsStores") {
				t.Errorf("%v.GetReviews: logged warning %q does not contain \"GoogleMapsStores\"", recv, logged.String())
			}
		}
	})
}
