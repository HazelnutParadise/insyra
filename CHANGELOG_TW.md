# 變更紀錄

影響 Insyra 使用者的變更，依套件分類，分法與 release note 相同。`## Unreleased` 收錄下一個版本會包含的內容。

v0.3.0 及更早的版本不重複收錄於此，請見 [GitHub Releases](https://github.com/HazelnutParadise/insyra/releases)。

English: [CHANGELOG.md](CHANGELOG.md)

## Unreleased

### Core

- **BREAKING**：Insyra 現在需要 Go 1.26，`go.mod` 的 `go` 指示為 `1.26.9`。Go 1.27 發布後，Go 1.25 就不再有修正版。2026-10-08 公布的十一則標準函式庫漏洞只從 Go 1.26.9 起修正，其中幾則在 `plot.SavePNG` 與 `DataTable.ToSQL` 會用到的 `net/http`。HTTP/2 那幾則在 `golang.org/x/net` 的修正版也需要 Go 1.26。使用 Go 1.21 以上時，除非設定 `GOTOOLCHAIN=local`，`go` 指令會自動下載 1.26.9 工具鏈。
- CCL 字串字面值裡連寫兩個引號代表一個引號，跟 Excel 一樣：`'it''s'` 是 `it's`，`"say ""hi"""` 是 `say "hi"`。以前字串裡沒辦法放跟外層相同的引號。連續兩個引號以前一定編譯失敗，所以原本能編譯的運算式意思都不變。中括號裡的欄名也適用同樣規則（`['O''Brien']`），statement mode 把腳本切成多個敘述時也照這個規則判斷。反斜線仍然是一般字元。
- CCL 新增 `DATEPART(d, unit)`，取出日期的某個部分並回傳數字：`'year'`、`'month'`、`'day'`、`'hour'`、`'minute'` 或 `'second'`，以日期本身的時區來算。這條線上的 `DAY`、`HOUR`、`MINUTE`、`SECOND` 是換算時間長度用的，要取日期的這些部分就用 `DATEPART`。
- CCL 把每一種 Go 整數型別與 `float32` 的格子都當成數字。`int8`、`int16` 與無號整數欄（`parquet.Read` 讀到檔案裡這類欄位時給的就是這些型別）過去不被 CCL 當成數字：`SUM` 回 0、`MAX` 回 `nil`、`A == 1` 回 false，全都不報錯，而 `A * 2`、`IF(A, 'y', 'n')` 與 `ROUND(A, 2)` 會失敗。`A.B` 也接受存成 `int32`、`int64`、`float32` 或其他整數型別的列號，過去會被拒絕；小數部分和 `float64` 列號一樣捨去。`ISNA` 與 `IFNA` 也把 `float32` 的 NaN 當成缺值。
- CCL 的列號是 NaN、無限大或超出 `int32` 範圍時，在每個平台上都會報錯。以前 `float64` 的這種列號在 arm64 會讀到第 0 列，在 amd64 會報錯。
- `engine/ccl` 可以註冊序列函式了：用 `RegisterSequenceFunction(name, fn)` 搭配新的 `SeqFunc` 型別，加入一個吃整欄、回傳同樣長度一欄的函式，就像 `LAG`、`CUMSUM` 那樣。以前從模組外只能註冊純量函式與彙總函式。註冊函式的說明也寫清楚函式登錄表的行為：整個程式共用同一份，可以從任意多個 goroutine 同時使用，名稱不分大小寫，同名再註冊一次會取代同一類的函式。什麼都不做的 `ResetEvalDepth`、`ResetFuncCallDepth` 標為 Deprecated。
- 讀寫不再非得經過硬碟上的檔案。`ReadCSV(r io.Reader, opts)` 可以從任何來源讀 CSV，例如 HTTP 回應、壓縮檔裡的檔案、`embed.FS` 檔案或記憶體中的位元組，`Encoding` 留空時會從開頭的位元組偵測編碼；`ReadCSV_FileWithOptions` 與 `ReadCSV_StringWithOptions` 現在走同一套程式，同樣的位元組不論走哪個入口都得到同樣的表格。`StreamCSV(r, opts, batchSize)` 以 `for dt, err := range insyra.StreamCSV(...)` 一批一批讀 CSV，記憶體裡只放當下這一批；標題列會命名每一批的欄位，型別逐批推斷。`ReadJSON` 接受 `io.Reader`，`ReadExcel(r, sheet, …)` 從來源讀活頁簿，`(*DataTable).WriteCSV` 與 `WriteJSON` 可寫到任何 `io.Writer`，`ToCSVWithOptions` 與 `ToJSON` 現在也透過它們寫檔。既有函式的簽名與結果都沒有改變。
- DataTable 現在可以寫成 Excel。`(*DataTable).ToExcel(path, ExcelWriteOptions)` 把表格寫成活頁簿裡的一張工作表，活頁簿其他工作表維持原樣，檔案不存在時會自動建立，所以一份報表可以一年放一張表。工作表已經存在時會回傳可用 `ErrSheetExists` 比對的錯誤，檔案完全不動；設定 `IfSheetExists: SheetExistsReplace` 才會取代那張表，而且保留它原本的位置，原本隱藏的也維持隱藏，其他工作表定義的名稱也留在原處。`Sheet` 預設為 `Sheet1`。數字、布林值與時間會存成 Excel 的原生值。檔案要整份寫完才會取代原檔，既有檔案的權限不變。`(*DataTable).WriteExcel(w, opts)` 可以把單一工作表的活頁簿寫到任何 `io.Writer`。
- `DataList.DataType()` 與 `DataTable.ColDataTypes()` 說明一欄裝的是哪種值：`DataTypeNumber`、`DataTypeString`、`DataTypeBool`、`DataTypeTime`、`DataTypeOther`、`DataTypeMixed`，沒有值時是 `DataTypeEmpty`。缺值（nil，以及任何浮點型別的 NaN）不算，整數和浮點數都算數字，看起來像數字的文字仍是文字。這條線上 decimal 判為 `DataTypeOther`，因為數值函式不把它當數字讀。`ShowTypes` 開頭多一列 `DataType`，在逐格的 Go 型別上方總結每一欄；DataList 的型別資訊也多一行 `DataType:`。
- DataList 與 DataTable 新增 `ShowHead(n)`、`ShowTail(n)` 以及 `ShowHeadTo`、`ShowTailTo`，是 `ShowRange(n)` 與 `ShowRange(-n)` 更直白的寫法。數量小於 1 時印出錯誤行。
- 在 nil 的 `DataList` 或 `DataTable`（例如 `GetCol` 找不到欄時回傳的值）上呼叫 `Err()`，會回報 `nil DataList` 或 `nil DataTable`，不再崩潰。只有 `Err()` 有這層保護，nil 值上的其他方法照樣會失敗。
- DataTable 的 `Show`、`ShowRange`、`ShowTypesRange` 改成只依要印出的列決定欄寬，不再格式化每一格，所以 `ShowRange(5)` 不會再隨表格變大而變慢。會印出全部列的視圖，輸出逐位元組不變。超過 25 列、只印前 20 列與後 5 列時，若某欄最寬的格子落在沒印出的列，該欄會變窄。`Show` 接受的介面匯出為 `Showable`。
- `DataTable.SliceRows(from, to)` 與 `SliceCols(from, to)` 像 Go 的 `s[from:to]` 那樣取連續的列或欄。`SliceRows` 用 0 起算的位置；`SliceCols` 的兩端可以是 Excel 欄字母或 int 位置，int 可為負數從尾端算，`nil` 代表不限該端：在四欄的表上 `SliceCols("B", "D")`、`SliceCols(1, 3)`、`SliceCols("B", -1)` 是同一個呼叫。無法使用的邊界會記錄錯誤並回傳空表。十個 `FilterRowsByRowIndex…`、`FilterColsByColIndex…` 方法以及 `Headers`、`SetHeaders` 改為 Deprecated，請改用 `SliceRows`、`SliceCols`、`ColNames`、`SetColNames`；它們在 0.3.x 線照常可用。`FilterColsByColIndexLessThan` 與 `…LessThanOrEqualTo` 收到超過最後一欄的字母時回傳所有欄，不再 panic。
- `DataTable.FilterRowsWhere(keep func(row *DataList) bool)` 把整列交給 `keep`，條件可以比較同一列的兩欄。`Filter` 與 `FilterRows` 改用呼叫開始時的欄位運作，判斷式裡新增或刪除欄位不會再讓篩選崩潰或讀錯欄。`FilterByCustomElement` 改為 Deprecated，`Filter` 做的是同一件事。
- `Rolling`、`Expanding`、`EWM` 的選項被拒絕時，各 reducer 回傳的空清單會在 `Err()` 帶著錯誤，原本是沒有錯誤的空清單。`ShiftCol`、`DiffCol`、`PctChangeCol` 與四個 `Cum…Col` 遇到不存在的欄時，回傳帶著表格錯誤的空清單；`GroupedColumnTransform.As` 遇到第一個轉換失敗的群組就停止。成功的呼叫結果不變。
- `GroupBy`、`Pivot`、`Merge` 的鍵以及 `Describe` 的相異值計數，把 `uintptr` 和其他整數寬度視為同一群，`Count` 與 `Counter` 原本就是如此。
- `DataList.Difference`、`MovingAverage`、`MovingStdev`、`WeightedMovingAverage` 改為 Deprecated，行為不變，在 0.4 移除。替代寫法 `Diff(1)`、`Rolling(RollingOptions{Window: w}).Mean()`、`Rolling(...).Std()` 與帶 `Weights` 的 `Rolling(...)` 會保留輸入長度，並在舊方法失敗或給 NaN 的地方給 `nil`；`Docs/DataList.md` 列出每一項差異。
- `SqrtRat`、`PowRat`、`SortTimes`、`F64orRat` 改為 Deprecated，它們和資料表無關，在 0.4 移除。`SqrtRat` 遇到 nil 或負值、`PowRat` 遇到 nil 底數時回傳 nil，原本會 panic；`PowRat` 收到負指數時回傳倒數次方，原本回傳 1。
- `FillNaNWithMean` 的 Deprecated 說明改成給出保留其行為的寫法 `dl.ReplaceNaNsWith(dl.Clone().ClearNilsAndNaNs().Mean())`；原本指向會連 nil 一起填的 `FillWithMean`。
- MySQL 上用 `Replace` 的 `ToSQL` 寫入失敗時會保留舊表。MySQL 在 `DROP TABLE` 時就結束交易，所以插入失敗後舊表和新資料都沒了。現在資料先寫進暫存表，再用一次 `RENAME TABLE` 換上，最後才刪掉舊表；被其他表以外鍵參照的表會在任何變更前被拒絕。帳號需要資料庫層級的 `CREATE`、`INSERT`、`ALTER`、`DROP` 權限。SQLite 與 PostgreSQL 照舊使用單一交易。
- `ScalerParams.PassThrough` 改為 Deprecated，在這條線意義不變，在 0.4 移除；0.4 的 `SimpleImputer` 會在 fit 時拒絕這種欄位。
- 新增 `insyra.IsNilList(dl)`，判斷一個 `IDataList` 背後是否沒有清單，包含內嵌 nil `*DataList` 的包裝（例如 `isr.DL`）。`plot` 圖表與 `stats` 函式收到這種包裝時，會當成 nil 清單處理，不再 panic（[#412](https://github.com/HazelnutParadise/insyra/issues/412)）。
- `ReadSlice2D` 的表格寬度改以最寬的一列為準。過去比第一列長的列，多出來的值會默默消失，也沒有錯誤：`[][]float64{{1}, {2, 3}}` 得到的 2x1 表格裡沒有 3。較短的列仍補 nil（[#410](https://github.com/HazelnutParadise/insyra/issues/410)）。
- `NewDataList` 收到包含自己的 slice 時，不再以 `fatal error: stack overflow` 讓整個程式結束。它最多往下展開 64 層，更深的部分當成一個儲存格保留；一般的巢狀照舊展開。
- `DataList.Shift` 會把儲存格裡的 slice 當成一個儲存格保留。過去它透過會展開所有 slice 的 `NewDataList` 建立結果，所以含有 `[]byte{1, 2}` 與 `3` 的清單經 `Shift(0)` 後會變成三格。
- `quant`、`ml`、`nn` 與 `DataTable.Resample` 中指出壞儲存格的錯誤訊息，現在一律寫 `row index N`，從 0 算起，與 `Get(N)` 一致；過去 `quant` 與 `Resample` 從 1 算（`row 3`），`ml` 與 `nn` 從 0 算但只寫 `row N`。只有文字改變，比對舊文字的程式需要更新。
- 讀入 JSON 的 scaler 會拒絕 `"ref"` 為空的欄位。過去它會綁到第一個沒有名稱的欄並默默改掉該欄的值。`Fit` 不會寫出空參照，所以只有手寫的 JSON 會遇到。
- 兩個巢狀值的字串內含編碼用的分隔字元時，不再被當成同一個值：清單裡有 `Cell([]any{"a,s:b"})` 與 `Cell([]any{"a", "b"})` 時，`Count([]any{"a", "b"})` 會回 2，`Counter` 只有一筆。巢狀字串現在加上長度前綴。非巢狀的儲存格，以及不含這類字串的巢狀值，計數與過去相同。
- `Config.TurnOnThreadSafety()` 在 `Config.Dangerously_TurnOffThreadSafety()` 之後把 `DataList` 與 `DataTable` 的實例鎖開回來，其他設定都不動。以前只有 `SetDefaultConfig()` 能開回來，但它會連 log 等級、彩色輸出、panic-on-error、錯誤 hook 與加速設定一起重設。`Dangerously_TurnOffThreadSafety` 的文件也寫明了關掉的範圍：`AtomicDo` 與 `AtomicDoAll` 的鎖，也就是 `DataList` 與 `DataTable` 的每個方法；`Config`、全域錯誤緩衝區與 log 仍然安全。
- `Config.SetLogger(*slog.Logger)` 讓 insyra 的日誌改送到 `log/slog` 的 logger，不再寫進標準 `log` 套件。每一行會變成一筆 record，保留原本的等級（Fatal 對應 `ERROR+4`），帶有 `package` 與 `func` 屬性，來源標示為寫日誌的那個 insyra 函式，而且不加 `[insyra - …]` 前綴和顏色碼。`SetLogger(nil)` 會改回標準 `log` 套件，`GetLogger` 回傳目前設定的 logger，`SetDefaultConfig` 會清除它。哪些行要送出仍由 `SetLogLevel` 決定，`LogFatal` 在沒有開 `SetDontPanic(true)` 時仍會結束程式。沒有設定 logger 時，輸出和以前完全一樣。CCL 的警告現在也會遵守日誌等級，並送到 logger、錯誤緩衝區與錯誤 hook。
- `ColIndexToNumber` 與 `ColNumberToIndex` 在 Excel 式欄位索引與從 0 起算的欄位編號之間轉換：`ColIndexToNumber("AA")` 是 `26`，`ColNumberToIndex(26)` 是 `"AA"`。`Config.GetUseColoredOutput` 回報 `SetUseColoredOutput` 設定的值。`ParseColIndex`、`CalcColIndex` 與 `Config.GetDoesUseColoredOutput` 標為 **Deprecated**，回傳值與新名稱相同，0.4 移除。`Config.Dangerously_TurnOffThreadSafety` 刻意保留底線作為警告；取值方法保留 `Get` 前綴，與對應的 `Set` 方法成對。
- `DataTable.IsEqualTo(other)` 與 `IsTheSameAs(other)` 用來比較兩張表。`IsEqualTo` 在兩張表的列數、欄數相同且每一格都相等時為 true（型別必須相同，兩個 NaN 視為相等，短欄超出的格子視為 `nil`），不看表名、欄名與列名。`IsTheSameAs` 另外要求表名、欄名與列名依序相同。
- `DataList` 與 `DataTable` 新增 `CreationTime()` 與 `LastModifiedTime()`，回傳依時鐘完整精度記錄的 `time.Time`，相隔不到一秒的兩次修改也分得出來。`GetCreationTimestamp` 與 `GetLastModifiedTimestamp` 標為 **Deprecated**，仍回傳 Unix 秒，0.4 移除。`DataList.IsTheSameAs` 仍以整秒比較這兩個時間，和以前一樣。
- `DataList.Quantile(p)` 以 pandas、R 與 gonum 使用的 0 到 1 刻度回傳 p 分位數：採用 R 的 type-7 分位數，與 `Percentile`、`Quartile` 使用的定義相同。`nil` 與 `NaN` 會被略過；`p` 超出 `[0, 1]`、儲存格不是數字或整個列表沒有數字時，會記錄警告並回傳 `NaN`。
- `DataList.Shuffled` 與 `DataTable.Shuffled` 回傳一份名為 `<名稱>_Shuffled` 的洗牌副本，原物件不變。這個版本的 `Shuffle` 仍回傳副本，也沒有棄用，但 0.4 的 `Shuffle` 會就地洗牌，要副本請用 `Shuffled`。
- `ReadSlice2D` 現在是把二維切片轉成 DataTable 的函式，`Slice2DToDataTable` 標為 **Deprecated**，0.4 移除。`ReadJSONFile`、`DataTable.ToJSONBytes` 與 `DataTable.ToJSONString` 取代 `ReadJSON_File`、`ToJSON_Bytes` 與 `ToJSON_String`；舊名稱標為 **Deprecated**，行為不變，0.4 移除。
- 修正 `Counter` 對每個 `NaN` 各建立一個取不回來的項目的問題。`NaN` 在 Go 中可以比較，但永遠不等於自己，所以每次 `counter[NaN]++` 都會產生一個沒有人查得到的新 key：一欄有三個 `NaN` 時，結果是三筆各為 1，而 `Count` 正確回報 3。現在同一欄的所有 `NaN` 合成一個項目，計數正確，用 `counter[insyra.ToMapKey(math.NaN())]` 讀取。含有 `NaN` 的陣列或結構同樣不等於自己，也比照處理。`counter[math.NaN()]` 仍然回傳 0，這點一直如此、也必然如此，因為 Go 的 map 無法比對 `NaN` key。
- 呼叫已經用回傳值告訴你成功時，insyra 不再另外寫一行日誌。`csvxl` 的轉換函式（`CSVToExcel`、`AppendCSVToExcel`、`ExcelToCSV`、`ExcelDirToCSV` 與它們已棄用的舊拼法），以及 `plot.SaveHTML`、`plot.SavePNG`、`gplot.SaveChart`、`py.PipInstall`、`PipUninstall`、`PipList` 的 Info 訊息都拿掉了。留下來的 Info 訊息都是回傳值沒說的事：insyra 猜出來的編碼、`ToSQL` 幫既有資料表加的欄位、`nn.Sequential.Fit` 與準備 Python 環境的進度，以及加速與 thread safety 開關的切換。預設等級維持 `LogLevelInfo`。只想看警告和錯誤，可以呼叫 `SetLogLevel(insyra.LogLevelWarning)`。
- CCL 中單獨作為整個運算式的欄位範圍或 `@`，現在是目前這一列的值。`AddColUsingCCL("r", "A:B")` 在每個儲存格放入該列 A、B 的值（`[]any`），過去每格放的是呼叫端無法使用的內部範圍值；`LAG(@, 1)`、`LAG(A:B, 1)`、`LEAD(@, 1)` 與 `LEAD(A:B, 1)` 會給每一列前一列或後一列的內容，過去 `LAG(@, 1)` 會把整張攤平的表位移成一個 slice，重複放進每一列。其他序列函數遇到整列時會回報錯誤，因為它們要對每個元素做運算；單獨的列範圍（例如 `1:2`）也是錯誤，訊息會提示改寫成 `A.(1:2)`。`SUM(A:C)`、`SUM(A.(0:1))` 與其他讀取範圍的地方不變。
- `CsvToExcel`、`CSVToExcel`、`AppendCsvToExcel` 與 `AppendCSVToExcel` 會照呼叫端寫的路徑讀取輸入檔。名為 `export.txt`、`DATA.CSV` 或完全沒有副檔名的 CSV 會當成它本身讀入；過去所有不以小寫 `.csv` 結尾的名稱都會被補上 `.csv`，這些檔案因此打不開。路徑指向的檔案不存在時仍會嘗試補上 `.csv`，所以以前讀得到的檔案現在照樣讀得到。`x` 與 `x.csv` 同時存在時，讀的是 `x`。
- 值是數字的 enum 現在都會印出名稱。`MergeMode`、`NaNPolicy`、`UnknownPolicy`、`LabelSort`、`ResampleFreq`、`SheetExistsPolicy`、`SQLActionIfTableExists`、`MergeDirection` 與 `ErrPoppingMode` 新增 `String()`，`quant`（`VaRMethod`、`OptionType`、`PortfolioObjective`）、`finance`（`DayCountBasis`、`PaymentTiming`）、`datafetch`、`parquet`、`stats`、`ml` 與 `lp` 的數字 enum 也一樣。`fmt.Print(quant.VaRParametric)` 印出 `VaRParametric`，不再是 `1`。不在常數之內的值印成 `型別名(n)`，原本就有 `String()` 的型別也一樣：`DataType(9)` 過去印 `unknown`、`LogLevel(9)` 印 `UNKNOWN`、`AggregateOp(99)` 印 `op(99)`、`ml.ClassAverage(9)` 印 `macro`。回報未知設定的錯誤會寫出名稱：`ValueAtRisk` 收到 `VaRMethod(3)` 時，訊息從 `VaR: unknown method 3` 改為 `VaR: unknown method VaRMethod(3)`。所有 enum 的底層型別都沒有變，呼叫端不用改動。
- **BREAKING**：`ToSQL` 遇到沒有名字的欄位，改用欄位字母（`A`、`B`…）當欄名，不再寫成 MySQL 和 PostgreSQL 都不接受的空白名稱 `""`。例如 `NewDataTable(NewDataList(1, 2, 3)).ToSQL(db, "t")` 在 MySQL 上以前會失敗並回傳 `Incorrect column name ''`，現在會寫出欄位 `A`。寫進 SQLite 的表也一樣，以前的 `""` 欄位現在會叫 `A`。如果已經有別的欄位用了這個字母當名稱（不分大小寫），在寫入任何東西之前就會回傳錯誤。
- **BREAKING**：CCL 把浮點數轉成文字時，改寫成一般的十進位數字。A = 0.0000001 時，`'x' & A` 現在是 `x0.0000001`，不再是 `x1e-07`；`CONCAT('n=', 1000000)` 是 `n=1000000`，不再是 `n=1e+06`；`LEN(1000000)` 是 7，跟整數欄位存 1000000 時一致，以前是 5。範圍包含 `&`、`CONCAT`、沒有指定格式的 `TOSTR`，以及所有字串函式。1e21 以上、NaN 與無限大維持原本寫法，`ToCSV`、`ToJSON`、`Show` 不變，因為科學記號的數字儲存格讀回來還是同一個數。用浮點數組字串的運算式會得到不同的字串；想要舊寫法請用 `TOSTR(x, '%g')`。
- 過去用 1 起算數字指出列或欄的錯誤與警告訊息（`ToExcel`、`ToSQL`、`csvxl` 的 Excel 轉 CSV、`mkt.RFM`），現在改寫成從 0 算的 `row index N` / `column index N`，與函式庫其他地方一致。
- 巢狀超過 32 層 `[]any` 的值，不再因為只有很深的葉子不同，就在 `Count` 與查找時被當成同一個值；超過深度限制後，容器以位址識別，所以這種值的另一份複本是不同的值。原本能用的行為不變，因為 `Count` 在 v0.3.2 對這類值會 panic。
- 把包含自己的 slice 用 `Append` 放進清單後，`Show` 印出它、或 `stats` 函式拒絕它時，不再因 stack overflow 讓程式結束，這種儲存格改以型別顯示。其他儲存格的顯示不變。
- 編碼偵測對相同的位元組給出相同的答案。chardet 的辨識器平手時，例如 24 位元組的 Big5 CSV，過去每次呼叫可能被判成 Shift_JIS、GB18030 或 Big5；現在平手時依編碼名稱決定。低信心猜測的處理方式不變。

### CLI

- `save <var> <file.xlsx>` 把表格寫成活頁簿裡的一張工作表：`sheet <name>` 指定工作表（預設 `Sheet1`），`if-exists replace` 會取代已經存在的工作表，不加則拒絕寫入。活頁簿裡的其他工作表都保留。
- `rolling` 的選項被函式庫拒絕時（例如 `minobs` 大於視窗），指令會帶著原因失敗，不再存下一個空結果。
- `anova twoway`、`anova repeated` 與新的 `friedman` 指令可直接用每列一筆觀測值的表格，也就是讀進 CSV 後常見的形狀：`anova twoway scores score drug dose`、`anova repeated trial value visit patient`、`friedman trial value visit patient`。第一個參數是 DataTable 變數時採用這種形式；用 DataList 變數時，`anova twoway` 與 `anova repeated` 和以前完全一樣，`friedman s1 s2 s3` 則是每位受試者一個清單。`friedman` 印出 `Q`、自由度與 p 值。
- CLI 新增等級檢定：`wilcoxon single <var> <mu>`、`wilcoxon paired <var1> <var2>`、`mannwhitney <var1> <var2>`（都可加 `two-sided`、`greater` 或 `less`）以及 `kruskal <group1> <group2> [groupN]`，分別印出 `W=… p=…`、`U=… p=…`、`H=… df=… p=…`。以前只能從 Go 使用。
- `fillna … ffill|bfill … missing nan|nil` 不再把另一種缺值算進 `limit`：對 `[1, nil, NaN]` 執行 `fillna x ffill limit 1 missing nan` 會得到 `[1, nil, 1]`，原本 `NaN` 會留著沒填。沒有指定 `limit` 時，以及 `mean`、`median`、`mode`、`interpolate`，結果都不變。
- `engine/dsl` 提供程式不必 import `cli/` 就能用的功能：`Manager`、`NewManager`、`DefaultManager` 用來開啟與建立環境，`Register` 讓你加入自己的指令，並可設定參數數量上限（例如 `MaxArgs(1)`）。`NewSession` 遇到還不存在的環境會建立它，多個 session 同時啟動也安全；`insyra` CLI 的 `--env` 仍會拒絕不存在的環境。
- `cli/env` 的每個環境操作只留一個名字，也就是 `Manager` 上的方法；`SetBasePath`、`LoadState`、`ReadHistory` 等套件層級函式改為呼叫 `Default()` 的 Deprecated 包裝，照常可用。`env clear --keep-history`、`env import --force`、`accel --mode` 等一次性旗標改成和指令一起宣告，行為不變。
- 會存下多個變數的指令（例如 `pca`、`kmeans`）現在會在輸出中列出存下的每個變數名稱，存下的值不變。
- `ttest single` 與 `ztest single` 在一次性模式接受負數的 `mu`；原本 `insyra ttest single x -1` 會把 `-1` 當成旗標而失敗。`wilcoxon single` 本來就可以。
- 新增 `quantile <var> <p>` 指令，印出 DataList 的分位數，`p` 範圍為 0 到 1；`percentile` 則是 0 到 100。`shuffle ... as` 指令改呼叫 `Shuffled`，所以在每個版本線上都不會改動來源變數的順序。
- 環境的預設檔案寫入失敗時，`env create`（以及任何會建立 `default` 環境的指令）會移除建到一半的資料夾，不再留下殘缺的環境；預設檔案的路徑上若是一個資料夾，會回報錯誤，不再略過。
- `convert <file.xlsx> sheet <name> <file.csv> [sheet <name> <file.csv> ...]` 只轉出指名的工作表，各自寫成指定的檔案，檔案可以在不同目錄。指定不存在的工作表、同一張工作表給兩次，或兩張工作表寫到同一個檔案，都會在寫出任何檔案前回錯誤。這個形式預設會在試算表會當成公式執行的文字前加單引號，給 `allowformulas true` 才原樣寫出。`convert <file.xlsx> <file.csv>` 照舊。新增的 `sheets <file.xlsx>` 依順序列出活頁簿的工作表。
- `col` 現在接受 Excel 風格的欄位字母：`insyra col t A as x`。整數仍是從 0 開始的位置，欄名會先於字母比對，所以原本能用的呼叫意思不變，名稱剛好叫 `A` 的欄位也仍然優先於 A 欄（[#438](https://github.com/HazelnutParadise/insyra/issues/438)）。
- 所有接受數字的指令參數，現在也可以填數字變數的名稱：`sample frac f`、`kmeans … k`、`shift x n`、`ttest … mu`、`quant` 各形式等，會使用該變數的值；過去填非數字的名稱會報錯。數字變數可由 `sum`、`mean`、`median`、`mode`、`stdev`、`var`、`min`、`max`、`range` 結尾加上 `as <var>` 產生（照常印出結果，只有寫了 `as` 才存下），並可用 `show <var>` 顯示。能解析成數字的字串仍視為數字，所以原本能用的寫法意義不變。
- 兩個 `insyra` 指令同時使用同一個環境時，不會再互相蓋掉對方的變數。過去存檔會把自己手上那份 `state.json` 整份寫回去，後存的就吃掉先存的；現在只把自己這個指令新增、修改、刪除的部分，疊在存檔當下檔案裡的內容上，過程只用一個很短的鎖和各自獨立的暫存檔。指令之間不會因此排隊等待。
- `state.json` 讀不了（被截斷、手動改成不合法的 JSON）時，指令現在會停下來並報錯，錯誤訊息指出檔名。以前指令會拿一份空的變數去跑，存檔時把檔案整個蓋掉，環境裡所有變數就此消失；`env open` 還會把前一個環境的變數帶進剛打開的環境。現在什麼都不執行，檔案維持原樣，可以讓你修復。完全沒有 `state.json` 的環境仍然從空的開始。
- 修正 Windows 上兩個指令同時儲存同一個環境時會遺失變數。Windows 在鎖檔正被另一個程序刪除時，建立鎖檔回報的是「存取被拒」而不是「已存在」，儲存因此直接放棄而不是等待。

### `ml` 與 `nn`

- `nn.CustomLoss{Name, Loss, Validate}` 讓 `Sequential.Fit` 能用呼叫者在 tape 上計算的 loss 訓練，原本只接受 `CrossEntropy`、`MSE`、`BCEWithLogits`。`Loss` 回傳由 tape 運算組成、型別為 float32 的純量 loss，`Validate` 可選，用來檢查每個批次的目標值，`Name` 用於錯誤訊息。`Fit` 會在第一個批次前拒絕沒有 `Loss` 的 `CustomLoss`，也會拒絕回傳 nil、不是 float32 純量，或不在 tape 上算出的結果。
- `Sequential.FitContext(ctx, x, y, cfg)` 讓訓練可以中途取消。每個批次前檢查 context；context 結束時回傳 `ctx.Err()` 與已完成的 epoch，模型保留已經做過的更新。`Fit` 就是以 `context.Background()` 呼叫 `FitContext`，結果不變。
- `nn.LayerNormShape(dims []int)` 對最後幾個維度做正規化，相當於 torch 的 `normalized_shape` 清單。`LayerNorm` 保留 `interface{}` 參數與原本接受的輸入。
- `nn` 的每個東西只留一個名字。下列名稱改為 Deprecated，意義不變，在 0.4 移除：十二個層建構子的 `New…` 版本（`NewDense`、`NewReLU`、`NewDropout`、`NewFunc`、`NewMultiHeadAttention`、`NewConv2D`、`NewMaxPool2D`、`NewAvgPool2D`、`NewGlobalAvgPool`、`NewBatchNorm2D`、`NewLayerNorm`、`NewEmbedding`；請用沒有 `New` 的名稱）；loss 選擇器 `SoftmaxCrossEntropy`、`MSELoss`、`BCEWithLogitsLoss`（請用 `CrossEntropy`、`MSE`、`BCEWithLogits`）；`Classifier`、`Regressor`（請用 `BoundClassifier`、`BoundRegressor`）；`MaxPoolOptions`、`AveragePoolOptions`（請用 `PoolOptions`）；`DataType` 與常數 `Float32`、`Float16`、`Float64`（請用 `DType` 與 `DType…` 常數）；`NewFloat32Tensor`、`NewTensorWithDType`（請用 `NewTensor`）；以及 `Tensor.Data`（請用 `Tensor.Float32Data`，它會把其他 dtype 回報為錯誤）。
- 在 nil 的迴歸模型（`LinearModel`、`RidgeModel`、`LogisticModel` 等迴歸系列）上呼叫 `Predict`，或把 nil 模型指標交給 `ExportONNX`，會回傳錯誤，不再 panic。`Features` 仍是模型值上的方法，`Fit` 系列函式的回傳型別不變。
- `ml.WriteONNX` 改為 Deprecated，`ExportONNX` 是同一個函式，參數仍是 `any`。`DecisionTreeOptions` 的別名 `DecisionTreeClassifierOptions`、`DecisionTreeRegressorOptions` 也改為 Deprecated。兩者都在 0.4 移除。
- 新增 `nn.CustomOptimizer{Name, Step}` 與 `Parameter.Update(fn)`，`Sequential.Fit` 可以用呼叫者自己寫的 optimizer 訓練，不再只限 `SGD`、`SGDMomentum`、`Adam`、`AdamW`。`Fit` 每個 batch 呼叫一次 `Step`，傳入模型 tape 上的所有參數；`Update` 把參數的 float32 資料和梯度交給 `fn`（loss 沒有影響到這個參數時，梯度是 nil），`fn` 寫進去的值會保留下來，手寫的訓練迴圈也能用。`Fit` 會在跑任何 batch 之前拒絕沒有 `Step` 的 `CustomOptimizer`。內建的 optimizer 行為不變。（[issue #266](https://github.com/HazelnutParadise/insyra/issues/266)）
- CPU 上的 2-D float32 `MatMul` 快了 1.2 到 2.9 倍：在 4 vCPU 的 Linux amd64 機器上，1×784×28 從 23.0 µs 降到 19.5 µs，64×784×28 從 694 µs 降到 331 µs，256×256×256 從 5.32 ms 降到 1.81 ms。每個輸出仍照原本的順序累加乘積，所以不是 NaN 的結果每個位元都和原本相同；兩個 NaN 在同一次加法相遇時，得到的 NaN 位元可能不同。（[#463](https://github.com/HazelnutParadise/insyra/issues/463)）

### `datafetch`

- 建構子回傳的客戶端改為匯出型別：`TWStock` 回傳 `*TWStockClient`，`YFinance` 回傳 `*YFinanceClient`，它的 `Ticker` 回傳 `*YFTicker`，`TWGeocoding` 回傳 `*TWGeocodingClient`，`GoogleMapsStores` 回傳 `*GoogleMapsStoresClient`。現在可以把客戶端存進自己的 struct 欄位或當成參數傳遞，既有的呼叫不必修改。不是由建構子建立的 TWSE／TPEx 或地理編碼客戶端（例如建構子出錯時回傳的 `nil` 或零值），過去會直接 panic 的地方現在回傳錯誤；原本就有回應的呼叫，回應不變。
- 每個抓取方法都有接收 `context.Context` 的 `...Context` 版本，做法與 `ReadSQLContext` 相同，原本的方法改為以 `context.Background()` 呼叫它：`TWStockClient` 的六個方法（例如 `DailyPricesContext`）、`ReverseContext`、`ReverseColsContext`、`ReverseTableContext`、`SearchContext`、`GetReviewsContext`，以及 `YFTicker` 所有會發出請求的 26 個方法（例如 `HistoryContext`）。context 會傳到限流等待、重試前的退避、每個 HTTP 請求與評論翻頁之間的等待。被取消的呼叫回傳 `ctx.Err()` 且不重試。`ReverseContext` 的請求若被 context 中斷，回報的是 context 的錯誤而不是 `ErrGeocodeTimeout`，`ErrGeocodeTimeout` 仍只代表服務沒在 `Timeout` 內回應。`ReverseCols` 批次中途被取消時，會回傳已解析的列，其餘標為 `pending`，與配額用完時相同。go-yfinance 無法中途停止已經開始的呼叫，所以 Yahoo Finance 的方法在 context 結束時會立刻返回，那次呼叫則留在背景跑完，期間仍可能送出剩下的請求。`ReverseTableByColName` 沒有 Context 版本，請把同樣的欄位以 Excel 式索引傳給 `ReverseTableContext`。
- `YFPeriodYearly` 改為 Deprecated，請改用抓取相同財報的 `YFPeriodAnnual`，它產生的表格仍標示 `yearly`。評論排序常數改為 `GoogleMapsStoreReviewSortByRelevance`、`GoogleMapsStoreReviewSortByNewest`、`GoogleMapsStoreReviewSortByHighestRating`、`GoogleMapsStoreReviewSortByLowestRating`，`SortByRelevance` 等四個舊名改為數值相同的 Deprecated 常數。`GoogleMapsStoreReviewsFetchingOptions.MaxWaitingInterval` 以 `time.Duration` 設定翻頁之間最長的等待，規則與原本的毫秒欄位相同，`MaxWaitingInterval_Milliseconds` 改為 Deprecated，兩個都設定時 `GetReviews` 會在發出請求前回傳 `nil` 並記錄警告。這些 Deprecated 名稱在 0.3.x 線上照常可用。
- `TWStock` 的方法遇到 HTTP 307 或 429 時，會明說是被來源節流：錯誤訊息指出來源認為請求太頻繁，請調大 `Interval` 與 `Retries`，`errors.Is(err, datafetch.ErrTWStockThrottled)` 也會成立（[#434](https://github.com/HazelnutParadise/insyra/issues/434)）。證交所會隨機以安全性頁面回 307，所以 `Docs/datafetch.md` 補上回填歷史時的起始設定值。預設值不變。
- 新增 `TWStockClient.OddLotQuotes(date, session, market)`，讀取指定市場的每日零股行情單，盤中（`TWOddLotIntraday`，2020-10-26 起）或盤後（`TWOddLotAfterHours`），市場可選證交所或櫃買中心（上櫃，盤後 2011 年 1 月起），回傳數值已轉型、兩個市場欄位相同的資料表；證交所顯示為 `--` 或櫃買顯示為 `0.00` 的價格是 `nil`，沒有報表的日子回傳空表（[#451](https://github.com/HazelnutParadise/insyra/issues/451)）。
- 新增 `TWStockClient.ShareholdingDistribution()`，讀取集保結算所每週的股權分散表（每檔證券在 17 個持股分級的人數與股數），回傳數值已轉型的資料表。只提供最新一週，需要時間序列請每週自行存檔（[#436](https://github.com/HazelnutParadise/insyra/issues/436)）。
- 新增 `TWStockClient.DailyQuotesOn(date, market)`，回傳任一過去交易日所有證券的收盤行情，來源是證交所（2004-02-11 起，含之後下市的證券）或櫃買中心，`Change` 帶正負號並以交易所自己的參考價計算，另有交易所的漲跌標記 `Sign`。新增 `TWStockClient.AdvanceDecline(date, market, commonStockOnly)`，交易所有公布時回傳官方的上漲、下跌家數（含漲停、跌停），沒有時由收盤行情計算（[#447](https://github.com/HazelnutParadise/insyra/issues/447)）。
- 新增 `datafetch.TAIFEX(cfg)`，讀取期交所免金鑰的公開資料：`InstitutionalFutures`、`InstitutionalOptions`（三大法人的成交與未平倉）、`LargeTradersFutures`、`LargeTradersOptions`（大額交易人的未沖銷部位）與 `PutCallRatio`，每個都是數值已轉型的資料表。OpenAPI 只有最新一天（Put/Call 比有 20 天），不讀取網站上更長的歷史（[#437](https://github.com/HazelnutParadise/insyra/issues/437)）。
- 新增 `TWStockClient.MonthlyRevenue(year, month, market)`，從公開資訊觀測站的彙總頁讀取所有上市櫃公司的月營收（與累計營收），2003 年 1 月起，每頁一次請求，回傳數值已轉型的資料表。`PageDate` 是整頁的出表日期，不是各公司公告日（[#435](https://github.com/HazelnutParadise/insyra/issues/435)）。
- 新增 `datafetch.FinMind(cfg)`，連到 [FinMind](https://finmindtrade.com)，一次請求就能拿到一檔證券的整段歷史。`FinMindClient.Dataset(name, dataID, from, to)` 把任何資料集回傳成數值已轉型的資料表，欄位照 FinMind 原樣。超過請求上限（`ErrFinMindLimit`）與方案權限不足（`ErrFinMindLevel`）可以和其他錯誤分辨，而且不會重試（[#433](https://github.com/HazelnutParadise/insyra/issues/433)）。
- 新增 `datafetch.JPX(cfg)` 與 `JPXClient.InvestorTypes(from, to, unit)`，讀取日本交易所集團每週的投資部門別売買状況（2016 年起）：每週、每個市場、14 個投資人類別各一列，單位為千日圓或千股。舊的 `.xls` 與 2026 年 9 月起每週一份的 `.xlsx` 都能讀，並處理假日週與跨年週。數字在下一週的第四個營業日公布（[#468](https://github.com/HazelnutParadise/insyra/issues/468)）。

### `stats`

- `DiagOf`、`DiagMatrix`、`DiagMatrixSize`、`IdentityMatrix` 以單一型別建立與讀取對角線，大小小於 1 或矩陣為 nil 時回傳錯誤而不是 panic。`Diag(x any, dims ...int)` 改為 Deprecated，行為不變（包括大小小於 1 時 panic），在 0.4 移除。
- `TwoWayANOVAFromTable(dt, valueCol, factorACol, factorBCol)`、`RepeatedMeasuresANOVAFromTable(dt, valueCol, conditionCol, subjectCol)` 與 `FriedmanTestFromTable(dt, valueCol, conditionCol, subjectCol)` 讀取每列一筆觀測值的表格，也就是 R 的 `aov` 與 `friedman.test` 讀的形狀，不必再手動拆成各格或每位受試者一個清單。欄位指定為既有欄的 Excel 欄字母，找不到時再比完全相同的欄名。因子層級依 `ToMapKey` 比較，所以 `int(1)` 與 `int64(1)` 是不同層級。`nil` 或 `NaN` 層級、不是有限數字的值、沒有觀測值的組合、各格內沒有重複觀測的雙因子表，以及缺少某個條件或同一條件量了兩次的受試者，都會被拒絕。結果與用同樣資料呼叫 `TwoWayANOVA`、`RepeatedMeasuresANOVA`、`FriedmanTest` 相同。
- `ChiSquareIndependenceTest` 在同一個 `AtomicDoAll` 裡一次讀取兩個清單，和兩樣本檢定的做法相同。
- `BartlettTest`、`KruskalWallis`、`FriedmanTest` 與其他卡方類檢定，在各組完全一致時不再以 `cephes: parameter out of bounds` panic：Bartlett 統計量會因捨入誤差略小於 0（兩組相同），`KruskalWallis` 或 `FriedmanTest` 遇到全部數值相同時統計量為 0。現在會回傳結果，p 值為 1；自由度不是正數時，p 值為 NaN 而不是 panic。原本沒有 panic 的結果不變。
- `PCA` 與 `KNNRegress` 的目標值遇到 `NaN` 或 `±Inf` 會回傳錯誤。過去它們只檢查儲存格能否轉成數字，所以 `PCA` 會回傳 `NaN` 特徵值、`KNNRegress` 會預測出 `NaN` 或 `±Inf`，而且沒有錯誤。原本就回傳錯誤的呼叫不變；原本回傳 `NaN` 結果的呼叫現在改回傳錯誤。
- **BREAKING（錯誤訊息文字）**：指出儲存格位置的錯誤訊息，現在改從 0 算起，與 `Get(i)` 收的數字一致，並明寫是索引：`data contains a non-finite value at row index 2: NaN`。`v0.3.4` 的假說檢定從 1 算（`row 3`），容易被當成 DataTable 自己的編號，與它差一。範圍包含假說檢定、回歸、相關、偏度、峰度與動差，以及回歸、GLM 與因素分析的 `(index N)`、`row N` 檢查；依傳入順序命名的群組、儲存格與受試者（`group 2`、`subject 3`）仍從 1 算。數值與錯誤型別不變，只有文字改變，比對舊文字的程式需要更新。
- `CutTreeByK` 與 `CutTreeByHeight` 會拒絕葉子或合併步驟被併入超過一次的樹（手動編輯 `state.json` 可能產生）。過去它們會在要求 1 群時回傳 2 群，且沒有錯誤。
- 檢定結果現在可以一起處理：共用的結果型別匯出為 `TestResult`，所有檢定結果都符合新的 `HypothesisTestResult` 介面（`Base() *TestResult`），同一個函式或同一個 slice 可以接 t 檢定、Mann-Whitney U 與卡方檢定的結果。`stats` 的每個結果型別都有版面一致的 `String()` 與 `Show()`，`fmt.Println(res)` 會印出可讀的結果，不再是 Go 的原始結構；`fmt.Fprintln(w, res)` 可寫到任何 writer。`ChiSquareTestResult.Show()` 與 `FactorAnalysisResult.Show()` 改印這個版面。
- **BREAKING**：`FactorAnalysis` 在 `Rotation.Method: FactorRotationOblimin` 且 `Rotation.Restarts` 大於 1 時，現在真的會做參數所描述的搜尋。Oblimin 過去每一輪都自己建一個單位矩陣當起點，忽略傳進來的起點，所以 `Restarts: 20` 是把同一份計算跑 20 次再回傳第一個結果，實測 20 個起點花 202 ms 回傳和 9.9 ms 一模一樣的答案。現在它和其他九個方法一樣從每個起點各旋轉一次，而且 `Delta: 0` 時與同樣 `Restarts` 的 Quartimin 結果逐位元相同，因為兩者本來就是同一個準則。在測試套件的 12 個生成資料集上，載荷最多移動 1.2e-5；在過因子的模型上（用三因子資料抽四個因子），額外的起點會找到單位矩陣起點到不了的另一個盆地，準則值低 47 倍。那個解的因子相關也高達 0.905：`Restarts` 要的是最低的準則值，不保證那個解就是你要的，這對任何多起點的斜交旋轉都成立。預設的 `Restarts: 1` 不變。SPSS 的 Direct Oblimin 是單起點，`Restarts: 1` 仍然是。
- 新增 `stats.VarianceRatio(returns, qs...)`，計算 Lo-MacKinlay 變異數比 VR(q)，同時給同質變異與異質變異穩健的 z 值；`stats.ChowDenning` 一次檢驗多個持有期。新增 `stats.KaplanMeier(durations, observed)`，從含有「還沒結束」（截尾）的持續時間估計存活曲線，附 Greenwood 標準誤與中位數，結果與 R 的 `survival::survfit` 一致（[#443](https://github.com/HazelnutParadise/insyra/issues/443)）。
- 新增 `stats.LinearRegressionWithOptions(opts, y, xs...)`，可回報穩健或 HAC 標準誤：`LinearRegressionOptions{CovType, Lag}` 可選 `CovHC0`～`CovHC3`（R `sandwich::vcovHC` 的異質變異穩健估計），或 `CovHAC`（Bartlett 權重、不做小樣本校正的 Newey-West，同 `sandwich::NeweyWest(prewhite = FALSE, adjust = FALSE)`）。lag 給負數（寫成 `stats.AutoLag`）時用 `NeweyWestLag(n)`。t 值與 p 值用自由度 n-k 的 t 分配，結果已與 statsmodels 比對。`LinearRegression` 的數字與過去完全相同；結果多了 `CovType` 與 `Lag` 欄位，`String()` 也會印出（[#478](https://github.com/HazelnutParadise/insyra/issues/478)）。
- 新增 `stats.AutoLag`（`-1`），代表「請用 `NeweyWestLag(n)`」的 lag；傳給 `NeweyWestMeanTest` 或 `LinearRegressionOptions.Lag`，就由 insyra 依樣本數選 lag，文件範例也改用它。`NeweyWestMeanTest` 與 `LinearRegressionOptions.Lag` 仍是負數自動、0 以上照用，既有呼叫不受影響。`quant` 的選項讀法相反（0 為自動、負數會回錯），`Docs/stats.md` 把兩種讀法並列說明。
- 新增 `stats.NeweyWestMeanTest(series, lag)`，用 Newey-West（Bartlett 核）標準誤檢定自我相關序列的平均數是否為零，例如重疊持有期的報酬，一般 t 檢定會把 t 值高估。`lag` 為負數（`stats.AutoLag`）時用 `NeweyWestLag(n)`，為 0 時等於以 `n` 為除數的單樣本 t 檢定（[#457](https://github.com/HazelnutParadise/insyra/issues/457)）。

### `csvxl`

- 編碼參數給空字串，或任何大小寫的 `"auto"`，現在都代表自動偵測，跟核心的 CSV 讀取函式一致。以前空字串代表直接當成 UTF-8，`"AUTO"` 則會被當成不支援的編碼而報錯。
- 活頁簿唯一的工作表就算名叫 `__insyra_placeholder__`，`AppendCsvToExcel` 也能取代它。以前它用的暫存工作表名稱剛好撞名，結果舊的儲存格都留下來。
- 函式名稱改用 Go 的拼法，新名稱把每個 CSV 和它的工作表寫在一起，不再用兩個 slice 依位置配對。`CSVToExcel(sheets []SheetFromCSV, output, csvEncoding...)` 與 `AppendCSVToExcel(sheets []SheetFromCSV, existingFile, csvEncoding...)` 接受 `SheetFromCSV{Path, Sheet}`，`Sheet` 留空就用檔名；`CSVDirToExcel`、`ReadCSVToString` 是原本的函式換上新名字。新增 `ExcelToCSV(excelFile, outputDir, options...)` 與 `ExcelDirToCSV`，接受 `ExcelToCSVOptions{Sheets, FileNames, AllowFormulas}`：`FileNames` 以工作表名稱對應 CSV 檔名，鍵不是要轉的工作表時回錯誤，`ExcelDirToCSV` 不接受它；試算表會當成公式執行的文字（以 `=`、`+`、`-`、`@` 開頭且不是數字）預設會在前面加單引號，設定 `AllowFormulas` 才原樣寫出；指定活頁簿沒有的工作表會回錯誤；檔名保留給定的副檔名。兩個 CSV 寫到同一張工作表，或兩張工作表寫到同一個 CSV 檔（不分大小寫，`ExcelDirToCSV` 也跨活頁簿比對），都會在寫出任何東西前回錯誤，`ExcelToCSV` 也會先檢查完所有檔名才寫第一個檔。新增 `SheetNames(excelFile)` 列出活頁簿的工作表。`CsvToExcel`、`AppendCsvToExcel`、`EachCsvToOneExcel`、`ReadCsvToString`、`ExcelToCsv`、`EachExcelToCsv` 改為 Deprecated，簽名、依位置配對的方式與輸出都不變，CLI 一般的 `convert` 仍使用它們。
- 取代工作表時（`AppendCSVToExcel`，或 `DataTable.ToExcel` 搭配 `SheetExistsReplace`），現在會把舊工作表擁有的東西一併從存檔中移除：註解、表格、繪圖，以及只有它們用到的圖片和圖表。以前這些會原封不動留在檔案裡、沒有任何地方參照，所以舊註解文字還在檔案內，新表格也不能沿用舊表格的名稱。驗證方式是讀存檔內容並用 excelize 重新開啟，沒有用 Excel 或 LibreOffice 實際開過。

### `parquet`

- `ReadFrom` 與 `StreamFrom` 從任何 `io.ReaderAt` 讀取 Parquet 檔，`WriteTo` 寫到任何 `io.Writer` 且不會關閉它。這條線的 `StreamFrom` 和 `Stream` 一樣用 channel 傳送批次。
- 新的 `WriteContext` 與 `WriteToContext` 搭配 `WriteOptions{Compression, RowGroupSize}` 指定壓縮方式與 row group 大小，context 結束時停止。`WriteContext` 先寫到同目錄的暫存檔，完整寫完才換上，寫入失敗時舊檔不受影響。`Write(dt, path)` 簽名不變，仍是原地寫入。
- Arrow 讀不了的檔案在每個讀取函式都回傳錯誤，不再 panic；row group 的列數比中繼資料少時，回傳指出哪些 row group 的錯誤，不再靜默少讀。
- `ApplyCCL` 使用 `@` 時每一列拿到自己的值；原本同一批 1,000 列都拿到該批最後一列的值。
- Arrow 依賴從 `github.com/apache/arrow/go/v17` 換成 `github.com/apache/arrow-go/v18` v18.8.0。讀出的表格相同，duration 欄仍讀成 `int64` 計數。
- `ApplyCCL` 改用同目錄裡自己專用名稱的暫存檔，並保留原檔的權限。原本固定寫到 `<path>.tmp`，會覆寫再刪掉使用者自己放在那個名稱的檔案。
- `ApplyCCL` 改寫檔案時，會沿用每一欄原本的壓縮格式，row group 也維持原檔最大的那一組的大小；腳本新增的欄位則採用第一欄的壓縮格式。過去一律寫成未壓縮、每 1,000 列一組：一個 20 萬列、用 Zstd 壓縮的檔案加一欄後，從 1.7 MB 變成 9.3 MB，分成 200 組。也可以像 `WriteContext` 一樣傳入 `WriteOptions` 自行指定這兩項。路徑是符號連結時會寫到連結指向的檔案，並保留檔案原本的權限，與其他寫檔函式一致。
- `FilterWithCCL` 與 `ApplyCCL` 因錯誤提早返回時，會停止讀檔。過去負責讀檔的 goroutine 會卡在送下一批資料，檔案也一直開著，直到 context 結束為止；用 `context.Background()` 的話，就是一直到程式結束。
- `FilterWithCCL` 與 `ApplyCCL` 的結果，現在和先用 `Read` 載入檔案再執行同一個運算式相同。過去它們一次讀 1,000 列、每一批各自計算，所以 `A > AVG(A)` 比的是該批的平均、`SUM(A) > 1000000` 加總的是一批、`# == 0` 會符合每一批的第一列、`A == A.0` 讀到的是該批的第一列，`NEW('c') = CUMSUM(A)` 則把該批整串累計值以文字寫進每個儲存格。聚合函式（`SUM`、`AVG`、`COUNT`、`MIN`、`MAX`、`VAR`、`VARP`、`STDEV`、`STDEVP`）、列索引 `#`、固定列（例如 `A.1500`）現在都對整個檔案計算；內建序列函式（`LAG`、`LEAD`、`DIFF`、`PCT_CHANGE`、`CUMSUM`、`CUMPROD`、`CUMMAX`、`CUMMIN`、`ROLLING_*`）可以跨批次，只保留跨過批次邊界所需的列；其他會讀到目前列以外的部分（`MEDIAN`、你註冊的函式、`A.(# - 1)`、放在其他運算式裡的序列函式），則讀入它所用到的整欄再計算。除非運算式需要整欄，記憶體仍只放一批。`ApplyCCL` 的後一條陳述式現在看得到前面陳述式新增或取代的欄位，`NEW('r') = @` 也會給每一列自己的值。多出來的讀檔次數寫在 `Docs/parquet.md`。
- `ApplyCCL` 現在接受所有欄位型別。沒有被任何陳述式寫入的欄位，會直接用檔案原本的資料寫回、完全不變：各種寬度的整數與浮點數、日期、decimal、list、struct 等；過去只要有 `int64`、`float64`、文字、布林、時間戳以外的欄位，整個呼叫就失敗。被陳述式寫入的欄位在所有新值都放得進去時，會保留原本的型別，現在也包含 `int8` 到 `int32`、無號整數、`float32`、`Date32` 與 `Date64`；放不進去時則和 `Write` 一樣決定型別。型別改由所有值決定，不再只看前 1,000 列，所以一開始是空值的欄位不會再被寫成文字，把小數指定給整數欄時會改成浮點數，不再被截掉。
- `Write` 會把所有儲存格都是 `[]byte` 的欄位存成 Parquet 的 `Binary` 欄位，讀回來還是同樣的位元組。以前它把每格寫成 Go 對切片的文字表示，`A-01` 會被存成 `[65 45 48 49]`，二進位欄位讀進來再寫出去，資料就壞了。`[]byte` 與其他種類的值混在同一欄時，若是合法 UTF-8 就寫成它的文字。這種欄位寫出來的檔案跟以前不同。其他型別在來回一趟後仍會變（`Date32` 變成 timestamp、decimal 變成文字、`Int16` 變成 `Int64`）。

### `pd`

- `FromDataList` 遇到空清單時回傳空 Series，原本回傳錯誤。

### `lp`

- `lp.Solve(model, opts)` 與 `lp.SolveFile(path, opts)` 用純 Go 的 go-milp 求解，回傳 `*Solution` 與錯誤，不需要安裝 GLPK；設 `Engine: lp.EngineGLPK` 則改用已安裝的 `glpsol`（透過 `GLPK_PATH` 或 `PATH` 尋找），不會替你安裝任何東西。`SolveFromFile` 與 `SolveModel` 保留原本的 GLPK 行為（包括自動安裝 GLPK），改為 Deprecated，在 0.4 移除。

### `lpgen`

- `LPModel.WriteLP(w)` 把模型以 LP 格式寫到任何 `io.Writer`。`GenerateLPFile` 簽名不變，寫出的檔案和以前相同。
- `ParseLingo(r)` 與 `ParseLingoFile(path)` 讀取 LINGO 模型，遇到讀不懂的敘述會回傳指出行號的錯誤；底線名稱的舊讀法則是直接略過。新讀法也讀 `@GIN`、`@FREE`、`@BND`。`ParseLingoModel_txt` 與 `ParseLingoModel_str` 行為不變，改為 Deprecated，在 0.4 移除。
- `GenerateLPFile` 先檢查目標函數型別再開檔。目標型別不合法的模型原本會把既有的 LP 檔截成只剩兩行標頭；現在不動既有檔案，也不建立新檔。

### `isr`

- isr 的表格與清單型別匯出為 `isr.DataTable` 與 `isr.DataList`，自己的函式可以直接接收：`func f(t *isr.DataTable)`。`isr.DT`、`isr.DL` 與 `t.DataTable` 不變。
- 新增 `(*isr.DataTable).To`，把表格寫到 `isr.CSV`、`isr.JSON` 或 `isr.Excel`，是 `DT.From` 的對應操作，並回傳 error。沒有 `FilePath` 的 `*isr.CSV` 或 `*isr.JSON` 會把輸出放進 `String` 或 `Bytes`。`CSV.OutputOpts` 新增 `IncludeBOM`；`JSON` 與 `Excel` 新增 `OutputOpts`（`UseColIndex`；`RowNames2FirstCol`、`ColNames2FirstRow` 與 `IfSheetExists`）。和 `ToCSV` 一樣，只有設定 `ColNames2FirstRow` 與 `RowNames2FirstCol` 時才會寫出欄名與列名。JSON 預設以欄名為鍵，設定 `UseColIndex` 才改用欄位索引。
- `DT.From` 與 `Push` 會指出 `Row` 或 `Col` 裡無法使用的型別與鍵：`DT.From(isr.Row{"A": 1, 3.5: 2})` 過去回報 `got unexpected type map[interface {}]interface {}`，現在改為 `isr.Row keys must be all column indices (string or int) or all Name selectors; key 3.5 has type float64`。

### `py`

- 把 Python 結果傳回 Go 的 IPC 伺服器，改成在 `Run…` 呼叫需要時才開啟，同時執行的最後一個呼叫結束時關閉；在 Unix 上關閉時會一併刪掉暫存目錄裡的 socket 檔，v0.3.4 每個執行過 Python 的程式都會留下一個。伺服器開不起來時（例如暫存目錄的路徑太長，放不下 Unix socket），呼叫會在準備環境、啟動 Python 之前回傳包住原因的錯誤，下一次呼叫會再試一次。v0.3.4 只記一則警告，照樣啟動 Python，最後回傳 `exit status 1`。伺服器接受連線失敗時現在會關閉，正在執行的呼叫會回傳錯誤，下一次呼叫會開一個新的伺服器。
- Python 行程失敗時（回傳值之前就崩潰，或因 context 結束被終止），呼叫會回傳這個失敗；context 結束時回傳 `ctx.Err()`。過去約有一半機率回傳 nil 錯誤且沒有結果，因為執行器同時看到失敗與行程結束，隨機挑了其中一個。Python 在行程失敗前已經送回的結果仍會回傳。
- 結果裡的表格與清單現在會解碼成表格與清單。`insyra.Return` 會把 dict、list、tuple 裡的 DataFrame 與 Series 轉成表格或清單的格式；過去 `json.dumps` 會失敗，錯誤是 `Object of type DataFrame is not JSON serializable`。型別是 `*insyra.DataTable`、`*insyra.DataList`、`insyra.IDataTable` 或 `insyra.IDataList` 的 struct 欄位、map 值、slice 或 array 元素，會解碼成表格或清單；過去走 JSON 解碼只會得到空表。這類型別的其他部分，解碼方式和 `encoding/json` 完全相同。結果本身是表格或清單時，名叫 `DataTable` 或 `DataList` 的 struct 欄位跟以前一樣收到整個結果；結果是 dict 時逐欄解碼。
- 空的 DataFrame 回傳時保留欄名；過去欄名會被改成 `a_1`、`b_1` 之類的名稱。
- 結果裡的 NaN 或無限大會解碼成 `math.NaN()`、`math.Inf(1)` 或 `math.Inf(-1)`。Python 的 `json.dumps` 會把它們寫成 `NaN`、`Infinity`、`-Infinity`，過去整個呼叫會回傳 `nil`，也不報錯：DataFrame 裡只要有一個缺值，拿回來的就是 nil。`None` 跟以前一樣解碼成 `nil`。因為其他原因讀不了的結果（例如 float64 放不下的 `10**400`），現在也會回傳錯誤並說明原因，過去同樣會變成 `nil`。Go 端拒收結果或沒有回應時，`insyra.Return` 會在 Python 裡 raise。
- 有一欄叫 `name` 的 pandas DataFrame，回傳後不會有表名。pandas 會把欄位當成屬性回傳，過去表名會變成那一欄印出來的內容。直接設在 DataFrame 本身的名稱一樣會成為表名。
- `RunCodef`、`RunFilef` 與它們的 `Context` 版本的參數，不會再變成 Python 程式碼。過去佔位字元是一個參數接一個參數、對整份程式碼輪流替換，後面的參數替換時，也會改到前面參數插進去的文字：傳入 `$v2` 和 `+__import__('os').system('id')+` 兩段文字，就會執行那個指令。`$v1` 也會吃掉 `$v10` 的開頭。JSON 寫不出來的值（例如 `[]any{文字, math.NaN()}`）會照 Go 的格式原樣寫進程式碼，裡面的文字就成了程式碼。現在佔位字元只在原本的模板裡找一次，依完整編號替換，也只轉換模板用到的參數。寫不成 Python 值的參數，會在 Python 啟動前回傳錯誤，並註明是哪個佔位字元。含有 NaN 或無限大的 `[]float64` 也一樣，過去會寫成 Python 不認得的 `NaN` 或 `+Inf`。
- 在 ARM64 的 Windows 上用 x86-64 的 Python 也能執行了。polars 的 CPU 檢查把機器讀成 ARM64，在每段程式開頭的 `import polars` 拋出 `unknown feature flag`，所以每次執行都失敗。產生的程式現在會在 ARM64 Windows 上的 x86-64 Python 設定 `POLARS_SKIP_CPU_CHECK=1`，已經設過這個變數時不動它。
- `py.Run[T](code)` 直接回傳解碼成 `T` 的 Python 結果，不必先宣告變數。新增 `PipInstallContext` 與 `PipUninstallContext`，context 結束時會停止安裝。所有接受 context 的函式傳入 nil 時回傳錯誤（過去會當機），傳入已結束的 context 時回傳 `ctx.Err()`。`RunCodeWithTimeout` 已棄用，請改用 `RunCodeContext` 搭配 `context.WithTimeout`，兩者行為相同。0.4 移除。

### `accel`

- 手動組出來的 `Buffer` 如果宣告為數值欄，但 `Values` 讀不了（`[]float32`、`[]int`、nil）、`Type` 與 `Values` 不符，或欄位長度不一致，`ExecuteNearestExact` 現在會回傳指出欄位的錯誤。過去它會回傳空結果且沒有錯誤（[#441](https://github.com/HazelnutParadise/insyra/issues/441)）。由 `ProjectDataList`、`ProjectDataTable` 建出的資料集不受影響；`bool`、`string`、`any` 欄位仍回報 `dtype-not-eligible`，不會出錯。

### `plot`

- `CreateWordCloud` 依圖上顯示的文字計數，清單裡有 slice 型儲存格時不再 panic（`hash of unhashable type`）（[#411](https://github.com/HazelnutParadise/insyra/issues/411)）。`1` 和 `"1"` 會算成同一個詞，和它們在圖上看起來一樣。
- 圖表資料含 `NaN` 或無限大時（`NaN` 的長條、空清單的箱型圖摘要、`NaN` 的儀表），`SaveHTML` 與 `SavePNG` 會回傳錯誤。過去它們會寫出一頁沒有圖的內容並回傳 nil（[#409](https://github.com/HazelnutParadise/insyra/issues/409)）。圖表建構函式不變，在這條線上仍會回傳圖表。

### `gplot`

- `CreateBarChart` 與 `CreateHistogram` 收到 `isr.DL` 這類包著 nil 清單的包裝時，會記錄警告並回傳 nil，不再 panic（[#412](https://github.com/HazelnutParadise/insyra/issues/412)）。
- `LinearRegression`、`PoissonRegression`、`LogisticRegression`、`Correlation`、`Covariance`、`ChiSquareGoodnessOfFit`、`ChiSquareIndependenceTest`、`Silhouette`、`KNNClassify`、`KNNRegress`、`Skewness`、`Kurtosis` 收到 nil 清單（直接傳入或包在 `isr.DL` 這類包裝裡）時回傳錯誤。過去會 panic，迴歸函式還是在自己啟動的 goroutine 裡 panic，讓程式結束。`insyra.ProcessData` 把 nil 清單當成不支援的輸入。
- `AppendCsvToExcel` 與 `ToExcel` 搭配 `SheetExistsReplace` 取代工作表時，會保留舊工作表的欄寬與檢視（縮放、格線、凍結或分割窗格），這些在 v0.3.4 會一起消失。隱藏的列與欄、合併範圍、列高、註解、超連結與該工作表自己的定義名稱仍會移除，因為它們會藏住或錯置新資料（[#401](https://github.com/HazelnutParadise/insyra/issues/401)）。
- 新增 `stats.AdjustPValues(pvals, method)`，一次檢驗很多假設時，用 Holm、Bonferroni、Hochberg、Benjamini-Hochberg 或 Benjamini-Yekutieli（`PAdjustHolm`、`PAdjustBonferroni`、`PAdjustHochberg`、`PAdjustBH`、`PAdjustBY`）校正一批 p 值，結果與 R 的 `p.adjust` 一致。`NaN` 維持 `NaN`，也不算進檢驗數（[#444](https://github.com/HazelnutParadise/insyra/issues/444)）。

### `quant`

- 新增 `quant.OBV(close, volume)` 與 `OBVFromReturns(returns, volume)`，回傳葛蘭碧的能量潮（OBV）；後者的漲跌方向取自報酬序列，因此可以用還原權息的報酬搭配原始成交股數。`SignedVolumeRatio(returns, volume, n)` 是 `n` 日內的無單位帶號成交量比，`Divergence(price, indicator, n, cooldown)` 標出價格創 `n` 日新高或新低、指標卻沒有跟上的日子（[#456](https://github.com/HazelnutParadise/insyra/issues/456)）。 `Divergence` 接受前面幾列是 NaN 的指標（例如 `RSI`），窗口碰到 NaN 的日子回傳 0。
- 新增 `quant.AdvanceDeclineRatio`、`AdvanceDeclineLine`、`OverboughtOversold` 與 `NetAdvanceRatio`，用每日上漲與下跌家數計算市場寬幅指標。缺值日輸出 `NaN` 並略過，窗口只算有效日，分母為 0 時回 `NaN`（[#448](https://github.com/HazelnutParadise/insyra/issues/448)）。
- 新增 `quant.RSI(prices, n, opts)`，以 Wilder 或簡單平均（`RSIWilder`、`RSISimple`）計算相對強弱指標，並可選擇缺值的處理方式（`RSIMissingError`、`RSIMissingFlat`、`RSIMissingBreak`）。新增 `quant.SharpeDifferenceTest(a, b, riskFree)`，用 Memmel 修正的 Jobson-Korkie 統計量檢定兩組報酬的 Sharpe 是否相同；有一組沒有變異時回傳 `ErrSharpeUndefined`（[#464](https://github.com/HazelnutParadise/insyra/issues/464)）。
- 新增 `quant.QuantileSort(table, dateCol, signalCol, returnCols, n, opts)`，每天各自把股票依訊號分成 `n` 組，回傳每日每組的平均報酬與檔數，以及最高組減最低組的價差；可設定每天最少檔數、同分處理方式（平均排名或 `pandas.qcut`）、權重欄與是否減掉當天平均（[#458](https://github.com/HazelnutParadise/insyra/issues/458)）。新增 `insyra.IsNilTable`，判斷資料表是否為 nil（含包裝），作法與 `IsNilList` 相同。
- 新增 `quant.ZigZag(prices, threshold)`，在不偷看未來的前提下找出價格序列的轉折點，每個轉折附極值位置與確認位置；`quant.Regimes(pivots, opts)` 把轉折點轉成多頭與空頭期間，可設定 Pagan 與 Sossounov 的最短階段與最短循環。新增 `quant.BacktestSignal(signal, assetReturns, opts)`，把 0～1 的部位訊號轉成淨報酬：訊號落後一列生效，可帶現金報酬與各自的買進、賣出成本（[#442](https://github.com/HazelnutParadise/insyra/issues/442)）。
- 新增 `quant.EventStudy(prices, events, dates, opts)` 與 `quant.SignalBuckets(signal, prices, dates, opts)`，量測事件之後、或依訊號分桶之後，多個持有期的超額報酬。部位至少在訊號後一列才進場，基準可用橫斷面平均、指數或序列自身的平均，t 檢定是對月平均做 Newey-West，`NeweyWestLag` 為 0 時自動、正數照用、負數會回錯。新增 `quant.Bias` 與 `quant.BiasZ`，計算乖離率與只用前幾天波動度調整的乖離率（[#445](https://github.com/HazelnutParadise/insyra/issues/445)、[#446](https://github.com/HazelnutParadise/insyra/issues/446)）。
- 新增 `quant.ToneCounts(texts, lexicon, opts)`，依你提供的字詞表計算每則文字的正面與負面詞數，支援否定詞，中文則用最長優先的子字串比對。新增 `quant.DailyTone(published, counts, tradingDays, opts)`，依交易日加總，週末與收盤後的新聞算到下一個交易日，並提供滾動淨語氣與不含當日的往前 z 分數（[#460](https://github.com/HazelnutParadise/insyra/issues/460)）。該 issue 的 NYT Archive 客戶端尚未包含。
- 新增 `quant.SMA`、`EMA`、`MACD`、`Bollinger`、`TrueRange`、`ATR` 與 `KD`，依 TA-Lib 的定義計算常用技術指標，並以種子固定的合成序列與 TA-Lib 逐筆比對。`KD` 有台灣券商軟體慣用的遞迴平滑（預設）與 TA-Lib STOCH 的移動平均兩種算法（[#474](https://github.com/HazelnutParadise/insyra/issues/474)）。
- `quant.FactorModel` 可多傳一個 `FactorModelOptions{CovType, Lag}`，讓 alpha 的 t 值改用 Newey-West（`stats.CovHAC`）或異質變異穩健標準誤，不再只有 OLS 標準誤；日報酬有自我相關時，OLS 會把 alpha 的 t 值算得太大。`Lag` 為 0 時用 `stats.NeweyWestLag(n)`，正數照用，負數會回錯。不給選項時結果不變，`FactorModelResult` 會記錄 `CovType` 與 `Lag`（[#478](https://github.com/HazelnutParadise/insyra/issues/478)）。
- `quant.ForwardOptions` 新增 `NoNeweyWest`，可在 `EventStudy` 與 `SignalBuckets` 關掉 Newey-West 校正。月序列的 t 檢定此時用 lag 0，變異數除以月數，等同 linearmodels 的 `cov_type="unadjusted"`、`debiased=False`，與除以少一的 `stats.SingleSampleTTest` 不同。`NeweyWestLag` 給負數會回錯，`NoNeweyWest` 與非 0 的 `NeweyWestLag` 並用也會回錯。

## v0.3.4

### Core

- 以另一個錯誤為原因的錯誤，現在用 `%w` 包住原因，而不是把它格式化成文字，所以 `errors.Is` 與 `errors.As` 能認出原因。涵蓋的地方有：偵測不到編碼的 CSV、讀取 Excel 工作表、CCL 的 `MID`、`SUBSTR`、`TONUM`、`VALUE`、`TOSTR`、`TEXT` 與序列、聚合函數、建立 `pd` Series，以及 `lp` 安裝 GLPK。訊息文字不變。
- `StandardScaler`、`MinMaxScaler`、`RobustScaler` 與 `MaxAbsScaler` 實作了 `json.Marshaler` 與 `json.Unmarshaler`，已擬合的 scaler 可以存起來之後再用。用 `json.Unmarshal` 讀回同一型別後，轉換結果與原本完全相同，以欄字母擬合的欄也一樣，NaN 參數讀回仍是 NaN。以前對 scaler 呼叫 `json.Marshal` 只會得到 `{}`。
- CCL 的 `TOSTR` 遇到動詞是標點或非 ASCII 字母、又沒有值可填的格式，會像字母動詞一樣回傳錯誤：v0.3.3 的 `TOSTR(A, '%v %_')` 會把 `1 %!_(MISSING)` 寫進儲存格。
- 內容相同的巢狀陣列與巢狀 slice，不論大小，對 `Count`、`Counter`、`Find` 與 `Replace` 系列方法，以及其他依值比對儲存格的查找，都是同一個值。v0.3.3 把大的巢狀 slice 寫成摘要、陣列則從不這樣做，所以內容超過約 1 KiB 後兩者就對不上了。
- **BREAKING**：`ReadJSON` 與 `ReadJSON_File` 會拒收不合法的 JSON，例如開頭多一個 0 的數字（`{"a":01}`），以前會讀成 1。解碼方式現在和 `encoding/json` 一致：超出 `float64` 範圍的數字（例如 `1e400`）保留成原本的文字，和 `ReadCSV` 的做法相同，以前整次讀取會失敗；字串裡的無效 UTF-8 會換成 U+FFFD，不再原樣保留；錯誤訊息的寫法也和 `encoding/json` 相同。這些來自 `github.com/goccy/go-json` v0.11，所有使用 insyra 的程式都會跟著用到這個版本。
- `ToJSON`、`ToJSON_Bytes` 與 `ToJSON_String` 輸出的內容和 `encoding/json.MarshalIndent` 完全相同。實測唯一的差別是很小的指數改寫成 `1e-7`，不再是 `1e-07`，數值不變。
- JSON 讀寫變快了。一張 10 萬列、5 欄、JSON 約 14 MB 的表，`ToJSON_Bytes` 從 66 ms 降到 45 ms，`ReadJSON` 從 129 ms 降到 90 ms（Apple M3，取 5 次最快）。

### CLI

- 修正 `insyra env import` 在沒有 `--force` 時，只要目標環境有檔案存在但讀不到，就會把非空的環境蓋掉。判斷目標是否為空的檢查把讀不到 `config.json` 當成「空的」，讀不到 `state.json` 與 `history.txt` 也一樣被忽略。檔案不存在仍然視為空，其他讀取失敗現在會停止匯入，並指出哪個環境無法確認。
- `accel` 不再檢查 `--precision`。這個旗標原本是用來選 `accel run` 的精度，v0.3.1 拿掉 `accel run` 之後，就沒有任何程式讀它。還在傳這個旗標的腳本照樣能跑，因為 `accel` 會略過用不到的參數，唯一的差別是 `--precision bogus` 這類無效的值不再報錯。Go 裡的 `accel.Precision` 設定不變。
- 一次性命令不再改動它還原的變數。`state.json` 現在連同 Go 型別儲存每個變數：DataTable 保留欄位順序、欄名、列名與每一格的型別。`insyra load c.csv as t` 之後另外執行 `insyra cols t`，欄位會照檔案的順序列出，不再變成字母序，欄字母在每個命令裡也都指向同一欄。`parsedates` 轉成日期的欄，到了 `resample` 還是日期，CCL 日期相減得到的欄仍是 `time.Duration`，`3.0` 也仍是 `float64`，不會變成 `int64`。`scale fit` 擬合的 scaler 與 `hclust` 的樹也會保存，`scale transform` 與 `cutree` 可以分開執行。以前 scaler 會消失，`cutree` 也不接受讀回來的樹。
- 環境無法保存的變數（例如 `regression` 的結果）會在儲存時印出一行 `warning:`，在 REPL 或腳本中每個變數只提示一次，不再無聲無息地被丟掉或變成 map。產生它的命令照常成功，其他變數照常保存。直接使用 `cli/env` 的 Go 程式可以用新增的 `Manager.SaveVariables` 取得同一份清單，`SaveState` 仍然只在檔案沒寫成時回傳錯誤。
- 先前版本寫入的 `state.json` 仍可讀取，下次儲存時改寫成新格式。含 NaN 值的環境現在可以 `env export`，`env import` 也會完整保留超過 2^53 的整數。

### `ml` 與 `nn`

- 新增 `Tape.Custom(name, inputs, output, vjp)`，把在 tape 外算出來的運算放上 tape：它的輸入會拿到反向規則回傳的梯度。以前在 tape 外算出的張量會讓它的輸入默默拿到零梯度。反向規則可以不是前向的導數，例如替硬門檻宣告一個平滑的替代梯度。`Custom` 會拒絕格式不對的宣告，包括先前記錄的運算已經讀過的輸出。反向規則回傳錯誤，或梯度的數量、型別、形狀不對時，`Backward` 會失敗並指出是哪個運算。（[issue #375](https://github.com/HazelnutParadise/insyra/issues/375)）
- 新增 `Tape.BackwardFrom(output, upstream)`，可以從 tape 上任何運算產生的張量開始反向傳播，並帶入呼叫者給的、形狀相同的上游梯度。（[issue #375](https://github.com/HazelnutParadise/insyra/issues/375)）
- 失敗的 `Backward` 不再讓 `Tape.Grad` 回傳算到一半的梯度：`Tape.Grad` 與 `Parameter.Grad` 都保留上一次成功的結果。
- 新增 `NewEdgeTopology`、`EdgeSum` 與 `Tape.EdgeSum`，處理以邊列表表示的圖：每個節點加總自己收到的加權邊，成本只跟邊數和數值量成正比，不需要 N×N 的稠密矩陣，tape 也會算出邊權重和節點數值的梯度。數值可以是 `[N]` 或帶批次的 `[B, N]`。每個輸出都是所有乘積的精確總和只捨入一次到最近的 float32，所以邊的順序和核心數量都改變不了結果，在每個平台上都一樣。大型圖會用滿所有核心。（[issue #379](https://github.com/HazelnutParadise/insyra/issues/379)）
- `Tanh` 對每個輸入、在每個平台上都回傳正確捨入的值，也就是真正的 `tanh(x)` 只捨入一次到最近的 float32。以前是把 Go 的 `math.Tanh` 捨入成 float32，而它的 float64 結果並非每個平台都一樣（在 arm64 會合併乘加，在 amd64 執行時依 CPU 選擇是否用 FMA，在 s390x 則是組合語言），所以只在那個結果剛好夠準的地方才正確。全部 2^32 個輸入都在 darwin/arm64 上比對過，2^-13 到 9.5 之間的輸入也在 linux/amd64 與 windows/amd64 上比對過。這三個平台上舊的結果原本就正確，所以沒有任何結果改變。
- `Tape.Tanh` 的梯度每一步都捨入成 float32。以前 Go 編譯器在 arm64 上會把 `1 - y*y` 合併成一次乘加，在 amd64 上不會，同一個梯度在兩邊可能差最後一位。在 arm64 上，10 萬個隨機梯度有 24,892 個改變。現在每個平台上位元都相同。

### `stats`

- `PairedTTest`、`SingleSampleWilcoxon`、`PairedWilcoxon`、`MannWhitneyU`、`OneWayANOVA`、`TwoWayANOVA`、`RepeatedMeasuresANOVA`、`KruskalWallis` 與 `FriedmanTest` 改為拒絕 `NaN` 或 `±Inf` 的格子，這正是 `stats` 文件一直對每個數值入口的描述。它們原本只檢查格子能不能轉成數字，所以這些值會進入計算，而且錯誤是 nil：`PairedTTest` 和三種 ANOVA 回傳 `NaN` 的統計量與 p 值，排序類檢定則回傳看起來正常的結果，因為 `NaN` 一樣會被排出名次。`KruskalWallis` 對 `[1, 2, NaN, 4]` 與 `[1, 2, 3, 4]` 回報 H = 0.54、p = 0.46。現在錯誤訊息和其他檢定一致，list 與位置都從 1 起算，例如 `group 2 contains a non-finite value at row 3: NaN`、`cell (A=2, B=1) contains a non-numeric value at row 2: <nil>` 與 `subject 2 contains a non-finite value at condition 2: NaN`，成對與雙樣本檢定則用 `data1` 或 `data2` 指出是哪個 list。以前的訊息是沒有位置的 `invalid numeric value in data1`，或從 0 起算的 `invalid data at group 0 index 2`。`LeveneTest` 與 `BartlettTest` 的組號也改從 1 起算（以前的 `group 1` 指的是第二組），空組、空格與條件數不符的受試者錯誤也一樣。`nil` 的 list（不論是否帶型別）會得到空 list 會得到的錯誤。`OneWayANOVA`、`KruskalWallis` 與 `FriedmanTest` 以前遇到它會讓整個程式結束，因為問題發生在 `recover` 接不到的 goroutine 裡，`TwoWayANOVA`、`RepeatedMeasuresANOVA` 與 `SingleSampleWilcoxon` 則會 panic。全為有限數值的輸入結果不變。CLI 的 `ttest paired`、`anova` 與 `ftest levene|bartlett` 指令會印出新的訊息。
- 本身不是 `*insyra.DataList` 的 list（例如 `isr.DL` 建立的 list）現在會照原本存的樣子讀取。`stats` 以前會用 `NewDataList` 重建這種 list，而它會把一格 slice 拆成好幾個數字，所以 `SingleSampleTTest` 等會轉換輸入的函式把這一格算成多個觀察值，四格的 list 進到 `PairedTTest` 的長度檢查時也變成了五格。現在這一格會被拒絕，和同一格放在 `*insyra.DataList` 裡的結果一樣：`data contains a non-numeric value at row 4: [10 11]`。
- `CutTreeByK` 與 `CutTreeByHeight` 遇到合併、高度與標籤數量對不上，或合併對象既不是葉節點也不是先前合併的樹時，改為回傳指出問題的錯誤。高度比合併少的樹以前會 panic，而 CLI 現在會在一次性指令之間從 `state.json` 還原 `hclust` 的樹，所以手動改過的檔案就足以讓下一次 `cutree` 結束整個程式。

### `csvxl`

- **BREAKING（輸出）**：`AppendCsvToExcel` 改用全新的工作表取代既有的同名工作表，不再就地清空儲存格，所以 v0.3.3 會保留的舊工作表欄寬、檢視與合併範圍，附加之後就不在了，只屬於這張工作表的定義名稱也會一起消失。就地清空只移除值與公式，列與儲存格上的其他東西都還在，結果把資料藏了起來：把五列的 CSV 附加到第 2、3 列被隱藏的工作表上，新的第 2、3 列在 Excel 裡仍是隱藏的，`ExcelToCsv` 讀回時五列只剩三列，舊的註解與超連結也留在新值上。這次以資料能正確讀回為優先，不再保留舊工作表的格式。工作表仍保留原本的位置，原本隱藏的也維持隱藏，作用中的工作表不變，活頁簿其他地方的定義名稱與公式也都保留。名稱只差大小寫的工作表，現在會改用呼叫時給的名稱：附加到 `TARGET` 會把名為 `Target` 的工作表改名，v0.3.3 則維持原名。取代工作表的時間也不再隨舊工作表的內容變長：取代一張有 2 萬個公式的工作表原本要 106 毫秒，現在是 3 毫秒。保留舊工作表格式的做法由 [#401](https://github.com/HazelnutParadise/insyra/issues/401) 追蹤。
- 修正 `AppendCsvToExcel` 在 CSV 讀不到時把工作表清空。它先換掉工作表才讀 CSV，所以檔案不存在、編碼偵測不出來或 CSV 格式錯誤時，存回的活頁簿裡那張工作表是空的，呼叫只回報 `1 files failed to append`。現在會先把每個 CSV 完整讀完，讀不到的 CSV 不會動到它的工作表，原本沒有那張工作表時也不會多出一張空白的，同一次呼叫的其他 CSV 照樣附加。v0.3.3 也有這個問題。

## v0.3.3

### Core

- 修正整數排序失去精度。過去所有整數都經 float64 比較，任何兩個大於 2^53 的 `int64` 都會被視為相等，`Sort`、`SortBy`、`Pivot`、`Describe` 的 min/max 因此排錯。現在整數以精確方式比較，含混合有號／無號與超出 `int64` 的值。
- 修正 `HermiteInterpolation` 用錯基底，完全不滿足它名稱所宣稱的導數條件。現在會通過每個值、符合每個給定導數，並精確重現低次多項式。
- `TryParseTime` 接受常見的無時區版面：`2006-01-02 15:04:05`、`2006-01-02T15:04:05`、`2006-01-02 15:04`，以及 `2006/01/02 15:04:05`、`2006/01/02 15:04`（以 `/` 分隔的日期不接受 `T`），一律視為 UTC。CCL 的日期函式與 `datafetch` 過去會把這些字串當成純文字。
- 修正 `ShowTypes` 超過 26 欄時印成 `A, AA, AB, B, …`，欄位有沒有名稱都一樣；現在與 `Show` 同順序。`ShowRange` 的文件改為與實作一致：end 為排除，負數 end 由尾端往回數且仍為排除（同 Python slice），要顯示到最後請傳 `nil`。
- `DataList`／`DataTable` 的 `Close()` 不再丟棄已在等鎖的操作。Close 停止的是加鎖，不是已排隊的工作。
- 修正 `DataList.ReplaceLast` 在 list 以 `NaN` 結尾時，改掉最後一個 `NaN` 而不是最後一個等於 `oldValue` 的格子（`[5, NaN].ReplaceLast(5, 0)` 得到 `[5, 0]`）。
- 修正 `ReadJSON_File` 把整數字面值讀成 `float64`、而 `ReadJSON` 讀成 `int64` 的不一致；兩者現在走同一條解碼路徑，從檔案讀大整數不失真，內容為單一物件的檔案載入為一列。
- 修正 API 審查找到的多個 `DataTable` 行為：`GetElementByNumberIndex`、`SetRowToColNames`、`SetColToRowNames` 遇到越界索引不再 panic（改設 `Err()`），`GetElementByNumberIndex` 也接受負的欄索引；`Filter*` 沒有符合時回傳可安全呼叫方法的空表；`FilterRows`／`FilterCols` 對參差不齊的表不再 panic；`DropRowsByIndex` 以原始列數換算負索引並忽略重複（過去 `(-1, 0)` 會留下第 0 列、`(1, 1)` 會刪掉兩列）；`Transpose` 保留所有列名（超過原欄數的列名過去會遺失）；`AppendRowsByColIndex` 補欄到指定索引而不是丟掉值；`DropColsContainNumber`／`DropRowsContainNumber` 認得所有內建整數與浮點型別（CSV 推斷出的 `int64` 欄過去不會被刪），`time.Duration` 這類具名數值型別仍會保留；`Mean` 以數值格數作分母。
- `Config.SetLogLevel`、`SetUseColoredOutput`、`SetDontPanic`、`SetDefaultErrHandlingFunc` 改為原子操作，另一個 goroutine 正在寫 log 時呼叫它們不再是 data race。`SetDefaultErrHandlingFunc` 設定的 hook 改由單一 goroutine 依序處理，佇列可容納 1,024 個呼叫，不再每個 warning 開一個 goroutine；遇到佇列已滿的呼叫仍會送到 hook，改由它自己的 goroutine 執行，順序可能不同。import 套件不再印出「Welcome to Insyra」橫幅，改由 CLI REPL 啟動時印出。
- `DetectEncoding` 不再因多位元組字元恰好被 8 KB 取樣邊界切開而誤判 UTF-8 檔案。
- `DataList.IsEqualTo` 與 `IsTheSameAs` 遇到 Go 無法用 `==` 比較的格子（例如含 slice 的 struct）不再 panic，這類格子依型別與內容比較。`ClearNaNs`、`ClearNils`、`ClearNilsAndNaNs`、`ClearNumbers`、`DropAll`、`ClearStrings` 改為單趟過濾，結果不變，不再是平方級原地刪除或每次呼叫開 goroutine；`Update` 在 `Err()` 記的是自己而不是 `ReplaceAtIndex`。`DataTable.FindColsIfContains`／`FindColsIfContainsAll` 不再在每個不含該值的欄上留下警告；`Count` 與 `Clone` 移除 goroutine 分派。`AppendRowsByColName`（連帶 `ReadJSON`／`ReadJSON_File`）新增欄位時依欄名排序，同一份 JSON 不再每次讀出不同的欄序。
- CCL：`NULL`、`TRUE`、`FALSE` 不分大小寫都是關鍵字而非欄位參照；`@` 當值使用時每列拿到自己的 slice（過去每格都顯示最後一列）；序列函數放在另一個序列或聚合函數裡（`LAG(LAG(A,1),1)`、`SUM(LAG(A,1))`）保留整欄；`LAG`／`LEAD`／`ROLLING_*` 的位移或 `REPEAT` 次數為 `NaN`、無限大或超出 `int64`，以及 `REPEAT` 結果長度超出 `int` 時，回錯誤而非 panic，比欄位還長的位移或視窗仍然得到 `nil`；函數註冊表可在另一個 goroutine 求值時安全新增；`engine/ccl.NewMapContext` 依欄名排序，`A`／`B` 指向固定的欄。
- `ToCSV` 會回傳最後一次 flush 的錯誤，寫入成功時也會回傳關閉檔案的錯誤；小表寫到已斷的 pipe 過去會回報成功。
- `DataList` 與 `DataTable` 新增 `PopErr()`，回傳目前的 `Err()` 並清除它；另新增 `SetErr(packageName, funcName, msg, args...)`，以 insyra 自身方法相同的方式在實例上記錄錯誤（寫一筆警告並取代 `Err()`）。`IDataList` 與 `IDataTable` 都包含這兩個方法。
- 全域錯誤緩衝區上限為 `ErrorBufferCapacity`（1536 筆），滿了丟最舊的，不再無限成長。文件定位改為診斷用日誌而非錯誤處理 API；其中九個存取函式（`PopError`、`PopErrorByPackageName`、`PopErrorByFuncName`、`PopErrorAndCallback`、`PeekError`、`GetErrorsByLevel`、`GetErrorsByPackage`、`PopErrorInfo`、`HasErrorAboveLevel`）標為 deprecated，改用 `GetAllErrors`、`PopAllErrors`、`HasError`、`GetErrorCount`、`ClearErrors`。
- 讀取 CSV（`ReadCSV_File` 與 `csvxl` 的 CSV 讀取函式）能解碼 UTF-16／32、ISO-8859 各分部、Windows-1250 到 1258、KOI8-R／U、Shift-JIS、ISO-2022-JP、EUC-JP、EUC-KR、IBM866、Macintosh，以及常見別名；分隔符號與大小寫都不影響（`ISO-8859-1`、`iso8859_1`、`latin1` 視為相同）。過去只有 UTF-8、UTF-16、Big5、GB18030 有解碼器：偵測器判斷正確的 `iso-8859-1` 會被原樣塞進表，儲存格不是有效的 UTF-8；沒有 BOM 的 `utf-16be` 則被當成 little-endian 讀取。其他名稱的讀法維持不變，包括偵測器可能回報、但以原始位元組讀入的 ISO-2022-KR、ISO-2022-CN、IBM424 與 IBM420；`csvxl.ReadCsvToString` 例外（見 `csvxl`）。
- `DetectEncoding` 能辨識 UTF-32 的 BOM；過去會被判成 UTF-16，因為後者的 BOM 是前者的前綴。
- 新增 `DataTable.ToCSVWithOptions` 與 `CSVWriteOptions`，其中 `SanitizeFormulas` 會在開頭為 `=`、`+`、`-`、`@` 的儲存格前加上單引號，避免試算表把它當公式執行。預設關閉，因為它會改變寫出的值；`ToCSV` 的輸出不變。
- 修正 SQLite 上 `ToSQL` 無法附加到名稱含空白的資料表：查詢既有欄位的語句沒有像其他語句一樣為識別字加引號。
- CCL 的錯誤會說清楚是哪個階段、出在哪裡。編譯失敗指出運算式、問題所在的位元組偏移量與該處的文字（`cannot compile "(A B)" at offset 3 (near "B"): expected ')'`）；執行期失敗指出列號（`cannot evaluate "A / B" at row 1: division by zero`），不依賴列的運算式則不報列號，而不是報一個誤導的。`AddColUsingCCL`、`EditColByIndexUsingCCL`、`EditColByNameUsingCCL` 過去兩種失敗共用 `Failed to apply CCL on DataTable after 213.792µs` 這個前綴，既回答不了「是我公式寫錯還是資料有問題」，還把碼錶讀數放在原因該在的位置。`position N` 原本在一個地方是位元組偏移、在另一個地方是 token 序號，現在一律是位元組偏移，並且會對齊到字元邊界，非 ASCII 的運算式才指得到真的位置。兩則訊息會印出 Go 的內部結構（`unexpected token: {5 )}`、`invalid range operands: &{: 0x1d48…}`），現在改印原始文字。`ExecuteCCL` 裡的失敗會指出是哪一條語句，五行的腳本不會再只回 `Failed to execute CCL statement: division by zero` 卻沒說是哪一行。錯誤本身是匯出型別 `engine/ccl` 的 `CompileError` 與 `EvalError`，`ErrorInfo` 也加上 `Cause` 欄位與 `Unwrap`，所以可以用 `errors.As(dt.Err(), &compileErr)` 判斷，不必比對訊息字串。`Err()` 在呼叫成功後回傳 nil 的 `*ErrorInfo`，`Unwrap` 對它回傳 nil，所以同樣的檢查在公式成功時不會 panic。它們仍和其他 `DataTable` 錯誤一樣以警告記錄。
- CCL 不再重複計算答案不會變的東西。`AddColUsingCCL`、`EditColByIndexUsingCCL`、`EditColByNameUsingCCL` 裡沒有用到 `#` 的聚合函數，改為在逐列走訪之前算一次（沒有任何列的表不預先計算），而不是每一列都對整欄的新副本重算一遍：20,000 列的 `A / SUM(A)` 從 2.1 秒變成 1.0 毫秒，z-score `(A - AVG(A)) / STDEV(A)` 從 6.2 秒變成 1.8 毫秒。有用到 `#` 的聚合會依賴當前列，仍然逐列計算。同一條路徑上另外三項：字串只有在首字元可能是日期開頭時才交給日期解析器、`REGEX_MATCH` 每個樣式只編譯一次而不是每列編譯、`ROLLING_*` 整欄只轉換一次而不是每個視窗把每個元素各轉一次。**結果完全沒有改變**；`ROLLING_*` 刻意維持與視窗成正比的計算量，因為改用累加器會產生重算視窗所沒有的誤差飄移。
- `AtomicDoAll` 遇到 nil 的實例（nil 的 `*DataList`、nil 的 `*DataTable` 或 nil 值）會直接略過，不再 panic；同時傳入的其他實例照樣上鎖。
- 修正 `GroupBy` 算出來的是 `Aggregate` 執行當下的父表資料，而不是分組當下的資料。它原本只保留父表欄位的指標，所以兩次呼叫之間對父表的修改會跑進結果（原本是 `3` 的一組加總變成 `102`），而且 `Aggregate` 讀那些資料時沒有上鎖，其他 goroutine 可能同時在寫。現在 `GroupBy` 會複製它分組用的欄位資料。
- 修正依值搜尋、計數、取代與刪除時，找不到以另一種 Go 整數型別儲存的整數。CSV 讀進來的整數是 `int64`，Go 程式裡寫的 `2` 是 `int`，原本用 `==` 比對，`int64` 和 `int` 永遠不相等，所以對 CSV 讀入的表，`Count(2)` 回傳 0，`FindAll(2)`、`FindRowsIfContains(2)` 什麼都找不到，`Replace(2, 0)` 什麼都沒改。現在 `Count`、`FindFirst`、`FindLast`、`FindAll`、各個 `Replace` 方法、`DropAll`、`FindRowsIfContains(All)`、`FindColsIfContains(All)`、`DropRowsContain`、`DropColsContain` 都依數值比對整數，`int8` 到 `int64`、`uint8` 到 `uint64` 一律如此。小數仍然不等於整數，要找 `2.0` 請用 `2.0` 搜尋；小數、字串、布林值、`nil` 與 `NaN` 的比對結果都和原本完全相同。`IsEqualTo` 與 `IsTheSameAs` 仍然連型別一起比較。遇到 Go 無法用 `==` 比較的儲存格（例如含 slice 的 struct），這些查找也不再 panic，而是依型別與內容比對。比對方式改成依要找的值的型別，每次呼叫只挑一次，所以搜尋小數或字串也變快了：一百萬格的 `Count`，小數從 2.3 毫秒降到 1.7 毫秒，字串從 2.8 毫秒降到 2.0 毫秒。
- 修正 CCL 在 amd64 與 arm64 上答案不同的問題：當計數、小數位數或日期位移大到超出 Go 轉成整數或時長的範圍時，這種轉換在 Go 裡沒有定義。在 Linux 與 Windows 上 `MID('abc', 2, 10^300)` 回傳 `""`，Mac 上卻是 `"bc"`；`ROUND(x, NaN)` 在一邊回傳 `NaN`、在另一邊四捨五入成整數；`DATEADD(d, 10^300, 'day')` 在兩邊各算出不同的錯誤日期。現在字元數或位置超過字串結尾，一律代表「到結尾為止」。`NaN` 的計數或小數位數、`DATEADD` 的天／月／年數為無限大或超出 `int64`，或以時分秒計超過約 292 年、日期加減超過 106,751 天、`DAY`／`HOUR`／`MINUTE`／`SECOND` 超過約 292 年的秒數，以及為 `NaN` 或超出 int32 範圍的列範圍邊界，都改為回報錯誤，不再產生繞回的值。一般數值的結果不變，捨去小數的轉換也照舊，例如 `DATEADD` 的小數天數。
- 修正 `Show`（以及其他所有顯示路徑）在 amd64 與 arm64 上印出不同的數字。格式化以 `v == float64(int(v))` 判斷浮點數是否為整數，而該轉換在 Go 裡對超出 `int` 範圍的值沒有定義結果：`2^63` 在 Mac 上印成 `9223372036854775807`，看起來像精確整數但比實際值少 1，在 Linux 與 Windows 上印成 `9.2234e+18`。超出 `int64` 範圍的值現在一律使用指數形式。範圍內的整數不變。
- 修正毫秒的 Unix 時間戳超過 2262-04-11 後讀成錯誤日期。轉換時先把毫秒乘成奈秒，而該乘法在它自己接受的區間內就會溢位 `int64`，`99999999999999` 因此回傳 2216 年而不是 5138 年。
- 修正六個安靜出錯的值。16 位（微秒）時間戳被當成秒，`2023-11-14` 變成 `53872825-06-17`；毫秒的區間也沒接到微秒，中間留了一段會掉到秒的範圍。`CalcColIndex` 在 `int` 範圍的最上緣產生出自己的 `ParseColIndex` 讀不回來的欄位名稱。`Show` 把 `9999.99999` 印成 `10000`——四捨五入到小數第四位後再去掉尾零，看起來就是個整數。`IsNumeric` 與數值讀取路徑對「以數值 kind 為底的具名型別」（`type Celsius float64`、`time.Duration`）看法不同：`IsNumeric` 說這種儲存格是數字，`ToFloat64Safe` 卻拒絕、`ToFloat64` 給 `0`，所以 `Mean` 與 `Sum` 會跳過它，`ToF64Slice` 會把它變成 `0`。兩邊現在都讀出它的值（`time.Duration` 以奈秒讀取）。`DataList.NearestNeighborInterpolation` 對 NaN 的 x 回傳第一個值——與 NaN 的比較永遠為 false，搜尋根本沒動過——而 `LagrangeInterpolation` 與 `NewtonInterpolation` 回傳 NaN 且 `Err()` 沒有任何紀錄；三者現在都回傳 NaN 並把失敗記在 `Err()`，與原本就會拒絕的 `LinearInterpolation`／`QuadraticInterpolation` 一致。
- 修正兩張沒有欄位名稱的表無法垂直合併。`NewDataTable(NewDataList(...))` 建出來的每一欄都沒有名稱，而重複名稱的檢查把空字串當成自己的重複，所以最單純的建構式產生的形狀無法與同類合併。沒有名稱的欄位現在依「在無名欄中的位置」對齊，有名稱的仍然依名稱對齊。
- CCL：超出 `float64` 範圍的數字字面值改為回報錯誤，不再靜默變成 `+Inf`——`strconv.ParseFloat` 的錯誤本來被丟掉了。指數形式現在是合法的字面值：`1e5`、`1.5e-3`、`2E+3` 都能編譯，過去會被拆成數字加識別字然後以 `unexpected token` 失敗，儘管 `VALUE('1e3')` 一直可用、CCL 自己的字串輸出也用指數形式。`e` 後面要有數字才會併入數字，所以名為 `E` 或 `E1` 的欄位不受影響。`TOSTR(1.5, '%d')` 與 `TOSTR(1, '%')` 改為回報格式不符，不再把 Go 自己的抱怨——`%!d(float64=1.5)`、`%!(NOVERB)`——寫進儲存格。原本就出現在值或格式裡、只是看起來像這種抱怨的文字，例如對 `Item (MISSING)` 執行 `TOSTR(A, '%s')`，照原樣寫入。
- `Show` 與其他顯示路徑對實作 `fmt.Stringer` 的 struct 值，改為印出它自己的文字。格式化函式最後會走到一個分支，對 struct 直接印 `<pkg.Type>`，從來沒問過它能不能自己印，所以 Parquet 的十進位欄位每一列都顯示成 `<decimal.Decimal>`。沒有 `String()` 的 struct 仍然顯示型別名稱，nil 指標仍然顯示 `<nil>`，儲存的值不變。
- 不是合法 UTF-8 的字串格子改以十六進位顯示，與 `[]byte` 原本的顯示方式相同，不再是加了引號的亂碼。除了看不懂，加引號的位元組還會讓整列歪掉一欄：`runewidth` 把 NUL 算 0 欄寬、把非法位元組算成一個 `U+FFFD`，而終端機實際畫幾欄由它自己決定，任何寬度計算都不可能算對。現在這種字串顯示成 `00ff41`，超過 20 個位元組會截斷並附上總長度；十六進位是 ASCII，欄位就對得齊。合法的 UTF-8 完全不受影響，中日韓文字與 emoji 也一樣，儲存的值不變。
- `Counter` 遇到 Go 無法當 map key 的格子值不再 panic，各種查找也與它一致。`Counter` 以格子值當 `map[any]int` 的 key，不可雜湊的值會讓程式直接死掉；而 `Count`、`FindAll`、`Replace`、`DropAll`、`IsEqualTo`、`IsTheSameAs` 與 `DataTable` 的查找、刪除方法把它當成跟什麼都不相等，所以 `Count` 對明明存在於兩列的值回報 0。這不需要刻意就會遇到：`ReadSQL` 刻意把二進位欄位保留成 `[]byte`（BLOB、BYTEA、BINARY、VARBINARY、LONGBLOB）。現在這種值以型別與內容識別：`[]byte{0, 255}` 與另一個 `[]byte{0, 255}` 相符，但與 `[]int{0, 255}` 不相符，裡面的整數則和 `Count` 一樣依數值比較。`Counter` 用新的 `insyra.UncomparableKey` 當它的 key，由新的 `insyra.ToMapKey(v)` 產生，供需要自行索引整張 map、或自己用格子值建 map 的呼叫端使用；可比較的值仍以自身當 key，`counter[1]` 寫法不變，印出來的替代 key 會顯示型別與縮短後的內容，例如 `[]uint8(00ff41)`。要查單一值的次數請用 `Count`，它以數值比對整數，而 map 是以 Go 型別當 key。`IsEqualTo` 裡單獨一格的 `NaN` 仍然不等於另一個 `NaN`；在 slice、map 或 struct 裡面，`NaN` 是內容的一部分，所以兩格 `[]any{NaN}` 相等，也彼此相符。`GroupBy`、`Pivot`、`Merge` 與 `Describe` 的 key 維持不變。
- `insyra.Cell(v)` 讓一個值在建構子會攤平它的情況下仍然佔一格。`NewDataList` 刻意攤平切片，好讓建構清單讀起來像建構 pandas Series，而在此之前沒有辦法只讓其中一個引數例外，只能離開建構子改用 `Append`，但那沒辦法跟其他值寫在同一次呼叫裡。`NewDataList(insyra.Cell([]int{1, 2}), 3, "a")` 是三格，第一格以原本的型別持有那個切片。每個接受呼叫端傳入值的入口都接受這個標記並拆掉它，包括 `Append`、`Update`、`InsertAt`、`DataList` 與 `DataTable` 的 `Replace` 和 `Replace…With` 系列、`Shift` 的填補值、`UpdateElement` 與兩個列附加方法，其中也包含本來就不攤平的那些，所以 `Append(Cell(x))` 與 `Append(x)` 意思相同，`Count(Cell(x))` 與 `Count(x)` 也一致。沒有標記的切片仍然攤平。
- `insyra.UncomparableKey` 改以值自己的寫法顯示。它原本印的是編碼內容，對結構而言就是其欄位，所以實作 `fmt.Stringer` 的不可比較值在印出的 `Counter` 結果裡會顯示內部欄位：Parquet `Decimal128` 欄讀成的 `decimal.Decimal` 會印出其 `big.Int` 的正負號與位元組。現在這種值顯示它自己的文字，例如 `decimal.Decimal(-340.0221114815)`，套用同一個截斷上限；沒有 `String()` 的值顯示不變。識別仍然由編碼決定而不是文字，因為 `String` 可能失真，兩個不同的值若文字相同絕不能被併成一組。
- 修正 `SortBy` 處理不存在欄位的方式。過去找不到的索引、名稱或數字是由內部的 `GetCol`、`GetColByName`、`GetColByNumber` 回報，而不是 `SortBy`，多層排序其中一層無效時其他層照樣套用。現在會在移動任何一列之前檢查每一層，找不到欄位時在 `SortBy` 記錄錯誤並保持表格不變。同一個設定同時給了 `ColumnIndex`、`ColumnName`、`ColumnNumber` 其中多個時，仍照文件的優先順序（索引、名稱、數字）排序，並新增一則警告指出被忽略的欄位，`Err()` 維持 nil。沒有指定任何欄位的設定仍依第一欄排序，`Docs/DataTable.md` 現在有寫明：`ColumnNumber` 不為零才算有指定，所以它的零值不會蓋過名稱或索引。
- `engine/biindex`：`BiIndex.Set` 把名稱移到另一個 id 時，現在會釋放該名稱原本的 id，之後的 `Assign` 能再次配發。過去舊 id 被刪掉卻沒有釋放，成了任何 `Assign` 都用不到的空洞。
- CCL：日期加減天數時保留一天中不滿一小時的部分，這正是 `Docs/CCL.md` 一直寫的：數字代表天數。過去天數被截成整小時，`A + 0.0625` 只移動 1 小時而不是 1 小時 30 分，`A + 0.001` 完全不動；現在分別移動 1 小時 30 分與 86.4 秒。整天與半天的移動量與原本相同。
- CCL：以 `>`、`<`、`>=`、`<=` 比較讀不成數字的字串與數字時回報錯誤，這正是 `Docs/CCL.md` 一直寫的（`"hello" > 5 // Error`）。過去回傳 `false`，所以某列的 `A` 是文字時，`A > 5` 看起來就像一個正常的答案。`==` 與 `!=` 對這種組合仍回報不相等，`'5'` 這類數字字串仍以數字比較，`nil` 仍比較為 `false`，兩個不都是數字的字串之間也仍為 `false`。CCL 讀得出來的日期字串（`'2024-01-02'`、RFC3339）與空字串都不算文字：它們照舊比較為 `false`，所以從 CSV 或 Excel 載入的日期欄位用 `A > 0`、含空白格的欄位用 `A > 5`，仍然會產生欄位。
- CCL：`AND()` 與 `OR()` 遇到讀不成布林值的引數時回報錯誤，不再當成 `false`，與 `&&`、`||` 一致——`Docs/CCL.md` 一直說它們是同一件事。過去 `AND('abc', TRUE)` 回答 `false`，和真的比較結果分不出來，而 `'abc' && TRUE` 會指出這個字。轉換讀得懂的值都沒有改變：數字（`0` 為 false）、`nil`，以及 `true`／`yes`／`1`／`false`／`no`／`0`／`''` 這些字串。引數個數也沒有改變（`AND()` 仍是 `true`、`OR()` 仍是 `false`），短路同樣保留：`AND()` 在第一個 false 停下、`OR()` 在第一個 true 停下，被跳過的引數不會被讀取，`AND(B != 0, A / B > 1)` 仍然擋得住除以零。

### CLI

- 會解析到環境目錄之外的環境名稱現在會被拒絕：空白或絕對路徑的名稱，以及經由 `..` 跳出目錄的名稱（`../x`、`a/../../x`）。過去名稱直接接在環境目錄後面，`../x` 會在目錄外建立或刪除資料夾。其他名稱照常可用，包含含空格或非 ASCII 字元的名稱。
- 命令登錄表加上鎖，多個 goroutine（嵌入端）同時註冊命令不再是 data race。
- 修正 `col`、`row`、`movavg`、`expsmooth`、`diff` 找不到或算不出結果時把 nil 存進變數，下一次存檔整個 session panic 的問題；現在回錯誤且不存。
- 含 `NaN` 或 ±Inf 的變數（例如有空白格的 CSV）能完整存檔與還原；過去這種表會變成空字串，這種 list 或一般值則讓整次存檔失敗。其他變數的寫法與過去完全相同，既有的 state 檔也照常能讀。
- `--env`、`--no-color`、`--log-level` 放在 `newdl`、`addcol`、`addrow`、`show` 前面時會生效，不再被當成資料寫進 default 環境。
- `run` 遇到腳本裡的 `env open` 不再開啟互動 REPL；腳本自己呼叫自己超過 16 層會停止。
- `db connect` 寫進 `history.txt`、REPL 歷史與 `env export` 時密碼會被遮罩（URL、`user:pass@`、`password=` 三種形式，`password=` 的值加了引號或含空格時整段遮罩，`=` 前後有空格、單引號內用反斜線跳脫引號，或大括號內寫 `}}` 時也一樣）；history 檔以 0600 建立。
- `accel` 的 Usage 不再宣稱有不存在的 `run` 子命令。
- `help` 現在如實列出 `pca`、`regression`、`count` 的參數：前兩者可以用 `as <var>` 存結果，`count` 的 value 是必填，不再標成選填。`save … sql` 的用法錯誤訊息也跟 Usage 一致，列出 `rownames [true|false]`。
- `count`、`find`、`replace` 現在能對上 CSV 載入的表，以及 one-shot 模式下每次還原的變數裡的整數。原本打的 `2` 是 `int`，存著的是 `int64`，永遠比對不到：`count x 2` 印出 0，`find x 2` 印出 `[]`，`replace x 2 0` 印出 `replaced` 卻什麼都沒改。
- 資料庫連線不再印出 gorm 的查詢日誌。它的預設 logger 在查詢失敗或過慢時會把綁定參數內插進訊息，所以 `WHERE token = ?` 會把 token 印在終端機以及任何收集它的地方。CLI 本來就自己回報錯誤，不會少掉什麼。
- 錯誤訊息會指出是哪個指令、哪個引數，不再直接把標準函式庫的文字丟回來。`ttest single x abc` 過去回答 `strconv.ParseFloat: parsing "abc": invalid syntax`，現在是 `ttest: invalid mu "abc", expected a number`。`ttest`、`ztest`、`anova`、`chisq`、`movavg`、`expsmooth`、`shift`、`diffn`、`pctchange`、`rolling`、`expanding`、`ewm`、`quartile`、`percentile` 與 `fetch` 共 21 處統一成這個寫法。訊息背後的 `strconv` 錯誤仍然保留，`errors.Is(err, strconv.ErrSyntax)` 與過去一樣成立。
- `kmeans` 與 `knn` 的選項鍵不分大小寫——`NSTART 3` 可以用了——未知的選項會列出有哪些。`knn` 的 `weighting` 與 `algorithm` 會先不分大小寫地對照允許的值，不再把字串直接往下傳，所以拼錯會在這裡被擋下，而不是變成函式庫收到一個不認識的模式。
- `clone`、`replace`、`clean`、`fillna`、`count` 能分辨「變數不存在」與「變數存在但型別不對」。五個過去都對明明就在的變數說「variable not found」，害人去找一個根本不存在的拼字錯誤。
- `help` 的表格依最長的指令名稱對齊，`knn_neighbors` 不再把描述擠歪；`read` 與 `env` 補上 Forms 與 Examples——`env` 有九個子指令，過去一個都沒列。`help` 與 Tab 補全改為在鎖保護下讀取指令登錄表，透過新匯出的 `commands.LookupCommand` 與 `commands.SnapshotRegistry`。
- `read sales.csv as x` 會告訴你該怎麼做，不再回答 `unknown option "as"`。`read` 只做預覽，內部自己補了一個別名，使用者再給一個就變成第二個 `as`，抱怨的地方完全不對。現在的訊息是「read only previews a file. Use `load sales.csv as <var>` to keep it」。只有位於別名位置的 `as` 會被拒絕，所以 `read book.xlsx sheet as` 仍可預覽名為 `as` 的工作表。

### `ml` 與 `nn`

- `ml/mltest.RunConformance` 遇到 `Classes()` 回傳 nil 的實作時會判定失敗，不再直接 panic。`ml.Classifier` 是公開介面，本 repo 以外的程式也能實作；測試套件過去對結果直接呼叫 `Len()`，沒有任何檢查。

### `datafetch`

- Google Maps 爬蟲的 `Search` 恢復可用。Google 不再把店家 ID 放在爬蟲讀取的頁面裡，所以過去任何查詢都回傳空結果，也沒有警告。現在改讀 Maps 網頁本身請求的搜尋結果清單，一次請求就取得最多 20 家店和店名，不必再為每家店多開一個頁面，查不到時也會警告。`GoogleMapsStores()` 不再於執行期從 GitHub repo 下載端點與請求標頭，建立時不需要網路，也不會回傳 nil，每個請求都有 30 秒逾時。`GetReviews` 的進度改記在 debug log，不再印到標準輸出，`MaxWaitingInterval_Milliseconds` 剛好是 1000 時不再 panic，`SortBy` 或 `MaxWaitingInterval_Milliseconds` 為零時直接使用預設值，不再警告。`GetReviews` 也恢復可用。Google 會拒絕它原本送出的評論請求，Google 地圖對未登入的訪客也只顯示五則評論，所以現在改讀 Google 搜尋結果中評論視窗的評論頁：每頁 10 則，四種排序都能用，不需要登入（[#249](https://github.com/HazelnutParadise/insyra/issues/249)）。`ReviewDate` 改為 UTC 的發布日期（`YYYY-MM-DD`），`Content` 的 `<br>` 改為換行，`ReviewerState` 與 `ReviewerLevel` 則一律為空，因為新的評論頁不再提供。
- `GoogleMapsStoreReview` 新增四個欄位，由 `GetReviews` 填入，`ToDataTable` 也會產生同名的欄：`ReviewID`、`Language`（評論的語言代碼，例如 `zh-Hant`，只給星等沒寫內容的評論則為空）、`ReviewerReviewCount`（評論者寫過幾則評論）與 `ReviewerPhotoCount`（評論者上傳過幾張相片）。

### `stats`

- `KMeans` 的初始中心改為相異列，與 R 一致。資料含重複列時，單次啟動過去會抽到同一列兩次而回報 "empty cluster"（實測 50 個 seed 中有 44 個失敗）。現在抽到重複才從相異列重抽；本來就相異的抽樣完全不動，既有 seed 的結果逐位不變。
- 接受 `insyra.IDataList` 的函式對 `nil` 或非 `*insyra.DataList` 的實作不再 panic；值會被轉換，`nil` 以一般錯誤回報。
- `FactorAnalysisResult.RotationConverged` 改為回報旋轉實際上有沒有收斂。它過去恆為 `true`：多起點挑出來的候選從來沒帶收斂旗標，`fa.Rotate` 只好用預設值，所以一個在 1e-12 容忍度下只跑一次迭代就停的旋轉也回報收斂。現在旗標描述的是實際回傳的那個解，而多起點之間挑選時優先取收斂的解，全部未收斂時取準則值最佳者並回報 `false`。
- 旋轉沒有收斂時，每次搜尋只記錄一則警告，指出方法、起點數與迭代上限，而且只在回傳的解沒有收斂時才記錄，也就是 `RotationConverged` 為 `false` 的情況。過去 `GPForth` 與 `GPFoblq` 對每個跑到上限的起點都警告一次，多起點搜尋自己建的 Varimax 起點也會警告，所以 `Restarts` 大於 1 的呼叫可能記下好幾則警告、往全域錯誤緩衝區塞同樣多筆，而回傳的解其實已經收斂。那些個別起點現在只在 debug 層級回報。
- `Skewness` 與 `Kurtosis` 改為拒絕無法讀成有限數字的值，不再當成零，這正是 `stats` 文件一直對每個數值入口的描述。空白、文字、`NaN` 或 `Inf` 的格子會回報錯誤，指出 `sample` 與從 1 起算的列號：過去 `[1, nil, 3, 4]` 的 `Skewness` 回傳 `0`，`[1, "x", 3, 4]` 的 `Kurtosis` 回傳 `-1.64`。一如文件所寫，拼成數字的字串同樣會被拒絕。全數值輸入的結果不變。
- `SingleSampleTTest`、`TwoSampleTTest`、`SingleSampleZTest`、`TwoSampleZTest`、`FTestForVarianceEquality`、`BartlettTest`、`LeveneTest` 與 `CalculateMoment` 改為拒絕無法讀成有限數字的格子，錯誤指出序列與從 1 起算的列號，這正是 `stats` 文件一直對每個數值入口的描述。過去檢定的 n 取 list 長度，平均與標準差卻跳過那一格，`[1, 2, nil, 3]` 會得到 t = 4.00、p = 0.028，而不是 `[1, 2, 3]` 的 t = 3.46、p = 0.074，一個空白就把不顯著變成顯著。`CalculateMoment` 則把那一格當成零；原本就會拒絕的 `LeveneTest` 現在也同樣指出列號。`nil` 的 list（不論是否帶型別）在 `SingleSampleTTest`、`SingleSampleZTest` 與 `CalculateMoment` 回傳錯誤，不再 panic。全數值輸入的結果不變；檢定前請用 `ClearNils` 清掉空白。
- `FactorAnalysis` 在 `Rotation.Restarts` 大於 1 時，回傳的載荷已經不是被配適的那個模型。旋轉準則是在有約束的集合上最佳化，正交是 `T'T = I`、斜交是 `diag(T'T) = I`，而梯度投影演算法只有在起點落在該集合上時，才保證每一步都留在上面。`Restarts > 1` 時加入的起點中，有兩個是 Promax 與 Target 的旋轉矩陣，兩者都是斜交的，所以哪個起點在準則值上勝出，就決定了答案還算不算旋轉。以因子分析測試套件的 20 個資料集搭配四種抽取法實測，`Restarts >= 2` 時 `max|L·Φ·L' − Lu·Lu'|` 最大到 0.766，現在最大只有 2.4e-15。現在所有起點都是正交矩陣，依序為單位矩陣、Varimax 解、QR 產生的隨機正交矩陣，也就是 `GPArotation::Random.Start` 對兩個家族給的同一種起點，而且使用前會先驗證。`Restarts` 也改為就是起點總數：過去它只限制隨機起點的數量，三個啟發式起點無條件追加，所以 `Restarts: 2` 實際跑 4 個。**`Restarts >= 2` 的結果會變**，單一載荷最多差 2.04，因為它過去回傳的並不是該模型的旋轉；Oblimin 與 Promax 不受影響，前者本來就忽略起點，後者只跑一個起點。預設的 `Restarts: 1` 結果不變，實測 800 組資料集／抽取法／旋轉法組合逐位相同。
- `FactorAnalysis` 在 `Rotation.Restarts` 大於 1 時，各平台現在從同一組隨機起點開始。過去隨機起點的種子是把未旋轉載荷的每個位元雜湊而來，但抽取結果在不同架構上只能重現到浮點誤差，60 列合成表以 ML 抽 4 個因子，amd64 與 arm64 的載荷在第八位小數就不同。兩邊因此抽到毫不相干的起點，同一個呼叫可能回傳落在不同盆地的解。以因子分析測試套件的 20 個資料集搭配四種抽取法實測，`Restarts >= 2` 的 2400 組組合中有 197 組在 arm64 與 `GOARCH=amd64` 之間不一致，單一載荷最多差 2.1；改用固定種子後，只剩下 8 組在 `Restarts: 1` 本來就差 1.5e-5 以內的組合，那是抽取階段的浮點誤差，不是不同的解。在旋轉準則有多個極小值的資料上，多起點的結果在每個平台都可能和以前不同，因為隨機矩陣換了一組，simplimax 實測單一載荷最多差 2.15。`Restarts: 1` 不變，實測 800 組資料集／抽取法／旋轉法組合逐位相同，嚴格 R 對照套件在改動前後同樣是 1,842 個失敗葉節點、數值完全一致。

### `csvxl`

- 修正 `AppendCsvToExcel` 遇到同名工作表時舊儲存格殘留的問題：`excelize.NewSheet` 對既有名稱只回傳原工作表，所以只有新 CSV 覆蓋到的儲存格被改寫，其餘保留。現在寫入 CSV 前會就地清空既有工作表，舊儲存格與公式都不會殘留，工作表仍保留原本的位置與欄寬等設定，工作簿只有那一張工作表時也能完成。
- 修正 `AppendCsvToExcel`、`ExcelToCsv`、`EachExcelToCsv` 開啟的工作簿從未關閉。
- 錯誤改用 `%w` 包裝底層原因（`errors.Is(err, os.ErrNotExist)` 可用），輸出目錄改以 0755 建立而不是 0777，在 umask 為 0002 的系統上不再給群組寫入權限。
- `ExcelToCsv` 與 `EachExcelToCsv` 拒絕不會落在輸出目錄內的 CSV 檔名，例如由 `../x`、`a/b` 這類工作表名稱組成的檔名，惡意 workbook 過去可藉此截斷輸出目錄外的檔案。只檢查由工作表名稱組成的檔名，所以 `csvNames` 指定的檔名照原樣使用，名為 `.` 或 `..` 的工作表照常轉換。每張工作表先讀完才建立 CSV，最後一次 flush 的寫入錯誤也會回傳。
- `ReadCsvToString` 回傳 UTF-8 內容或錯誤，這正是它的文件一直寫的。遇到沒有解碼器的編碼，不論是指定的名稱或由 `Auto` 偵測到的（ISO-2022-KR、ISO-2022-CN、IBM424、IBM420），過去會回傳檔案的原始位元組（不是有效的 UTF-8）且錯誤為 nil；現在改為回傳列出支援編碼的錯誤。解碼表內或符合舊有子字串規則的名稱（`big5-hkscs`、`utf-8-sig`）讀法不變，大小寫也不影響：過去子字串規則是照使用者輸入的原字串比對，所以 `UTF-8-SIG`、`Utf-8-Sig`、`BIG5-HKSCS`、`X-GBK` 會被拒絕，小寫拼法卻讀得到，而會先把名稱轉小寫的 `ReadCSV_File` 對同一個檔案的判斷又和 `ReadCsvToString` 不一致。「unsupported encoding」錯誤訊息現在也會列出四組子字串家族（`utf8`、`big5`、`gb`、`utf16`），不再只列解碼表正規化後的鍵。`CsvToExcel`、`AppendCsvToExcel`、`EachCsvToOneExcel` 與 `ReadCSV_File` 遇到未知編碼仍不解碼直接讀取。

### `parquet`

- 修正 `ReadColumnOptions.MaxValues` 完全沒有作用。`ReadColumn` 現在先從檔案 metadata 加總所選 row group 的列數，超過上限時在讀取任何資料前就拒絕，這才是該欄位文件寫的行為。
- `Write` 關閉 Parquet writer 失敗時改為回傳錯誤，而不是只寫 log。檔尾在關閉時才寫入，過去關閉失敗會留下無法讀取的檔案卻回傳 `nil`。套件其他地方關閉時的錯誤改經 Insyra 的 logger 而非標準 `log` 套件，`Config.SetLogLevel` 對它們生效。
- 修正 `FilterWithCCL` 只回傳前 1000 列符合的資料。檔案以每批 1000 列讀取，而第一批之後的每一批都被接到結果欄位的複本上而不是欄位本身，所以 2500 列的檔案用每列都成立的條件過濾，回傳的是 1000 列，而且完全沒有錯誤。符合的列現在跨整段串流收集。
- 最後一批之後才發生的讀取錯誤不再被換成部分結果。讀取端的紀錄通道與錯誤通道是一起關閉的，`FilterWithCCL`、`ApplyCCL` 與 `Stream` 的 `select` 可能挑中任何一個，因此讀到一半失敗的檔案可能回傳截斷的表格而 error 為 nil，`ApplyCCL` 更會把截斷的結果覆蓋回原檔。三者現在都先讀錯誤通道再結束。成功的串流呼叫也不再每次都印出 `failed to close file … file already closed`。
- 讀取其他工具寫出的 Parquet 檔時，reader 不認識的欄位型別不再變成一串重複的字。`getVal` 的 fallback 回傳 `arr.String()`，也就是整個 array 的字串形式，忽略列索引，所以那一欄每一列都讀成類似 `[19000 19001 19002]` 的東西，而 `Read` 回傳 nil error、表格的 `Err()` 也是 nil。`parquet.Write` 只寫得出七種 Arrow 型別，所以這個問題在本函式庫自己寫的檔案上看不到，只在讀別人的檔案時發生。現在 `Date32`／`Date64` 讀成 UTC 午夜的 `time.Time`，`Int8`／`Int16` 與四種無號整數讀成同名的 Go 型別，`LargeString` 讀成 `string`，`Decimal128`／`Decimal256` 以檔案自己的未縮放整數與 scale 精確讀成 [go-decimal](https://github.com/TimLai666/go-decimal) 的 `decimal.Decimal`。其餘的 `List`、`Struct`、`Map`、`Time32`、`Time64`、`Duration`、`Interval` 讀成 `nil`，檔案其他欄位照常讀取。`Read`、`Stream` 與 `ReadColumn` 會以警告記錄原因，並寫進回傳表格或 list 的 `Err()`，內容是 `column "tags": unsupported Arrow column type list<item: int64>; its cells were read as nil`；有好幾欄不支援時，每一欄都會記錄警告，`Err()` 留下的是最後一欄。`FilterWithCCL` 對這種欄位同樣拿到 `nil`，但不記錄原因。可以用 `ReadOptions.Columns` 跳過該欄。Arrow 型別為 `Null` 的欄位讀成 `nil`，不記錄原因，因為它本來就只裝著 null。Dictionary 編碼的欄位讀成其值所屬的 Go 型別，不論 reader 直接還原成底層型別，或在檔案保存了 Arrow schema 時交出 Arrow dictionary。
- `decimal.Decimal` 依數值大小排序，不是依數字文字的字典順序，所以含 9.5、10.2、100.0 的欄位會照這個順序排，而不是 10.2、100.0、9.5。與 `time.Time` 一樣，它對 `Mean`、`Sum` 與 `IsNumeric` 而言不是數值。
- Parquet 的 `Binary`、`LargeBinary`、`FixedSizeBinary` 欄改為讀成 `[]byte`。過去每一列都是同一段文字，也就是整個 array 的字串形式，error 還是 nil，所以位元組整個遺失，二進位欄讀回來也跟文字欄沒兩樣。現在 `Read`、`Stream` 與 `ReadColumn` 讀出的每一格都是該列自己的位元組，從 Arrow 的緩衝區複製出來。`Show` 以十六進位顯示這一欄，JSON 匯出寫成 base64。`Write` 仍然把 `[]byte` 格子的欄位寫成字串欄，內容是 Go 對該切片的文字，例如 `[65 45 48 49]`，所以把這種欄寫回檔案保不住它的位元組。

### `mkt`

- 修正 `RFM` 遇到無法讀成數值的金額格子時讓整個程序崩潰的問題，現在跳過該列並以警告指出列號。數值字串仍照常讀成數字。`RFM` 與 `CustomerActivityIndex` 的輸出列依客戶 ID 排序，過去依 Go map 順序輸出、每次執行都不同。
- `RFM` 與 `CustomerActivityIndex` 套用預設 `DateFormat`／`TimeScale` 的提示改為 Debug 等級而非 Info。

### `finance`
- `RoundUnnecessary` 在需要捨入時改為回報錯誤，不再 panic。這個模式的用途是得知結果放不進指定的小數位數，而它過去是用中止程序來達成：`NPV(0.03, []{0, 1}, Options{Scale: 2, Mode: RoundUnnecessary})` 會 panic。現在回傳帶著精確值的錯誤。其他捨入模式不變。

### `lpgen`

- LINGO 解析器遇到括號順序顛倒的宣告（`@BIN)X(;`）不再 panic。過去它取第一個 `(` 與第一個 `)` 而不檢查誰在前面，切片邊界反過來就會當掉；現在這種宣告會像其他讀不懂的行一樣被略過，模型的其餘部分照常解析。

### `lp`

- `SolveFromFile` 與 `SolveModel` 回傳的附加資訊表列順序固定為 Status、Execution Time、Warnings、Full Output、Iterations、Nodes，過去依 Go map 順序每次不同。
- GLPK 下載、解壓或編譯失敗不再結束程式：失敗以警告記錄，之後呼叫 `SolveModel`／`SolveFromFile` 時透過附加資訊表回報找不到求解器。過去開啟 `Config.SetDontPanic(true)` 讓程式繼續時，Linux 與 macOS 會無止境地重試安裝，Windows 會把工作目錄加進 `PATH`，現在各系統都在第一個失敗就停止安裝。`SolveModel` 兩處建立暫存檔失敗改為記錄警告並回傳 `nil, nil`，也就是過去開啟 `SetDontPanic(true)` 時的回傳值，不再結束程式。
- `SolveFromFile` 收到超過一個 `timeoutSeconds` 時，改在尋找或安裝 GLPK 之前就拒絕，已經寫錯的呼叫不會再觸發安裝。它仍然記錄警告並回傳 `nil, nil`。
- 解壓 GLPK 時建立的目錄權限改為 0o755，不再是 0777，在 umask 為 0002 的系統上不再給群組寫入權限。

### `plot`

- `CreateRadarChart` 未提供 indicators 時不再結束程式，改為記錄警告並回傳沒有 indicators 的圖表，與過去開啟 `Config.SetDontPanic(true)` 時相同。`CreateHeatMap` 日曆模式的 X 型別錯誤或未設 `CalendarOpts` 時，改為記錄警告並回傳 `nil`，不再 panic。
- `nil` 的 `IDataList` 不再讓程式當掉。`CreateBarChart` 與 `CreateLineChart` 會記錄警告、略過 nil 的清單並畫出其餘部分，全部都是 nil 時才回傳 `nil`。`CreateBoxPlot` 以同樣方式略過序列裡 nil 的清單，清單全是 nil 的序列會被拿掉；原本就沒有任何清單的序列照舊保留。`CreateWordCloud` 回傳 `nil`。每個圖表都透過 `AtomicDo` 讀資料，而那會解參考接收者，所以夾在正常清單裡的一個 nil 過去會 panic。
- `SavePNG` 在輸出路徑沒有副檔名時回傳錯誤，不再在快照套件裡 panic，該套件是以副檔名決定圖片格式的。
- `SavePNG` 的線上備援（本機 Chrome／Chromium 渲染失敗時預設仍會使用）現在 60 秒放棄，回應最多讀 64 MiB。過去用的是沒有 timeout 的 `http.Client{}`——伺服器接了連線然後不講話就會永遠等下去——以及對遠端回應無上限的 `io.ReadAll`。

### `isr`

- `DT.From`、`Col`、`Row`、`Push`、`UseDL`、`UseDT` 遇到不支援的型別或讀檔失敗時不再結束程式，改為記錄警告，並回傳過去開啟 `Config.SetDontPanic(true)` 時的值。`DT.From` 讀不到來源或不支援輸入的型別時，回傳包著 `nil` `DataTable` 的 `DT`。`Col`、`Row` 的選擇器型別不支援時，回傳包著 `nil` `DataList` 的 `DL`。加不進去的 `Row` 或 `Col` 會被略過，其餘照常加入，回傳的表格會在 `Err()` 記錄錯誤。
- 修正 `DT.From(map[int]any{...})` 永遠產生空表格。鍵被直接轉成字串，`0` 變成 `"0"`，而 `AppendRowsByColIndex` 要的是 Excel 式的欄位索引，因此每個鍵都被拒絕。現在改用 `Row` 路徑相同的轉換，`0` 就是 A 欄。負數的鍵沒有對應欄位，會被回報。

### `gplot`

- `CreateHistogram` 用零值設定不再 panic：`Bins` 為 0 或負數時採用預設值 10。`CreateLineChart` 與 `CreateStepChart` 遇到無法繪製的序列（例如含 `NaN`）時改為記錄警告並略過該序列，不再 panic。
- 另外四個繪圖呼叫遇到一般的錯誤輸入也不再 panic。`CreateBarChart` 沒有 `XAxis` 時（零值設定就是這樣）改為比照 `plot.CreateBarChart` 把長條編號成 1、2、3……，不再在 gonum 的 `NominalX` 裡當掉。`CreateFunctionPlot` 收到 `nil` 函式時記錄警告並回傳 `nil`。`CreateHeatmapChart` 遇到比第 0 列短的列時記錄警告、指出第一個這樣的列並回傳 `nil`（比較長的列照常繪製，多出來的值會被忽略），`Colors` 為負數時比照 0 採用預設值 20。

### `py`

- IPC 監聽失敗改為記錄警告並讓伺服器保持關閉，不再結束程式，開啟 `Config.SetDontPanic(true)` 時也不再因為監聽器是 nil 而當掉。
- `ipc.WriteMessage` 在寫入任何位元組之前，拒絕超過讀取端上限（256 MiB）的訊息。過去會寫出一個對端會拒絕的長度，超過 4 GiB 時前綴還會被截斷，讓對端之後每一則訊息都解框錯位。
- 建立 Python 環境的目錄權限改為 0o755，不再是 0777，在 umask 為 0002 的系統上不再給群組寫入權限。

## v0.3.2

### Core

- `CSVReadOptions` 新增 opt-in 的 `AllowRaggedRows` 與 `TrimLeadingSpace`，`isr.CSV_inOpts` 也提供對應欄位。Ragged 模式會替短列補空字串，並把多出的 cell 保留在自動命名欄位；零值仍維持嚴格行為。（[issue #198](https://github.com/HazelnutParadise/insyra/issues/198)）
- 修正 CSV 檔案載入會經過「解析、重新序列化、再解析」兩次解析的問題，該流程會無聲丟掉只含單一空欄位的列。`ReadCSV_File` 與 `ReadCSV_String` 現在對任何輸入結果一致。
- 新增與 pandas 相容的 `DataList` 指數加權 reducer（`EWM().Mean/Var/Std`）、滾動 `Cov` 與 `Beta`，以及 `DataTable` 的 `EWMCol` 與日曆週期 `Resample`；`IDataList` 與 `IDataTable` 介面同步公開這些方法。

### CLI

- CSV 專用的 `load` 新增 `ragged true|false` 與 `trimspace true|false`，預設仍為嚴格模式，非 CSV 格式會明確拒絕這些選項。（[issue #198](https://github.com/HazelnutParadise/insyra/issues/198)）
- 新增 `ewm <var> alpha|span|halflife <value> mean|var|std [adjust yes|no] [bias yes|no] [minobs <n>]` 與 `resample <dt> <timecol> weekly|monthly|quarterly|yearly <col>:<op>[:<name>] ...`，並讓 `rolling` 支援需要第二個 DataList 的 `cov <other>` 與 `beta <other>`。原本只有 Go API 才能使用的指數加權、雙序列滾動與日曆週期彙總，現在在 CLI 也能操作。`resample` 的運算子名稱與 `groupby` 相同，時間欄必須是 `time.Time` 值。
- 新增 `quant sharpe|sortino|ir|maxdd|annret|calmar|drawdown|var|cvar|beta|capm|factor|bs|iv`，把整個 `quant` 套件（績效比率、尾端風險、市場曝險、因子歸因與歐式選擇權定價）收進單一命令，一個函式對應一種形式。序列引數是存放每期報酬（或權益曲線）的 DataList 變數；`periods`、`days`、`confidence` 是必填位置引數，因為函式庫本身就拒絕預設這些值，而 `rf`、`mar`、`q` 預設為 0，VaR 方法預設為 `historical`。純量形式會印出 `name=value` 並存成 `float64`；`capm` 與 `bs` 存成一列 DataTable，`factor` 每個因子存一列並另存 `<var>_alpha`，`drawdown` 存成 DataList。函式庫錯誤會原樣回傳，前面加上 `quant <form>:` 前綴。
- `fetch` 新增 `tw` 來源：`fetch tw <code> prices|adjprices <from> <to> [market]`、`fetch tw exrights <from> <to> [market]`、`fetch tw institutional|margin <date> [market]` 與 `fetch tw quotes [market]`，涵蓋 `datafetch.TWStock` 的全部六個方法，`.isr` 腳本因此能取得台股日線、還原股價、除權息參考價表、三大法人買賣超、融資融券餘額與全市場報價表。日期為 `YYYY-MM-DD`，`market` 為 `twse`、`tpex` 或 `auto`（預設）；日期格式錯誤、`from` 晚於 `to`、未知 market 都會在發出請求前被拒絕，函式庫錯誤（包含 `adjprices` 與 `exrights` 對 TPEx 的「不支援」錯誤）則原樣回傳並加上 `fetch tw:` 前綴。請求預設間隔 300 毫秒、重試 2 次，可用新的 `config fetch.tw.interval_ms <毫秒>` 覆寫。既有的 `fetch yahoo` 形式不變。
- 新增 `quant portfolio <returns_dt> minvar|target <r>|maxsharpe [rf <r>] [min <v1,...>] [max <v1,...>]` 與 `quant frontier <returns_dt> <points> [rf <r>] [min <v1,...>] [max <v1,...>]`，接上 `quant.OptimizePortfolio` 與 `quant.EfficientFrontier`。這是 `quant` 第一組吃 DataTable（每欄一個資產的對齊每期報酬）而非單一序列的形式，`.isr` 腳本因此不只能衡量既有部位，還能直接求出配置。`portfolio` 會每個資產印一行 `<asset>=<weight>` 再印一行摘要，並存成 `Asset, Weight` 的 DataTable，另存一列的 `<var>_stats`（`ExpectedReturn`、`Variance`、`Volatility`、`SharpeRatio`、`Iterations`、`Converged`）；`frontier` 每個點存一列，欄位先是上述固定欄，之後每個資產一欄權重。`min`／`max` 是依欄序給的逗號分隔逐資產界，預設為只做多的 `[0, 1]`；長度與欄數不符或含非數值會在呼叫求解器前就拒絕，未收斂則與函式庫一致，回報 `converged=false` 而不是錯誤。
- 修正 `.isr` 腳本吃掉路徑中所有反斜線的問題：`\` 過去被當成萬用跳脫字元，`load C:\Users\me\bars.csv` 會變成開啟 `C:Usersmebars.csv`，Windows 的絕對路徑在腳本中完全不能用。現在反斜線只跳脫引號與反斜線本身。

### `quant`

- 新增 `BlockBootstrap` 與 `PercentileBands`，用於機率式預測。`BlockBootstrap` 對報酬序列做整塊重抽樣（預設 moving block，`BootstrapConfig.Stationary` 可改用區塊長度服從幾何分布的 stationary bootstrap），產生 `Paths` 條模擬報酬序列與從 1.0 起算的複利權益路徑，相同 `Seed` 下結果逐位元相同。`PercentileBands` 在每個時點對所有路徑取指定百分位，使用與 `DataList.Percentile` 相同的 R type-7 分位數。無法讀取、NaN 或 Inf 的輸入值會回傳指出列號的錯誤，不會被當成 0。（[issue #199](https://github.com/HazelnutParadise/insyra/issues/199)）
- 新增 `Beta` 與 `CAPM`，從已對齊的每期報酬序列計算市場曝險。`CAPM` 回傳 beta、每期 alpha、R²、標準誤與觀察數；nil、未對齊、筆數不足、benchmark 變異數為零、無法讀取或非有限值的輸入都會拒絕。
- 新增歷史法與參數法 `ValueAtRisk`／`ConditionalValueAtRisk`，以及 `SortinoRatio`、`CalmarRatio`、`InformationRatio` 與 `DrawdownSeries`，用於尾端風險、下行、相對基準與回撤分析。新的序列輸入遇到無法讀取或非有限值時會拒絕。
- 新增 `FactorModel`，以具名因子對資產超額報酬做多因子歸因，回傳因子曝險、alpha、OLS 推論值、配適統計、殘差與因子名稱查詢。因子欄位照原值使用，只從資產報酬扣除無風險利率。
- 新增 `BlackScholes` 與 `ImpliedVolatility`，支援含連續股利率的歐式選擇權定價、五個 greeks 與由市場價格反推波動率。
- **BREAKING**：`SharpeRatio`、`MaxDrawdown`、`AnnualizedReturn`、`DeflatedSharpeRatio`（其 `trialSharpes`）與 `PBO`（每一欄）改為拒絕無法讀成有限數字的值，不再當成零。這五個是最後還走 `DataList.ToF64Slice` 的入口，那條路徑沒有失敗管道，空白、文字、`NaN` 或 `Inf` 會靜默變成 `0`——把 Sharpe 比率壓低、把最大回撤抹平，沒有錯誤，下游也分辨不出來，而同一個序列丟給 `SortinoRatio` 卻會得到錯誤。錯誤訊息會指出序列（`returns`、`equity`、`trialSharpes`，`PBO` 為 `column <j>`）與從 1 起算的列號，`nil` 序列回傳錯誤而不是 panic，全數值輸入的結果不變。序列有缺口請先清理再呼叫——`PctChange` 產生的欄位開頭那個空白可用 `ClearNils` 去掉。
- 新增 `OptimizePortfolio`、`OptimizePortfolioMoments` 與 `EfficientFrontier`，做均值—變異數投資組合最適化。權重恆為總和 1 且落在各資產的 `MinWeight`／`MaxWeight` 界內，預設為只做多的 `[0, 1]`——要放空必須明確給負的 `MinWeight`。`PortfolioConfig.Objective` 可選最小變異數、指定目標報酬或最大 Sharpe；`EfficientFrontier` 在最小變異數組合的報酬與界內可達最大報酬之間掃出效率前緣。解法是純 Go（加速投影梯度法搭配 bounded simplex 的精確投影），不需要外部最佳化器，代價是只支援總和為 1 與逐項上下界這兩種限制。`OptimizePortfolioMoments` 接受呼叫端自行估計的動差，並拒絕非對稱或非半正定的共變異數矩陣。`PortfolioResult.SharpeRatio` 為每期值。達到 `MaxIterations` 會以 `Converged: false` 回傳當時最佳權重，而不是回傳錯誤。

### `datafetch`

- 新增 `TWStock`，以型別化 `DataTable` 取得 TWSE／TPEx 的日線、三大法人、融資融券與全市場日行情，支援逐月歷史分頁、節流／重試、`Auto` 市場 fallback，以及測試中的 opt-in live 存取。
- 新增 `TWStock.ExRights` 與 `TWStock.DailyPricesAdjusted`。`ExRights` 回傳指定期間的 TWSE 除權除息計算結果表，含交易所自己的 `AdjFactor`（除權息參考價 ÷ 除權息前收盤價），超過一年的區間會自動分頁。`DailyPricesAdjusted` 在 `DailyPrices` 的所有欄位之外，再加上 `AdjFactor` 與向後調整的 `AdjOpen`、`AdjHigh`、`AdjLow`、`AdjClose`，採用與 Yahoo `Adj Close` 相同的慣例，報酬序列不再於除權息日出現假跌幅；`[from, to]` 以外的除權息不納入。兩個方法都只支援 TWSE：櫃買中心沒有可查詢歷史的除權息端點，`TWMarketTPEx` 會回傳明確的 "not supported" 錯誤，而不是空表。

## v0.3.1

### Core

- 新增 `CSVReadOptions`，以及 `ReadCSV_FileWithOptions`、`ReadCSV_StringWithOptions`。將 `RawStrings` 設為 true 時，每個 cell 都保留原始字串、跳過欄位級型別推斷，股票代號這類值不會再掉開頭的 0，空白 cell 也維持 `""` 而不是變成 NaN。`ReadCSV_File` 與 `ReadCSV_String` 的簽名和行為維持不變。
- 新增 fitted `SimpleImputer`，支援平均數、中位數、眾數與常數替代值。它會記住訓練表格的替代值並套用到後續表格，可在避免資料洩漏的前處理 pipeline 中使用；既有的 in-place `FillWith*` 方法維持不變。 它刻意沒有 `InverseTransform`，也不是 `insyra.Scaler`：補值無法還原，而一個永遠失敗的方法會讓型別斷言誤以為該能力存在。
- 新增 `Config.SetAcceleration` 與 `Config.GetAccelerationEnabled`。加速預設開啟，程式控制項會控管裝置呼叫點；`INSYRA_ACCEL_DISABLE_WGPU=1` 仍是部署環境的覆寫開關，兩者同時設定時由環境變數優先。
- CCL 欄位運算式現在把遞迴深度放在線上 evaluation stack 傳遞，不再讓每個 AST 節點都做 goroutine ID 與 `sync.Map` 記帳。在 Apple M3 上，issue #191 的運算式實測在 10k 列快 4.5 倍、100k 列快 6.0 倍，結果、上限與錯誤訊息維持不變。

### `isr`

- `CSV_inOpts` 新增 `RawStrings` 欄位，由 `DT.From` 傳遞給讀取端。

### `accel`

- 大型 `nn` 二維 float32 MatMul 現在透過 `accel.DeviceMatMul` 預設使用裝置。它保留每個輸出沿 `k` 的序列累加順序，透過既有 `accel` report 記錄 fallback 理由，沒有裝置或後端失敗時仍由精確的 CPU 路徑作答。設定 `INSYRA_ACCEL_DISABLE_WGPU=1` 可關閉後端。
- wgpu 後端升級到 v0.30.35，修掉上游 Metal 的 checkptr 崩潰——`go test -race` 現在能完整覆蓋裝置路徑而不再跳過，race guard 已全部移除。升級前後裝置數值 parity 不變。
- `accel` 現在會在真實硬體上執行。`ExecuteDataList`、`ExecuteDataTable`、`ExecuteProjectedDataset` 會在 `ExecutionResult.Reductions` 回傳每個欄位算出來的值，並附上實測的 `Transfer`、`Dispatch`、`Readback` 時間與 `BytesUploaded`。
- `accel.Session` 現在可以並發使用。所有公開方法都在 session 鎖後序列化，多個 goroutine 可以共用同一個 session；先前並發呼叫 `ExecuteDataList` 會在快取與 report 狀態上產生資料競爭。裝置提交也在行程層級序列化，因為所有 session 共用同一個 GPU handle。
- 新增 `accel.Default()`，這是整個行程共用、第一次取用時才建立的 session。探測只會執行一次，常駐快取跨運算共用，對它呼叫 `Close` 不會有作用，因為沒有任何呼叫端擁有它的生命週期。單純 import 這個套件仍然不會開啟任何裝置。
- 新增 `accel.OpSquaredDistance` 與 `Session.ExecuteDistances`，在 GPU 上計算每一列到各查詢點的平方歐氏距離，並提供 `accel.SquaredDistancesCPU` 作為參考實作。裝置結果會在執行平台上驗證與參考實作位元一致。
- 新增 `accel.OpNearestQuery` 與 `Session.ExecuteNearestQuery`、`accel.NearestQueryCPU`，回報每一列最接近的查詢點及其平方距離。取最小值的動作在裝置上完成，所以結果大小隨列數成長而非隨「列×查詢點」成長——在 Apple M3 上、64 個查詢點時比 CPU 快 13.6 倍。距離相同時取索引較小者。
- `accel` 現在包含在 `allpkgs` 裡，照標準方式 `go get .../allpkgs` 安裝就會自動註冊 GPU 後端。註冊是惰性的——在開啟 accel session 之前不會探測任何裝置。
- GPU 執行改為內建。後端是純 Go 的 WebGPU 實作（[gogpu/wgpu](https://github.com/gogpu/wgpu)），在 `accel` 初始化時自行註冊，不需要額外安裝、也不需要為了 side effect 而 import。可在 `CGO_ENABLED=0` 下建置，macOS 走 Metal、Linux 與 Windows 走 Vulkan、Windows 也可走 DirectX 12。沒有 import `accel` 的程式不會編譯到它，但 gogpu 的 module 會出現在 `go list -m all`。設定 `INSYRA_ACCEL_DISABLE_WGPU=1` 可以不改程式就關掉。
- **破壞性變更：** `BackendAllocator`、`RegisterBackendAllocator`、`AllocationRecord`、`AllocatorKind` 由 `BackendExecutor`、`RegisterBackendExecutor`、`ExecuteRequest`/`ExecuteResponse`、`ExecutorKind` 取代。舊的介面既不能帶入運算，也無法回傳值，真實後端無法實作。
- **破壞性變更：** `ExecutionResult.Allocator` 與 `ExecutionResult.AllocatorKind` 改名為 `Executor` 與 `ExecutorKind`；`ExecutionResult.BytesMoved` 移除。該欄位是用固定的後端常數推算而非實測，改由 `BytesUploaded` 與三個實測時間取代。
- GPU 執行需要明確指定精度。WGSL 沒有 `f64`，Apple GPU 也沒有雙精度硬體，因此除非把 `WorkloadEstimate.Precision` 設為 `accel.PrecisionFloat32`，否則不會把 `float64` 欄位降精度。未指定時會回退到 CPU，理由為 `precision-not-accepted`。
- 新增 fallback 理由 `no-backend-executor`、`precision-not-accepted`、`dtype-not-eligible`、`shader-compile-failed`、`buffer-too-large`、`readback-timeout`、`execution-failed`。
- 後端回報的 CPU／軟體 adapter 一律不視為加速裝置，因此沒有 GPU 驅動的機器會回退到 CPU，而不是跑軟體直譯器還宣稱是加速。
- 投影欄位便宜很多。資料集指紋不再把每個值轉成文字再 hash，`projectValues` 也不再對每個元素走 `reflect`（舊路徑每個值都會在 heap 上配一次記憶體）。4 Mi 的 `float64` 欄位：投影從 4,194,308 次配置降到 4 次、`ProjectDataList` 從 357 ms 降到 43 ms、端到端的 GPU 欄位加總從 354 ms 降到 48 ms。指紋只存在於單一 session，所以 hash 值改變不會被外部觀察到。
- `ExecuteDistances` 和 `ExecuteNearestQuery` 現在不論裝置有沒有執行都會回傳結果。之前沒有 GPU 的機器只會拿到 `Accelerated: false`、理由 `no-accelerator` 和一個空 slice，等於每個呼叫端都得自己發現並改叫 CPU 版本。裝置存在但執行失敗、逾時或超過緩衝區上限時也一樣。`Accelerated` 和 `FallbackReason` 仍然照實回報工作跑在哪裡，可觀察性沒有降低。因為請求本身被拒絕的情況——`precision-not-accepted`、`dtype-not-eligible`、`workload-unsupported`——仍然不回傳結果，因為在 CPU 上算出來的正好是呼叫端拒絕的東西。strict GPU 模式仍然回傳錯誤而不是 CPU 結果。
- 新增 `accel.OpNearestShortlist` 和 `Session.ExecuteNearestExact`，回傳每列最近的 M 個查詢點，值是精確的 `float64`，但大部分計算仍然跑在 GPU 上。裝置用單精度排序，每列回傳一份候選清單，外加它捨棄掉的最好候選的距離；主機把這份清單用 `float64` 重算後決定，遇到單精度分不出勝負的列就改用全部查詢點重算。結果與 `accel.NearestExactCPU` 完全相同，所以不需要降精度的 opt-in。在 Apple M3 上以 200,000 列對照吃滿八核的主機實測：16 維 2.5 倍、64 維 3.4 倍（都是 1024 個查詢點），而每列的距離計算量低於約 2048 次時會比主機慢，此時執行期會拒絕使用裝置並回報 `workload-not-profitable`。`ExactNearestResult.Rechecked` 會回報有多少列走了完整路徑。
- `ExecuteNearestExact` 的主機端現在會用滿所有核心。沒有裝置時的路徑，以及驗證裝置回傳候選清單的那一段，工作量超過門檻時都會切給 `GOMAXPROCS` 個 goroutine，低於門檻則維持單執行緒。200,000 列乘 16 維、1024 個查詢點時，沒有 GPU 的機器會走的那條路徑從 1.575 秒降到 306 毫秒。候選清單也改成以列為主而不是以候選為主的排列，驗證單一列不再需要跨越整個陣列。
- `ExecuteNearestExact` 在有 GPU 的機器上要求九個以上的鄰居時不再 panic。候選清單寬度會被夾到裝置的八個槽位，但判斷仍然索引第 `m-1` 個位置；現在裝置服務不了的請求會直接略過裝置，改由主機作答。單精度距離溢位成無限大時也不再信任那份候選清單——那種情況下排序沒有任何資訊，而邊界檢查會因為錯誤的理由通過。判斷候選清單可否信任的誤差界也放寬了，涵蓋差值本身的捨入，以及平方項小於最小正規 `float32` 的情況。
- **破壞性變更：**移除 `OpSum`、`OpSquaredDistance`、`OpNearestQuery`，連同 `ExecuteDataList`、`ExecuteDataTable`、`ExecuteProjectedDataset`、`ExecuteDistances`、`ExecuteNearestQuery`、`SquaredDistancesCPU`、`NearestQueryCPU`、對應的 WGSL kernel，以及 CLI 的 `accel run <var>`。每一個都拿吃滿所有核心的主機量過而且輸了：欄位加總是 0.7 倍，因為每個元素搬一次值只做一次加法；距離矩陣要讀回的結果隨列數乘查詢點數成長；單精度最近鄰回傳 f32，而它原本要服務的 float64 呼叫端用不了。`ExecuteNearestExact` 取代了最後這個，回傳精確的 float64 答案。被移除的表面從未出現在任何 release。`accel devices`、`accel cache`、`accel plan` 不受影響。
- 大型 `ExecuteNearestExact` 提交現在會切成連續的 16,000 列裝置 chunks，依輸入順序合併，不會因讀回逾時直接失去裝置路徑。精確的 `float64` 主機決策維持不變，16,000 列以下仍走單次提交，`ExecutionResult.Chunks` 會回報實際提交次數。
- `accel` 新增硬式裝置選擇。`INSYRA_ACCEL_DEVICES` 會在探測邊界遮罩裝置，`Config.Devices` 會限制單一 session，兩者取交集；`PreferredDevices` 仍只在交集內做軟性排序。兩者都接受裝置 ID 與從零開始的探測索引，找不到對應裝置的項目會出現在 `Report.UnmatchedDeviceSelectors`。交集為空時，自動模式會以 `device-selection-empty` 回退 CPU，strict 模式則回傳錯誤。
- `accel` 新增 `single`、預設的 `auto` 與 `forced` 三種分派策略，依實測的 32k／8k 列飽和地板跨裝置執行 assignment。每個 assignment 會回報裝置、列範圍、wall time、chunks 與 fallback，單一 assignment 失敗只會讓自己的列回到 CPU。正確性已在單裝置硬體上逐 bit 驗證，多 GPU wall clock 尚未量測。
- 加速現在會在每個 session 第一次使用裝置，以及第一次符合條件的 fallback，各記錄一行 info；每次執行的 placement 會在 debug 層級記錄。輸出由 root `Config` 的 log level 控制。`Config.SetAcceleration` 也會在狀態真正改變時記錄一行 info，中途切換開關的當下就看得到。

### `stats`

- `PCA` 對沒有欄位的表格回傳錯誤而不是 panic。形狀守衛本來就存在，但位置在它要保護的 `mat.NewDense` 呼叫下方 34 行，所以零欄位的表格會用 `mat: zero length in matrix dimension` 讓呼叫端崩潰。

- 線性、多項式、指數與對數回歸結果新增 `Predict`。方法沿用 GLM 的 prediction 簽名，針對新資料回傳 response scale 的點估計，並檢查 predictor 數量與資料列長度。R 的標準誤與 prediction interval 目前仍不在 API 範圍內。
- 新增 `KMeansResult.Assign`，將已 fitted 的中心套用到新觀測值，回傳從 1 開始的中心索引與該中心的平方歐氏距離。
- `PCAResult` 現在會回傳每個欄位 fitted 時使用的中心化與縮放參數，以及訓練資料的 scores，呼叫端可以用同一組 decomposition 投影新觀測值。
- Logistic 與 Poisson 回歸結果現在公開 fitted `Link`，命名與 `GLMResult.Link` 一致，讓呼叫端能在 `stats` 外用線性預測值重現 response prediction。
- 對分群或降維的進入點傳入 nil 表格現在會回傳錯誤而不是 panic。`KMeans`、`DBSCAN`、`Silhouette`、`HierarchicalAgglomerative`、`PCA` 和 `KMeansResult.Assign` 都在驗證之前就解參考表格，所以 nil interface 和 typed nil 兩種都會讓呼叫端崩潰。
- **BREAKING**：無法讀成有限數字的值改為拒絕，不再當成零。影響 `LinearRegression`、`PolynomialRegression`、`ExponentialRegression`、`LogarithmicRegression`、`PoissonRegression`、`GLM`、`Correlation`、`Covariance`、`CorrelationMatrix` 與 `CorrelationAnalysis`，預測變數與目標變數皆同。這些路徑原本把每個值送進一個沒有失敗管道的轉換，缺值、空白或文字會靜默變成 `0`——六筆觀測中的一個空白，就把 Pearson 係數從 0.9992 移到 0.9879，沒有錯誤，下游也分辨不出來。分群、PCA 與 KNN 原本就拒絕，因素分析原本就刪除該筆觀測。錯誤訊息會指出序列與列號，`Docs/stats.md` 也列出每個家族的處理方式。
- 新增 `RidgeRegression` 與 `LassoRegression`，完全採用 scikit-learn 的目標函數——L2 懲罰 `||y − Xβ||² + α·||β||²` 以封閉解求解、L1 懲罰 `(1/2n)·||y − Xβ||² + α·||β||₁` 以座標下降求解——截距不受懲罰、不做標準化，兩者都逐係數對 scikit-learn 驗證通過。Ridge 能處理讓 `LinearRegression` 失敗的共線性預測變數；lasso 把被懲罰淘汰的係數壓到精確的零，且未收斂時如實回報而非隱藏。兩種結果都不帶標準誤、t 值與 p 值，因為古典推論不適用於受懲罰的估計。參照實作是 scikit-learn 而非 R 的 glmnet：glmnet 預設標準化且懲罰縮放不同，同一份資料會算出不同的係數。
- 新增 `WeightedLinearRegression`（WLS）：加權常態方程式搭配精確古典推論——係數、標準誤、t 值、p 值、加權 R² 與預測全部逐欄位對 statsmodels 的 `WLS` 驗證通過。權重必須嚴格為正；零權重直接拒絕而不是猜測排除語意，因為各參照實作對自由度的處理不一致，猜出來的標準誤誰都對不上。
- 自動演算法的 KNN 可使用 GPU：blank import `accel/knnbridge` 後，有利可圖的形狀會走 exact-nearest 裝置運算，答案由 CPU 以 `float64` 重算裁定——裝置結果與暴力搜尋逐 index 相同，已在硬體上驗證，且加速以接線自身的方向實測（100k×32 對全核心：2k 測試列 1.4 倍到 10k 測試列 3.7 倍）。明確指名的演算法、`k > 7`、小形狀與沒有裝置的機器都照舊走 CPU 路徑；不 import 則 `stats` 完全不帶加速器依賴。
- CPU 自動演算法 KNN 現在會先用呼叫端的測試列做固定且可重現的抽樣，再決定是否採用 ball tree。若剪枝仍檢查過多訓練資料，就改走精確暴力搜尋；明確指定的演算法仍會照指定執行，探測只選擇精確搜尋路徑，因此結果維持逐 bit 一致。

### `nn`

- 新增 `Sequential.Fit` 作為確定性的訓練入口，要求明確指定 optimizer 與 loss，使用由 seed 驅動的 `rng.Perm` 小批次順序，支援透過 `Predict` 驗證、callback 進度，以及每個 epoch 一行的 root logger 資訊。Fit 路徑與文件中的 tape 手寫 loop 逐位元重現 loss 序列，排程、提前停止、checkpoint 與 `DataTable` 整合仍不在 v1 範圍內。
- 新增 resize 算子波，範圍由兩個新發佈 checkpoint 的盤點決定：`Resize`（nearest 與 linear、scales 或 sizes、asymmetric 與 pytorch_half_pixel 座標模式）、opset-9 `Upsample`、`Floor`、`InstanceNormalization`，以及 reflect 模式的 `Pad`。FCN-ResNet50（語意分割）與 mosaic-9（快速風格轉換）現在原封不動跑通並與 `onnxruntime` 一致；ConvTranspose 與 TopK 因為沒有目標模型需要而不建。
- 大型二維 float32 MatMul 現在不需 blank import 就會預設使用裝置。只有達到實測 16Mi MAC 地板的乘法會詢問裝置，批次或較小形狀維持位元一致的 CPU 路徑。設定 `INSYRA_ACCEL_DISABLE_WGPU=1` 或呼叫 `nn.RegisterDeviceMatMul(nil)` 可恢復 CPU-only。在 8 核 M3／Metal 上實測，裝置結果在階梯每一階都與 CPU 位元一致，勝幅從地板處的 1.35 倍到 4096 方陣的 52 倍，實測 encoder layer 從全核 CPU 約 0.9 秒降到 234 毫秒。
- autodiff tape 新增 2-D Conv、MaxPool、AveragePool、GlobalAveragePool 與推論模式 BatchNormalization 的 CNN VJP。Grouped Conv、非對稱 padding、stride、pooling denominator 語意，以及 BatchNormalization 三種梯度都通過 finite difference；固定權重 CNN 走一步 Adam 後與 PyTorch 對齊。
- 在 autodiff tape 上新增 `Layer` 與 `Sequential` surface，支援 eager 維度檢查、以 seed 控制的 He 初始化 `Dense`、activation、`Dropout`、`Flatten` 與 `Func` layer。`Predict` 會以結構方式略過 training-only layer，參數命名遵循 torch `nn.Sequential`，並在 LoadWeights 邊界把 torch Linear 的 `[out,in]` SafeTensors 權重轉成 `[in,out]`。Sequential MNIST proof 重現 `0.350281` 與 `0.163855` 的平均 loss，兩個 epoch 後達到 `95.84%` 準確率。
- 完成可訓練的 layer catalog，新增 `Conv2D`、`MaxPool2D`、`AvgPool2D`、`GlobalAvgPool`、訓練／推論語意不同的 `BatchNorm2D`、`LayerNorm` 與 `Embedding`。訓練模式 BatchNorm 對齊 torch 的 biased normalization、unbiased running variance 更新與三項梯度；Embedding 對重複 index 做 scatter-add。Torch state-dict 可載入卷積與 normalization buffers，gated CNN catalog proof 使用文件化的 30,000 筆子集，兩個 epoch 達到 `97.27%` MNIST 測試準確率。
- 新增 deterministic 的 `SaveSafeTensors`、`Sequential.SaveWeights` 與 `Sequential.ExportONNX`。儲存的 state dict 使用 torch 名稱、包含 BatchNorm running statistics，並反轉 Dense transpose；ONNX 匯出支援的推論 layer、略過 Dropout，並依 layer 位置拒絕 Func 或 Embedding。訓練後的 MLP 與 CNN 匯出結果可在 `nn` 精確 round-trip，也能在 `onnxruntime` 以 float32 容差通過。
- 新增融合且取 mean 的 `MSELoss` 與 `BCEWithLogitsLoss` tape 運算、每個參數各自保存狀態的 `SGDMomentum`、`CosineAnnealingLR` 與 global-norm `ClipGradNorm`。六步 BCE 訓練流程在每一步的 learning rate、loss、clip 前 norm 與所有參數都與 PyTorch 對齊。
- 新增與 torch 相容的 `MultiHeadAttention(embed, heads)` 與 `Residual(layers...)` layer。無 mask 的 batch-first self-attention 組合既有 tape 運算，巢狀 layer state 名稱可透過 SafeTensors round-trip 並處理 torch projection transpose，layer 組成的 encoder 走一步 AdamW 後與 PyTorch 對齊。ONNX 匯出會依 layer 位置與 kind 拒絕這兩種 composite layer。

### `ml`

- ONNX 匯出現在能產生獨立 runtime 接受的模型。兩個缺陷之所以存活，是因為 round-trip 驗證需要裝了 `onnxruntime` 的 `python3`，而它在跑過的每一台機器上都被靜默跳過。每個非字串屬性都帶著一個多餘的空字串資料欄，onnxruntime 在執行前就以 invalid graph 拒絕；樹節點又是最深葉子先寫入，而 runtime 把每棵樹在陣列中的第一個節點當作根——於是它走了一個節點就拒絕整個模型。節點現在以根為先寫入，round-trip（線性、logistic、兩種樹、pipeline）已對 onnxruntime 通過。

- `mltest.RunConformance` 現在會使用傳入的訓練標籤。它原本收下之後就丟掉，所以呼叫端傳的值沒有任何東西拿去檢查。對 `Classifier` 而言，現在會驗證 `Classes()` 涵蓋模型 fit 時看過的每一個標籤——類別集合少了一個，就代表模型永遠預測不出那個標籤。

- 新增 `ml.Clusterer`，分群模型實作這個 optional 介面來宣告自己的預測是分組指派而不是量測值，並回報 fit 收斂出幾個群。回歸指標現在會拒絕這種模型。之前 `KMeansModel` 既不是 `Classifier` 也不是連續值預測器，所以沒有東西阻止 `RMSE` 去計算它的群編號，回傳一個算術上正確但沒有意義的數字。

- 在套件外撰寫的指標現在可以宣告自己需要什麼輸入。`ClassLabelMetric` 和 `ProbabilityMetric` 已匯出，呼叫端自己寫的指標可以像內建指標一樣要求類別標籤或機率。之前的路由是靠未匯出的標記介面，所以外部指標永遠無法要求機率——它會靜默收到模型的預測值，`Prediction.Probabilities` 為 nil 且沒有任何錯誤。兩個介面現在會讀取回傳值而不只是偵測方法存在：實作了但回傳 `false`，與完全沒實作等價。`Prediction` 的欄位也記載了各自在什麼情況下會被填入。

- `mltest.RunConformance` 現在會以數值而不只是欄位名稱檢查機率順序。它原本比對欄名與類別名，而一個用自己的 `Classes()` 產生欄名的模型從構造上就一定通過——所以機率值放在錯誤標籤底下的模型，能通過一個正是為了抓這件事而寫的檢查。現在每一列裡 `Predict` 回傳的類別，必須是機率最大的那一欄所對應的類別。改名欄位的檢查也一併加嚴：改名後的名稱是原名的超字串，所以只提到傳入欄位的錯誤訊息也能矇混過關。

- `ROCAUC` 現在會拒絕不屬於任何一個機率類別的真實標籤，而不是把它當成負類。之前它會對自己根本沒理解的資料回報完美的鑑別力（AUC 為 1 且 error 為 nil），而 `LogLoss` 對同樣的輸入是拒絕的。兩個指標不再互相矛盾。

- 新增統一的估計器與轉換器協定，包裝 `stats` 現有 fitted 模型，支援依欄名繫結特徵、可選的機率能力、PCA 轉換、KNN wrapper，以及 `ml/mltest` conformance 檢查。
- 新增 `ml.NewPipeline` 與 `ml.NewColumnTransformer`，可將前處理與模型一起 fitted、限制轉換器只處理指定欄位，並重用 fitted 結果避免前處理資料洩漏。
- 新增可由 seed 重現的 k-fold 與分層切分、每折重新 fit 的估計器交叉驗證，以及分類與迴歸指標，包含 accuracy、log loss、ROC AUC、混淆矩陣、RMSE、MAE 與 R²。
- 新增可重現的 histogram 決策樹分類與迴歸，支援數值欄位分位數分箱、類別子集合切分、學習缺失值路徑、成長界限、類別機率與特徵重要度。
- 強化 `ml` 協定與 pipeline：拒絕未命名特徵欄位和目前無法支援的迴歸 offset，讓 logistic 模型區分類別標籤與機率，讓 fitted pipeline 保留支援的能力與輸入欄位順序，並保留小量級目標值的決策樹迴歸精度。
- 新增不依賴 C 的 ONNX 匯出，支援線性與 logistic 模型、決策樹，以及包含支援的 scaler 和 encoder 的 fitted pipeline。無法支援的模型會在寫入前拒絕，匯出測試在環境具備 `onnxruntime` 時會做獨立 runtime round-trip。
- `Score` 用指定的指標評估已配適的模型，不重新配適。它走的是 `CrossValidate` 同一套相容性檢查與預測組裝，因此需要機率的指標、或需要從回報機率的模型取得類別標籤的指標，兩條路徑得到的服務完全一致。scikit-learn 把預設指標掛在 estimator 類別上，Go 沒有地方掛，所以指標是參數。
- 對套件外自訂的指標為 **BREAKING**：`Metric` 新增 `Direction`，宣告分數越大越好還是越小越好。`CrossValidationResult` 會帶著它，`Better` 依它比較兩個結果。沒有這個宣告，比較兩個平均值的呼叫者有一半機率挑到較差的模型，因為內建指標一半是越大越好、一半是越小越好。回傳可排序分數卻宣告 `NoDirection` 的指標會被拒絕，而不是給它一個預設方向。
- 配適完成的 pipeline 會回報 `TransformedFeatureNames`，也就是所有步驟跑完後最終估計器實際配適的欄位。兩欄輸入、其中一欄編碼成三欄的 pipeline，原本回報兩個特徵名稱與四個重要度，兩者無從對齊。一致性檢查工具現在要求模型的重要度數量與特徵名稱數量相符。
- 新增 `PrecisionMetric`、`RecallMetric` 與 `F1Metric`，支援 macro、micro、weighted 與 binary 平均，並提供 `Precision`、`Recall`、`F1` 直接函式，全部對 scikit-learn 的 `precision_recall_fscore_support` 驗證通過。預設平均是 macro，而不是 scikit-learn 的 binary 配 `pos_label=1`——在任意標籤上那是猜測；binary 平均必須指名正類，因為與 ROC AUC 不同，這些分數會隨正類選擇而改變。從未被預測的類別貢獻 precision 0，與 `zero_division=0` 一致。
- 新增 `FitRidgeRegression` 與 `FitLassoRegression`，把新的 `stats` 估計器包進 estimator 協定，依名稱綁定特徵並通過一致性檢查。
- `GridSearch` 在完全相同的折上交叉驗證各個具名候選估計器——未提供種子時抽取一個並回報在結果上，讓比較公平且可重現——依指標宣告的方向排名、平手保留較早的候選，並回傳以全部資料重新配適的贏家。網格以具名估計器清單的形式直接提供，因為參數網格展開正是本協定刻意不做的 `clone()` 反射。
- 新增隨機森林（`FitRandomForestClassifier`、`FitRandomForestRegressor`）：bootstrap 重抽以列索引 multiset 表達、共用一份特徵編碼，每個分裂限制在隨機特徵子集（分類 √p、迴歸全部 p），以機率平均做預測，重要度為各樹重要度的再正規化平均。樹平行配適，但所有隨機抽取都源自單一種子，未指定時抽一個並回報在模型上——同一種子永遠重現同一座森林。類別在重抽前就從完整目標收集，因此某棵樹的 bootstrap 樣本缺類別也不可能造成機率欄位錯位。
- 新增梯度提升（`FitGradientBoostingRegressor`、`FitGradientBoostingClassifier`）：迴歸以平方損失擬合殘差，二元分類以 logistic 損失搭配 Newton 葉值更新，預設值採 scikit-learn，殘差歸零時提前停止並回報實際輪數。多類別目標明確拒絕並說明限制，不做近似。
- 新增 `FitWeightedLinearRegression(x, y, weights)`。權重只作用於該次配適：`CrossValidate` 沒有權重通道，此限制明文記載而非讓權重與折列靜默錯位。
- `CrossValidateWeighted` 把樣本權重送進配適：每折的估計器收到的權重，是用建構該折的同一份索引子集出來的，對齊由建構保證。`Estimator` 新增選用的 `FitWeighted`——沒提到權重的一切照舊。留出集評分維持不加權，與 scikit-learn 預設一致；沒有 `FitWeighted` 的估計器直接拒絕，不會靜默改用不加權配適。
- ONNX 匯出涵蓋新家族：ridge、lasso 與 WLS 走線性迴歸器路徑，兩種森林與兩種提升以多樹 ensemble 匯出——森林葉值乘 1/T 讓 runtime 的加總等於平均，boosting 把學習率烘進葉權重、先驗作為 base value。二元分類器採用 runtime 的單分數慣例：寫兩類權重時機率完全正確但 label 全部回傳 1，因此雙類 ensemble 每葉只帶一個分數，補數與 0.5 門檻由 runtime 計算。七個家族全部通過獨立 onnxruntime round-trip。
- 決策樹新增 `ExactSplits`：每對相鄰相異數值的中點都是分裂候選——scikit-learn 的 CART 搜尋——與預設的直方圖搜尋並存。分裂準則本來就相同（Gini、變異數），因此 exact 樹對 scikit-learn 做逐預測驗證：分類在探測網格上逐 label 精確、迴歸在單精度容差內。直方圖因 O(MaxBins) 成本維持預設；兩個選項同時設定會被拒絕，ensemble 透過 Tree 選項繼承此選擇。
- 匯出的 logistic 模型改帶兩列係數（skl2onnx 的二元慣例），修正 onnxruntime 下的機率輸出：單列形式仰賴 runtime 在二元路徑套用 LOGISTIC 轉換，而 onnxruntime 不套用——它把原始決策分數當機率回傳，之前 round-trip 只比 label 所以沒人發現。由 `nn` 的雙參照 round-trip 抓到；label 一直是對的。

### `nn`

- 新增帶 seed 的 tape `Dropout` wrapper，使用 inverted scaling，並在 VJP
  透過相同 mask 傳遞梯度。eval 路徑由呼叫端不呼叫 wrapper 來保持 identity。
- 新增與 PyTorch 相容的 decoupled `AdamW` weight decay 與 `StepLR` helper，
  以固定 PyTorch MLP 驗證多步 loss 和參數 parity，並明確驗證它與 coupled L2
  的差異。
- 新增純 Go 的 float32 ONNX 推論，支援聚焦的 MLP operator 家族。模型以 `protowire` 解碼，在載入時驗證，再用具名輸入與輸出執行。格式錯誤會回傳錯誤而不 panic，未支援的 operator 會一次列出。獨立張量 kernel 與固定權重 MLP 都已對 `onnxruntime` 驗證。
- `nn` 現在能以純 Go 讀回 `ml` 匯出的迴歸器、樹 ensemble 與帶前處理的 pipeline，使用 `ai.onnx.ml` 運算子域執行。`BindRegressor` 與 `BindClassifier` 以結構型介面把載入的網路接進 `ml` 協定，依欄名綁定輸入並通過一致性檢查。strict closure 測試以配適模型與 `onnxruntime` 雙重比對二元 logistic 分類器的 label 與機率。
- `nn` 現在支援帶有 NumPy 風格前導 batch 廣播的 N-D batched `MatMul`、LayerNormalization、GELU、reduction 與 shape-control kernel、比較、切片、分割，以及任意 rank 的 Transpose。二維 MatMul 路徑仍保留為快速路徑。
- 新增各 kernel 的單算子 `onnxruntime` parity，以及固定權重 transformer encoder proof，涵蓋雙頭 self-attention、feed-forward GELU、residual connection 與 LayerNormalization。另提供 `INSYRA_NN_REAL_MODEL` 手動 smoke path，執行支援的本機模型並列印輸出 shape。
- 新增純 Go 的 CNN 推論 kernel，支援 2-D Conv、MaxPool、AveragePool、GlobalAveragePool、推論模式 BatchNormalization 與 constant Pad。Conv 支援顯式與自動 padding、strides、dilations、groups、depthwise groups 與選用 bias；pooling 和 Pad 會清楚拒絕不支援的 ONNX 模式。單算子 parity 會列舉屬性組合，固定權重的 MNIST 類 CNN 也已與 `onnxruntime` 完成端到端比對。
- `MatMul`（二維與批次路徑）和 `Conv` 現在會在大型工作負載使用所有 CPU 核心，同時保留每個輸出的序列累加順序，因此結果與序列版本位元完全一致。8 核心 M3 實測中，encoder layer 從 3.35 秒降至約 0.9 秒、MNIST 級 CNN forward 從 526 毫秒降至約 120 毫秒（可重現約 3.8 倍與 4.4 倍，單次最佳達 4.6 倍與 5.4 倍）；小型輸入仍走序列路徑。
- 新增 `LoadSafeTensors(io.Reader)`，會以原生 dtype 精確載入具名的 `F32`、`I64` 與 `BOOL` 張量。格式錯誤、offset、shape、重複名稱與未支援的 dtype 都會回傳包含名稱的錯誤；`__metadata__` 會被接受並忽略。loader 與混合 dtype fixture 已對 Python `safetensors` 做 round-trip 驗證。
- 新增 MLP kernel 的 float32 反向模式 autodiff tape，包含 `MatMul`、支援廣播的 `Add`、`Relu`、`Sigmoid`、`Tanh`、`Gemm` 屬性、融合的 `SoftmaxCrossEntropy`，以及單步 `SGD` 更新。tape 重用推論 kernel，圖執行器維持不變。
- 將 float32 autodiff tape 擴充到 attention 訓練家族：支援廣播還原的批次 `MatMul`、軸向 `Softmax`、`LayerNormalization`、精確式 `Gelu`、`Erf`、`Sqrt`、`Pow`、`ReduceMean`、shape 運算的 VJP，以及每個參數各自保存狀態的 bias-corrected `Adam` 更新。未實作的反向運算會指出運算名稱並拒絕，不會偽造零梯度。
- 新增 `Clip`、`ConstantOfShape` 與執行期計算的 int64 shape/control tensor，補齊 published MobileNetV2 與 MiniLM-L6-v2 checkpoint 所需的執行支援。新增 gated `INSYRA_NN_REAL_MODELS_DIR` 測試，以固定輸入逐元素和 `onnxruntime` 比對兩個模型。
- 新增由資料集 gate 控制的 MNIST 收斂驗證：固定 seed 的 He 初始化 `784 -> 128 -> 10` MLP 以 Adam 訓練每批 128 筆、每個 epoch 重新 shuffle 的 minibatch，在本機 IDX 資料集兩個 epoch 內達到 95% 以上測試準確率，並加入不依賴資料集的二元 micro-convergence 測試。IDX reader 與初始化 helper 維持在測試端，不新增公開 API。
- `LoadSafeTensors` 現在接受 `F16` 與 `BF16` checkpoint，並以位元完全相同的方式拓寬成 f32。ONNX `FLOAT16` 與 `BFLOAT16` initializer 走同一條路徑，Cast 到半精度時會先依儲存格式四捨五入再拓寬回來；圖內運算維持 f32，quantized dtype 仍拒絕載入。
- 新增 detector 所需的 ONNX 推論支援：`LeakyRelu`、`Exp`、`Ceil`、`Round`、`Tile`、`ReduceMin`、批次 `NonMaxSuppression`，以及帶有 validated GraphProto body、子 scope、loop-carried value 與 scan output 的 `Loop`。單算子與 synthetic Loop parity 已對 `onnxruntime` 通過，gated tiny-YOLOv3 的 selection indices 精確相同，boxes 與 scores 通過 f32 容差。

### CLI

- `load <file.csv>` 新增 `infer true|false` 選項，預設 `true`。指定 `infer false` 時所有 cell 都讀為原始字串。JSON 與 Excel 檔案不接受這個選項。
- 修正 `fetch yahoo <ticker> <method>` 會用 usage 訊息拒絕文件上的三個引數形式（例如 `fetch yahoo AAPL quote as q`）：引數數量是在 `as <var>` 被剝掉之後才檢查的。
- 新增 `DataList.ParseDates(layouts ...string)` 與 `DataTable.ParseDatesCols(cols, layouts...)`：字串格子依序嘗試指定的 Go layout（預設為 `ReadSQLOptions.ParseDates` 使用的 ISO 格式）轉成 UTC `time.Time`，既有的 `time.Time` 原樣保留，無法解析的一律變成 `nil`，不會出現半轉換的欄位。`load sql … parsedates` 改走同一個方法，因此解析不了的值現在會變成 `nil`（以前會留成字串），而被 `DType` 指定型別的欄位交給 `DType` 處理。
- 新增 `parsedates <var> [cols <c1,c2>] [layout <go-layout>] [as <var>]`，CSV 載入的日期欄轉換後就能進 `resample`；純量變數（例如 `quant … as s` 的結果）重新載入環境時保留 `float64`／`int64` 型別，不再變成 `json.Number`；`newdl`、`addrow`、`addcol` 在一次性模式下接受負數（`insyra newdl 0.01 -0.004`），因為這三個命令不再解析 flag，請改用 `help newdl` 而非 `--help`。
