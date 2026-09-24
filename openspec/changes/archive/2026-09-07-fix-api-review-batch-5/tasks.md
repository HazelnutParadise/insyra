# Tasks: fix-api-review-batch-5

## 1. Tests first

- [x] 1.1 `internal/algorithms/batch5_test.go`：CompareAny 大整數與混合有號／無號；Hermite 三種驗證
- [x] 1.2 `internal/utils/batch5_test.go`：常見版面可解析、非日期不誤判
- [x] 1.3 `stats/internal/clustering/batch5_test.go`：重複列 50 個 seed 全過；無重複時結果不變
- [x] 1.4 `batch5_test.go`：Sort／Rank 大整數、Close 不丟操作、ShowTypes 欄序、ShowRange 負數 end

## 2. Implementation

- [x] 2.1 `internal/algorithms/sort.go`：`compareIntegers` 與 `asInt64`／`asUint64`
- [x] 2.2 `datalist.go`：`Rank` 以原始儲存格排序與判定並列
- [x] 2.3 `internal/algorithms/interpolation.go`：標準 Hermite 基底
- [x] 2.4 `stats/internal/clustering/cluster.go`：`redrawIfDuplicate`，僅單次啟動時生效
- [x] 2.5 `internal/utils/utils.go`：`TryParseTime` 版面清單（長的優先）
- [x] 2.6 `show.go`：`sortColIndices`；`ShowRange` doc comment
- [x] 2.7 `internal/core/atomic.go`：鎖後不再因 closed 跳過回呼

## 3. Docs, changelog, review ledger

- [x] 3.1 `Docs/DataList.md`（ShowRange 範圍規則、Hermite）、`Docs/stats.md`（KMeans 初始中心）
- [x] 3.2 `CHANGELOG.md` 與 `CHANGELOG_TW.md`：Core、`stats`
- [x] 3.3 `api-review.md`：IN-2、IN-3、IN-4、IN-5、IN-6、IN-8、IN-9 標已修正；關閉 #331–#336

## 4. Verification

- [x] 4.1 `go test ./...` 全綠；`go test -race` core／internal／clustering；`golangci-lint run` 0 issues
- [x] 4.2 `openspec validate fix-api-review-batch-5 --strict` 通過
