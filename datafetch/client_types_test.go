package datafetch_test

import (
	"testing"

	"github.com/HazelnutParadise/insyra/datafetch"
)

// #251: every constructor returns a type a caller can name in its own
// declarations, such as the fields of a struct that holds the clients a
// program builds at start-up.
type clients struct {
	stocks   *datafetch.TWStockClient
	yahoo    *datafetch.YFinanceClient
	ticker   *datafetch.YFTicker
	geocoder *datafetch.TWGeocodingClient
	maps     *datafetch.GoogleMapsStoresClient
}

func TestConstructorsReturnExportedTypes(t *testing.T) {
	var c clients
	var err error
	if c.stocks, err = datafetch.TWStock(datafetch.TWStockConfig{}); err != nil {
		t.Fatalf("TWStock: %v", err)
	}
	if c.yahoo, err = datafetch.YFinance(datafetch.YFinanceConfig{}); err != nil {
		t.Fatalf("YFinance: %v", err)
	}
	c.ticker = c.yahoo.Ticker("AAPL")
	if c.geocoder, err = datafetch.TWGeocoding(datafetch.TWGeocodingConfig{}); err != nil {
		t.Fatalf("TWGeocoding: %v", err)
	}
	c.maps = datafetch.GoogleMapsStores()

	if c.stocks == nil || c.yahoo == nil || c.ticker == nil || c.geocoder == nil || c.maps == nil {
		t.Fatalf("a constructor returned nil: %+v", c)
	}
}
