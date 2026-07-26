# Design: csv-raw-read-option

## Context

CSV 讀取邏輯集中在 `read.go`:`ReadCSV_File`(含編碼偵測)與 `ReadCSV_String` 各自實作「parse rows → 建欄 → 填資料」後呼叫 `inferCSVColumnTypes(dt)`。兩函數本體高度重複。`ReadCSV_File` 尾端已有 `encoding ...string` 可變參數,無法再無痛追加參數。isr 層以 `CSV_inOpts` struct 包裝,CLI `load` 以 `parseFileLoadOptions` 解析 key-value 選項。

## Goals / Non-Goals

**Goals:**
- 提供關閉型別推斷的 CSV 讀法,cell 保留原始字串。
- 既有 API 簽名與行為零變動。
- 選項集中於可擴充的 struct,未來加選項不再新增函數。

**Non-Goals:**
- Per-column 型別指定(如 `StringCols`)— 等有真實需求再擴充。
- Excel/JSON 讀取行為調整(`ReadExcelSheet` 本來就不推斷,見 Follow-ups)。
- decimal 型別支援 — 由呼叫端自行解析。

## Decisions

1. **Options struct + 新函數對,而非 `_Raw` 函數對或改簽名**:`ReadCSV_FileWithOptions` / `ReadCSV_StringWithOptions` 接 `CSVReadOptions`。與 `parquet.Read(ctx, path, opt)` 風格一致;`_Raw` 對會讓 API 隨選項數翻倍;改簽名是 breaking change。
2. **零值即現行為**:欄位取名 `RawStrings`(而非 `InferTypes`),使 `CSVReadOptions{}` 的零值對應「照舊推斷」,舊函數可直接委派。
3. **消重**:抽出私有函數 `csvRowsToDataTable(rows [][]string, opts CSVReadOptions) *DataTable` 承載共用的建欄/填資料/推斷邏輯;File 版只多編碼偵測,String 版只多 BOM 剝除。不新增其他抽象。
4. **空 cell 語意**:raw 模式空 cell 保留 `""`。呼叫端自行辨識空值,符合 issue 期望(不再是 NaN)。
5. **CLI 選項名用 `infer`**:與「type inference」概念直接對應,布林形式與既有 `headers`/`rownames` 一致;非 CSV 格式帶 `infer` 回錯誤,比照 `sheet`/`encoding` 的既有檢查模式。

## Risks / Trade-offs

- [root API 出現第三種讀法] → 文件明確標示 WithOptions 為建議入口,舊函數說明指向新函數;不 deprecate,避免 churn。
- [重構共用邏輯可能改變既有行為] → 以既有測試 + 新增「零值 options 與舊函數輸出一致」測試把關。

## Open Questions

無 — API 形狀已與維護者確認(options struct)。
