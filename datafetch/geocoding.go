package datafetch

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/datafetch/internal/limiter"
	json "github.com/goccy/go-json"
)

// defaults for the TWGeocoding fetcher.
const (
	defaultGeocodeTimeout   = 15 * time.Second
	defaultGeocodeUserAgent = "Mozilla/5.0 (compatible; insyra-datafetch/1.0; +https://github.com/HazelnutParadise/insyra)"
	defaultGeocodeBaseURL   = "https://geocoding.zuola.com/api/reverse"
	defaultGeocodeBackoff   = 300 * time.Millisecond
)

// GeocodeStatus values written into the batch result's GeocodeStatus column.
const (
	geocodeStatusOK      = "ok"
	geocodeStatusNotFnd  = "not_found"
	geocodeStatusPending = "pending"
	geocodeStatusInvalid = "invalid_coordinate"
)

// TWGeocodingConfig configures a TWGeocoding fetcher. A zero value is valid and
// yields sensible defaults.
type TWGeocodingConfig struct {
	// Timeout is the per-request timeout. Default: 15s.
	Timeout time.Duration
	// Interval is the minimum spacing between requests (client-side throttle).
	// 0 disables throttling. Must be >= 0.
	Interval time.Duration
	// UserAgent is the HTTP User-Agent header. Default: a browser-like UA.
	UserAgent string
	// Retries is the number of retry attempts for transient failures. Must be >= 0.
	Retries int
	// RetryBackoff is the base backoff between retries. Must be >= 0. Default: 300ms.
	RetryBackoff time.Duration
	// BaseURL overrides the reverse-geocoding endpoint (for tests/mocks or a
	// future paid/self-hosted tier). Default: the official zuola endpoint.
	BaseURL string
	// Cache, when non-nil, serves and stores definitive results to conserve the
	// scarce request quota. Use NewMemoryGeocodeCache or NewFileGeocodeCache.
	Cache GeocodeCache
}

func (cfg TWGeocodingConfig) normalize() (TWGeocodingConfig, error) {
	out := cfg
	if out.Timeout <= 0 {
		out.Timeout = defaultGeocodeTimeout
	}
	if out.Interval < 0 {
		return TWGeocodingConfig{}, errors.New("datafetch: TWGeocoding Interval must be >= 0")
	}
	if out.UserAgent == "" {
		out.UserAgent = defaultGeocodeUserAgent
	}
	if out.Retries < 0 {
		return TWGeocodingConfig{}, errors.New("datafetch: TWGeocoding Retries must be >= 0")
	}
	if out.RetryBackoff < 0 {
		return TWGeocodingConfig{}, errors.New("datafetch: TWGeocoding RetryBackoff must be >= 0")
	}
	if out.RetryBackoff == 0 {
		out.RetryBackoff = defaultGeocodeBackoff
	}
	if out.BaseURL == "" {
		out.BaseURL = defaultGeocodeBaseURL
	}
	return out, nil
}

// ReverseGeocodeResult is the typed result of a single reverse-geocode lookup.
type ReverseGeocodeResult struct {
	Lat         float64 `json:"lat"`
	Lng         float64 `json:"lng"`
	VillCode    string  `json:"villcode"`
	CountyName  string  `json:"county_name"`
	TownName    string  `json:"town_name"`
	VillageName string  `json:"village_name"`
	VillageEng  string  `json:"village_eng"`
	CountyID    string  `json:"county_id"`
	CountyCode  string  `json:"county_code"`
	TownID      string  `json:"town_id"`
	TownCode    string  `json:"town_code"`
}

// ToDataTable returns a single-row DataTable representation of the result, with
// the same column order as the batch methods' output (minus GeocodeStatus).
func (r *ReverseGeocodeResult) ToDataTable() *insyra.DataTable {
	if r == nil {
		return insyra.NewDataTable()
	}
	return insyra.NewDataTable(
		newNamedCol("Lat", []any{r.Lat}),
		newNamedCol("Lng", []any{r.Lng}),
		newNamedCol("County", []any{r.CountyName}),
		newNamedCol("Town", []any{r.TownName}),
		newNamedCol("Village", []any{r.VillageName}),
		newNamedCol("VillageEng", []any{r.VillageEng}),
		newNamedCol("VillCode", []any{r.VillCode}),
		newNamedCol("CountyCode", []any{r.CountyCode}),
		newNamedCol("TownCode", []any{r.TownCode}),
		newNamedCol("CountyID", []any{r.CountyID}),
		newNamedCol("TownID", []any{r.TownID}),
	)
}

