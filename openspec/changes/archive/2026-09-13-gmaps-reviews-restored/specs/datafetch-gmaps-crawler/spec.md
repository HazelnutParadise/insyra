## ADDED Requirements

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
