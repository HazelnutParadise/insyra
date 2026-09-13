package datafetch

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/HazelnutParadise/insyra"
)

// #249. The crawler reads endpoints Google changes without notice, so these
// tests stand in a local server for Google and pin what the crawler sends and
// how it reads the answer. The live checks are in
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
	c.storeReviewUrl = server.URL + "/reviews"
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

// reviewRecord lays out one review the way a review page does, filling only
// the positions GetReviews reads: the rating at 1, the times at 2, the
// reviewer at 3 and the text at 27.
func reviewRecord(rating int, relative string, posted time.Time, name, contribURL, text string) []any {
	record := make([]any, 49)
	record[1] = rating
	record[2] = []any{relative, nil, strconv.FormatInt(posted.UnixMilli(), 10)}
	record[3] = []any{name, "https://lh3.googleusercontent.com/a-/avatar", contribURL, 18, 173, []any{nil, 1}}
	record[5] = "Ci9DQUlRQUNvZENodHljRjlvT21sNVlUSm9lVmM1ZVRSRmNFZFZWbXRMWTBGMVdYYxAB"
	record[26] = "zh-Hant"
	record[27] = text
	return record
}

// reviewPage lays out a review page: the records at [1][10][2] and the next
// page's token at [1][10][6], which the last page does not have.
func reviewPage(t *testing.T, next string, records ...any) string {
	t.Helper()
	block := make([]any, 7)
	block[2] = records
	if next != "" {
		block[6] = next
	}
	outer := make([]any, 11)
	outer[10] = block
	b, err := json.Marshal([]any{nil, outer})
	if err != nil {
		t.Fatal(err)
	}
	return ")]}'\n" + string(b)
}

// reviewRequest is what one review request asked for, read back out of its
// reqpld parameter.
type reviewRequest struct {
	sort    float64
	size    float64
	storeID string
	token   string
}

func readReviewRequest(r *http.Request) (reviewRequest, bool) {
	var payload []any
	if json.Unmarshal([]byte(r.URL.Query().Get("reqpld")), &payload) != nil || len(payload) < 2 {
		return reviewRequest{}, false
	}
	outer, _ := payload[1].([]any)
	if len(outer) < 10 {
		return reviewRequest{}, false
	}
	args, _ := outer[9].([]any)
	if len(args) < 20 {
		return reviewRequest{}, false
	}
	var req reviewRequest
	req.sort, _ = args[1].(float64)
	req.size, _ = args[9].(float64)
	if ids, _ := args[11].([]any); len(ids) == 1 {
		req.storeID, _ = ids[0].(string)
	}
	req.token, _ = args[19].(string)
	return req, true
}

// reviewServer answers each review request with the page its token names, and
// records the requests in the order they arrived.
func reviewServer(t *testing.T, pages map[string]string) (*httptest.Server, func() []reviewRequest) {
	t.Helper()
	var mu sync.Mutex
	var seen []reviewRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req, ok := readReviewRequest(r)
		if !ok {
			t.Errorf("unreadable review request: %s", r.URL.RawQuery)
		}
		mu.Lock()
		seen = append(seen, req)
		mu.Unlock()
		page, found := pages[req.token]
		if !found {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = w.Write([]byte(page))
	}))
	t.Cleanup(server.Close)
	return server, func() []reviewRequest {
		mu.Lock()
		defer mu.Unlock()
		return append([]reviewRequest(nil), seen...)
	}
}