// geocodeAPIResponse mirrors the zuola reverse API JSON envelope.
type geocodeAPIResponse struct {
	OK      bool   `json:"ok"`
	Error   string `json:"error"`
	Message string `json:"message"`
	Result  struct {
		VillCode    string `json:"villcode"`
		CountyName  string `json:"county_name"`
		TownName    string `json:"town_name"`
		VillageName string `json:"village_name"`
		VillageEng  string `json:"village_eng"`
		CountyID    string `json:"county_id"`
		CountyCode  string `json:"county_code"`
		TownID      string `json:"town_id"`
		TownCode    string `json:"town_code"`
	} `json:"result"`
}

// twGeocoder is a stateful reverse-geocoding fetcher. Construct it with TWGeocoding.
type twGeocoder struct {
	cfg     TWGeocodingConfig
	client  *http.Client
	limiter *limiter.IntervalLimiter
	cache   GeocodeCache
}

// TWGeocoding creates a Taiwan reverse-geocoding fetcher backed by
// geocoding.zuola.com. Note the free tier is limited to 15 requests/hour per IP;
// use a GeocodeCache and the batch methods' de-duplication to conserve it.
func TWGeocoding(cfg TWGeocodingConfig) (*twGeocoder, error) {
	normalized, err := cfg.normalize()
	if err != nil {
		return nil, err
	}
	return &twGeocoder{
		cfg:     normalized,
		client:  &http.Client{Timeout: normalized.Timeout},
		limiter: limiter.NewIntervalLimiter(normalized.Interval),
		cache:   normalized.Cache,
	}, nil
}

// Reverse resolves a single coordinate to its Taiwan administrative region.
// It returns ErrGeocodeNotFound when the point is outside any village, a
// *RateLimitError when the quota is exhausted, and ErrGeocodeTimeout on timeout.
func (g *twGeocoder) Reverse(lat, lng float64) (*ReverseGeocodeResult, error) {
	key := geocodeCacheKey(lat, lng)
	if g.cache != nil {
		if cached, ok := g.cache.Get(key); ok {
			if cached == nil { // cached not_found marker
				return nil, ErrGeocodeNotFound
			}
			res := *cached
			res.Lat, res.Lng = lat, lng
			return &res, nil
		}
	}

	var lastErr error
	for attempt := 0; attempt <= g.cfg.Retries; attempt++ {
		if err := g.limiter.Wait(context.Background()); err != nil {
			return nil, err
		}

		res, err := g.doReverse(lat, lng)
		if err == nil {
			if g.cache != nil {
				g.cache.Set(key, res)
			}
			return res, nil
		}

		lastErr = err
		if errors.Is(err, ErrGeocodeNotFound) {
			if g.cache != nil {
				g.cache.Set(key, nil) // cache the definitive miss
			}
			return nil, err
		}
		if !geocodeRetryable(err) {
			return nil, err
		}
		if attempt < g.cfg.Retries {
			g.sleepBackoff(attempt)
		}
	}
	return nil, lastErr
}

