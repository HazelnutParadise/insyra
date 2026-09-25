// Google Maps store search and reviews.
//
// Both read endpoints Google's own pages call, which Google changes without
// notice. Search replays the result list the Maps page requests. Reviews come
// from the review window on Google Search results, because Google Maps itself
// shows a signed-out visitor only five reviews (#249, measured 2026-09-13).

package datafetch

import (
	"bytes"
	"fmt"
	"html"
	"io"
	"math/rand/v2"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/HazelnutParadise/insyra"
	json "github.com/goccy/go-json"
)

const (
	gmapsSearchURL = "https://www.google.com.tw/search"

	// gmapsReviewURL is what the review window on Google Search results calls
	// for each page of reviews. It needs no cookies and pages with a token.
	gmapsReviewURL = "https://www.google.com.tw/httpservice/web/PrivateLocalSearchUiDataService/GetLocalBoqProxy"

	// gmapsReviewsPerPage is the page size the review window asks for.
	gmapsReviewsPerPage = 10

	// gmapsSearchPB is the request descriptor the Maps web page sends with a
	// search, cut down to the fields the result list depends on: 7i20 asks for
	// 20 stores, and without 10b1 or 34m19 Google returns no store records.
	// Measured on 2026-09-13, it returns the same stores as the page's full
	// 1,643-character descriptor.
	gmapsSearchPB = "!7i20!10b1!34m19!2b1!3b1!4b1!6b1!8m6!1b1!3b1!4b1!5b1!6b1!7b1!9b1!12b1!14b1!20b1!23b1!25b1!26b1!31b1"

	// gmapsSearchResults is where a search response keeps its result list.
	// Each entry holds its store record at 1, with the feature ID at 10 and
	// the name at 11.
	gmapsSearchResults = 64

	gmapsUserAgent       = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/140.0.0.0 Safari/537.36"
	gmapsRequestTimeout  = 30 * time.Second
	gmapsMaxResponseSize = 64 << 20
)

var (
	gmapsFeatureIDRe = regexp.MustCompile(`^0x[0-9a-f]+:0x[0-9a-f]+$`)
	gmapsContribIDRe = regexp.MustCompile(`/contrib/(\d+)`)
)

// GoogleMapsStoreReview is a struct for Google Maps store reviews.
type GoogleMapsStoreReview struct {
	Reviewer            string `json:"reviewer"`
	ReviewerID          string `json:"reviewer_id"`
	ReviewerState       string `json:"reviewer_state"`
	ReviewerLevel       int    `json:"reviewer_level"`
	ReviewerReviewCount int    `json:"reviewer_review_count"` // How many reviews the reviewer has written
	ReviewerPhotoCount  int    `json:"reviewer_photo_count"`  // How many photos the reviewer has posted
	ReviewID            string `json:"review_id"`
	ReviewTime          string `json:"review_time"`
	ReviewDate          string `json:"review_date"`
	Language            string `json:"language"` // The review's language code, such as zh-Hant; empty when the review has no text
	Content             string `json:"content"`
	Rating              int    `json:"rating"`
}

// GoogleMapsStoreReviews is a slice of GoogleMapsStoreReview.
type GoogleMapsStoreReviews []GoogleMapsStoreReview

// GoogleMapsStoreReviewsFetchingOptions is a struct for options when fetching reviews.
// A zero field means its default.
type GoogleMapsStoreReviewsFetchingOptions struct {
	SortBy GoogleMapsStoreReviewSortBy
	// MaxWaitingInterval_Milliseconds is the maximum waiting interval in milliseconds between requests.
	// It must be at least 1000; zero means 5000.
	MaxWaitingInterval_Milliseconds uint
}

type GoogleMapsStoreReviewSortBy uint8

const (
	// SortByRelevance 按相關性排序
	SortByRelevance GoogleMapsStoreReviewSortBy = 1
	// SortByNewest 按最新排序
	SortByNewest GoogleMapsStoreReviewSortBy = 2
	// SortByRating 按評分排序
	SortByHighestRating GoogleMapsStoreReviewSortBy = 3
	// SortByLowestRating 按最低評分排序
	SortByLowestRating GoogleMapsStoreReviewSortBy = 4
)

