# Tasks: stats-test-settings

## 1. 共用設定

- [x] 1.1 先寫會失敗的測試：四種設定型別的零值等於雙尾與 0.95、未知的對立假設、`NaN`／0 以外超出 (0, 1) 的信賴水準、兩個設定值，各自回傳 spec 規定的完整錯誤字串；記錄在舊程式上的結果（編譯失敗，以及以舊簽名呼叫時 `NaN` 信賴水準被當成 0.95 的實測輸出）。舊程式上 `stats/test_options_test.go` 編譯失敗於 `undefined: stats.ZTestOptions`、`undefined: stats.WilcoxonOptions`、`undefined: stats.MannWhitneyUOptions`、`undefined: stats.TTestOptions`。以舊簽名在 43cb6519 上實測：`SingleSampleTTest(x, 50, math.NaN())` 回傳 `[46.747, 58.939]`（與 0.95 相同），`SingleSampleWilcoxon(x, 50, TwoSided, math.NaN())` 回傳 `[46.8, 59.3]`（與預設相同），`SingleSampleZTest(x, 50, 10, TwoSided, math.NaN())` 回傳 `[NaN, NaN]`，三者錯誤皆為 nil；`MannWhitneyU(x, y, "")` 回傳 `invalid alternative hypothesis`
- [x] 1.2 新增 `stats/test_options.go`：泛型的「最多一個設定值」helper `oneOptions` 與讀取 `Alternative`、`ConfidenceLevel` 的 `resolveTestSettings`；`TTestOptions`、`ZTestOptions`、`WilcoxonOptions`、`MannWhitneyUOptions` 各放在對應檢定的檔案；刪掉用不到的 `resolveOptionalConfidenceLevel`；`stats/AGENTS.md` 的建構塊表加入兩個 helper、`tPValue` 與 `tMarginOfErrorOneSided`

## 2. t 檢定的單尾檢定

- [x] 2.1 先寫會失敗的測試：spec 的單尾情境（單樣本 greater、成對 less 90%、無變異資料對 `Less` 的 p 值為 1），以及雙尾結果與變更前逐位元相同；參考值取自 R 4.6.1 `t.test`
- [x] 2.2 `distutil.go` 新增 `tPValue(t, df, alt)`，`mathutil.go` 新增 `tMarginOfErrorOneSided`；三個 t 檢定改用 `opts ...TTestOptions`，區間以 `ciByAlternative` 建立；無變異資料的分支改走 `tPValue`；`stats/ttest_test.go` 既有的 R 對照值不改即通過
- [x] 2.3 `stats/testdata/crosslang_baseline.R` 與 `.py` 新增直接呼叫 R `t.test` 與 SciPy `ttest_1samp`／`ttest_ind`／`ttest_rel` 的 `t_test` 方法（R 的雙尾寫成 `two.sided`，腳本負責轉換）；`TestCrossLangTTestAlternative` 涵蓋單樣本、雙樣本等變異、雙樣本 Welch、成對 × 三種對立假設 × 0.95 與 0.9，共 24 個子測試，以 `INSYRA_REQUIRE_REFERENCE_TOOLCHAINS=1`、R 4.6.1、SciPy 1.13.1 在本機全數通過

## 3. z 檢定與秩檢定

- [x] 3.1 z 檢定改用 `opts ...ZTestOptions`；Wilcoxon 兩個函式改用 `opts ...WilcoxonOptions`，`MannWhitneyU` 改用 `opts ...MannWhitneyUOptions`；1.1 的測試通過。`TestZTest_InvalidInputs` 原本斷言信賴水準 0 是錯誤，改為斷言 0 等於 0.95、負值是錯誤，因為零值現在表示預設

## 4. 多組檢定與因素分析

