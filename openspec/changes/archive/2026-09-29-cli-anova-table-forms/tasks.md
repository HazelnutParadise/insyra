# Tasks: cli-anova-table-forms

## 1. anova 的表格形式

- [x] 1.1 先寫會失敗的測試：`anova twoway t score drug dose` 的輸出等於 `stats.TwoWayANOVAFromTable` 的結果，也等於同樣資料切成清單後的 `anova twoway 2 2 …`；`anova repeated t value cond subj` 同樣對照程式庫與清單形式；欄位 token 用字母、編號與 `name:` 的結果相同；參數數量不對、欄位不存在、缺少觀察值都回傳錯誤；既有清單形式的輸出不變
- [x] 1.2 `cli/commands/hypothesis.go`：`twoway` 與 `repeated` 在第一個參數是 DataTable 變數時走表格形式，更新 `Forms` 與 `Examples`；1.1 通過

## 2. friedman 指令

- [x] 2.1 先寫會失敗的測試：`friedman` 的表格形式與清單形式輸出相同，也等於 `stats.FriedmanTestFromTable`；參數數量不對與程式庫的錯誤都回傳錯誤；`help friedman` 列出兩種形式
- [x] 2.2 註冊 `friedman`（`Usage`、`Forms`、`Examples`、`Args`），實作兩種形式；2.1 通過
- [x] 2.3 審查：`runFriedmanCommand` 原本直接讀 `Registry["friedman"]`，`registry.go` 註明這在另一個 goroutine 註冊時不安全；改用和 `accel` 一樣的 `friedmanUsage` 常數，並把兩種形式重複的輸出收成 `printFriedman`；`-race` 下測試通過

## 3. 驗證

- [x] 3.1 `gofmt`、`go build ./...`、`go vet ./...`、`go test ./...`、`golangci-lint run`（0 issues）通過；以建好的 CLI 實際跑過文件裡的每個範例

## 4. 文件

- [x] 4.1 `Docs/cli-dsl.md`：指令索引新增 `friedman` 一列，指令分組加入 `friedman`，新增表格形式的說明段落（形式、何時走表格形式、欄位 token 規則、輸出）
- [x] 4.2 `CHANGELOG.md` 與 `CHANGELOG_TW.md` 的 `## Unreleased` 在 `### CLI` 末尾新增條目
- [x] 4.3 `delivery-status.md` 的 Latest Milestones 最上方新增條目
- [x] 4.4 檢查 `skills/use-insyra-cli/`：不列指令清單，本變更不需修改
- [x] 4.5 `openspec validate cli-anova-table-forms --strict` 通過