type googleMapsStoreCrawler struct {
	client         *http.Client
	headers        map[string]string
	storeSearchUrl string
	storeReviewUrl string
}

type GoogleMapsStoreData struct {
	ID   string
	Name string
}

// GoogleMapsStores returns a crawler for Google Maps store data. It needs no
// network access and never returns nil.
func GoogleMapsStores() *googleMapsStoreCrawler {
	return &googleMapsStoreCrawler{
		client:         &http.Client{Timeout: gmapsRequestTimeout},
		headers:        map[string]string{"User-Agent": gmapsUserAgent},
		storeSearchUrl: gmapsSearchURL,
		storeReviewUrl: gmapsReviewURL,
	}
}

// Search searches Google Maps for stores matching storeName and returns up to
// 20 of them, in Google's order. It returns nil when the request fails or no
// store comes back, and logs a warning saying which.
func (c *googleMapsStoreCrawler) Search(storeName string) []GoogleMapsStoreData {
	params := url.Values{}
	params.Set("tbm", "map")
	params.Set("hl", "zh-TW")
	params.Set("gl", "tw")
	params.Set("q", storeName)
	params.Set("pb", gmapsSearchPB)

	body, err := c.get(c.storeSearchUrl + "?" + params.Encode())
	if err != nil {
		insyra.LogWarning("datafetch", "GoogleMapsStores.Search", "Failed to search: %v. Returning nil.", err)
		return nil
	}
	stores, err := parseGoogleMapsSearch(body)
	if err != nil {
		insyra.LogWarning("datafetch", "GoogleMapsStores.Search", "%v. Returning nil.", err)
		return nil
	}
	if len(stores) == 0 {
		insyra.LogWarning("datafetch", "GoogleMapsStores.Search", "Google returned no stores for %q. If the store exists, the response format may have changed. Returning nil.", storeName)
		return nil
	}
	return stores
}

