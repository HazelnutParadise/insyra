package datafetch

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/HazelnutParadise/insyra"
)

const geocodeSuccessBody = `{"ok":true,"query":{"lat":24.9884079,"lng":121.4598882},"result":{"villcode":"65000130032","county_name":"新北市","town_name":"土城區","village_name":"青雲里","village_eng":"Qingyun Vil.","county_id":"F","county_code":"65000","town_id":"F19","town_code":"65000130"}}`

const geocodeNotFoundBody = `{"ok":false,"error":"not_found","message":"No village polygon contains this point."}`

// newGeocoder builds a fetcher pointed at the given mock server URL, with retries
// off and no throttle unless a test overrides via the returned config mutation.
func newTestGeocoder(t *testing.T, baseURL string, mutate func(*TWGeocodingConfig)) *twGeocoder {
	t.Helper()
	cfg := TWGeocodingConfig{BaseURL: baseURL, RetryBackoff: time.Millisecond}
	if mutate != nil {
		mutate(&cfg)
	}
	g, err := TWGeocoding(cfg)
	if err != nil {
		t.Fatalf("TWGeocoding returned error: %v", err)
	}
	return g
}

func TestTWGeocodingConfigDefaults(t *testing.T) {
	g, err := TWGeocoding(TWGeocodingConfig{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if g.cfg.Timeout != defaultGeocodeTimeout {
		t.Errorf("Timeout = %v, want %v", g.cfg.Timeout, defaultGeocodeTimeout)
	}
	if g.cfg.BaseURL != defaultGeocodeBaseURL {
		t.Errorf("BaseURL = %q, want %q", g.cfg.BaseURL, defaultGeocodeBaseURL)
	}
	if g.cfg.Interval != 0 {
		t.Errorf("Interval = %v, want 0", g.cfg.Interval)
	}
}

func TestTWGeocodingConfigInvalid(t *testing.T) {
	for name, cfg := range map[string]TWGeocodingConfig{
		"negative interval": {Interval: -time.Second},
		"negative retries":  {Retries: -1},
		"negative backoff":  {RetryBackoff: -time.Second},
	} {
		if _, err := TWGeocoding(cfg); err == nil {
			t.Errorf("%s: expected error, got nil", name)
		}
	}
}

func TestReverseSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(geocodeSuccessBody))
	}))
	defer srv.Close()

	g := newTestGeocoder(t, srv.URL, nil)
	res, err := g.Reverse(24.9884079, 121.4598882)
	if err != nil {
		t.Fatalf("Reverse error: %v", err)
	}
	if res.CountyName != "新北市" || res.TownName != "土城區" || res.VillageName != "青雲里" {
		t.Errorf("unexpected region: %+v", res)
	}
	if res.VillCode != "65000130032" || res.TownCode != "65000130" {
		t.Errorf("unexpected codes: %+v", res)
	}
	if res.Lat != 24.9884079 || res.Lng != 121.4598882 {
		t.Errorf("coordinates not echoed: %+v", res)
	}
}

func TestReverseNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(geocodeNotFoundBody))
	}))
	defer srv.Close()

	g := newTestGeocoder(t, srv.URL, nil)
	_, err := g.Reverse(0, 0)
	if !errors.Is(err, ErrGeocodeNotFound) {
		t.Fatalf("expected ErrGeocodeNotFound, got %v", err)
	}
}

