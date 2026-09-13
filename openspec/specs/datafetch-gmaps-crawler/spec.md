# datafetch-gmaps-crawler Specification

## Purpose
定義 `datafetch` 的 Google Maps 爬蟲送出什麼請求、怎麼讀回應、失敗時怎麼回報：建立爬蟲不下載遠端設定，搜尋一次取得店家 ID 與名稱，評論以翻頁代碼每頁 10 則取得並對應到欄位，請求有逾時，抓評論的選項零值即預設且不會 panic。
## Requirements
### Requirement: Search reads the Maps result list

`Search` SHALL 以一次請求取得 Google 地圖搜尋結果清單，回傳店家的 feature ID 與名稱，SHALL 保留 Google 的順序，SHALL 略過重複的店家與沒有 feature ID 的項目。沒有取得任何店家時 SHALL 回傳 nil 並記錄警告，說明回應格式可能已改變。請求失敗時 SHALL 回傳 nil 並在警告中寫出原因。

#### Scenario: A response with stores
- **WHEN** 搜尋回應的結果清單有兩家店、一筆重複、一筆不是店家資料、一筆 ID 格式不對
- **THEN** 回傳那兩家店，順序與回應相同

#### Scenario: A response with no stores
- **WHEN** 搜尋回應沒有結果清單
- **THEN** 回傳 nil，並記錄指出沒有店家的警告

#### Scenario: A failed request
- **WHEN** 伺服器回應 HTTP 500
- **THEN** 回傳 nil，警告寫出狀態碼

### Requirement: The crawler needs no remote configuration

`GoogleMapsStores()` SHALL NOT 在執行期下載設定，SHALL NOT 回傳 nil。爬蟲送出的每個請求 SHALL 有逾時，且讀取回應 SHALL 有大小上限。

#### Scenario: Creating a crawler offline
- **WHEN** 呼叫 `GoogleMapsStores()`
- **THEN** 不發出任何請求，回傳非 nil 的爬蟲，其請求有逾時

### Requirement: Review fetching options and progress

`GetReviews` SHALL 把 `SortBy` 或 `MaxWaitingInterval_Milliseconds` 的零值視為預設值且不記錄警告，SHALL 在等待上限剛好為 1000 毫秒時正常翻頁而不 panic，且 SHALL NOT 把進度印到標準輸出。

#### Scenario: The minimum waiting interval
- **WHEN** 以 `MaxWaitingInterval_Milliseconds: 1000` 抓取兩頁
- **THEN** 兩頁都取得，兩頁之間至少等待一秒，不 panic

#### Scenario: Setting only one option
- **WHEN** 只設定 `SortBy`，或只設定 `MaxWaitingInterval_Milliseconds`
- **THEN** 沒有警告，未設定的欄位使用預設值

### Requirement: GetReviews walks the review pages with a page token

`GetReviews` SHALL 每次請求 10 則指定排序的評論，第一頁不帶翻頁代碼，之後每頁 SHALL 帶上一頁回傳的代碼。它 SHALL 在取得 `pageCount` 頁後停止，或在某頁沒有下一頁代碼時停止；`pageCount` 為 0 時 SHALL 取得所有頁。`SortBy` 的 1 到 4 SHALL 原樣作為請求的排序值。

#### Scenario: Two pages until the token runs out
- **WHEN** 以 `pageCount` 0、`SortByNewest` 抓取，第一頁回傳下一頁代碼、第二頁沒有
- **THEN** 送出兩個請求，排序值都是 2、頁面大小都是 10，第一個代碼為空、第二個是第一頁回傳的代碼，並回傳兩頁的所有評論

#### Scenario: Stopping at pageCount
- **WHEN** 以 `pageCount` 2 抓取，而第二頁仍回傳下一頁代碼
- **THEN** 只送出兩個請求

### Requirement: GetReviews maps a review record to its fields

`GetReviews` SHALL 從每筆評論讀出評論者名稱、評論者個人頁連結中的數字 ID、相對時間、星等與內文，SHALL 把時間戳換成 UTC 的 `YYYY-MM-DD` 作為 `ReviewDate`，SHALL 把內文的 `<br>` 換成換行並解碼 HTML 實體。回應缺少評論區塊時 SHALL 回傳 nil 並記錄警告，說明格式可能已改變；評論清單為空時 SHALL NOT 視為錯誤。

#### Scenario: Reading a record
- **WHEN** 一筆評論的星等為 5、相對時間為「2 個月前」、時間戳為 2026-07-05 12:00 UTC、內文為 `第一行<br>第二行 &amp; 更多`
- **THEN** `Rating` 為 5、`ReviewTime` 為「2 個月前」、`ReviewDate` 為 `2026-07-05`、`Content` 為兩行且 `&amp;` 解碼為 `&`，`ReviewerID` 為個人頁連結中的數字

#### Scenario: A response with no review block
- **WHEN** 回應是錯誤陣列而不是評論頁
- **THEN** 回傳 nil，並記錄提到 `GetReviews` 與格式的警告

#### Scenario: A page with no reviews
- **WHEN** 評論清單為空
- **THEN** 不記錄警告