// parseGoogleMapsSearch reads the stores out of a search response, keeping
// Google's order and dropping repeats and entries without a feature ID.
func parseGoogleMapsSearch(body []byte) ([]GoogleMapsStoreData, error) {
	var data []any
	if err := json.Unmarshal(stripGoogleJSONPrefix(body), &data); err != nil {
		return nil, fmt.Errorf("failed to decode the search response: %w", err)
	}

	results, _ := extractValue(data, gmapsSearchResults).([]any)
	var stores []GoogleMapsStoreData
	seen := make(map[string]struct{})
	for _, item := range results {
		entry, _ := item.([]any)
		id, _ := extractValue(entry, 1, 10).(string)
		if !gmapsFeatureIDRe.MatchString(id) {
			continue
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		name, _ := extractValue(entry, 1, 11).(string)
		stores = append(stores, GoogleMapsStoreData{ID: id, Name: name})
	}
	return stores, nil
}

// GetReviews fetches reviews for the store with the given ID, 10 per page.
// The pageCount parameter specifies the number of pages to fetch.
// If pageCount is 0, all reviews will be fetched.
// Returns a list of reviews.
// Returns nil if failed to fetch reviews.
//
// ReviewerState and ReviewerLevel are always empty: the review pages no longer
// carry a reviewer's status line or guide level.
func (c *googleMapsStoreCrawler) GetReviews(storeId string, pageCount int, options ...GoogleMapsStoreReviewsFetchingOptions) GoogleMapsStoreReviews {
	fetchingOptions := GoogleMapsStoreReviewsFetchingOptions{
		SortBy:                          SortByRelevance,
		MaxWaitingInterval_Milliseconds: 5000,
	}
	if len(options) == 1 {
		// A zero field keeps its default; only a value that is set and out of
		// range is worth a warning.
		if sortBy := options[0].SortBy; sortBy > SortByLowestRating {
			insyra.LogWarning("datafetch", "GoogleMapsStores.GetReviews", "SortBy is invalid. Using default value.")
		} else if sortBy != 0 {
			fetchingOptions.SortBy = sortBy
		}
		if wait := options[0].MaxWaitingInterval_Milliseconds; wait != 0 && wait < 1000 {
			insyra.LogWarning("datafetch", "GoogleMapsStores.GetReviews", "MaxWaitingInterval is too small. Using default value.")
		} else if wait != 0 {
			fetchingOptions.MaxWaitingInterval_Milliseconds = wait
		}
	} else if len(options) > 1 {
		insyra.LogWarning("datafetch", "GoogleMapsStores.GetReviews", "at most one GoogleMapsStoreReviewsFetchingOptions may be given, got %d", len(options))
		return nil
	}

	token := ""
	reviews := []GoogleMapsStoreReview{}

	for page := 1; pageCount == 0 || page <= pageCount; page++ {
		insyra.LogDebug("datafetch", "GoogleMapsStores.GetReviews", "fetching reviews on page %d", page)

		body, err := c.get(c.storeReviewUrl + "?" + gmapsReviewQuery(storeId, fetchingOptions.SortBy, token))
		if err != nil {
			insyra.LogWarning("datafetch", "GoogleMapsStores.GetReviews", "Failed to fetch reviews: %v. Returning nil.", err)
			return nil
		}
		pageReviews, next, err := parseGoogleMapsReviewPage(body)
		if err != nil {
			insyra.LogWarning("datafetch", "GoogleMapsStores.GetReviews", "%v. Returning nil.", err)
			return nil
		}
		reviews = append(reviews, pageReviews...)
		token = next

		// 若無下一頁，結束迴圈
		if token == "" || page == pageCount {
			break
		}

		// 隨機等待 1 秒到 MaxWaitingInterval，防止被 Google 封鎖。上限剛好是 1000
		// 時只有一種等待時間，rand.IntN 的參數仍須大於零。
		maxWait := int(fetchingOptions.MaxWaitingInterval_Milliseconds)
		waitTime := 1000 + rand.IntN(maxWait-1000+1)
		insyra.LogDebug("datafetch", "GoogleMapsStores.GetReviews", "waiting %.1fs before fetching the next page", float64(waitTime)/1000)
		time.Sleep(time.Duration(waitTime) * time.Millisecond)
	}

	return reviews
}

// gmapsReviewQuery builds the query for one page of reviews. reqpld is the
// argument list the review window sends: the sort order at 1, the page size
// at 9, the store at 11 and the page token at 19 vary, and the other values
// are what the window sends.
func gmapsReviewQuery(storeID string, sortBy GoogleMapsStoreReviewSortBy, token string) string {
	args := make([]any, 24)
	args[1] = int(sortBy)
	args[9] = gmapsReviewsPerPage
	args[11] = []any{storeID}
	args[16] = []any{1, 1, nil, []any{[]any{3}, []any{4}, []any{5}, []any{6}, []any{7}}}
	args[19] = token
	args[23] = 0
	request := make([]any, 10)
	request[9] = args
	payload, _ := json.Marshal([]any{nil, request})

	params := url.Values{}
	params.Set("hl", "zh-TW")
	params.Set("reqpld", string(payload))
	params.Set("msc", "gwsrpc")
	params.Set("opi", "89978449")
	return params.Encode()
}

// parseGoogleMapsReviewPage reads one page of reviews and the token for the
// next page, which is empty on the last page. A page keeps its reviews at
// [1][10][2] and the token at [1][10][6].
func parseGoogleMapsReviewPage(body []byte) ([]GoogleMapsStoreReview, string, error) {
	var data []any
	if err := json.Unmarshal(stripGoogleJSONPrefix(body), &data); err != nil {
		return nil, "", fmt.Errorf("failed to decode the review page: %w", err)
	}
	block, ok := extractValue(data, 1, 10).([]any)
	if !ok {
		return nil, "", fmt.Errorf("the review page has no review block; the response format may have changed")
	}

	records, _ := extractValue(block, 2).([]any)
	reviews := make([]GoogleMapsStoreReview, 0, len(records))
	for _, item := range records {
		record, ok := item.([]any)
		if !ok {
			continue
		}
		reviews = append(reviews, GoogleMapsStoreReview{
			Reviewer:            extractString(record, 3, 0),
			ReviewerID:          gmapsContribID(extractString(record, 3, 2)),
			ReviewerReviewCount: extractInt(record, 3, 3),
			ReviewerPhotoCount:  extractInt(record, 3, 4),
			ReviewID:            extractString(record, 5),
			ReviewTime:          extractString(record, 2, 0),
			ReviewDate:          gmapsReviewDate(extractString(record, 2, 2)),
			Language:            extractString(record, 26),
			Content:             html.UnescapeString(strings.ReplaceAll(extractString(record, 27), "<br>", "\n")),
			Rating:              extractInt(record, 1),
		})
	}
	next, _ := extractValue(block, 6).(string)
	return reviews, next, nil
}

// gmapsContribID reads the reviewer's ID out of their contributions link.
func gmapsContribID(contribURL string) string {
	if m := gmapsContribIDRe.FindStringSubmatch(contribURL); m != nil {
		return m[1]
	}
	return ""
}

// gmapsReviewDate turns a review's millisecond timestamp into its UTC date.
func gmapsReviewDate(millis string) string {
	ms, err := strconv.ParseInt(millis, 10, 64)
	if err != nil {
		return ""
	}
	return time.UnixMilli(ms).UTC().Format("2006-01-02")
}

// ToDataTable converts the reviews to a DataTable.
func (reviews GoogleMapsStoreReviews) ToDataTable() *insyra.DataTable {
	dt := insyra.NewDataTable()
	for _, review := range reviews {
		dt.AppendRowsByColName(
			map[string]any{
				"Reviewer":            review.Reviewer,
				"ReviewerID":          review.ReviewerID,
				"ReviewerState":       review.ReviewerState,
				"ReviewerLevel":       review.ReviewerLevel,
				"ReviewerReviewCount": review.ReviewerReviewCount,
				"ReviewerPhotoCount":  review.ReviewerPhotoCount,
				"ReviewID":            review.ReviewID,
				"ReviewTime":          review.ReviewTime,
				"ReviewDate":          review.ReviewDate,
				"Language":            review.Language,
				"Content":             review.Content,
				"Rating":              review.Rating,
			},
		)
	}

	return dt
}

// get sends a GET request with the crawler's headers and returns the body of
// a 200 response, read up to gmapsMaxResponseSize.
func (c *googleMapsStoreCrawler) get(rawURL string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}
	res, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP status %d", res.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(res.Body, gmapsMaxResponseSize))
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	return body, nil
}

