# Tasks: chisq-labels-follow-number-text

## 1. 類別標籤照程式庫的數字文字規則

- [x] 1.1 先寫會失敗的測試：`1500000.0` 與 `0.00001` 的類別在適合度檢定與獨立性檢定的列名、欄名為 `1500000`、`0.00001`；以這些標籤為鍵的 `p` 可用，舊寫法 `1.5e+06` 被拒絕；字串、整數、`1.0`、`nil` 的標籤不變；在舊程式上失敗
- [x] 1.2 `stats/chi_square.go` 兩個檢定都改用 `utils.ValueText`，doc comment 改寫標籤規則；1.1 通過

## 2. 獨立性檢定同一時點讀兩個 list

- [x] 2.1 先寫會失敗的測試：沿用 `resizingPair`，另一個 goroutine 以 `AtomicDoAll` 同時改變兩個 list 的長度，反覆呼叫 `ChiSquareIndependenceTest`，不得出現長度不一致的錯誤；在舊程式上失敗
- [x] 2.2 以 `insyra.AtomicDoAll` 讀取兩個 list；2.1 通過

## 3. 驗證

- [x] 3.1 `gofmt`、`go build ./...`、`go vet ./...`、`go test ./...`（含 `-race` 跑 stats）、`golangci-lint run`（0 issues）通過；stats 跨語言套件以 `INSYRA_REQUIRE_REFERENCE_TOOLCHAINS=1` 執行通過

## 4. 文件

- [x] 4.1 `Docs/stats.md` 與 `Docs/cli-dsl.md` 的標籤規則改寫，附大數字的例子
- [x] 4.2 `CHANGELOG.md` 與 `CHANGELOG_TW.md` 的 `## Unreleased` 在 `` ### `stats` `` 末尾新增 BREAKING 條目
- [x] 4.3 `api-review.md` 的 ST-6 列補上這次的修正
- [x] 4.4 刪除 `AGENTS.md` 中已解決的卡方 follow-up
- [x] 4.5 `delivery-status.md` 的 Latest Milestones 最上方新增條目
- [x] 4.6 `openspec validate chisq-labels-follow-number-text --strict` 通過
