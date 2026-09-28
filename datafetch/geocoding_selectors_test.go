package datafetch

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/HazelnutParadise/insyra"
)

// #212 (DF-5): ReverseTable takes each column as the library's selector, so
// one method serves an Excel-style index, a name and a position.

func latLngTable() *insyra.DataTable {
	lat := insyra.NewDataList(24.98)
	lat.SetName("lat")
	lng := insyra.NewDataList(121.45)
	lng.SetName("lng")
	return insyra.NewDataTable(lat, lng)
}

// countingGeocodeServer answers every request with a success and counts them.
func countingGeocodeServer(t *testing.T) (*httptest.Server, *int32) {
	t.Helper()
	var count int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&count, 1)
		_, _ = w.Write([]byte(geocodeSuccessBody))
	}))
	t.Cleanup(srv.Close)
	return srv, &count
}

func TestReverseTableTakesColumnSelectors(t *testing.T) {
	srv, _ := countingGeocodeServer(t)
	g := newTestGeocoder(t, srv.URL, nil)
	src := latLngTable()

	calls := map[string]func() (*insyra.DataTable, error){
		`"A", "B"`: func() (*insyra.DataTable, error) { return g.ReverseTable(src, "A", "B") },
		`insyra.Name("lat"), insyra.Name("lng")`: func() (*insyra.DataTable, error) {
			return g.ReverseTable(src, insyra.Name("lat"), insyra.Name("lng"))
		},
		`0, 1`:   func() (*insyra.DataTable, error) { return g.ReverseTable(src, 0, 1) },
		`-2, -1`: func() (*insyra.DataTable, error) { return g.ReverseTable(src, -2, -1) },
		`ReverseTableByColName(dt, "lat", "lng")`: func() (*insyra.DataTable, error) {
			return g.ReverseTableByColName(src, "lat", "lng")
		},
	}
	for name, call := range calls {
		dt, err := call()
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		if got := dt.GetColByName("County").Data(); len(got) != 1 || got[0] != "新北市" {
			t.Errorf("%s: County is %v, want [新北市]", name, got)
		}
	}
}

func TestReverseTableReadsABareStringAsAnIndex(t *testing.T) {
	srv, count := countingGeocodeServer(t)
	g := newTestGeocoder(t, srv.URL, nil)

	_, err := g.ReverseTable(latLngTable(), "lat", "lng")

	if err == nil {
		t.Fatal(`ReverseTable(dt, "lat", "lng") read the strings as column names`)
	}
	for _, want := range []string{"latitude", `"lat"`, "insyra.Name"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the error %q does not mention %s", err, want)
		}
	}
	if n := atomic.LoadInt32(count); n != 0 {
		t.Errorf("sent %d requests before refusing, want none", n)
	}
}

func TestReverseTableNamesTheMissingColumn(t *testing.T) {
	srv, count := countingGeocodeServer(t)
	g := newTestGeocoder(t, srv.URL, nil)

	_, err := g.ReverseTable(latLngTable(), insyra.Name("lat"), insyra.Name("longitude"))

	if err == nil || !strings.Contains(err.Error(), "longitude column") {
		t.Fatalf("got %v, want an error about the longitude column", err)
	}
	if n := atomic.LoadInt32(count); n != 0 {
		t.Errorf("sent %d requests before refusing, want none", n)
	}
}
