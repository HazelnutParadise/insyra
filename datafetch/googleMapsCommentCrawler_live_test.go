//go:build gmaps_live

// Live smoke test against Google Maps.
//
// Excluded from the default build so `go test ./...` never depends on Google
// or sends it traffic. Run explicitly with:
//
//	go test -tags gmaps_live -run TestGoogleMapsSearchLive ./datafetch/
package datafetch

import (
	"strings"
	"testing"
)

func TestGoogleMapsSearchLive(t *testing.T) {
	stores := GoogleMapsStores().Search("鼎泰豐")
	if len(stores) == 0 {
		t.Fatal("Search returned no stores; Google may have changed the response")
	}
	for _, s := range stores {
		if !gmapsFeatureIDRe.MatchString(s.ID) {
			t.Errorf("store %q has an ID that is not a feature ID: %q", s.Name, s.ID)
		}
	}
	if !strings.Contains(stores[0].Name, "鼎泰豐") {
		t.Errorf("the first result is %q, which does not look like a match", stores[0].Name)
	}
	t.Logf("live result: %d stores, first %s (%s)", len(stores), stores[0].Name, stores[0].ID)
}