// doReverse performs one HTTP request without retry/cache logic.
func (g *twGeocoder) doReverse(lat, lng float64) (*ReverseGeocodeResult, error) {
	req, err := http.NewRequest(http.MethodGet, g.cfg.BaseURL, nil)
	if err != nil {
		return nil, err
	}
	q := req.URL.Query()
	q.Set("lat", strconv.FormatFloat(lat, 'f', -1, 64))
	q.Set("lng", strconv.FormatFloat(lng, 'f', -1, 64))
	req.URL.RawQuery = q.Encode()
	req.Header.Set("User-Agent", g.cfg.UserAgent)

	resp, err := g.client.Do(req)
	if err != nil {
		if isTimeoutErr(err) {
			return nil, ErrGeocodeTimeout
		}
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	switch resp.StatusCode {
	case http.StatusOK:
		body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		if err != nil {
			return nil, err
		}
		var apiResp geocodeAPIResponse
		if err := json.Unmarshal(body, &apiResp); err != nil {
			return nil, fmt.Errorf("datafetch: geocoding decode failed: %w", err)
		}
		if !apiResp.OK {
			if apiResp.Error == "not_found" {
				return nil, ErrGeocodeNotFound
			}
			return nil, fmt.Errorf("datafetch: geocoding error: %s", strings.TrimSpace(apiResp.Message))
		}
		return &ReverseGeocodeResult{
			Lat:         lat,
			Lng:         lng,
			VillCode:    apiResp.Result.VillCode,
			CountyName:  apiResp.Result.CountyName,
			TownName:    apiResp.Result.TownName,
			VillageName: apiResp.Result.VillageName,
			VillageEng:  apiResp.Result.VillageEng,
			CountyID:    apiResp.Result.CountyID,
			CountyCode:  apiResp.Result.CountyCode,
			TownID:      apiResp.Result.TownID,
			TownCode:    apiResp.Result.TownCode,
		}, nil
	case http.StatusNotFound:
		return nil, ErrGeocodeNotFound
	case http.StatusTooManyRequests:
		return nil, parseRateLimitHeaders(resp.Header)
	default:
		return nil, fmt.Errorf("datafetch: geocoding unexpected HTTP status %d", resp.StatusCode)
	}
}

func (g *twGeocoder) sleepBackoff(attempt int) {
	if g.cfg.RetryBackoff <= 0 {
		return
	}
	time.Sleep(g.cfg.RetryBackoff * time.Duration(attempt+1))
}

// ReverseCols reverse-geocodes parallel latitude/longitude lists and returns a
// DataTable with the original coordinates, the resolved region columns, and a
// GeocodeStatus column. Identical coordinates are de-duplicated so each unique
// point costs at most one request. Per-row failures are tolerated. If the quota
// is exhausted mid-batch, the already-resolved rows are returned together with a
// *RateLimitError, and the remaining rows are marked "pending".
func (g *twGeocoder) ReverseCols(lat, lng *insyra.DataList) (*insyra.DataTable, error) {
	if lat == nil || lng == nil {
		return nil, errors.New("datafetch: ReverseCols requires non-nil lat and lng lists")
	}
	if lat.Len() != lng.Len() {
		return nil, fmt.Errorf("datafetch: ReverseCols lat/lng length mismatch (%d vs %d)", lat.Len(), lng.Len())
	}
	n := lat.Len()

	latVals := make([]any, n)
	lngVals := make([]any, n)
	countyVals := make([]any, n)
	townVals := make([]any, n)
	villageVals := make([]any, n)
	villageEngVals := make([]any, n)
	villCodeVals := make([]any, n)
	countyCodeVals := make([]any, n)
	townCodeVals := make([]any, n)
	countyIDVals := make([]any, n)
	townIDVals := make([]any, n)
	statusVals := make([]any, n)
	for i := range n {
		countyVals[i] = ""
		townVals[i] = ""
		villageVals[i] = ""
		villageEngVals[i] = ""
		villCodeVals[i] = ""
		countyCodeVals[i] = ""
		townCodeVals[i] = ""
		countyIDVals[i] = ""
		townIDVals[i] = ""
	}

	type batchOutcome struct {
		res    *ReverseGeocodeResult
		status string
	}
	apply := func(i int, o batchOutcome) {
		statusVals[i] = o.status
		if o.res != nil {
			countyVals[i] = o.res.CountyName
			townVals[i] = o.res.TownName
			villageVals[i] = o.res.VillageName
			villageEngVals[i] = o.res.VillageEng
			villCodeVals[i] = o.res.VillCode
			countyCodeVals[i] = o.res.CountyCode
			townCodeVals[i] = o.res.TownCode
			countyIDVals[i] = o.res.CountyID
			townIDVals[i] = o.res.TownID
		}
	}

	seen := make(map[string]batchOutcome)
	var rateErr *RateLimitError
	halted := false

	for i := range n {
		latF, ok1 := toFloat(lat.Get(i))
		lngF, ok2 := toFloat(lng.Get(i))
		if !ok1 || !ok2 {
			statusVals[i] = geocodeStatusInvalid
			continue
		}
		latVals[i] = latF
		lngVals[i] = lngF

		// Serve de-duplicated coordinates from this batch's memo first — even
		// after a rate-limit halt, an already-resolved coordinate has a known
		// answer and needs no request.
		key := geocodeCacheKey(latF, lngF)
		if outcome, ok := seen[key]; ok {
			apply(i, outcome)
			continue
		}
		if halted {
			statusVals[i] = geocodeStatusPending
			continue
		}

		res, err := g.Reverse(latF, lngF)
		switch {
		case err == nil:
			outcome := batchOutcome{res: res, status: geocodeStatusOK}
			seen[key] = outcome
			apply(i, outcome)
		case errors.Is(err, ErrGeocodeNotFound):
			outcome := batchOutcome{status: geocodeStatusNotFnd}
			seen[key] = outcome
			apply(i, outcome)
		default:
			var rl *RateLimitError
			if errors.As(err, &rl) {
				rateErr = rl
				halted = true
				statusVals[i] = geocodeStatusPending
				continue
			}
			outcome := batchOutcome{status: "error: " + err.Error()}
			seen[key] = outcome
			apply(i, outcome)
		}
	}

	dt := insyra.NewDataTable(
		newNamedCol("Lat", latVals),
		newNamedCol("Lng", lngVals),
		newNamedCol("County", countyVals),
		newNamedCol("Town", townVals),
		newNamedCol("Village", villageVals),
		newNamedCol("VillageEng", villageEngVals),
		newNamedCol("VillCode", villCodeVals),
		newNamedCol("CountyCode", countyCodeVals),
		newNamedCol("TownCode", townCodeVals),
		newNamedCol("CountyID", countyIDVals),
		newNamedCol("TownID", townIDVals),
		newNamedCol("GeocodeStatus", statusVals),
	)
	dt.SetName("TWGeocoding.Reverse")

	if rateErr != nil {
		return dt, rateErr
	}
	return dt, nil
}

// ReverseTable reverse-geocodes the given DataTable's latitude/longitude columns,
// addressed by Excel-style column index ("A", "B", ...). See ReverseCols for the
// output shape and batch semantics.
func (g *twGeocoder) ReverseTable(dt *insyra.DataTable, latCol, lngCol string) (*insyra.DataTable, error) {
	if dt == nil {
		return nil, errors.New("datafetch: ReverseTable requires a non-nil DataTable")
	}
	lat := dt.GetCol(latCol)
	lng := dt.GetCol(lngCol)
	if lat == nil || lng == nil {
		return nil, fmt.Errorf("datafetch: ReverseTable could not resolve columns %q / %q", latCol, lngCol)
	}
	return g.ReverseCols(lat, lng)
}

// ReverseTableByColName reverse-geocodes the given DataTable's latitude/longitude
// columns, addressed by column name. See ReverseCols for the output shape and
// batch semantics.
func (g *twGeocoder) ReverseTableByColName(dt *insyra.DataTable, latColName, lngColName string) (*insyra.DataTable, error) {
	if dt == nil {
		return nil, errors.New("datafetch: ReverseTableByColName requires a non-nil DataTable")
	}
	lat := dt.GetColByName(latColName)
	lng := dt.GetColByName(lngColName)
	if lat == nil || lng == nil {
		return nil, fmt.Errorf("datafetch: ReverseTableByColName could not resolve columns %q / %q", latColName, lngColName)
	}
	return g.ReverseCols(lat, lng)
}

// ---- helpers ----

func newNamedCol(name string, vals []any) *insyra.DataList {
	dl := insyra.NewDataList(vals...)
	dl.SetName(name)
	return dl
}

// geocodeCacheKey builds the coordinate-based cache key, quantized to 6 decimal
// places (~0.1 m). That resolution is far finer than GPS accuracy or village
// boundaries, so a cache hit returns the correct region in practice; it is not a
// coarse grid that could mix up neighbouring villages.
func geocodeCacheKey(lat, lng float64) string {
	return fmt.Sprintf("%.6f,%.6f", lat, lng)
}

// parseRateLimitHeaders reads the X-Ratelimit-* headers into a *RateLimitError.
// Missing/unparseable headers leave the corresponding field at its zero value.
func parseRateLimitHeaders(h http.Header) *RateLimitError {
	rl := &RateLimitError{}
	if v := h.Get("X-Ratelimit-Limit"); v != "" {
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
			rl.Limit = n
		}
	}
	if v := h.Get("X-Ratelimit-Remaining"); v != "" {
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
			rl.Remaining = n
		}
	}
	if v := h.Get("X-Ratelimit-Reset"); v != "" {
		if n, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64); err == nil {
			rl.ResetAt = time.Unix(n, 0)
		}
	}
	return rl
}

