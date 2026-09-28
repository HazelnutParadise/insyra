# Tasks: stats-test-results

## 1. 匯出共同結果型別

- [x] 1.1 先寫會失敗的測試：九個結果型別的指標都能指派給 `stats.HypothesisTestResult`；`Base()` 回傳 `&res.TestResult`；把 t 檢定、MWU、卡方獨立性檢定的結果放進同一個 slice，`Base().PValue` 等於各自的 `PValue`；在舊程式上編譯失敗於 `undefined: stats.HypothesisTestResult` 與 `res.Base undefined (type *stats.TTestResult has no field or method Base)`
- [x] 1.2 `stats/structs.go` 把 `testResultBase` 改名為 `TestResult`，新增 `HypothesisTestResult` 介面與 `(*TestResult).Base()`；九個結果型別與其建構處改用新名稱，`git grep testResultBase -- '*.go'` 已無結果；1.1 的測試通過

## 2. 不適用的欄位一律為 nil

- [x] 2.1 先寫會失敗的測試：`PairedTTest` 填入 `Mean`、`Mean2`、`N2`；單樣本與雙樣本 t 檢定的 nil 欄位；精確路徑的 Wilcoxon 與 MWU 的 `Z` 為 nil、近似路徑不為 nil，差值全為零（`Method` 為 `"undefined"`）時也為 nil；`BartlettTest` 的 `DF2` 為 nil、`LeveneTest` 的 `DF2` 為 9；在舊程式上編譯失敗於 `cannot use r.Mean (variable of type *float64) as float64 value`、`mismatched types float64 and untyped nil`（`Z`、`DF2`）
- [x] 2.2 `TTestResult.Mean` 改為 `float64`，`PairedTTest` 以 `meanOfF64` 算出兩組平均數；`WilcoxonTestResult.Z`、`MannWhitneyUResult.Z`、`FTestResult.DF2` 改為 `*float64`；既有測試改讀新型別，斷言的值不改。`stats/ttest_test.go` 的成對表格原本斷言 `Mean`、`Mean2`、`N2` 為 nil，這正是本變更推翻的規則，改為斷言 `N2` 等於成對數、`Mean - *Mean2` 等於已釘住的 `MeanDiff`；兩個 doc comment 裡與程式不符的精確分布門檻（Wilcoxon 的 `n_eff <= 50`、MWU 的 `n1, n2 <= 25`）改成程式實際用的 `< 50`；2.1 的測試通過

## 3. 驗證

- [x] 3.1 `go build ./...`、`go vet ./...`、`go test ./...`、`golangci-lint run`（0 issues）全數通過；stats 跨語言套件以 `INSYRA_REQUIRE_REFERENCE_TOOLCHAINS=1` 跑過，26 個測試全數通過，其中 `TestCrossLangWilcoxon*`、`TestCrossLangMannWhitneyU` 讀的是新的 `*Z`

## 4. 文件

- [x] 4.1 `Docs/stats.md` 的「Common Result Structure」改寫為 `TestResult` 與 `HypothesisTestResult`，附一個處理任意檢定結果的範例；t 檢定、F 檢定與無母數檢定的結果型別區塊改成新欄位型別並說明何時為 nil；範例放進實際程式編譯執行過
- [x] 4.2 `CHANGELOG.md` 與 `CHANGELOG_TW.md` 的 `## Unreleased` 在 `` ### `stats` `` 末尾新增條目：`TestResult` 與 `HypothesisTestResult`（新功能），`TTestResult.Mean`、`Z`、`DF2` 的型別變更（BREAKING，註明以 `fmt` 印出這兩個欄位仍會編譯但會印出指標）
- [x] 4.3 `api-review.md` 的 ST-5 列標為 `~~Med~~ 已修正（stats-test-results：…）`，問題清單的 ST-5 列同步
- [x] 4.4 `delivery-status.md` 的 Latest Milestones 最上方新增條目
- [x] 4.5 `npx -y @fission-ai/openspec@latest validate stats-test-results --strict` 通過
