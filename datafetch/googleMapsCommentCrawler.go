package datafetch

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/HazelnutParadise/insyra"
	jsoniter "github.com/json-iterator/go"
	"golang.org/x/exp/rand"
)

// GoogleMapsStoreComment is a struct for Google Maps store comments.
type GoogleMapsStoreComment struct {
	Reviewer      string    `json:"reviewer"`
	ReviewerID    string    `json:"reviewer_id"`
	ReviewerState string    `json:"reviewer_state"`
	ReviewerLevel int       `json:"reviewer_level"`
	CommentTime   string    `json:"comment_time"`
	CommentDate   time.Time `json:"comment_date"`
	Content       string    `json:"content"`
	Rating        int       `json:"rating"`
}

type googleMapsStoreComments []GoogleMapsStoreComment

type GoogleMapsStoreCommentsFetchingOptions struct {
	SortBy             GoogleMapsStoreCommentSortBy
	MaxWaitingInterval uint
}

type GoogleMapsStoreCommentSortBy uint8

const (
	// SortByRelevance 按相關性排序
	SortByRelevance GoogleMapsStoreCommentSortBy = 1
	// SortByNewest 按最新排序
	SortByNewest GoogleMapsStoreCommentSortBy = 2
	// SortByRating 按評分排序
	SortByHighestRating GoogleMapsStoreCommentSortBy = 3
	// SortByLowestRating 按最低評分排序
	SortByLowestRating GoogleMapsStoreCommentSortBy = 4
)

var json = jsoniter.ConfigFastest

type googleMapsStoreCrawler struct {
	headers         map[string]string
	storeNameUrl    string
	storeSearchUrl  string
	storeCommentUrl string
}

type storeData struct {
	ID   string
	Name string
}

// GoogleMapsStores returns a crawler for Google Maps store data.
// Returns nil if failed to initialize.
func GoogleMapsStores() *googleMapsStoreCrawler {
	const configUrl = "https://raw.githubusercontent.com/TimLai666/google-maps-store-comment-crawler/main/crawler_config.json"
	res, err := http.Get(configUrl)
	if err != nil || res.StatusCode != 200 {
		insyra.LogWarning("datafetch.GoogleMapsStores: Failed to fetch GoogleMapsStoreCommentCrawler config. Error: %v. Returning nil.", err)
		return nil
	}
	defer res.Body.Close()

	config := struct {
		Headers         map[string]string `json:"headers"`
		StoreNameUrl    string            `json:"storeNameUrl"`
		StoreSearchUrl  string            `json:"storeSearchUrl"`
		StoreCommentUrl string            `json:"commentUrl"`
	}{}
	err = json.NewDecoder(res.Body).Decode(&config)
	if err != nil {
		insyra.LogWarning("datafetch.GoogleMapsStores: Failed to decode GoogleMapsStoreCommentCrawler config. Error: %v. Returning nil.", err)
		return nil
	}

	return &googleMapsStoreCrawler{
		headers:         config.Headers,
		storeNameUrl:    config.StoreNameUrl,
		storeSearchUrl:  config.StoreSearchUrl,
		storeCommentUrl: config.StoreCommentUrl,
	}
}

// Search searches for stores with the given name.
// Returns a list of store data.
// Returns nil if failed to search.
func (c *googleMapsStoreCrawler) Search(storeName string) []storeData {
	url := strings.Replace(c.storeSearchUrl, "{store_name}", storeName, 1)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		insyra.LogWarning("datafetch.GoogleMapsStores().Search: Failed to create request. Error: %v. Returning nil.", err)
		return nil
	}
	for k, v := range c.headers {
		req.Header.Add(k, v)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		insyra.LogWarning("datafetch.GoogleMapsStores().Search: Failed to send request. Error: %v. Returning nil.", err)
		return nil
	}
	defer res.Body.Close()

	resTxt, err := io.ReadAll(res.Body)
	if err != nil {
		insyra.LogWarning("datafetch.GoogleMapsStores().Search: Failed to read response. Error: %v. Returning nil.", err)
		return nil
	}

	// 定義正則表達式
	pattern := regexp.MustCompile(`0x.{16}:0x.{16}`)

	// 取得匹配的 storeId（使用 map 來去重）
	storeIdSet := make(map[string]struct{})
	matches := pattern.FindAllString(string(resTxt), -1)
	for _, match := range matches {
		cleanedId := strings.ReplaceAll(match, "\\", "")
		storeIdSet[cleanedId] = struct{}{}
	}

	// 轉換為 slice
	var storeIdList []string
	for storeId := range storeIdSet {
		storeIdList = append(storeIdList, storeId)
	}

	// 同步處理：逐個獲取商店名稱
	var storeList []storeData
	for _, storeId := range storeIdList {
		storeName, err := c.getStoreName(storeId)
		if err != nil {
			insyra.LogWarning("datafetch.GoogleMapsStores().Search: Error fetching store name for %s: %v\n", storeId, err)
			continue // 碰到錯誤就跳過
		}
		storeList = append(storeList, storeData{
			ID:   storeId,
			Name: storeName,
		})
	}

	return storeList
}