// geocodeRetryable reports whether an error is worth retrying. Only timeouts and
// transient network errors qualify; not-found and rate-limit are definitive.
func geocodeRetryable(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrGeocodeNotFound) || errors.Is(err, ErrGeocodeRateLimited) {
		return false
	}
	if errors.Is(err, ErrGeocodeTimeout) {
		return true
	}
	var ne net.Error
	return errors.As(err, &ne)
}

func isTimeoutErr(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var ne net.Error
	return errors.As(err, &ne) && ne.Timeout()
}

// toFloat coerces a DataList element into a float64 coordinate.
func toFloat(v any) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case float32:
		return float64(x), true
	case int:
		return float64(x), true
	case int64:
		return float64(x), true
	case int32:
		return float64(x), true
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(x), 64)
		return f, err == nil
	case nil:
		return 0, false
	default:
		// Fallback for json.Number and other numeric-stringer types.
		f, err := strconv.ParseFloat(strings.TrimSpace(fmt.Sprintf("%v", x)), 64)
		return f, err == nil
	}
}

// ---- cache ----

// GeocodeCache is an optional store for reverse-geocode results, keyed by the
// exact coordinate. Implementations must be safe for concurrent use. Only
// definitive outcomes are ever stored: a non-nil result for a success, and a nil
// result to mark a definitive not_found. Transient failures are never cached.
type GeocodeCache interface {
	// Get returns (result, true) for a cached success, (nil, true) for a cached
	// not_found, and (nil, false) when the key is absent.
	Get(key string) (*ReverseGeocodeResult, bool)
	// Set stores a success (non-nil r) or a not_found marker (nil r).
	Set(key string, r *ReverseGeocodeResult)
}