func TestGetReviewsReadsReviewPages(t *testing.T) {
	const store = "0x3442a819486a12c5:0x999160f21e520bde"
	posted := time.Date(2026, 7, 5, 12, 0, 0, 0, time.UTC)
	server, requests := reviewServer(t, map[string]string{
		"": reviewPage(t, "page-2",
			reviewRecord(5, "2 個月前", posted, "陌生人。", "https://www.google.com/maps/contrib/111798366699800592810/reviews?hl=zh-Hant-TW", "第一行<br>第二行 &amp; 更多"),
			reviewRecord(4, "3 個月前", posted.AddDate(0, -1, 0), "Mia", "https://www.google.com/maps/contrib/114576/reviews", "好吃"),
		),
		"page-2": reviewPage(t, "",
			reviewRecord(1, "1 年前", posted.AddDate(-1, 0, 0), "Dan", "https://www.google.com/maps/contrib/107256/reviews", "太吵"),
		),
	})

	reviews := crawlerFor(server).GetReviews(store, 0,
		GoogleMapsStoreReviewsFetchingOptions{SortBy: SortByNewest, MaxWaitingInterval_Milliseconds: 1000})

	if len(reviews) != 3 {
		t.Fatalf("got %d reviews over two pages, want 3: %+v", len(reviews), reviews)
	}
	want := GoogleMapsStoreReview{
		Reviewer:   "陌生人。",
		ReviewerID: "111798366699800592810",
		ReviewTime: "2 個月前",
		ReviewDate: "2026-07-05",
		Content:    "第一行\n第二行 & 更多",
		Rating:     5,
	}
	if reviews[0] != want {
		t.Errorf("first review is %+v, want %+v", reviews[0], want)
	}
	if reviews[2].Rating != 1 || reviews[2].ReviewDate != "2025-07-05" || reviews[2].Reviewer != "Dan" {
		t.Errorf("the second page's review is %+v", reviews[2])
	}

	wantRequests := []reviewRequest{
		{sort: 2, size: 10, storeID: store, token: ""},
		{sort: 2, size: 10, storeID: store, token: "page-2"},
	}
	if got := requests(); !reflect.DeepEqual(got, wantRequests) {
		t.Errorf("requests were %+v, want %+v", got, wantRequests)
	}
}

// A waiting interval of exactly 1000 used to reach rand.IntN(0), which panics.
// The second page still has a token, so this also pins stopping at pageCount.
func TestGetReviewsAtTheMinimumWaitingInterval(t *testing.T) {
	posted := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	server, requests := reviewServer(t, map[string]string{
		"":       reviewPage(t, "page-2", reviewRecord(5, "", posted, "A", "", "")),
		"page-2": reviewPage(t, "page-3", reviewRecord(4, "", posted, "B", "", "")),
	})

	start := time.Now()
	reviews := crawlerFor(server).GetReviews("0x1:0x2", 2, GoogleMapsStoreReviewsFetchingOptions{MaxWaitingInterval_Milliseconds: 1000})

	if len(reviews) != 2 {
		t.Errorf("got %d reviews over two pages, want 2", len(reviews))
	}
	if n := len(requests()); n != 2 {
		t.Errorf("sent %d requests for pageCount 2, want 2", n)
	}
	if elapsed := time.Since(start); elapsed < time.Second {
		t.Errorf("waited %v between pages, want at least 1s", elapsed)
	}
}

// A zero field in the options means its default, so setting only one field
// is not a mistake worth a warning. An empty review list is not one either.
func TestGetReviewsZeroOptionsAreDefaults(t *testing.T) {
	logged := captureGmapsWarnings(t)
	server, requests := reviewServer(t, map[string]string{"": reviewPage(t, "")})

	crawlerFor(server).GetReviews("0x1:0x2", 1, GoogleMapsStoreReviewsFetchingOptions{SortBy: SortByNewest})
	crawlerFor(server).GetReviews("0x1:0x2", 1, GoogleMapsStoreReviewsFetchingOptions{MaxWaitingInterval_Milliseconds: 2000})

	if logged.Len() != 0 {
		t.Errorf("setting one option logged a warning: %q", logged.String())
	}
	got := requests()
	if len(got) != 2 || got[0].sort != 2 || got[1].sort != 1 {
		t.Errorf("sort orders sent were %+v, want newest (2) then relevance (1)", got)
	}
}

func TestGetReviewsWarnsWhenThePageHasNoReviewBlock(t *testing.T) {
	logged := captureGmapsWarnings(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(")]}'\n" + `[["er",null,null,null,null,403]]`))
	}))
	defer server.Close()

	if reviews := crawlerFor(server).GetReviews("0x1:0x2", 1); reviews != nil {
		t.Errorf("GetReviews returned %v for a response with no review block", reviews)
	}
	if !strings.Contains(logged.String(), "GetReviews") || !strings.Contains(logged.String(), "format") {
		t.Errorf("no warning explained the unreadable page: %q", logged.String())
	}
}
