# Tasks: long-format-entry-points

## 1. R 參考值

- [x] 1.1 新增 `stats/testdata/long_format_reference.R`，以打亂列順序、水準為文字的長格式資料呼叫 `aov(value ~ A * B)`（平衡設計）、`aov(value ~ cond + Error(subj/cond))` 與 `friedman.test(value ~ cond | subj)`（含同分），輸出資料本身與平方和、自由度、F、統計量、p 值；產生並提交 `stats/testdata/long_format_reference.txt`

## 2. 讀取長格式表格

- [x] 2.1 測試：字串選擇器（Excel 字母優先、再比欄名）、無法解析的選擇器回傳錯誤且呼叫端 `Err()` 為 nil、nil 表格、數值欄的非數值與非有限值、因子欄的 nil 與 `NaN`、`int` 與 `int64` 的 1 為同一水準
- [x] 2.2 `stats/long_format.go`：在表格的複本上解析字串欄位，讀出數值與各因子的水準（依第一次出現的順序，並以 `ToMapKey` 比較水準）

## 3. 三個入口

- [x] 3.1 測試：三個函式與以第一次出現順序分組後呼叫清單版函式的結果逐欄位相等（`==`）；與 1.1 的 R 參考值相符；缺格、重複、缺少觀察值與水準不足的錯誤訊息；配置前檢查水準組合
- [x] 3.2 實作 `TwoWayANOVAFromTable`、`RepeatedMeasuresANOVAFromTable`、`FriedmanTestFromTable`，先檢查水準組合再配置記憶體，分組後呼叫既有的清單版函式

## 4. 驗證

- [x] 4.1 驗證通過：`go test ./stats/...` 與相關測試通過
- [x] 4.2 欄位解析在表格複本上執行，呼叫端表格不留錯誤

## 5. 文件

- [x] 5.1 `Docs/stats.md`：三個函式各自的說明（表格形狀、水準的比較方式、拒絕的輸入、與清單版的關係），補上 `RepeatedMeasuresANOVA` 章節，更新 Two Way ANOVA 說明
- [x] 5.2 `CHANGELOG.md` 與 `CHANGELOG_TW.md` 的 `## Unreleased` 在 `### stats` 末尾新增條目