type memoryGeocodeCache struct {
	mu sync.RWMutex
	m  map[string]*ReverseGeocodeResult
}

// NewMemoryGeocodeCache returns an in-process, concurrency-safe cache.
func NewMemoryGeocodeCache() GeocodeCache {
	return &memoryGeocodeCache{m: make(map[string]*ReverseGeocodeResult)}
}

func (c *memoryGeocodeCache) Get(key string) (*ReverseGeocodeResult, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, ok := c.m[key]
	if !ok {
		return nil, false
	}
	if v == nil {
		return nil, true
	}
	cp := *v
	return &cp, true
}

func (c *memoryGeocodeCache) Set(key string, r *ReverseGeocodeResult) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if r == nil {
		c.m[key] = nil
		return
	}
	cp := *r
	c.m[key] = &cp
}

type fileGeocodeCache struct {
	mu   sync.Mutex
	path string
	m    map[string]*ReverseGeocodeResult
}

// NewFileGeocodeCache returns a cache backed by a JSON file at path. Existing
// contents are loaded on construction and every Set is persisted, so resolved
// coordinates survive across runs — the highest-value option under the 15
// requests/hour quota. Unreadable/corrupt files start from an empty cache.
func NewFileGeocodeCache(path string) GeocodeCache {
	c := &fileGeocodeCache{path: path, m: make(map[string]*ReverseGeocodeResult)}
	c.load()
	return c
}

func (c *fileGeocodeCache) load() {
	data, err := os.ReadFile(c.path)
	if err != nil {
		return // absent or unreadable → empty cache
	}
	loaded := make(map[string]*ReverseGeocodeResult)
	if err := json.Unmarshal(data, &loaded); err != nil {
		return // corrupt → empty cache
	}
	c.m = loaded
}

func (c *fileGeocodeCache) Get(key string) (*ReverseGeocodeResult, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	v, ok := c.m[key]
	if !ok {
		return nil, false
	}
	if v == nil {
		return nil, true
	}
	cp := *v
	return &cp, true
}

func (c *fileGeocodeCache) Set(key string, r *ReverseGeocodeResult) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if r == nil {
		c.m[key] = nil
	} else {
		cp := *r
		c.m[key] = &cp
	}
	c.persist()
}

func (c *fileGeocodeCache) persist() {
	data, err := json.Marshal(c.m)
	if err != nil {
		insyra.LogWarning("datafetch", "NewFileGeocodeCache", "Failed to marshal cache: %v", err)
		return
	}
	if err := os.WriteFile(c.path, data, 0o644); err != nil {
		insyra.LogWarning("datafetch", "NewFileGeocodeCache", "Failed to write cache file %q: %v", c.path, err)
	}
}