// GetComments fetches comments for the store with the given ID.
// The pageCount parameter specifies the number of pages to fetch.
// If pageCount is 0, all comments will be fetched.
// Returns a list of comments.
// Returns nil if failed to fetch comments.
func (c *googleMapsStoreCrawler) GetComments(storeId string, pageCount int, options ...GoogleMapsStoreCommentsFetchingOptions) googleMapsStoreComments {
	fetchingOptions := GoogleMapsStoreCommentsFetchingOptions{
		SortBy:             SortByRelevance,
		MaxWaitingInterval: 5000,
	}
	if len(options) == 1 {
		fetchingOptions = options[0]
	} else if len(options) > 1 {
		insyra.LogWarning("datafetch.GoogleMapsStores().GetComments: Got too many options. Using default options.")
	}

	commentUrl := c.storeCommentUrl
	headers := c.headers

	nextToken := ""
	comments := []GoogleMapsStoreComment{}
	page := 1

	for pageCount == 0 || page <= pageCount {
		fmt.Printf("fetching comments on page %d...\n", page)

		// 組合請求參數
		params := url.Values{}
		params.Set("authuser", "0")
		params.Set("hl", "zh-TW")
		params.Set("gl", "tw")
		params.Set("pb", fmt.Sprintf("!1m6!1s%s!6m4!4m1!1e1!4m1!1e3!2m2!1i10!2s%s!5m2!1s0OBwZ4OnGsrM1e8PxIjW6AI!7e81!8m5!1b1!2b1!3b1!5b1!7b1!11m0!13m1!1e%d",
			storeId, nextToken, fetchingOptions.SortBy))

		// 建立 HTTP 請求
		req, err := http.NewRequest("GET", commentUrl+"?"+params.Encode(), nil)
		if err != nil {
			insyra.LogWarning("datafetch.GoogleMapsStores().GetComments: Failed to create request. Error: %v. Returning nil.", err)
			return nil
		}
		for key, value := range headers {
			req.Header.Set(key, value)
		}

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			insyra.LogWarning("datafetch.GoogleMapsStores().GetComments: Failed to send request. Error: %v. Returning nil.", err)
			return nil
		}
		defer resp.Body.Close()

		// 確保回應狀態碼為 200 OK
		if resp.StatusCode != http.StatusOK {
			insyra.LogWarning("datafetch.GoogleMapsStores().GetComments: Failed to fetch comments. HTTP status code: %d. Returning nil.", resp.StatusCode)
			return nil
		}

		// 讀取回應內容
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			insyra.LogWarning("datafetch.GoogleMapsStores().GetComments: Failed to read response. Error: %v. Returning nil.", err)
			return nil
		}

		// Google 回應有 `)]}'` 前綴，需去除前 4 個字元
		jsonData := []interface{}{}
		if err := json.Unmarshal(body[4:], &jsonData); err != nil {
			insyra.LogWarning("datafetch.GoogleMapsStores().GetComments: Failed to decode JSON. Error: %v. Returning nil.", err)
			return nil
		}

		// 解析 `nextToken`
		if len(jsonData) > 1 {
			nextToken, _ = jsonData[1].(string)
		}

		// 解析評論數據
		if len(jsonData) > 2 {
			rawComments, ok := jsonData[2].([]interface{})
			if ok {
				for _, item := range rawComments {
					commentData, _ := item.([]interface{})
					if len(commentData) < 3 {
						continue
					}

					reviewer := extractString(commentData, 0, 1, 4, 5, 0)
					reviewerID := extractString(commentData, 0, 0)
					reviewerState := extractString(commentData, 0, 1, 4, 5, 10, 0)
					reviewerLevel := extractInt(commentData, 0, 1, 4, 5, 9)
					commentTime := extractString(commentData, 0, 1, 6)
					commentDate := strings.Join([]string{
						extractString(commentData, 0, 2, 2, 0, 1, 21, 6, -1, 0),
						strings.Repeat("0", 2-len(extractString(commentData, 0, 2, 2, 0, 1, 21, 6, -1, 1))) + extractString(commentData, 0, 2, 2, 0, 1, 21, 6, -1, 1),
						strings.Repeat("0", 2-len(extractString(commentData, 0, 2, 2, 0, 1, 21, 6, -1, 2))) + extractString(commentData, 0, 2, 2, 0, 1, 21, 6, -1, 2),
					}, "-")
					content := extractString(commentData, 0, 2, -1, 0, 0)
					rating := extractInt(commentData, 0, 2, 0, 0)

					commentDateObj, err := time.Parse("2006-01-02", commentDate)
					if err != nil {
						insyra.LogWarning("datafetch.GoogleMapsStores().GetComments: Failed to parse comment date. Error: %v. Returning nil.", err)
						return nil
					}

					comments = append(comments, GoogleMapsStoreComment{
						Reviewer:      reviewer,
						ReviewerID:    reviewerID,
						ReviewerState: reviewerState,
						ReviewerLevel: reviewerLevel,
						CommentTime:   commentTime,
						CommentDate:   commentDateObj,
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

		// 隨機等待時間，防止被 Google 封鎖
		waitTime := rand.Intn(int(fetchingOptions.MaxWaitingInterval)-1000) + 1000
		fmt.Printf("Waiting %.1fs before fetching the next page...\n", float64(waitTime)/1000)
		time.Sleep(time.Duration(waitTime) * time.Millisecond)

		page++
	}

	return comments
}

// ToDataTable converts the comments to a DataTable.
func (comments googleMapsStoreComments) ToDataTable() *insyra.DataTable {
	dt := insyra.NewDataTable()
	for _, comment := range comments {
		dt.AppendRowsByColName(
			map[string]interface{}{
				"Reviewer":      comment.Reviewer,
				"ReviewerID":    comment.ReviewerID,
				"ReviewerState": comment.ReviewerState,
				"ReviewerLevel": comment.ReviewerLevel,
				"CommentTime":   comment.CommentTime,
				"CommentDate":   comment.CommentDate,
				"Content":       comment.Content,
				"Rating":        comment.Rating,
			},
		)
	}

	return dt
}

func (c *googleMapsStoreCrawler) getStoreName(storeId string) (string, error) {
	url := strings.Replace(c.storeNameUrl, "{store_id}", storeId, 1)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	for k, v := range c.headers {
		req.Header.Add(k, v)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("cannot get store data, HTTP status code: %d", res.StatusCode)
	}

	// 讀取 HTML 內容
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}
	html := string(body)

	// 使用正則表達式匹配 <meta itemprop="name">
	metaTagPattern := regexp.MustCompile(`<meta[^>]*itemprop=["']name["'][^>]*>`)
	metaTags := metaTagPattern.FindAllString(html, -1)
	if len(metaTags) == 0 {
		return "", fmt.Errorf("cannot get store data")
	}

	// 從 meta 標籤提取名稱
	name := ""
	namePattern := regexp.MustCompile(`".*·`)
	for _, tag := range metaTags {
		match := namePattern.FindString(tag)
		if match != "" {
			name = match[1 : len(match)-2] // 去掉首尾多餘的字元
			break
		}
	}

	if name == "" {
		return "", fmt.Errorf("cannot get store data")
	}

	return name, nil
}

// extractString 從 JSON 層級結構中擷取字串
func extractString(data []interface{}, indices ...int) string {
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
func extractInt(data []interface{}, indices ...int) int {
	val := extractValue(data, indices...)
	if num, ok := val.(float64); ok {
		return int(num)
	}
	return 0
}

// extractValue 用於遍歷 JSON 層級結構
func extractValue(data []interface{}, indices ...int) interface{} {
	current := interface{}(data)
	for _, idx := range indices {
		arr, ok := current.([]interface{})
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
