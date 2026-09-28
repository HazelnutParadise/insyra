## MODIFIED Requirements

### Requirement: Review fetching options and progress

`GetReviews` SHALL 把 `SortBy`、`MaxWaitingInterval` 或已 Deprecated 的 `MaxWaitingInterval_Milliseconds` 的零值視為預設值且不記錄警告，等待上限的預設為 5 秒。`MaxWaitingInterval` 或 `MaxWaitingInterval_Milliseconds` 設定為小於一秒的非零值時，SHALL 記錄警告並改用預設值。兩個等待欄位都設定時，SHALL 記錄警告並在發出任何請求前回傳 nil。每兩頁之間的等待 SHALL 是一秒到等待上限之間的隨機時間，等待上限剛好為一秒時 SHALL 正常翻頁而不 panic，且 SHALL NOT 把進度印到標準輸出。

#### Scenario: The minimum waiting interval
- **WHEN** 以 `MaxWaitingInterval_Milliseconds: 1000` 抓取兩頁
- **THEN** 兩頁都取得，兩頁之間至少等待一秒，不 panic

#### Scenario: The minimum waiting interval as a duration
- **WHEN** 以 `MaxWaitingInterval: time.Second` 抓取兩頁
- **THEN** 兩頁都取得，兩頁之間至少等待一秒，沒有警告

#### Scenario: Setting only one option
- **WHEN** 只設定 `SortBy`，或只設定 `MaxWaitingInterval`，或只設定 `MaxWaitingInterval_Milliseconds`
- **THEN** 沒有警告，未設定的欄位使用預設值

#### Scenario: Both waiting fields set
- **WHEN** 同時設定 `MaxWaitingInterval: 2 * time.Second` 與 `MaxWaitingInterval_Milliseconds: 2000`
- **THEN** 回傳 nil，記錄警告，且沒有發出請求

#### Scenario: A waiting interval under one second
- **WHEN** 以 `MaxWaitingInterval: 500 * time.Millisecond` 抓取一頁
- **THEN** 記錄等待上限太小的警告，仍取得那一頁

## ADDED Requirements

### Requirement: Review sort orders carry their type's name

The review sort orders SHALL be `GoogleMapsStoreReviewSortByRelevance` (1), `GoogleMapsStoreReviewSortByNewest` (2), `GoogleMapsStoreReviewSortByHighestRating` (3) and `GoogleMapsStoreReviewSortByLowestRating` (4). `SortByRelevance`, `SortByNewest`, `SortByHighestRating` and `SortByLowestRating` SHALL remain for one release as Deprecated constants with the same values, each doc comment naming its replacement.

#### Scenario: The old names keep their values
- **WHEN** 比較四個舊常數與對應的新常數
- **THEN** 數值相同，舊常數的 doc comment 有指名新常數的 `Deprecated:` 段落