// stripGoogleJSONPrefix removes the )]}' line Google puts in front of its JSON
// responses.
func stripGoogleJSONPrefix(body []byte) []byte {
	return bytes.TrimPrefix(body, []byte(")]}'"))
}

// extractString 從 JSON 層級結構中擷取字串
func extractString(data []any, indices ...int) string {
	val := extractValue(data, indices...)
	if str, ok := val.(string); ok {
		return str
	}
	if num, ok := val.(float64); ok {
		return fmt.Sprintf("%.0f", num) // 轉換為整數格式的字串
	}
	return ""
}

// extractInt 從 JSON 層級結構中擷取整數
func extractInt(data []any, indices ...int) int {
	val := extractValue(data, indices...)
	if num, ok := val.(float64); ok {
		return int(num)
	}
	return 0
}

// extractValue 用於遍歷 JSON 層級結構
func extractValue(data []any, indices ...int) any {
	current := any(data)
	for _, idx := range indices {
		arr, ok := current.([]any)
		if !ok {
			return nil
		}

		// **支援 `.at(-1)`**
		if idx < 0 {
			idx = len(arr) + idx
		}

		// **防止索引超界**
		if idx < 0 || idx >= len(arr) {
			return nil
		}

		current = arr[idx]
	}
	return current
}