func TestReverseEmptyVillageEng(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"ok":true,"result":{"county_name":"南投縣","town_name":"信義鄉","village_name":"某里","village_eng":""}}`))
	}))
	defer srv.Close()

	g := newTestGeocoder(t, srv.URL, nil)
	res, err := g.Reverse(23.5, 120.9)
	if err != nil {
		t.Fatalf("Reverse error: %v", err)
	}
	if res.VillageEng != "" {
		t.Errorf("VillageEng = %q, want empty", res.VillageEng)
	}
	if res.CountyName != "南投縣" {
		t.Errorf("CountyName = %q", res.CountyName)
	}
}

func TestReverseRateLimited(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Ratelimit-Limit", "15")
		w.Header().Set("X-Ratelimit-Remaining", "0")
		w.Header().Set("X-Ratelimit-Reset", "1783699200")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	g := newTestGeocoder(t, srv.URL, func(c *TWGeocodingConfig) { c.Retries = 3 })
	_, err := g.Reverse(24.98, 121.45)
	if !errors.Is(err, ErrGeocodeRateLimited) {
		t.Fatalf("expected ErrGeocodeRateLimited, got %v", err)
	}
	var rl *RateLimitError
	if !errors.As(err, &rl) {
		t.Fatalf("expected *RateLimitError, got %T", err)
	}
	if rl.Limit != 15 || rl.Remaining != 0 {
		t.Errorf("rate-limit fields = %+v", rl)
	}
	if rl.ResetAt.Unix() != 1783699200 {
		t.Errorf("ResetAt = %v", rl.ResetAt)
	}
}

func TestReverseNoRetryOnRateLimit(t *testing.T) {
	var count int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&count, 1)
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	g := newTestGeocoder(t, srv.URL, func(c *TWGeocodingConfig) { c.Retries = 5 })
	_, _ = g.Reverse(24.98, 121.45)
	if got := atomic.LoadInt32(&count); got != 1 {
		t.Errorf("rate-limit was retried: %d requests, want 1", got)
	}
}

func TestReverseTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		_, _ = w.Write([]byte(geocodeSuccessBody))
	}))
	defer srv.Close()

	g := newTestGeocoder(t, srv.URL, func(c *TWGeocodingConfig) { c.Timeout = 40 * time.Millisecond })
	_, err := g.Reverse(24.98, 121.45)
	if !errors.Is(err, ErrGeocodeTimeout) {
		t.Fatalf("expected ErrGeocodeTimeout, got %v", err)
	}
}

func TestReverseRetryThenSucceed(t *testing.T) {
	var count int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&count, 1) == 1 {
			// Abruptly drop the connection to produce a transient network error.
			hj, ok := w.(http.Hijacker)
			if !ok {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			conn, _, err := hj.Hijack()
			if err == nil {
				_ = conn.Close()
			}
			return
		}
		_, _ = w.Write([]byte(geocodeSuccessBody))
	}))
	defer srv.Close()

	g := newTestGeocoder(t, srv.URL, func(c *TWGeocodingConfig) { c.Retries = 2 })
	res, err := g.Reverse(24.98, 121.45)
	if err != nil {
		t.Fatalf("expected success after retry, got %v", err)
	}
	if res.CountyName != "新北市" {
		t.Errorf("unexpected result: %+v", res)
	}
	if got := atomic.LoadInt32(&count); got < 2 {
		t.Errorf("expected at least 2 attempts, got %d", got)
	}
}

func TestReverseColsDedup(t *testing.T) {
	var count int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&count, 1)
		_, _ = w.Write([]byte(geocodeSuccessBody))
	}))
	defer srv.Close()

	g := newTestGeocoder(t, srv.URL, nil)
	lat := insyra.NewDataList(24.98, 24.98, 24.98)
	lng := insyra.NewDataList(121.45, 121.45, 121.45)
	dt, err := g.ReverseCols(lat, lng)
	if err != nil {
		t.Fatalf("ReverseCols error: %v", err)
	}
	if got := atomic.LoadInt32(&count); got != 1 {
		t.Errorf("duplicate coordinates not deduped: %d requests, want 1", got)
	}
	status := dt.GetColByName("GeocodeStatus").Data()
	if len(status) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(status))
	}
	for i, s := range status {
		if s != geocodeStatusOK {
			t.Errorf("row %d status = %v, want ok", i, s)
		}
	}
}

func TestReverseColsPerRowNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("lat") == "0" {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(geocodeNotFoundBody))
			return
		}
		_, _ = w.Write([]byte(geocodeSuccessBody))
	}))
	defer srv.Close()

	g := newTestGeocoder(t, srv.URL, nil)
	lat := insyra.NewDataList(24.98, 0.0)
	lng := insyra.NewDataList(121.45, 0.0)
	dt, err := g.ReverseCols(lat, lng)
	if err != nil {
		t.Fatalf("ReverseCols error: %v", err)
	}
	status := dt.GetColByName("GeocodeStatus").Data()
	if status[0] != geocodeStatusOK {
		t.Errorf("row 0 status = %v, want ok", status[0])
	}
	if status[1] != geocodeStatusNotFnd {
		t.Errorf("row 1 status = %v, want not_found", status[1])
	}
	county := dt.GetColByName("County").Data()
	if county[0] != "新北市" {
		t.Errorf("row 0 county = %v", county[0])
	}
	if county[1] != "" {
		t.Errorf("row 1 county = %v, want empty", county[1])
	}
}

func TestReverseTableIndexVsName(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(geocodeSuccessBody))
	}))
	defer srv.Close()

	g := newTestGeocoder(t, srv.URL, nil)

	latCol := insyra.NewDataList(24.98)
	latCol.SetName("lat")
	lngCol := insyra.NewDataList(121.45)
	lngCol.SetName("lng")
	src := insyra.NewDataTable(latCol, lngCol)

	byIndex, err := g.ReverseTable(src, "A", "B")
	if err != nil {
		t.Fatalf("ReverseTable(index) error: %v", err)
	}
	byName, err := g.ReverseTableByColName(src, "lat", "lng")
	if err != nil {
		t.Fatalf("ReverseTableByColName error: %v", err)
	}
	ci := byIndex.GetColByName("County").Data()
	cn := byName.GetColByName("County").Data()
	if len(ci) != 1 || len(cn) != 1 || ci[0] != cn[0] || ci[0] != "新北市" {
		t.Errorf("index vs name mismatch: %v vs %v", ci, cn)
	}
}

func TestReverseTableInvalidColumn(t *testing.T) {
	g := newTestGeocoder(t, "http://example.invalid", nil)
	latCol := insyra.NewDataList(24.98)
	latCol.SetName("lat")
	src := insyra.NewDataTable(latCol)
	if _, err := g.ReverseTableByColName(src, "lat", "does_not_exist"); err == nil {
		t.Error("expected error for missing column, got nil")
	}
}

func TestReverseColsRateLimitMidBatch(t *testing.T) {
	var count int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&count, 1) == 1 {
			_, _ = w.Write([]byte(geocodeSuccessBody))
			return
		}
		w.Header().Set("X-Ratelimit-Limit", "15")
		w.Header().Set("X-Ratelimit-Remaining", "0")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	g := newTestGeocoder(t, srv.URL, nil)
	lat := insyra.NewDataList(24.98, 25.01, 25.05)
	lng := insyra.NewDataList(121.45, 121.50, 121.55)
	dt, err := g.ReverseCols(lat, lng)
	var rl *RateLimitError
	if !errors.As(err, &rl) {
		t.Fatalf("expected *RateLimitError, got %v", err)
	}
	status := dt.GetColByName("GeocodeStatus").Data()
	if status[0] != geocodeStatusOK {
		t.Errorf("row 0 status = %v, want ok", status[0])
	}
	if status[1] != geocodeStatusPending || status[2] != geocodeStatusPending {
		t.Errorf("rows 1/2 status = %v/%v, want pending", status[1], status[2])
	}
	// The third row must never have been requested.
	if got := atomic.LoadInt32(&count); got != 2 {
		t.Errorf("expected 2 requests before halt, got %d", got)
	}
}

func TestReverseColsDedupServedAfterHalt(t *testing.T) {
	var count int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&count, 1) == 1 {
			_, _ = w.Write([]byte(geocodeSuccessBody))
			return
		}
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	g := newTestGeocoder(t, srv.URL, nil)
	// Row 2 duplicates row 0's coordinate; row 1 triggers the halt.
	lat := insyra.NewDataList(24.98, 25.01, 24.98)
	lng := insyra.NewDataList(121.45, 121.50, 121.45)
	dt, err := g.ReverseCols(lat, lng)
	var rl *RateLimitError
	if !errors.As(err, &rl) {
		t.Fatalf("expected *RateLimitError, got %v", err)
	}
	status := dt.GetColByName("GeocodeStatus").Data()
	if status[0] != geocodeStatusOK || status[1] != geocodeStatusPending || status[2] != geocodeStatusOK {
		t.Errorf("statuses = %v, want [ok pending ok]", status)
	}
	if got := atomic.LoadInt32(&count); got != 2 {
		t.Errorf("duplicate after halt issued a request: %d, want 2", got)
	}
}

func TestReverseColsInvalidCoordinate(t *testing.T) {
	var count int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&count, 1)
		_, _ = w.Write([]byte(geocodeSuccessBody))
	}))
	defer srv.Close()

	g := newTestGeocoder(t, srv.URL, nil)
	lat := insyra.NewDataList("not-a-number", 24.98)
	lng := insyra.NewDataList(121.45, 121.45)
	dt, err := g.ReverseCols(lat, lng)
	if err != nil {
		t.Fatalf("ReverseCols error: %v", err)
	}
	status := dt.GetColByName("GeocodeStatus").Data()
	if status[0] != geocodeStatusInvalid {
		t.Errorf("row 0 status = %v, want invalid_coordinate", status[0])
	}
	if status[1] != geocodeStatusOK {
		t.Errorf("row 1 status = %v, want ok", status[1])
	}
	// Only the valid row should have hit the network.
	if got := atomic.LoadInt32(&count); got != 1 {
		t.Errorf("invalid coordinate issued a request: %d, want 1", got)
	}
	if lat0 := dt.GetColByName("Lat").Data()[0]; lat0 != nil {
		t.Errorf("invalid row Lat = %v, want nil", lat0)
	}
}

func TestReverseGeocodeResultToDataTable(t *testing.T) {
	r := &ReverseGeocodeResult{Lat: 24.98, Lng: 121.45, CountyName: "新北市", TownName: "土城區"}
	dt := r.ToDataTable()
	if county := dt.GetColByName("County").Data(); len(county) != 1 || county[0] != "新北市" {
		t.Errorf("County column = %v", county)
	}
	if town := dt.GetColByName("Town").Data(); len(town) != 1 || town[0] != "土城區" {
		t.Errorf("Town column = %v", town)
	}
	// Nil receiver returns an empty table without panicking.
	var nilRes *ReverseGeocodeResult
	_ = nilRes.ToDataTable()
}

func TestCacheHitAvoidsNetwork(t *testing.T) {
	var count int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&count, 1)
		_, _ = w.Write([]byte(geocodeSuccessBody))
	}))
	defer srv.Close()

	g := newTestGeocoder(t, srv.URL, func(c *TWGeocodingConfig) { c.Cache = NewMemoryGeocodeCache() })
	if _, err := g.Reverse(24.98, 121.45); err != nil {
		t.Fatalf("first Reverse error: %v", err)
	}
	if _, err := g.Reverse(24.98, 121.45); err != nil {
		t.Fatalf("second Reverse error: %v", err)
	}
	if got := atomic.LoadInt32(&count); got != 1 {
		t.Errorf("cache miss: %d requests, want 1", got)
	}
}

func TestCacheNotFoundCached(t *testing.T) {
	var count int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&count, 1)
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(geocodeNotFoundBody))
	}))
	defer srv.Close()

	g := newTestGeocoder(t, srv.URL, func(c *TWGeocodingConfig) { c.Cache = NewMemoryGeocodeCache() })
	if _, err := g.Reverse(0, 0); !errors.Is(err, ErrGeocodeNotFound) {
		t.Fatalf("first: %v", err)
	}
	if _, err := g.Reverse(0, 0); !errors.Is(err, ErrGeocodeNotFound) {
		t.Fatalf("second: %v", err)
	}
	if got := atomic.LoadInt32(&count); got != 1 {
		t.Errorf("not_found not cached: %d requests, want 1", got)
	}
}

func TestCacheTransientNotCached(t *testing.T) {
	var count int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&count, 1)
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	g := newTestGeocoder(t, srv.URL, func(c *TWGeocodingConfig) { c.Cache = NewMemoryGeocodeCache() })
	if _, err := g.Reverse(24.98, 121.45); !errors.Is(err, ErrGeocodeRateLimited) {
		t.Fatalf("first: %v", err)
	}
	if _, err := g.Reverse(24.98, 121.45); !errors.Is(err, ErrGeocodeRateLimited) {
		t.Fatalf("second: %v", err)
	}
	if got := atomic.LoadInt32(&count); got != 2 {
		t.Errorf("transient error was cached: %d requests, want 2", got)
	}
}

func TestFileGeocodeCachePersists(t *testing.T) {
	path := filepath.Join(t.TempDir(), "geocache.json")
	key := geocodeCacheKey(24.98, 121.45)

	c1 := NewFileGeocodeCache(path)
	c1.Set(key, &ReverseGeocodeResult{CountyName: "新北市", TownName: "土城區"})
	c1.Set(geocodeCacheKey(0, 0), nil) // not_found marker

	c2 := NewFileGeocodeCache(path)
	got, ok := c2.Get(key)
	if !ok || got == nil || got.CountyName != "新北市" {
		t.Errorf("success not persisted: ok=%v got=%+v", ok, got)
	}
	nf, ok := c2.Get(geocodeCacheKey(0, 0))
	if !ok || nf != nil {
		t.Errorf("not_found marker not persisted: ok=%v nf=%+v", ok, nf)
	}
	if _, ok := c2.Get(geocodeCacheKey(1, 1)); ok {
		t.Error("absent key reported as present")
	}
}
