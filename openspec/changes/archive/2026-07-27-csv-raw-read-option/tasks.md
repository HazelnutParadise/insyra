# Tasks: csv-raw-read-option

## 1. Root package (read.go)

- [x] 1.1 新增 `CSVReadOptions` struct 與私有 `csvRowsToDataTable` 共用邏輯(建欄、填資料、依 `RawStrings` 決定是否呼叫 `inferCSVColumnTypes`)
- [x] 1.2 新增 `ReadCSV_FileWithOptions` / `ReadCSV_StringWithOptions`,舊 `ReadCSV_File` / `ReadCSV_String` 改為薄包裝
- [x] 1.3 `read_test.go` 新增測試:raw 模式保留 `0050`/`00878` 開頭零、`"2,000"` 原樣、空 cell 為 `""`;零值 options 與舊函數輸出一致

## 2. isr

- [x] 2.1 `isr/csv.go` 的 `CSV_inOpts` 加 `RawStrings bool`;`isr/dt.go` 改呼叫 WithOptions 版本
- [x] 2.2 isr 測試:`DT.From(CSV{InputOpts: {RawStrings: true}})` 全字串

## 3. CLI

- [x] 3.1 `cli/commands/load.go`:`fileLoadOptions` 加 `Infer`/`InferSet`,解析 `infer true|false`,非 CSV 帶 `infer` 回錯誤,更新 Usage/Forms/Examples
- [x] 3.2 CLI 測試:`load x.csv infer false` 全字串、非 CSV 帶 `infer` 報錯

## 4. Docs & skills

- [x] 4.1 `Docs/DataTable.md`:新增 `CSVReadOptions` 與兩個 WithOptions 函數說明(含使用時機)
- [x] 4.2 `Docs/cli-dsl.md`:`load` 選項與範例加 `infer`
- [x] 4.3 `skills/insyra/SKILL.md`:讀 CSV 範例補 RawStrings 用法
- [x] 4.4 `skills/use-insyra-cli/SKILL.md`:load 範例加 `infer false`
- [x] 4.5 AGENTS.md `## Follow-ups` 記錄 `ReadExcelSheet` 不做型別推斷的既有不一致

## 5. Verification

- [x] 5.1 `go test ./...` 全綠、`golangci-lint run` 無新錯誤
- [x] 5.2 REPL 手動驗證 `load x.csv infer false`
