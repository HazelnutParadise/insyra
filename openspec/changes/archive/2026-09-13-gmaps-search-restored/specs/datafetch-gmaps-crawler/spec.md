## ADDED Requirements

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
