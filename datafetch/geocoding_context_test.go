package datafetch

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/HazelnutParadise/insyra"
)

// #250: every TWGeocodingClient fetch has a Context form. A done context stops
// the wait, the request and the retry backoff, and a batch keeps the rows it
// already resolved.

func TestReverseContextStopsOnACancelledContext(t *testing.T) {
	srv, count := countingGeocodeServer(t)
	g := newTestGeocoder(t, srv.URL, nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := g.ReverseContext(ctx, 24.98, 121.45); !errors.Is(err, context.Canceled) {
		t.Errorf("got %v, want context.Canceled", err)
	}
	if n := atomic.LoadInt32(count); n != 0 {
		t.Errorf("sent %d requests under a cancelled context, want none", n)
	}
}

func TestGeocodingRefusesANilContext(t *testing.T) {
	srv, count := countingGeocodeServer(t)
	g := newTestGeocoder(t, srv.URL, nil)
	var nilCtx context.Context

	if _, err := g.ReverseContext(nilCtx, 24.98, 121.45); !errors.Is(err, errNilContext) {
		t.Errorf("ReverseContext returned %v, want errNilContext", err)
	}
	if _, err := g.ReverseColsContext(nilCtx, insyra.NewDataList(24.98), insyra.NewDataList(121.45)); !errors.Is(err, errNilContext) {
		t.Errorf("ReverseColsContext returned %v, want errNilContext", err)
	}
	if n := atomic.LoadInt32(count); n != 0 {
		t.Errorf("sent %d requests, want none", n)
	}
}

func TestReverseContextReportsTheCallersDeadline(t *testing.T) {
	var count int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&count, 1)
		select {
		case <-r.Context().Done():
		case <-time.After(5 * time.Second):
		}
		_, _ = w.Write([]byte(geocodeSuccessBody))
	}))
	defer srv.Close()
	g := newTestGeocoder(t, srv.URL, func(c *TWGeocodingConfig) { c.Retries = 2 })

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err := g.ReverseContext(ctx, 24.98, 121.45)

	if !errors.Is(err, context.DeadlineExceeded) || errors.Is(err, ErrGeocodeTimeout) {
		t.Errorf("got %v, want context.DeadlineExceeded and not ErrGeocodeTimeout", err)
	}
	if elapsed := time.Since(start); elapsed > 3*time.Second {
		t.Errorf("returned after %v; the context ended after 50ms", elapsed)
	}
	if n := atomic.LoadInt32(&count); n != 1 {
		t.Errorf("the server saw %d requests, want 1: a request the caller cut off is not retried", n)
	}
}

// cancelOnFirstSet cancels its context when the first result is stored, that
// is, right after the first request has been read in full.
type cancelOnFirstSet struct {
	cancel context.CancelFunc
	once   sync.Once
}

func (c *cancelOnFirstSet) Get(string) (*ReverseGeocodeResult, bool) { return nil, false }
func (c *cancelOnFirstSet) Set(string, *ReverseGeocodeResult)        { c.once.Do(c.cancel) }

func TestReverseColsContextKeepsWhatItResolved(t *testing.T) {
	srv, count := countingGeocodeServer(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	g := newTestGeocoder(t, srv.URL, func(c *TWGeocodingConfig) { c.Cache = &cancelOnFirstSet{cancel: cancel} })

	// The third row repeats the first.
	lat := insyra.NewDataList(24.98, 25.01, 24.98, 25.05)
	lng := insyra.NewDataList(121.45, 121.50, 121.45, 121.55)
	dt, err := g.ReverseColsContext(ctx, lat, lng)

	if !errors.Is(err, context.Canceled) {
		t.Errorf("got %v, want context.Canceled", err)
	}
	if dt == nil {
		t.Fatal("a cancelled batch returned no table")
	}
	want := []any{geocodeStatusOK, geocodeStatusPending, geocodeStatusOK, geocodeStatusPending}
	if got := dt.GetColByName("GeocodeStatus").Data(); !reflect.DeepEqual(got, want) {
		t.Errorf("GeocodeStatus is %v, want %v", got, want)
	}
	if county := dt.GetColByName("County").Data()[2]; county != "新北市" {
		t.Errorf("the repeated row's County is %v, want 新北市 from the batch's first answer", county)
	}
	if n := atomic.LoadInt32(count); n != 1 {
		t.Errorf("sent %d requests, want 1", n)
	}
}

func TestReverseTableContextTakesSelectors(t *testing.T) {
	srv, _ := countingGeocodeServer(t)
	g := newTestGeocoder(t, srv.URL, nil)

	dt, err := g.ReverseTableContext(context.Background(), latLngTable(), insyra.Name("lat"), insyra.Name("lng"))

	if err != nil {
		t.Fatal(err)
	}
	if got := dt.GetColByName("County").Data(); len(got) != 1 || got[0] != "新北市" {
		t.Errorf("County is %v, want [新北市]", got)
	}
}
