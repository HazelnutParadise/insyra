## ADDED Requirements

### Requirement: GetReviews reports the review ID, language and reviewer counts

`GetReviews` SHALL 在每筆評論填入評論 ID、評論的語言代碼、評論者寫過的評論數與上傳過的相片數。`ToDataTable` SHALL 為這四個欄位各產生一欄，欄名與欄位名稱相同。

#### Scenario: Reading the extra fields
- **WHEN** 一筆評論的 ID 為 `Ci9D…`、語言為 `zh-Hant`、評論者寫過 18 則評論並上傳 173 張相片
- **THEN** `ReviewID`、`Language`、`ReviewerReviewCount`、`ReviewerPhotoCount` 分別為這些值

#### Scenario: A review with only a star rating
- **WHEN** 一筆評論只有星等，沒有內文
- **THEN** `Language` 與 `Content` 都為空，其餘欄位照常填入

#### Scenario: Converting to a table
- **WHEN** 把含這四個欄位的評論轉成 DataTable
- **THEN** 表格有 `ReviewID`、`Language`、`ReviewerReviewCount`、`ReviewerPhotoCount` 四欄，值與欄位相同
