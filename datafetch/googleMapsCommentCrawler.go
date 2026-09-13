// Google Maps store search and reviews.
//
// Both read endpoints the Google Maps web page calls, which Google changes
// without notice. Search was restored on 2026-09-13. The review endpoint has
// answered HTTP 403 since at least that date, so GetReviews currently fails
// (#249).

package datafetch

import (
	"bytes"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/HazelnutParadise/insyra"
	json "github.com/goccy/go-json"
)

const (
	gmapsSearchURL = "https://www.google.com.tw/search"
	gmapsReviewURL = "https://www.google.com.tw/maps/rpc/listugcposts"

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

var gmapsFeatureIDRe = regexp.MustCompile(`^0x[0-9a-f]+:0x[0-9a-f]+$`)

// GoogleMapsStoreReview is a struct for Google Maps store reviews.
type GoogleMapsStoreReview struct {
	Reviewer      string `json:"reviewer"`
	ReviewerID    string `json:"reviewer_id"`
	ReviewerState string `json:"reviewer_state"`
	ReviewerLevel int    `json:"reviewer_level"`
	ReviewTime    string `json:"review_time"`
	ReviewDate    string `json:"review_date"`
	Content       string `json:"content"`
	Rating        int    `json:"rating"`
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

// GetReviews fetches reviews for the store with the given ID.
// The pageCount parameter specifies the number of pages to fetch.
// If pageCount is 0, all reviews will be fetched.
// Returns a list of reviews.
// Returns nil if failed to fetch reviews.
//
// Google currently answers this request with HTTP 403, so GetReviews returns
// nil with a warning (#249).
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
		insyra.LogWarning("datafetch", "GoogleMapsStores.GetReviews", "Got too many options. Using default options.")
	}

	nextToken := ""
	reviews := []GoogleMapsStoreReview{}

	for page := 1; pageCount == 0 || page <= pageCount; page++ {
		insyra.LogDebug("datafetch", "GoogleMapsStores.GetReviews", "fetching reviews on page %d", page)

		// 組合請求參數
		params := url.Values{}
		params.Set("authuser", "0")
		params.Set("hl", "zh-TW")
		params.Set("gl", "tw")
		params.Set("pb", fmt.Sprintf("!1m6!1s%s!6m4!4m1!1e1!4m1!1e3!2m2!1i10!2s%s!5m2!1s0OBwZ4OnGsrM1e8PxIjW6AI!7e81!8m5!1b1!2b1!3b1!5b1!7b1!11m0!13m1!1e%d",
			storeId, nextToken, fetchingOptions.SortBy))

		body, err := c.get(c.storeReviewUrl + "?" + params.Encode())
		if err != nil {
			insyra.LogWarning("datafetch", "GoogleMapsStores.GetReviews", "Failed to fetch reviews: %v. Returning nil.", err)
			return nil
		}
		jsonData := []any{}
		if err := json.Unmarshal(stripGoogleJSONPrefix(body), &jsonData); err != nil {
			insyra.LogWarning("datafetch", "GoogleMapsStores.GetReviews", "Failed to decode JSON. Error: %v. Returning nil.", err)
			return nil
		}

		// 解析 `nextToken`
		if len(jsonData) > 1 {
			nextToken, _ = jsonData[1].(string)
		}

		// 解析評論數據
		if len(jsonData) > 2 {
			rawReviews, ok := jsonData[2].([]any)
			if ok {
				for _, item := range rawReviews {
					reviewData, _ := item.([]any)
					if len(reviewData) < 3 {
						continue
					}

					reviewer := extractString(reviewData, 0, 1, 4, 5, 0)
					reviewerID := extractString(reviewData, 0, 0)
					reviewerState := extractString(reviewData, 0, 1, 4, 5, 10, 0)
					reviewerLevel := extractInt(reviewData, 0, 1, 4, 5, 9)
					reviewTime := extractString(reviewData, 0, 1, 6)
					reviewDate := strings.Join([]string{
						strings.Repeat("0", 4-len(extractString(reviewData, 0, 2, 2, 0, 1, 21, 6, -1, 0))) + extractString(reviewData, 0, 2, 2, 0, 1, 21, 6, -1, 0),
						strings.Repeat("0", 2-len(extractString(reviewData, 0, 2, 2, 0, 1, 21, 6, -1, 1))) + extractString(reviewData, 0, 2, 2, 0, 1, 21, 6, -1, 1),
						strings.Repeat("0", 2-len(extractString(reviewData, 0, 2, 2, 0, 1, 21, 6, -1, 2))) + extractString(reviewData, 0, 2, 2, 0, 1, 21, 6, -1, 2),
					}, "-")
					content := extractString(reviewData, 0, 2, -1, 0, 0)
					rating := extractInt(reviewData, 0, 2, 0, 0)

					reviews = append(reviews, GoogleMapsStoreReview{
						Reviewer:      reviewer,
						ReviewerID:    reviewerID,
						ReviewerState: reviewerState,
						ReviewerLevel: reviewerLevel,
						ReviewTime:    reviewTime,
						ReviewDate:    reviewDate,
						Content:       content,
						Rating:        rating,
					})
				}
			}
		}

		// 若無下一頁，結束迴圈
		if nextToken == "" || page == pageCount {
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

// ToDataTable converts the reviews to a DataTable.
func (reviews GoogleMapsStoreReviews) ToDataTable() *insyra.DataTable {
	dt := insyra.NewDataTable()
	for _, review := range reviews {
		dt.AppendRowsByColName(
			map[string]any{
				"Reviewer":      review.Reviewer,
				"ReviewerID":    review.ReviewerID,
				"ReviewerState": review.ReviewerState,
				"ReviewerLevel": review.ReviewerLevel,
				"ReviewTime":    review.ReviewTime,
				"ReviewDate":    review.ReviewDate,
				"Content":       review.Content,
				"Rating":        review.Rating,
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
