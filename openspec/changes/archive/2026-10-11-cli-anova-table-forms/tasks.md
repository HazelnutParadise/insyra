# Tasks: cli-anova-table-forms

## 1. anova 的表格形式

- [x] 1.1 測試：`anova twoway t score drug dose` 的輸出等於 `stats.TwoWayANOVAFromTable` 的結果，也等於同樣資料切成清單後的 `anova twoway 2 2 …`；`anova repeated t value cond subj` 同樣對照程式庫與清單形式；參數數量不對、欄位不存在、缺少觀察值都回傳錯誤；第一個參數能讀成整數時優先走清單形式；既有清單形式的輸出不變
- [x] 1.2 `cli/commands/hypothesis.go`：`twoway` 與 `repeated` 在第一個參數為 DataTable 且不是整數時走表格形式，更新 `Forms` 與 `Examples`

## 2. friedman 指令

- [x] 2.1 測試：`friedman` 的表格形式與清單形式輸出相同，也等於 `stats.FriedmanTestFromTable`；參數數量不對與程式庫的錯誤都回傳錯誤；`help friedman` 列出兩種形式
- [x] 2.2 註冊 `friedman`（`Usage`、`Forms`、`Examples`、`Args`），實作表格與清單兩種形式

## 3. 驗證

- [x] 3.1 驗證通過：`go test ./cli/commands/...` 通過

## 4. 文件

- [x] 4.1 `Docs/cli-dsl.md`：指令索引新增 `friedman`，指令分組加入 `friedman`，新增表格形式的說明段落
- [x] 4.2 `CHANGELOG.md` 與 `CHANGELOG_TW.md` 的 `## Unreleased` 在 `### CLI` 末尾新增條目
