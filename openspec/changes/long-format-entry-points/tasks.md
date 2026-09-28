# Tasks: long-format-entry-points

## 1. R 參考值

- [x] 1.1 新增 `stats/testdata/long_format_reference.R`，以打亂列順序、水準為文字的長格式資料呼叫 `aov(value ~ A * B)`（平衡設計）、`aov(value ~ cond + Error(subj/cond))` 與 `friedman.test(value ~ cond | subj)`（含同分），輸出資料本身與平方和、自由度、F、統計量、p 值；產生並提交 `stats/testdata/long_format_reference.txt`

## 2. 讀取長格式表格

- [x] 2.1 先寫會失敗的測試：選擇器三種寫法、無法解析的選擇器回傳錯誤且呼叫端 `Err()` 為 nil、nil 表格、數值欄的非數值與非有限值、因子欄的 nil 與 `NaN`、`int` 與 `int64` 的 1 為同一水準；在舊程式上編譯失敗於函式未定義
- [x] 2.2 `stats/long_format.go`：在表格的複本上解析欄位，讀出數值與各因子的水準（依第一次出現的順序）；2.1 通過

## 3. 三個入口

- [x] 3.1 先寫會失敗的測試：三個函式與以第一次出現順序分組後呼叫清單版函式的結果逐欄位相等（`==`）；與 1.1 的 R 參考值相符；缺格、重複、缺少觀察值與水準不足的錯誤訊息
- [x] 3.2 實作 `TwoWayANOVAFromTable`、`RepeatedMeasuresANOVAFromTable`、`FriedmanTestFromTable`，分組後呼叫既有的清單版函式；3.1 通過

## 4. 驗證

- [x] 4.1 `gofmt`、`go build ./...`、`go vet ./...`、`go test ./...`、`golangci-lint run`（0 issues）通過；stats 跨語言套件以 `INSYRA_REQUIRE_REFERENCE_TOOLCHAINS=1` 執行通過

- [x] 4.2 查找欄位失敗時，複本上的 `GetCol` 仍會寫一行 log；已寫進 `Docs/stats.md`，並把「根套件提供不留副作用的欄位解析」記為 `AGENTS.md` follow-up

## 5. 文件

- [x] 5.1 `Docs/stats.md`：三個函式各自的說明（表格形狀、水準的比較方式、拒絕的輸入、與清單版的關係），補上原本沒有的 `RepeatedMeasuresANOVA` 章節，刪去 Two Way ANOVA 的「沒有長格式入口」；範例實際編譯執行過
- [x] 5.2 `CHANGELOG.md` 與 `CHANGELOG_TW.md` 的 `## Unreleased` 在 `` ### `stats` `` 末尾新增條目
- [x] 5.3 `api-review.md` 的 ST-7 列標為 `~~Med~~ 已修正（long-format-entry-points）`，問題清單同步
- [x] 5.4 `delivery-status.md` 的 Latest Milestones 最上方新增條目
- [x] 5.5 檢查 `skills/insyra/` 與 `skills/use-insyra-cli/`：兩者都不列函式，CLI 的 `anova` 形式不變，本變更不需修改
- [x] 5.6 `openspec validate long-format-entry-points --strict` 通過
