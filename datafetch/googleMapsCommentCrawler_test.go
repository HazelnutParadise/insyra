package datafetch

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/HazelnutParadise/insyra"
)

// #249. The crawler reads endpoints Google changes without notice, so these
// tests stand in a local server for Google and pin what the crawler sends and
// how it reads the answer. The live check is in
// googleMapsCommentCrawler_live_test.go.

func captureGmapsWarnings(t *testing.T) *bytes.Buffer {
	t.Helper()
	level := insyra.Config.GetLogLevel()
	insyra.Config.SetLogLevel(insyra.LogLevelWarning)
	var buf bytes.Buffer
	prev := log.Writer()
	log.SetOutput(&buf)
	t.Cleanup(func() {
		log.SetOutput(prev)
		insyra.Config.SetLogLevel(level)
	})
	return &buf
}

func crawlerFor(server *httptest.Server) *googleMapsStoreCrawler {
	c := GoogleMapsStores()
	c.storeSearchUrl = server.URL + "/search"
	c.storeReviewUrl = server.URL + "/listugcposts"
	return c
}

// searchEntry lays out one result the way a search response does: the store
// record is the entry's second element, with the feature ID at 10 and the
// name at 11.
func searchEntry(id, name any) []any {
	record := make([]any, 12)
	record[10] = id
	record[11] = name
	return []any{nil, record}
}

func searchResponse(t *testing.T, entries ...any) string {
	t.Helper()
	top := make([]any, gmapsSearchResults+1)
	top[gmapsSearchResults] = entries
	b, err := json.Marshal(top)
	if err != nil {
		t.Fatal(err)
	}
	return ")]}'\n" + string(b)
}

func TestGoogleMapsStoresNeedsNoNetwork(t *testing.T) {
	c := GoogleMapsStores()
	if c == nil {
		t.Fatal("GoogleMapsStores returned nil")
	}
	if c.client.Timeout <= 0 {
		t.Error("requests have no timeout")
	}
	if c.storeSearchUrl != gmapsSearchURL || c.storeReviewUrl != gmapsReviewURL {
		t.Errorf("unexpected endpoints: %q, %q", c.storeSearchUrl, c.storeReviewUrl)
	}
}

func TestGoogleMapsSearchReadsTheResultList(t *testing.T) {
	const (
		banqiao = "0x3442a819486a12c5:0x999160f21e520bde"
		xinyi   = "0x3442a9821392326d:0x93faef9e2359365c"
	)
	var got searchRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = searchRequest{path: r.URL.Path, query: r.URL.Query(), agent: r.Header.Get("User-Agent")}
		_, _ = w.Write([]byte(searchResponse(t,
			searchEntry(banqiao, "鼎泰豐 板橋店"),
			searchEntry(xinyi, "鼎泰豐 信義店"),
			searchEntry(banqiao, "鼎泰豐 板橋店"), // a repeat
			[]any{nil, "not a record"},
			searchEntry("ChIJ-not-a-feature-id", "wrong id shape"),
			searchEntry(12345.0, "a number, not an id"),
		)))
	}))
	defer server.Close()

	stores := crawlerFor(server).Search("鼎泰豐")

	want := []GoogleMapsStoreData{{ID: banqiao, Name: "鼎泰豐 板橋店"}, {ID: xinyi, Name: "鼎泰豐 信義店"}}
	if !reflect.DeepEqual(stores, want) {
		t.Errorf("Search returned %v, want %v", stores, want)
	}
	if got.path != "/search" || got.query.Get("tbm") != "map" || got.query.Get("q") != "鼎泰豐" || got.query.Get("pb") != gmapsSearchPB {
		t.Errorf("unexpected request: path %q, query %v", got.path, got.query)
	}
	if !strings.Contains(got.agent, "Chrome/") {
		t.Errorf("the request did not send the browser user agent: %q", got.agent)
	}
}

type searchRequest struct {
	path  string
	query url.Values
	agent string
}

func TestGoogleMapsSearchWarnsWhenNoStoreComesBack(t *testing.T) {
	logged := captureGmapsWarnings(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(")]}'\n[]"))
	}))
	defer server.Close()

	if stores := crawlerFor(server).Search("nothing here"); stores != nil {
		t.Errorf("Search returned %v for a response with no stores", stores)
	}
	if !strings.Contains(logged.String(), "Search") || !strings.Contains(logged.String(), "no stores") {
		t.Errorf("no warning explained the empty result: %q", logged.String())
	}
}

func TestGoogleMapsSearchReportsAFailedRequest(t *testing.T) {
	logged := captureGmapsWarnings(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	if stores := crawlerFor(server).Search("鼎泰豐"); stores != nil {
		t.Errorf("Search returned %v for a failed request", stores)
	}
	if !strings.Contains(logged.String(), "500") {
		t.Errorf("the warning does not give the status: %q", logged.String())
	}
}

// A waiting interval of exactly 1000 used to reach rand.IntN(0), which panics.
func TestGetReviewsAtTheMinimumWaitingInterval(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next := "page-2"
		if calls.Add(1) > 1 {
			next = ""
		}
		_, _ = w.Write([]byte(`)]}'` + "\n" + `[null,"` + next + `",[[1,2,3]]]`))
	}))
	defer server.Close()

	start := time.Now()
	reviews := crawlerFor(server).GetReviews("0x1:0x2", 2, GoogleMapsStoreReviewsFetchingOptions{MaxWaitingInterval_Milliseconds: 1000})

	if len(reviews) != 2 {
		t.Errorf("got %d reviews over two pages, want 2", len(reviews))
	}
	if calls.Load() != 2 {
		t.Errorf("sent %d requests, want 2", calls.Load())
	}
	if elapsed := time.Since(start); elapsed < time.Second {
		t.Errorf("waited %v between pages, want at least 1s", elapsed)
	}
}

// A zero field in the options means its default, so setting only one field
// is not a mistake worth a warning.
func TestGetReviewsZeroOptionsAreDefaults(t *testing.T) {
	logged := captureGmapsWarnings(t)
	var sortArg string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pb := r.URL.Query().Get("pb")
		sortArg = pb[strings.LastIndex(pb, "!"):]
		_, _ = w.Write([]byte(`)]}'` + "\n" + `[null,"",[]]`))
	}))
	defer server.Close()

	crawlerFor(server).GetReviews("0x1:0x2", 1, GoogleMapsStoreReviewsFetchingOptions{SortBy: SortByNewest})
	if logged.Len() != 0 {
		t.Errorf("a zero MaxWaitingInterval logged a warning: %q", logged.String())
	}
	if sortArg != "!1e2" {
		t.Errorf("SortByNewest was sent as %q, want !1e2", sortArg)
	}

	crawlerFor(server).GetReviews("0x1:0x2", 1, GoogleMapsStoreReviewsFetchingOptions{MaxWaitingInterval_Milliseconds: 2000})
	if logged.Len() != 0 {
		t.Errorf("a zero SortBy logged a warning: %q", logged.String())
	}
	if sortArg != "!1e1" {
		t.Errorf("a zero SortBy was sent as %q, want relevance !1e1", sortArg)
	}
}
