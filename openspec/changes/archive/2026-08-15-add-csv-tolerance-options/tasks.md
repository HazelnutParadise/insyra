# Tasks: add-csv-tolerance-options

## 1. Root package (read.go)

- [x] 1.1 `CSVReadOptions` 加 `AllowRaggedRows bool` 與 `TrimLeadingSpace bool`（含 doc comment，說明預設嚴格行為不變、短列補空、長列自動加欄）
- [x] 1.2 `ReadCSV_FileWithOptions` / `ReadCSV_StringWithOptions` 兩處 reader 依選項設定 `FieldsPerRecord = -1` 與 `TrimLeadingSpace = true`
- [x] 1.3 `csvRowsToDataTable` 支援 ragged rows：短列補 `""` 維持列對齊；長列多出的 cell 動態新增自動命名欄位，先前各列在新欄補 `""`；與 `FirstColToRowNames`（單欄列只有列名）、`FirstRowToColNames` 正確互動
- [x] 1.4 `read_test.go` 新增測試：表尾註記列（短列補空）、列尾逗號（多一個空欄）、長列非空多欄資料保留且先前列補空、引號前空白（`TrimLeadingSpace`）可解析、預設（零值選項）對上述輸入仍回傳錯誤、`RawStrings + AllowRaggedRows` 組合全字串、String 與 File 兩版行為一致

## 2. isr

- [x] 2.1 `isr/csv.go` 的 `CSV_inOpts` 加 `AllowRaggedRows` / `TrimLeadingSpace`；`isr/dt.go` 傳遞至 options
- [x] 2.2 isr 測試：`DT.From(CSV{InputOpts: {AllowRaggedRows: true, TrimLeadingSpace: true}})` 載入含雜訊 CSV 成功

## 3. CLI

- [x] 3.1 `cli/commands/load.go`：`fileLoadOptions` 加 `Ragged`/`RaggedSet`、`TrimSpace`/`TrimSpaceSet`，解析 `ragged true|false`、`trimspace true|false`（`parseFlexBool`），非 CSV 帶選項回錯誤，更新 Usage/Forms/Examples
- [x] 3.2 CLI 測試：`load x.csv ragged true trimspace true` 成功載入雜訊 CSV、非 CSV 帶選項報錯

## 4. Docs, changelog & skills

- [x] 4.1 `Docs/DataTable.md`：`CSVReadOptions` 說明補兩個新選項（含短列/長列語意與使用時機）
- [x] 4.2 `Docs/cli-dsl.md`：`load` 選項與範例加 `ragged`、`trimspace`
- [x] 4.3 `skills/insyra/SKILL.md`（及 references 若有 CSV 段落）：補新選項用法
- [x] 4.4 `skills/use-insyra-cli/SKILL.md`（及 references 若有 load 段落）：load 範例加新選項
- [x] 4.5 `CHANGELOG.md` 與 `CHANGELOG_TW.md` `## Unreleased` 下 `### Core` 與 `### CLI` 各加對應條目（附 issue #198 連結）

## 5. Verification

- [x] 5.1 `go test ./...` 全綠、`golangci-lint run` 無新錯誤
- [x] 5.2 `openspec validate add-csv-tolerance-options --strict` 通過
