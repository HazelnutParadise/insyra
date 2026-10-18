## REMOVED Requirements

### Requirement: Review fetching options and progress
**Reason**: It named deprecated APIs that were removed in 0.4, and its scenarios cannot be edited in place; it is restated under a new name in this change.
**Migration**: See the requirement "Review fetching waits within bounds".

### Requirement: Review sort orders carry their type's name
**Reason**: It named deprecated APIs that were removed in 0.4, and its scenarios cannot be edited in place; it is restated under a new name in this change.
**Migration**: See the requirement "Review sort orders are numbered one to four".

## ADDED Requirements

### Requirement: Review fetching waits within bounds

`GetReviews` SHALL 把 `SortBy` 或 `MaxWaitingInterval` 的零值視為預設值且不記錄警告，等待上限的預設為 5 秒。`MaxWaitingInterval` 設定為小於一秒的非零值時，SHALL 記錄警告並改用預設值。每兩頁之間的等待 SHALL 是一秒到等待上限之間的隨機時間，等待上限剛好為一秒時 SHALL 正常翻頁而不 panic，且 SHALL NOT 把進度印到標準輸出。

#### Scenario: The minimum waiting interval as a duration
- **WHEN** 以 `MaxWaitingInterval: time.Second` 抓取兩頁
- **THEN** 兩頁都取得，兩頁之間至少等待一秒，沒有警告，不 panic

#### Scenario: Setting only one option
- **WHEN** 只設定 `SortBy`，或只設定 `MaxWaitingInterval`
- **THEN** 沒有警告，未設定的欄位使用預設值

#### Scenario: A waiting interval under one second
- **WHEN** 以 `MaxWaitingInterval: 500 * time.Millisecond` 抓取一頁
- **THEN** 記錄等待上限太小的警告，仍取得那一頁

### Requirement: Review sort orders are numbered one to four

The review sort orders SHALL be `GoogleMapsStoreReviewSortByRelevance` (1), `GoogleMapsStoreReviewSortByNewest` (2), `GoogleMapsStoreReviewSortByHighestRating` (3) and `GoogleMapsStoreReviewSortByLowestRating` (4). The unprefixed `SortByRelevance`, `SortByNewest`, `SortByHighestRating` and `SortByLowestRating` SHALL NOT exist.

#### Scenario: The sort order values
- **WHEN** 讀取四個 `GoogleMapsStoreReviewSortBy…` 常數的值
- **THEN** 依序為 1、2、3、4