- [x] 4.1 `OneWayANOVA`、`TwoWayANOVA`、`RepeatedMeasuresANOVA`、`KruskalWallis`、`FriedmanTest` 改收 `[]insyra.IDataList`；新增 `TestKSampleTestsTakeASlice`：nil slice 回錯誤不 panic
- [x] 4.2 先寫會失敗的測試：`FactorAnalysis(dt)` 與 `FactorAnalysis(dt, DefaultFactorAnalysisOptions())` 載荷逐格相同、兩個設定值回錯誤；再把參數改成 `opts ...FactorAnalysisOptions`

## 5. 呼叫端

- [x] 5.1 `stats` 內所有測試改用新簽名，斷言的值不改；`cli/commands/hypothesis.go` 改用新簽名，輸出不變（z 檢定原本傳 0.95，現在省略，等於同一個值）
- [x] 5.2 `go build ./...`、`go vet ./...`、`go test ./...`、`golangci-lint run`（0 issues）全數通過；stats 跨語言套件以 `INSYRA_REQUIRE_REFERENCE_TOOLCHAINS=1 go test -run TestCrossLang -skip 'TestCrossLang(ClusteringGeneratedCorpus|KNNGeneratedCorpus|FactorAnalysis)' ./stats/` 通過

## 6. 文件

- [x] 6.1 `Docs/stats.md`：t、z、Wilcoxon、MWU、ANOVA、Kruskal-Wallis、Friedman、FactorAnalysis 的簽名、參數與範例改成新形式；新增「Test Settings」一節與原本缺的「Two Sample Z-Test」一節；「Confidence Levels」一節改寫；範例中一個把 `*TwoWayANOVAResult` 指派給 `*OneWayANOVAResult` 變數而無法編譯的寫法一併改正；文件與兩篇教學的範例都放進實際程式編譯執行過
- [x] 6.2 `Docs/tutorials/nonparametric-tests-when-normality-fails.md` 與 `Docs/tutorials/ab-test-decision-with-statistics.md` 的呼叫改成新簽名，兩篇的完整程式都能編譯並執行
- [x] 6.3 檢查 `skills/insyra/`、`skills/use-insyra-cli/`：兩者都沒有列出這些函式的簽名，教的原則（尾端可省略參數只收一個值、結果是結構而不只一個數）也沒有改變，CLI 行為不變，因此不改
- [x] 6.4 `CHANGELOG.md` 與 `CHANGELOG_TW.md` 的 `## Unreleased` 在 `` ### `stats` `` 末尾新增條目：兩條 BREAKING（設定值、slice）與兩條新功能（單尾 t 檢定、`FactorAnalysis` 可省略設定）；CLI 輸出不變故不另立 CLI 條目
- [x] 6.5 `api-review.md` 的 ST-4 列標為 `~~Med~~ 已修正（stats-test-settings：…）`，問題清單的 ST-4 列同步
- [x] 6.6 `delivery-status.md` 的 Latest Milestones 最上方新增條目
- [x] 6.7 `AGENTS.md` Follow-ups 新增：`equalVariance` 的預設值待擁有者決定；回歸函式的 `GLM(opts, …)` 與 `...WithOptions` 雙名，以及 GLM／logistic／Poisson 的信賴水準超出範圍時退回 0.95（實測 1.5 與 `NaN` 都回報 0.95 的區間）
- [x] 6.7a 送出前的對抗式審查找到三個 P2，已修正：`Docs/stats.md` 無變異資料的 p 值說明改為依方向為 0 或 1；spec 的「±Inf 統計量 p 值為 0 或 1」限定自由度有定義時（兩組都沒有變異的 Welch 檢定自由度為 `NaN`，實測 p 值為 `NaN`）；`stats-test-input` 的「Parametric tests refuse unreadable cells」也加入 delta，改為只改呼叫形式。另外兩項實測為既有問題，記為 `AGENTS.md` follow-up：雙尾 t／z 的 p 值在極小處捨入成 0（t = 17.66 時為 0，R 為 3.31e-20），以及 z 檢定的 `sigma` 為 `NaN` 時不回錯誤
- [x] 6.8 `npx -y @fission-ai/openspec@latest validate stats-test-settings --strict` 通過
