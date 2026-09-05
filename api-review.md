# API Review — 生產環境可用性逐項審查

目標：insyra 每一個公開符號都經過設計審查後才打勾。打勾的定義是「已讀過實作、對照下列準則、寫下結論」，不是「看起來沒問題」。
產生方式：`go/ast` 掃描所有非 `internal` 套件的匯出符號（不含 `_test.go`），共 1763 項。清單本身不含 `internal/`，因為那不是使用者會呼叫的介面。
修法：發現的問題不在這裡直接改，彙整成 OpenSpec change 再處理（依 AGENTS.md）。

## 審查準則

每一項都要對照下面 14 條，寫得出「哪幾條過、哪幾條不過」才能打勾。「看起來沒問題」不算。

**A. 該不該存在**
1. 功能與範圍：解決的是使用者真的會遇到的問題嗎？和同套件或其他套件的 API 重疊嗎？單一職責嗎？拿掉它會損失什麼？（減法優先）
2. 公開邊界：這個符號真的需要匯出嗎？只是實作細節就該進 `internal/`。一旦公開就是相容性承諾。

**B. 好不好用（直覺性）**
3. 可發現性：只看名稱和簽名，不看文件，猜得到它做什麼、回傳什麼嗎？
4. 最常見用法是最短路徑：寫出「最典型的三行呼叫」，如果要先建物件、串三個方法、或傳一堆 nil 才能做最基本的事，就不及格。
5. 心智模型：目標使用者是資料分析者。對照 pandas / numpy / R / Excel 的對應操作，行為、預設值、命名方向一致嗎？（例如 `head`、`dropna`、`groupby` 的語意）
6. 對稱與可預測：`Read/Write`、`Get/Set`、`ByIndex/ByName`、`Col/Row` 成對出現且行為對稱嗎？同一件事在不同套件有幾種寫法？
7. 零值與預設：不給選項時的行為是最安全、最常用的嗎？options struct 的零值可用嗎？有沒有「不傳就 panic」的隱藏前提？

**C. 像不像業界標準（Go 慣例與知名程式庫）**
8. Go 簽名慣例：`ctx` 放第一個參數；選項用 struct 或 functional options，不用 variadic 冒充選填；不用 `any` 逃避型別；不用裸 `bool` 參數；接受介面、回傳具體型別；I/O 接受 `io.Reader`/`io.Writer` 而非只吃路徑；序列考慮 `iter.Seq`（go.mod 已是 1.25）。
9. Go 命名慣例：縮寫全大寫（`CSV`、`ID`、`URL`、`JSON`）；不加 `Get` 前綴；套件名不重複在符號裡（避免 `plot.PlotBar`）；boolean 用 `Is/Has`；`New`/`From`/`To`/`Parse` 依 Go 標準庫用法。
10. 對照標竿：與標準庫（`encoding/csv`、`database/sql`、`net/http`）和領域內成熟程式庫（gonum、go-echarts、Arrow、gorgonia）的做法比，差異是有理由的還是隨手寫的？

**D. 會不會壞（生產環境）**
11. 錯誤處理：失敗不能被吞；`%w` 包裝可 `errors.Is/As`；提供 sentinel 或 typed error 讓呼叫端分流；不對使用者輸入 panic；部分成功要能表達；錯誤訊息帶上下文；不拿 log 代替回傳錯誤；`Err()` 模式與回傳 `error` 兩種風格在同一套件內不能混。
12. 資源、併發、邊界：`ctx` 取消有效；goroutine/channel/檔案/`*excelize.File` 不洩漏；呼叫兩次、中途中斷、空輸入、nil receiver、超大輸入各會怎樣；thread safety 在文件明講。
13. 資料正確性（insyra 特有）：nil / NaN / 空字串 / 混合型別的處理明確且有文件；型別轉換不得捏造值（`ToF64Slice` 零值教訓）；遵守 ENG.md 精度契約；排序穩定；浮點比較。
14. 安全與可觀測：路徑與暫存檔權限；外部請求有 timeout；輸入大小上限；不 log 敏感資料；成功路徑不該噪音 log；能注入時鐘、HTTP client 等依賴以利測試。

**E. 契約**
- 每個匯出符號都有 Go 風格 doc comment（「Read reads …」，不是「Read: read …」），且描述與實作一致；`Docs/*.md` 與 `skills/` 同步；棄用要標 `Deprecated:`。這條每項必檢，違反直接列入發現。

嚴重度：**High** 生產環境會靜默出錯、資料損失或資源耗盡；**Med** 誤用不易察覺、契約不符、明顯不符業界慣例、資源洩漏；**Low** 命名、文件、直覺性小瑕疵。

## 進度

| 套件 | 項目數 | 狀態 |
| --- | --- | --- |
| `.`（core） | 565 | **完成** |
| `accel` | 127 | 未開始 |
| `cli` 系列（cli, commands, env, repl, style） | 79 | 未開始 |
| `csvxl` | 9 | **完成** |
| `datafetch` | 42 | 未開始 |
| `engine` 系列（algorithms, atomic, biindex, ccl, dsl, ring） | 41 | 未開始 |
| `finance` | 60 | **完成** |
| `gplot` | 15 | 未開始 |
| `isr` | 44 | **完成** |
| `lp` / `lpgen` | 12 | 未開始 |
| `mkt` | 14 | **完成** |
| `ml` / `ml/mltest` | 217 | 未開始 |
| `nn` | 231 | 未開始 |
| `parallel` | 5 | **完成** |
| `parquet` | 12 | **完成** |
| `pd` | 8 | 未開始 |
| `plot` | 80 | 未開始 |
| `py` | 14 | 未開始 |
| `quant` | 50 | **完成** |
| `stats` | 197 | **完成** |
| 無匯出符號：`accel/knnbridge`, `allpkgs`, `benchmark`, `cmd/insyra`, `engine`, `tools/gendocs` | 0 | 不需審查 |

## 發現彙整（依套件）

「已修正」表示該項在 `fix-api-review-batch-1`（2026-09-05）處理完畢；D-4 只修了「當 0」的部分，「跳過」的政策決定仍待處理。

### csvxl

| 編號 | 嚴重度 | 問題 | 位置 | 建議 |
| --- | --- | --- | --- | --- |
| C-1 | High | 單一 CSV 轉換失敗時只累加計數，錯誤內容丟棄；Excel 仍然存檔（含空 sheet），最後回傳「N files failed」。呼叫端不知道哪個檔、為什麼 | csvxl/convert.go:64-67, 116-119 | 用 `errors.Join` 帶檔名回傳每個失敗；或改成 fail-fast 不存檔。二選一要由你決定：批次容錯還是全有全無 |
| C-2 | Med | `csvEncoding ...string` / `encoding ...string` 拿 variadic 當選填參數，傳兩個以上才在執行期報錯；未知編碼字串（如 `"latin1"`）靜默走 raw 讀取 | convert.go:31-37, 86-92; convertDir.go:16; read_csv.go:11 | 改成明確參數或 options struct；未知編碼回傳錯誤 |
| C-3 | ~~Med~~ 已修正 | `AppendCsvToExcel` doc 說「sheet 已存在會被覆寫」，但 excelize `NewSheet` 對既有名稱只回傳索引不清空（已查 v2.11.0 sheet.go:57-59），結果是新資料蓋在舊資料上，舊資料超出範圍的儲存格殘留 | convert.go:83-107 | 存在時先 `DeleteSheet` 再建，或改 doc 說明是合併 |
| C-4 | ~~Med~~ 已修正 | `excelize.OpenFile` 回傳的 `*File` 從未 `Close()`：`AppendCsvToExcel`、`ExcelToCsv`、`EachExcelToCsv`（每個檔案各漏一次） | convert.go:94, 136; convertDir.go:38 | `defer f.Close()` |
| C-5 | Low | 錯誤用 `%v` 包裝，呼叫端無法 `errors.Is(err, os.ErrNotExist)`；`read_csv.go` 已用 `%w`，套件內不一致 | convert.go 全檔 | 改 `%w` |
| C-6 | Low | `UTF8/Big5/Auto` 是裸 string 常數，而實際比對用 `strings.Contains`，任何字串都會被接受 | convert.go:22-26 | typed `Encoding` string 型別 |
| C-7 | Low | 目錄用 `os.ModePerm`（0777）建立 | convert.go:143; convertDir.go:50 | 0755 |
| C-8 | Low | 路徑沒有 `.csv` 結尾就自動補；`ExcelToCsv` 的 `onlyContainSheets` 指到不存在的 sheet 靜默略過；`EachExcelToCsv` log 標錯函式名 | convert.go:44, 158-164; convertDir.go:62 | 不補副檔名（或改 doc）；找不到的 sheet 回錯；修 log |
| C-10 | Med | 全套件只吃檔案路徑，沒有 `io.Reader`/`io.Writer` 版本：記憶體中的 CSV、HTTP 回應、`embed.FS` 都得先落地成檔案才能轉（準則 8、10） | 全套件 | 核心改成 Reader/Writer，路徑版當薄包裝 |
| C-11 | Med | `CsvToExcel(csvFiles, sheetNames, ...)` 用兩個平行切片靠索引對位，錯一格就對到別的 sheet；`ExcelToCsv(…, csvNames, onlyContainSheets...)` 同樣問題（準則 4、8） | convert.go:31, 135 | `[]SheetSpec{Path, Sheet}` 一個切片 |
| C-12 | Low | 命名不符 Go 慣例：`Csv` 應為 `CSV`；`EachCsvToOneExcel` 讀起來要想一下（「每個 CSV 到一個 Excel」）；doc comment 缺 Go 風格開頭（準則 3、9、E） | 全套件 | v1 前統一改名 |
| C-9 | Low | 每次成功都以 Info 等級寫 log。這是跨套件模式（parquet 除外），要在 core 的 logger 審查時一併決定 library 該不該在成功路徑上 log | 全套件 | 待 core 決定 |

### parallel

| 編號 | 嚴重度 | 問題 | 位置 | 建議 |
| --- | --- | --- | --- | --- |
| P-1 | High | 整個 API 以 `any` 進出：函式是 `any`，結果是 `[][]any`。worker panic 被轉成 `error` 塞進結果槽，與函式本身回傳 `error` 的情況無法區分；型別錯誤只能在執行期發現 | parallel/parallel_computing.go:16, 24-58, 61 | 見 P-4 |
| P-2 | Med | `Run` 呼叫兩次會重新執行所有函式並覆寫結果；沒有 context、沒有併發上限 | parallel_computing.go:24 | 記錄已執行狀態；或整包重設計 |
| P-3 | Low | `AwaitNoResult` doc 說「避免結果收集開銷」，但 `Run` 一律收集 | parallel_computing.go:66-70 | 修 doc 或真的分開 |
| P-5 | Med | 最常見用法要串三步 `GroupUp(...).Run().AwaitResult()`，業界標竿 `errgroup.Group` 是 `g.Go(f); g.Wait()` 兩步且回傳 error（準則 4、10） | 全套件 | 併入 P-4 決策 |
| P-4 | 決策 | 套件內部只有 `datatable.go`、`mkt/rfm.go`、`stats/anova.go` 三處使用，都是「跑幾個無回傳閉包再等」的用法。標準庫 `sync.WaitGroup` / `errgroup.Group` 已涵蓋。要嘛用 generics 重做成型別安全版本，要嘛標 Deprecated 並把三處改回 WaitGroup | 全套件 | 建議後者：一個公開套件維護成本高於三處 WaitGroup |

### parquet

| 編號 | 嚴重度 | 問題 | 位置 | 建議 |
| --- | --- | --- | --- | --- |
| Q-1 | ~~High~~ 已修正 | `ReadColumnOptions.MaxValues` doc 承諾「超過就回錯避免記憶體爆掉」，但整個套件沒有任何地方讀這個欄位（grep 只命中宣告處）。使用者以為有保護，其實沒有 | parquet/api.go:25-28, 288 | 實作它，或刪掉欄位 |
| Q-2 | Med | `Write(dt, path)` 沒有 ctx、沒有選項；壓縮方式與 chunk size（1Mi）寫死。`Read` 有 ctx，`Write` 沒有，不對稱 | api.go:131-166 | 加 `WriteOptions`（可先空）與 ctx |
| Q-3 | Med | `Write` 直接開目標路徑寫，中途失敗留下半個檔案；同套件 `ApplyCCL` 已做 tmp+rename 原子替換，做法不一致 | api.go:132 vs ccl.go:591-621 | Write 也走 tmp+rename |
| Q-4 | Med | 關閉資源的錯誤用標準庫 `log.Printf`，繞過 insyra `Config` 的 log level 與格式；其他套件都用 `insyra.LogWarning` 等 | api.go, internal.go, ccl.go 多處 | 改用 insyra logger |
| Q-5 | Low | `ApplyCCL` doc 範例引用不存在的 `CCLFilterOptions{}` | ccl.go:573 | 修 doc |
| Q-6 | Low | `FilterWithCCL` / `ApplyCCL` batchSize 寫死 1000，無法調 | ccl.go:456, 577 | 選項或常數說明 |
| Q-8 | Med | 所有函式只吃路徑；Arrow reader 本來就吃 `io.ReaderAt`，卻沒有暴露 `ReadFrom(r io.ReaderAt, size)` / `WriteTo(w io.Writer)`，S3、HTTP、記憶體來源都得先落地（準則 8、10） | api.go 全檔 | 加 Reader/Writer 版本，路徑版包裝它 |
| Q-9 | Low | `Stream` 回傳兩個 channel 是 Go 1.23 之前的寫法；`iter.Seq2[*DataTable, error]` 讓 `for dt, err := range` 直接用，也自然解決 Q-7 的洩漏契約（準則 8） | api.go:246 | 改 `iter.Seq2`，舊簽名保留一版 |
| Q-10 | Low | doc comment 是「Read: read …」冒號風格，不是 Go 的「Read reads …」；`FileInfo`、`ColumnInfo`、`RowGroupInfo` 無 doc（準則 E） | api.go | 補齊 |
| Q-7 | Low | `Stream` 若消費者中途停止讀取又不 cancel ctx，producer goroutine 永久阻塞在 send；doc 沒寫「必須 drain 或 cancel」。記錄不會遺失（unbuffered channel，已推演 close 順序） | api.go:246-286 | doc 明講使用契約 |

### core — 基礎層（version, config, logger, error_buffer, atomic, interfaces, utils, read）

| 編號 | 嚴重度 | 問題 | 位置 | 建議 |
| --- | --- | --- | --- | --- |
| K-1 | High | `LogFatal` 走 `log.Fatalf` 直接 `os.Exit(1)`，全 repo 33 個呼叫點（`isr.DT.From` 讀檔失敗、`lp` 下載 GLPK 失敗、`gplot.SaveChart` 存檔失敗都會殺掉整個程序）。程式庫不得結束宿主程序（準則 10、11）。`SetDontPanic(true)` 後 `LogFatal` 只印 log 就返回，呼叫端拿到 nil 繼續跑，下一行就 nil deref panic，所以「不 panic」是假的 | logger.go:9-25；isr/dt.go、isr/use.go、lp/、plot/radar.go、gplot/save_chart.go | 移除 `LogFatal`；33 處改回傳 `error` 或設 `Err()`。`SetDontPanic` 一併刪除 |
| K-2 | ~~High~~ 已修正 | `SliceToF64` 對非數值填 0，而 `stats.Kurtosis`、`stats.Skewness` 仍經由它讀資料。2026-08-01 已決議 `stats` 不得捏造零值（AGENTS.md follow-up），這條路徑漏掉了 | utils.go:28-38；stats/kurtosis.go:25；stats/skewness.go:26 | `stats` 兩處改走 `numericSeries` 式的拒絕路徑；`SliceToF64` 改回傳 `([]float64, error)` 或標 Deprecated |
| K-3 | High | 沒有 `LogLevelError`。等級只有 Debug/Info/Warning/Fatal，所以實例上 `Err()` 記到的失敗全部是 `LogLevelWarning`，「警告」與「這次操作失敗了」無法區分；`HasErrorAboveLevel(LogLevelWarning)` 也因此沒有意義 | config.go:25-34；datalist.go:2140-2145 | 加 `LogLevelError`，`warn` 改 `fail`；這是資料模型變更，要走 OpenSpec |
| K-4 | Med | 全域錯誤 ring buffer（1536 筆）共 14 個匯出函式：`PopError`、`PopErrorInfo`、`PopErrorAndCallback`、`PopErrorByPackageName`、`PopErrorByFuncName`、`PeekError`、`GetAllErrors`、`GetErrorsByLevel`、`GetErrorsByPackage`、`PopAllErrors`、`HasError`、`HasErrorAboveLevel`、`GetErrorCount`、`ClearErrors`。全程序共用一個緩衝，多 goroutine 下拿到的是別人的錯；`PopError` 空緩衝回 `(LogLevelInfo, "")` 當哨兵，與 `PopErrorInfo` 回 nil 兩套語意並存。業界做法是回傳 error，實例層 `Err()` 已經有了（準則 1、6、10） | error_buffer.go 全檔 | 減到 `GetAllErrors`、`PopAllErrors`、`ClearErrors` 三個，其餘標 Deprecated；或整個 buffer 改為 opt-in |
| K-5 | Med | `pushError` 對每筆錯誤 `go errHandlingFunc(...)`，goroutine 無上限且順序不保證；使用者的 handler 若慢或阻塞，每個 warning 就漏一個 goroutine | error_buffer.go:68-70 | 同步呼叫，或單一 consumer goroutine + 有界 channel |
| K-6 | Med | `Config` 是 `*configStruct` 全域，型別未匯出：使用者無法在自己的函式簽名裡引用它，也不能建立第二份設定；`logLevel`、`coloredOutput`、`dontPanic`、`defaultErrHandlingFunc` 是裸欄位，`SetLogLevel` 與熱路徑 `GetLogLevel` 並行時是 data race（只有 threadSafe、acceleration 用了 atomic） | config.go:9-19, 36-70 | 匯出 `type Config struct`；四個欄位改 atomic 或加 mutex |
| K-7 | Med | `IDataList` 含未匯出方法 `updateTimestamp()`，`IDataTable` 含 `getRowNameByIndex`、`getMaxColLength`、`updateTimestamp`：外部不可能實作，介面等同具體型別。但全 repo 71 個檔案用它當參數型別，回傳卻都是 `*DataList`/`*DataTable`。約 120 個方法的介面沒有抽象價值（「介面越大抽象越弱」，準則 1、8、10） | interfaces.go:6, 121 | 二選一：刪未匯出方法讓它可實作、並拆成小介面；或整個移除改用具體型別。這是 API 設計決策，需要你拍板 |
| K-8 | Med | `AtomicDoAll(f func(), instances ...any)`：型別是 `any`，傳錯型別只 warning 然後「跳過不鎖」，呼叫端以為鎖住了其實在裸奔（準則 8、12） | atomic.go:109-131 | 定義 `type Lockable interface{ atomicActor() *core.AtomicActor }`，參數改 `...Lockable` |
| K-9 | ~~Med~~ 已修正 | `ReadJSON_File` 直接 `json.Unmarshal` 不用 `UseNumber`，整數變 float64；`ReadJSON` 走 `unmarshalJSONRows` 保留 int64。同一個檔案從兩個入口讀，型別不同，大整數 ID 在 `ReadJSON_File` 會失真（準則 6、13） | read.go:410-431 vs 442-460 | `ReadJSON_File` 改成 `os.ReadFile` + `ReadJSON(bytes)` |
| K-10 | Med | `DetectEncoding` 只看前 8KB，且 `utf8.Valid` 在多位元組字元被切在 8192 邊界時回 false，接著交給 chardet 可能判成別的編碼；chardet 回傳的名稱（`shift_jis`、`iso-8859-1`、`gb-18030`）csvxl 只認 big5/gb/utf-16，其餘靜默當 UTF-8 讀 | utils.go:279-322；csvxl/convert.go:213-222 | 邊界回退到最後一個完整 rune 再驗證；不支援的編碼回錯而非靜默 |
| K-11 | Med | CSV 只有 `_File` 與 `_String` 兩種入口，沒有 `io.Reader` 版本；Excel、JSON 同樣只吃路徑。與 C-10、Q-8 同一問題 | read.go | 加 `ReadCSV(r io.Reader, opts)`，檔案與字串版包裝它 |
| K-12 | Med | `ToFloat64`、`ToFloat64Safe`、`ReadSlice2D` 用 `var` 匯出函式值：使用者可以在執行期覆寫（`insyra.ToFloat64 = ...`），godoc 也不會列在 Functions 區；`ReadSlice2D` 與 `Slice2DToDataTable` 是同一件事兩個名字（準則 2、6） | utils.go:22-23；read.go:22 | 改成 `func` 包裝；別名擇一標 Deprecated |
| K-13 | Low | 命名不符 Go 慣例且與 golint 衝突：`ReadCSV_File`、`ReadCSV_String`、`ReadJSON_File`、`ToJSON_Bytes`、`ToJSON_String`、`Dangerously_TurnOffThreadSafety` 用底線；`GetDoesUseColoredOutput`、`GetDontPanicStatus` 疊字；`ParseColIndex`/`CalcColIndex` 一對函式動詞不對稱（準則 9） | config.go；read.go；utils.go:244-251 | v1 前統一：`ReadCSVFile`、`ColIndexToNumber`/`ColNumberToIndex`、`ColoredOutput()` |
| K-14 | Low | `ReadCSV_File(path, bool, bool, encoding ...string)`、`ReadExcelSheet(path, sheet, bool, bool)` 兩個裸 bool 加 variadic；`_WithOptions` 版本已存在，舊簽名該退場。`ReadExcelSheet` 沒有 options 版，也沒有型別推斷（既有 follow-up） | read.go:115, 384 | 舊簽名標 Deprecated；加 `ReadExcelSheetWithOptions` 沿用 `CSVReadOptions` 的欄位 |
| K-15 | Low | 核心套件匯出了與資料表無關的工具：`SqrtRat`、`PowRat`（repo 內無人用）、`SortTimes`（只有 mkt 用一次，`slices.SortFunc` 可替代）、`ProcessData` 回傳 `([]any, int)` 而 int 就是 `len()`、失敗時回 `nil, 0` 靠 log 通知。`F64orRat` 是 internal 介面的匯出別名（準則 1、2） | utils.go:43-87, 90-112, 255-264, 20 | `SqrtRat`/`PowRat`/`SortTimes` 移入 internal 或刪除；`ProcessData` 改回 `([]any, error)` |
| K-16 | Low | `atomic.go` 的 doc comment 是亂碼（`憒??典??蔭??`，來源檔曾以錯誤編碼存檔）；`SetDefaultConfig` 的 doc 寫成「DefaultConfig returns…」；`ReadCSV_FileWithOptions` 用 `"csvxl"` 當套件名寫 log（實際在 core）；`Slice2DToDataTable` 對空切片回錯而 pandas 允許空表（準則 5、E） | atomic.go:31, 37, 44, 49；config.go:97；read.go:139, 51 | 修文件；空切片回空表 |
| K-17 | Low | `LogInfo`/`LogWarning` 等四個 logger 函式是唯一的 log 出口，只能輸出到標準 `log`，不能接 `slog`/zap，也沒有 `io.Writer` 可設；成功路徑（Info）預設開啟，生產環境使用者要自己關（準則 14） | logger.go | 提供 `SetLogger(*slog.Logger)` 或 `SetOutput(io.Writer)`；預設等級改 Warning |

### core — DataList（datalist.go, datalist_*.go, describe_options.go）

實測方式：以拋棄式測試檔在 repo 內執行，結果記在各條「驗證」欄；未實測的標「推論」。

| 編號 | 嚴重度 | 問題 | 位置 | 建議 |
| --- | --- | --- | --- | --- |
| D-1 | ~~High~~ 已修正 | **Bug（已實測）**：`ReplaceLast(old, new)` 在 old 不是 NaN 時，`else if` 分支仍會把最後一個 NaN 換掉。`[5, NaN].ReplaceLast(5, 0)` 得到 `[5, 0]`，5 沒動、NaN 被改了。`replaceFirst_notAtomic` 是對的，兩者不對稱 | datalist.go:353-373 vs datalist_notatomic.go:27-47 | 改成與 replaceFirst 相同的分支結構；補迴歸測試 |
| D-2 | ~~High~~ 已修正 | **資料半毀（已實測）**：`Normalize`、`Standardize`、`ClearOutliers`、`Difference`、`FillNaNWithMean` 用 `conv.ParseF64` 原地改寫，遇到非數值 panic 再 recover。`[1, "x", 3].Normalize()` 回傳 nil，且資料已變成 `[0, "x", 3]`。沒有回滾，沒有原子性（準則 11、13）。lock 本身有用 defer 釋放，不會死鎖（已實測） | datalist.go:733-843, 1864-1890 | 先掃描全部可轉換再寫入；失敗時資料不動、設 `Err()`、回傳自身 |
| D-3 | High | 失敗時有三種回傳形狀：回 **nil**（`Normalize`、`MovingAverage`、`WeightedMovingAverage`、`ExponentialSmoothing`、`DoubleExponentialSmoothing`、`MovingStdev`、`Difference`、`Diff`、`PctChange`）、回**空 list**（`Sample`、`SampleFrac`、`Shuffle`、`Map`、`Describe`、所有 Rolling/EWM/Expanding reducer）、回**自身 + Err()**（其餘）。`Err()` 的 doc 說「不打斷鏈式呼叫」，回 nil 就是打斷：`dl.MovingAverage(0).Sort()` 直接 nil deref panic（準則 6、7、11） | 上列各函式 | 統一為「回傳自身或同長度 nil list，錯誤放 Err()」；nil 回傳全部移除 |
| D-4 | ~~High~~ 已修正 / Med | 非數值元素的待遇不一致。**High**：`Rank`、`ExponentialSmoothing`、`DoubleExponentialSmoothing`、六個 `*Interpolation` 走 `ToF64Slice`，非數值被當成 0 參與計算：`[3, "b", 1].Rank()` = `[3, 1, 2]`（已實測），"b" 排第一。**Med**：`Sum/Mean/Max/Min/Median/Var/...` 跳過非數值後給出數字（`["a", 1, 2].Mean()` = 1.5，已實測）。跳過本身是合理慣例（R 回 NA 加警告，pandas raise，跳過是第三種），問題在同一份資料有四種待遇：`Mean` 跳過、`Rank` 當 0、`Normalize` 半毀、`FillWithMean` 整欄拒絕；而且跳過只留一筆 Warning 在被 D-5 塞滿雜訊的 `Err()` 裡，等於沒通知（準則 6、13） | datalist.go:1051-1095, 930-980, 1148-1860；datalist_interpolation.go 全檔 | 「當 0」的九個方法改成失敗或跳過；「跳過」保留但寫成一條全套件規則並進 doc，同時處理 D-5 讓 Warning 可見 |
| D-5 | Med | 正常結果被記成錯誤：`Get` 越界、`FindFirst/FindLast` 找不到、`FindAll`/`Count` 在空 list（已實測 `Count` 讓 `Err()` 非 nil）、`Sort`/`Pop`/`Map`/`ToF64Slice` 在空 list 都呼叫 `warn`。呼叫端無法用 `Err() != nil` 判斷「真的失敗」（準則 11） | datalist.go:148-160, 248-327, 441-456, 1007-1012 | 「找不到」「空」不設 Err；只有無效參數與型別錯誤才設 |
| D-6 | Med | `FindFirst`/`FindLast` 回傳 `any`（int 或 nil），呼叫端要型別斷言；`Get` 越界回 nil，與元素本身是 nil 無法區分。DataTable 的 `GetRowIndexByName` 已用 `(int, bool)`，同一套件兩種慣例（準則 6、8） | datalist.go:148, 248, 274 | 改 `(int, bool)` / `(any, bool)`；舊簽名標 Deprecated |
| D-7 | Med | 原地修改與回傳新 list 從名稱看不出來。原地：`Sort`、`Reverse`、`Normalize`、`Standardize`、`Clear*`、`Replace*`、`Fill*`、`Upper/Lower/Capitalize`、`Parse*`、`Drop*`。新 list：`Filter`、`Map`、`Concat`、`Rank`、`Shift/Diff/PctChange/Cum*`、`MovingAverage`、`Sample`、`Shuffle`。pandas 預設回新物件；使用者對 `GetCol` 拿到的欄位做 `Sort()` 會不會動到表，要看 DataTable 段（準則 5、6） | 全 DataList | 文件明列兩類；長期考慮 `Sorted()`/`SortInPlace()` 命名或全部回新 list |
| D-8 | Med | variadic 冒充選填參數：`Sort(ascending ...bool)`、`Rank(ascending ...bool)`、`FillForward/FillBackward(limit ...int)`、`FillByInterpolation(extrapolate ...bool)`、`Shift(periods, fill ...any)`、`Sample(..., options ...SamplingOptions)`、`Describe(options ...DescribeOptions)`。`Sort` 多傳一個參數只在執行期 warn。同一型別上 `Rolling(RollingOptions)`、`EWM(EWMOptions)` 已是 options struct，新舊並存（準則 8） | 各函式 | 新 API 一律 options struct；舊的標 Deprecated |
| D-9 | Med | 效能：`ClearNaNs`/`ClearNils`/`ClearOutliers` 在迴圈裡 `append(data[:i], data[i+1:]...)`，O(n²)；`DropAll`/`ClearStrings` 為一次線性掃描開 NumCPU 個 goroutine，10 個元素也開，且 `ClearStrings` 留著「此處之後尚未提升性能」註解；`Data()` 每次整份複製，`Get(i)` 每個元素取一次 actor lock，沒有 iterator。逐元素走訪一個 DataList 要嘛 n 次鎖、要嘛全複製（準則 8、12；Go 1.23 `iter.Seq` 是標準做法） | datalist.go:479-571, 615-731, 41-48, 148 | 單趟過濾；加 `All() iter.Seq2[int, any]` |
| D-10 | Med | `WeightedMean(weights any)`、`WeightedMovingAverage(w int, weights any)` 收 `any` 再用 `ProcessData` 猜型別；`RollingOptions.Weights []float64` 是型別化的。同一件事兩種簽名（準則 8） | datalist.go:880, 1292 | 改 `[]float64` |
| D-11 | Med | 相等語意有三套：`IsEqualTo`/`FindAll`/`Count`/`ReplaceAll` 用 `==`（NaN 永不相等，已實測 `IsEqualTo` 自己的 Clone 為 false；元素不可比較時 panic）；`Counter()` 用 map key（同樣 panic）；`FillWithMode` 用 `reflect.DeepEqual`（O(n²)）。`FindAll` 有特判 NaN，`IsEqualTo` 沒有（準則 6、13） | datalist.go:1892-1918, 191-199；datalist_impute.go:182 | 一個套件內部 `equalAny` 統一 NaN 與不可比較型別的處理 |
| D-12 | Med | 時間戳用 Unix 秒：`GetCreationTimestamp() int64`、`updateTimestamp` 同一秒內多次修改看不出來、`IsTheSameAs` 比秒級時間戳，同一秒建立的兩個 list 會被判「相同」。業界回 `time.Time`（準則 10） | datalist.go:2064-2087, 1920 | 改 `time.Time`（或 UnixNano）；`IsTheSameAs` 語意重新定義 |
| D-13 | Med | 同一個概念三種尺度：`Quartile(q int)` 用 1..3 魔數、`Percentile(p)` 用 0..100、`DescribeOptions.Percentiles` 用 0..1（pandas `quantile` 用 0..1）。`Mode()` 全部頻率相同時回 nil，pandas 回全部值（準則 5、6） | datalist.go:1735, 1816, 1431；describe_options.go:13 | 加 `Quantile(p float64)` 0..1 為主 API；`Mode` 行為寫進文件或對齊 pandas |
| D-14 | Med | 重複介面：`Difference` vs `Diff(1)`、`FillNaNWithMean` vs `FillWithMean`、`MovingAverage(w)` vs `Rolling({Window:w}).Mean()`、`MovingStdev` vs `Rolling().Std()`、`ExponentialSmoothing(a)` vs `EWM({Alpha:a}).Mean()`、`WeightedMovingAverage` vs `Rolling({Weights}).Mean()`。舊的一組正是 D-3/D-4 問題最集中的地方；新的一組（Rolling/EWM/Expanding）設計品質明顯較好（準則 1、6） | datalist.go:821-1005, 1864 | 舊六個標 Deprecated 指向新 API，下一個 minor 移除 |
| D-15 | Low | 小行為不一致：`InsertAt` 越界改成 append（pandas raise）；`Update` 的 Err 記成 `ReplaceAtIndex`（函式名錯）；`Sample`/`Shuffle` 把結果改名 `name_Sampled`/`name_Shuffled`，`Filter`/`Concat` 丟掉名稱，`Shift/Diff/Cum*` 保留名稱。名稱傳遞規則沒有一致（準則 6、7） | datalist.go:221, 203；datalist_sampling.go | 統一：衍生 list 一律沿用原名 |
| D-16 | Low | `ParseNumbers` 把所有元素轉成 float64，int64 大於 2^53 會失真，與 CSV 讀入保留 int64 的決策矛盾；`Capitalize` 寫死 `language.English`（準則 13） | datalist.go:1977, 1132 | 整數保留整數；語言由參數決定或用 `language.Und` |
| D-17 | Low | 文件：`Len` 無 doc；`ToF64Slice` doc 重複一行且沒寫「非數值變 0」（既有 follow-up）；`MovingStdev` doc 寫 `MovingStdDev`；`Drop` doc 寫「Returns an error」但回傳 `*DataList`；`Rolling*`/`EWM*` 的 doc 是本套件寫得最完整的，其餘應比照（準則 E） | datalist.go:606, 2015-2019, 982, 456 | 修 |
| D-18 | Low | Rolling/EWM/Expanding 選項無效時，reducer 回傳「空 list」而不是「同長度全 nil」。使用者把結果 `AppendCols` 進表會因長度不符再錯一次，離真正原因更遠（準則 7） | datalist_window.go:301-306；datalist_ewm.go:107-115 | 回同長度全 nil |
| D-19 | Med | `init.go` 在 import 時就以 Info 印出「Welcome to Insyra」橫幅，且用 `LogInfo("", "", …)`。程式庫不得在 import 時輸出（準則 14；併入 K-17 的 logger 決策） | init.go:7 | 移除橫幅，改由 CLI 自己印 |

### core — DataTable 第一批（datatable.go、colname/rowname/name/colindex、swap/sort/map/json/csv、summary/describe、filters、replace、sampling、window、resample、groupby、pivot、merge）

實測方式同 DataList；「已實測」表示以拋棄式測試跑過。

| 編號 | 嚴重度 | 問題 | 位置 | 建議 |
| --- | --- | --- | --- | --- |
| T-1 | High | **panic（已實測）**：`GetElementByNumberIndex(0, 5)` 對不存在的欄直接 index out of range；`SetRowToColNames(99)`、`SetColToRowNames("ZZ")` 取回 nil 後解引用 nil pointer。三個公開方法對錯誤輸入 panic 而不是設 `Err()`（準則 11） | datatable.go:280-292, 508-540 | 加邊界檢查與 nil 檢查 |
| T-2 | High | **內部資料外洩（已實測）**：`Data()`／`ToMap()` 直接回傳 `col.data`，呼叫端改 map 裡的 slice 就改到表內部，且繞過 actor lock。DataList 的 `Data()` 是 copy-on-read，同一套件兩種契約（準則 6、12） | datatable.go:1232-1267 | 逐欄 `slices.Clone` |
| T-3 | High | **資料遺失（已實測）**：`AppendRowsByColIndex(map{"Z": 42})` 在兩欄表上：Z 不存在時「新增一欄」卻是加在尾端變成 C，再用 Z 重新解析仍越界，42 被丟掉，只留下一列全 nil | datatable.go:161-217 | 補齊到目標索引，或回錯 |
| T-4 | High | **Filter 結果與原表共用欄位（已實測）**：`FilterColsBy*`（7 個）、`FilterCols`、`FilterColsByColNameContains` 回傳的新表直接放入 `dt.columns[i]` 指標與 `dt.rowNames` 指標。改過濾結果會改到原表，兩個表各自的 actor lock 互不知情，是 data race（準則 12）。`Filter`／`FilterRows`／`FilterRowsByRowNameContains` 有複製，同檔案兩種做法 | datatable_filters.go:12-190, 386-440 | 一律 Clone 欄與 rowNames |
| T-5 | Med | **panic（已實測）**：找不到欄或列時 Filter 系列回傳 `&DataTable{}`（rowNames 為 nil），之後呼叫 `GetRowIndexByName`、`SwapRowsByName` 等直接 nil deref；`FilterRows`／`FilterCols` 以第一欄長度當列數且不檢查邊界，jagged 表 index out of range | datatable_filters.go 多處 | 回 `NewDataTable()`；用 `getMaxColLength` 並用 `cellAt` |
| T-6 | Med | **Bug（已實測）**：`DropRowsByIndex(-1, 0)` 只刪最後一列，第 0 列沒刪（先排序再換算負索引，adjusted 變成 -1 被跳過）；`DropRowsByIndex(1, 1)` 重複索引刪掉兩列（[0,1,2,3] 變 [2,3]） | datatable.go:959-983 | 先正規化負索引、去重、再由大到小刪 |
| T-7 | Med | **Bug（已實測）**：`DropColsContainNumber`／`DropRowsContainNumber` 只認 `int` 與 `float64`，CSV 推斷出的 `int64` 欄不算數字，整欄留下 | datatable.go:826-857, 1049-1086 | 改用 `IsNumeric` |
| T-8 | Med | **Bug（已實測）**：`Transpose` 只把前 ncols 個列名搬成欄名（迴圈變數是舊欄索引），3 列 2 欄的表轉置後第 3 個列名遺失；而且是原地轉置又回傳自己，doc 沒說（pandas `.T` 回新表） | datatable.go:1341-1382 | 迴圈改用列數；doc 標明 in-place |
| T-9 | Med | **Bug（已實測）**：`ChangeRowName("b", "a")` 在 "a" 已存在時，BiIndex 把 "a" 從第 0 列移到第 1 列，第 0 列悄悄失去名字。其他 setter 都走 `safeRowName`，這個沒有 | datatable_rowname.go:111-129 | 走 `safeRowName` 或回錯 |
| T-10 | Med | `Mean() any`：回傳 `any`（永遠是 float64），分母用 rows×cols 含非數值與 nil 格子：`[2,"x"],[4,nil]` 得 1.5（6/4），是靠分母捏造（已實測） | datatable.go:1324-1338 | 回 float64，只數數值格 |
| T-11 | Med | `GetCol(index)` 先 `ToUpper` 再退回名稱查詢，`GetCol("price")` 找不到名為 price 的欄（已實測，專案記憶已有此陷阱）；`ReplaceInCol("a", …)` 把名稱 "a" 當成 Excel 索引 A（已實測），欄名 "b" 若在第 0 欄會改到第 1 欄。名稱與索引共用一個 string 參數是整個 DataTable 的結構性歧義（準則 3、6） | datatable.go:297-317；datatable_replace.go:361 | `GetCol` 不退回名稱；長期：索引用 typed `ColIndex`，名稱用 `ByName` |
| T-12 | Med | `ToJSON_Bytes`／`ToJSON_String` 遇到 NaN 回 nil／空字串只設 Err（已實測），呼叫端拿到空 JSON 不會察覺；`ToCSV` 用 `%v` 輸出 `time.Time` 成 `2024-01-02 03:04:05 +0000 UTC`，`ParseDates` 預設 layout 讀不回來，CSV 往返壞掉（已實測）；`ToCSV(path, bool, bool, bool)` 三個裸 bool 且無 `io.Writer` 版本 | datatable_json.go:85-105；datatable_csv.go:13 | JSON 回 error；CSV 時間用 RFC3339；加 options struct 與 `WriteCSV(w io.Writer)` |
| T-13 | Med | `Filter(func(row, col, value) bool)` 與 `FilterRows` 是「任一格子符合就留整列」，不是列謂詞。最常見的 `A > B` 這種跨欄條件無法表達，只能繞去 CCL；`FilterByCustomElement` 與 `Filter` 重複（準則 4、5） | datatable_filters.go:333-440 | 加 `FilterRowsWhere(func(row *DataList) bool)` |
| T-14 | Med | `SetColNames` 給的名字比欄多時自動新增空欄（已實測），pandas 是長度不符即 raise；`AppendCols` 遇同名自動改成 `name_1` 不通知 | datatable_colname.go:163-185；datatable.go:69 | 長度不符回錯；同名至少 warn |
| T-15 | Med | `mergeVertical` 對沒有欄名的表（`NewDataTable(NewDataList(...))` 預設）判定「重複欄名 ""」而回錯，兩張無名表無法垂直合併（推論，未實測）；`Merge(other IDataTable, ...)` 內部立刻斷言 `*DataTable`，介面參數只是裝飾（K-7） | datatable_merge.go:31, 389-400 | 無名欄以位置對齊；參數改 `*DataTable` |
| T-16 | Med | GroupBy 的 `columnsSnapshot` 是欄位指標的淺拷貝，`Aggregate` 在鎖外讀 `sourceCol.data`；父表被並行修改時是 data race（程式碼註解自己承認）。與 Rolling／EWM 深拷貝快照的做法不一致 | datatable_groupby.go:139-146 | 深拷貝或在 Aggregate 期間持鎖 |
| T-17 | Med | 聚合相關 API 三種寫法：`Aggregate` 用 typed `AggregateOp`，`Pivot.AggFunc` 用字串（含 "avg"、"std" 別名），`Resample` 用 `AggregateOp`。GroupBy 的 key 把 `int 1` 與 `float64 1.0` 分成兩組（CSV 讀進來的 int64 與手動建的 float 會分家），pandas 視為同一組（準則 5、6） | datatable_pivot.go:44, 545-580；datatable_groupby.go:210 | Pivot 改收 `AggregateOp`；數值 key 正規化 |
| T-18 | Med | 效能：`Count` 為了加總各欄用 `asyncutil.ParallelForEach` 再經 float64 `Sum` 轉回 int；`Clone` 用 `parallel.GroupUp` 跑兩件小事；`Map` 每格經 `originalCol.Get`（每格一次鎖）；`containsSubstring` 手寫遞迴，長字串遞迴深度等於字串長度，`strings.Contains` 就有 | datatable.go:1271-1282, 1384-1410, 1568-1571；datatable_map.go:30 | 直接迴圈；`strings.Contains` |
| T-19 | Med | `FindColsIfContains`／`FindColsIfContainsAll` 用 `FindFirst != nil` 判斷，每個不含該值的欄都會觸發一次 warn 進 `Err()`（D-5 跨欄放大）；`FindRowsIfAllElementsContainSubstring` 把非字串格子視為「符合」，全數字的列會被當作符合（準則 13） | datatable.go:652-690, 626-650 | 內部用不設 Err 的查找；非字串視為不符 |
| T-20 | Low | `replace` 系列的 `mode ...int` 用 0/1/-1 魔數當 variadic 選項；`ReplaceInCol` doc 說「index or name」實作只吃索引（T-11）；NaN 判定 InRow/InCol 只認 float64，表層版用 `isNilOrNaN` 認 float32（準則 8、E） | datatable_replace.go 全檔 | typed `ReplaceMode`；統一 `isNilOrNaN` |
| T-21 | Low | 13 個 `FilterColsByColIndexGreaterThan…`／`FilterRowsByRowIndexLessThanOrEqualTo…` 長名方法做的是切片，pandas 是 `iloc[a:b]`；`Headers`／`SetHeaders` 是 `ColNames`／`SetColNames` 的別名；`Counter` 與 DataList 重複；`SimpleRandomSample` 已 Deprecated（準則 1） | datatable_filters.go；datatable_colname.go:159, 187 | 收斂成 `SliceRows(from, to)`／`SliceCols(from, to)`，舊的標 Deprecated |
| T-22 | Low | 缺 doc：`NewDataTable`、`GetElementByNumberIndex`、`GetColByNumber`、`GetColByName`、`GetRowByName`、`NumRows`、`NumCols`、`Data`、`GetCreationTimestamp`、`GetLastModifiedTimestamp`、colname.go 前 6 個方法。`AppendRowsByColIndex` doc 標題寫成 `AppendRowsByIndex`；`ToJSON_Bytes` 的 Err 記成 `ToJSON_Byte`（準則 E） | 各檔 | 補 |
| T-23 | Low | `SortBy` 的 `DataTableSortConfig` 零值 `ColumnNumber: 0` 無法與「沒指定」區分，空 config 會默默用第 0 欄排序；找不到欄時只 `LogWarning` 不設 Err | datatable_sort.go:7-40 | `ColumnNumber` 改 `*int` 或加 `HasColumnNumber` |
| T-24 | OK | 設計較好、可當範本的部分：`Resample` 回傳 `error`；`Pivot`／`Unpivot` 同時回 error 與設 `Err()`；`GroupBy`／`Aggregate` 的 options struct 與 `AggregateOp.String()`；`SamplingOptions` 有 seed；`SummaryTo(io.Writer)`；Rolling／EWM 的表層包裝。這些是 v1 API 該長的樣子 | — | — |

### core — DataTable 第二批（encode、scale、simple_imputer、impute、to_sql、from_sql、ccl、show）

| 編號 | 嚴重度 | 問題 | 位置 | 建議 |
| --- | --- | --- | --- | --- |
| E-1 | Med | 四個 CCL 方法用 `defer recover()` 把 panic 轉成 warn，但函式回傳值是未命名的 `*DataTable`，recover 後回傳 **nil**（`return <-resultDtChan` 不會執行）。CCL 引擎任何 panic 都讓 `dt.AddColUsingCCL(...).Show()` 變成 nil deref；`resultDtChan` 在同步的 `AtomicDo` 裡毫無作用（推論，未實測） | datatable_ccl.go:13-60, 80-146, 147-235 | 命名回傳值並在 recover 中設為 dt；或不 recover，讓引擎錯誤走 error |
| E-2 | Med | `ExecuteCCL` 多條語句在第 n 條失敗時，前 n-1 條已套用，表處於半改狀態，doc 沒說；四個 CCL 方法只用 `Err()` 回報，沒有 error 回傳，CLI 與 parquet 的 CCL 都拿 error，同一功能兩種契約（準則 6、11） | datatable_ccl.go:147-235 | 先編譯全部再套用，或在 doc 明講「逐條套用」；加 `ExecuteCCLErr` 回 error |
| E-3 | Med | 核心套件直接依賴 `gorm`：`ToSQL(db *gorm.DB, …)`、`ReadSQL(db *gorm.DB, …)`。所有只用 DataTable 的使用者都要拉進 gorm 及其相依圖；標準做法是收 `*sql.DB`／`database/sql` 介面，gorm 當可選轉接（準則 8、10） | datatable_to_sql.go:48, 57；datatable_from_sql.go:546-601 | 改收 `*sql.DB`（或 `interface{ QueryContext; ExecContext }`），gorm 使用者傳 `db.DB()` |
| E-4 | Med | `ReadSQLOptions.WhereClause`／`OrderBy` 是直接拼進 SQL 的原始字串，`Params` 只綁 `Query`；使用者把資料塞進 WhereClause 就是 SQL injection，doc 沒有警語（準則 14）。對照組：`ToSQL` 端有做識別字引號與型別白名單 | datatable_from_sql.go:500-545, 674-712 | doc 標明「不得放入使用者資料」；或提供 `Where(expr, args...)` 綁參數 |
| E-5 | Low | 選項重複與過時註解：`ReadSQLOptions.IndexCol` 是 `RowNameColumn` 的別名；`ToSQLOptions.IfExists` 註解寫 `"fail", "replace", "append"` 字串但型別是 int enum；`SQLActionIfTableExistsFail` 命名冗長（準則 1、E） | datatable_from_sql.go:500-509；datatable_to_sql.go:20-43 | 留一個；修註解；enum 改 `TableExistsFail` |
| E-6 | Low | `NewSimpleImputer(strategy, constant ...any)`：常數用 variadic 傳，數量錯誤要到 `Fit` 才報；`Scaler.Fit(dt, cols ...string)` cols 必填卻是 variadic，零個參數是執行期錯誤（準則 8） | datatable_simple_imputer.go:39；datatable_scale.go:169 | `NewConstantImputer(value)`；cols 改 `[]string` |
| E-7 | Low | `DataTable.FillForward(limit int, cols …)` 與 `DataList.FillForward(limit ...int)` 簽名不對稱；`FillWithMean`／`FillWithMedian`／`FillByInterpolation` 對非數值欄靜默跳過（不 warn），pandas 會填所有欄（準則 6） | datatable_impute.go:41-96 | 對稱簽名；跳過時至少 warn |
| E-8 | Low | `Show()` 預設印全部列，百萬列的表會灌爆終端（pandas 預設 60 列並省略中段）；`ShowRange(startEnd ...any)` 用 `any` 收 `(5)`、`(-5)`、`(2, 10)`、`(2, nil)` 四種形狀；`Show(label, object showable, …)` 的參數型別 `showable` 未匯出，使用者無法在自己的函式簽名引用（準則 4、8） | show.go:27-70, 713-750 | 預設 head/tail 截斷；`ShowRange(start, end int)` + `Head(n)`／`Tail(n)`；匯出 `Showable` |
| E-9 | OK | 設計良好、可當範本：encode 三件組（options struct、typed policy enum、fitted encoder 有 `Transform`／`InverseTransform`／`Options()`、錯誤全部回 error、輸出欄名碰撞偵測）；四個 Scaler 共用仿射核心、`Params()` 可檢視、compile-time 介面檢查；`SimpleImputer` 明確不提供 InverseTransform 並寫出理由；`ToSQL` 的識別字引號與型別白名單；`ReadSQLStream` 把 goroutine／連線洩漏契約寫進 doc（parquet.Stream 應比照） | — | — |

### isr

| 編號 | 嚴重度 | 問題 | 位置 | 建議 |
| --- | --- | --- | --- | --- |
| I-1 | High | `isr` 是 README 建議的入口，卻是 `LogFatal` 最密集的地方：`DT.From` 讀 CSV／Excel／JSON 失敗、`Col`／`Row`／`Push` 型別不對、`UseDL`／`UseDT` 型別不對，全部直接結束程序（K-1 的使用者端表現）。檔案不存在這種最常見的錯誤沒有任何回復手段 | isr/dt.go:48-155, 178, 193, 268-318；isr/use.go:15, 30 | 走 `Err()` 或提供 `TryFrom(...) (*dt, error)` |
| I-2 | Med | `DT.From`／`DL.From` 回傳 `*dt`／`*dl`，型別未匯出。使用者無法寫 `func f(t *isr.dt)`，表在自己的函式間傳遞只能退回 `.DataTable`，語法糖到函式邊界就失效；golint 也把「匯出函式回傳未匯出型別」列為錯誤（準則 3、8、9） | isr/dl.go:14；isr/dt.go:13 | 匯出為 `isr.DL`／`isr.DT` 型別（現有的 `DL`／`DT` 是 var，要改名） |
| I-3 | Med | 型別安全交給執行期：`From(item any)` 是 20 分支的 type switch，`Row`／`Col` 是 `map[any]any`，`At(row any, col any)`，`Name("x")` 回傳未匯出的 `name`。編譯器擋不住任何誤用，錯了就是 I-1 的 Fatal（準則 8） | isr/dt.go:20-28, 41-155, 197-235；isr/name.go | 分成 `FromCSV(CSV)`、`FromRows(Rows)` 等具名建構子 |
| I-4 | Low | 命名：`CSV_inOpts`、`CSV_outOpts`、`Excel_inOpts` 底線；`FirstCol2RowNames` 用 `2` 代 To；`Of` 是 `From` 別名、`Push` 是 `AppendCols` 別名；`DLs = []insyra.IDataList` 混用介面（K-7）（準則 1、9） | isr/csv.go；isr/excel.go；isr/dl.go:10, 38；isr/dt.go:159 | v1 前統一 |
| I-5 | Low | `Pivot`／`Unpivot` 把底層 error 丟掉只留 `Err()`（doc 有寫）；`CSV.OutputOpts` 存在但 `From` 用不到，沒有對應的輸出函式 | isr/pivot.go:31-44；isr/csv.go:19 | 保留 error 版本；刪或實作 OutputOpts |
| I-6 | OK | `groupby.go`／`window.go`／`pivot.go` 三個新檔是好的包裝：一對一轉發、doc 指回底層語意、`Rolling` 選項 struct | — | — |

### stats

| 編號 | 嚴重度 | 問題 | 位置 | 建議 |
| --- | --- | --- | --- | --- |
| ST-1 | High | **錯誤的顯著性（已實測）**：`SingleSampleTTest`、`TwoSampleTTest`、`SingleSampleZTest`、`TwoSampleZTest`、`FTestForVarianceEquality`、`BartlettTest`、`LeveneTest` 用 `Len()` 當 n，卻用 `Mean()`／`Stdev()`／`Var()`／`Median()` 算統計量，而這些方法會跳過非數值格子（D-4）。`[1, 2, nil, 3]` 的單樣本 t 檢定：n=4、mean 用 3 個值算，得 t=4.00、p=0.028；正確是 t=3.46、p=0.074。一個空白把不顯著變成顯著。v0.3.1 的 changelog 說 `stats` 已拒絕不可讀值，但這七個檢定沒有走 `numericValues`，`PairedTTest`、ANOVA、非參數檢定則都有拒絕（準則 13） | stats/ttest.go:30-125, 129-215；stats/ztest.go 全檔；stats/ftest.go:18-60, 64-180 | 全部改走 `numericSlice`，與其他檢定一致 |
| ST-2 | High | `CalculateMoment(dl, n, central)` 對 n≥3 用 `ToF64Slice`，非數值當 0；`Skewness`／`Kurtosis` 已在本輪修正前置驗證，但這個公開函式直接餵 DataList 仍會捏造（K-2 同族） | stats/moments.go:35-100 | 改走 `numericSlice` |
| ST-3 | Med | 每個雙樣本函式都 `data1.(*insyra.DataList)` 未檢查型別斷言：介面參數 `IDataList` 只是裝飾，傳入其他實作直接 panic。K-7 的證據最集中處（10 餘處） | stats/ttest.go:136-137, 226-227；ztest.go；ftest.go:22-23；correlation.go:365-366, 398-399；nonparam_mwu.go | 參數改 `*insyra.DataList`，或斷言失敗回錯 |
| ST-4 | Med | 同一族 API 的簽名不一致：t 檢定 `confidenceLevel ...float64` 選填、z 檢定 `confidenceLevel float64` 必填；z／Wilcoxon／MWU 有 `alternative`，t 檢定沒有（只能雙尾，R 的 `t.test` 有）；`LeveneTest(groups []IDataList)` 用切片、`OneWayANOVA(groups ...IDataList)` 用 variadic；`KMeans(…, opts ...KMeansOptions)` 與 `FactorAnalysis(dt, opt FactorAnalysisOptions)` 一個選填一個必填（準則 6、8） | ttest.go；ztest.go；ftest.go:64；anova.go:63；clustering.go:104；factor_analysis.go:347 | 統一為 options struct；t 檢定加 `Alternative` |
| ST-5 | Med | 結果型別的共同部分 `testResultBase` 未匯出：`Statistic`、`PValue`、`DF`、`CI`、`EffectSizes` 被提升可用，但使用者無法寫一個接受「任何檢定結果」的函式，也無法用介面判斷；`TTestResult.Mean *float64` 與 `ZTestResult.Mean float64` 選填欄位一個用指標一個不用（準則 3、6） | stats/structs.go；ttest.go:13；ztest.go:10 | 匯出 `TestResult` 基底或定義 `interface{ Stat() float64; P() float64 }` |
| ST-6 | Med | `ChiSquareTestResult.ContingencyTable` 的每格是 `[2]float64{observed, expected}` 陣列塞進 DataTable，全套件沒有其他 API 能處理這種格子，`Show` 印出 `[5 4.5]`；`ChiSquareGoodnessOfFit` 的 `p` 依「類別字串排序後」的順序對位，doc 自己標了 IMPORTANT 警告（準則 5、7） | stats/chi_square.go:14-19, 60-70 | 拆成 `Observed`、`Expected` 兩張表；GoF 改收 `map[string]float64` |
| ST-7 | Med | `TwoWayANOVA(aLevels, bLevels int, cells ...IDataList)` 要使用者自己按 row-major 排 a×b 個 cell，pandas／R 收長格式加因子欄；`RepeatedMeasuresANOVA(subjects ...)`、`FriedmanTest(subjects ...)` 每個受試者一個 list，同樣不是資料表的自然形狀（準則 5） | stats/anova.go:112, 253；nonparam_friedman.go:36 | 加收 `(dt, valueCol, factorCols...)` 的長格式入口 |
| ST-8 | Low | 效應量正負號：t 檢定保留方向（註解說 paired 已修），z 檢定用 `math.Abs` 丟掉方向；`SingleSampleTTest` 常數資料回 NaN／Inf 統計量與 p=0 沒有寫進 doc（準則 6、E） | stats/ztest.go:57, 122；ttest.go:76-108 | 統一保留方向；補 doc |
| ST-9 | Low | `Show()` 只在 `ChiSquareTestResult` 與 `FactorAnalysisResult` 上有，其餘結果型別沒有，也沒有 `io.Writer` 版本；`FactorAnalysisResult` 15 個 `IDataTable` 欄位（K-7）；`Diag(x any, dims ...int) (any, error)` 進出都是 `any`（準則 8） | chi_square.go:21；factor_analysis.go:180；diag.go:11 | 統一 `String()`；Diag 拆成 `DiagOf(*mat.Dense)`／`DiagMatrix([]float64)` |
| ST-10 | OK | 做得好的部分：regression／GLM／clustering／KNN／PCA／non-parametric 全部先驗證輸入再計算、回 error、結果 struct 欄位齊全且對 R 驗證；`numericinput.go` 的說明是本專案最清楚的設計文件之一；`RegisterKNNDeviceSearcher` 讓 accel 反向掛入而不讓 stats 依賴 accel | — | — |

### quant

| 編號 | 嚴重度 | 問題 | 位置 | 建議 |
| --- | --- | --- | --- | --- |
| QU-1 | Low | enum 風格：`VaRMethod`、`OptionType`、`PortfolioObjective` 用 `uint8`，`stats` 用 string，core 用 int；同一個程式庫三種 enum 寫法，printf `%d` 出來的錯誤訊息（`unknown method 3`）也不如 string 可讀（準則 6、9） | quant/risk.go:16；options.go:11；portfolio.go:12 | 全庫統一為 typed int + `String()`，或 string |
| QU-2 | Low | 參數型別 `insyra.IDataList`／`IDataTable`（K-7）；`CAPM`／`Beta` 先 `asset.Len()` 比長度再 `numericSeries`，nil 檢查與長度檢查在 `numericSeries` 之前重複實作（各函式自己寫一次） | quant/capm.go:40-70 | 改具體型別；長度檢查併入 helper |
| QU-3 | Low | `PercentileBands(paths, percentiles []float64)` 的百分位尺度要與 D-13（0..1 vs 0..100）一起統一；`WalkForward[P any]` 用索引區間回呼，使用者要自己切資料，沒有收 DataTable 的版本（準則 4） | quant/bootstrap.go:217；walkforward.go | 文件標明尺度；加 DataTable 版 |
| QU-4 | OK | 範本等級：每個函式先 `numericSeries` 拒絕不可讀值並指出列號、全部回 error、doc 寫清單位（per-period vs annualized、calendar days）、`BootstrapConfig.Seed` 語意明確、`PortfolioConfig` 只預設容忍度其餘一律驗證、`Converged=false` 不當錯誤。其他套件應以此為準 | — | — |

### mkt

| 編號 | 嚴重度 | 問題 | 位置 | 建議 |
| --- | --- | --- | --- | --- |
| MK-1 | High | **panic（已實測）**：`RFM` 用 `conv.ParseF64` 讀金額欄，遇到 `"abc"` 直接 panic 穿出 `AtomicDo`，整個程序崩潰。三個公開函式（`RFM`、`CustomerActivityIndex`、`BasketAnalysis`）失敗時只 `LogWarning` 後回 nil，沒有 error 回傳、沒有 `Err()`；欄名打錯時 `GetColIndexByName` 回空字串，之後每列 `GetElement(i, "")` 都是 nil，結果是「一張空表、零錯誤」（準則 11） | mkt/rfm.go:26-80, 100；cai.go:38-60；basket.go:38-56 | 三個函式改回 `(result, error)`；金額走 `ToFloat64Safe` 並指出列號 |
| MK-2 | Med | 輸出列順序來自 Go map 迭代（`for customerID := range customerLastTradingDayMap`），每次執行 RFM／CAI 的列順序都不同，結果不可重現、無法 diff；`BasketAnalysis` 有排序（準則 13） | rfm.go:236；cai.go:180 | 依 CustomerID 排序輸出 |
| MK-3 | Med | 每個欄位都提供 `XxxColIndex` + `XxxColName` 兩個欄位（三個 config 共 8 對），「同時給時 index 優先」是把 T-11 的歧義寫進設定檔；`DateFormat` 用自訂的 `"YYYY-MM-DD"` 記法再轉 Go layout，`NumGroups uint`；`var CAI = CustomerActivityIndex` 是可被覆寫的函式變數（K-12）（準則 3、6、8） | mkt/rfm.go:12-22；cai.go:12-22；basket.go:12-17 | 只留一個欄位參照（名稱或索引擇一）；`CAI` 改 func |
| MK-4 | Low | 預設值套用時以 Info 等級 log（DateFormat、TimeScale），噪音；用 `parallel.GroupUp`（P-4）與 `insyra.SortTimes`（K-15）；CAI 對每位客戶排序兩次 | rfm.go:60-70, 157；cai.go:66-73, 118 | 移除 log；直接迴圈 |

### finance

| 編號 | 嚴重度 | 問題 | 位置 | 建議 |
| --- | --- | --- | --- | --- |
| FI-1 | Med | 全部 43 個函式的參數與回傳都是 `github.com/TimLai666/go-decimal/decimal.Decimal`。這是作者自己的 decimal 套件而非社群通用的 `shopspring/decimal`，使用者呼叫任何函式都必須引入這個第三方型別；型別一旦改版整個 finance API 跟著 breaking（準則 8、10） | finance 全套件 | 這是設計決策，至少在 Docs 明講並釘住版本；或提供 `float64` 便利版 |
| FI-2 | Med | `ScheduleTable` 回傳的 DataTable 格子是 `decimal.Decimal`，core 的 `ToFloat64Safe` 不認識它，這張表的 `Mean`／`Sum`／`Describe` 全部失效，doc 只說「用 `.String()` 轉文字」；且回傳型別是 `insyra.IDataTable`（K-7）（準則 6、13） | finance/amortization.go:93-119 | 提供 `float64` 欄位版本，或讓 core 認識 decimal |
| FI-3 | Low | `RoundUnnecessary` 模式下需要捨入時「panics with decimal.ErrRoundingNecessary」（doc 原文），程式庫選項導致 panic；`opts ...Options` variadic「最後一個生效」（D-8）；`var Zero` 可被覆寫（K-12）（準則 11） | finance/options.go:66, 128-140；helpers.go:22 | 該模式改回 error；Zero 改 func 或文件註明不可改 |
| FI-4 | OK | 其餘是範本等級：每個函式驗證參數並回 error、`Options` 零值可用且逐欄位獨立預設、Excel 對應（`basis`、`type`）寫明、`NPV` 與 `NPVExcel` 的 t=0／t=1 差異講清楚、精度以 guard digits 處理 | — | — |

## 逐項清單


## . (565)

- [x] `const ErrPoppingModeFIFO ErrPoppingMode` (error_buffer.go:16) — K-4 全域 buffer 過大 API
- [x] `const ErrPoppingModeLIFO` (error_buffer.go:18) — K-4
- [x] `const ImputeConstant ImputationStrategy` (datatable_simple_imputer.go:17) — OK（E-9）
- [x] `const ImputeMean ImputationStrategy` (datatable_simple_imputer.go:14) — OK（E-9）
- [x] `const ImputeMedian ImputationStrategy` (datatable_simple_imputer.go:15) — OK（E-9）
- [x] `const ImputeMode ImputationStrategy` (datatable_simple_imputer.go:16) — OK（E-9）
- [x] `const LabelSortByFrequency` (datatable_encode.go:44) — OK（E-9）
- [x] `const LabelSortFirstSeen LabelSort` (datatable_encode.go:40) — OK（E-9）
- [x] `const LabelSortLexicographic` (datatable_encode.go:42) — OK（E-9）
- [x] `const LogLevelDebug LogLevel` (config.go:27) — OK；但整組缺 Error 等級，見 K-3
- [x] `const LogLevelFatal` (config.go:33) — K-1 Fatal 對程式庫是錯的等級
- [x] `const LogLevelInfo` (config.go:29) — OK
- [x] `const LogLevelWarning` (config.go:31) — K-3 被拿來代表失敗
- [x] `const MergeDirectionHorizontal MergeDirection` (datatable_merge.go:21) — OK
- [x] `const MergeDirectionVertical` (datatable_merge.go:22) — OK
- [x] `const MergeModeInner MergeMode` (datatable_merge.go:12) — OK
- [x] `const MergeModeLeft` (datatable_merge.go:14) — OK
- [x] `const MergeModeOuter` (datatable_merge.go:13) — OK
- [x] `const MergeModeRight` (datatable_merge.go:15) — OK
- [x] `const NaNAsCategory NaNPolicy` (datatable_encode.go:16) — OK（E-9）
- [x] `const NaNError` (datatable_encode.go:18) — OK（E-9）
- [x] `const NaNSkip` (datatable_encode.go:20) — OK（E-9）
- [x] `const OpCountAll` (datatable_groupby.go:28) — OK
- [x] `const OpCount` (datatable_groupby.go:26) — OK
- [x] `const OpCustom` (datatable_groupby.go:44) — OK
- [x] `const OpFirst` (datatable_groupby.go:38) — OK
- [x] `const OpLast` (datatable_groupby.go:40) — OK
- [x] `const OpMax` (datatable_groupby.go:24) — OK
- [x] `const OpMean` (datatable_groupby.go:18) — OK
- [x] `const OpMedian` (datatable_groupby.go:20) — OK
- [x] `const OpMin` (datatable_groupby.go:22) — OK
- [x] `const OpNUnique` (datatable_groupby.go:42) — OK
- [x] `const OpStdevP` (datatable_groupby.go:32) — OK
- [x] `const OpStdev` (datatable_groupby.go:30) — OK
- [x] `const OpSum AggregateOp` (datatable_groupby.go:16) — OK 每個常數有 doc
- [x] `const OpVarP` (datatable_groupby.go:36) — OK
- [x] `const OpVar` (datatable_groupby.go:34) — OK
- [x] `const ResampleMonthly` (datatable_resample.go:14) — OK
- [x] `const ResampleQuarterly` (datatable_resample.go:15) — OK
- [x] `const ResampleWeekly ResampleFreq` (datatable_resample.go:13) — OK（週日結束，同 pandas W）
- [x] `const ResampleYearly` (datatable_resample.go:16) — OK
- [x] `const SQLActionIfTableExistsAppend` (datatable_to_sql.go:42) — E-5 命名冗長；語意 OK
- [x] `const SQLActionIfTableExistsFail SQLActionIfTableExists` (datatable_to_sql.go:40) — E-5 命名冗長；語意 OK
- [x] `const SQLActionIfTableExistsReplace` (datatable_to_sql.go:41) — E-5 命名冗長；語意 OK
- [x] `const UnknownAsNew` (datatable_encode.go:32) — OK（E-9）
- [x] `const UnknownError` (datatable_encode.go:30) — OK（E-9）
- [x] `const UnknownIgnore UnknownPolicy` (datatable_encode.go:28) — OK（E-9）
- [x] `const VersionName` (version.go:4) — OK
- [x] `const Version` (version.go:3) — OK；release 時要同步 bump（AGENTS.md 已記）
- [x] `func (dl *DataList) Append(values ...any) *DataList` (datalist.go:102) — OK
- [x] `func (dl *DataList) AppendDataList(other IDataList) *DataList` (datalist.go:131) — OK；先鎖 other 再鎖自己的做法正確
- [x] `func (dl *DataList) Capitalize() *DataList` (datalist.go:1132) — D-16 寫死英文
- [x] `func (dl *DataList) Clear() *DataList` (datalist.go:598) — OK
- [x] `func (dl *DataList) ClearErr() *DataList` (datalist.go:2123) — OK；但 D-5 讓 Err 充滿雜訊
- [x] `func (dl *DataList) ClearNaNs() *DataList` (datalist.go:697) — D-9 O(n²)
- [x] `func (dl *DataList) ClearNils() *DataList` (datalist.go:710) — D-9 O(n²)
- [x] `func (dl *DataList) ClearNilsAndNaNs() *DataList` (datalist.go:723) — D-9
- [x] `func (dl *DataList) ClearNumbers() *DataList` (datalist.go:675) — OK 單趟；`filteredData := dl.data[:0]` 原地壓縮正確
- [x] `func (dl *DataList) ClearOutliers(stdDevs float64) *DataList` (datalist.go:733) — D-2 半毀、D-9 O(n²)；Debug log 逐元素輸出（大表噪音）
- [x] `func (dl *DataList) ClearStrings() *DataList` (datalist.go:615) — D-9 無謂平行化、遺留註解
- [x] `func (dl *DataList) Clone() *DataList` (datalist.go:166) — OK；不複製 lastError，creation 時間重設，doc 稱 deep copy 但元素是淺拷貝（元素為指標時共享）
- [x] `func (dl *DataList) Concat(other IDataList) *DataList` (datalist.go:112) — D-15 丟名稱；鎖序正確
- [x] `func (dl *DataList) Count(value any) int` (datalist.go:182) — D-5 空 list 設 Err（已實測）；D-11
- [x] `func (dl *DataList) Counter() map[any]int` (datalist.go:191) — D-11 不可比較元素 panic
- [x] `func (dl *DataList) CumMax() *DataList` (datalist_window.go:143) — OK（nil 語意對齊 pandas skipna，doc 清楚）
- [x] `func (dl *DataList) CumMin() *DataList` (datalist_window.go:149) — OK
- [x] `func (dl *DataList) CumProd() *DataList` (datalist_window.go:137) — OK
- [x] `func (dl *DataList) CumSum() *DataList` (datalist_window.go:132) — OK
- [x] `func (dl *DataList) Data() []any` (datalist.go:41) — D-9 每次整份複製；文件有寫 copy-on-read：OK
- [x] `func (dl *DataList) Describe(options ...DescribeOptions) *DataTable` (datalist_describe.go:4) — OK 設計；失敗回空表（D-3）
- [x] `func (dl *DataList) Diff(periods int) *DataList` (datalist_window.go:64) — OK 語意；失敗回 nil（D-3）
- [x] `func (dl *DataList) Difference() *DataList` (datalist.go:1864) — D-14 重複、D-2 用 ParseF64、D-3 回 nil
- [x] `func (dl *DataList) DoubleExponentialSmoothing(alpha, beta float64) *DataList` (datalist.go:955) — D-4 走 ToF64Slice、D-3 回 nil、D-14
- [x] `func (dl *DataList) Drop(index int) *DataList` (datalist.go:458) — OK；D-17 doc 錯
- [x] `func (dl *DataList) DropAll(toDrop ...any) *DataList` (datalist.go:479) — D-9 無謂平行化；D-11
- [x] `func (dl *DataList) DropIfContains(substring string) *DataList` (datalist.go:573) — OK；名稱沒說只作用於字串（doc 有寫）
- [x] `func (dl *DataList) EWM(opts EWMOptions) *EWMDataList` (datalist_ewm.go:30) — OK：options struct、參數驗證完整、pandas 對齊，是本套件的範本
- [x] `func (dl *DataList) Err() *ErrorInfo` (datalist.go:2117) — OK 設計；D-5 雜訊、K-3 等級
- [x] `func (dl *DataList) Expanding(minObs int) *ExpandingDataList` (datalist_window.go:570) — OK；minObs 用 int 參數而非 options struct，與 Rolling 不一致（Low）
- [x] `func (dl *DataList) ExponentialSmoothing(alpha float64) *DataList` (datalist.go:930) — D-4、D-3、D-14
- [x] `func (dl *DataList) FillBackward(limit ...int) *DataList` (datalist_impute.go:97) — D-8 variadic limit；邏輯 OK
- [x] `func (dl *DataList) FillByInterpolation(extrapolate ...bool) *DataList` (datalist_impute.go:227) — D-8 variadic bool；邏輯 OK，非數值時整體拒絕（正確做法）
- [x] `func (dl *DataList) FillForward(limit ...int) *DataList` (datalist_impute.go:71) — D-8；邏輯 OK
- [x] `func (dl *DataList) FillNaNWithMean() *DataList` (datalist.go:821) — 已 Deprecated：OK；D-2
- [x] `func (dl *DataList) FillWithMean() *DataList` (datalist_impute.go:124) — OK：混合型別拒絕填入（正確）
- [x] `func (dl *DataList) FillWithMedian() *DataList` (datalist_impute.go:154) — OK
- [x] `func (dl *DataList) FillWithMode() *DataList` (datalist_impute.go:182) — OK 語意；D-11 DeepEqual O(n²)
- [x] `func (dl *DataList) Filter(filterFunc func(any) bool) *DataList` (datalist.go:330) — OK；D-15 丟名稱；filterFunc panic 會穿出（無 recover，與 Map 不一致）
- [x] `func (dl *DataList) FindAll(value any) []int` (datalist.go:300) — D-5；D-11
- [x] `func (dl *DataList) FindFirst(value any) any` (datalist.go:248) — D-6 回 any；D-5
- [x] `func (dl *DataList) FindLast(value any) any` (datalist.go:274) — D-6；D-5
- [x] `func (dl *DataList) GMean() float64` (datalist.go:1346) — D-4 跳過非正數與非數值
- [x] `func (dl *DataList) Get(index int) any` (datalist.go:148) — D-6 nil 二義、D-5
- [x] `func (dl *DataList) GetCreationTimestamp() int64` (datalist.go:2064) — D-12 Unix 秒
- [x] `func (dl *DataList) GetLastModifiedTimestamp() int64` (datalist.go:2073) — D-12
- [x] `func (dl *DataList) GetName() string` (datalist.go:2090) — OK
- [x] `func (dl *DataList) HermiteInterpolation(x float64, derivatives []float64) float64` (datalist_interpolation.go:117) — D-4 走 ToF64Slice
- [x] `func (dl *DataList) IQR() float64` (datalist.go:1789) — OK；D-4
- [x] `func (dl *DataList) InsertAt(index int, value any) *DataList` (datalist.go:221) — D-15 越界 append
- [x] `func (dl *DataList) IsEqualTo(anotherDl *DataList) bool` (datalist.go:1892) — D-11 NaN 永不相等（已實測）
- [x] `func (dl *DataList) IsTheSameAs(anotherDl *DataList) bool` (datalist.go:1920) — D-12 秒級比較
- [x] `func (dl *DataList) LagrangeInterpolation(x float64) float64` (datalist_interpolation.go:63) — D-4
- [x] `func (dl *DataList) Len() int` (datalist.go:606) — OK；D-17 無 doc
- [x] `func (dl *DataList) LinearInterpolation(x float64) float64` (datalist_interpolation.go:11) — D-4
- [x] `func (dl *DataList) Lower() *DataList` (datalist.go:1119) — OK
- [x] `func (dl *DataList) MAD() float64` (datalist.go:1490) — D-4；命名：MAD 通常指 median absolute deviation，這裡是 mean absolute deviation around median，doc 要講清楚（Low）
- [x] `func (dl *DataList) Map(mapFunc func(int, any) any) *DataList` (datalist_map.go:5) — OK；recover 保留原值是否該視為錯誤，與 Filter 不一致（Low）；空 list 設 Err（D-5）
- [x] `func (dl *DataList) Max() float64` (datalist.go:1180) — D-4
- [x] `func (dl *DataList) Mean() float64` (datalist.go:1258) — D-4（已實測 [a,1,2] = 1.5）
- [x] `func (dl *DataList) Median() float64` (datalist.go:1383) — D-4
- [x] `func (dl *DataList) Min() float64` (datalist.go:1219) — D-4
- [x] `func (dl *DataList) Mode() []float64` (datalist.go:1431) — D-13 全同頻率回 nil；D-4
- [x] `func (dl *DataList) MovingAverage(windowSize int) *DataList` (datalist.go:847) — D-3、D-14
- [x] `func (dl *DataList) MovingStdev(windowSize int) *DataList` (datalist.go:983) — D-3、D-14、D-17 doc 名稱錯；每個 window 建一個 DataList（效能）
- [x] `func (dl *DataList) NearestNeighborInterpolation(x float64) float64` (datalist_interpolation.go:81) — D-4
- [x] `func (dl *DataList) NewtonInterpolation(x float64) float64` (datalist_interpolation.go:99) — D-4
- [x] `func (dl *DataList) Normalize() *DataList` (datalist.go:765) — D-2（已實測半毀）、D-3 回 nil
- [x] `func (dl *DataList) ParseDates(layouts ...string) *DataList` (datalist_dates.go:29) — OK：語意明確且文件完整；不可解析變 nil 是設計決策（可考慮 coerce/raise 選項）
- [x] `func (dl *DataList) ParseNumbers() *DataList` (datalist.go:1977) — D-16 int 變 float64
- [x] `func (dl *DataList) ParseStrings() *DataList` (datalist.go:1997) — OK
- [x] `func (dl *DataList) PctChange(periods int) *DataList` (datalist_window.go:96) — OK 語意；D-3 回 nil
- [x] `func (dl *DataList) Percentile(p float64) float64` (datalist.go:1816) — D-13 尺度 0..100；D-4
- [x] `func (dl *DataList) Pop() any` (datalist.go:441) — D-5 空 list 設 Err
- [x] `func (dl *DataList) QuadraticInterpolation(x float64) float64` (datalist_interpolation.go:37) — D-4
- [x] `func (dl *DataList) Quartile(q int) float64` (datalist.go:1735) — D-13 魔數；D-4
- [x] `func (dl *DataList) Range() float64` (datalist.go:1681) — OK；D-4
- [x] `func (dl *DataList) Rank(ascending ...bool) *DataList` (datalist.go:1051) — D-4（已實測 "b" 被當 0）、D-8
- [x] `func (dl *DataList) ReplaceAll(oldValue, newValue any) *DataList` (datalist.go:376) — OK；D-11
- [x] `func (dl *DataList) ReplaceFirst(oldValue, newValue any) *DataList` (datalist.go:345) — OK
- [x] `func (dl *DataList) ReplaceLast(oldValue, newValue any) *DataList` (datalist.go:353) — D-1 Bug（已實測）
- [x] `func (dl *DataList) ReplaceNaNsAndNilsWith(value any) *DataList` (datalist.go:431) — OK
- [x] `func (dl *DataList) ReplaceNaNsWith(value any) *DataList` (datalist.go:410) — OK
- [x] `func (dl *DataList) ReplaceNilsWith(value any) *DataList` (datalist.go:418) — OK
- [x] `func (dl *DataList) ReplaceOutliers(stdDevs float64, replacement float64) *DataList` (datalist.go:388) — OK；非數值保留（正確）；閾值用樣本 stdev 應寫進 doc
- [x] `func (dl *DataList) Reverse() *DataList` (datalist.go:1097) — OK
- [x] `func (dl *DataList) Rolling(opts RollingOptions) *RollingDataList` (datalist_window.go:219) — OK：與 EWM 同為範本；Center+Weights 對齊註解清楚
- [x] `func (dl *DataList) Sample(n int, withReplacement bool, options ...SamplingOptions) *DataList` (datalist_sampling.go:6) — D-3 回空 list、D-15 改名、D-8
- [x] `func (dl *DataList) SampleFrac(frac float64, withReplacement bool, options ...SamplingOptions) *DataList` (datalist_sampling.go:42) — 同 Sample
- [x] `func (dl *DataList) SetName(newName string) *DataList` (datalist.go:2099) — OK
- [x] `func (dl *DataList) Shift(periods int, fill ...any) *DataList` (datalist_window.go:19) — OK 語意；D-8 variadic fill
- [x] `func (dl *DataList) Show()` (show.go:713) — E-8 預設印全部
- [x] `func (dl *DataList) ShowRange(startEnd ...any)` (show.go:738) — E-8
- [x] `func (dl *DataList) ShowRangeTo(w io.Writer, startEnd ...any)` (show.go:744) — OK 接受 io.Writer；E-8 range 參數用 any
- [x] `func (dl *DataList) ShowTo(w io.Writer)` (show.go:719) — OK
- [x] `func (dl *DataList) ShowTypes()` (show.go:1052) — E-8 預設印全部
- [x] `func (dl *DataList) ShowTypesRange(startEnd ...any)` (show.go:1064) — E-8
- [x] `func (dl *DataList) ShowTypesRangeTo(w io.Writer, startEnd ...any)` (show.go:1070) — OK 接受 io.Writer；E-8 range 參數用 any
- [x] `func (dl *DataList) ShowTypesTo(w io.Writer)` (show.go:1058) — OK
- [x] `func (dl *DataList) Shuffle(options ...SamplingOptions) *DataList` (datalist_sampling.go:56) — D-3、D-15、D-8
- [x] `func (dl *DataList) Sort(ascending ...bool) *DataList` (datalist.go:1007) — D-8；D-5 空 list 設 Err；混合型別排序有 recover 回滾：OK
- [x] `func (dl *DataList) Standardize() *DataList` (datalist.go:796) — D-2 半毀
- [x] `func (dl *DataList) Stdev() float64` (datalist.go:1536) — D-4
- [x] `func (dl *DataList) StdevP() float64` (datalist.go:1559) — D-4
- [x] `func (dl *DataList) Sum() float64` (datalist.go:1148) — D-4
- [x] `func (dl *DataList) Summary()` (datalist_summary.go:12) — OK（委派 SummaryTo）
- [x] `func (dl *DataList) SummaryTo(w io.Writer)` (datalist_summary.go:17) — OK：接受 io.Writer 是正確模式，Show* 應比照
- [x] `func (dl *DataList) ToF64Slice() []float64` (datalist.go:2020) — D-4 根源；既有 follow-up；D-17 doc
- [x] `func (dl *DataList) ToStringSlice() []string` (datalist.go:2042) — OK；空 list 設 Err（D-5）
- [x] `func (dl *DataList) Update(index int, newValue any) *DataList` (datalist.go:203) — OK；D-15 Err 函式名錯
- [x] `func (dl *DataList) Upper() *DataList` (datalist.go:1106) — OK
- [x] `func (dl *DataList) Var() float64` (datalist.go:1582) — D-4
- [x] `func (dl *DataList) VarP() float64` (datalist.go:1632) — D-4
- [x] `func (dl *DataList) WeightedMean(weights any) float64` (datalist.go:1292) — D-10 `any`；D-4
- [x] `func (dl *DataList) WeightedMovingAverage(windowSize int, weights any) *DataList` (datalist.go:880) — D-10、D-3、D-14
- [x] `func (dt *DataTable) AddColUsingCCL(newColName, cclFormula string) *DataTable` (datatable_ccl.go:13) — E-1 recover 後回 nil；E-2 只走 Err()
- [x] `func (dt *DataTable) AppendCols(columns ...*DataList) *DataTable` (datatable.go:69) — OK 鎖法正確；同名自動改名不通知（T-14）
- [x] `func (dt *DataTable) AppendRowsByColIndex(rowsData ...map[string]any) *DataTable` (datatable.go:161) — T-3 資料遺失（已實測）；T-22 doc 標題錯
- [x] `func (dt *DataTable) AppendRowsByColName(rowsData ...map[string]any) *DataTable` (datatable.go:219) — OK；同名欄取第一個（T-14 相關）
- [x] `func (dt *DataTable) AppendRowsFromDataList(rowsData ...*DataList) *DataTable` (datatable.go:116) — OK
- [x] `func (dt *DataTable) AtomicDo(f func(*DataTable))` (atomic.go:64) — OK 語意清楚，文件把跨實例陷阱講明；K-16 檔內註解亂碼
- [x] `func (dt *DataTable) ChangeColName(oldName, newName string) *DataTable` (datatable_colname.go:48) — OK 走 safeColName
- [x] `func (dt *DataTable) ChangeRowName(oldName, newName string) *DataTable` (datatable_rowname.go:111) — T-9 Bug（已實測）
- [x] `func (dt *DataTable) ClearErr() *DataTable` (datatable.go:1615) — OK
- [x] `func (dt *DataTable) Clone() *DataTable` (datatable.go:1384) — OK 深拷貝；T-18 用 parallel.GroupUp
- [x] `func (dt *DataTable) Close()` (atomic.go:79) — OK；doc 亂碼（K-16）；Close 後再用會怎樣未說明
- [x] `func (dt *DataTable) ColNames() []string` (datatable_colname.go:146) — OK
- [x] `func (dt *DataTable) ColNamesToFirstRow() *DataTable` (datatable_colname.go:109) — OK
- [x] `func (dt *DataTable) Count(value any) int` (datatable.go:1271) — T-18 無謂平行化與 float 轉 int
- [x] `func (dt *DataTable) Counter() map[any]int` (datatable.go:1284) — T-21 與 DataList 重複；不可比較元素 panic（D-11）
- [x] `func (dt *DataTable) CumMaxCol(col string) *DataList` (datatable_window.go:89) — OK
- [x] `func (dt *DataTable) CumMinCol(col string) *DataList` (datatable_window.go:98) — OK
- [x] `func (dt *DataTable) CumProdCol(col string) *DataList` (datatable_window.go:80) — OK
- [x] `func (dt *DataTable) CumSumCol(col string) *DataList` (datatable_window.go:71) — OK
- [x] `func (dt *DataTable) Data(useNamesAsKeys ...bool) map[string][]any` (datatable.go:1232) — T-2 內部 slice 外洩（已實測）；variadic bool（D-8）
- [x] `func (dt *DataTable) Describe(options ...DescribeOptions) *DataTable` (datatable_describe.go:6) — OK；失敗回空表（D-3）
- [x] `func (dt *DataTable) DiffCol(col string, periods int) *DataList` (datatable_window.go:45) — OK；nil 轉空 list 掩蓋失敗（D-3）
- [x] `func (dt *DataTable) DropColNames() *DataTable` (datatable_colname.go:128) — OK；空表 warn 進 Err（D-5）
- [x] `func (dt *DataTable) DropColsByIndex(columnIndices ...string) *DataTable` (datatable.go:761) — OK
- [x] `func (dt *DataTable) DropColsByName(columnNames ...string) *DataTable` (datatable.go:745) — OK；同名只刪第一個（T-14）
- [x] `func (dt *DataTable) DropColsByNumber(columnIndices ...int) *DataTable` (datatable.go:780) — OK；負索引不支援，與 GetColByNumber 不一致（Low）
- [x] `func (dt *DataTable) DropColsContain(value ...any) *DataTable` (datatable.go:919) — OK；NaN 特判
- [x] `func (dt *DataTable) DropColsContainExcelNA() *DataTable` (datatable.go:953) — OK 薄包裝
- [x] `func (dt *DataTable) DropColsContainNaN() *DataTable` (datatable.go:889) — OK
- [x] `func (dt *DataTable) DropColsContainNil() *DataTable` (datatable.go:859) — OK
- [x] `func (dt *DataTable) DropColsContainNumber() *DataTable` (datatable.go:826) — T-7 不認 int64（已實測）
- [x] `func (dt *DataTable) DropColsContainString() *DataTable` (datatable.go:796) — OK
- [x] `func (dt *DataTable) DropRowNames() *DataTable` (datatable_rowname.go:167) — OK；D-5
- [x] `func (dt *DataTable) DropRowsByIndex(rowIndices ...int) *DataTable` (datatable.go:959) — T-6 Bug（已實測）
- [x] `func (dt *DataTable) DropRowsByName(rowNames ...string) *DataTable` (datatable.go:985) — OK
- [x] `func (dt *DataTable) DropRowsContain(value ...any) *DataTable` (datatable.go:1178) — OK
- [x] `func (dt *DataTable) DropRowsContainExcelNA() *DataTable` (datatable.go:1225) — OK
- [x] `func (dt *DataTable) DropRowsContainNaN() *DataTable` (datatable.go:1132) — OK 單趟重建
- [x] `func (dt *DataTable) DropRowsContainNil() *DataTable` (datatable.go:1088) — OK 單趟重建
- [x] `func (dt *DataTable) DropRowsContainNumber() *DataTable` (datatable.go:1049) — T-7；O(n²) 逐列 append 刪除
- [x] `func (dt *DataTable) DropRowsContainString() *DataTable` (datatable.go:1010) — O(n²) 逐列刪除（D-9）
- [x] `func (dt *DataTable) EWMCol(col string, opts EWMOptions) *EWMDataList` (datatable_window.go:128) — OK 範本
- [x] `func (dt *DataTable) EditColByIndexUsingCCL(colIndex, cclFormula string) *DataTable` (datatable_ccl.go:63) — E-1 recover 後回 nil；E-2 只走 Err()
- [x] `func (dt *DataTable) EditColByNameUsingCCL(colName, cclFormula string) *DataTable` (datatable_ccl.go:100) — E-1 recover 後回 nil；E-2 只走 Err()
- [x] `func (dt *DataTable) Err() *ErrorInfo` (datatable.go:1609) — OK；K-3/D-5
- [x] `func (dt *DataTable) ExecuteCCL(cclStatements string) *DataTable` (datatable_ccl.go:147) — E-1 recover 後回 nil；E-2 只走 Err()
- [x] `func (dt *DataTable) ExpandingCol(col string, minObs int) *ExpandingDataList` (datatable_window.go:118) — OK
- [x] `func (dt *DataTable) FillBackward(limit int, cols ...string) *DataTable` (datatable_impute.go:50) — E-7 簽名不對稱
- [x] `func (dt *DataTable) FillByInterpolation(cols ...string) *DataTable` (datatable_impute.go:88) — E-7 非數值欄靜默跳過
- [x] `func (dt *DataTable) FillForward(limit int, cols ...string) *DataTable` (datatable_impute.go:41) — E-7 簽名不對稱
- [x] `func (dt *DataTable) FillWithMean(cols ...string) *DataTable` (datatable_impute.go:58) — E-7 非數值欄靜默跳過
- [x] `func (dt *DataTable) FillWithMedian(cols ...string) *DataTable` (datatable_impute.go:69) — E-7 非數值欄靜默跳過
- [x] `func (dt *DataTable) FillWithMode(cols ...string) *DataTable` (datatable_impute.go:80) — OK
- [x] `func (dt *DataTable) Filter(filterFunc func(rowIndex int, columnIndex string, value any) bool) *DataTable` (datatable_filters.go:334) — T-13 任一格子符合即留列；資料有複製：OK
- [x] `func (dt *DataTable) FilterByCustomElement(filterFunc func(value any) bool) *DataTable` (datatable_filters.go:325) — T-13 與 Filter 重複
- [x] `func (dt *DataTable) FilterCols(filterFunc func(rowIndex int, rowName string, x any) bool) *DataTable` (datatable_filters.go:398) — T-4 共用 rowNames；T-5 以第一欄當列數
- [x] `func (dt *DataTable) FilterColsByColIndexEqualTo(columnIndexLetter string) *DataTable` (datatable_filters.go:62) — T-4 共用欄位、T-5 &DataTable{}、T-21
- [x] `func (dt *DataTable) FilterColsByColIndexGreaterThan(columnIndexLetter string) *DataTable` (datatable_filters.go:13) — T-4、T-5、T-21
- [x] `func (dt *DataTable) FilterColsByColIndexGreaterThanOrEqualTo(columnIndexLetter string) *DataTable` (datatable_filters.go:37) — T-4、T-5、T-21
- [x] `func (dt *DataTable) FilterColsByColIndexLessThan(columnIndexLetter string) *DataTable` (datatable_filters.go:87) — T-4、T-5、T-21；同上越界風險
- [x] `func (dt *DataTable) FilterColsByColIndexLessThanOrEqualTo(columnIndexLetter string) *DataTable` (datatable_filters.go:112) — T-4、T-5、T-21；索引越界時 `dt.columns[:colIdx+1]` 會 panic（推論）
- [x] `func (dt *DataTable) FilterColsByColNameContains(substring string) *DataTable` (datatable_filters.go:169) — T-4、T-5
- [x] `func (dt *DataTable) FilterColsByColNameEqualTo(columnName string) *DataTable` (datatable_filters.go:139) — T-4（已實測共用）、T-5（已實測 panic）
- [x] `func (dt *DataTable) FilterRows(filterFunc func(colIndex, colName string, x any) bool) *DataTable` (datatable_filters.go:462) — T-13 任一格子；T-5 jagged panic（已實測）
- [x] `func (dt *DataTable) FilterRowsByRowIndexEqualTo(index int) *DataTable` (datatable_filters.go:208) — T-21；經 Filter
- [x] `func (dt *DataTable) FilterRowsByRowIndexGreaterThan(threshold int) *DataTable` (datatable_filters.go:194) — T-21
- [x] `func (dt *DataTable) FilterRowsByRowIndexGreaterThanOrEqualTo(threshold int) *DataTable` (datatable_filters.go:201) — T-21
- [x] `func (dt *DataTable) FilterRowsByRowIndexLessThan(threshold int) *DataTable` (datatable_filters.go:215) — T-21
- [x] `func (dt *DataTable) FilterRowsByRowIndexLessThanOrEqualTo(threshold int) *DataTable` (datatable_filters.go:222) — T-21
- [x] `func (dt *DataTable) FilterRowsByRowNameContains(substring string) *DataTable` (datatable_filters.go:249) — OK 複製；rowNames nil 有防護
- [x] `func (dt *DataTable) FilterRowsByRowNameEqualTo(rowName string) *DataTable` (datatable_filters.go:231) — OK
- [x] `func (dt *DataTable) FindColsIfAllElementsContainSubstring(substring string) []string` (datatable.go:717) — T-19 非字串視為符合；T-18 遞迴 contains
- [x] `func (dt *DataTable) FindColsIfAnyElementContainsSubstring(substring string) []string` (datatable.go:691) — T-18
- [x] `func (dt *DataTable) FindColsIfContains(value any) []string` (datatable.go:652) — T-19 每欄 warn
- [x] `func (dt *DataTable) FindColsIfContainsAll(values ...any) []string` (datatable.go:667) — T-19
- [x] `func (dt *DataTable) FindRowsIfAllElementsContainSubstring(substring string) []int` (datatable.go:626) — T-19 非字串視為符合
- [x] `func (dt *DataTable) FindRowsIfAnyElementContainsSubstring(substring string) []int` (datatable.go:606) — T-18
- [x] `func (dt *DataTable) FindRowsIfContains(value any) []int` (datatable.go:547) — OK；經 DataList.FindAll，空欄 warn（D-5）
- [x] `func (dt *DataTable) FindRowsIfContainsAll(values ...any) []int` (datatable.go:574) — OK；`==` 對 NaN 無效（D-11）
- [x] `func (dt *DataTable) GetCol(index string) *DataList` (datatable.go:297) — T-11 ToUpper 後退回名稱（已實測）
- [x] `func (dt *DataTable) GetColByName(name string) *DataList` (datatable.go:337) — OK 回 Clone；找不到 warn（D-5）；T-22 無 doc
- [x] `func (dt *DataTable) GetColByNumber(index int) *DataList` (datatable.go:319) — OK；T-22 無 doc
- [x] `func (dt *DataTable) GetColIndexByName(name string) string` (datatable_colindex.go:6) — OK；找不到 warn（D-5）
- [x] `func (dt *DataTable) GetColIndexByNumber(number int) string` (datatable_colindex.go:19) — OK
- [x] `func (dt *DataTable) GetColNameByIndex(index string) string` (datatable_colname.go:83) — OK
- [x] `func (dt *DataTable) GetColNameByNumber(index int) string` (datatable_colname.go:66) — OK；T-22 無 doc
- [x] `func (dt *DataTable) GetColNumberByName(name string) int` (datatable_colname.go:97) — 回 -1 且 warn；與 GetRowIndexByName 的 (int,bool) 不一致（D-6）
- [x] `func (dt *DataTable) GetCreationTimestamp() int64` (datatable.go:1591) — D-12 Unix 秒；T-22
- [x] `func (dt *DataTable) GetElement(rowIndex int, columnIndex string) any` (datatable.go:258) — OK；越界 warn（D-5）
- [x] `func (dt *DataTable) GetElementByNumberIndex(rowIndex int, columnIndex int) any` (datatable.go:280) — T-1 panic（已實測）
- [x] `func (dt *DataTable) GetLastModifiedTimestamp() int64` (datatable.go:1595) — D-12
- [x] `func (dt *DataTable) GetName() string` (datatable_name.go:6) — OK
- [x] `func (dt *DataTable) GetRow(index int) *DataList` (datatable.go:355) — OK 補 nil
- [x] `func (dt *DataTable) GetRowByName(name string) *DataList` (datatable.go:383) — 跳過短欄而非補 nil，與 GetRow 不一致（Low）；T-22
- [x] `func (dt *DataTable) GetRowIndexByName(name string) (int, bool)` (datatable_rowname.go:88) — OK (int,bool) 是正確形狀；找不到 warn（D-5）
- [x] `func (dt *DataTable) GetRowNameByIndex(index int) (string, bool)` (datatable_rowname.go:57) — OK
- [x] `func (dt *DataTable) GroupBy(keyCols ...string) *GroupedDataTable` (datatable_groupby.go:152) — OK 設計；T-16 淺拷貝快照；T-17 int/float key 分家
- [x] `func (dt *DataTable) Headers() []string` (datatable_colname.go:159) — T-21 別名
- [x] `func (dt *DataTable) LabelEncode(opts LabelEncodeOptions) (*DataTable, *LabelEncoder, error)` (datatable_encode.go:137) — OK（E-9）
- [x] `func (dt *DataTable) Map(mapFunc func(rowIndex int, colIndex string, element any) any) *DataTable` (datatable_map.go:8) — OK；T-18 每格取鎖；recover 保留原值（與 DataList.Map 一致）
- [x] `func (dt *DataTable) MaxAbsScale(cols ...string) (*DataTable, *MaxAbsScaler, error)` (datatable_scale.go:148) — OK 回 (table, scaler, error) 三值（E-9）；cols variadic（E-6）
- [x] `func (dt *DataTable) Mean() any` (datatable.go:1324) — T-10 回 any、分母捏造（已實測）
- [x] `func (dt *DataTable) Merge(other IDataTable, direction MergeDirection, mode MergeMode, on ...string) (*DataTable, error)` (datatable_merge.go:29) — OK 回 error；`on ...string` 0/1/2 個有各自語意（D-8）；T-15；欄名衝突後綴 `_other` 不可設定（pandas suffixes）
- [x] `func (dt *DataTable) MinMaxScale(featureMin, featureMax float64, cols ...string) (*DataTable, *MinMaxScaler, error)` (datatable_scale.go:128) — OK 回 (table, scaler, error) 三值（E-9）；cols variadic（E-6）
- [x] `func (dt *DataTable) NumCols() int` (datatable.go:1315) — OK；T-22
- [x] `func (dt *DataTable) NumRows() int` (datatable.go:1307) — OK；T-22
- [x] `func (dt *DataTable) OneHotEncode(opts OneHotOptions) (*DataTable, *OneHotEncoder, error)` (datatable_encode.go:124) — OK（E-9）
- [x] `func (dt *DataTable) OrdinalEncode(opts OrdinalEncodeOptions) (*DataTable, *OrdinalEncoder, error)` (datatable_encode.go:150) — OK（E-9）
- [x] `func (dt *DataTable) ParseDatesCols(cols []string, layouts ...string) *DataTable` (datalist_dates.go:46) — OK；找不到的欄位跳過並 warn（部分成功，doc 有寫）
- [x] `func (dt *DataTable) PctChangeCol(col string, periods int) *DataList` (datatable_window.go:58) — 同 DiffCol
- [x] `func (dt *DataTable) Pivot(cfg PivotConfig) (*DataTable, error)` (datatable_pivot.go:101) — OK：error + Err 雙通道；T-17 AggFunc 字串；FillNA 語意清楚
- [x] `func (dt *DataTable) Replace(oldValue, newValue any) *DataTable` (datatable_replace.go:11) — OK
- [x] `func (dt *DataTable) ReplaceInCol(colIndex string, oldValue, newValue any, mode ...int) *DataTable` (datatable_replace.go:133) — T-11 名稱被當索引（已實測）；T-20
- [x] `func (dt *DataTable) ReplaceInRow(rowIndex int, oldValue, newValue any, mode ...int) *DataTable` (datatable_replace.go:52) — T-20 mode 魔數；錯誤經 Err：OK
- [x] `func (dt *DataTable) ReplaceNaNsAndNilsInCol(colIndex string, newValue any, mode ...int) *DataTable` (datatable_replace.go:193) — T-11、T-20；NaN 只認 float64
- [x] `func (dt *DataTable) ReplaceNaNsAndNilsInRow(rowIndex int, newValue any, mode ...int) *DataTable` (datatable_replace.go:112) — T-20；NaN 只認 float64
- [x] `func (dt *DataTable) ReplaceNaNsAndNilsWith(newValue any) *DataTable` (datatable_replace.go:35) — OK 用 isNilOrNaN
- [x] `func (dt *DataTable) ReplaceNaNsInCol(colIndex string, newValue any, mode ...int) *DataTable` (datatable_replace.go:153) — T-11、T-20
- [x] `func (dt *DataTable) ReplaceNaNsInRow(rowIndex int, newValue any, mode ...int) *DataTable` (datatable_replace.go:72) — T-20
- [x] `func (dt *DataTable) ReplaceNaNsWith(newValue any) *DataTable` (datatable_replace.go:19) — OK
- [x] `func (dt *DataTable) ReplaceNilsInCol(colIndex string, newValue any, mode ...int) *DataTable` (datatable_replace.go:173) — T-11、T-20
- [x] `func (dt *DataTable) ReplaceNilsInRow(rowIndex int, newValue any, mode ...int) *DataTable` (datatable_replace.go:92) — T-20
- [x] `func (dt *DataTable) ReplaceNilsWith(newValue any) *DataTable` (datatable_replace.go:27) — OK
- [x] `func (dt *DataTable) Resample(timeCol string, freq ResampleFreq, aggs ...ResampleAgg) (*DataTable, error)` (datatable_resample.go:30) — OK：回 error、參數驗證完整（T-24 範本）；period end 以各列 Location 計算，混合時區會分組異常（Low）
- [x] `func (dt *DataTable) RobustScale(cols ...string) (*DataTable, *RobustScaler, error)` (datatable_scale.go:138) — OK 回 (table, scaler, error) 三值（E-9）；cols variadic（E-6）
- [x] `func (dt *DataTable) RollingCol(col string, opts RollingOptions) *RollingDataList` (datatable_window.go:109) — OK 範本
- [x] `func (dt *DataTable) RowNames() []string` (datatable_rowname.go:184) — OK
- [x] `func (dt *DataTable) RowNamesToFirstCol() *DataTable` (datatable_rowname.go:134) — OK（註解記錄了先前的 bug 修法）
- [x] `func (dt *DataTable) Sample(n int, withReplacement bool, options ...SamplingOptions) *DataTable` (datatable_sampling.go:142) — OK 設計；失敗回空表（D-3）；改名 _Sampled（D-15）
- [x] `func (dt *DataTable) SampleFrac(frac float64, withReplacement bool, options ...SamplingOptions) *DataTable` (datatable_sampling.go:163) — 同 Sample
- [x] `func (dt *DataTable) SetColNameByIndex(index string, name string) *DataTable` (datatable_colname.go:3) — OK；T-22 無 doc
- [x] `func (dt *DataTable) SetColNameByNumber(numberIndex int, name string) *DataTable` (datatable_colname.go:27) — OK
- [x] `func (dt *DataTable) SetColNames(colNames []string) *DataTable` (datatable_colname.go:163) — T-14 自動加欄（已實測）
- [x] `func (dt *DataTable) SetColToRowNames(columnIndex string) *DataTable` (datatable.go:508) — T-1 panic（已實測）
- [x] `func (dt *DataTable) SetHeaders(headers []string) *DataTable` (datatable_colname.go:187) — T-21 別名
- [x] `func (dt *DataTable) SetName(name string) *DataTable` (datatable_name.go:15) — OK
- [x] `func (dt *DataTable) SetRowNameByIndex(index int, name string) *DataTable` (datatable_rowname.go:16) — OK；doc 完整（範本）
- [x] `func (dt *DataTable) SetRowNames(rowNames []string) *DataTable` (datatable_rowname.go:201) — OK
- [x] `func (dt *DataTable) SetRowToColNames(rowIndex int) *DataTable` (datatable.go:527) — T-1 panic（已實測）
- [x] `func (dt *DataTable) ShiftCol(col string, periods int, fill ...any) *DataList` (datatable_window.go:36) — OK 薄包裝；找不到欄回空 list（D-18）
- [x] `func (dt *DataTable) Show()` (show.go:40) — E-8 預設印全部
- [x] `func (dt *DataTable) ShowRange(startEnd ...any)` (show.go:63) — E-8
- [x] `func (dt *DataTable) ShowRangeTo(w io.Writer, startEnd ...any)` (show.go:69) — OK 接受 io.Writer；E-8 range 參數用 any
- [x] `func (dt *DataTable) ShowTo(w io.Writer)` (show.go:46) — OK
- [x] `func (dt *DataTable) ShowTypes()` (show.go:437) — E-8 預設印全部
- [x] `func (dt *DataTable) ShowTypesRange(startEnd ...any)` (show.go:460) — E-8
- [x] `func (dt *DataTable) ShowTypesRangeTo(w io.Writer, startEnd ...any)` (show.go:466) — OK 接受 io.Writer；E-8 range 參數用 any
- [x] `func (dt *DataTable) ShowTypesTo(w io.Writer)` (show.go:443) — OK
- [x] `func (dt *DataTable) Shuffle(options ...SamplingOptions) *DataTable` (datatable_sampling.go:177) — 同 Sample
- [x] `func (dt *DataTable) SimpleRandomSample(sampleSize int) *DataTable` (datatable_sampling.go:228) — 已 Deprecated：OK
- [x] `func (dt *DataTable) Size() (numRows int, numCols int)` (datatable.go:1298) — OK
- [x] `func (dt *DataTable) SortBy(configs ...DataTableSortConfig) *DataTable` (datatable_sort.go:17) — T-23 零值 config；演算法（cycle→swap）已驗算正確
- [x] `func (dt *DataTable) StandardScale(cols ...string) (*DataTable, *StandardScaler, error)` (datatable_scale.go:118) — OK 回 (table, scaler, error) 三值（E-9）；cols variadic（E-6）
- [x] `func (dt *DataTable) Summary()` (datatable_summary.go:16) — OK 委派 SummaryTo
- [x] `func (dt *DataTable) SummaryTo(w io.Writer)` (datatable_summary.go:23) — OK 範本
- [x] `func (dt *DataTable) SwapColsByIndex(columnIndex1 string, columnIndex2 string) *DataTable` (datatable_swap.go:47) — OK
- [x] `func (dt *DataTable) SwapColsByName(columnName1 string, columnName2 string) *DataTable` (datatable_swap.go:12) — OK
- [x] `func (dt *DataTable) SwapColsByNumber(columnNumber1 int, columnNumber2 int) *DataTable` (datatable_swap.go:70) — OK 支援負索引
- [x] `func (dt *DataTable) SwapRowsByIndex(rowIndex1 int, rowIndex2 int) *DataTable` (datatable_swap.go:111) — OK；jagged 補 nil
- [x] `func (dt *DataTable) SwapRowsByName(rowName1 string, rowName2 string) *DataTable` (datatable_swap.go:143) — OK；rowNames 為 nil 時 panic（T-5）
- [x] `func (dt *DataTable) To2DSlice() [][]any` (datatable.go:1413) — OK 複製
- [x] `func (dt *DataTable) ToCSV(filePath string, setRowNamesToFirstCol bool, setColNamesToFirstRow bool, includeBOM bool) error` (datatable_csv.go:14) — T-12 時間格式、三個 bool、無 Writer 版
- [x] `func (dt *DataTable) ToJSON(filePath string, useColNames bool) error` (datatable_json.go:58) — OK 回 error
- [x] `func (dt *DataTable) ToJSON_Bytes(useColNames bool) []byte` (datatable_json.go:85) — T-12 NaN 回 nil（已實測）；K-13 底線命名
- [x] `func (dt *DataTable) ToJSON_String(useColNames bool) string` (datatable_json.go:102) — T-12；K-13
- [x] `func (dt *DataTable) ToMap(useNamesAsKeys ...bool) map[string][]any` (datatable.go:1264) — T-2 別名同樣外洩
- [x] `func (dt *DataTable) ToSQL(db *gorm.DB, tableName string, options ...ToSQLOptions) error` (datatable_to_sql.go:48) — E-3 gorm；OK 委派 ctx 版
- [x] `func (dt *DataTable) ToSQLContext(ctx context.Context, db *gorm.DB, tableName string, options ...ToSQLOptions) error` (datatable_to_sql.go:57) — OK ctx 版；E-3 gorm
- [x] `func (dt *DataTable) TrainTestSplit(trainFrac float64, options ...SamplingOptions) (*DataTable, *DataTable)` (datatable_sampling.go:188) — OK 設計（PreserveOrder、seed）；失敗回兩個空表（D-3）
- [x] `func (dt *DataTable) Transpose() *DataTable` (datatable.go:1341) — T-8 Bug（已實測）+ 原地未說明
- [x] `func (dt *DataTable) Unpivot(cfg UnpivotConfig) (*DataTable, error)` (datatable_pivot.go:329) — OK
- [x] `func (dt *DataTable) UpdateCol(index string, dl *DataList) *DataTable` (datatable.go:431) — OK 鎖兩者並複製
- [x] `func (dt *DataTable) UpdateColByNumber(index int, dl *DataList) *DataTable` (datatable.go:453) — OK
- [x] `func (dt *DataTable) UpdateElement(rowIndex int, columnIndex string, value any) *DataTable` (datatable.go:409) — OK；欄不存在 warn 後仍 updateTimestamp（Low）
- [x] `func (dt *DataTable) UpdateRow(index int, dl *DataList) *DataTable` (datatable.go:476) — OK；dl 較短時只更新前段，doc 未說（Low）
- [x] `func (e *EWMDataList) Mean() *DataList` (datalist_ewm.go:85) — OK
- [x] `func (e *EWMDataList) Std() *DataList` (datalist_ewm.go:99) — OK
- [x] `func (e *EWMDataList) Var() *DataList` (datalist_ewm.go:93) — OK
- [x] `func (e *ExpandingDataList) Max() *DataList` (datalist_window.go:644) — OK
- [x] `func (e *ExpandingDataList) Mean() *DataList` (datalist_window.go:620) — OK
- [x] `func (e *ExpandingDataList) Median() *DataList` (datalist_window.go:657) — OK
- [x] `func (e *ExpandingDataList) Min() *DataList` (datalist_window.go:631) — OK
- [x] `func (e *ExpandingDataList) Std() *DataList` (datalist_window.go:672) — OK
- [x] `func (e *ExpandingDataList) Sum() *DataList` (datalist_window.go:609) — OK
- [x] `func (e *ExpandingDataList) Var() *DataList` (datalist_window.go:683) — OK
- [x] `func (e *LabelEncoder) Classes() []any` (datatable_encode.go:287) — OK（E-9）
- [x] `func (e *LabelEncoder) Inverse(values ...any) ([]any, error)` (datatable_encode.go:316) — OK（E-9）
- [x] `func (e *LabelEncoder) InverseTransform(dt *DataTable) (*DataTable, error)` (datatable_encode.go:274) — OK（E-9）
- [x] `func (e *LabelEncoder) Kind() string` (datatable_encode.go:284) — OK（E-9）
- [x] `func (e *LabelEncoder) Options() LabelEncodeOptions` (datatable_encode.go:308) — OK（E-9）
- [x] `func (e *LabelEncoder) OutputColumn() string` (datatable_encode.go:300) — OK（E-9）
- [x] `func (e *LabelEncoder) SourceColumn() string` (datatable_encode.go:292) — OK（E-9）
- [x] `func (e *LabelEncoder) Transform(dt *DataTable) (*DataTable, error)` (datatable_encode.go:259) — OK（E-9）
- [x] `func (e *OneHotEncoder) Categories() map[string][]any` (datatable_encode.go:210) — OK（E-9）
- [x] `func (e *OneHotEncoder) Columns() []string` (datatable_encode.go:237) — OK（E-9）
- [x] `func (e *OneHotEncoder) InverseTransform(dt *DataTable) (*DataTable, error)` (datatable_encode.go:185) — OK（E-9）
- [x] `func (e *OneHotEncoder) Kind() string` (datatable_encode.go:207) — OK（E-9）
- [x] `func (e *OneHotEncoder) Options() OneHotOptions` (datatable_encode.go:249) — OK（E-9）
- [x] `func (e *OneHotEncoder) OutputColumns() []string` (datatable_encode.go:219) — OK（E-9）
- [x] `func (e *OneHotEncoder) OutputColumnsByColumn() map[string][]string` (datatable_encode.go:225) — OK（E-9）
- [x] `func (e *OneHotEncoder) Transform(dt *DataTable) (*DataTable, error)` (datatable_encode.go:163) — OK（E-9）
- [x] `func (e *OrdinalEncoder) Classes() []any` (datatable_encode.go:349) — OK（E-9）
- [x] `func (e *OrdinalEncoder) Inverse(values ...any) ([]any, error)` (datatable_encode.go:380) — OK（E-9）
- [x] `func (e *OrdinalEncoder) InverseTransform(dt *DataTable) (*DataTable, error)` (datatable_encode.go:336) — OK（E-9）
- [x] `func (e *OrdinalEncoder) Kind() string` (datatable_encode.go:346) — OK（E-9）
- [x] `func (e *OrdinalEncoder) Options() OrdinalEncodeOptions` (datatable_encode.go:370) — OK（E-9）
- [x] `func (e *OrdinalEncoder) OutputColumn() string` (datatable_encode.go:362) — OK（E-9）
- [x] `func (e *OrdinalEncoder) SourceColumn() string` (datatable_encode.go:354) — OK（E-9）
- [x] `func (e *OrdinalEncoder) Transform(dt *DataTable) (*DataTable, error)` (datatable_encode.go:321) — OK（E-9）
- [x] `func (e ErrorInfo) Error() string` (error_buffer.go:32) — OK 實作 error 介面
- [x] `func (g *GroupedDataTable) Aggregate(configs ...AggregateConfig) *DataTable` (datatable_groupby.go:286) — OK；錯誤設在 parent.Err 並回空表（D-3）；OpSum 等繼承 D-4 Med 跳過語意
- [x] `func (g *GroupedDataTable) AggregateAll(op AggregateOp) *DataTable` (datatable_groupby.go:350) — OK
- [x] `func (g *GroupedDataTable) Count() *DataTable` (datatable_groupby.go:393) — OK
- [x] `func (g *GroupedDataTable) CumMaxCol(col string) *GroupedColumnTransform` (datatable_groupby_window.go:179) — OK
- [x] `func (g *GroupedDataTable) CumMinCol(col string) *GroupedColumnTransform` (datatable_groupby_window.go:191) — OK
- [x] `func (g *GroupedDataTable) CumProdCol(col string) *GroupedColumnTransform` (datatable_groupby_window.go:167) — OK
- [x] `func (g *GroupedDataTable) CumSumCol(col string) *GroupedColumnTransform` (datatable_groupby_window.go:155) — OK
- [x] `func (g *GroupedDataTable) Describe(options ...DescribeOptions) *DataTable` (datatable_groupby_describe.go:5) — OK
- [x] `func (g *GroupedDataTable) DiffCol(col string, periods int) *GroupedColumnTransform` (datatable_groupby_window.go:131) — OK；nil 轉空 list 掩蓋失敗（D-3）
- [x] `func (g *GroupedDataTable) ExpandingCol(col string, minObs int) *GroupedExpandingCol` (datatable_groupby_window.go:296) — OK
- [x] `func (g *GroupedDataTable) PctChangeCol(col string, periods int) *GroupedColumnTransform` (datatable_groupby_window.go:143) — 同 DiffCol
- [x] `func (g *GroupedDataTable) RollingCol(col string, opts RollingOptions) *GroupedRollingCol` (datatable_groupby_window.go:216) — OK 範本
- [x] `func (g *GroupedDataTable) ShiftCol(col string, periods int, fill ...any) *GroupedColumnTransform` (datatable_groupby_window.go:119) — OK 薄包裝；找不到欄回空 list（D-18）
- [x] `func (ge *GroupedExpandingCol) Max() *GroupedColumnTransform` (datatable_groupby_window.go:334) — OK 薄包裝，語意同 DataList 版
- [x] `func (ge *GroupedExpandingCol) Mean() *GroupedColumnTransform` (datatable_groupby_window.go:322) — T-10 回 any、分母捏造（已實測）
- [x] `func (ge *GroupedExpandingCol) Median() *GroupedColumnTransform` (datatable_groupby_window.go:340) — OK 薄包裝，語意同 DataList 版
- [x] `func (ge *GroupedExpandingCol) Min() *GroupedColumnTransform` (datatable_groupby_window.go:328) — OK 薄包裝，語意同 DataList 版
- [x] `func (ge *GroupedExpandingCol) Std() *GroupedColumnTransform` (datatable_groupby_window.go:346) — OK 薄包裝，語意同 DataList 版
- [x] `func (ge *GroupedExpandingCol) Sum() *GroupedColumnTransform` (datatable_groupby_window.go:316) — OK 薄包裝，語意同 DataList 版
- [x] `func (ge *GroupedExpandingCol) Var() *GroupedColumnTransform` (datatable_groupby_window.go:352) — OK 薄包裝，語意同 DataList 版
- [x] `func (gr *GroupedRollingCol) Apply(fn func(window []any) any) *GroupedColumnTransform` (datatable_groupby_window.go:278) — OK 薄包裝，語意同 DataList 版
- [x] `func (gr *GroupedRollingCol) Max() *GroupedColumnTransform` (datatable_groupby_window.go:254) — OK 薄包裝，語意同 DataList 版
- [x] `func (gr *GroupedRollingCol) Mean() *GroupedColumnTransform` (datatable_groupby_window.go:242) — T-10 回 any、分母捏造（已實測）
- [x] `func (gr *GroupedRollingCol) Median() *GroupedColumnTransform` (datatable_groupby_window.go:260) — OK 薄包裝，語意同 DataList 版
- [x] `func (gr *GroupedRollingCol) Min() *GroupedColumnTransform` (datatable_groupby_window.go:248) — OK 薄包裝，語意同 DataList 版
- [x] `func (gr *GroupedRollingCol) Std() *GroupedColumnTransform` (datatable_groupby_window.go:266) — OK 薄包裝，語意同 DataList 版
- [x] `func (gr *GroupedRollingCol) Sum() *GroupedColumnTransform` (datatable_groupby_window.go:236) — OK 薄包裝，語意同 DataList 版
- [x] `func (gr *GroupedRollingCol) Var() *GroupedColumnTransform` (datatable_groupby_window.go:272) — OK 薄包裝，語意同 DataList 版
- [x] `func (i *SimpleImputer) Fit(dt *DataTable, cols ...string) error` (datatable_simple_imputer.go:82) — OK（E-9）
- [x] `func (i *SimpleImputer) FitTransform(dt *DataTable, cols ...string) (*DataTable, error)` (datatable_simple_imputer.go:138) — OK（E-9）
- [x] `func (i *SimpleImputer) Kind() string` (datatable_simple_imputer.go:57) — OK（E-9）
- [x] `func (i *SimpleImputer) Params() map[string]ScalerParams` (datatable_simple_imputer.go:65) — OK（E-9）
- [x] `func (i *SimpleImputer) Transform(dt *DataTable) (*DataTable, error)` (datatable_simple_imputer.go:146) — OK（E-9）
- [x] `func (l LogLevel) String() string` (error_buffer.go:37) — OK；放在 error_buffer.go 不在 config.go，找不到（Low）
- [x] `func (o AggregateOp) String() string` (datatable_groupby.go:48) — OK
- [x] `func (r *RollingDataList) Apply(fn func(window []any) any) *DataList` (datalist_window.go:435) — OK；MinObs 以數值計、傳給 fn 的是原始 window：doc 有寫
- [x] `func (r *RollingDataList) Beta(other *DataList) *DataList` (datalist_window.go:544) — OK
- [x] `func (r *RollingDataList) Corr(other *DataList) *DataList` (datalist_window.go:528) — OK；`pearson` 回 any 而非 float64（內部，Low）
- [x] `func (r *RollingDataList) Cov(other *DataList) *DataList` (datalist_window.go:535) — OK
- [x] `func (r *RollingDataList) Max() *DataList` (datalist_window.go:381) — OK
- [x] `func (r *RollingDataList) Mean() *DataList` (datalist_window.go:346) — OK
- [x] `func (r *RollingDataList) Median() *DataList` (datalist_window.go:395) — OK
- [x] `func (r *RollingDataList) Min() *DataList` (datalist_window.go:368) — OK
- [x] `func (r *RollingDataList) Std() *DataList` (datalist_window.go:410) — OK
- [x] `func (r *RollingDataList) Sum() *DataList` (datalist_window.go:328) — OK
- [x] `func (r *RollingDataList) Var() *DataList` (datalist_window.go:421) — OK
- [x] `func (s *DataList) AtomicDo(f func(*DataList))` (atomic.go:29) — 同 DataTable.AtomicDo；receiver 名 `s` 與其他方法的 `dl` 不一致（Low）
- [x] `func (s *DataList) Close()` (atomic.go:44) — 同 DataTable.Close
- [x] `func (t *GroupedColumnTransform) As(name string) *DataList` (datatable_groupby_window.go:28) — OK；錯誤時回空 list（D-18）
- [x] `func AtomicDoAll(f func(), instances ...any)` (atomic.go:109) — K-8 `...any` 且型別錯誤時靜默不鎖
- [x] `func CalcColIndex(colNumber int) (colIndex string, ok bool)` (utils.go:249) — OK 功能；K-13 命名不對稱
- [x] `func ClearErrors()` (error_buffer.go:195) — K-4（保留候選）
- [x] `func ConvertLongDataToWide(data, factor IDataList, independents []IDataList, aggFunc func([]float64) float64) IDataTable` (utils.go:132) — 已標 Deprecated 且指向替代：OK；錯誤用 fmt.Println（隨 Deprecated 一起走）
- [x] `func DetectEncoding(filePath string) (string, error)` (utils.go:279) — K-10 8KB 邊界與不支援編碼靜默
- [x] `func GetAllErrors() []ErrorInfo` (error_buffer.go:263) — K-4（保留候選）
- [x] `func GetErrorCount() int` (error_buffer.go:202) — K-4
- [x] `func GetErrorsByLevel(level LogLevel) []ErrorInfo` (error_buffer.go:282) — K-4
- [x] `func GetErrorsByPackage(packageName string) []ErrorInfo` (error_buffer.go:303) — K-4
- [x] `func HasError() bool` (error_buffer.go:211) — K-4
- [x] `func HasErrorAboveLevel(level LogLevel) bool` (error_buffer.go:220) — K-4
- [x] `func IsNumeric(v any) bool` (utils.go:221) — OK；reflect 分支覆蓋 named types
- [x] `func LogDebug(packageName, funcName, msg string, args ...any)` (logger.go:44) — K-17 無法接自訂 logger
- [x] `func LogFatal(packageName, funcName, msg string, args ...any)` (logger.go:9) — K-1 程式庫呼叫 os.Exit
- [x] `func LogInfo(packageName, funcName, msg string, args ...any)` (logger.go:60) — K-17；預設開啟造成成功路徑噪音（C-9 的根源）
- [x] `func LogWarning(packageName, funcName, msg string, args ...any)` (logger.go:27) — K-17；同時 pushError（K-4、K-5）
- [x] `func NewDataList(values ...any) *DataList` (datalist.go:81) — OK；遞迴攤平巢狀 slice 是驚喜行為但 doc 有寫；typed slice 直接可用是好的體驗
- [x] `func NewDataTable(columns ...*DataList) *DataTable` (datatable.go:47) — OK；T-22 無 doc
- [x] `func NewDefaultMinMaxScaler() *MinMaxScaler` (datatable_scale.go:109) — OK（E-9）
- [x] `func NewMaxAbsScaler() *MaxAbsScaler` (datatable_scale.go:115) — OK（E-9）
- [x] `func NewMinMaxScaler(featureMin, featureMax float64) *MinMaxScaler` (datatable_scale.go:104) — OK（E-9）
- [x] `func NewRobustScaler() *RobustScaler` (datatable_scale.go:112) — OK（E-9）
- [x] `func NewSimpleImputer(strategy ImputationStrategy, constant ...any) *SimpleImputer` (datatable_simple_imputer.go:44) — E-6 variadic constant
- [x] `func NewStandardScaler() *StandardScaler` (datatable_scale.go:101) — OK（E-9）
- [x] `func ParseColIndex(colIndex string) (colNumber int, ok bool)` (utils.go:244) — OK；K-13 命名
- [x] `func PeekError(mode ErrPoppingMode) *ErrorInfo` (error_buffer.go:236) — K-4
- [x] `func PopAllErrors() []ErrorInfo` (error_buffer.go:325) — K-4（保留候選）
- [x] `func PopError(mode ErrPoppingMode) (LogLevel, string)` (error_buffer.go:83) — K-4 空緩衝哨兵 `(LogLevelInfo, "")` 含糊
- [x] `func PopErrorAndCallback(mode ErrPoppingMode, callback func(errType LogLevel, packageName string, funcName string, errMsg string))` (error_buffer.go:178) — K-4 空緩衝哨兵 `(LogLevelInfo, "")` 含糊
- [x] `func PopErrorByFuncName(packageName, funcName string, mode ErrPoppingMode) (LogLevel, string)` (error_buffer.go:139) — K-4 空緩衝哨兵 `(LogLevelInfo, "")` 含糊
- [x] `func PopErrorByPackageName(packageName string, mode ErrPoppingMode) (LogLevel, string)` (error_buffer.go:100) — K-4 空緩衝哨兵 `(LogLevelInfo, "")` 含糊
- [x] `func PopErrorInfo(mode ErrPoppingMode) *ErrorInfo` (error_buffer.go:346) — K-4 空緩衝哨兵 `(LogLevelInfo, "")` 含糊
- [x] `func PowRat(base *big.Rat, exponent int) *big.Rat` (utils.go:105) — K-15 無人用
- [x] `func ProcessData(input any) ([]any, int)` (utils.go:43) — K-15 回傳 `(data, len)` 冗餘、失敗無 error
- [x] `func ReadCSV_File(filePath string, setFirstColToRowNames bool, setFirstRowToColNames bool, encoding ...string) (*DataTable, error)` (read.go:115) — K-13 底線命名；K-14 裸 bool + variadic
- [x] `func ReadCSV_FileWithOptions(filePath string, opts CSVReadOptions) (*DataTable, error)` (read.go:127) — K-13 底線命名；K-14 裸 bool + variadic
- [x] `func ReadCSV_String(csvString string, setFirstColToRowNames bool, setFirstRowToColNames bool) (*DataTable, error)` (read.go:356) — K-13；K-14
- [x] `func ReadCSV_StringWithOptions(csvString string, opts CSVReadOptions) (*DataTable, error)` (read.go:365) — K-13；K-14
- [x] `func ReadExcelSheet(filePath string, sheetName string, setFirstColToRowNames bool, setFirstRowToColNames bool) (*DataTable, error)` (read.go:384) — K-14 無 options、無型別推斷（既有 follow-up）；未 Close 前已 defer Close：OK
- [x] `func ReadJSON(data any) (*DataTable, error)` (read.go:465) — OK 功能；`any` 六路 switch 可接受為便利入口；與 ReadJSON_File 型別不一致（K-9）
- [x] `func ReadJSON_File(filePath string) (*DataTable, error)` (read.go:410) — K-9 整數失真
- [x] `func ReadSQL(db *gorm.DB, tableName string, options ...ReadSQLOptions) (*DataTable, error)` (datatable_from_sql.go:65) — OK 委派；E-3
- [x] `func ReadSQLContext(ctx context.Context, db *gorm.DB, tableName string, options ...ReadSQLOptions) (*DataTable, error)` (datatable_from_sql.go:71) — OK；E-3、E-4
- [x] `func ReadSQLStream(ctx context.Context, db *gorm.DB, tableName string, options ...ReadSQLOptions) (<-chan ReadSQLChunk, error)` (datatable_from_sql.go:120) — OK 洩漏契約寫在 doc（範本）；E-3
- [x] `func SetDefaultConfig()` (config.go:98) — OK；K-16 doc 錯；名稱 `Reset` 更貼切（Low）
- [x] `func Show(label string, object showable, startEnd ...any)` (show.go:27) — E-8 showable 未匯出；variadic any
- [x] `func Slice2DToDataTable(data any) (*DataTable, error)` (read.go:26) — OK；K-16 空切片回錯
- [x] `func SliceToF64(input []any) []float64` (utils.go:28) — K-2 捏造零值且 stats 仍在用
- [x] `func SortTimes(times []time.Time)` (utils.go:255) — K-15
- [x] `func SqrtRat(x *big.Rat) *big.Rat` (utils.go:90) — K-15
- [x] `type AggregateConfig struct { SourceCol string As string Op AggregateOp Custom func(group *DataList) any }` (datatable_groupby.go:85) — OK 欄位 doc 完整
- [x] `type AggregateOp int` (datatable_groupby.go:12) — OK 有 String()
- [x] `type CSVReadOptions struct { FirstColToRowNames bool FirstRowToColNames bool Encoding string RawStrings bool AllowRaggedRows bool TrimLeadingSpace bool }` (read.go:93) — OK 零值可用、欄位有 doc：這是套件內最合格的 options struct，其他地方應比照
- [x] `type DataList struct { data []any name string creationTimestamp int64 lastModifiedTimestamp atomic.Int64 atomicActor core.AtomicActor lastError *ErrorInfo }` (datalist.go:25) — OK；欄位全私有
- [x] `type DataListScaler interface { FitDataList(dl *DataList) error TransformDataList(dl *DataList) (*DataList, error) FitTransformDataList(dl *DataList) (*DataList, error) InverseTransformDataList(dl *DataList) (*DataList, error) }` (datatable_scale.go:52) — OK（E-9）
- [x] `type DataTable struct { columns []*DataList rowNames *core.BiIndex name string creationTimestamp int64 lastModifiedTimestamp atomic.Int64 atomicActor core.AtomicActor lastError *ErrorInfo }` (datatable.go:33) — OK 欄位私有
- [x] `type DataTableSortConfig struct { ColumnIndex string ColumnNumber int ColumnName string Descending bool }` (datatable_sort.go:7) — T-23 零值歧義
- [x] `type DescribeOptions struct { Percentiles []float64 IncludeAll bool }` (describe_options.go:13) — OK；D-13 尺度
- [x] `type EWMDataList struct { srcData []any srcName string opts EWMOptions alpha float64 parent *DataList err string }` (datalist_ewm.go:19) — OK；`err string` 應為 error（Low）
- [x] `type EWMOptions struct { Alpha float64 Span float64 HalfLife float64 Adjust bool Bias bool MinObs int }` (datalist_ewm.go:8) — OK 範本
- [x] `type Encoder interface { Transform(dt *DataTable) (*DataTable, error) InverseTransform(dt *DataTable) (*DataTable, error) Kind() string }` (datatable_encode.go:80) — OK（E-9）
- [x] `type ErrPoppingMode int` (error_buffer.go:11) — K-4
- [x] `type ErrorInfo struct { Level LogLevel PackageName string FuncName string Message string Timestamp time.Time }` (error_buffer.go:23) — OK；`Level` 用 LogLevel 而非 error 等級（K-3）
- [x] `type ExpandingDataList struct { srcData []any srcName string minObs int parent *DataList err string }` (datalist_window.go:561) — OK
- [x] `type F64orRat = utils.F64orRat` (utils.go:20) — K-15 internal 介面的匯出別名
- [x] `type GroupedColumnTransform struct { parent *GroupedDataTable sourceCol int sourceLabel string perGroupFn func(group *DataList) *DataList err string }` (datatable_groupby_window.go:17) — OK；`.As(name)` 終端設計清楚
- [x] `type GroupedDataTable struct { parent *DataTable keyColNumbers []int keyColLabels []string columnsSnapshot []*DataList rowsByGroup map[string][]int groupOrder []string groupKeyValues map[string][]any initErr string }` (datatable_groupby.go:106) — OK；doc 明講非並行安全
- [x] `type GroupedExpandingCol struct { parent *GroupedDataTable sourceCol int sourceLabel string minObs int err string }` (datatable_groupby_window.go:286) — OK
- [x] `type GroupedRollingCol struct { parent *GroupedDataTable sourceCol int sourceLabel string opts RollingOptions err string }` (datatable_groupby_window.go:206) — OK
- [x] `type IDataList interface { AtomicDo(func(*DataList)) GetCreationTimestamp() int64 GetLastModifiedTimestamp() int64 updateTimestamp() GetName() string SetName(string) *DataList Data() []any Append(values ...any) *DataList Concat(other IDataList) *DataList AppendDataList(other IDataList) *DataList Get(index int) any Clone() *DataList Count(value any) int Counter() map[any]int Update(index int, value any) *DataList InsertAt(index int, value any) *DataList FindFirst(any) any FindLast(any) any FindAll(any) []int Filter(func(any) bool) *DataList ReplaceFirst(any, any) *DataList ReplaceLast(any, any) *DataList ReplaceAll(any, any) *DataList ReplaceOutliers(float64, float64) *DataList Pop() any Drop(index int) *DataList DropAll(...any) *DataList DropIfContains(string) *DataList Clear() *DataList ClearStrings() *DataList ClearNumbers() *DataList ClearNaNs() *DataList ClearNils() *DataList ClearNilsAndNaNs() *DataList ClearOutliers(float64) *DataList ReplaceNaNsWith(any) *DataList ReplaceNilsWith(any) *DataList ReplaceNaNsAndNilsWith(any) *DataList Normalize() *DataList Standardize() *DataList FillNaNWithMean() *DataList FillWithMean() *DataList FillForward(limit ...int) *DataList FillBackward(limit ...int) *DataList FillWithMedian() *DataList FillWithMode() *DataList FillByInterpolation(extrapolate ...bool) *DataList MovingAverage(int) *DataList WeightedMovingAverage(int, any) *DataList ExponentialSmoothing(float64) *DataList DoubleExponentialSmoothing(float64, float64) *DataList EWM(EWMOptions) *EWMDataList MovingStdev(int) *DataList Len() int Sample(n int, withReplacement bool, options ...SamplingOptions) *DataList SampleFrac(frac float64, withReplacement bool, options ...SamplingOptions) *DataList Shuffle(options ...SamplingOptions) *DataList Sort(ascending ...bool) *DataList Map(mapFunc func(int, any) any) *DataList Rank(ascending ...bool) *DataList Reverse() *DataList Upper() *DataList Lower() *DataList Capitalize() *DataList Sum() float64 Max() float64 Min() float64 Mean() float64 WeightedMean(weights any) float64 GMean() float64 Median() float64 Mode() []float64 MAD() float64 Stdev() float64 StdevP() float64 Var() float64 VarP() float64 Range() float64 Quartile(int) float64 IQR() float64 Percentile(float64) float64 Difference() *DataList Describe(...DescribeOptions) *DataTable Summary() Err() *ErrorInfo ClearErr() *DataList IsEqualTo(*DataList) bool IsTheSameAs(*DataList) bool Show() ShowRange(startEnd ...any) ShowTypes() ShowTypesRange(startEnd ...any) ParseNumbers() *DataList ParseStrings() *DataList ParseDates(layouts ...string) *DataList ToF64Slice() []float64 ToStringSlice() []string LinearInterpolation(float64) float64 QuadraticInterpolation(float64) float64 LagrangeInterpolation(float64) float64 NearestNeighborInterpolation(float64) float64 NewtonInterpolation(float64) float64 HermiteInterpolation(float64, []float64) float64 }` (interfaces.go:6) — K-7 不可實作、過大；`Capitalize() *DataList // Statistics` 註解錯位
- [x] `type IDataTable interface { AtomicDo(func(*DataTable)) AppendCols(columns ...*DataList) *DataTable AppendRowsFromDataList(rowsData ...*DataList) *DataTable AppendRowsByColIndex(rowsData ...map[string]any) *DataTable AppendRowsByColName(rowsData ...map[string]any) *DataTable GetElement(rowIndex int, columnIndex string) any GetElementByNumberIndex(rowIndex int, columnIndex int) any GetCol(index string) *DataList GetColByNumber(index int) *DataList GetColByName(name string) *DataList GetRow(index int) *DataList GetRowByName(name string) *DataList UpdateElement(rowIndex int, columnIndex string, value any) *DataTable UpdateCol(index string, dl *DataList) *DataTable UpdateColByNumber(index int, dl *DataList) *DataTable UpdateRow(index int, dl *DataList) *DataTable SetColToRowNames(columnIndex string) *DataTable SetRowToColNames(rowIndex int) *DataTable ChangeColName(oldName, newName string) *DataTable GetColNameByNumber(index int) string GetColIndexByName(name string) string GetColIndexByNumber(number int) string GetColNumberByName(name string) int SetColNameByIndex(index string, name string) *DataTable SetColNameByNumber(numberIndex int, name string) *DataTable ColNamesToFirstRow() *DataTable DropColNames() *DataTable ColNames() []string Headers() []string SetColNames(colNames []string) *DataTable SetHeaders(headers []string) *DataTable FindRowsIfContains(value any) []int FindRowsIfContainsAll(values ...any) []int FindRowsIfAnyElementContainsSubstring(substring string) []int FindRowsIfAllElementsContainSubstring(substring string) []int FindColsIfContains(value any) []string FindColsIfContainsAll(values ...any) []string FindColsIfAnyElementContainsSubstring(substring string) []string FindColsIfAllElementsContainSubstring(substring string) []string DropColsByName(columnNames ...string) *DataTable DropColsByIndex(columnIndices ...string) *DataTable DropColsByNumber(columnIndices ...int) *DataTable DropColsContainString() *DataTable DropColsContainNumber() *DataTable DropColsContainNil() *DataTable DropColsContainNaN() *DataTable DropColsContain(value ...any) *DataTable DropColsContainExcelNA() *DataTable DropRowsByIndex(rowIndices ...int) *DataTable DropRowsByName(rowNames ...string) *DataTable DropRowsContainString() *DataTable DropRowsContainNumber() *DataTable DropRowsContainNil() *DataTable DropRowsContainNaN() *DataTable DropRowsContain(value ...any) *DataTable DropRowsContainExcelNA() *DataTable Data(useNamesAsKeys ...bool) map[string][]any ToMap(useNamesAsKeys ...bool) map[string][]any Show() ShowTypes() ShowRange(startEnd ...any) ShowTypesRange(startEnd ...any) GetRowIndexByName(name string) (int, bool) GetRowNameByIndex(index int) (string, bool) SetRowNameByIndex(index int, name string) *DataTable ChangeRowName(oldName, newName string) *DataTable RowNamesToFirstCol() *DataTable DropRowNames() *DataTable RowNames() []string SetRowNames(rowNames []string) *DataTable GetCreationTimestamp() int64 GetLastModifiedTimestamp() int64 getRowNameByIndex(index int) (string, bool) getMaxColLength() int updateTimestamp() GetName() string SetName(name string) *DataTable Size() (numRows int, numCols int) NumRows() int NumCols() int Count(value any) int Mean() any Describe(...DescribeOptions) *DataTable Summary() Err() *ErrorInfo ClearErr() *DataTable Transpose() *DataTable Clone() *DataTable To2DSlice() [][]any SimpleRandomSample(sampleSize int) *DataTable Sample(n int, withReplacement bool, options ...SamplingOptions) *DataTable SampleFrac(frac float64, withReplacement bool, options ...SamplingOptions) *DataTable Shuffle(options ...SamplingOptions) *DataTable TrainTestSplit(trainFrac float64, options ...SamplingOptions) (*DataTable, *DataTable) Map(mapFunc func(rowIndex int, colIndex string, element any) any) *DataTable SortBy(configs ...DataTableSortConfig) *DataTable Filter(filterFunc func(rowIndex int, columnIndex string, value any) bool) *DataTable FilterByCustomElement(f func(value any) bool) *DataTable FilterRows(filterFunc func(colIndex, colName string, x any) bool) *DataTable FilterCols(filterFunc func(rowIndex int, rowName string, x any) bool) *DataTable FilterColsByColIndexGreaterThan(threshold string) *DataTable FilterColsByColIndexGreaterThanOrEqualTo(threshold string) *DataTable FilterColsByColIndexLessThan(threshold string) *DataTable FilterColsByColIndexLessThanOrEqualTo(threshold string) *DataTable FilterColsByColIndexEqualTo(index string) *DataTable FilterColsByColNameEqualTo(name string) *DataTable FilterColsByColNameContains(substring string) *DataTable FilterRowsByRowNameEqualTo(name string) *DataTable FilterRowsByRowNameContains(substring string) *DataTable FilterRowsByRowIndexGreaterThan(threshold int) *DataTable FilterRowsByRowIndexGreaterThanOrEqualTo(threshold int) *DataTable FilterRowsByRowIndexLessThan(threshold int) *DataTable FilterRowsByRowIndexLessThanOrEqualTo(threshold int) *DataTable FilterRowsByRowIndexEqualTo(index int) *DataTable SwapColsByName(columnName1 string, columnName2 string) *DataTable SwapColsByIndex(columnIndex1 string, columnIndex2 string) *DataTable SwapColsByNumber(columnNumber1 int, columnNumber2 int) *DataTable SwapRowsByIndex(rowIndex1 int, rowIndex2 int) *DataTable SwapRowsByName(rowName1 string, rowName2 string) *DataTable ToCSV(filePath string, setRowNamesToFirstCol bool, setColNamesToFirstRow bool, includeBOM bool) error ToJSON(filePath string, useColNames bool) error ToJSON_Bytes(useColNames bool) []byte ToJSON_String(useColNames bool) string ToSQL(db *gorm.DB, tableName string, options ...ToSQLOptions) error Merge(other IDataTable, direction MergeDirection, mode MergeMode, on ...string) (*DataTable, error) EWMCol(string, EWMOptions) *EWMDataList Resample(string, ResampleFreq, ...ResampleAgg) (*DataTable, error) ParseDatesCols(cols []string, layouts ...string) *DataTable AddColUsingCCL(newColName, ccl string) *DataTable Replace(oldValue, newValue any) *DataTable ReplaceNaNsWith(newValue any) *DataTable ReplaceNilsWith(newValue any) *DataTable ReplaceNaNsAndNilsWith(newValue any) *DataTable FillForward(int, ...string) *DataTable FillBackward(int, ...string) *DataTable FillWithMean(...string) *DataTable FillWithMedian(...string) *DataTable FillWithMode(...string) *DataTable FillByInterpolation(...string) *DataTable OneHotEncode(opts OneHotOptions) (*DataTable, *OneHotEncoder, error) LabelEncode(opts LabelEncodeOptions) (*DataTable, *LabelEncoder, error) OrdinalEncode(opts OrdinalEncodeOptions) (*DataTable, *OrdinalEncoder, error) StandardScale(cols ...string) (*DataTable, *StandardScaler, error) MinMaxScale(featureMin, featureMax float64, cols ...string) (*DataTable, *MinMaxScaler, error) RobustScale(cols ...string) (*DataTable, *RobustScaler, error) MaxAbsScale(cols ...string) (*DataTable, *MaxAbsScaler, error) ReplaceInRow(rowIndex int, oldValue, newValue any, mode ...int) *DataTable ReplaceNaNsInRow(rowIndex int, newValue any, mode ...int) *DataTable ReplaceNilsInRow(rowIndex int, newValue any, mode ...int) *DataTable ReplaceNaNsAndNilsInRow(rowIndex int, newValue any, mode ...int) *DataTable ReplaceInCol(colIndex string, oldValue, newValue any, mode ...int) *DataTable ReplaceNaNsInCol(colIndex string, newValue any, mode ...int) *DataTable ReplaceNilsInCol(colIndex string, newValue any, mode ...int) *DataTable ReplaceNaNsAndNilsInCol(colIndex string, newValue any, mode ...int) *DataTable }` (interfaces.go:121) — K-7；`Mean() any` 回 any（DataTable 段再審）
- [x] `type ImputationStrategy string` (datatable_simple_imputer.go:11) — OK（E-9）
- [x] `type LabelEncodeOptions struct { Column string NewColumn string SortBy LabelSort HandleNaN NaNPolicy Unknown UnknownPolicy KeepOriginal bool }` (datatable_encode.go:60) — OK（E-9）
- [x] `type LabelEncoder struct { opts LabelEncodeOptions sourceRef string sourceName string encodedName string classes []any keyToID map[string]int }` (datatable_encode.go:104) — OK（E-9）
- [x] `type LabelSort int` (datatable_encode.go:36) — OK（E-9）
- [x] `type LogLevel int` (config.go:23) — K-3 缺 Error
- [x] `type MaxAbsScaler struct{ scaler }` (datatable_scale.go:98) — OK（E-9）
- [x] `type MergeDirection int` (datatable_merge.go:18) — OK
- [x] `type MergeMode int` (datatable_merge.go:9) — OK；命名 pandas 用 how=inner/outer/left/right，語意一致
- [x] `type MinMaxScaler struct{ scaler }` (datatable_scale.go:90) — OK（E-9）
- [x] `type NaNPolicy int` (datatable_encode.go:12) — OK（E-9）
- [x] `type OneHotEncoder struct { opts OneHotOptions columns []oneHotColumnState outputColumns []string }` (datatable_encode.go:87) — OK（E-9）
- [x] `type OneHotOptions struct { Columns []string DropFirst bool HandleNaN NaNPolicy Unknown UnknownPolicy Prefix string Separator string KeepOriginal bool SortCategories bool }` (datatable_encode.go:48) — OK（E-9）
- [x] `type OrdinalEncodeOptions struct { Column string Order []any NewColumn string HandleNaN NaNPolicy Unknown UnknownPolicy KeepOriginal bool }` (datatable_encode.go:70) — OK（E-9）
- [x] `type OrdinalEncoder struct { opts OrdinalEncodeOptions sourceRef string sourceName string encodedName string classes []any keyToID map[string]int }` (datatable_encode.go:114) — OK（E-9）
- [x] `type PivotConfig struct { Index []string Columns string Values string AggFunc string Custom func(group *DataList) any FillNA any SortCols bool }` (datatable_pivot.go:26) — OK doc 完整；T-17 AggFunc 字串
- [x] `type ReadSQLChunk struct { Table *DataTable Err error }` (datatable_from_sql.go:97) — OK：Table/Err 二擇一，doc 有寫
- [x] `type ReadSQLOptions struct { RowNameColumn string IndexCol string Query string Params []any Columns []string Schema string Limit int Offset int WhereClause string OrderBy string ParseDates []string DType map[string]reflect.Type ChunkSize int }` (datatable_from_sql.go:19) — E-4 WhereClause 注入面；E-5 IndexCol 別名
- [x] `type ResampleAgg struct { Col string Op AggregateOp As string }` (datatable_resample.go:21) — OK
- [x] `type ResampleFreq int` (datatable_resample.go:10) — OK
- [x] `type RobustScaler struct{ scaler }` (datatable_scale.go:94) — OK（E-9）
- [x] `type RollingDataList struct { srcData []any srcName string opts RollingOptions parent *DataList err string }` (datalist_window.go:208) — OK
- [x] `type RollingOptions struct { Window int MinObs int Center bool Weights []float64 }` (datalist_window.go:196) — OK 範本
- [x] `type SQLActionIfTableExists int` (datatable_to_sql.go:37) — E-5 命名冗長；語意 OK
- [x] `type SamplingOptions struct { Seed uint64 UseSeed bool PreserveOrder bool }` (datatable_sampling.go:14) — OK；`UseSeed bool` + `Seed` 可用 `*uint64` 取代（Low）
- [x] `type Scaler interface { Fit(dt *DataTable, cols ...string) error Transform(dt *DataTable) (*DataTable, error) FitTransform(dt *DataTable, cols ...string) (*DataTable, error) InverseTransform(dt *DataTable) (*DataTable, error) Params() map[string]ScalerParams Kind() string }` (datatable_scale.go:42) — E-6 cols variadic 必填
- [x] `type ScalerParams struct { Column string Kind string Replacement any PassThrough bool Mean float64 Std float64 Min float64 Max float64 Median float64 Q1 float64 Q3 float64 IQR float64 MaxAbs float64 OutputMin float64 OutputMax float64 }` (datatable_scale.go:12) — OK（E-9）
- [x] `type SimpleImputer struct { strategy ImputationStrategy constant any constantArgs int columns []simpleImputerColumn fitted bool }` (datatable_simple_imputer.go:25) — OK（E-9）
- [x] `type StandardScaler struct{ scaler }` (datatable_scale.go:87) — OK（E-9）
- [x] `type ToSQLOptions struct { IfExists SQLActionIfTableExists RowNames bool ColumnTypes map[string]string Schema string BatchSize int }` (datatable_to_sql.go:20) — E-5 IfExists 註解過時；其餘欄位 doc 完整
- [x] `type UnknownPolicy int` (datatable_encode.go:24) — OK（E-9）
- [x] `type UnpivotConfig struct { IDVars []string ValueVars []string VarName string ValueName string DropNA bool }` (datatable_pivot.go:71) — OK
- [x] `var Config *configStruct` (config.go:21) — K-6 型別未匯出、欄位非 atomic
- [x] `var ReadSlice2D` (read.go:22) — K-12 可被覆寫的 var、重複命名
- [x] `var ToFloat64Safe` (utils.go:23) — K-12
- [x] `var ToFloat64` (utils.go:22) — K-12；失敗回 0 無法區分（ToF64Slice follow-up 的根源）
- [x] `func (c *configStruct) Dangerously_TurnOffThreadSafety()` (config.go:90) — K-13 底線命名；K-6 型別未匯出
- [x] `func (c *configStruct) GetAccelerationEnabled() bool` (config.go:82) — OK atomic
- [x] `func (c *configStruct) GetDefaultErrHandlingFunc() func(errType LogLevel, packageName string, funcName string, errMsg string)` (config.go:64) — K-6 非 atomic 讀寫
- [x] `func (c *configStruct) GetDoesUseColoredOutput() bool` (config.go:48) — K-13 命名（GetDoesUse/GetDontPanicStatus）；K-6；SetDontPanic 依 K-1 應移除
- [x] `func (c *configStruct) GetDontPanicStatus() bool` (config.go:56) — K-13 命名（GetDoesUse/GetDontPanicStatus）；K-6；SetDontPanic 依 K-1 應移除
- [x] `func (c *configStruct) GetLogLevel() LogLevel` (config.go:40) — K-6 非 atomic 讀寫
- [x] `func (c *configStruct) SetAcceleration(enabled bool)` (config.go:69) — OK atomic
- [x] `func (c *configStruct) SetDefaultErrHandlingFunc(fn func(errType LogLevel, packageName string, funcName string, errMsg string))` (config.go:60) — K-6 非 atomic 讀寫
- [x] `func (c *configStruct) SetDontPanic(dontPanic bool)` (config.go:52) — K-13 命名（GetDoesUse/GetDontPanicStatus）；K-6；SetDontPanic 依 K-1 應移除
- [x] `func (c *configStruct) SetLogLevel(level LogLevel)` (config.go:36) — K-6 非 atomic 讀寫
- [x] `func (c *configStruct) SetUseColoredOutput(colored bool)` (config.go:44) — K-13 命名（GetDoesUse/GetDontPanicStatus）；K-6；SetDontPanic 依 K-1 應移除
- [x] `func (c *dataTableContext) GetAllData() ([]any, error)` (ccl.go:166) — 非公開：未匯出型別，沒有任何匯出路徑可取得，僅供 internal/ccl 介面；不需審查
- [x] `func (c *dataTableContext) GetCell(colIndex, rowIndex int) (any, error)` (ccl.go:49) — 非公開：未匯出型別，沒有任何匯出路徑可取得，僅供 internal/ccl 介面；不需審查
- [x] `func (c *dataTableContext) GetCellByName(colName string, rowIndex int) (any, error)` (ccl.go:59) — 非公開：未匯出型別，沒有任何匯出路徑可取得，僅供 internal/ccl 介面；不需審查
- [x] `func (c *dataTableContext) GetCol(index int) any` (ccl.go:20) — 非公開：未匯出型別，沒有任何匯出路徑可取得，僅供 internal/ccl 介面；不需審查
- [x] `func (c *dataTableContext) GetColByName(name string) (any, error)` (ccl.go:27) — 非公開：未匯出型別，沒有任何匯出路徑可取得，僅供 internal/ccl 介面；不需審查
- [x] `func (c *dataTableContext) GetColCount() int` (ccl.go:108) — 非公開：未匯出型別，沒有任何匯出路徑可取得，僅供 internal/ccl 介面；不需審查
- [x] `func (c *dataTableContext) GetColData(index int) ([]any, error)` (ccl.go:112) — 非公開：未匯出型別，沒有任何匯出路徑可取得，僅供 internal/ccl 介面；不需審查
- [x] `func (c *dataTableContext) GetColDataByName(name string) ([]any, error)` (ccl.go:124) — 非公開：未匯出型別，沒有任何匯出路徑可取得，僅供 internal/ccl 介面；不需審查
- [x] `func (c *dataTableContext) GetColIndexByName(colName string) (int, error)` (ccl.go:97) — 非公開：未匯出型別，沒有任何匯出路徑可取得，僅供 internal/ccl 介面；不需審查
- [x] `func (c *dataTableContext) GetCurrentRow() any` (ccl.go:45) — 非公開：未匯出型別，沒有任何匯出路徑可取得，僅供 internal/ccl 介面；不需審查
- [x] `func (c *dataTableContext) GetRowAt(rowIndex int) (any, error)` (ccl.go:70) — 非公開：未匯出型別，沒有任何匯出路徑可取得，僅供 internal/ccl 介面；不需審查
- [x] `func (c *dataTableContext) GetRowCount() int` (ccl.go:135) — 非公開：未匯出型別，沒有任何匯出路徑可取得，僅供 internal/ccl 介面；不需審查
- [x] `func (c *dataTableContext) GetRowIndex() int` (ccl.go:41) — 非公開：未匯出型別，沒有任何匯出路徑可取得，僅供 internal/ccl 介面；不需審查
- [x] `func (c *dataTableContext) GetRowIndexByName(rowName string) (int, error)` (ccl.go:86) — 非公開：未匯出型別，沒有任何匯出路徑可取得，僅供 internal/ccl 介面；不需審查
- [x] `func (c *dataTableContext) SetRowIndex(index int) error` (ccl.go:142) — 非公開：未匯出型別，沒有任何匯出路徑可取得，僅供 internal/ccl 介面；不需審查
- [x] `func (s *scaler) Fit(dt *DataTable, cols ...string) error` (datatable_scale.go:172) — OK（E-9 範本）；經 StandardScaler 等嵌入對外可見；Fit 的 cols variadic 見 E-6
- [x] `func (s *scaler) FitDataList(dl *DataList) error` (datatable_scale.go:360) — OK（E-9 範本）；經 StandardScaler 等嵌入對外可見；Fit 的 cols variadic 見 E-6
- [x] `func (s *scaler) FitTransform(dt *DataTable, cols ...string) (*DataTable, error)` (datatable_scale.go:216) — OK（E-9 範本）；經 StandardScaler 等嵌入對外可見；Fit 的 cols variadic 見 E-6
- [x] `func (s *scaler) FitTransformDataList(dl *DataList) (*DataList, error)` (datatable_scale.go:383) — OK（E-9 範本）；經 StandardScaler 等嵌入對外可見；Fit 的 cols variadic 見 E-6
- [x] `func (s *scaler) InverseTransform(dt *DataTable) (*DataTable, error)` (datatable_scale.go:233) — OK（E-9 範本）；經 StandardScaler 等嵌入對外可見；Fit 的 cols variadic 見 E-6
- [x] `func (s *scaler) InverseTransformDataList(dl *DataList) (*DataList, error)` (datatable_scale.go:398) — OK（E-9 範本）；經 StandardScaler 等嵌入對外可見；Fit 的 cols variadic 見 E-6
- [x] `func (s *scaler) Kind() string` (datatable_scale.go:158) — OK（E-9 範本）；經 StandardScaler 等嵌入對外可見；Fit 的 cols variadic 見 E-6
- [x] `func (s *scaler) Params() map[string]ScalerParams` (datatable_scale.go:161) — OK（E-9 範本）；經 StandardScaler 等嵌入對外可見；Fit 的 cols variadic 見 E-6
- [x] `func (s *scaler) Transform(dt *DataTable) (*DataTable, error)` (datatable_scale.go:226) — OK（E-9 範本）；經 StandardScaler 等嵌入對外可見；Fit 的 cols variadic 見 E-6
- [x] `func (s *scaler) TransformDataList(dl *DataList) (*DataList, error)` (datatable_scale.go:392) — OK（E-9 範本）；經 StandardScaler 等嵌入對外可見；Fit 的 cols variadic 見 E-6

## accel (142)

- [ ] `const BackendCPU Backend` (types.go:29)
- [ ] `const BackendCUDA Backend` (types.go:30)
- [ ] `const BackendMetal Backend` (types.go:31)
- [ ] `const BackendUnknown Backend` (types.go:28)
- [ ] `const BackendWebGPU Backend` (types.go:32)
- [ ] `const DataTypeAny DataType` (types.go:70)
- [ ] `const DataTypeBool DataType` (types.go:66)
- [ ] `const DataTypeFloat64 DataType` (types.go:68)
- [ ] `const DataTypeInt64 DataType` (types.go:67)
- [ ] `const DataTypeString DataType` (types.go:69)
- [ ] `const DataTypeUnknown DataType` (types.go:65)
- [ ] `const DeviceTypeCPU DeviceType` (types.go:39)
- [ ] `const DeviceTypeDiscrete DeviceType` (types.go:41)
- [ ] `const DeviceTypeIntegrated DeviceType` (types.go:40)
- [ ] `const DeviceTypeUnknown DeviceType` (types.go:38)
- [ ] `const DeviceTypeVirtual DeviceType` (types.go:42)
- [ ] `const ExecutorKindNone ExecutorKind` (types.go:143)
- [ ] `const ExecutorKindRegistered ExecutorKind` (types.go:144)
- [ ] `const ExecutorKindUnknown ExecutorKind` (types.go:142)
- [ ] `const FallbackReasonBufferTooLarge FallbackReason` (types.go:88)
- [ ] `const FallbackReasonCPUOnly FallbackReason` (types.go:78)
- [ ] `const FallbackReasonDTypeNotEligible FallbackReason` (types.go:86)
- [ ] `const FallbackReasonDeviceSelectionEmpty FallbackReason` (types.go:84)
- [ ] `const FallbackReasonDiscoveryError FallbackReason` (types.go:79)
- [ ] `const FallbackReasonExecutionFailed FallbackReason` (types.go:90)
- [ ] `const FallbackReasonNoAccelerator FallbackReason` (types.go:77)
- [ ] `const FallbackReasonNoBackendExecutor FallbackReason` (types.go:83)
- [ ] `const FallbackReasonNone FallbackReason` (types.go:76)
- [ ] `const FallbackReasonPrecisionNotAccepted FallbackReason` (types.go:85)
- [ ] `const FallbackReasonReadbackTimeout FallbackReason` (types.go:89)
- [ ] `const FallbackReasonShaderCompileFailed FallbackReason` (types.go:87)
- [ ] `const FallbackReasonStrictGPUUnavailable FallbackReason` (types.go:80)
- [ ] `const FallbackReasonWorkloadNotProfitable FallbackReason` (types.go:82)
- [ ] `const FallbackReasonWorkloadUnsupported FallbackReason` (types.go:81)
- [ ] `const MemoryClassDevice MemoryClass` (types.go:50)
- [ ] `const MemoryClassShared MemoryClass` (types.go:49)
- [ ] `const MemoryClassUnknown MemoryClass` (types.go:48)
- [ ] `const MergePolicyBackendNative MergePolicy` (types.go:136)
- [ ] `const MergePolicyCPU MergePolicy` (types.go:135)
- [ ] `const MergePolicyUnknown MergePolicy` (types.go:134)
- [ ] `const ModeAuto Mode` (types.go:8)
- [ ] `const ModeCPU Mode` (types.go:9)
- [ ] `const ModeGPU Mode` (types.go:10)
- [ ] `const ModeStrictGPU Mode` (types.go:11)
- [ ] `const OpNearestShortlist Op` (types.go:109)
- [ ] `const OpUnknown Op` (types.go:98)
- [ ] `const PrecisionExact Precision` (types.go:120)
- [ ] `const PrecisionFloat32 Precision` (types.go:121)
- [ ] `const ProbeSourceEnvStub ProbeSource` (types.go:59)
- [ ] `const ProbeSourceNative ProbeSource` (types.go:58)
- [ ] `const ProbeSourceSDK ProbeSource` (types.go:57)
- [ ] `const ProbeSourceUnknown ProbeSource` (types.go:56)
- [ ] `const ShardStrategyAuto ShardStrategy` (types.go:21)
- [ ] `const ShardStrategyForced ShardStrategy` (types.go:22)
- [ ] `const ShardStrategySingle ShardStrategy` (types.go:20)
- [ ] `const WorkloadClassColumnar WorkloadClass` (types.go:128)
- [ ] `const WorkloadClassUnknown WorkloadClass` (types.go:127)
- [ ] `func (r Report) Duration() time.Duration` (types.go:207)
- [ ] `func (s *Session) CacheSnapshot() CacheSnapshot` (cache.go:27)
- [ ] `func (s *Session) Close() error` (session.go:168)
- [ ] `func (s *Session) Closed() bool` (session.go:178)
- [ ] `func (s *Session) Config() Config` (session.go:58)
- [ ] `func (s *Session) Devices() []Device` (session.go:68)
- [ ] `func (s *Session) Discover() error` (discovery.go:41)
- [ ] `func (s *Session) ExecuteNearestExact(dataset *Dataset, queries [][]float64, m int, workload WorkloadEstimate) (ExactNearestResult, error)` (exact.go:157)
- [ ] `func (s *Session) LastReport() *Report` (session.go:101)
- [ ] `func (s *Session) PlanShardable() ShardPlan` (planner.go:21)
- [ ] `func (s *Session) PlanShardableWorkload(workload WorkloadEstimate) ShardPlan` (planner.go:25)
- [ ] `func (s *Session) ProjectDataList(dl *insyra.DataList) (*Dataset, error)` (dataset.go:10)
- [ ] `func (s *Session) ProjectDataTable(dt *insyra.DataTable) (*Dataset, error)` (dataset.go:35)
- [ ] `func (s *Session) RecordReport(report Report) error` (session.go:124)
- [ ] `func (s *Session) RegisterDevice(device Device) error` (session.go:111)
- [ ] `func (s *Session) Report() Report` (session.go:78)
- [ ] `func (s *Session) Reports() []Report` (session.go:91)
- [ ] `func Default() *Session` (default.go:27)
- [ ] `func DefaultConfig() Config` (types.go:324)
- [ ] `func DeviceMatMul(a []float32, aRows, aCols int, b []float32, bRows, bCols int) ([]float32, error)` (nn_matmul.go:15)
- [ ] `func NearestExactCPU(dataset *Dataset, queries [][]float64, m int) ([]uint32, []float64, int, error)` (exact.go:442)
- [ ] `func NewSession(cfgs ...Config) *Session` (session.go:34)
- [ ] `func Open(cfg Config) (*Session, error)` (session.go:26)
- [ ] `func RegisterBackendExecutor(backend Backend, executor BackendExecutor) error` (executor.go:83)
- [ ] `func RegisterDiscoverer(d Discoverer)` (discovery.go:20)
- [ ] `func RegisterSDKProbe(probe SDKProbe)` (sdk_probe.go:34)
- [ ] `func ResetBackendExecutorsForTest()` (executor.go:98)
- [ ] `func ResetDefaultForTest()` (default.go:50)
- [ ] `func ResetDiscoverersForTest()` (discovery.go:26)
- [ ] `func ResetSDKProbesForTest()` (sdk_probe.go:46)
- [ ] `type Backend string` (types.go:25)
- [ ] `type BackendExecutor interface { Name() string Execute(ctx context.Context, req ExecuteRequest) (ExecuteResponse, error) }` (executor.go:73)
- [ ] `type Buffer struct { Name string Type DataType Values any Nulls []bool Validity []byte StringOffsets []uint32 StringData []byte Len int }` (types.go:214)
- [ ] `type CacheDeviceUsage struct { DeviceID string ResidentBuffers int ResidentBytes uint64 BudgetBytes uint64 }` (types.go:248)
- [ ] `type CacheEntry struct { Key string DatasetName string DatasetID string Lineage string BufferName string Type DataType Len int ResidentBytes uint64 DeviceIDs []string DeviceResidentBytes map[string]uint64 LastAccess time.Time accessOrdinal uint64 }` (types.go:233)
- [ ] `type CacheSnapshot struct { ResidentBuffers int ResidentBytes uint64 BudgetBytes uint64 EvictedBuffers uint64 EvictedBytes uint64 DeviceUsage []CacheDeviceUsage Entries []CacheEntry }` (types.go:255)
- [ ] `type Config struct { Mode Mode ShardStrategy ShardStrategy PreferredBackends []Backend MemoryBudget MemoryBudgetPolicy Strict bool EnableFallback bool Devices []string PreferredDevices []string ReportHistorySize int DiscoveryTimeout time.Duration }` (types.go:152)
- [ ] `type DataType string` (types.go:62)
- [ ] `type Dataset struct { Name string Fingerprint string Lineage string Rows int Buffers []Buffer }` (types.go:225)
- [ ] `type Device struct { ID string Name string Vendor string Backend Backend ProbeSource ProbeSource Type DeviceType MemoryClass MemoryClass SharedMemory bool BudgetBytes uint64 Score float64 CapabilitySummary map[string]bool DriverVersion string ComputeCapability string PCIBusID string }` (types.go:175)
- [ ] `type DeviceType string` (types.go:35)
- [ ] `type Discoverer interface { Name() string Discover(cfg Config) ([]Device, error) }` (discovery.go:10)
- [ ] `type ExactNearestResult struct { ExecutionResult Index []uint32 Distance []float64 Rows int Queries int M int Rechecked int }` (exact.go:18)
- [ ] `type ExecuteColumn struct { Name string Values []float32 }` (executor.go:22)
- [ ] `type ExecuteRequest struct { Op Op Device Device Columns []ExecuteColumn Precision Precision Queries [][]float32 Shortlist int }` (executor.go:30)
- [ ] `type ExecuteResponse struct { Reductions map[string]float64 Distances []float32 NearestIndex []uint32 ShortlistIndex []uint32 ShortlistDistance []float32 ShortlistBoundary []float32 Transfer time.Duration Dispatch time.Duration Readback time.Duration BytesUploaded uint64 }` (executor.go:48)
- [ ] `type ExecutionResult struct { Accelerated bool FallbackReason FallbackReason MergePolicy MergePolicy Executor string ExecutorKind ExecutorKind Assignments []ShardAssignment DeviceIDs []string Chunks int Op Op Precision Precision Reductions map[string]float64 Counts map[string]int Transfer time.Duration Dispatch time.Duration Readback time.Duration BytesUploaded uint64 }` (types.go:293)
- [ ] `type ExecutorKind string` (types.go:139)
- [ ] `type FallbackReason string` (types.go:73)
- [ ] `type MemoryBudgetPolicy struct { DeviceFraction float64 SharedFraction float64 }` (types.go:147)
- [ ] `type MemoryClass string` (types.go:45)
- [ ] `type MergePolicy string` (types.go:131)
- [ ] `type Mode string` (types.go:5)
- [ ] `type Op string` (types.go:95)
- [ ] `type Precision string` (types.go:117)
- [ ] `type ProbeSource string` (types.go:53)
- [ ] `type Report struct { Mode Mode Accelerated bool SelectedBackend Backend DiscoveredDeviceIDs []string SelectedDeviceIDs []string SelectedDevices []string UnmatchedDeviceSelectors []UnmatchedDeviceSelector FallbackReason FallbackReason StartedAt time.Time FinishedAt time.Time GeneratedAt time.Time Metrics map[string]float64 }` (types.go:192)
- [ ] `type SDKProbe interface { Name() string Backend() Backend Probe(cfg Config) ([]Device, error) }` (sdk_probe.go:19)
- [ ] `type Session struct { mu sync.Mutex cfg Config devices []Device reports []Report cache *residentCache closed bool accelerationInfoLogged bool fallbackInfoLogged bool shared bool }` (session.go:12)
- [ ] `type ShardAssignment struct { DeviceID string Backend Backend Weight float64 SharePercent float64 Rows int Bytes uint64 BudgetBytes uint64 RowStart int RowEnd int WallTime time.Duration FallbackReason FallbackReason Chunks int }` (types.go:278)
- [ ] `type ShardPlan struct { Accelerated bool Backend Backend DeviceIDs []string Assignments []ShardAssignment TotalBudgetBytes uint64 Heterogeneous bool MergePolicy MergePolicy FallbackReason FallbackReason }` (planner.go:10)
- [ ] `type ShardStrategy string` (types.go:17)
- [ ] `type UnmatchedDeviceSelector struct { Bound string Selector string }` (types.go:170)
- [ ] `type WorkloadClass string` (types.go:124)
- [ ] `type WorkloadEstimate struct { Class WorkloadClass Rows int Bytes uint64 Dimensions int Op Op Precision Precision }` (types.go:265)
- [ ] `var ErrBufferTooLarge` (executor.go:15)
- [ ] `var ErrNativeProbeUnavailable` (discoverers_builtin.go:10)
- [ ] `var ErrReadbackTimeout` (executor.go:16)
- [ ] `var ErrSDKProbeUnavailable` (sdk_probe.go:12)
- [ ] `var ErrShaderCompile` (executor.go:14)
- [ ] `func (d builtinDiscoverer) Discover(cfg Config) ([]Device, error)` (discoverers_builtin.go:80)
- [ ] `func (d builtinDiscoverer) Name() string` (discoverers_builtin.go:76)
- [ ] `func (l *windowsNVMLLoader) Device(index int) (nvmlDeviceInfo, error)` (sdk_probe_nvml_windows.go:162)
- [ ] `func (l *windowsNVMLLoader) DeviceCount() (int, error)` (sdk_probe_nvml_windows.go:150)
- [ ] `func (l *windowsNVMLLoader) DriverVersion() (string, error)` (sdk_probe_nvml_windows.go:137)
- [ ] `func (l *windowsNVMLLoader) Init() error` (sdk_probe_nvml_windows.go:111)
- [ ] `func (l *windowsNVMLLoader) Shutdown() error` (sdk_probe_nvml_windows.go:118)
- [ ] `func (p nvmlSDKProbe) Backend() Backend` (sdk_probe_nvml.go:44)
- [ ] `func (p nvmlSDKProbe) Name() string` (sdk_probe_nvml.go:43)
- [ ] `func (p nvmlSDKProbe) Probe(cfg Config) ([]Device, error)` (sdk_probe_nvml.go:46)
- [ ] `func (p wgpuProbe) Backend() Backend` (backend_wgpu.go:51)
- [ ] `func (p wgpuProbe) Name() string` (backend_wgpu.go:49)
- [ ] `func (p wgpuProbe) Probe(_ Config) ([]Device, error)` (backend_wgpu.go:53)
- [ ] `func (wgpuExecutor) Execute(ctx context.Context, req ExecuteRequest) (ExecuteResponse, error)` (backend_wgpu.go:104)
- [ ] `func (wgpuExecutor) Name() string` (backend_wgpu.go:102)

## accel/knnbridge (0)


## allpkgs (0)


## benchmark (0)


## cli (2)

- [ ] `func Execute() error` (root.go:19)
- [ ] `func NewRootCommand() *cobra.Command` (root.go:23)

## cli/commands (8)

- [ ] `func BuildCobraCommands(ctx *ExecContext) []*cobra.Command` (registry.go:86)
- [ ] `func CloseAllDBConns(ctx *ExecContext)` (db_conn.go:184)
- [ ] `func Dispatch(ctx *ExecContext, name string, args []string) error` (registry.go:66)
- [ ] `func Register(handler *CommandHandler) error` (registry.go:49)
- [ ] `type CommandHandler struct { Name string Aliases []string Usage string Description string Forms []string Examples []string DisableFlagParsing bool Run func(ctx *ExecContext, args []string) error }` (registry.go:29)
- [ ] `type DBConn struct { Name string Dialect string DSN string DB *gorm.DB }` (db_conn.go:15)
- [ ] `type ExecContext struct { Vars map[string]any DBConns map[string]*DBConn EnvName string EnvPath string Output io.Writer InREPL bool OpenREPL func(ctx *ExecContext) error Env *env.Manager }` (registry.go:14)
- [ ] `var Registry` (registry.go:47)

## cli/env (60)

- [ ] `func (m *Manager) AppendHistory(envName, command string) error` (state.go:161)
- [ ] `func (m *Manager) BasePath() (string, error)` (manager.go:98)
- [ ] `func (m *Manager) Clear(name string, keepHistory bool) error` (manager.go:228)
- [ ] `func (m *Manager) Create(name string) error` (manager.go:180)
- [ ] `func (m *Manager) Delete(name string) error` (manager.go:214)
- [ ] `func (m *Manager) EnsureBaseStructure() error` (manager.go:153)
- [ ] `func (m *Manager) EnsureDefaultEnvironment() error` (manager.go:141)
- [ ] `func (m *Manager) EnvsDirName() string` (manager.go:112)
- [ ] `func (m *Manager) EnvsPath() (string, error)` (manager.go:122)
- [ ] `func (m *Manager) Exists(name string) bool` (manager.go:171)
- [ ] `func (m *Manager) Export(name, outputPath string) error` (manager.go:319)
- [ ] `func (m *Manager) GlobalConfigPath() (string, error)` (config.go:38)
- [ ] `func (m *Manager) Import(inputPath, targetName string, force bool) (string, error)` (manager.go:370)
- [ ] `func (m *Manager) Info(name string) (EnvironmentInfo, error)` (manager.go:297)
- [ ] `func (m *Manager) List() ([]EnvironmentInfo, error)` (manager.go:269)
- [ ] `func (m *Manager) LoadGlobalConfig() (GlobalConfig, error)` (config.go:46)
- [ ] `func (m *Manager) LoadState(envName string) (*State, error)` (state.go:57)
- [ ] `func (m *Manager) Open(name string) (string, error)` (manager.go:200)
- [ ] `func (m *Manager) ReadHistory(envName string) ([]string, error)` (state.go:178)
- [ ] `func (m *Manager) Rename(oldName, newName string) error` (manager.go:248)
- [ ] `func (m *Manager) ResolveEnvPath(name string) (string, error)` (manager.go:130)
- [ ] `func (m *Manager) RestoreVariables(envName string) (map[string]any, error)` (state.go:91)
- [ ] `func (m *Manager) SaveGlobalConfig(cfg GlobalConfig) error` (config.go:80)
- [ ] `func (m *Manager) SaveState(envName string, vars map[string]any) error` (state.go:25)
- [ ] `func (m *Manager) SetBasePath(path string)` (manager.go:83)
- [ ] `func (m *Manager) SetEnvsDirName(name string)` (manager.go:92)
- [ ] `func (m *Manager) UpdateGlobalConfig(key, value string) (GlobalConfig, error)` (config.go:95)
- [ ] `func AppendHistory(envName, command string) error` (state.go:222)
- [ ] `func BasePath() (string, error)` (manager.go:523)
- [ ] `func Clear(name string, keepHistory bool) error` (manager.go:534)
- [ ] `func Create(name string) error` (manager.go:531)
- [ ] `func Default() *Manager` (manager.go:72)
- [ ] `func Delete(name string) error` (manager.go:533)
- [ ] `func EnsureBaseStructure() error` (manager.go:529)
- [ ] `func EnsureDefaultEnvironment() error` (manager.go:528)
- [ ] `func EnvsPath() (string, error)` (manager.go:524)
- [ ] `func Exists(name string) bool` (manager.go:530)
- [ ] `func Export(name, outputPath string) error` (manager.go:538)
- [ ] `func GlobalConfigPath() (string, error)` (config.go:124)
- [ ] `func Import(inputPath, targetName string, force bool) (string, error)` (manager.go:539)
- [ ] `func Info(name string) (EnvironmentInfo, error)` (manager.go:537)
- [ ] `func List() ([]EnvironmentInfo, error)` (manager.go:536)
- [ ] `func LoadGlobalConfig() (GlobalConfig, error)` (config.go:126)
- [ ] `func LoadState(envName string) (*State, error)` (state.go:214)
- [ ] `func NewManager(basePath, envsDirName string) *Manager` (manager.go:61)
- [ ] `func Open(name string) (string, error)` (manager.go:532)
- [ ] `func ReadHistory(envName string) ([]string, error)` (state.go:226)
- [ ] `func Rename(oldName, newName string) error` (manager.go:535)
- [ ] `func ResolveEnvPath(name string) (string, error)` (manager.go:525)
- [ ] `func RestoreVariables(envName string) (map[string]any, error)` (state.go:218)
- [ ] `func SaveGlobalConfig(cfg GlobalConfig) error` (config.go:128)
- [ ] `func SaveState(envName string, vars map[string]any) error` (state.go:210)
- [ ] `func SetBasePath(path string)` (manager.go:78)
- [ ] `func UpdateGlobalConfig(key, value string) (GlobalConfig, error)` (config.go:130)
- [ ] `type EnvironmentInfo struct { Name string Path string LastAccess time.Time VariableCount int }` (manager.go:20)
- [ ] `type ExportPayload struct { SchemaVersion int `json:"schemaVersion"` ExportedAt string `json:"exportedAt"` Environment string `json:"environment"` State *State `json:"state"` History []string `json:"history"` Config json.RawMessage `json:"config"` }` (manager.go:27)
- [ ] `type GlobalConfig struct { DefaultEnv string `json:"defaultEnv"` LogLevel string `json:"logLevel"` NoColor bool `json:"noColor"` AccelMode string `json:"accelMode"` FetchTWIntervalMS int `json:"fetchTWIntervalMS"` }` (config.go:17)
- [ ] `type Manager struct { mu sync.RWMutex basePath string envsDirName string }` (manager.go:44)
- [ ] `type SerializedVariable struct { Type string `json:"type"` Name string `json:"name,omitempty"` Data any `json:"data"` }` (state.go:14)
- [ ] `type State struct { Variables map[string]SerializedVariable `json:"variables"` LastAccess string `json:"lastAccess"` }` (state.go:20)

## cli/repl (8)

- [ ] `func (session *DSLSession) Context() *commands.ExecContext` (api.go:85)
- [ ] `func (session *DSLSession) Execute(line string) error` (api.go:62)
- [ ] `func (session *DSLSession) ExecuteFile(path string) error` (api.go:92)
- [ ] `func NewAutoCompleter(ctx *commands.ExecContext) readline.AutoCompleter` (completer.go:17)
- [ ] `func NewDSLSession(mgr *env.Manager, envName string, output io.Writer) (*DSLSession, error)` (api.go:24)
- [ ] `func Start(ctx *commands.ExecContext) error` (repl.go:15)
- [ ] `type DSLSession struct { ctx *commands.ExecContext }` (api.go:15)
- [ ] `func (c *simpleCompleter) Do(line []rune, pos int) ([][]rune, int)` (completer.go:21)

## cli/style (2)

- [ ] `func ErrorText(message string) string` (output.go:15)
- [ ] `func WarningText(message string) string` (output.go:19)

## cmd/insyra (0)


## csvxl (9)

- [x] `const Auto` (convert.go:25) — OK。但三個常數是裸 string，非 typed `Encoding`，拼錯字不會被編譯器擋。見 C-6
- [x] `const Big5` (convert.go:24) — 同上
- [x] `const UTF8` (convert.go:23) — 同上
- [x] `func AppendCsvToExcel(csvFiles []string, sheetNames []string, existingFile string, csvEncoding ...string) error` (convert.go:86) — C-1 失敗被吞、C-2 variadic 選項、C-3 doc 說會覆蓋現有 sheet 但實際只覆蓋重疊的儲存格、C-4 excelize File 未 Close、C-5 %v 包錯誤、C-10 只吃路徑、C-11 平行切片、C-12 命名
- [x] `func CsvToExcel(csvFiles []string, sheetNames []string, output string, csvEncoding ...string) error` (convert.go:31) — C-1、C-2、C-5；另外個別檔案失敗後仍存檔並回傳成功數字給 log、C-10、C-11、C-12
- [x] `func EachCsvToOneExcel(dir string, output string, encoding ...string) error` (convertDir.go:16) — C-2；Glob 只抓 `*.csv` 小寫；OK 其餘委派 CsvToExcel、C-12 名稱要想一下
- [x] `func EachExcelToCsv(dir string, outputDir string) error` (convertDir.go:32) — C-4 每個 xlsx 開了不關；log 標成 EachCsvToOneExcel（錯名）；C-7 0777
- [x] `func ExcelToCsv(excelFile string, outputDir string, csvNames []string, onlyContainSheets ...string) error` (convert.go:135) — C-4 未 Close；onlyContainSheets 指到不存在的 sheet 靜默略過（打錯字得到零輸出無錯誤）；C-7 0777；C-5、C-10、C-11、C-12
- [x] `func ReadCsvToString(filePath string, encoding ...string) (string, error)` (read_csv.go:11) — C-2；缺 doc comment；錯誤已用 %w（此檔正確）、C-10、C-12

## datafetch (89)

- [ ] `const SortByHighestRating GoogleMapsStoreReviewSortBy` (googleMapsCommentCrawler.go:50)
- [ ] `const SortByLowestRating GoogleMapsStoreReviewSortBy` (googleMapsCommentCrawler.go:52)
- [ ] `const SortByNewest GoogleMapsStoreReviewSortBy` (googleMapsCommentCrawler.go:48)
- [ ] `const SortByRelevance GoogleMapsStoreReviewSortBy` (googleMapsCommentCrawler.go:46)
- [ ] `const TWMarketAuto TWMarket` (twstock.go:77)
- [ ] `const TWMarketTPEx TWMarket` (twstock.go:76)
- [ ] `const TWMarketTWSE TWMarket` (twstock.go:75)
- [ ] `const YFPeriodAnnual YFPeriod` (yfinance.go:57)
- [ ] `const YFPeriodQuarterly YFPeriod` (yfinance.go:59)
- [ ] `const YFPeriodYearly YFPeriod` (yfinance.go:58)
- [ ] `func (e *RateLimitError) Error() string` (geocoding_errors.go:33)
- [ ] `func (e *RateLimitError) Unwrap() error` (geocoding_errors.go:42)
- [ ] `func (r *ReverseGeocodeResult) ToDataTable() *insyra.DataTable` (geocoding.go:102)
- [ ] `func (reviews GoogleMapsStoreReviews) ToDataTable() *insyra.DataTable` (googleMapsCommentCrawler.go:315)
- [ ] `func GoogleMapsStores() *googleMapsStoreCrawler` (googleMapsCommentCrawler.go:69)
- [ ] `func NewFileGeocodeCache(path string) GeocodeCache` (geocoding.go:583)
- [ ] `func NewMemoryGeocodeCache() GeocodeCache` (geocoding.go:544)
- [ ] `func TWGeocoding(cfg TWGeocodingConfig) (*twGeocoder, error)` (geocoding.go:150)
- [ ] `func TWStock(cfg TWStockConfig) (*twStock, error)` (twstock.go:94)
- [ ] `func YFinance(cfg YFinanceConfig) (*yahooFinance, error)` (yfinance.go:108)
- [ ] `type GeocodeCache interface { Get(key string) (*ReverseGeocodeResult, bool) Set(key string, r *ReverseGeocodeResult) }` (geocoding.go:530)
- [ ] `type GoogleMapsStoreData struct { ID string Name string }` (googleMapsCommentCrawler.go:62)
- [ ] `type GoogleMapsStoreReview struct { Reviewer string `json:"reviewer"` ReviewerID string `json:"reviewer_id"` ReviewerState string `json:"reviewer_state"` ReviewerLevel int `json:"reviewer_level"` ReviewTime string `json:"review_time"` ReviewDate string `json:"review_date"` Content string `json:"content"` Rating int `json:"rating"` }` (googleMapsCommentCrawler.go:21)
- [ ] `type GoogleMapsStoreReviewSortBy uint8` (googleMapsCommentCrawler.go:42)
- [ ] `type GoogleMapsStoreReviews []GoogleMapsStoreReview` (googleMapsCommentCrawler.go:33)
- [ ] `type GoogleMapsStoreReviewsFetchingOptions struct { SortBy GoogleMapsStoreReviewSortBy MaxWaitingInterval_Milliseconds uint }` (googleMapsCommentCrawler.go:36)
- [ ] `type RateLimitError struct { Limit int Remaining int ResetAt time.Time }` (geocoding_errors.go:27)
- [ ] `type ReverseGeocodeResult struct { Lat float64 `json:"lat"` Lng float64 `json:"lng"` VillCode string `json:"villcode"` CountyName string `json:"county_name"` TownName string `json:"town_name"` VillageName string `json:"village_name"` VillageEng string `json:"village_eng"` CountyID string `json:"county_id"` CountyCode string `json:"county_code"` TownID string `json:"town_id"` TownCode string `json:"town_code"` }` (geocoding.go:86)
- [ ] `type TWGeocodingConfig struct { Timeout time.Duration Interval time.Duration UserAgent string Retries int RetryBackoff time.Duration BaseURL string Cache GeocodeCache }` (geocoding.go:39)
- [ ] `type TWMarket string` (twstock.go:72)
- [ ] `type TWStockConfig struct { Timeout time.Duration Interval time.Duration UserAgent string Retries int RetryBackoff time.Duration Concurrency int }` (twstock.go:27)
- [ ] `type YFFinancialStatementTables struct { Values *insyra.DataTable Items *insyra.DataTable Meta *insyra.DataTable }` (yfinance_tables.go:22)
- [ ] `type YFHistoryParams = models.HistoryParams` (yfinance.go:49)
- [ ] `type YFOptionChainTables struct { Calls *insyra.DataTable Puts *insyra.DataTable Underlying *insyra.DataTable Expiration time.Time }` (yfinance_tables.go:14)
- [ ] `type YFPeriod string` (yfinance.go:54)
- [ ] `type YFinanceConfig struct { Timeout time.Duration Interval time.Duration UserAgent string Retries int RetryBackoff time.Duration Concurrency int }` (yfinance.go:28)
- [ ] `var ErrGeocodeNotFound` (geocoding_errors.go:15)
- [ ] `var ErrGeocodeRateLimited` (geocoding_errors.go:20)
- [ ] `var ErrGeocodeTimeout` (geocoding_errors.go:17)
- [ ] `var ErrInvalidSymbol` (yfinance_errors.go:13)
- [ ] `var ErrRateLimited` (yfinance_errors.go:11)
- [ ] `var ErrTimeout` (yfinance_errors.go:12)
- [ ] `func (c *fileGeocodeCache) Get(key string) (*ReverseGeocodeResult, bool)` (geocoding.go:601)
- [ ] `func (c *fileGeocodeCache) Set(key string, r *ReverseGeocodeResult)` (geocoding.go:615)
- [ ] `func (c *googleMapsStoreCrawler) GetReviews(storeId string, pageCount int, options ...GoogleMapsStoreReviewsFetchingOptions) GoogleMapsStoreReviews` (googleMapsCommentCrawler.go:169)
- [ ] `func (c *googleMapsStoreCrawler) Search(storeName string) []GoogleMapsStoreData` (googleMapsCommentCrawler.go:107)
- [ ] `func (c *memoryGeocodeCache) Get(key string) (*ReverseGeocodeResult, bool)` (geocoding.go:548)
- [ ] `func (c *memoryGeocodeCache) Set(key string, r *ReverseGeocodeResult)` (geocoding.go:562)
- [ ] `func (g *twGeocoder) Reverse(lat, lng float64) (*ReverseGeocodeResult, error)` (geocoding.go:166)
- [ ] `func (g *twGeocoder) ReverseCols(lat, lng *insyra.DataList) (*insyra.DataTable, error)` (geocoding.go:282)
- [ ] `func (g *twGeocoder) ReverseTable(dt *insyra.DataTable, latCol, lngCol string) (*insyra.DataTable, error)` (geocoding.go:410)
- [ ] `func (g *twGeocoder) ReverseTableByColName(dt *insyra.DataTable, latColName, lngColName string) (*insyra.DataTable, error)` (geocoding.go:425)
- [ ] `func (t *ticker) Actions() (*insyra.DataTable, error)` (yfinance.go:425)
- [ ] `func (t *ticker) AnalystPriceTargets() (*insyra.DataTable, error)` (yfinance.go:838)
- [ ] `func (t *ticker) BalanceSheet(freq YFPeriod) (*YFFinancialStatementTables, error)` (yfinance.go:558)
- [ ] `func (t *ticker) Calendar() (*insyra.DataTable, error)` (yfinance.go:515)
- [ ] `func (t *ticker) CashFlow(freq YFPeriod) (*YFFinancialStatementTables, error)` (yfinance.go:576)
- [ ] `func (t *ticker) Dividends() (*insyra.DataTable, error)` (yfinance.go:377)
- [ ] `func (t *ticker) EPSRevisions() (*insyra.DataTable, error)` (yfinance.go:792)
- [ ] `func (t *ticker) EPSTrend() (*insyra.DataTable, error)` (yfinance.go:768)
- [ ] `func (t *ticker) Earnings() (*insyra.DataTable, error)` (yfinance.go:712)
- [ ] `func (t *ticker) EarningsEstimate() (*insyra.DataTable, error)` (yfinance.go:720)
- [ ] `func (t *ticker) EarningsHistory() (*insyra.DataTable, error)` (yfinance.go:744)
- [ ] `func (t *ticker) FastInfo() (*insyra.DataTable, error)` (yfinance.go:687)
- [ ] `func (t *ticker) FundsData() (*insyra.DataTable, error)` (yfinance.go:917)
- [ ] `func (t *ticker) GrowthEstimates() (*insyra.DataTable, error)` (yfinance.go:893)
- [ ] `func (t *ticker) History(params YFHistoryParams) (*insyra.DataTable, error)` (yfinance.go:275)
- [ ] `func (t *ticker) IncomeStatement(freq YFPeriod) (*YFFinancialStatementTables, error)` (yfinance.go:540)
- [ ] `func (t *ticker) Info() (*insyra.DataTable, error)` (yfinance.go:353)
- [ ] `func (t *ticker) InsiderTransactions() (*insyra.DataTable, error)` (yfinance.go:663)
- [ ] `func (t *ticker) InstitutionalHolders() (*insyra.DataTable, error)` (yfinance.go:617)
- [ ] `func (t *ticker) MajorHolders() (*insyra.DataTable, error)` (yfinance.go:594)
- [ ] `func (t *ticker) MutualFundHolders() (*insyra.DataTable, error)` (yfinance.go:640)
- [ ] `func (t *ticker) News(count int, tab models.NewsTab) (*insyra.DataTable, error)` (yfinance.go:491)
- [ ] `func (t *ticker) OptionChain(date string) (*YFOptionChainTables, error)` (yfinance.go:473)
- [ ] `func (t *ticker) Options() (*insyra.DataTable, error)` (yfinance.go:449)
- [ ] `func (t *ticker) Quote() (*insyra.DataTable, error)` (yfinance.go:314)
- [ ] `func (t *ticker) Recommendations() (*insyra.DataTable, error)` (yfinance.go:815)
- [ ] `func (t *ticker) RevenueEstimate() (*insyra.DataTable, error)` (yfinance.go:861)
- [ ] `func (t *ticker) Splits() (*insyra.DataTable, error)` (yfinance.go:401)
- [ ] `func (t *ticker) Sustainability() (*insyra.DataTable, error)` (yfinance.go:885)
- [ ] `func (t *ticker) TopHoldings() (*insyra.DataTable, error)` (yfinance.go:926)
- [ ] `func (t *twStock) AllDailyQuotes(market TWMarket) (*insyra.DataTable, error)` (twstock.go:851)
- [ ] `func (t *twStock) DailyPrices(code string, from, to time.Time, market TWMarket) (*insyra.DataTable, error)` (twstock.go:193)
- [ ] `func (t *twStock) DailyPricesAdjusted(code string, from, to time.Time, market TWMarket) (*insyra.DataTable, error)` (twstock.go:492)
- [ ] `func (t *twStock) ExRights(from, to time.Time, market TWMarket) (*insyra.DataTable, error)` (twstock.go:361)
- [ ] `func (t *twStock) InstitutionalTrades(date time.Time, market TWMarket) (*insyra.DataTable, error)` (twstock.go:589)
- [ ] `func (t *twStock) MarginBalance(date time.Time, market TWMarket) (*insyra.DataTable, error)` (twstock.go:723)
- [ ] `func (y *yahooFinance) Ticker(symbol string) *ticker` (yfinance.go:251)

## engine (0)


## engine/algorithms (3)

- [ ] `func CompareAny(a, b any) int` (algorithms.go:11)
- [ ] `func GetTypeSortingRank(v any) int` (algorithms.go:6)
- [ ] `func ParallelSortStableFunc[S ~[]E, E any](x S, cmp func(E, E) int)` (algorithms.go:16)

## engine/atomic (9)

- [ ] `func AtomicDoN(actors []*Actor, f func())` (atomic.go:51)
- [ ] `func AtomicDoNWithInit(actors []*Actor, initHooks []func(), f func())` (atomic.go:57)
- [ ] `func AtomicDoWithInit[T any](actor *Actor, owner *T, f func(*T), initHook func())` (atomic.go:33)
- [ ] `func AtomicDo[T any](actor *Actor, owner *T, f func(*T))` (atomic.go:28)
- [ ] `func DefaultGroup() *Group` (atomic.go:17)
- [ ] `func NewActor(group *Group) *Actor` (atomic.go:23)
- [ ] `func NewGroup() *Group` (atomic.go:12)
- [ ] `type Actor = core.AtomicActor` (atomic.go:9)
- [ ] `type Group = core.AtomicGroup` (atomic.go:6)

## engine/biindex (2)

- [ ] `func NewBiIndex(cap int) *BiIndex` (biindex.go:9)
- [ ] `type BiIndex = core.BiIndex` (biindex.go:6)

## engine/ccl (23)

- [ ] `func Bind(n CCLNode, colNameMap map[string]int) (CCLNode, error)` (ccl.go:29)
- [ ] `func CompileExpression(expression string) (CCLNode, error)` (ccl.go:19)
- [ ] `func CompileMultiline(script string) ([]CCLNode, error)` (ccl.go:24)
- [ ] `func Evaluate(n CCLNode, ctx Context) (any, error)` (ccl.go:34)
- [ ] `func EvaluateStatement(n CCLNode, ctx Context) (*EvaluationResult, error)` (ccl.go:39)
- [ ] `func GetAssignmentTarget(n CCLNode) (string, bool)` (ccl.go:44)
- [ ] `func GetExpressionNode(n CCLNode) CCLNode` (ccl.go:54)
- [ ] `func GetNewColInfo(n CCLNode) (string, CCLNode, bool)` (ccl.go:49)
- [ ] `func IsAssignmentNode(n CCLNode) bool` (ccl.go:59)
- [ ] `func IsNewColNode(n CCLNode) bool` (ccl.go:64)
- [ ] `func IsRowDependent(n CCLNode) bool` (ccl.go:69)
- [ ] `func NewMapContext(data map[string][]any) (*MapContext, error)` (ccl.go:14)
- [ ] `func RegisterAggregateFunction(name string, fn AggFunc)` (ccl.go:84)
- [ ] `func RegisterFunction(name string, fn Func)` (ccl.go:79)
- [ ] `func RegisterStandardFunctions()` (ccl.go:74)
- [ ] `func ResetEvalDepth()` (ccl.go:91)
- [ ] `func ResetFuncCallDepth()` (ccl.go:96)
- [ ] `type AggFunc = internalccl.AggFunc` (ccl.go:11)
- [ ] `type CCLNode = internalccl.CCLNode` (ccl.go:7)
- [ ] `type Context = internalccl.Context` (ccl.go:6)
- [ ] `type EvaluationResult = internalccl.EvaluationResult` (ccl.go:8)
- [ ] `type Func = internalccl.Func` (ccl.go:10)
- [ ] `type MapContext = internalccl.MapContext` (ccl.go:9)

## engine/dsl (2)

- [ ] `func NewSession(mgr *env.Manager, envName string, output io.Writer) (*Session, error)` (dsl.go:24)
- [ ] `type Session = repl.DSLSession` (dsl.go:13)

## engine/ring (2)

- [ ] `func NewRing[T any](capacity int) *Ring[T]` (ring.go:10)
- [ ] `type Ring[T any] = core.Ring[T]` (ring.go:7)

## finance (60)

- [x] `const Basis30_360EU DayCountBasis` (daycount.go:29) — OK
- [x] `const Basis30_360US DayCountBasis` (daycount.go:17) — OK
- [x] `const BasisActual360 DayCountBasis` (daycount.go:24) — OK
- [x] `const BasisActual365 DayCountBasis` (daycount.go:27) — OK
- [x] `const BasisActualActual DayCountBasis` (daycount.go:21) — OK
- [x] `const DefaultScale int32` (options.go:20) — OK
- [x] `const PaymentBegin PaymentTiming` (options.go:16) — OK
- [x] `const PaymentEnd PaymentTiming` (options.go:13) — OK
- [x] `const Round05Up RoundingMode` (options.go:62) — OK
- [x] `const RoundCeiling RoundingMode` (options.go:54) — OK
- [x] `const RoundDown RoundingMode` (options.go:51) — OK
- [x] `const RoundFloor RoundingMode` (options.go:57) — OK
- [x] `const RoundHalfDown RoundingMode` (options.go:45) — OK
- [x] `const RoundHalfEven RoundingMode` (options.go:42) — OK
- [x] `const RoundHalfUp RoundingMode` (options.go:37) — OK
- [x] `const RoundUnnecessary RoundingMode` (options.go:66) — FI-3 panic 模式
- [x] `const RoundUp RoundingMode` (options.go:48) — OK
- [x] `func AccrInt(issue, firstInterest, settlement time.Time, rate, par decimal.Decimal, freq int, basis DayCountBasis, calcMethod bool, opts ...Options) (decimal.Decimal, error)` (bonds.go:399) — OK 驗證 + error（FI-4）；FI-1 decimal 型別
- [x] `func AmortizationSchedule(rate decimal.Decimal, nper int, pv, fv decimal.Decimal, timing PaymentTiming, opts ...Options) ([]AmortizationRow, error)` (amortization.go:22) — OK 驗證 + error（FI-4）；FI-1 decimal 型別
- [x] `func AnnualFromContinuous(continuous decimal.Decimal, opts ...Options) (decimal.Decimal, error)` (rate.go:83) — OK 驗證 + error（FI-4）；FI-1 decimal 型別
- [x] `func ContinuousFromAnnual(effective decimal.Decimal, opts ...Options) (decimal.Decimal, error)` (rate.go:69) — OK 驗證 + error（FI-4）；FI-1 decimal 型別
- [x] `func CumIPMT(rate decimal.Decimal, nper int, pv decimal.Decimal, startPeriod, endPeriod int, timing PaymentTiming, opts ...Options) (decimal.Decimal, error)` (ipmt.go:95) — OK 驗證 + error（FI-4）；FI-1 decimal 型別
- [x] `func CumPPMT(rate decimal.Decimal, nper int, pv decimal.Decimal, startPeriod, endPeriod int, timing PaymentTiming, opts ...Options) (decimal.Decimal, error)` (ipmt.go:103) — OK 驗證 + error（FI-4）；FI-1 decimal 型別
- [x] `func DDB(cost, salvage decimal.Decimal, life, per int, factor decimal.Decimal, opts ...Options) (decimal.Decimal, error)` (depreciation.go:65) — OK 驗證 + error（FI-4）；FI-1 decimal 型別
- [x] `func Duration(settlement, maturity time.Time, coupon, yld decimal.Decimal, freq int, basis DayCountBasis, opts ...Options) (decimal.Decimal, error)` (bonds.go:278) — OK 驗證 + error（FI-4）；FI-1 decimal 型別
- [x] `func EffectiveRate(nominal decimal.Decimal, periodsPerYear int, opts ...Options) (decimal.Decimal, error)` (rate.go:15) — OK 驗證 + error（FI-4）；FI-1 decimal 型別
- [x] `func FV(rate decimal.Decimal, nper int, pmt, pv decimal.Decimal, timing PaymentTiming, opts ...Options) (decimal.Decimal, error)` (tvm.go:147) — OK 驗證 + error（FI-4）；FI-1 decimal 型別
- [x] `func FromFloat(f float64) (decimal.Decimal, error)` (helpers.go:49) — OK 驗證 + error（FI-4）；FI-1 decimal 型別
- [x] `func FromInt(n int) decimal.Decimal` (helpers.go:42) — OK 驗證 + error（FI-4）；FI-1 decimal 型別
- [x] `func IPMT(rate decimal.Decimal, per, nper int, pv, fv decimal.Decimal, timing PaymentTiming, opts ...Options) (decimal.Decimal, error)` (ipmt.go:16) — OK 驗證 + error（FI-4）；FI-1 decimal 型別
- [x] `func IRR(cashflows []decimal.Decimal, guess decimal.Decimal, opts ...Options) (decimal.Decimal, error)` (npv.go:57) — OK 驗證 + error（FI-4）；FI-1 decimal 型別
- [x] `func MDuration(settlement, maturity time.Time, coupon, yld decimal.Decimal, freq int, basis DayCountBasis, opts ...Options) (decimal.Decimal, error)` (bonds.go:367) — OK 驗證 + error（FI-4）；FI-1 decimal 型別
- [x] `func MIRR(cashflows []decimal.Decimal, financeRate, reinvestRate decimal.Decimal, opts ...Options) (decimal.Decimal, error)` (mirr.go:22) — OK 驗證 + error（FI-4）；FI-1 decimal 型別
- [x] `func MustNew(s string) decimal.Decimal` (helpers.go:36) — OK 命名依 Go 慣例（Must 前綴）
- [x] `func NPER(rate, pmt, pv, fv decimal.Decimal, timing PaymentTiming, opts ...Options) (decimal.Decimal, error)` (tvm.go:167) — OK 驗證 + error（FI-4）；FI-1 decimal 型別
- [x] `func NPV(rate decimal.Decimal, cashflows []decimal.Decimal, opts ...Options) (decimal.Decimal, error)` (npv.go:18) — OK 驗證 + error（FI-4）；FI-1 decimal 型別
- [x] `func NPVExcel(rate decimal.Decimal, cashflows []decimal.Decimal, opts ...Options) (decimal.Decimal, error)` (npv.go:34) — OK 驗證 + error（FI-4）；FI-1 decimal 型別
- [x] `func New(s string) (decimal.Decimal, error)` (helpers.go:30) — OK 驗證 + error（FI-4）；FI-1 decimal 型別
- [x] `func NominalRate(effective decimal.Decimal, periodsPerYear int, opts ...Options) (decimal.Decimal, error)` (rate.go:42) — OK 驗證 + error（FI-4）；FI-1 decimal 型別
- [x] `func PMT(rate decimal.Decimal, nper int, pv, fv decimal.Decimal, timing PaymentTiming, opts ...Options) (decimal.Decimal, error)` (tvm.go:82) — OK 驗證 + error（FI-4）；FI-1 decimal 型別
- [x] `func PPMT(rate decimal.Decimal, per, nper int, pv, fv decimal.Decimal, timing PaymentTiming, opts ...Options) (decimal.Decimal, error)` (ipmt.go:55) — OK 驗證 + error（FI-4）；FI-1 decimal 型別
- [x] `func PV(rate decimal.Decimal, nper int, pmt, fv decimal.Decimal, timing PaymentTiming, opts ...Options) (decimal.Decimal, error)` (tvm.go:102) — OK 驗證 + error（FI-4）；FI-1 decimal 型別
- [x] `func Price(settlement, maturity time.Time, rate, yld, redemption decimal.Decimal, freq int, basis DayCountBasis, opts ...Options) (decimal.Decimal, error)` (bonds.go:91) — OK 驗證 + error（FI-4）；FI-1 decimal 型別
- [x] `func RATE(nper int, pmt, pv, fv decimal.Decimal, timing PaymentTiming, guess decimal.Decimal, opts ...Options) (decimal.Decimal, error)` (tvm.go:230) — OK 驗證 + error（FI-4）；FI-1 decimal 型別
- [x] `func SLN(cost, salvage decimal.Decimal, life int, opts ...Options) (decimal.Decimal, error)` (depreciation.go:14) — OK 驗證 + error（FI-4）；FI-1 decimal 型別
- [x] `func SYD(cost, salvage decimal.Decimal, life, per int, opts ...Options) (decimal.Decimal, error)` (depreciation.go:34) — OK 驗證 + error（FI-4）；FI-1 decimal 型別
- [x] `func ScheduleTable(rate decimal.Decimal, nper int, pv, fv decimal.Decimal, timing PaymentTiming, opts ...Options) (insyra.IDataTable, error)` (amortization.go:97) — FI-2
- [x] `func TBillEq(settlement, maturity time.Time, discount decimal.Decimal, opts ...Options) (decimal.Decimal, error)` (tbill.go:28) — OK 驗證 + error（FI-4）；FI-1 decimal 型別
- [x] `func TBillPrice(settlement, maturity time.Time, discount decimal.Decimal, opts ...Options) (decimal.Decimal, error)` (tbill.go:98) — OK 驗證 + error（FI-4）；FI-1 decimal 型別
- [x] `func TBillYield(settlement, maturity time.Time, pr decimal.Decimal, opts ...Options) (decimal.Decimal, error)` (tbill.go:125) — OK 驗證 + error（FI-4）；FI-1 decimal 型別
- [x] `func VDB(cost, salvage decimal.Decimal, life int, startPeriod, endPeriod, factor decimal.Decimal, noSwitch bool, opts ...Options) (decimal.Decimal, error)` (depreciation.go:117) — OK 驗證 + error（FI-4）；FI-1 decimal 型別
- [x] `func XIRR(values []decimal.Decimal, dates []time.Time, guess decimal.Decimal, opts ...Options) (decimal.Decimal, error)` (xnpv.go:36) — OK 驗證 + error（FI-4）；FI-1 decimal 型別
- [x] `func XNPV(rate decimal.Decimal, values []decimal.Decimal, dates []time.Time, opts ...Options) (decimal.Decimal, error)` (xnpv.go:19) — OK 驗證 + error（FI-4）；FI-1 decimal 型別
- [x] `func Yield(settlement, maturity time.Time, rate, pr, redemption decimal.Decimal, freq int, basis DayCountBasis, guess decimal.Decimal, opts ...Options) (decimal.Decimal, error)` (bonds.go:211) — OK 驗證 + error（FI-4）；FI-1 decimal 型別
- [x] `type AmortizationRow struct { Period int Payment decimal.Decimal Interest decimal.Decimal Principal decimal.Decimal Balance decimal.Decimal }` (amortization.go:11) — OK
- [x] `type DayCountBasis uint8` (daycount.go:13) — OK
- [x] `type Options struct { Scale int32 Mode RoundingMode }` (options.go:100) — OK 零值可用（範本）；FI-3 variadic
- [x] `type PaymentTiming uint8` (options.go:8) — OK
- [x] `type RoundingMode string` (options.go:32) — OK
- [x] `var Zero` (helpers.go:22) — FI-3 可覆寫

## gplot (22)

- [ ] `func CreateBarChart(config BarChartConfig, data any) *plot.Plot` (bar.go:24)
- [ ] `func CreateFunctionPlot(config FunctionPlotConfig, function func(x float64) float64) *plot.Plot` (function.go:22)
- [ ] `func CreateHeatmapChart(config HeatmapChartConfig, data any) *plot.Plot` (heatmap.go:61)
- [ ] `func CreateHistogram(config HistogramConfig, data any) *plot.Plot` (histogram.go:19)
- [ ] `func CreateLineChart(config LineChartConfig, data any) *plot.Plot` (line.go:24)
- [ ] `func CreateScatterPlot(config ScatterPlotConfig, data any) *plot.Plot` (scatter.go:24)
- [ ] `func CreateStepChart(config StepChartConfig, data any) *plot.Plot` (step.go:25)
- [ ] `func SaveChart(plt *plot.Plot, filename string)` (save_chart.go:14)
- [ ] `type BarChartConfig struct { Title string XAxis []string XAxisName string YAxisName string BarWidth float64 ErrorBars []float64 }` (bar.go:13)
- [ ] `type FunctionPlotConfig struct { Title string XAxisName string YAxisName string XMin float64 XMax float64 YMin float64 YMax float64 }` (function.go:11)
- [ ] `type HeatmapChartConfig struct { Title string XAxisName string YAxisName string XAxis []float64 YAxis []float64 Colors int Alpha float64 }` (heatmap.go:13)
- [ ] `type HistogramConfig struct { Title string XAxisName string YAxisName string Bins int }` (histogram.go:10)
- [ ] `type LineChartConfig struct { Title string XAxis []float64 XAxisName string YAxisName string }` (line.go:15)
- [ ] `type ScatterPlotConfig struct { Title string XAxisName string YAxisName string }` (scatter.go:16)
- [ ] `type StepChartConfig struct { Title string XAxis []float64 XAxisName string YAxisName string StepStyle string }` (step.go:15)
- [ ] `func (d *barErrorData) Len() int` (bar.go:97)
- [ ] `func (d *barErrorData) XY(i int) (float64, float64)` (bar.go:101)
- [ ] `func (d *barErrorData) YError(i int) (float64, float64)` (bar.go:105)
- [ ] `func (g *gridData) Dims() (c, r int)` (heatmap.go:31)
- [ ] `func (g *gridData) X(c int) float64` (heatmap.go:44)
- [ ] `func (g *gridData) Y(r int) float64` (heatmap.go:52)
- [ ] `func (g *gridData) Z(c, r int) float64` (heatmap.go:39)

## isr (44)

- [x] `func Name(value string) name` (name.go:7) — I-3 回傳未匯出型別
- [x] `func PtrDL[T *insyra.DataList | dl](l T) *dl` (dl.go:20) — 已 Deprecated：OK
- [x] `func PtrDT[T *insyra.DataTable | dt](t T) *dt` (dt.go:36) — 已 Deprecated：OK
- [x] `func UseDL[T *insyra.DataList | dl](l T) *dl` (use.go:8) — I-1 LogFatal；I-2 回傳未匯出型別
- [x] `func UseDT[T *insyra.DataTable | dt](t T) *dt` (use.go:23) — I-1 LogFatal；I-2 回傳未匯出型別
- [x] `type CSV struct { FilePath string String string InputOpts CSV_inOpts OutputOpts CSV_outOpts }` (csv.go:3) — I-4 命名；I-5 OutputOpts 無用
- [x] `type CSV_inOpts struct { FirstCol2RowNames bool FirstRow2ColNames bool Encoding string RawStrings bool AllowRaggedRows bool TrimLeadingSpace bool }` (csv.go:10) — I-4 命名；I-5 OutputOpts 無用
- [x] `type CSV_outOpts struct { RowNames2FirstCol bool ColNames2FirstRow bool }` (csv.go:19) — I-4 命名；I-5 OutputOpts 無用
- [x] `type Col map[any]any` (dt.go:28) — I-3 map[any]any
- [x] `type Cols = []Col` (dt.go:31) — OK 別名
- [x] `type DLs = []insyra.IDataList` (dl.go:10) — I-4 混用 IDataList
- [x] `type Excel struct { FilePath string SheetName string InputOpts Excel_inOpts }` (excel.go:4) — I-4 命名；I-5 OutputOpts 無用
- [x] `type Excel_inOpts struct { FirstCol2RowNames bool FirstRow2ColNames bool }` (excel.go:10) — I-4 命名；I-5 OutputOpts 無用
- [x] `type JSON struct { FilePath string Bytes []byte }` (json.go:3) — OK 輸入描述 struct；I-4
- [x] `type Pivot struct { Index []string Columns string Values string Agg string Custom func(group *insyra.DataList) any FillNA any SortCols bool }` (pivot.go:8) — OK（I-6）；Pivot.Agg 仍是字串（T-17）
- [x] `type Rolling struct { Window int MinObs int Center bool Weights []float64 }` (window.go:17) — OK（I-6）；Pivot.Agg 仍是字串（T-17）
- [x] `type Row map[any]any` (dt.go:20) — I-3 map[any]any
- [x] `type Rows = []Row` (dt.go:23) — OK 別名
- [x] `type Unpivot struct { IDVars []string ValueVars []string VarName string ValueName string DropNA bool }` (pivot.go:20) — OK（I-6）；Pivot.Agg 仍是字串（T-17）
- [x] `var DL` (dl.go:6) — I-2 用 var 當命名空間，型別未匯出
- [x] `var DT` (dt.go:11) — I-2 用 var 當命名空間，型別未匯出
- [x] `func (d dl) From(data ...any) *dl` (dl.go:26) — I-1 LogFatal；I-3 any；Of 是別名（I-4）
- [x] `func (d dl) Of(data ...any) *dl` (dl.go:34) — I-1 LogFatal；I-3 any；Of 是別名（I-4）
- [x] `func (d dt) From(item any) *dt` (dt.go:42) — I-1 LogFatal；I-3 any；Of 是別名（I-4）
- [x] `func (d dt) Of(item any) *dt` (dt.go:162) — I-1 LogFatal；I-3 any；Of 是別名（I-4）
- [x] `func (l *dl) At(index int) any` (dl.go:39) — I-3 any×2；錯誤只 warn 回 nil
- [x] `func (l *dl) Push(data ...any) *dl` (dl.go:44) — I-1；I-3；I-4 別名
- [x] `func (t *dt) At(row any, col any) any` (dt.go:199) — I-3 any×2；錯誤只 warn 回 nil
- [x] `func (t *dt) CCL(cclStatements string) *dt` (dt.go:324) — OK 轉發；E-1/E-2 繼承
- [x] `func (t *dt) Col(col any) *dl` (dt.go:167) — I-1 LogFatal；I-3
- [x] `func (t *dt) CumMax(col string) *insyra.DataList` (window.go:50) — OK 轉發（I-6）
- [x] `func (t *dt) CumMin(col string) *insyra.DataList` (window.go:55) — OK 轉發（I-6）
- [x] `func (t *dt) CumProd(col string) *insyra.DataList` (window.go:45) — OK 轉發（I-6）
- [x] `func (t *dt) CumSum(col string) *insyra.DataList` (window.go:40) — OK 轉發（I-6）
- [x] `func (t *dt) Diff(col string, periods int) *insyra.DataList` (window.go:30) — OK 轉發（I-6）
- [x] `func (t *dt) ExpandingOn(col string, minObs int) *insyra.ExpandingDataList` (window.go:72) — OK（I-6）
- [x] `func (t *dt) GroupBy(keyCols ...string) *insyra.GroupedDataTable` (groupby.go:12) — OK（I-6）
- [x] `func (t *dt) PctChange(col string, periods int) *insyra.DataList` (window.go:35) — OK 轉發（I-6）
- [x] `func (t *dt) Pivot(p Pivot) *dt` (pivot.go:31) — I-5 丟掉 error
- [x] `func (t *dt) Push(data any) *dt` (dt.go:239) — I-1；I-3；I-4 別名
- [x] `func (t *dt) RollingOn(col string, r Rolling) *insyra.RollingDataList` (window.go:62) — OK（I-6）
- [x] `func (t *dt) Row(row any) *dl` (dt.go:184) — I-1 LogFatal；I-3
- [x] `func (t *dt) Shift(col string, periods int, fill ...any) *insyra.DataList` (window.go:25) — OK 轉發（I-6）
- [x] `func (t *dt) Unpivot(u Unpivot) *dt` (pivot.go:46) — I-5 丟掉 error

## lp (2)

- [ ] `func SolveFromFile(lpFile string, timeoutSeconds ...int) (*insyra.DataTable, *insyra.DataTable)` (lp.go:21)
- [ ] `func SolveModel(model *lpgen.LPModel, timeoutSeconds ...int) (*insyra.DataTable, *insyra.DataTable)` (lp.go:83)

## lpgen (10)

- [ ] `func (lp *LPModel) AddBinaryVar(varName string) *LPModel` (lpgen.go:44)
- [ ] `func (lp *LPModel) AddBound(bound string) *LPModel` (lpgen.go:38)
- [ ] `func (lp *LPModel) AddConstraint(constr string) *LPModel` (lpgen.go:32)
- [ ] `func (lp *LPModel) AddIntegerVar(varName string) *LPModel` (lpgen.go:57)
- [ ] `func (lp *LPModel) GenerateLPFile(filename string)` (lpgen.go:70)
- [ ] `func (lp *LPModel) SetObjective(objType, obj string) *LPModel` (lpgen.go:25)
- [ ] `func NewLPModel() *LPModel` (lpgen.go:20)
- [ ] `func ParseLingoModel_str(modelStr string) *LPModel` (lingo.go:120)
- [ ] `func ParseLingoModel_txt(filePath string) *LPModel` (lingo.go:15)
- [ ] `type LPModel struct { Objective string ObjectiveType string Constraints []string Bounds []string BinaryVars []string IntegerVars []string }` (lpgen.go:11)

## mkt (14)

- [x] `const TimeScaleDaily TimeScale` (time_scale.go:9) — OK
- [x] `const TimeScaleHourly TimeScale` (time_scale.go:8) — OK
- [x] `const TimeScaleMonthly TimeScale` (time_scale.go:11) — OK
- [x] `const TimeScaleWeekly TimeScale` (time_scale.go:10) — OK
- [x] `const TimeScaleYearly TimeScale` (time_scale.go:12) — OK
- [x] `func BasketAnalysis(dt insyra.IDataTable, config BasketConfig) *BasketResult` (basket.go:38) — MK-1 無 error；輸出有排序：OK
- [x] `func CustomerActivityIndex(dt insyra.IDataTable, caiConfig CAIConfig) insyra.IDataTable` (cai.go:38) — MK-1 無 error；MK-2
- [x] `func RFM(dt insyra.IDataTable, rfmConfig RFMConfig) insyra.IDataTable` (rfm.go:26) — MK-1 panic（已實測）、無 error；MK-2 順序不定
- [x] `type BasketConfig struct { OrderIDColIndex string OrderIDColName string ProductIDColIndex string ProductIDColName string }` (basket.go:12) — MK-3 Index/Name 雙欄位
- [x] `type BasketResult struct { Support insyra.IDataTable Confidence insyra.IDataTable Lift insyra.IDataTable }` (basket.go:21) — OK；欄位為 IDataTable（K-7）
- [x] `type CAIConfig struct { CustomerIDColIndex string CustomerIDColName string TradingDayColIndex string TradingDayColName string DateFormat string TimeScale TimeScale }` (cai.go:12) — MK-3 Index/Name 雙欄位
- [x] `type RFMConfig struct { CustomerIDColIndex string CustomerIDColName string TradingDayColIndex string TradingDayColName string AmountColIndex string AmountColName string NumGroups uint DateFormat string TimeScale TimeScale }` (rfm.go:12) — MK-3 Index/Name 雙欄位
- [x] `type TimeScale string` (time_scale.go:5) — OK
- [x] `var CAI` (cai.go:22) — MK-3 可覆寫的函式變數

## ml (230)

- [ ] `const BinaryAverage` (model_selection.go:474)
- [ ] `const ClassificationMetric MetricKind` (model_selection.go:22)
- [ ] `const HigherIsBetter` (model_selection.go:66)
- [ ] `const LowerIsBetter` (model_selection.go:69)
- [ ] `const MacroAverage ClassAverage` (model_selection.go:461)
- [ ] `const MicroAverage` (model_selection.go:465)
- [ ] `const NoDirection MetricDirection` (model_selection.go:64)
- [ ] `const RegressionMetric MetricKind` (model_selection.go:24)
- [ ] `const WeightedAverage` (model_selection.go:468)
- [ ] `func (AccuracyMetric) Direction() MetricDirection` (model_selection.go:714)
- [ ] `func (AccuracyMetric) Evaluate(yTrue *insyra.DataList, prediction Prediction) (MetricResult, error)` (model_selection.go:715)
- [ ] `func (AccuracyMetric) Kind() MetricKind` (model_selection.go:713)
- [ ] `func (AccuracyMetric) Name() string` (model_selection.go:712)
- [ ] `func (AccuracyMetric) NeedsClassLabels() bool` (model_selection.go:719)
- [ ] `func (ConfusionMatrixMetric) Direction() MetricDirection` (model_selection.go:762)
- [ ] `func (ConfusionMatrixMetric) Evaluate(yTrue *insyra.DataList, prediction Prediction) (MetricResult, error)` (model_selection.go:763)
- [ ] `func (ConfusionMatrixMetric) Kind() MetricKind` (model_selection.go:761)
- [ ] `func (ConfusionMatrixMetric) Name() string` (model_selection.go:760)
- [ ] `func (ConfusionMatrixMetric) NeedsClassLabels() bool` (model_selection.go:767)
- [ ] `func (F1Metric) Direction() MetricDirection` (model_selection.go:636)
- [ ] `func (F1Metric) Kind() MetricKind` (model_selection.go:635)
- [ ] `func (F1Metric) Name() string` (model_selection.go:634)
- [ ] `func (F1Metric) NeedsClassLabels() bool` (model_selection.go:637)
- [ ] `func (LogLossMetric) Direction() MetricDirection` (model_selection.go:726)
- [ ] `func (LogLossMetric) Evaluate(yTrue *insyra.DataList, prediction Prediction) (MetricResult, error)` (model_selection.go:727)
- [ ] `func (LogLossMetric) Kind() MetricKind` (model_selection.go:725)
- [ ] `func (LogLossMetric) Name() string` (model_selection.go:724)
- [ ] `func (LogLossMetric) NeedsProbabilities() bool` (model_selection.go:731)
- [ ] `func (MAEMetric) Direction() MetricDirection` (model_selection.go:785)
- [ ] `func (MAEMetric) Evaluate(yTrue *insyra.DataList, prediction Prediction) (MetricResult, error)` (model_selection.go:786)
- [ ] `func (MAEMetric) Kind() MetricKind` (model_selection.go:784)
- [ ] `func (MAEMetric) Name() string` (model_selection.go:783)
- [ ] `func (PrecisionMetric) Direction() MetricDirection` (model_selection.go:604)
- [ ] `func (PrecisionMetric) Kind() MetricKind` (model_selection.go:603)
- [ ] `func (PrecisionMetric) Name() string` (model_selection.go:602)
- [ ] `func (PrecisionMetric) NeedsClassLabels() bool` (model_selection.go:605)
- [ ] `func (R2Metric) Direction() MetricDirection` (model_selection.go:796)
- [ ] `func (R2Metric) Evaluate(yTrue *insyra.DataList, prediction Prediction) (MetricResult, error)` (model_selection.go:797)
- [ ] `func (R2Metric) Kind() MetricKind` (model_selection.go:795)
- [ ] `func (R2Metric) Name() string` (model_selection.go:794)
- [ ] `func (RMSEMetric) Direction() MetricDirection` (model_selection.go:774)
- [ ] `func (RMSEMetric) Evaluate(yTrue *insyra.DataList, prediction Prediction) (MetricResult, error)` (model_selection.go:775)
- [ ] `func (RMSEMetric) Kind() MetricKind` (model_selection.go:773)
- [ ] `func (RMSEMetric) Name() string` (model_selection.go:772)
- [ ] `func (ROCAUCMetric) Direction() MetricDirection` (model_selection.go:749)
- [ ] `func (ROCAUCMetric) Evaluate(yTrue *insyra.DataList, prediction Prediction) (MetricResult, error)` (model_selection.go:750)
- [ ] `func (ROCAUCMetric) Kind() MetricKind` (model_selection.go:748)
- [ ] `func (ROCAUCMetric) Name() string` (model_selection.go:747)
- [ ] `func (ROCAUCMetric) NeedsProbabilities() bool` (model_selection.go:754)
- [ ] `func (RecallMetric) Direction() MetricDirection` (model_selection.go:620)
- [ ] `func (RecallMetric) Kind() MetricKind` (model_selection.go:619)
- [ ] `func (RecallMetric) Name() string` (model_selection.go:618)
- [ ] `func (RecallMetric) NeedsClassLabels() bool` (model_selection.go:621)
- [ ] `func (a ClassAverage) String() string` (model_selection.go:478)
- [ ] `func (d MetricDirection) String() string` (model_selection.go:73)
- [ ] `func (m *DecisionTreeClassifier) Classes() *insyra.DataList` (decision_tree.go:905)
- [ ] `func (m *DecisionTreeClassifier) ExportONNX(w io.Writer) error` (onnx_export.go:65)
- [ ] `func (m *DecisionTreeClassifier) FeatureImportances() []float64` (decision_tree.go:947)
- [ ] `func (m *DecisionTreeClassifier) LeafValues() []float64` (decision_tree.go:956)
- [ ] `func (m *DecisionTreeClassifier) Predict(dt *insyra.DataTable) (*insyra.DataList, error)` (decision_tree.go:912)
- [ ] `func (m *DecisionTreeClassifier) PredictProba(dt *insyra.DataTable) (*insyra.DataTable, error)` (decision_tree.go:927)
- [ ] `func (m *DecisionTreeRegressor) ExportONNX(w io.Writer) error` (onnx_export.go:68)
- [ ] `func (m *DecisionTreeRegressor) FeatureImportances() []float64` (decision_tree.go:982)
- [ ] `func (m *DecisionTreeRegressor) LeafValues() []float64` (decision_tree.go:989)
- [ ] `func (m *DecisionTreeRegressor) Predict(dt *insyra.DataTable) (*insyra.DataList, error)` (decision_tree.go:967)
- [ ] `func (m *ExponentialModel) Predict(dt *insyra.DataTable) (*insyra.DataList, error)` (models.go:393)
- [ ] `func (m *GLMModel) Predict(dt *insyra.DataTable) (*insyra.DataList, error)` (models.go:443)
- [ ] `func (m *GradientBoostingClassifier) Classes() *insyra.DataList` (gradient_boosting.go:394)
- [ ] `func (m *GradientBoostingClassifier) ExportONNX(w io.Writer) error` (onnx_export.go:63)
- [ ] `func (m *GradientBoostingClassifier) FeatureImportances() []float64` (gradient_boosting.go:401)
- [ ] `func (m *GradientBoostingClassifier) Predict(dt *insyra.DataTable) (*insyra.DataList, error)` (gradient_boosting.go:364)
- [ ] `func (m *GradientBoostingClassifier) PredictProba(dt *insyra.DataTable) (*insyra.DataTable, error)` (gradient_boosting.go:380)
- [ ] `func (m *GradientBoostingRegressor) ExportONNX(w io.Writer) error` (onnx_export.go:60)
- [ ] `func (m *GradientBoostingRegressor) FeatureImportances() []float64` (gradient_boosting.go:339)
- [ ] `func (m *GradientBoostingRegressor) Predict(dt *insyra.DataTable) (*insyra.DataList, error)` (gradient_boosting.go:327)
- [ ] `func (m *KMeansModel) Clusters() int` (models.go:451)
- [ ] `func (m *KMeansModel) Predict(dt *insyra.DataTable) (*insyra.DataList, error)` (models.go:459)
- [ ] `func (m *KNNClassifier) Classes() *insyra.DataList` (models.go:486)
- [ ] `func (m *KNNClassifier) Predict(dt *insyra.DataTable) (*insyra.DataList, error)` (models.go:478)
- [ ] `func (m *KNNClassifier) PredictProba(dt *insyra.DataTable) (*insyra.DataTable, error)` (models.go:493)
- [ ] `func (m *KNNRegressor) Predict(dt *insyra.DataTable) (*insyra.DataList, error)` (models.go:515)
- [ ] `func (m *LassoModel) ExportONNX(w io.Writer) error` (onnx_export.go:48)
- [ ] `func (m *LassoModel) Predict(dt *insyra.DataTable) (*insyra.DataList, error)` (models.go:381)
- [ ] `func (m *LinearModel) ExportONNX(w io.Writer) error` (onnx_export.go:46)
- [ ] `func (m *LinearModel) Predict(dt *insyra.DataTable) (*insyra.DataList, error)` (models.go:363)
- [ ] `func (m *LogarithmicModel) Predict(dt *insyra.DataTable) (*insyra.DataList, error)` (models.go:399)
- [ ] `func (m *LogisticModel) Classes() *insyra.DataList` (models.go:411)
- [ ] `func (m *LogisticModel) ExportONNX(w io.Writer) error` (onnx_export.go:64)
- [ ] `func (m *LogisticModel) Predict(dt *insyra.DataTable) (*insyra.DataList, error)` (models.go:405)
- [ ] `func (m *LogisticModel) PredictProba(dt *insyra.DataTable) (*insyra.DataTable, error)` (models.go:418)
- [ ] `func (m *PoissonModel) Predict(dt *insyra.DataTable) (*insyra.DataList, error)` (models.go:437)
- [ ] `func (m *PolynomialModel) Predict(dt *insyra.DataTable) (*insyra.DataList, error)` (models.go:387)
- [ ] `func (m *RandomForestClassifier) Classes() *insyra.DataList` (random_forest.go:256)
- [ ] `func (m *RandomForestClassifier) ExportONNX(w io.Writer) error` (onnx_export.go:54)
- [ ] `func (m *RandomForestClassifier) FeatureImportances() []float64` (random_forest.go:263)
- [ ] `func (m *RandomForestClassifier) Predict(dt *insyra.DataTable) (*insyra.DataList, error)` (random_forest.go:223)
- [ ] `func (m *RandomForestClassifier) PredictProba(dt *insyra.DataTable) (*insyra.DataTable, error)` (random_forest.go:241)
- [ ] `func (m *RandomForestRegressor) ExportONNX(w io.Writer) error` (onnx_export.go:57)
- [ ] `func (m *RandomForestRegressor) FeatureImportances() []float64` (random_forest.go:289)
- [ ] `func (m *RandomForestRegressor) Predict(dt *insyra.DataTable) (*insyra.DataList, error)` (random_forest.go:270)
- [ ] `func (m *RidgeModel) ExportONNX(w io.Writer) error` (onnx_export.go:47)
- [ ] `func (m *RidgeModel) Predict(dt *insyra.DataTable) (*insyra.DataList, error)` (models.go:375)
- [ ] `func (m *WeightedLinearModel) ExportONNX(w io.Writer) error` (onnx_export.go:51)
- [ ] `func (m *WeightedLinearModel) Predict(dt *insyra.DataTable) (*insyra.DataList, error)` (models.go:369)
- [ ] `func (m F1Metric) Evaluate(yTrue *insyra.DataList, prediction Prediction) (MetricResult, error)` (model_selection.go:638)
- [ ] `func (m PrecisionMetric) Evaluate(yTrue *insyra.DataList, prediction Prediction) (MetricResult, error)` (model_selection.go:606)
- [ ] `func (m RecallMetric) Evaluate(yTrue *insyra.DataList, prediction Prediction) (MetricResult, error)` (model_selection.go:622)
- [ ] `func (p *PCATransformer) Features() []string` (models.go:539)
- [ ] `func (p *PCATransformer) Transform(dt *insyra.DataTable) (*insyra.DataTable, error)` (models.go:546)
- [ ] `func (t *ColumnTransformer) Transform(dt *insyra.DataTable) (*insyra.DataTable, error)` (pipeline.go:106)
- [ ] `func Accuracy(yTrue, yPred *insyra.DataList) (float64, error)` (model_selection.go:381)
- [ ] `func Better(a, b *CrossValidationResult) (bool, error)` (model_selection.go:118)
- [ ] `func ConfusionMatrix(yTrue, yPred *insyra.DataList) (*ConfusionMatrixResult, error)` (model_selection.go:419)
- [ ] `func CrossValidate(x *insyra.DataTable, y *insyra.DataList, estimator Estimator, k int, metric Metric, options ...insyra.SamplingOptions) (*CrossValidationResult, error)` (model_selection.go:188)
- [ ] `func CrossValidateWeighted(x *insyra.DataTable, y *insyra.DataList, weights *insyra.DataList, estimator Estimator, k int, metric Metric, options ...insyra.SamplingOptions) (*CrossValidationResult, error)` (model_selection.go:344)
- [ ] `func ExportONNX(w io.Writer, fitted any) error` (onnx_export.go:23)
- [ ] `func F1(yTrue, yPred *insyra.DataList) (float64, error)` (model_selection.go:590)
- [ ] `func FitDecisionTreeClassifier(x *insyra.DataTable, y *insyra.DataList, opts ...DecisionTreeOptions) (*DecisionTreeClassifier, error)` (decision_tree.go:83)
- [ ] `func FitDecisionTreeRegressor(x *insyra.DataTable, y *insyra.DataList, opts ...DecisionTreeOptions) (*DecisionTreeRegressor, error)` (decision_tree.go:102)
- [ ] `func FitExponentialRegression(x *insyra.DataTable, y *insyra.DataList) (Model, error)` (models.go:188)
- [ ] `func FitGLM(x *insyra.DataTable, y *insyra.DataList, opts GLMOptions) (Model, error)` (models.go:263)
- [ ] `func FitGradientBoostingClassifier(x *insyra.DataTable, y *insyra.DataList, opts ...GradientBoostingOptions) (*GradientBoostingClassifier, error)` (gradient_boosting.go:168)
- [ ] `func FitGradientBoostingRegressor(x *insyra.DataTable, y *insyra.DataList, opts ...GradientBoostingOptions) (*GradientBoostingRegressor, error)` (gradient_boosting.go:94)
- [ ] `func FitKMeans(x *insyra.DataTable, k int, opts ...KMeansOptions) (Model, error)` (models.go:278)
- [ ] `func FitKNNClassifier(x *insyra.DataTable, y *insyra.DataList, k int, opts ...KNNOptions) (ProbaModel, error)` (models.go:311)
- [ ] `func FitKNNRegressor(x *insyra.DataTable, y *insyra.DataList, k int, opts ...KNNOptions) (Model, error)` (models.go:337)
- [ ] `func FitLassoRegression(x *insyra.DataTable, y *insyra.DataList, alpha float64, options ...stats.LassoOptions) (Model, error)` (models.go:176)
- [ ] `func FitLinearRegression(x *insyra.DataTable, y *insyra.DataList) (Model, error)` (models.go:110)
- [ ] `func FitLogarithmicRegression(x *insyra.DataTable, y *insyra.DataList) (Model, error)` (models.go:203)
- [ ] `func FitLogisticRegression(x *insyra.DataTable, y *insyra.DataList, opts ...LogisticOptions) (ProbaModel, error)` (models.go:218)
- [ ] `func FitPCA(x *insyra.DataTable, components int) (Transformer, error)` (models.go:299)
- [ ] `func FitPoissonRegression(x *insyra.DataTable, y *insyra.DataList, opts ...PoissonOptions) (Model, error)` (models.go:239)
- [ ] `func FitPolynomialRegression(x *insyra.DataTable, y *insyra.DataList, degree int) (Model, error)` (models.go:122)
- [ ] `func FitRandomForestClassifier(x *insyra.DataTable, y *insyra.DataList, opts ...RandomForestOptions) (*RandomForestClassifier, error)` (random_forest.go:55)
- [ ] `func FitRandomForestRegressor(x *insyra.DataTable, y *insyra.DataList, opts ...RandomForestOptions) (*RandomForestRegressor, error)` (random_forest.go:70)
- [ ] `func FitRidgeRegression(x *insyra.DataTable, y *insyra.DataList, alpha float64) (Model, error)` (models.go:160)
- [ ] `func FitWeightedLinearRegression(x *insyra.DataTable, y *insyra.DataList, weights *insyra.DataList) (Model, error)` (models.go:145)
- [ ] `func GridSearch(x *insyra.DataTable, y *insyra.DataList, candidates []Estimator, k int, metric Metric, options ...insyra.SamplingOptions) (*GridSearchResult, error)` (grid_search.go:45)
- [ ] `func KFold(dt *insyra.DataTable, k int, options ...insyra.SamplingOptions) ([]*insyra.DataTable, error)` (model_selection.go:144)
- [ ] `func LogLoss(yTrue *insyra.DataList, probabilities *insyra.DataTable, classes ...*insyra.DataList) (float64, error)` (model_selection.go:398)
- [ ] `func MAE(yTrue, yPred *insyra.DataList) (float64, error)` (model_selection.go:667)
- [ ] `func NewColumnTransformer(transformer Transformer, columns ...string) *ColumnTransformer` (pipeline.go:93)
- [ ] `func NewPipeline(steps []Step, estimator Estimator) Estimator` (pipeline.go:13)
- [ ] `func Precision(yTrue, yPred *insyra.DataList) (float64, error)` (model_selection.go:577)
- [ ] `func R2(yTrue, yPred *insyra.DataList) (float64, error)` (model_selection.go:683)
- [ ] `func RMSE(yTrue, yPred *insyra.DataList) (float64, error)` (model_selection.go:650)
- [ ] `func ROCAUC(yTrue *insyra.DataList, probabilities *insyra.DataTable, classes ...*insyra.DataList) (float64, error)` (model_selection.go:409)
- [ ] `func Recall(yTrue, yPred *insyra.DataList) (float64, error)` (model_selection.go:584)
- [ ] `func Score(model Model, x *insyra.DataTable, y *insyra.DataList, metric Metric) (MetricResult, error)` (model_selection.go:297)
- [ ] `func StratifiedKFold(dt *insyra.DataTable, labels *insyra.DataList, k int, options ...insyra.SamplingOptions) ([]*insyra.DataTable, error)` (model_selection.go:166)
- [ ] `func TransformColumns(transformer Transformer, columns ...string) Transformer` (pipeline.go:101)
- [ ] `func WriteONNX(w io.Writer, fitted any) error` (onnx_export.go:44)
- [ ] `type AccuracyMetric struct{}` (model_selection.go:710)
- [ ] `type ClassAverage int` (model_selection.go:452)
- [ ] `type ClassLabelMetric interface { Metric NeedsClassLabels() bool }` (model_selection.go:813)
- [ ] `type Classifier interface { Model Classes() *insyra.DataList }` (interfaces.go:24)
- [ ] `type Clusterer interface { Model Clusters() int }` (interfaces.go:77)
- [ ] `type ColumnTransformer struct { transformer Transformer columns []string }` (pipeline.go:85)
- [ ] `type ConfusionMatrixMetric struct{}` (model_selection.go:758)
- [ ] `type ConfusionMatrixResult struct { Labels []any Counts [][]int }` (model_selection.go:136)
- [ ] `type CrossValidationResult struct { Metric string Direction MetricDirection Scores []float64 Mean float64 FoldResults []MetricResult }` (model_selection.go:104)
- [ ] `type DecisionTreeClassifier struct { modelBase Root *DecisionTreeNode classes *insyra.DataList featureSchemas []treeFeature featureImportances []float64 }` (decision_tree.go:64)
- [ ] `type DecisionTreeClassifierOptions = DecisionTreeOptions` (decision_tree.go:34)
- [ ] `type DecisionTreeNode struct { IsLeaf bool Feature int FeatureName string Categorical bool Threshold float64 Categories []any MissingGoLeft bool UnseenCategoryGoLeft bool Samples int Gain float64 Impurity float64 Value float64 Prediction any Probabilities []float64 ClassCounts []int64 Left *DecisionTreeNode Right *DecisionTreeNode splitBin int leftCategories map[uint32]struct{} }` (decision_tree.go:40)
- [ ] `type DecisionTreeOptions struct { MaxDepth int MaxLeaves int MinSamplesLeaf int MaxBins int ExactSplits bool CategoricalFeatures []string }` (decision_tree.go:16)
- [ ] `type DecisionTreeRegressor struct { modelBase Root *DecisionTreeNode featureSchemas []treeFeature featureImportances []float64 quantizerScale float64 }` (decision_tree.go:74)
- [ ] `type DecisionTreeRegressorOptions = DecisionTreeOptions` (decision_tree.go:35)
- [ ] `type Estimator struct { Name string Fit func(x *insyra.DataTable, y *insyra.DataList) (Model, error) FitWeighted func(x *insyra.DataTable, y *insyra.DataList, weights *insyra.DataList) (Model, error) }` (interfaces.go:97)
- [ ] `type ExponentialModel struct { Result *stats.ExponentialRegressionResult modelBase }` (models.go:42)
- [ ] `type Exporter interface { Model ExportONNX(w io.Writer) error }` (interfaces.go:84)
- [ ] `type F1Metric struct { Average ClassAverage PositiveClass any }` (model_selection.go:629)
- [ ] `type GLMModel struct { Result *stats.GLMResult modelBase }` (models.go:66)
- [ ] `type GLMOptions = stats.GLMOptions` (interfaces.go:113)
- [ ] `type GradientBoostingClassifier struct { modelBase classes *insyra.DataList base float64 learningRate float64 trees []*treeFit importances []float64 Stages int }` (gradient_boosting.go:46)
- [ ] `type GradientBoostingOptions struct { Stages int LearningRate float64 Tree DecisionTreeOptions }` (gradient_boosting.go:13)
- [ ] `type GradientBoostingRegressor struct { modelBase base float64 learningRate float64 trees []*treeFit importances []float64 Stages int }` (gradient_boosting.go:28)
- [ ] `type GridSearchResult struct { BestIndex int BestName string BestModel Model Results []*CrossValidationResult Seed uint64 }` (grid_search.go:13)
- [ ] `type Importances interface { Model FeatureImportances() []float64 }` (interfaces.go:45)
- [ ] `type InverseTransformer interface { InverseTransform(dt *insyra.DataTable) (*insyra.DataTable, error) }` (interfaces.go:30)
- [ ] `type KMeansModel struct { Result *stats.KMeansResult modelBase }` (models.go:72)
- [ ] `type KMeansOptions = stats.KMeansOptions` (interfaces.go:114)
- [ ] `type KNNClassifier struct { Result *stats.KNNClassificationResult modelBase train *insyra.DataTable labels *insyra.DataList k int options KNNOptions hasOption bool }` (models.go:87)
- [ ] `type KNNOptions = stats.KNNOptions` (interfaces.go:115)
- [ ] `type KNNRegressor struct { Result *stats.KNNRegressionResult modelBase train *insyra.DataTable targets *insyra.DataList k int options KNNOptions hasOption bool }` (models.go:100)
- [ ] `type LassoModel struct { Result *stats.LassoRegressionResult modelBase }` (models.go:36)
- [ ] `type LinearModel struct { Result *stats.LinearRegressionResult modelBase }` (models.go:12)
- [ ] `type LogLossMetric struct{}` (model_selection.go:722)
- [ ] `type LogarithmicModel struct { Result *stats.LogarithmicRegressionResult modelBase }` (models.go:48)
- [ ] `type LogisticModel struct { Result *stats.LogisticRegressionResult modelBase }` (models.go:54)
- [ ] `type LogisticOptions = stats.LogisticRegressionOptions` (interfaces.go:111)
- [ ] `type MAEMetric struct{}` (model_selection.go:781)
- [ ] `type Metric interface { Name() string Kind() MetricKind Direction() MetricDirection Evaluate(yTrue *insyra.DataList, prediction Prediction) (MetricResult, error) }` (model_selection.go:92)
- [ ] `type MetricDirection int` (model_selection.go:58)
- [ ] `type MetricKind string` (model_selection.go:17)
- [ ] `type MetricResult struct { Name string Score float64 ConfusionMatrix *ConfusionMatrixResult }` (model_selection.go:51)
- [ ] `type Model interface { Features() []string Predict(dt *insyra.DataTable) (*insyra.DataList, error) }` (interfaces.go:17)
- [ ] `type PCATransformer struct { Result *stats.PCAResult modelBase }` (models.go:79)
- [ ] `type PoissonModel struct { Result *stats.PoissonRegressionResult modelBase }` (models.go:60)
- [ ] `type PoissonOptions = stats.PoissonRegressionOptions` (interfaces.go:112)
- [ ] `type PolynomialModel struct { Result *stats.PolynomialRegressionResult modelBase }` (models.go:18)
- [ ] `type PrecisionMetric struct { Average ClassAverage PositiveClass any }` (model_selection.go:597)
- [ ] `type Prediction struct { Values *insyra.DataList Probabilities *insyra.DataTable Classes *insyra.DataList }` (model_selection.go:30)
- [ ] `type ProbaModel interface { Classifier PredictProba(dt *insyra.DataTable) (*insyra.DataTable, error) }` (interfaces.go:35)
- [ ] `type ProbabilityMetric interface { Metric NeedsProbabilities() bool }` (model_selection.go:827)
- [ ] `type R2Metric struct{}` (model_selection.go:792)
- [ ] `type RMSEMetric struct{}` (model_selection.go:770)
- [ ] `type ROCAUCMetric struct{}` (model_selection.go:745)
- [ ] `type RandomForestClassifier struct { modelBase trees []*treeFit classes *insyra.DataList importances []float64 Seed int64 }` (random_forest.go:34)
- [ ] `type RandomForestOptions struct { Trees int MaxFeatures int Seed *int64 Tree DecisionTreeOptions }` (random_forest.go:16)
- [ ] `type RandomForestRegressor struct { modelBase trees []*treeFit importances []float64 Seed int64 }` (random_forest.go:46)
- [ ] `type RecallMetric struct { Average ClassAverage PositiveClass any }` (model_selection.go:613)
- [ ] `type RidgeModel struct { Result *stats.RidgeRegressionResult modelBase }` (models.go:30)
- [ ] `type Step struct { Name string Fit func(x *insyra.DataTable, y *insyra.DataList) (Transformer, error) }` (interfaces.go:90)
- [ ] `type TransformedFeatures interface { Model TransformedFeatureNames() []string }` (interfaces.go:58)
- [ ] `type Transformer interface { Transform(dt *insyra.DataTable) (*insyra.DataTable, error) }` (interfaces.go:12)
- [ ] `type WeightedLinearModel struct { Result *stats.WeightedLinearRegressionResult modelBase }` (models.go:24)
- [ ] `func (m modelBase) Features() []string` (helpers.go:14)
- [ ] `func (p *fittedPipeline) ExportONNX(w io.Writer) error` (onnx_export.go:71)
- [ ] `func (p *fittedPipeline) Features() []string` (pipeline.go:215)
- [ ] `func (p *fittedPipeline) Predict(dt *insyra.DataTable) (*insyra.DataList, error)` (pipeline.go:233)
- [ ] `func (p *fittedPipeline) TransformedFeatureNames() []string` (pipeline.go:226)
- [ ] `func (p *fittedPipelineClassifier) Classes() *insyra.DataList` (pipeline.go:267)
- [ ] `func (p *fittedPipelineClassifierImportances) Classes() *insyra.DataList` (pipeline.go:311)
- [ ] `func (p *fittedPipelineClassifierImportances) FeatureImportances() []float64` (pipeline.go:319)
- [ ] `func (p *fittedPipelineImportances) FeatureImportances() []float64` (pipeline.go:299)
- [ ] `func (p *fittedPipelineProba) Classes() *insyra.DataList` (pipeline.go:277)
- [ ] `func (p *fittedPipelineProba) PredictProba(dt *insyra.DataTable) (*insyra.DataTable, error)` (pipeline.go:285)
- [ ] `func (p *fittedPipelineProbaImportances) Classes() *insyra.DataList` (pipeline.go:329)
- [ ] `func (p *fittedPipelineProbaImportances) FeatureImportances() []float64` (pipeline.go:349)
- [ ] `func (p *fittedPipelineProbaImportances) PredictProba(dt *insyra.DataTable) (*insyra.DataTable, error)` (pipeline.go:337)

## ml/mltest (1)

- [ ] `func RunConformance(t *testing.T, model ml.Model, x *insyra.DataTable, y *insyra.DataList)` (conformance.go:22)

## nn (273)

- [ ] `const DTypeBFloat16 DType` (tensor.go:30)
- [ ] `const DTypeBool DType` (tensor.go:27)
- [ ] `const DTypeFloat16 DType` (tensor.go:17)
- [ ] `const DTypeFloat32 DType` (tensor.go:16)
- [ ] `const DTypeFloat64 DType` (tensor.go:18)
- [ ] `const DTypeFloat8 DType` (tensor.go:29)
- [ ] `const DTypeInt16 DType` (tensor.go:21)
- [ ] `const DTypeInt32 DType` (tensor.go:23)
- [ ] `const DTypeInt64 DType` (tensor.go:25)
- [ ] `const DTypeInt8 DType` (tensor.go:19)
- [ ] `const DTypeString DType` (tensor.go:28)
- [ ] `const DTypeUInt16 DType` (tensor.go:22)
- [ ] `const DTypeUInt32 DType` (tensor.go:24)
- [ ] `const DTypeUInt64 DType` (tensor.go:26)
- [ ] `const DTypeUInt8 DType` (tensor.go:20)
- [ ] `const DTypeUnknown DType` (tensor.go:15)
- [ ] `const Float16` (tensor.go:37)
- [ ] `const Float32` (tensor.go:36)
- [ ] `const Float64` (tensor.go:38)
- [ ] `func (m *BoundClassifier) Classes() *insyra.DataList` (protocol.go:163)
- [ ] `func (m *BoundClassifier) Features() []string` (protocol.go:155)
- [ ] `func (m *BoundClassifier) Predict(dt *insyra.DataTable) (*insyra.DataList, error)` (protocol.go:172)
- [ ] `func (m *BoundClassifier) PredictProba(dt *insyra.DataTable) (*insyra.DataTable, error)` (protocol.go:198)
- [ ] `func (m *BoundRegressor) Features() []string` (protocol.go:49)
- [ ] `func (m *BoundRegressor) Predict(dt *insyra.DataTable) (*insyra.DataList, error)` (protocol.go:58)
- [ ] `func (m *Model) Inputs() []ValueInfo` (model.go:86)
- [ ] `func (m *Model) OpsetVersion() int64` (model.go:102)
- [ ] `func (m *Model) Outputs() []ValueInfo` (model.go:94)
- [ ] `func (m *Model) Run(inputs map[string]*Tensor) (outputs map[string]*Tensor, err error)` (model_run.go:9)
- [ ] `func (p *Parameter) Grad() *Tensor` (autodiff.go:39)
- [ ] `func (p *Parameter) Value() *Tensor` (autodiff.go:31)
- [ ] `func (s *CosineAnnealingLR) LR(step int) float32` (autodiff_practice.go:111)
- [ ] `func (s *Sequential) ExportONNX(w io.Writer) error` (onnx_export.go:15)
- [ ] `func (s *Sequential) Fit(x, y *Tensor, cfg FitConfig) (*FitResult, error)` (fit.go:200)
- [ ] `func (s *Sequential) Forward(t *Tape, x *Tensor) (*Tensor, error)` (sequential.go:56)
- [ ] `func (s *Sequential) LoadWeights(weights map[string]*Tensor) error` (sequential.go:146)
- [ ] `func (s *Sequential) NamedParameters() map[string]*Parameter` (sequential.go:119)
- [ ] `func (s *Sequential) Parameters() []*Parameter` (sequential.go:106)
- [ ] `func (s *Sequential) Predict(x *Tensor) (*Tensor, error)` (sequential.go:79)
- [ ] `func (s *Sequential) SaveWeights(w io.Writer) error` (sequential.go:202)
- [ ] `func (s *StepLR) LR(step int) float32` (autodiff_practice.go:79)
- [ ] `func (t *Tape) Adam(rate float32) error` (autodiff.go:671)
- [ ] `func (t *Tape) AdamW(rate, weightDecay float32) error` (autodiff.go:678)
- [ ] `func (t *Tape) Add(a, b *Tensor) (*Tensor, error)` (autodiff.go:124)
- [ ] `func (t *Tape) AveragePool(input *Tensor, kernelShape []int, options ...PoolOptions) (*Tensor, error)` (autodiff_cnn.go:50)
- [ ] `func (t *Tape) BCEWithLogitsLoss(logits, targets *Tensor) (*Tensor, error)` (autodiff.go:504)
- [ ] `func (t *Tape) Backward(loss *Tensor) error` (autodiff.go:521)
- [ ] `func (t *Tape) BatchNormTraining(input, scale, bias, runningMean, runningVariance *Tensor, options ...float32) (*Tensor, error)` (autodiff_cnn.go:176)
- [ ] `func (t *Tape) BatchNormalization(input, scale, bias, mean, variance *Tensor, epsilonValues ...float32) (*Tensor, error)` (autodiff_cnn.go:82)
- [ ] `func (t *Tape) BatchNormalizationTraining(input, scale, bias, runningMean, runningVariance *Tensor, options ...float32) (*Tensor, error)` (autodiff_cnn.go:101)
- [ ] `func (t *Tape) ClipGradNorm(maxNorm float32) (float32, error)` (autodiff.go:637)
- [ ] `func (t *Tape) Concat(inputs []*Tensor, axis int) (*Tensor, error)` (autodiff.go:391)
- [ ] `func (t *Tape) Conv(input, weights, bias *Tensor, options ...ConvOptions) (*Tensor, error)` (autodiff_cnn.go:11)
- [ ] `func (t *Tape) Div(a, b *Tensor) (*Tensor, error)` (autodiff.go:112)
- [ ] `func (t *Tape) Dropout(input *Tensor, probability float32) (*Tensor, error)` (autodiff_practice.go:12)
- [ ] `func (t *Tape) Embedding(table, indices *Tensor) (*Tensor, error)` (autodiff_catalog.go:7)
- [ ] `func (t *Tape) EmbeddingLookup(indices, table *Tensor) (*Tensor, error)` (autodiff_catalog.go:42)
- [ ] `func (t *Tape) Erf(input *Tensor) (*Tensor, error)` (autodiff.go:241)
- [ ] `func (t *Tape) Flatten(input *Tensor, axes ...int) (*Tensor, error)` (autodiff.go:309)
- [ ] `func (t *Tape) Gelu(input *Tensor, approximate ...string) (*Tensor, error)` (autodiff.go:219)
- [ ] `func (t *Tape) Gemm(a, b, c *Tensor, options ...GemmOptions) (*Tensor, error)` (autodiff.go:447)
- [ ] `func (t *Tape) GlobalAveragePool(input *Tensor) (*Tensor, error)` (autodiff_cnn.go:67)
- [ ] `func (t *Tape) Grad(param *Tensor) (*Tensor, error)` (autodiff.go:570)
- [ ] `func (t *Tape) LayerNormalization(input, scale, bias *Tensor, axis int, epsilon float32) (*Tensor, error)` (autodiff.go:199)
- [ ] `func (t *Tape) MSELoss(prediction, target *Tensor) (*Tensor, error)` (autodiff.go:486)
- [ ] `func (t *Tape) MatMul(a, b *Tensor) (*Tensor, error)` (autodiff.go:85)
- [ ] `func (t *Tape) MaxPool(input *Tensor, kernelShape []int, options ...PoolOptions) (*Tensor, error)` (autodiff_cnn.go:32)
- [ ] `func (t *Tape) Mul(a, b *Tensor) (*Tensor, error)` (autodiff.go:100)
- [ ] `func (t *Tape) Param(value *Tensor) (*Parameter, error)` (autodiff.go:70)
- [ ] `func (t *Tape) Pow(left, right *Tensor) (*Tensor, error)` (autodiff.go:265)
- [ ] `func (t *Tape) ReduceMean(input *Tensor, axes []int, keepdims bool) (*Tensor, error)` (autodiff.go:277)
- [ ] `func (t *Tape) Relu(input *Tensor) (*Tensor, error)` (autodiff.go:136)
- [ ] `func (t *Tape) Reshape(input *Tensor, shape []int) (*Tensor, error)` (autodiff.go:296)
- [ ] `func (t *Tape) SGD(rate float32) error` (autodiff.go:581)
- [ ] `func (t *Tape) SGDMomentum(rate, momentum float32) error` (autodiff.go:604)
- [ ] `func (t *Tape) Sigmoid(input *Tensor) (*Tensor, error)` (autodiff.go:148)
- [ ] `func (t *Tape) Slice(input *Tensor, starts, ends, axes, steps []int64) (*Tensor, error)` (autodiff.go:375)
- [ ] `func (t *Tape) Softmax(input *Tensor, axes ...int) (*Tensor, error)` (autodiff.go:172)
- [ ] `func (t *Tape) SoftmaxCrossEntropy(logits, labels *Tensor) (*Tensor, error)` (autodiff.go:469)
- [ ] `func (t *Tape) Split(input *Tensor, splits []int, axis int, outputCount ...int) ([]*Tensor, error)` (autodiff.go:413)
- [ ] `func (t *Tape) Sqrt(input *Tensor) (*Tensor, error)` (autodiff.go:253)
- [ ] `func (t *Tape) Squeeze(input *Tensor, axes []int) (*Tensor, error)` (autodiff.go:359)
- [ ] `func (t *Tape) Tanh(input *Tensor) (*Tensor, error)` (autodiff.go:160)
- [ ] `func (t *Tape) Transpose(input *Tensor, perms ...[]int) (*Tensor, error)` (autodiff.go:322)
- [ ] `func (t *Tape) Unsqueeze(input *Tensor, axes []int) (*Tensor, error)` (autodiff.go:343)
- [ ] `func (t *Tensor) BoolData() ([]bool, error)` (tensor.go:162)
- [ ] `func (t *Tensor) DType() DType` (tensor.go:95)
- [ ] `func (t *Tensor) Data() []float32` (tensor.go:121)
- [ ] `func (t *Tensor) Float32Data() ([]float32, error)` (tensor.go:129)
- [ ] `func (t *Tensor) Int64Data() ([]int64, error)` (tensor.go:140)
- [ ] `func (t *Tensor) Len() int` (tensor.go:173)
- [ ] `func (t *Tensor) Shape() []int` (tensor.go:103)
- [ ] `func (t *Tensor) Strides() []int` (tensor.go:112)
- [ ] `func (t *Tensor) StringData() ([]string, error)` (tensor.go:151)
- [ ] `func Add(a, b *Tensor) (*Tensor, error)` (kernels.go:1121)
- [ ] `func AveragePool(input *Tensor, kernelShape []int, options ...PoolOptions) (*Tensor, error)` (kernels.go:357)
- [ ] `func AvgPool2D(kernel int, options ...PoolOptions) Layer` (layers_catalog.go:150)
- [ ] `func BatchNorm2D(features int) Layer` (layers_catalog.go:223)
- [ ] `func BatchNormalization(input, scale, bias, mean, variance *Tensor, epsilonValues ...float32) (*Tensor, error)` (kernels.go:612)
- [ ] `func BindClassifier(model *Model, inputName string, features []string, classes *insyra.DataList) (*BoundClassifier, error)` (protocol.go:113)
- [ ] `func BindRegressor(model *Model, inputName string, features []string) (*BoundRegressor, error)` (protocol.go:27)
- [ ] `func Cast(input *Tensor, to DType) (*Tensor, error)` (kernels.go:1899)
- [ ] `func Ceil(input *Tensor) (*Tensor, error)` (kernels.go:1182)
- [ ] `func Clip(input, minimum, maximum *Tensor) (*Tensor, error)` (kernels.go:1239)
- [ ] `func Concat(inputs []*Tensor, axis int) (*Tensor, error)` (control_kernels.go:88)
- [ ] `func Constant(value *Tensor) (*Tensor, error)` (kernels.go:2014)
- [ ] `func ConstantOfShape(shapeTensor, value *Tensor) (*Tensor, error)` (control_kernels.go:314)
- [ ] `func Conv(input, weights, bias *Tensor, options ...ConvOptions) (*Tensor, error)` (kernels.go:183)
- [ ] `func Conv2D(in, out, kernel int, options ...ConvOptions) Layer` (layers_catalog.go:18)
- [ ] `func Dense(in, out int) Layer` (layers.go:50)
- [ ] `func Div(a, b *Tensor) (*Tensor, error)` (control_kernels.go:9)
- [ ] `func Dropout(p float32) Layer` (layers.go:171)
- [ ] `func Embedding(vocab, dims int) Layer` (layers_catalog.go:400)
- [ ] `func Equal(left, right *Tensor) (*Tensor, error)` (control_kernels.go:597)
- [ ] `func Erf(input *Tensor) (*Tensor, error)` (kernels.go:1209)
- [ ] `func Exp(input *Tensor) (*Tensor, error)` (kernels.go:1177)
- [ ] `func Expand(input *Tensor, targetShape []int) (*Tensor, error)` (control_kernels.go:257)
- [ ] `func Flatten(input *Tensor, axes ...int) (*Tensor, error)` (kernels.go:1816)
- [ ] `func Floor(input *Tensor) (*Tensor, error)` (kernels.go:1155)
- [ ] `func Func(fn func(*Tape, *Tensor) (*Tensor, error)) Layer` (layers.go:214)
- [ ] `func Gather(data, indices *Tensor, axis int) (*Tensor, error)` (control_kernels.go:159)
- [ ] `func Gelu(input *Tensor, approximate ...string) (*Tensor, error)` (kernels.go:1285)
- [ ] `func Gemm(a, b, c *Tensor, options ...GemmOptions) (*Tensor, error)` (kernels.go:79)
- [ ] `func GlobalAveragePool(input *Tensor) (*Tensor, error)` (kernels.go:422)
- [ ] `func GlobalAvgPool() Layer` (layers_catalog.go:201)
- [ ] `func Greater(left, right *Tensor) (*Tensor, error)` (control_kernels.go:602)
- [ ] `func GreaterOrEqual(left, right *Tensor) (*Tensor, error)` (control_kernels.go:567)
- [ ] `func Identity(input *Tensor) (*Tensor, error)` (kernels.go:1750)
- [ ] `func InstanceNormalization(input, scale, bias *Tensor, epsilonValues ...float32) (*Tensor, error)` (kernels.go:656)
- [ ] `func LayerNorm(dims interface{}) Layer` (layers_catalog.go:319)
- [ ] `func LayerNormalization(input, scale, bias *Tensor, axis int, epsilon float32) (*Tensor, error)` (kernels.go:1311)
- [ ] `func LeakyRelu(input *Tensor, alpha ...float32) (*Tensor, error)` (kernels.go:1160)
- [ ] `func LoadONNX(r io.Reader) (model *Model, err error)` (model.go:66)
- [ ] `func LoadSafeTensors(r io.Reader) (tensors map[string]*Tensor, err error)` (safetensors.go:21)
- [ ] `func MatMul(a, b *Tensor) (*Tensor, error)` (kernels.go:956)
- [ ] `func MaxPool(input *Tensor, kernelShape []int, options ...PoolOptions) (*Tensor, error)` (kernels.go:298)
- [ ] `func MaxPool2D(kernel int, options ...PoolOptions) Layer` (layers_catalog.go:140)
- [ ] `func Mul(a, b *Tensor) (*Tensor, error)` (kernels.go:1137)
- [ ] `func MultiHeadAttention(embed, heads int) Layer` (layers_attention.go:24)
- [ ] `func NewAvgPool2D(kernel int, options ...PoolOptions) Layer` (layers_catalog.go:155)
- [ ] `func NewBatchNorm2D(features int) Layer` (layers_catalog.go:228)
- [ ] `func NewBoolTensor(shape []int, data []bool) (*Tensor, error)` (tensor.go:80)
- [ ] `func NewConv2D(in, out, kernel int, options ...ConvOptions) Layer` (layers_catalog.go:30)
- [ ] `func NewCosineAnnealingLR(initialRate float32, tMax int) (*CosineAnnealingLR, error)` (autodiff_practice.go:99)
- [ ] `func NewDense(in, out int) Layer` (layers.go:55)
- [ ] `func NewDropout(p float32) Layer` (layers.go:174)
- [ ] `func NewEmbedding(vocab, dims int) Layer` (layers_catalog.go:403)
- [ ] `func NewFlatten() Layer` (layers.go:196)
- [ ] `func NewFloat32Tensor(shape []int, data []float32) (*Tensor, error)` (tensor.go:65)
- [ ] `func NewFunc(fn func(*Tape, *Tensor) (*Tensor, error)) Layer` (layers.go:217)
- [ ] `func NewGelu() Layer` (layers.go:162)
- [ ] `func NewGlobalAvgPool() Layer` (layers_catalog.go:204)
- [ ] `func NewInt64Tensor(shape []int, data []int64) (*Tensor, error)` (tensor.go:70)
- [ ] `func NewLayerNorm(dims interface{}) Layer` (layers_catalog.go:333)
- [ ] `func NewMaxPool2D(kernel int, options ...PoolOptions) Layer` (layers_catalog.go:145)
- [ ] `func NewMultiHeadAttention(embed, heads int) Layer` (layers_attention.go:30)
- [ ] `func NewReLU() Layer` (layers.go:149)
- [ ] `func NewSequential(t *Tape, layers ...Layer) (*Sequential, error)` (sequential.go:20)
- [ ] `func NewSigmoid() Layer` (layers.go:152)
- [ ] `func NewStepLR(initialRate, gamma float32, stepSize int) (*StepLR, error)` (autodiff_practice.go:65)
- [ ] `func NewStringTensor(shape []int, data []string) (*Tensor, error)` (tensor.go:75)
- [ ] `func NewTanh() Layer` (layers.go:157)
- [ ] `func NewTape(seed ...int64) *Tape` (autodiff.go:56)
- [ ] `func NewTensor(shape []int, data []float32) (*Tensor, error)` (tensor.go:59)
- [ ] `func NewTensorWithDType(dtype DType, shape []int, data []float32) (*Tensor, error)` (tensor.go:87)
- [ ] `func NonMaxSuppression(boxes, scores, maxOutputBoxesPerClass, iouThreshold, scoreThreshold *Tensor, centerPointBox ...int) (*Tensor, error)` (kernels.go:1547)
- [ ] `func Pad(input *Tensor, pads []int, values ...float32) (*Tensor, error)` (kernels.go:718)
- [ ] `func PadReflect(input *Tensor, pads []int) (*Tensor, error)` (kernels.go:772)
- [ ] `func Pow(left, right *Tensor) (*Tensor, error)` (kernels.go:1220)
- [ ] `func ReLU() Layer` (layers.go:144)
- [ ] `func ReduceMean(input *Tensor, axes []int, keepdims bool) (*Tensor, error)` (kernels.go:1368)
- [ ] `func ReduceMin(input *Tensor, axes []int, keepdims bool) (*Tensor, error)` (kernels.go:1435)
- [ ] `func RegisterDeviceMatMul(fn DeviceMatMul)` (device_matmul.go:22)
- [ ] `func Relu(input *Tensor) (*Tensor, error)` (kernels.go:1145)
- [ ] `func Reshape(input *Tensor, shape []int) (*Tensor, error)` (kernels.go:1756)
- [ ] `func Residual(layers ...Layer) Layer` (layers_attention.go:246)
- [ ] `func Resize(input, scales, sizes *Tensor, options ...ResizeOptions) (*Tensor, error)` (kernels.go:452)
- [ ] `func Round(input *Tensor) (*Tensor, error)` (kernels.go:1188)
- [ ] `func SaveSafeTensors(w io.Writer, tensors map[string]*Tensor) error` (safetensors.go:133)
- [ ] `func Shape(input *Tensor, bounds ...int) (*Tensor, error)` (control_kernels.go:286)
- [ ] `func Sigmoid(input *Tensor) (*Tensor, error)` (kernels.go:1193)
- [ ] `func Slice(input *Tensor, starts, ends, axes, steps []int64) (*Tensor, error)` (control_kernels.go:362)
- [ ] `func Softmax(input *Tensor, axes ...int) (*Tensor, error)` (kernels.go:1699)
- [ ] `func Split(input *Tensor, splits []int, axis int, outputCount ...int) ([]*Tensor, error)` (control_kernels.go:500)
- [ ] `func Sqrt(input *Tensor) (*Tensor, error)` (kernels.go:1214)
- [ ] `func Squeeze(input *Tensor, axes []int) (*Tensor, error)` (control_kernels.go:212)
- [ ] `func Sub(a, b *Tensor) (*Tensor, error)` (kernels.go:1129)
- [ ] `func Tanh(input *Tensor) (*Tensor, error)` (kernels.go:1204)
- [ ] `func Tile(input, repeats *Tensor) (*Tensor, error)` (kernels.go:1499)
- [ ] `func Transpose(input *Tensor, perms ...[]int) (*Tensor, error)` (kernels.go:1845)
- [ ] `func Unsqueeze(input *Tensor, axes []int) (*Tensor, error)` (control_kernels.go:22)
- [ ] `func Where(condition, left, right *Tensor) (*Tensor, error)` (control_kernels.go:687)
- [ ] `type Adam struct { Rate float32 }` (fit.go:60)
- [ ] `type AdamW struct { Rate float32 WeightDecay float32 }` (fit.go:76)
- [ ] `type AveragePoolOptions = PoolOptions` (kernels.go:171)
- [ ] `type BCEWithLogits struct{}` (fit.go:141)
- [ ] `type BCEWithLogitsLoss = BCEWithLogits` (fit.go:162)
- [ ] `type BoundClassifier struct { model *Model inputName string features []string inputSpec ValueInfo probabilities ValueInfo classes *insyra.DataList }` (protocol.go:98)
- [ ] `type BoundRegressor struct { model *Model inputName string features []string inputSpec ValueInfo output ValueInfo }` (protocol.go:12)
- [ ] `type Classifier = BoundClassifier` (protocol.go:108)
- [ ] `type ConvOptions struct { Pads []int AutoPad string Strides []int Dilations []int Group int NoBias bool }` (kernels.go:144)
- [ ] `type CosineAnnealingLR struct { initialRate float32 tMax int }` (autodiff_practice.go:92)
- [ ] `type CrossEntropy struct{}` (fit.go:103)
- [ ] `type DType string` (tensor.go:12)
- [ ] `type DataType = DType` (tensor.go:42)
- [ ] `type DeviceMatMul func(a []float32, aRows, aCols int, b []float32, bRows, bCols int) ([]float32, error)` (device_matmul.go:8)
- [ ] `type EvalLayer interface { PredictForward(x *Tensor) (*Tensor, error) }` (layers.go:25)
- [ ] `type FitConfig struct { Epochs int BatchSize int Seed int64 NoShuffle bool Optimizer OptimizerSpec Loss LossSpec ValX *Tensor ValY *Tensor Progress func(FitEpoch) Quiet bool }` (fit.go:165)
- [ ] `type FitEpoch struct { Epoch int Epochs int TrainLoss float64 ValLoss float64 HasValLoss bool Elapsed time.Duration RowsPerSecond float64 }` (fit.go:180)
- [ ] `type FitResult struct { TrainLosses []float64 ValLosses []float64 Epochs []FitEpoch Elapsed time.Duration }` (fit.go:191)
- [ ] `type GemmOptions struct { Alpha float32 Beta float32 TransA bool TransB bool }` (kernels.go:70)
- [ ] `type Layer interface { Build(t *Tape) error Forward(t *Tape, x *Tensor) (*Tensor, error) Parameters() []*Parameter }` (layers.go:10)
- [ ] `type LossSpec interface { fitLossName() string fitLossValidate(*Tensor, *Tensor) error fitLoss(*Tape, *Tensor, *Tensor) (*Tensor, error) }` (fit.go:96)
- [ ] `type MSE struct{}` (fit.go:122)
- [ ] `type MSELoss = MSE` (fit.go:161)
- [ ] `type MaxPoolOptions = PoolOptions` (kernels.go:170)
- [ ] `type Model struct { inputSpecs []ValueInfo outputSpecs []ValueInfo nodes []modelNode initializers map[string]*Tensor opsetVersion int64 }` (model.go:36)
- [ ] `type OptimizerSpec interface { fitOptimizerName() string fitOptimizerValidate() error fitOptimizerStep(*Tape) error }` (fit.go:15)
- [ ] `type Parameter struct { value *Tensor grad *Tensor velocity []float32 adamM []float32 adamV []float32 adamStep uint64 }` (autodiff.go:21)
- [ ] `type PoolOptions struct { Pads []int AutoPad string Strides []int CountIncludePad bool CeilMode int StorageOrder int }` (kernels.go:159)
- [ ] `type Regressor = BoundRegressor` (protocol.go:21)
- [ ] `type ResizeOptions struct { Mode string CoordinateTransformationMode string NearestMode string }` (kernels.go:175)
- [ ] `type SGD struct { Rate float32 }` (fit.go:22)
- [ ] `type SGDMomentum struct { Rate float32 Momentum float32 }` (fit.go:38)
- [ ] `type Sequential struct { layers []Layer tape *Tape }` (sequential.go:12)
- [ ] `type SoftmaxCrossEntropy = CrossEntropy` (fit.go:160)
- [ ] `type StepLR struct { initialRate float32 gamma float32 stepSize int }` (autodiff_practice.go:58)
- [ ] `type Tape struct { ops []tapeOp params []*Parameter marked map[*Tensor]*Parameter grads map[*Tensor]*Tensor rng *rand.Rand }` (autodiff.go:12)
- [ ] `type Tensor struct { dtype DType shape []int strides []int data []float32 int64Data []int64 boolData []bool stringData []string }` (tensor.go:47)
- [ ] `type TrainingOnly interface { TrainingOnly() }` (layers.go:18)
- [ ] `type ValueInfo struct { Name string DType DType Shape []int HasShape bool }` (model.go:11)
- [ ] `func (*globalAvgPoolLayer) Build(*Tape) error` (layers_catalog.go:206)
- [ ] `func (*globalAvgPoolLayer) Parameters() []*Parameter` (layers_catalog.go:210)
- [ ] `func (l *activationLayer) Build(*Tape) error` (layers.go:130)
- [ ] `func (l *activationLayer) Forward(t *Tape, x *Tensor) (*Tensor, error)` (layers.go:131)
- [ ] `func (l *activationLayer) Parameters() []*Parameter` (layers.go:140)
- [ ] `func (l *batchNorm2DLayer) Build(t *Tape) error` (layers_catalog.go:230)
- [ ] `func (l *batchNorm2DLayer) Forward(t *Tape, x *Tensor) (*Tensor, error)` (layers_catalog.go:273)
- [ ] `func (l *batchNorm2DLayer) Parameters() []*Parameter` (layers_catalog.go:290)
- [ ] `func (l *batchNorm2DLayer) PredictForward(x *Tensor) (*Tensor, error)` (layers_catalog.go:283)
- [ ] `func (l *conv2DLayer) Build(t *Tape) error` (layers_catalog.go:34)
- [ ] `func (l *conv2DLayer) Forward(t *Tape, x *Tensor) (*Tensor, error)` (layers_catalog.go:75)
- [ ] `func (l *conv2DLayer) Parameters() []*Parameter` (layers_catalog.go:91)
- [ ] `func (l *denseLayer) Build(t *Tape) error` (layers.go:57)
- [ ] `func (l *denseLayer) Forward(t *Tape, x *Tensor) (*Tensor, error)` (layers.go:91)
- [ ] `func (l *denseLayer) Parameters() []*Parameter` (layers.go:111)
- [ ] `func (l *dropoutLayer) Build(*Tape) error` (layers.go:176)
- [ ] `func (l *dropoutLayer) Forward(t *Tape, x *Tensor) (*Tensor, error)` (layers.go:182)
- [ ] `func (l *dropoutLayer) Parameters() []*Parameter` (layers.go:188)
- [ ] `func (l *dropoutLayer) TrainingOnly()` (layers.go:189)
- [ ] `func (l *embeddingLayer) Build(t *Tape) error` (layers_catalog.go:405)
- [ ] `func (l *embeddingLayer) Forward(t *Tape, x *Tensor) (*Tensor, error)` (layers_catalog.go:430)
- [ ] `func (l *embeddingLayer) Parameters() []*Parameter` (layers_catalog.go:437)
- [ ] `func (l *flattenLayer) Build(*Tape) error` (layers.go:198)
- [ ] `func (l *flattenLayer) Forward(t *Tape, x *Tensor) (*Tensor, error)` (layers.go:199)
- [ ] `func (l *flattenLayer) Parameters() []*Parameter` (layers.go:205)
- [ ] `func (l *funcLayer) Build(*Tape) error` (layers.go:219)
- [ ] `func (l *funcLayer) Forward(t *Tape, x *Tensor) (*Tensor, error)` (layers.go:225)
- [ ] `func (l *funcLayer) Parameters() []*Parameter` (layers.go:241)
- [ ] `func (l *globalAvgPoolLayer) Forward(t *Tape, x *Tensor) (*Tensor, error)` (layers_catalog.go:207)
- [ ] `func (l *layerNormLayer) Build(t *Tape) error` (layers_catalog.go:335)
- [ ] `func (l *layerNormLayer) Forward(t *Tape, x *Tensor) (*Tensor, error)` (layers_catalog.go:369)
- [ ] `func (l *layerNormLayer) Parameters() []*Parameter` (layers_catalog.go:379)
- [ ] `func (l *multiHeadAttentionLayer) Build(t *Tape) error` (layers_attention.go:34)
- [ ] `func (l *multiHeadAttentionLayer) Forward(t *Tape, x *Tensor) (*Tensor, error)` (layers_attention.go:86)
- [ ] `func (l *multiHeadAttentionLayer) Parameters() []*Parameter` (layers_attention.go:187)
- [ ] `func (l *pool2DLayer) Build(*Tape) error` (layers_catalog.go:170)
- [ ] `func (l *pool2DLayer) Forward(t *Tape, x *Tensor) (*Tensor, error)` (layers_catalog.go:180)
- [ ] `func (l *pool2DLayer) Parameters() []*Parameter` (layers_catalog.go:190)
- [ ] `func (l *residualLayer) Build(t *Tape) error` (layers_attention.go:250)
- [ ] `func (l *residualLayer) Forward(t *Tape, x *Tensor) (*Tensor, error)` (layers_attention.go:265)
- [ ] `func (l *residualLayer) Parameters() []*Parameter` (layers_attention.go:313)
- [ ] `func (l *residualLayer) PredictForward(x *Tensor) (*Tensor, error)` (layers_attention.go:273)

## parallel (5)

- [x] `func (pg *ParallelGroup) AwaitNoResult()` (parallel_computing.go:68) — P-3 doc 宣稱省掉結果收集，但 Run 一律收集
- [x] `func (pg *ParallelGroup) AwaitResult() [][]any` (parallel_computing.go:61) — P-1 回傳 [][]any；未呼叫 Run 直接 Await 立即回傳全 nil，無警示
- [x] `func (pg *ParallelGroup) Run() *ParallelGroup` (parallel_computing.go:24) — P-1 panic 轉成 error 塞進結果槽，與函式自己回傳的 error 無法區分；P-2 呼叫兩次會重跑；無 context、無併發上限
- [x] `func GroupUp(fns ...any) *ParallelGroup` (parallel_computing.go:16) — P-1 參數型別 any，錯誤只能在執行期發現、P-5 三步串接
- [x] `type ParallelGroup struct { fns []any results [][]any wg sync.WaitGroup }` (parallel_computing.go:9) — 欄位全私有，零值可用但無意義；建議整體見 P-4

## parquet (27)

- [x] `func ApplyCCL(ctx context.Context, path string, cclScript string) error` (ccl.go:576) — Q-4 stdlib log；Q-5 doc 範例引用不存在的 CCLFilterOptions；Q-6 batchSize 寫死 1000；tmp+rename 原子替換：OK；空輸入不覆蓋原檔：OK
- [x] `func FilterWithCCL(ctx context.Context, path string, filterExpr string) (*insyra.DataTable, error)` (ccl.go:455) — Q-6 batchSize 寫死；回傳新表不動原檔：OK
- [x] `func Inspect(path string) (FileInfo, error)` (api.go:54) — OK。無 ctx 可接受（純 metadata）；Q-4、Q-8、Q-10
- [x] `func Read(ctx context.Context, path string, opt ReadOptions) (*insyra.DataTable, error)` (api.go:168) — OK 主流程；RowGroups 越界未先驗證（依賴 arrow 回錯，未實測）；Q-4、Q-8 只吃路徑、Q-10 doc 風格
- [x] `func ReadColumn(ctx context.Context, path string, column string, opt ReadColumnOptions) (*insyra.DataList, error)` (api.go:288) — Q-1 `MaxValues` 完全沒讀，doc 承諾的保護不存在；實作是整表 Read 再取 A 欄
- [x] `func Stream(ctx context.Context, path string, opt ReadOptions, batchSize int) (<-chan *insyra.DataTable, <-chan error)` (api.go:246) — Q-7 消費者若不 cancel ctx 就停止讀取，goroutine 永久阻塞；doc 未說明必須 drain / cancel；記錄無遺失（unbuffered，已推演）、Q-9 建議 iter.Seq2
- [x] `func Write(dt insyra.IDataTable, path string) error` (api.go:131) — Q-2 無 ctx、無選項（壓縮、chunk 1Mi 寫死）；Q-3 直接寫目標路徑，失敗留下半個檔案，與 ApplyCCL 的 tmp+rename 不一致；Q-4、Q-8、Q-10
- [x] `type ColumnInfo struct { Name string PhysicalType string LogicalType string Repetition string }` (api.go:40) — OK，缺 doc（Q-10）
- [x] `type FileInfo struct { NumRows int64 NumRowGroups int Version string CreatedBy string Metadata map[string]string Columns []ColumnInfo RowGroups []RowGroupInfo }` (api.go:30) — OK，缺 doc（Q-10）
- [x] `type ReadColumnOptions struct { RowGroups []int MaxValues int64 }` (api.go:25) — Q-1 MaxValues 是死欄位
- [x] `type ReadOptions struct { Columns []string RowGroups []int }` (api.go:19) — OK
- [x] `type RowGroupInfo struct { NumRows int64 TotalByteSize int64 TotalCompressedSize int64 }` (api.go:47) — OK，缺 doc（Q-10）
- [ ] `func (c *parquetContext) GetAllData() ([]any, error)` (ccl.go:210)
- [ ] `func (c *parquetContext) GetCell(colIndex, rowIndex int) (any, error)` (ccl.go:99)
- [ ] `func (c *parquetContext) GetCellByName(colName string, rowIndex int) (any, error)` (ccl.go:117)
- [ ] `func (c *parquetContext) GetCol(index int) any` (ccl.go:73)
- [ ] `func (c *parquetContext) GetColByName(name string) (any, error)` (ccl.go:80)
- [ ] `func (c *parquetContext) GetColCount() int` (ccl.go:156)
- [ ] `func (c *parquetContext) GetColData(index int) ([]any, error)` (ccl.go:182)
- [ ] `func (c *parquetContext) GetColDataByName(name string) ([]any, error)` (ccl.go:202)
- [ ] `func (c *parquetContext) GetColIndexByName(colName string) (int, error)` (ccl.go:148)
- [ ] `func (c *parquetContext) GetCurrentRow() any` (ccl.go:95)
- [ ] `func (c *parquetContext) GetRowAt(rowIndex int) (any, error)` (ccl.go:125)
- [ ] `func (c *parquetContext) GetRowCount() int` (ccl.go:163)
- [ ] `func (c *parquetContext) GetRowIndex() int` (ccl.go:91)
- [ ] `func (c *parquetContext) GetRowIndexByName(rowName string) (int, error)` (ccl.go:144)
- [ ] `func (c *parquetContext) SetRowIndex(index int) error` (ccl.go:170)

## pd (8)

- [ ] `func (s *Series) ToDataList() (*insyra.DataList, error)` (series.go:131)
- [ ] `func (t *DataFrame) ToDataTable() (*insyra.DataTable, error)` (dataframe.go:121)
- [ ] `func FromDataList(dl insyra.IDataList) (*Series, error)` (series.go:14)
- [ ] `func FromDataTable(dt insyra.IDataTable) (*DataFrame, error)` (dataframe.go:19)
- [ ] `func FromGPandasDataFrame(df *gpdf.DataFrame) (*DataFrame, error)` (dataframe.go:255)
- [ ] `func FromGPandasSeries(gpds gpdc.Series) (*Series, error)` (series.go:121)
- [ ] `type DataFrame struct { *gpdf.DataFrame }` (dataframe.go:13)
- [ ] `type Series struct { gpdc.Series }` (series.go:10)

## plot (80)

- [ ] `const LabelPositionBottom LabelPosition` (positions.go:16)
- [ ] `const LabelPositionInside LabelPosition` (positions.go:19)
- [ ] `const LabelPositionInsideBottom LabelPosition` (positions.go:23)
- [ ] `const LabelPositionInsideBottomLeft LabelPosition` (positions.go:25)
- [ ] `const LabelPositionInsideBottomRight LabelPosition` (positions.go:27)
- [ ] `const LabelPositionInsideLeft LabelPosition` (positions.go:20)
- [ ] `const LabelPositionInsideRight LabelPosition` (positions.go:21)
- [ ] `const LabelPositionInsideTop LabelPosition` (positions.go:22)
- [ ] `const LabelPositionInsideTopLeft LabelPosition` (positions.go:24)
- [ ] `const LabelPositionInsideTopRight LabelPosition` (positions.go:26)
- [ ] `const LabelPositionLeft LabelPosition` (positions.go:17)
- [ ] `const LabelPositionRight LabelPosition` (positions.go:18)
- [ ] `const LabelPositionTop LabelPosition` (positions.go:15)
- [ ] `const PositionBottom Position` (positions.go:7)
- [ ] `const PositionLeft Position` (positions.go:8)
- [ ] `const PositionRight Position` (positions.go:9)
- [ ] `const PositionTop Position` (positions.go:6)
- [ ] `const ThemeChalk Theme` (themes.go:8)
- [ ] `const ThemeEssos Theme` (themes.go:9)
- [ ] `const ThemeInfographic Theme` (themes.go:10)
- [ ] `const ThemeMacarons Theme` (themes.go:11)
- [ ] `const ThemePurplePassion Theme` (themes.go:12)
- [ ] `const ThemeRiverAxisTypeTime ThemeRiverAxisType` (themeriver.go:19)
- [ ] `const ThemeRoma Theme` (themes.go:13)
- [ ] `const ThemeRomantic Theme` (themes.go:14)
- [ ] `const ThemeShine Theme` (themes.go:15)
- [ ] `const ThemeVintage Theme` (themes.go:16)
- [ ] `const ThemeWalden Theme` (themes.go:17)
- [ ] `const ThemeWesteros Theme` (themes.go:18)
- [ ] `const ThemeWonderland Theme` (themes.go:19)
- [ ] `const WordCloudShapeArrow WordCloudShape` (wordcloud.go:20)
- [ ] `const WordCloudShapeCircle WordCloudShape` (wordcloud.go:14)
- [ ] `const WordCloudShapeDiamond WordCloudShape` (wordcloud.go:18)
- [ ] `const WordCloudShapePin WordCloudShape` (wordcloud.go:19)
- [ ] `const WordCloudShapeRect WordCloudShape` (wordcloud.go:15)
- [ ] `const WordCloudShapeRoundRect WordCloudShape` (wordcloud.go:16)
- [ ] `const WordCloudShapeTriangle WordCloudShape` (wordcloud.go:17)
- [ ] `func CreateBarChart(config BarChartConfig, data ...insyra.IDataList) *charts.Bar` (bar.go:40)
- [ ] `func CreateBoxPlot(config BoxPlotConfig, series ...BoxPlotSeries) *charts.BoxPlot` (boxplot.go:45)
- [ ] `func CreateFunnelChart(config FunnelChartConfig, data map[string]float64) *charts.Funnel` (funnel.go:30)
- [ ] `func CreateGaugeChart(config GaugeChartConfig, value float64) *charts.Gauge` (gauge.go:27)
- [ ] `func CreateHeatMap[X heapMapAxisValue, Y heapMapAxisValue](config HeatMapConfig, points ...heatMapPoint[X, Y]) *charts.HeatMap` (heatmap.go:71)
- [ ] `func CreateKlineChart(config KlineChartConfig, klinePoints ...KlinePoint) *charts.Kline` (kline.go:42)
- [ ] `func CreateLineChart(config LineChartConfig, data ...insyra.IDataList) *charts.Line` (line.go:45)
- [ ] `func CreatePieChart(config PieChartConfig, data ...PieItem) *charts.Pie` (pie.go:39)
- [ ] `func CreateRadarChart(config RadarChartConfig, series []RadarSeries) *charts.Radar` (radar.go:38)
- [ ] `func CreateSankeyChart(config SankeyChartConfig, links ...SankeyLink) *charts.Sankey` (sankey.go:36)
- [ ] `func CreateScatterChart(config ScatterChartConfig, data map[string][]ScatterPoint) *charts.Scatter` (scatter.go:51)
- [ ] `func CreateThemeRiverChart(config ThemeRiverChartConfig, data ...ThemeRiverData) *charts.ThemeRiver` (themeriver.go:52)
- [ ] `func CreateWordCloud(config WordCloudConfig, data insyra.IDataList) *charts.WordCloud` (wordcloud.go:38)
- [ ] `func HeatMapMissingPoint[X heapMapAxisValue, Y heapMapAxisValue](x X, y Y) heatMapPoint[X, Y]` (heatmap.go:65)
- [ ] `func HeatMapPoint[X heapMapAxisValue, Y heapMapAxisValue](x X, y Y, value float64) heatMapPoint[X, Y]` (heatmap.go:60)
- [ ] `func SaveHTML(chart Renderable, path string, animation ...bool) error` (save_chart.go:39)
- [ ] `func SavePNG(chart Renderable, pngPath string, useOnlineServiceOnFail ...bool) error` (save_chart.go:62)
- [ ] `type BarChartConfig struct { Width string Height string BackgroundColor string Theme Theme Title string Subtitle string TitlePos Position HideLegend bool LegendPos Position XAxis []string XAxisName string YAxisName string YAxisMin *float64 YAxisMax *float64 YAxisSplitNumber *int YAxisFormatter string Colors []string ShowLabels bool LabelPos LabelPosition }` (bar.go:15)
- [ ] `type BoxPlotConfig struct { Width string Height string BackgroundColor string Theme Theme Title string Subtitle string TitlePos Position HideLegend bool LegendPos Position XAxis []string XAxisName string YAxisName string YAxisMin *float64 YAxisMax *float64 YAxisSplitNumber *int YAxisFormatter string }` (boxplot.go:23)
- [ ] `type BoxPlotSeries struct { Name string Data []insyra.IDataList Color string Fill bool }` (boxplot.go:15)
- [ ] `type FunnelChartConfig struct { Width string Height string BackgroundColor string Theme Theme Title string Subtitle string TitlePos Position HideLegend bool LegendPos Position ShowLabels bool LabelPos LabelPosition }` (funnel.go:13)
- [ ] `type GaugeChartConfig struct { Width string Height string BackgroundColor string Theme Theme Title string Subtitle string TitlePos Position HideLegend bool LegendPos Position SeriesName string }` (gauge.go:12)
- [ ] `type HeatMapConfig struct { Width string Height string BackgroundColor string Theme Theme Title string Subtitle string TitlePos Position XAxis []string YAxis []string Colors []string Min *float64 Max *float64 UseCalendar bool CalendarOpts *opts.Calendar }` (heatmap.go:25)
- [ ] `type KlineChartConfig struct { Width string Height string BackgroundColor string Theme Theme Title string Subtitle string TitlePos Position DateFormat string DataZoom bool }` (kline.go:26)
- [ ] `type KlinePoint struct { Date time.Time `json:"date"` Open float64 `json:"open"` High float64 `json:"high"` Low float64 `json:"low"` Close float64 `json:"close"` }` (kline.go:17)
- [ ] `type LabelPosition string` (positions.go:12)
- [ ] `type LineChartConfig struct { Width string Height string BackgroundColor string Theme Theme Title string Subtitle string TitlePos Position HideLegend bool LegendPos Position XAxis []string XAxisName string YAxisName string YAxis []string YAxisMin *float64 YAxisMax *float64 YAxisSplitNumber *int YAxisFormatter string Colors []string ShowLabels bool LabelPos string Smooth bool FillArea bool }` (line.go:15)
- [ ] `type PieChartConfig struct { Width string Height string BackgroundColor string Theme Theme Title string Subtitle string TitlePos Position HideLegend bool LegendPos Position Colors []string ShowLabels bool ShowPercent bool LabelPos LabelPosition RoseType string Radius []string Center []string }` (pie.go:18)
- [ ] `type PieItem struct { Name string Value float64 }` (pie.go:12)
- [ ] `type Position string` (positions.go:3)
- [ ] `type RadarChartConfig struct { Width string Height string BackgroundColor string Theme Theme Title string Subtitle string TitlePos Position HideLegend bool LegendPos Position Indicators []string MaxValues map[string]float32 }` (radar.go:15)
- [ ] `type RadarSeries struct { Name string Values []float32 Color string }` (radar.go:31)
- [ ] `type Renderable interface { Render(w io.Writer) error RenderContent() []byte }` (save_chart.go:33)
- [ ] `type SankeyChartConfig struct { Width string Height string BackgroundColor string Theme Theme Title string Subtitle string TitlePos Position Nodes []string Curveness float32 Color string ShowLabels bool }` (sankey.go:20)
- [ ] `type SankeyLink struct { Source string `json:"source"` Target string `json:"target"` Value float32 `json:"value"` }` (sankey.go:13)
- [ ] `type ScatterChartConfig struct { Width string Height string BackgroundColor string Theme Theme Title string Subtitle string TitlePos Position HideLegend bool LegendPos Position XAxisName string XAxisMin *float64 XAxisMax *float64 XAxisSplitNumber *int XAxisFormatter string YAxisName string YAxisMin *float64 YAxisMax *float64 YAxisSplitNumber *int YAxisFormatter string Colors []string ShowLabels bool LabelPos LabelPosition SplitLine bool Symbol []string SymbolSize int }` (scatter.go:20)
- [ ] `type ScatterPoint struct { X float64 Y float64 }` (scatter.go:14)
- [ ] `type Theme string` (themes.go:5)
- [ ] `type ThemeRiverAxisType string` (themeriver.go:12)
- [ ] `type ThemeRiverChartConfig struct { Width string Height string BackgroundColor string Theme Theme Title string Subtitle string TitlePos Position HideLegend bool LegendPos Position AxisType ThemeRiverAxisType AxisData []string AxisMin *float64 AxisMax *float64 }` (themeriver.go:32)
- [ ] `type ThemeRiverData struct { Date string Value float64 Name string }` (themeriver.go:25)
- [ ] `type WordCloudConfig struct { Width string Height string BackgroundColor string Theme Theme Title string Subtitle string TitlePos Position Shape WordCloudShape SizeRange []float32 }` (wordcloud.go:24)
- [ ] `type WordCloudShape string` (wordcloud.go:11)

## py (14)

- [ ] `func PipFreeze() ([]string, error)` (py.go:374)
- [ ] `func PipInstall(dep string) error` (py.go:301)
- [ ] `func PipList() (map[string]string, error)` (py.go:338)
- [ ] `func PipUninstall(dep string) error` (py.go:319)
- [ ] `func ReinstallPyEnv() error` (py.go:22)
- [ ] `func RunCode(out any, code string) error` (py.go:80)
- [ ] `func RunCodeContext(ctx context.Context, out any, code string) error` (py.go:169)
- [ ] `func RunCodeWithTimeout(timeout time.Duration, out any, code string) error` (py.go:217)
- [ ] `func RunCodef(out any, code string, args ...any) error` (py.go:86)
- [ ] `func RunCodefContext(ctx context.Context, out any, code string, args ...any) error` (py.go:176)
- [ ] `func RunFile(out any, filePath string) error` (py.go:53)
- [ ] `func RunFileContext(ctx context.Context, out any, filePath string) error` (py.go:185)
- [ ] `func RunFilef(out any, filePath string, args ...any) error` (py.go:67)
- [ ] `func RunFilefContext(ctx context.Context, out any, filePath string, args ...any) error` (py.go:198)

## quant (50)

- [x] `const MaximumSharpe` (portfolio.go:24) — OK 有 doc
- [x] `const MinimumVariance PortfolioObjective` (portfolio.go:18) — OK 有 doc
- [x] `const OptionCall OptionType` (options.go:15) — OK 有 doc
- [x] `const OptionPut` (options.go:17) — OK 有 doc
- [x] `const TargetReturn` (portfolio.go:21) — OK 有 doc
- [x] `const VaRHistorical VaRMethod` (risk.go:21) — OK 有 doc
- [x] `const VaRParametric` (risk.go:24) — OK 有 doc
- [x] `func (r *FactorModelResult) Exposure(name string) (float64, bool)` (factor.go:56) — OK（QU-4）
- [x] `func (r *PortfolioResult) Weight(name string) (float64, bool)` (portfolio.go:85) — OK（QU-4）
- [x] `func (r *WalkForwardResult) AnnualizedReturn(days int) (float64, error)` (walkforward.go:129) — OK（QU-4）
- [x] `func (r *WalkForwardResult) MaxDrawdown() (float64, error)` (walkforward.go:122) — OK（QU-4）
- [x] `func (r *WalkForwardResult) Sharpe(riskFreeRate, periodsPerYear float64) (float64, error)` (walkforward.go:116) — OK（QU-4）
- [x] `func AnnualizedReturn(equity insyra.IDataList, days int) (float64, error)` (performance.go:119) — OK（QU-4）
- [x] `func Beta(asset, market insyra.IDataList) (float64, error)` (capm.go:73) — OK；QU-2
- [x] `func BlackScholes(in BSInput) (*BSResult, error)` (options.go:54) — OK（QU-4）
- [x] `func BlockBootstrap(returns insyra.IDataList, cfg BootstrapConfig) (*BootstrapResult, error)` (bootstrap.go:71) — OK（QU-4）
- [x] `func CAPM(asset, market insyra.IDataList, riskFreeRate float64) (*CAPMResult, error)` (capm.go:40) — OK；QU-2
- [x] `func CalmarRatio(equity insyra.IDataList, days int) (float64, error)` (risk.go:182) — OK（QU-4）
- [x] `func ConditionalValueAtRisk(returns insyra.IDataList, confidence float64, method VaRMethod) (float64, error)` (risk.go:74) — OK（QU-4）
- [x] `func DeflatedSharpeRatio(observedSR float64, n int, skew, kurt float64, trialSharpes insyra.IDataList) (float64, error)` (overfitting.go:94) — OK（QU-4）
- [x] `func DrawdownSeries(equity insyra.IDataList) (*insyra.DataList, error)` (risk.go:252) — OK（QU-4）
- [x] `func EfficientFrontier(returns insyra.IDataTable, points int, cfg PortfolioConfig) ([]PortfolioResult, error)` (portfolio.go:160) — OK（QU-4）
- [x] `func ExpectedMaxSharpe(sharpeVariance float64, nTrials int) (float64, error)` (overfitting.go:56) — OK（QU-4）
- [x] `func FactorModel(asset insyra.IDataList, factors insyra.IDataTable, riskFreeRate float64) (*FactorModelResult, error)` (factor.go:79) — OK（QU-4）
- [x] `func ImpliedVolatility(price float64, in BSInput) (float64, error)` (options.go:157) — OK（QU-4）
- [x] `func InformationRatio(returns, benchmark insyra.IDataList, periodsPerYear float64) (float64, error)` (risk.go:211) — OK（QU-4）
- [x] `func MaxDrawdown(equity insyra.IDataList) (float64, error)` (performance.go:78) — OK（QU-4）
- [x] `func OptimizePortfolio(returns insyra.IDataTable, cfg PortfolioConfig) (*PortfolioResult, error)` (portfolio.go:118) — OK（QU-4）
- [x] `func OptimizePortfolioMoments(mean []float64, cov [][]float64, names []string, cfg PortfolioConfig) (*PortfolioResult, error)` (portfolio.go:137) — OK（QU-4）
- [x] `func PBO(perf insyra.IDataTable, nSplits int) (float64, error)` (overfitting.go:134) — OK（QU-4）
- [x] `func PercentileBands(paths [][]float64, percentiles []float64) ([][]float64, error)` (bootstrap.go:217) — OK；QU-3 尺度
- [x] `func ProbabilisticSharpeRatio(observedSR, benchmarkSR float64, n int, skew, kurt float64) (float64, error)` (overfitting.go:32) — OK（QU-4）
- [x] `func SharpeRatio(returns insyra.IDataList, riskFreeRate, periodsPerYear float64) (float64, error)` (performance.go:37) — OK（QU-4）
- [x] `func SortinoRatio(returns insyra.IDataList, minimumAcceptableReturn, periodsPerYear float64) (float64, error)` (risk.go:140) — OK（QU-4）
- [x] `func ValueAtRisk(returns insyra.IDataList, confidence float64, method VaRMethod) (float64, error)` (risk.go:35) — OK（QU-4）
- [x] `func WalkForward[P any]( n int, cfg WalkForwardConfig, optimize func(trainStart, trainEnd int) P, evaluate func(p P, testStart, testEnd int) []float64, ) (*WalkForwardResult, error)` (walkforward.go:66) — OK generics 用法合理；QU-3
- [x] `type BSInput struct { Spot float64 Strike float64 Rate float64 DividendYield float64 Volatility float64 TimeToExpiry float64 Type OptionType }` (options.go:23) — OK（QU-4）
- [x] `type BSResult struct { Price float64 Delta float64 Gamma float64 Vega float64 Theta float64 Rho float64 }` (options.go:37) — OK（QU-4）
- [x] `type BootstrapConfig struct { Horizon int BlockSize int Paths int Seed uint64 Stationary bool }` (bootstrap.go:14) — OK（QU-4）
- [x] `type BootstrapResult struct { Returns [][]float64 Equity [][]float64 }` (bootstrap.go:45) — OK（QU-4）
- [x] `type CAPMResult struct { Beta float64 Alpha float64 RSquared float64 BetaStdErr float64 AlphaStdErr float64 N int }` (capm.go:15) — OK（QU-4）
- [x] `type FactorModelResult struct { Alpha float64 AlphaStdErr float64 AlphaTValue float64 AlphaPValue float64 FactorNames []string Exposures []float64 StdErrs []float64 TValues []float64 PValues []float64 RSquared float64 AdjustedRSquared float64 N int Residuals []float64 }` (factor.go:12) — OK（QU-4）
- [x] `type OptionType uint8` (options.go:11) — QU-1 uint8 enum
- [x] `type PortfolioConfig struct { Objective PortfolioObjective TargetReturn float64 RiskFreeRate float64 MinWeight []float64 MaxWeight []float64 Tolerance float64 MaxIterations int }` (portfolio.go:31) — OK（QU-4）
- [x] `type PortfolioObjective uint8` (portfolio.go:12) — QU-1 uint8 enum
- [x] `type PortfolioResult struct { Weights []float64 AssetNames []string ExpectedReturn float64 Variance float64 Volatility float64 SharpeRatio float64 Iterations int Converged bool }` (portfolio.go:60) — OK（QU-4）
- [x] `type VaRMethod uint8` (risk.go:16) — QU-1 uint8 enum
- [x] `type WalkForwardConfig struct { TrainSize int TestSize int Anchored bool }` (walkforward.go:10) — OK（QU-4）
- [x] `type WalkForwardFold struct { TrainStart int TrainEnd int TestStart int TestEnd int OOSReturns []float64 }` (walkforward.go:25) — OK（QU-4）
- [x] `type WalkForwardResult struct { Folds []WalkForwardFold OOSReturns []float64 Equity []float64 }` (walkforward.go:36) — OK（QU-4）

## stats (197)

- [x] `const AggloAverage AgglomerativeMethod` (clustering.go:67) — OK
- [x] `const AggloCentroid AgglomerativeMethod` (clustering.go:72) — OK
- [x] `const AggloComplete AgglomerativeMethod` (clustering.go:65) — OK
- [x] `const AggloMcQuitty AgglomerativeMethod` (clustering.go:70) — OK
- [x] `const AggloMedian AgglomerativeMethod` (clustering.go:71) — OK
- [x] `const AggloSingle AgglomerativeMethod` (clustering.go:66) — OK
- [x] `const AggloWardD AgglomerativeMethod` (clustering.go:68) — OK
- [x] `const AggloWardD2 AgglomerativeMethod` (clustering.go:69) — OK
- [x] `const Binomial GLMFamily` (consts.go:10) — OK
- [x] `const Cloglog GLMLink` (consts.go:20) — OK
- [x] `const FactorCountFixed FactorCountMethod` (factor_analysis.go:63) — OK
- [x] `const FactorCountKaiser FactorCountMethod` (factor_analysis.go:64) — OK
- [x] `const FactorExtractionMINRES FactorExtractionMethod` (factor_analysis.go:27) — OK
- [x] `const FactorExtractionML FactorExtractionMethod` (factor_analysis.go:26) — OK
- [x] `const FactorExtractionPAF FactorExtractionMethod` (factor_analysis.go:25) — OK
- [x] `const FactorExtractionPCA FactorExtractionMethod` (factor_analysis.go:24) — OK
- [x] `const FactorRotationBentlerQ FactorRotationMethod` (factor_analysis.go:44) — OK
- [x] `const FactorRotationBentlerT FactorRotationMethod` (factor_analysis.go:41) — OK
- [x] `const FactorRotationGeominQ FactorRotationMethod` (factor_analysis.go:43) — OK
- [x] `const FactorRotationGeominT FactorRotationMethod` (factor_analysis.go:40) — OK
- [x] `const FactorRotationNone FactorRotationMethod` (factor_analysis.go:35) — OK
- [x] `const FactorRotationOblimin FactorRotationMethod` (factor_analysis.go:39) — OK
- [x] `const FactorRotationPromax FactorRotationMethod` (factor_analysis.go:45) — OK
- [x] `const FactorRotationQuartimax FactorRotationMethod` (factor_analysis.go:37) — OK
- [x] `const FactorRotationQuartimin FactorRotationMethod` (factor_analysis.go:38) — OK
- [x] `const FactorRotationSimplimax FactorRotationMethod` (factor_analysis.go:42) — OK
- [x] `const FactorRotationVarimax FactorRotationMethod` (factor_analysis.go:36) — OK
- [x] `const FactorScoreAndersonRubin FactorScoreMethod` (factor_analysis.go:56) — OK
- [x] `const FactorScoreBartlett FactorScoreMethod` (factor_analysis.go:55) — OK
- [x] `const FactorScoreNone FactorScoreMethod` (factor_analysis.go:53) — OK
- [x] `const FactorScoreRegression FactorScoreMethod` (factor_analysis.go:54) — OK
- [x] `const Gaussian GLMFamily` (consts.go:12) — OK
- [x] `const Greater AlternativeHypothesis` (types.go:7) — OK
- [x] `const Identity GLMLink` (consts.go:18) — OK
- [x] `const KNNAuto KNNAlgorithm` (knn.go:18) — OK
- [x] `const KNNBallTree KNNAlgorithm` (knn.go:21) — OK
- [x] `const KNNBruteForce KNNAlgorithm` (knn.go:19) — OK
- [x] `const KNNDistanceWeighting KNNWeighting` (knn.go:16) — OK
- [x] `const KNNKDTree KNNAlgorithm` (knn.go:20) — OK
- [x] `const KNNUniformWeighting KNNWeighting` (knn.go:15) — OK
- [x] `const KendallCorrelation` (correlation.go:23) — OK
- [x] `const KurtosisAdjusted` (kurtosis.go:15) — OK
- [x] `const KurtosisBiasAdjusted` (kurtosis.go:16) — OK
- [x] `const KurtosisG2 KurtosisMethod` (kurtosis.go:14) — OK
- [x] `const Less AlternativeHypothesis` (types.go:8) — OK
- [x] `const Log GLMLink` (consts.go:16) — OK
- [x] `const Logit GLMLink` (consts.go:17) — OK
- [x] `const PearsonCorrelation CorrelationMethod` (correlation.go:22) — OK
- [x] `const Poisson GLMFamily` (consts.go:11) — OK
- [x] `const PredictClass PredictType` (glm_predict.go:15) — OK
- [x] `const PredictLinear PredictType` (glm_predict.go:13) — OK
- [x] `const PredictResponse PredictType` (glm_predict.go:14) — OK
- [x] `const Probit GLMLink` (consts.go:19) — OK
- [x] `const SepError SeparationPolicy` (consts.go:25) — OK
- [x] `const SepRidge SeparationPolicy` (consts.go:26) — OK
- [x] `const SepWarn SeparationPolicy` (consts.go:24) — OK
- [x] `const SkewnessAdjusted` (skewness.go:15) — OK
- [x] `const SkewnessBiasAdjusted` (skewness.go:16) — OK
- [x] `const SkewnessG1 SkewnessMethod` (skewness.go:14) — OK
- [x] `const SpearmanCorrelation` (correlation.go:24) — OK
- [x] `const TwoSided AlternativeHypothesis` (types.go:6) — OK
- [x] `const VarimaxGPArotation VarimaxAlgorithm` (factor_analysis.go:83) — OK
- [x] `const VarimaxKaiser VarimaxAlgorithm` (factor_analysis.go:84) — OK
- [x] `func (r *ChiSquareTestResult) Show()` (chi_square.go:21) — ST-9 無 Writer 版
- [x] `func (r *ExponentialRegressionResult) Predict(typ PredictType, newXs ...insyra.IDataList) (*insyra.DataList, error)` (regression.go:190) — OK 走 gatherRegressionInputs 拒絕不可讀值；ST-4 選項風格
- [x] `func (r *FactorAnalysisResult) Show(startEndRange ...any)` (factor_analysis.go:180) — ST-9 無 Writer 版
- [x] `func (r *GLMResult) Predict(typ PredictType, newXs ...insyra.IDataList) (*insyra.DataList, error)` (glm_predict.go:53) — OK 回 error；PredictType 用字串 enum（可接受）
- [x] `func (r *GLMResult) PredictWithOffset(typ PredictType, offset insyra.IDataList, newXs ...insyra.IDataList) (*insyra.DataList, error)` (glm_predict.go:68) — OK 回 error；PredictType 用字串 enum（可接受）
- [x] `func (r *LassoRegressionResult) Predict(typ PredictType, newXs ...insyra.IDataList) (*insyra.DataList, error)` (regression_regularized.go:280) — OK 走 gatherRegressionInputs 拒絕不可讀值；ST-4 選項風格
- [x] `func (r *LinearRegressionResult) Predict(typ PredictType, newXs ...insyra.IDataList) (*insyra.DataList, error)` (regression.go:142) — OK 走 gatherRegressionInputs 拒絕不可讀值；ST-4 選項風格
- [x] `func (r *LogarithmicRegressionResult) Predict(typ PredictType, newXs ...insyra.IDataList) (*insyra.DataList, error)` (regression.go:206) — OK 走 gatherRegressionInputs 拒絕不可讀值；ST-4 選項風格
- [x] `func (r *LogisticRegressionResult) Predict(typ PredictType, newXs ...insyra.IDataList) (*insyra.DataList, error)` (glm_predict.go:18) — OK 走 gatherRegressionInputs 拒絕不可讀值；ST-4 選項風格
- [x] `func (r *PoissonRegressionResult) Predict(typ PredictType, newXs ...insyra.IDataList) (*insyra.DataList, error)` (glm_predict.go:25) — OK 走 gatherRegressionInputs 拒絕不可讀值；ST-4 選項風格
- [x] `func (r *PoissonRegressionResult) PredictWithOffset(typ PredictType, offset insyra.IDataList, newXs ...insyra.IDataList) (*insyra.DataList, error)` (glm_predict.go:40) — OK 走 gatherRegressionInputs 拒絕不可讀值；ST-4 選項風格
- [x] `func (r *PolynomialRegressionResult) Predict(typ PredictType, newXs ...insyra.IDataList) (*insyra.DataList, error)` (regression.go:165) — OK 走 gatherRegressionInputs 拒絕不可讀值；ST-4 選項風格
- [x] `func (r *RidgeRegressionResult) Predict(typ PredictType, newXs ...insyra.IDataList) (*insyra.DataList, error)` (regression_regularized.go:272) — OK 走 gatherRegressionInputs 拒絕不可讀值；ST-4 選項風格
- [x] `func (r *WeightedLinearRegressionResult) Predict(typ PredictType, newXs ...insyra.IDataList) (*insyra.DataList, error)` (regression_weighted.go:151) — OK 走 gatherRegressionInputs 拒絕不可讀值；ST-4 選項風格
- [x] `func (result *KMeansResult) Assign(dataTable insyra.IDataTable) ([]int, []float64, error)` (clustering.go:34) — OK
- [x] `func BartlettSphericity(dataTable insyra.IDataTable) (chiSquare float64, pValue float64, df int, err error)` (correlation.go:439) — OK；回 4 個裸值（Low）
- [x] `func BartlettTest(groups []insyra.IDataList) (*FTestResult, error)` (ftest.go:112) — ST-1 n 與統計量不一致（已實測）；ST-3；ST-4
- [x] `func CalculateMoment(dl insyra.IDataList, n int, central bool) (float64, error)` (moments.go:35) — ST-2 ToF64Slice
- [x] `func ChiSquareGoodnessOfFit(input insyra.IDataList, p []float64, rescaleP bool) (*ChiSquareTestResult, error)` (chi_square.go:69) — ST-6
- [x] `func ChiSquareIndependenceTest(rowData, colData insyra.IDataList) (*ChiSquareTestResult, error)` (chi_square.go:161) — ST-6
- [x] `func Correlation(dlX, dlY insyra.IDataList, method CorrelationMethod) (*CorrelationResult, error)` (correlation.go:389) — OK 先 requireNumericPair；ST-3 斷言
- [x] `func CorrelationAnalysis(dataTable insyra.IDataTable, method CorrelationMethod) (corrMatrix *insyra.DataTable, pMatrix *insyra.DataTable, chiSquare float64, pValue float64, df int, err error)` (correlation.go:28) — OK 走 numericValues；CorrelationAnalysis 回 6 個值（應包成 struct，Low）
- [x] `func CorrelationMatrix(dataTable insyra.IDataTable, method CorrelationMethod) (corrMatrix *insyra.DataTable, pMatrix *insyra.DataTable, err error)` (correlation.go:53) — OK 走 numericValues；CorrelationAnalysis 回 6 個值（應包成 struct，Low）
- [x] `func Covariance(dlX, dlY insyra.IDataList) (float64, error)` (correlation.go:359) — OK 先 requireNumericPair；ST-3 斷言
- [x] `func CutTreeByHeight(tree *HierarchicalResult, h float64) ([]int, error)` (clustering.go:165) — OK 走 numericMatrixFromTable；ST-4 opts variadic
- [x] `func CutTreeByK(tree *HierarchicalResult, k int) ([]int, error)` (clustering.go:152) — OK 走 numericMatrixFromTable；ST-4 opts variadic
- [x] `func DBSCAN(dataTable insyra.IDataTable, eps float64, minPts int, opts ...DBSCANOptions) (*DBSCANResult, error)` (clustering.go:178) — OK 走 numericMatrixFromTable；ST-4 opts variadic
- [x] `func DefaultFactorAnalysisOptions() FactorAnalysisOptions` (factor_analysis.go:222) — OK options 必填 + Default 建構子；ST-9
- [x] `func Diag(x any, dims ...int) (any, error)` (diag.go:11) — ST-9 any 進出
- [x] `func ExponentialRegression(dlY, dlX insyra.IDataList) (*ExponentialRegressionResult, error)` (regression.go:327) — OK 走 gatherRegressionInputs 拒絕不可讀值；ST-4 選項風格
- [x] `func FTestForNestedModels(rssReduced, rssFull float64, dfReduced, dfFull int) (*FTestResult, error)` (ftest.go:207) — OK 純數值介面，參數驗證完整
- [x] `func FTestForRegression(ssr, sse float64, df1, df2 int) (*FTestResult, error)` (ftest.go:183) — OK 純數值介面，參數驗證完整
- [x] `func FTestForVarianceEquality(data1, data2 insyra.IDataList) (*FTestResult, error)` (ftest.go:18) — ST-1 n 與統計量不一致（已實測）；ST-3；ST-4
- [x] `func FactorAnalysis(dt insyra.IDataTable, opt FactorAnalysisOptions) (*FactorModel, error)` (factor_analysis.go:347) — OK options 必填 + Default 建構子；ST-9
- [x] `func FriedmanTest(subjects ...insyra.IDataList) (*FriedmanTestResult, error)` (nonparam_friedman.go:36) — ST-7 資料形狀；驗證輸入 OK
- [x] `func GLM(opts GLMOptions, dlY insyra.IDataList, dlXs ...insyra.IDataList) (*GLMResult, error)` (regression_glm.go:51) — OK options struct + 驗證
- [x] `func HierarchicalAgglomerative(dataTable insyra.IDataTable, method AgglomerativeMethod) (*HierarchicalResult, error)` (clustering.go:133) — OK 走 numericMatrixFromTable；ST-4 opts variadic
- [x] `func KMeans(dataTable insyra.IDataTable, centers int, opts ...KMeansOptions) (*KMeansResult, error)` (clustering.go:104) — OK 走 numericMatrixFromTable；ST-4 opts variadic
- [x] `func KNNClassify(trainData insyra.IDataTable, trainLabels insyra.IDataList, testData insyra.IDataTable, k int, opts ...KNNOptions) (*KNNClassificationResult, error)` (knn.go:58) — OK；RegisterKNNDeviceSearcher 設計佳
- [x] `func KNNRegress(trainData insyra.IDataTable, trainTargets insyra.IDataList, testData insyra.IDataTable, k int, opts ...KNNOptions) (*KNNRegressionResult, error)` (knn.go:97) — OK 走 gatherRegressionInputs 拒絕不可讀值；ST-4 選項風格
- [x] `func KNearestNeighbors(trainData insyra.IDataTable, testData insyra.IDataTable, k int, opts ...KNNOptions) (*KNNNeighborsResult, error)` (knn.go:121) — OK；RegisterKNNDeviceSearcher 設計佳
- [x] `func KruskalWallis(groups ...insyra.IDataList) (*KruskalWallisResult, error)` (nonparam_kw.go:34) — OK：驗證輸入、alt、CL、對 R 驗證（範本）；ST-3
- [x] `func Kurtosis(data any, method ...KurtosisMethod) (float64, error)` (kurtosis.go:20) — 本輪已修（fix-api-review-batch-1）；`data any` 參數過寬（準則 8）
- [x] `func LassoRegression(dlY insyra.IDataList, alpha float64, dlXs []insyra.IDataList, options ...LassoOptions) (*LassoRegressionResult, error)` (regression_regularized.go:139) — OK 走 gatherRegressionInputs 拒絕不可讀值；ST-4 選項風格
- [x] `func LeveneTest(groups []insyra.IDataList) (*FTestResult, error)` (ftest.go:64) — ST-1 n 與統計量不一致（已實測）；ST-3；ST-4
- [x] `func LinearRegression(dlY insyra.IDataList, dlXs ...insyra.IDataList) (*LinearRegressionResult, error)` (regression.go:225) — OK 走 gatherRegressionInputs 拒絕不可讀值；ST-4 選項風格
- [x] `func LogarithmicRegression(dlY, dlX insyra.IDataList) (*LogarithmicRegressionResult, error)` (regression.go:423) — OK 走 gatherRegressionInputs 拒絕不可讀值；ST-4 選項風格
- [x] `func LogisticRegression(dlY insyra.IDataList, dlXs ...insyra.IDataList) (*LogisticRegressionResult, error)` (regression_logistic.go:58) — OK 走 gatherRegressionInputs 拒絕不可讀值；ST-4 選項風格
- [x] `func LogisticRegressionWithOptions(opts LogisticRegressionOptions, dlY insyra.IDataList, dlXs ...insyra.IDataList) (*LogisticRegressionResult, error)` (regression_logistic.go:62) — OK 走 gatherRegressionInputs 拒絕不可讀值；ST-4 選項風格
- [x] `func MannWhitneyU(data1, data2 insyra.IDataList, alt AlternativeHypothesis, confidenceLevel ...float64) (*MannWhitneyUResult, error)` (nonparam_mwu.go:42) — OK：驗證輸入、alt、CL、對 R 驗證（範本）；ST-3
- [x] `func NormCDF(x float64) float64` (normal.go:18) — OK doc 清楚，錯誤契約明確（範本）
- [x] `func NormPPF(p float64) (float64, error)` (normal.go:31) — OK doc 清楚，錯誤契約明確（範本）
- [x] `func OneWayANOVA(groups ...insyra.IDataList) (*OneWayANOVAResult, error)` (anova.go:63) — OK 驗證輸入；ST-4
- [x] `func PCA(dataTable insyra.IDataTable, nComponents ...int) (*PCAResult, error)` (pca.go:27) — OK 拒絕非數值；`nComponents ...int` variadic（ST-4）
- [x] `func PairedTTest(data1, data2 insyra.IDataList, confidenceLevel ...float64) (*TTestResult, error)` (ttest.go:221) — OK 驗證非數值；ST-3 斷言；ST-4 variadic CL
- [x] `func PairedWilcoxon(data1, data2 insyra.IDataList, alt AlternativeHypothesis, confidenceLevel ...float64) (*WilcoxonTestResult, error)` (nonparam_wilcoxon.go:81) — OK：驗證輸入、alt、CL、對 R 驗證（範本）；ST-3
- [x] `func PoissonRegression(dlY insyra.IDataList, dlXs ...insyra.IDataList) (*PoissonRegressionResult, error)` (regression_poisson.go:52) — OK 走 gatherRegressionInputs 拒絕不可讀值；ST-4 選項風格
- [x] `func PoissonRegressionWithOptions(opts PoissonRegressionOptions, dlY insyra.IDataList, dlXs ...insyra.IDataList) (*PoissonRegressionResult, error)` (regression_poisson.go:56) — OK 走 gatherRegressionInputs 拒絕不可讀值；ST-4 選項風格
- [x] `func PolynomialRegression(dlY, dlX insyra.IDataList, degree int) (*PolynomialRegressionResult, error)` (regression.go:504) — OK 走 gatherRegressionInputs 拒絕不可讀值；ST-4 選項風格
- [x] `func RegisterKNNDeviceSearcher(fn KNNDeviceSearcher)` (knn.go:39) — OK；RegisterKNNDeviceSearcher 設計佳
- [x] `func RepeatedMeasuresANOVA(subjects ...insyra.IDataList) (*RepeatedMeasuresANOVAResult, error)` (anova.go:253) — ST-7 資料形狀；驗證輸入 OK
- [x] `func RidgeRegression(dlY insyra.IDataList, alpha float64, dlXs ...insyra.IDataList) (*RidgeRegressionResult, error)` (regression_regularized.go:85) — OK 走 gatherRegressionInputs 拒絕不可讀值；ST-4 選項風格
- [x] `func Silhouette(dataTable insyra.IDataTable, labels insyra.IDataList) (*SilhouetteResult, error)` (clustering.go:200) — OK 走 numericMatrixFromTable；ST-4 opts variadic
- [x] `func SingleSampleTTest(data insyra.IDataList, mu float64, confidenceLevel ...float64) (*TTestResult, error)` (ttest.go:30) — ST-1 n 與統計量不一致（已實測）；ST-3；ST-4
- [x] `func SingleSampleWilcoxon(data insyra.IDataList, mu float64, alt AlternativeHypothesis, confidenceLevel ...float64) (*WilcoxonTestResult, error)` (nonparam_wilcoxon.go:43) — OK：驗證輸入、alt、CL、對 R 驗證（範本）；ST-3
- [x] `func SingleSampleZTest(data insyra.IDataList, mu float64, sigma float64, alternative AlternativeHypothesis, confidenceLevel float64) (*ZTestResult, error)` (ztest.go:18) — ST-1 n 與統計量不一致（已實測）；ST-3；ST-4
- [x] `func Skewness(sample any, method ...SkewnessMethod) (float64, error)` (skewness.go:20) — 本輪已修（fix-api-review-batch-1）；`data any` 參數過寬（準則 8）
- [x] `func TwoSampleTTest(data1, data2 insyra.IDataList, equalVariance bool, confidenceLevel ...float64) (*TTestResult, error)` (ttest.go:129) — ST-1 n 與統計量不一致（已實測）；ST-3；ST-4
- [x] `func TwoSampleZTest(data1, data2 insyra.IDataList, sigma1, sigma2 float64, alternative AlternativeHypothesis, confidenceLevel float64) (*ZTestResult, error)` (ztest.go:75) — ST-1 n 與統計量不一致（已實測）；ST-3；ST-4
- [x] `func TwoWayANOVA(factorALevels, factorBLevels int, cells ...insyra.IDataList) (*TwoWayANOVAResult, error)` (anova.go:112) — ST-7 資料形狀；驗證輸入 OK
- [x] `func WeightedLinearRegression(dlY insyra.IDataList, dlWeights insyra.IDataList, dlXs ...insyra.IDataList) (*WeightedLinearRegressionResult, error)` (regression_weighted.go:42) — OK 走 gatherRegressionInputs 拒絕不可讀值；ST-4 選項風格
- [x] `type ANOVAResultComponent struct { SumOfSquares float64 DF int F float64 P float64 EtaSquared float64 }` (anova.go:13) — OK
- [x] `type AgglomerativeMethod string` (clustering.go:62) — OK typed enum
- [x] `type AlternativeHypothesis string` (types.go:3) — OK typed enum
- [x] `type BartlettTestResult struct { ChiSquare float64 DegreesOfFreedom int PValue float64 SampleSize int }` (factor_analysis.go:129) — OK 欄位齊全
- [x] `type ChiSquareTestResult struct { testResultBase ContingencyTable *insyra.DataTable }` (chi_square.go:14) — ST-6
- [x] `type CorrelationMethod int` (correlation.go:19) — OK typed enum
- [x] `type CorrelationResult struct { testResultBase }` (correlation.go:384) — OK 欄位齊全；ST-5 基底未匯出
- [x] `type DBSCANOptions struct { BorderPoints *bool }` (clustering.go:84) — OK options struct
- [x] `type DBSCANResult struct { Cluster []int IsSeed []bool }` (clustering.go:88) — OK 欄位齊全
- [x] `type EffectSizeEntry struct { Type string Value float64 }` (structs.go:11) — OK；Type 用字串（可改 typed enum，Low）
- [x] `type ExponentialRegressionResult struct { Intercept float64 Slope float64 Residuals []float64 RSquared float64 AdjustedRSquared float64 StandardErrorIntercept float64 StandardErrorSlope float64 TValueIntercept float64 TValueSlope float64 PValueIntercept float64 PValueSlope float64 ConfidenceIntervalIntercept [2]float64 ConfidenceIntervalSlope [2]float64 }` (regression.go:93) — OK 欄位齊全
- [x] `type FTestResult struct { testResultBase DF2 float64 }` (ftest.go:12) — OK 欄位齊全；ST-5 基底未匯出
- [x] `type FactorAnalysisOptions struct { Count FactorCountSpec Extraction FactorExtractionMethod Rotation FactorRotationOptions Scoring FactorScoreMethod MaxIter int MinErr float64 OptimFactr float64 OptimMaxIter int }` (factor_analysis.go:98) — OK options struct
- [x] `type FactorAnalysisResult struct { Loadings insyra.IDataTable UnrotatedLoadings insyra.IDataTable Structure insyra.IDataTable Uniquenesses insyra.IDataTable Communalities insyra.IDataTable SamplingAdequacy insyra.IDataTable BartlettTest *BartlettTestResult Phi insyra.IDataTable RotationMatrix insyra.IDataTable Eigenvalues insyra.IDataTable ExplainedProportion insyra.IDataTable CumulativeProportion insyra.IDataTable Scores insyra.IDataTable ScoreCoefficients insyra.IDataTable ScoreCovariance insyra.IDataTable Converged bool RotationConverged bool Iterations int CountUsed int Messages []string }` (factor_analysis.go:137) — OK 欄位齊全
- [x] `type FactorCountMethod string` (factor_analysis.go:60) — OK typed enum
- [x] `type FactorCountSpec struct { Method FactorCountMethod FixedK int EigenThreshold float64 MaxFactors int }` (factor_analysis.go:72) — OK options struct
- [x] `type FactorExtractionMethod string` (factor_analysis.go:21) — OK typed enum
- [x] `type FactorModel struct { FactorAnalysisResult }` (factor_analysis.go:212) — OK
- [x] `type FactorRotationMethod string` (factor_analysis.go:32) — OK typed enum
- [x] `type FactorRotationOptions struct { Method FactorRotationMethod Kappa float64 Delta float64 GeominEpsilon float64 Restarts int VarimaxAlgorithm VarimaxAlgorithm }` (factor_analysis.go:88) — OK options struct
- [x] `type FactorScoreMethod string` (factor_analysis.go:50) — OK typed enum
- [x] `type FriedmanTestResult struct { testResultBase NSubjects int KConditions int }` (nonparam_friedman.go:23) — OK 欄位齊全；ST-5 基底未匯出
- [x] `type GLMFamily string` (consts.go:5) — OK typed enum
- [x] `type GLMLink string` (consts.go:6) — OK typed enum
- [x] `type GLMOptions struct { Family GLMFamily Link GLMLink ConfidenceLevel float64 MaxIter int Tolerance float64 Offset insyra.IDataList Weights insyra.IDataList }` (regression_glm.go:11) — OK options struct
- [x] `type GLMResult struct { Family GLMFamily Link GLMLink Coefficients []float64 StandardErrors []float64 ZValues []float64 PValues []float64 ConfidenceIntervals [][2]float64 LinearPredictors []float64 FittedValues []float64 Residuals []float64 PearsonResiduals []float64 DevianceResiduals []float64 Deviance float64 NullDeviance float64 LogLikelihood float64 NullLogLikelihood float64 AIC float64 BIC float64 PearsonChi2 float64 Dispersion float64 DFResidual int Iterations int Converged bool ConfidenceLevel float64 family glmFamily link glmLink hasOffset bool }` (regression_glm.go:21) — OK 欄位齊全
- [x] `type HierarchicalResult struct { Merge [][2]int Height []float64 Order []int Labels []string Method AgglomerativeMethod DistMethod string }` (clustering.go:75) — OK 欄位齊全
- [x] `type KMeansOptions struct { NStart int IterMax int Seed *int64 }` (clustering.go:14) — OK options struct
- [x] `type KMeansResult struct { Cluster []int Centers insyra.IDataTable TotSS float64 WithinSS []float64 TotWithinSS float64 BetweenSS float64 Size []int Iter int IFault int }` (clustering.go:20) — OK 欄位齊全
- [x] `type KNNAlgorithm string` (knn.go:12) — OK typed enum
- [x] `type KNNClassificationResult struct { Predictions insyra.IDataList Classes insyra.IDataList Probabilities insyra.IDataTable }` (knn.go:43) — OK 欄位齊全
- [x] `type KNNDeviceSearcher = internalknn.DeviceSearcher` (knn.go:33) — OK internal 型別別名，用途有 doc
- [x] `type KNNNeighborsResult struct { Indices [][]int Distances [][]float64 }` (knn.go:53) — OK 欄位齊全
- [x] `type KNNOptions struct { Weighting KNNWeighting Algorithm KNNAlgorithm LeafSize int }` (knn.go:24) — OK options struct
- [x] `type KNNRegressionResult struct { Predictions []float64 }` (knn.go:49) — OK 欄位齊全
- [x] `type KNNWeighting string` (knn.go:11) — OK typed enum
- [x] `type KruskalWallisResult struct { testResultBase NTotal int GroupRankSum []float64 }` (nonparam_kw.go:22) — OK 欄位齊全；ST-5 基底未匯出
- [x] `type KurtosisMethod int` (kurtosis.go:11) — OK typed enum
- [x] `type LassoOptions struct { Tolerance float64 MaxIterations int }` (regression_regularized.go:62) — OK options struct
- [x] `type LassoRegressionResult struct { Coefficients []float64 Alpha float64 Residuals []float64 RSquared float64 AdjustedRSquared float64 Converged bool Iterations int }` (regression_regularized.go:42) — OK 欄位齊全
- [x] `type LinearRegressionResult struct { Slope float64 Intercept float64 StandardError float64 StandardErrorIntercept float64 TValue float64 TValueIntercept float64 PValue float64 PValueIntercept float64 ConfidenceIntervalIntercept [2]float64 ConfidenceIntervalSlope [2]float64 Coefficients []float64 StandardErrors []float64 TValues []float64 PValues []float64 ConfidenceIntervals [][2]float64 Residuals []float64 RSquared float64 AdjustedRSquared float64 }` (regression.go:55) — OK 欄位齊全
- [x] `type LogarithmicRegressionResult struct { Intercept float64 Slope float64 Residuals []float64 RSquared float64 AdjustedRSquared float64 StandardErrorIntercept float64 StandardErrorSlope float64 TValueIntercept float64 TValueSlope float64 PValueIntercept float64 PValueSlope float64 ConfidenceIntervalIntercept [2]float64 ConfidenceIntervalSlope [2]float64 }` (regression.go:111) — OK 欄位齊全
- [x] `type LogisticRegressionOptions struct { ConfidenceLevel float64 MaxIter int Tolerance float64 PositiveClass any SeparationPolicy SeparationPolicy Ridge float64 }` (regression_logistic.go:13) — OK options struct
- [x] `type LogisticRegressionResult struct { Link GLMLink Coefficients []float64 StandardErrors []float64 ZValues []float64 PValues []float64 ConfidenceIntervals [][2]float64 OddsRatios []float64 OddsRatioCIs [][2]float64 LinearPredictors []float64 FittedProbabilities []float64 Residuals []float64 PearsonResiduals []float64 DevianceResiduals []float64 Deviance float64 NullDeviance float64 LogLikelihood float64 NullLogLikelihood float64 AIC float64 BIC float64 McFaddenR2 float64 CoxSnellR2 float64 NagelkerkeR2 float64 DFResidual int Iterations int Converged bool SeparationDetected bool Penalized bool Ridge float64 PositiveClass any ClassLabels []any ConfidenceLevel float64 family glmFamily link glmLink }` (regression_logistic.go:22) — OK 欄位齊全
- [x] `type MannWhitneyUResult struct { testResultBase U1 float64 U2 float64 Z float64 Method string }` (nonparam_mwu.go:23) — OK 欄位齊全；ST-5 基底未匯出
- [x] `type OneWayANOVAResult struct { Factor ANOVAResultComponent Within ANOVAResultComponent TotalSS float64 }` (anova.go:29) — OK 欄位齊全
- [x] `type PCAResult struct { Components insyra.IDataTable Center []float64 Scale []float64 Scores insyra.IDataTable Eigenvalues []float64 ExplainedVariance []float64 }` (pca.go:17) — OK 欄位齊全
- [x] `type PoissonRegressionOptions struct { ConfidenceLevel float64 MaxIter int Tolerance float64 Offset insyra.IDataList DispersionCheck bool }` (regression_poisson.go:11) — OK options struct
- [x] `type PoissonRegressionResult struct { Link GLMLink Coefficients []float64 StandardErrors []float64 ZValues []float64 PValues []float64 ConfidenceIntervals [][2]float64 IncidenceRateRatios []float64 IRRConfidenceIntervals [][2]float64 LinearPredictors []float64 FittedRates []float64 FittedValues []float64 Residuals []float64 PearsonResiduals []float64 DevianceResiduals []float64 Deviance float64 NullDeviance float64 LogLikelihood float64 NullLogLikelihood float64 AIC float64 BIC float64 PearsonChi2 float64 DispersionStatistic float64 OverDispersed bool DFResidual int Iterations int Converged bool ConfidenceLevel float64 family glmFamily link glmLink hasOffset bool }` (regression_poisson.go:19) — OK 欄位齊全
- [x] `type PolynomialRegressionResult struct { Coefficients []float64 Degree int Residuals []float64 RSquared float64 AdjustedRSquared float64 StandardErrors []float64 TValues []float64 PValues []float64 ConfidenceIntervals [][2]float64 }` (regression.go:80) — OK 欄位齊全
- [x] `type PredictType string` (glm_predict.go:10) — OK typed enum
- [x] `type RepeatedMeasuresANOVAResult struct { Factor ANOVAResultComponent Subject ANOVAResultComponent Within ANOVAResultComponent TotalSS float64 }` (anova.go:35) — OK 欄位齊全
- [x] `type RidgeRegressionResult struct { Coefficients []float64 Alpha float64 Residuals []float64 RSquared float64 AdjustedRSquared float64 }` (regression_regularized.go:30) — OK 欄位齊全
- [x] `type SeparationPolicy string` (consts.go:7) — OK typed enum
- [x] `type SilhouettePoint struct { Cluster int Neighbor int SilWidth float64 }` (clustering.go:93) — OK
- [x] `type SilhouetteResult struct { Points []SilhouettePoint AverageSilhouette float64 }` (clustering.go:99) — OK 欄位齊全
- [x] `type SkewnessMethod int` (skewness.go:11) — OK typed enum
- [x] `type TTestResult struct { testResultBase Mean *float64 Mean2 *float64 MeanDiff *float64 N int N2 *int }` (ttest.go:13) — ST-5 指標選填不一致
- [x] `type TwoWayANOVAResult struct { FactorA ANOVAResultComponent FactorB ANOVAResultComponent Interaction ANOVAResultComponent Within ANOVAResultComponent TotalSS float64 }` (anova.go:21) — OK 欄位齊全
- [x] `type VarimaxAlgorithm string` (factor_analysis.go:80) — OK typed enum
- [x] `type WeightedLinearRegressionResult struct { Coefficients []float64 StandardErrors []float64 TValues []float64 PValues []float64 Residuals []float64 RSquared float64 AdjustedRSquared float64 }` (regression_weighted.go:20) — OK 欄位齊全
- [x] `type WilcoxonTestResult struct { testResultBase Z float64 Method string NEffective int }` (nonparam_wilcoxon.go:26) — OK：驗證輸入、alt、CL、對 R 驗證（範本）；ST-3
- [x] `type ZTestResult struct { testResultBase Mean float64 Mean2 *float64 N int N2 *int }` (ztest.go:10) — ST-5 指標選填不一致

## tools/gendocs (0)

