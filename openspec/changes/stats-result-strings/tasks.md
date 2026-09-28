# Tasks: stats-result-strings

## 1. 共用的輸出規則

- [x] 1.1 先寫會失敗的測試：以測試專用的 struct 驗證標題、欄位順序、內嵌欄位在前、nil 指標與介面省略、浮點數文字（`1500000`、`1e-07`、`NaN`）、60 個以內整列與超過時的前 20 後 5 加總數、巢狀 struct、表格格線與超過 60 列的省略、無色碼、nil 接收者回傳 `<nil>`；在舊程式上編譯失敗於 renderer 未定義
- [x] 1.2 `stats/result_text.go` 實作 renderer；1.1 通過

## 2. 每個結果型別的 String 與 Show

- [x] 2.1 先寫會失敗的測試：解析 `stats` 原始碼，每個名稱以 `Result` 結尾的匯出 struct 都自己宣告 `String` 與 `Show`；在舊程式上失敗並列出缺少的型別
- [x] 2.2 假設檢定的結果（`TestResult`、`TTestResult`、`ZTestResult`、`FTestResult`、`ChiSquareTestResult`、`CorrelationResult`、`WilcoxonTestResult`、`MannWhitneyUResult`、`KruskalWallisResult`、`FriedmanTestResult`）加上 `String` 與 `Show`，`ChiSquareTestResult.Show` 改為印出 `String()`；各型別至少一個輸出測試
- [x] 2.3 ANOVA、PCA、分群、KNN 的結果加上 `String` 與 `Show`；各型別至少一個輸出測試
- [x] 2.4 迴歸、`BartlettTestResult` 與 `FactorAnalysisResult` 加上 `String` 與 `Show`，`FactorAnalysisResult.Show` 無範圍時印出 `String()`、有範圍時照舊；各型別至少一個輸出測試；2.1 通過
- [x] 2.5 自行以零值與 nil 指標呼叫所有結果型別的 `String()`：`(*FactorModel)(nil).String()` 會 panic（透過值內嵌取得的方法先解參考），`(*FactorAnalysisResult)(nil).Show(5)` 同樣會 panic。先寫會失敗的測試，再為 `FactorModel` 宣告自己的 `String` 與 `Show`、為 `FactorAnalysisResult.Show` 加上 nil 檢查

## 3. 驗證

- [x] 3.1 `gofmt`、`go build ./...`、`go vet ./...`、`go test ./...`、`golangci-lint run`（0 issues）通過；stats 跨語言套件以 `INSYRA_REQUIRE_REFERENCE_TOOLCHAINS=1` 執行通過

## 4. 文件

- [x] 4.1 `Docs/stats.md` 的 Common Result Structure 新增「Printing a result」：`String`、`Show` 與 `fmt.Fprintln(w, res)` 的關係與版面規則；卡方與因素分析的 Show 段落改寫；範例實際編譯執行過
- [x] 4.2 `CHANGELOG.md` 與 `CHANGELOG_TW.md` 的 `## Unreleased` 在 `` ### `stats` `` 末尾新增條目，註明兩個既有 `Show` 的版面改變
- [x] 4.3 `api-review.md` 的 ST-9：`Show`／`String` 部分標為已修正（stats-result-strings），問題清單同步
- [x] 4.4 `delivery-status.md` 的 Latest Milestones 最上方新增條目
- [x] 4.5 `skills/insyra/SKILL.md` 的「Results are more than one number」補上印出整個結果的方式；`skills/use-insyra-cli/` 不受影響
- [x] 4.6 `openspec validate stats-result-strings --strict` 通過
