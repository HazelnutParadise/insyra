//go:build geocoding_live

// Live smoke test against the real geocoding.zuola.com endpoint.
//
// Excluded from the default build so `go test ./...` never spends the scarce
// 15-requests/hour quota. Run explicitly with:
//
//	go test -tags geocoding_live -run TestReverseLive ./datafetch/
package datafetch

import (
	"errors"
	"testing"
	"time"
)

func TestReverseLive(t *testing.T) {
	g, err := TWGeocoding(TWGeocodingConfig{Timeout: 20 * time.Second})
	if err != nil {
		t.Fatalf("TWGeocoding: %v", err)
	}
	res, err := g.Reverse(24.9884079, 121.4598882)
	if err != nil {
		if errors.Is(err, ErrGeocodeRateLimited) {
			t.Skipf("live quota exhausted: %v", err)
		}
		t.Fatalf("Reverse: %v", err)
	}
	if res.CountyName == "" || res.TownName == "" || res.VillageName == "" {
		t.Errorf("empty region fields in live result: %+v", res)
	}
	t.Logf("live result: %s%s%s (%s)", res.CountyName, res.TownName, res.VillageName, res.VillCode)
}
