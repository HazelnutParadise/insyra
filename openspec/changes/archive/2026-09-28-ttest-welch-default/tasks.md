# Tasks: ttest-welch-default

## 1. 函式庫

- [x] 1.1 先寫會失敗的測試：`TwoSampleTTest(x, y)` 的統計量、自由度與 p 值等於 R `t.test(x, y)`（Welch），且與 `TTestOptions{EqualVariance: false}` 逐位元相同；`TTestOptions{EqualVariance: true}` 等於 R `t.test(x, y, var.equal = TRUE)`；在舊程式上編譯失敗於 `unknown field EqualVariance in struct literal of type stats.TTestOptions` 與 `cannot use stats.TTestOptions{…} as bool value in argument to stats.TwoSampleTTest`
- [x] 1.2 `TTestOptions` 新增 `EqualVariance bool`，`TwoSampleTTest` 拿掉位置參數 `equalVariance` 改讀這個欄位；`stats` 內所有呼叫改用新形式，斷言的值不改；1.1 的測試通過

## 2. CLI

- [x] 2.1 先寫會失敗的測試：`ttest two a b` 的輸出等於 `ttest two a b unequal`、不等於 `ttest two a b equal`；舊程式上預設輸出是 `t=4.166937607487551`（Student），`unequal` 是 `t=4.031794463425829`，兩項斷言都失敗
- [x] 2.2 `cli/commands/hypothesis.go` 的 `ttest two` 沒有變異數寫法時執行 Welch 檢定，Forms 說明寫出 `default: unequal, Welch's test`；2.1 的測試與 `go test ./cli/...` 通過

## 3. 驗證

- [x] 3.1 `go build ./...`、`go vet ./...`、`go test ./...` 全數通過；`golangci-lint run` 以獨立快取（`GOLANGCI_LINT_CACHE`）執行為 0 issues，共用快取會混入其他 worktree 的檔案；stats 跨語言套件以 `INSYRA_REQUIRE_REFERENCE_TOOLCHAINS=1` 通過

## 4. 文件

- [x] 4.1 `Docs/stats.md` 的 Test Settings、Two Sample T-Test、選擇檢定的表格與範例改成新形式並寫出預設為 Welch；`Docs/tutorials/ab-test-decision-with-statistics.md` 的呼叫改成新形式（原本就傳 `false`，結果不變），完整程式編譯並執行過
- [x] 4.2 `CHANGELOG.md` 與 `CHANGELOG_TW.md` 的 `## Unreleased`：`` ### `stats` `` 末尾新增 BREAKING 條目，`### CLI` 末尾新增 BREAKING 條目
- [x] 4.3 `api-review.md` 的 ST-4 列改為不再有待決事項；刪掉 `AGENTS.md` 記錄這個待決問題的 follow-up
- [x] 4.4 `delivery-status.md` 的 Latest Milestones 最上方新增條目
- [x] 4.5 `npx -y @fission-ai/openspec@latest validate ttest-welch-default --strict` 通過
