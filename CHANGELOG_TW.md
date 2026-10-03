# 變更紀錄

影響 Insyra 使用者的變更，依套件分類，分法與 release note 相同。`## Unreleased` 收錄下一個版本會包含的內容。

v0.3.0 及更早的版本不重複收錄於此，請見 [GitHub Releases](https://github.com/HazelnutParadise/insyra/releases)。

English: [CHANGELOG.md](CHANGELOG.md)

## Unreleased

### Core
- **BREAKING**：`DataList` 的數值轉換不再改到一半才失敗，也不再把讀不出來的格子當成 `0`。`Normalize`、`Standardize`、`ClearOutliers`、`Difference`、`FillNaNWithMean` 會先掃過整份資料；格子既非數值也非 `nil`／`NaN` 時設定 `Err()`（指出列號）並保持所有格子原樣，過去 `[1, "x", 3].Normalize()` 會先把第一格改成 `0` 再回傳 `nil`。`nil` 與 `NaN` 格子原樣保留，且不計入轉換所用的平均、標準差、最小與最大值，所以含空白的 list 呼叫 `ClearOutliers` 現在會檢查每個數值格子，而不是在第一個空白就停住。`Rank`、`ExponentialSmoothing`、`DoubleExponentialSmoothing` 與六個 `*Interpolation` 方法不再經由 `ToF64Slice` 讀值：`Rank` 對 `nil`／`NaN` 給 `NaN` 名次且不佔名次位置，其他非數值格子則失敗（過去 `[3, "b", 1].Rank()` 把 `"b"` 當成 `0` 排第一）；平滑與插值方法要求整份資料為數值，否則失敗。全數值輸入的結果不變。
- `DataTable` 產生的結果不再和原表共用資料：`Data()` 與 `ToMap()` 回傳複本，每個 `Filter*` 的結果也擁有自己的欄位與列名，修改過濾後的表不會再改到原表。`ChangeRowName` 改成另一列已在使用的名字時，會替被改名的列加上後綴，不再悄悄拿走另一列的名字。`DropColsContainNumber`／`DropRowsContainNumber` 現在也會刪除含有具名數值型別（例如 `time.Duration`）或十進位數的欄與列，v0.3.3 會保留它們。`ToCSV` 把 `time.Time` 寫成 RFC 3339，`ParseDates` 讀得回來。`AddColUsingCCL`、`EditColByIndexUsingCCL`、`EditColByNameUsingCCL`、`ExecuteCCL` 攔下 panic 後回傳接收者，不再回傳 nil。
- `SetDefaultErrHandlingFunc` 設定的 hook 遇到佇列（上限 1,024 個呼叫）已滿時會略過那次呼叫，v0.3.3 則會另開一個 goroutine 送達。訊息仍會留在錯誤緩衝區，第一次略過時會透過標準 logger 印出一則警告。
- `DataList.IsEqualTo` 與 `IsTheSameAs` 把兩個 `NaN` 儲存格視為相等，list 現在會等於自己的 clone。v0.3.3 裡，單獨一格的 `NaN` 不等於任何其他 `NaN`。
- CCL：Excel 式參照超過最後一欄（三欄表寫 `E`）回傳錯誤，不再得到整欄 nil。日期相減以秒參與比較與除法，`(A - B) > 0` 不再靜默得到 false。`SUM`、`AVG`、`MEDIAN`、`STDEV`、`STDEVP`、`VAR`、`VARP` 會跳過 `NaN`，與 `MAX`、`MIN` 一致。`REPEAT` 拒絕負數次數，包括 v0.3.3 當成 `0` 的 `-0.5` 這類小數，也拒絕超過 64 MiB 的結果。
- `ToCSV` 與 `ToJSON` 先寫入暫存檔再 rename 到目標位置，寫入失敗時不會留下截斷的檔案。
- 在某個實例的 `AtomicDo` 內呼叫 `AtomicDoAll` 不再與另一個做鏡像操作的 goroutine 死鎖：改成不再加鎖、內聯執行回呼，與巢狀 `AtomicDo` 同一規則。
- **BREAKING**：insyra 不再結束你的程式。`LogFatal` 改為記錄失敗後返回，不再呼叫 `os.Exit(1)`，所以存圖失敗、GLPK 安裝失敗或檔案讀不到都不會讓程式中止。想要原本的 fail-fast，開 `Config.SetPanicOnError(true)`：任何被記錄的錯誤都會帶著 `*ErrorInfo` panic（可 recover，不是 `os.Exit`）。`SetDontPanic`／`GetDontPanicStatus` 仍是新開關的反向別名，已標為 **Deprecated**，會在下下個版本移除。
- 新增介於 Warning 與 Fatal 之間的 `LogLevelError` 與 `LogError`。所有寫進實例 `Err()` 的紀錄改用 Error 等級，Warning 回歸「做完了但值得注意」的原意。
- **BREAKING**：`DataList` 與 `DataTable` 的 `Err()` 改為**黏性**，會保留第一個失敗，直到呼叫 `ClearErr()` 或 `PopErr()` 為止，串接結尾檢查一次就能看到根因，不再只看到最後的症狀。`SetErr` 也不再覆蓋已經記錄的錯誤，v0.3.3 會覆蓋。
- **BREAKING**：搜尋某個值而沒找到不再設定 `Err()`。`FindFirst`、`FindLast`、`FindAll`、`Count` 與空 list 的統計量改以 Warning 記錄，不動 `Err()`，黏性錯誤才不會被一般提問填滿。指涉不存在的東西（越界索引、`GetColByName` 找不存在的欄）仍算錯誤，因為呼叫端除了一個裸 nil 之外沒有別的訊號。
- **BREAKING**：`DataList` 的轉換方法不再回傳 `nil`。`Normalize`、`MovingAverage`、`WeightedMovingAverage`、`ExponentialSmoothing`、`DoubleExponentialSmoothing`、`MovingStdev`、`Difference`、`Diff`、`PctChange`、`Rank` 失敗時回傳帶著錯誤的空 list（接收者也會記錄），`dl.MovingAverage(0).Sort()` 不再因 nil 而 panic。查找類（如 `GetColByName`）找不到時仍回 `nil`；針對上列轉換寫的 `result == nil` 檢查會失效，請改成 `result.PopErr()`。
- `Rank` 以精確方式比較整數，與 v0.3.3 起的 `Sort` 一致：排序與判定並列都依原始儲存格，不再用 float64 副本，兩個大於 2^53 的 `int64` 不再得到相同的名次。
- **BREAKING**：讀取 insyra 無法解碼的 CSV 編碼改為回傳錯誤並列出支援的編碼，不再把原始位元組當成儲存格塞進表（那些內容不是有效的 UTF-8）。核心的 CSV 讀取函式與 `csvxl` 的 `CsvToExcel`、`AppendCsvToExcel`、`EachCsvToOneExcel` 都是如此，後三者在 v0.3.3 仍會原樣讀入這類檔案。偵測器判定為 IBM420 或 IBM424 的檔案沒有解碼器，也一樣回傳錯誤。
- **BREAKING**：編碼名稱只查一張對照表，大小寫與分隔符號不影響。v0.3.3 另外接受任何含有 `utf8`、`big5`、`gb`、`utf16` 的名稱，這條規則已移除，像 `utf8mb4` 這樣的名稱現在會回傳錯誤。常見的別名都已列進對照表：`utf-8-sig` 與 `utf-8-bom` 讀成 UTF-8 並去掉開頭的位元組順序標記，`big5-hkscs`、`csbig5`、`cn-big5`、`x-x-big5` 讀成 Big5，`x-gbk`、`gb_2312-80`、`csgb2312`、`csiso58gb231280`、`chinese`、`iso-ir-58` 讀成 GBK。
- 取樣太短導致偵測器無法判定編碼時，`DetectEncoding` 改為記警告並假設是 UTF-8，不再讓整個讀取失敗。
- CSV 輸出預設會防範公式注入：開頭是 `=`、`+`、`-`、`@`、又不只是單純數字的文字，寫出時會在前面加上單引號，試算表開啟時會當成文字顯示，不會執行。數字，以及像 `-5`、`+886912345678` 這樣只是數字的文字，一律不變。要讓程式原樣讀回時，用 `CSVWriteOptions{AllowFormulas: true}`，CLI 的 `save` 則是 `allowformulas true`。**BREAKING（輸出）**：`ToCSV` 以前除非設定 `SanitizeFormulas`，否則會原樣寫出這類文字。`SanitizeFormulas` 現在沒有作用，已標為 **Deprecated**，請直接移除。
- **BREAKING**：CCL 只計算需要的部分，也不再硬算一個讀不通的運算式。`&&`、`||`、`CASE` 比照原本就短路的 `IF`，`B != 0 && A / B > 1` 在 B 為 0 的列回 false，不會再報出那個守衛本來就是要避免的 division by zero。函數引數之間必須剛好一個逗號：`SUM(A B)` 過去被接受並把兩欄一起加總，尾逗號則被忽略。`&` 的優先級降到 `+`、`-` 之下（與 Excel 一致），`'a' & 1 + 2` 得到 `"a3"` 而不是錯誤（`'1' & '2' + 3` 也從 `15` 變成 `"15"`）。`Docs/CCL.md` 補上完整的優先級表，包含兩個容易踩到的地方：`-2^2` 是 `4`、`2^3^2` 是 `64`。
- **BREAKING**：做不到的運算，CCL 不再給出一個值。兩個字串之間改用字典序比較（過去 `'abc' < 'abd'` 與 `'abc' > 'abd'` 都是 false，文字欄根本沒有順序）。以 `>`、`<`、`>=`、`<=` 比較空字串、日期字串或日期與數字時，現在和其他文字一樣回錯，v0.3.3 則回答 `false`，所以日期欄的 `A > 0`、含空白格欄位的 `A > 5` 現在都會失敗。`nil` 串接成空字串，不再把 Go 的 `"<nil>"` 寫進儲存格，與原本就回 `"x"` 的 `UPPER(nil) & 'x'` 一致。單獨使用的欄位範圍改為對當前列求值，與「裸寫 `A` 就等於 `A.#`」同一條規則：`AddColUsingCCL("r", "A:B")` 每一格得到該列的 A、B 值，不再是內部的 `ccl.ColumnRange` 結構。`LAG`／`LEAD` 接受整列，`LAG(@, 1)` 讓每一列拿到前一列，不再是整張攤平的表位移一格。會做算術的序列函數仍然拒絕整列，裸寫的列範圍（`1:2`）也仍然回錯，訊息改成提示要接在欄位上。`AND()`／`OR()` 要求至少兩個引數，過去 `AND()` 回 true、`OR()` 回 false。列索引、範圍界線與滾動視窗必須是整數：`A.(1.7)` 過去讀第 1 列、`ROLLING_MEAN(A, 2.9)` 用視窗 2，NaN 索引在 arm64 讀第 0 列、在 amd64 卻報錯。`1 && 0` 與 `'yes' && true` 仍然可用，文件原本說它們會報錯，現在改成列出 evaluator 實際使用的強制轉換表。
- **BREAKING**：insyra 把數字寫成文字時，統一採用 `ToJSON` 原本就在用的規則。Go 的預設格式在數字到一百萬時就改用科學記號，所以 150 萬的營收在 CCL 裡會變成 `營收：1.5e+06`，`ToCSV` 寫進檔案的是 `1.5e+06`，`ToStringSlice` 回傳的也是它，只有 `ToJSON` 寫的是 `1500000`。現在 0.000001 到 10 的 21 次方（不含）之間的數字一律寫成完整數字（`1500000`、`0.00001`），範圍以外的維持原本的科學記號寫法，一個字都不變（`1e-07`、`1e+21`）。讀回來的數字完全相同。`OneHotEncode` 與 `Pivot` 用數值產生的欄名也跟著改，`price_1.5e+06` 會變成 `price_1500000`，用舊欄名取欄的程式會找不到。浮點數類別如果因此跟 `int` 或字串類別寫成一樣，會在 fit 時被拒絕，這跟 `int(1)` 與 `float64(1)` 原本的處理方式相同。`nil` 類別的欄名仍是 `<nil>`，排序結果與 `Show` 在終端機的顯示都不變。
- **BREAKING**：`AtomicDoAll` 的參數從 `...any` 改成 `...Lockable`。它原本什麼都收，遇到鎖不了的值只記一則警告，然後在那個值沒上鎖的情況下照樣執行回呼，所以傳錯東西的人以為自己受到保護，其實沒有。`*DataList`、`*DataTable` 與 `isr` 的包裝型別都符合 `Lockable`，其他型別現在會編譯失敗，用 `[]any` 組的切片要改成 `[]insyra.Lockable`。
- **BREAKING**：`ExecuteCCL` 改為全有或全無。原本腳本在第三條語句失敗時，前兩條已經套用，表停在改了一半的狀態，也沒有任何地方告訴呼叫者。現在腳本先在表的私有副本上執行，全部成功才寫回，`Err()` 會指出是哪一條失敗。
- 編碼器依數值辨識整數類別，與 v0.3.3 起的搜尋、計數相同：`OrdinalEncode` 的 `Order: []any{1, 2, 3}` 現在能對上 CSV 的欄，不再報錯。在 `int` 資料上 fit 的編碼器可以轉換 `int64` 資料，同一欄裡的 `int(1)` 與 `int64(1)` 也算同一個類別。
- CCL 的 `DATEADD` 拒絕超過 2,147,483,647 天、月或年的位移，`LAG`、`LEAD`、`ROLLING_*` 超出同一範圍的位移或視窗也會回錯。v0.3.3 兩者都接受到 `int64` 的範圍。
- **BREAKING**：`mkt.RFM`、`mkt.CustomerActivityIndex` 與 `plot.CreateKlineChart` 收的日期格式樣式，改由掃描器解析，不再是逐個樣式對整個字串做取代。同一個字母的連續段落是一個 token，所以 `MMM` 是短月份名稱（過去會變成 `011`）、`MMMM` 是完整名稱，`A`／`a` 是 AM/PM。字面文字現在要用方括號——`"[Date]: YYYY"`——因為沒有括起來的 `D` 是日期 token；在這次改動之前 `"Date: YYYY"` 會變成 `"2ate: 2006"`，所以含字面文字的樣式本來也不能用。`hh` 與 `h` 仍然對應 24 小時制。
- 讀取 Excel 現在傳給 excelize 512 MB 的解壓上限，不再沿用它 16 GB 的預設值，所以一個解壓後比主機記憶體還大的小檔案會被拒絕而不是讀進來。`insyra.ExcelReadOptions` 匯出，讓 `csvxl` 套用同一個上限。
- `ToJSON` 對實作 `fmt.Stringer`、但沒有實作 `json.Marshaler` 或 `encoding.TextMarshaler` 的值，改為匯出它的文字。過去原值會直接交給 marshaller，欄位未匯出的 struct 會被寫成 `{}`，所以 Parquet 的十進位欄位匯出成 `{}`。實作上述任一介面的值不受影響，`time.Time` 維持 RFC 3339 形式。
- **BREAKING**：`GroupBy`、`Pivot` 與 `Merge` 不再把列印結果碰巧相同的相異值併成一組。key 編碼器的 fallback 是 `%T:%v`，不會往內遞迴，所以 `[]any{1}` 與 `[]any{"1"}` 產生同一個 key，整數與字串變成同一組。編碼器現在會遞迴進入 slice、array、map 與 struct，map 依編碼後的 key 排序輸出使結果與迭代順序無關，並在深度 64 停止，因為自我參照的值會耗盡堆疊，而那在 Go 裡是 `recover` 接不到的 fatal error。一般純量的 key 逐位元組不變。
- 修正 `Counter` 對每個 `NaN` 產生一個取不回來的項目。`NaN` 對 Go 而言可比較，卻永遠不等於自己，所以每次 `counter[NaN]++` 都建立一個誰也查不到的新 key：含三個 `NaN` 的欄位會回報三個計數為 1 的項目，而 `Count` 正確地回答 3。現在整欄的 `NaN` 合併成一個項目並帶正確計數，以 `counter[insyra.ToMapKey(math.NaN())]` 讀取。含 `NaN` 的陣列或結構同樣處理，它們一樣不等於自己。`counter[math.NaN()]` 仍然回 0，這一直如此也必然如此，因為 Go 的 map 比對不到 `NaN` key。
- **BREAKING**：格子裡的定點十進位值算是數值。`Mean`、`Sum`、`Describe`、`stats` 與其他所有數值路徑都讀得到它，過去它們會回 `NaN` 而且沒有錯誤，而金額欄位正是這種情況，因為 `finance.ScheduleTable` 與 Parquet 的 `Decimal128` 欄都放十進位值。`IsNumeric` 跟著一致，因為「對函式庫的一半是數值、對另一半讀不到」正是 v0.3.3 已經為 `time.Duration` 這類具名數值型別關掉的裂縫。十進位現在也與其他數值一起排序，而不是排在字串之後自成一組，所以混合欄位會依值交錯；兩個十進位之間仍以十進位本身的比較決定，超出 `float64` 十六位有效數字的差異照樣分辨得出。判斷依據是形狀，也就是能報出自己的文字與小數位數、且該文字可解析為數字，所以每個值都會經過的 `internal/utils` 不依賴任何特定十進位套件，任何同樣形狀的函式庫都適用。
- **BREAKING**：Insyra 現在需要 Go 1.26，`go.mod` 的 `go` 指示為 `1.26.8`。`lp` 套件新的預設求解器 [go-milp](https://github.com/daniel-sullivan/go-milp) 需要 Go 1.26。使用 Go 1.21 以上時，除非設定 `GOTOOLCHAIN=local`，`go` 指令會自動下載 1.26.8 工具鏈。0.3.x 版本線維持 Go 1.25。
- **BREAKING**：CCL 的裸識別字一律是 Excel 式欄索引，賦值左右兩側都一樣。`qty_1 * 2` 與 `price = A` 過去在識別字無法解析成欄位字母時會退回欄名查表，於是欄名能不能用取決於它有沒有數字或底線，而 `price * 2` 從來不會成功。兩種寫法現在都是錯誤，訊息會告訴你正確寫法：欄名為 `price` 的表上，`price * 2` 會說明 `price` 被讀成欄索引 PRICE、超出最後一欄，而名為 `price` 的欄要寫成 `['price']`。`['price']` 與 `['price'] = ...` 不變，本來就是文件寫的欄名寫法。 括號索引的內容不是欄位字母時，例如 `[qty_1]`，現在在編譯階段就以同樣的訊息失敗，不再拖到第 0 列。
- **BREAKING**：所有選欄位的參數與設定欄位現在只吃同一種選擇器，而且只有三種寫法：`string` 是 Excel 式索引（`"A"`、`"B"`、... `"AA"`），`insyra.Name("price")` 是逐字比對的欄名，`int` 是 0 起算的位置、負數從尾端算。過去函式庫裡同時有四套規則：`GetCol` 先當索引、失敗再把字串轉大寫當欄名找；`GroupBy`、`Pivot`、`Unpivot`、`Resample`、window 的 `*Col` 方法、編碼器、縮放器與填補家族則是先查欄名再當索引；`DataTableSortConfig` 有三個欄位加一套優先順序；`mkt` 的設定則是八組 `XxxColIndex`／`XxxColName`。**現在裸字串永遠不是欄名**，所以 `GroupBy("revenue")`、`CumSumCol("revenue")`、`StandardScale("age")`、`FillWithMean("value")`、`PivotConfig{Values: "sales"}` 與 `Resample("ts", ...)` 要改寫成 `insyra.Name(...)`；這類呼叫大多會直接報錯，因為欄名很少剛好解成範圍內的索引，而且表上真的有同名欄位時，錯誤訊息會告訴你要寫哪個 `Name(...)`。`DataTableSortConfig` 的 `ColumnIndex`／`ColumnName`／`ColumnNumber` 收成一個 `Col`，`mkt` 的八組也各收成一個欄位（`CustomerIDCol`、`TradingDayCol`、`AmountCol`、`OrderIDCol`、`ProductIDCol`）。`GetCol`、`UpdateCol`、`SetColToRowNames` 與四個 `Replace*InCol` 仍保留明講版本，而且補齊了：新增 `GetColByIndex`、`UpdateColByIndex`、`UpdateColByName`、`SetColToRowNamesByName`、`ReplaceInColByName`、`ReplaceNaNsInColByName`、`ReplaceNilsInColByName`、`ReplaceNaNsAndNilsInColByName`。`isr.Name` 現在就是 `insyra.Name`，兩邊建立的選擇器可以互通。欄名不參與解析，只影響錯誤訊息；欄位剛好叫做你給的索引時，索引仍然勝出，並記一筆同時講出兩種讀法的警告。
- **BREAKING**：`ReadSQLStream` 改回傳 `iter.Seq2[*DataTable, error]`，用法是 `for dt, err := range insyra.ReadSQLStream(ctx, db, "orders")`，並移除 `ReadSQLChunk`。舊的 channel 版本只要呼叫端中途離開迴圈，就會漏掉一個 goroutine 和一條資料庫連線，**就算照文件說的 cancel context 也一樣**：讀取端的取消分支會把 context 的錯誤送到一個已經沒人在讀的 channel，然後一直卡在那裡直到程式結束，同時握著查詢結果不放。現在每一批都在呼叫端自己的 goroutine 裡讀，迴圈不管怎麼結束都會立刻關閉查詢結果。查詢改在迴圈開始時才執行，所以建立或執行查詢的失敗會成為迴圈的第一個值，而不是第二個回傳值。
- 以另一個錯誤為原因的錯誤，現在用 `%w` 包住原因，而不是把它格式化成文字，所以 `errors.Is` 與 `errors.As` 能認出原因，例如偵測不到編碼的 CSV、失敗的 CCL 序列或聚合函數，以及 `csvxl`、`pd`、`datafetch` 過去把被包住的失敗寫成字串的地方。訊息文字不變。
- 讀寫不再非得經過硬碟上的檔案。`ReadCSV(r io.Reader, opts)` 可以從任何來源讀 CSV，例如 HTTP 回應、壓縮檔裡的檔案、`embed.FS` 檔案或記憶體中的位元組，`Encoding` 留空時會從開頭的位元組偵測編碼；`ReadCSV_FileWithOptions` 與 `ReadCSV_StringWithOptions` 現在都呼叫它，同樣的位元組不論走哪個入口都得到同樣的表格。`StreamCSV(r, opts, batchSize)` 以 `for dt, err := range insyra.StreamCSV(...)` 一批一批讀 CSV，記憶體裡只放當下這一批；標題列會命名每一批的欄位，型別逐批推斷。`ReadJSON` 接受 `io.Reader`，`ReadExcel(r, sheet, …)` 從來源讀活頁簿並套用與 `ReadExcelSheet` 相同的解壓上限，`(*DataTable).WriteCSV` 與 `WriteJSON` 可寫到任何 `io.Writer`，`ToCSVWithOptions` 與 `ToJSON` 現在也透過它們寫檔。既有函式的簽名都沒有改變。
- DataTable 現在可以寫成 Excel。`(*DataTable).ToExcel(path, ExcelWriteOptions)` 把表格寫成活頁簿裡的一張工作表，活頁簿其他工作表維持原樣，檔案不存在時會自動建立，所以一份報表可以一年放一張表。工作表已經存在時會回傳可用 `ErrSheetExists` 比對的錯誤，檔案完全不動；設定 `IfSheetExists: SheetExistsReplace` 才會取代那張表，而且保留它原本的位置，原本隱藏的也維持隱藏，其他工作表定義的名稱也留在原處。`Sheet` 預設為 `Sheet1`。數字、布林值與時間會存成 Excel 的原生值。`(*DataTable).WriteExcel(w, opts)` 可以把單一工作表的活頁簿寫到任何 `io.Writer`。
- **BREAKING**：`ToFloat64`、`ToFloat64Safe` 與 `ReadSlice2D` 改成一般函式，不再是存著函式的變數。呼叫方式完全不變，只有對它們賦值會編譯失敗：以前 `insyra.ToFloat64Safe = …` 會換掉 `stats`、`ml`、`nn`、`quant` 與核心共用的轉換，讓整個程式的結果都跟著變。把二維 slice 轉成 DataTable 現在只有 `ReadSlice2D` 一個名字；`Slice2DToDataTable` 仍可使用，已標為 **Deprecated**，下一版移除。
- 各種 scaler、`SimpleImputer`，以及 one-hot、label、ordinal 編碼器找不到欄位時，現在會跟其他欄位選擇器一樣說明原因。以前對有 `Age` 欄的表格呼叫 `NewStandardScaler().FitTransform(dt, "Age")`，只會回報 `column Age not found`，看起來像欄位不存在；現在會說明 `"Age"` 被當成 Excel 式欄位索引，並提示改寫成 `Name("Age")`。
- **BREAKING（行為，簽名不變）**：結尾的選填參數最多只能給一個值。`DataList.Shift` 與 `DataTable.ShiftCol` 給兩個補值、`FillForward`／`FillBackward` 給兩個 limit、`FillByInterpolation` 給兩個旗標、`Sample`／`SampleFrac`／`Shuffle`／`TrainTestSplit` 給兩個 `SamplingOptions`、`Describe` 給兩個 `DescribeOptions`，以前都會默默用第一個、丟掉其他的；現在會記錄錯誤，而且不做任何改動。`Rank(true, false)` 以前會記錄錯誤但照樣排名，現在不會排名。`ReadSQL`、`ReadSQLContext`、`ReadSQLStream`、`ToSQL`、`ToSQLContext` 與 `ReadCSV_File` 收到第二個設定包或編碼時會回傳錯誤。
- `DataList.DataType()` 與 `DataTable.ColDataTypes()` 讓程式能判斷一欄存的是什麼資料：`DataTypeNumber`、`DataTypeString`、`DataTypeBool`、`DataTypeTime`、`DataTypeOther`、`DataTypeMixed`，沒有任何值時是 `DataTypeEmpty`。缺值不算進去，整數和小數都算數字，看起來像數字的文字仍然是文字。以前只有 `ShowTypes` 能把每一格的 Go 型別印出來給人看；現在 `ShowTypes` 最上面會多一列 `DataType`，把同樣的總結放在每格型別的上方。
- **BREAKING**：`DataTable.FillByInterpolation` 在欄位前面多了 `extrapolate bool`，跟 `FillForward` 把 limit 放在前面一樣，因為表格版本以前完全無法外插。`dt.FillByInterpolation(cols...)` 要改成 `dt.FillByInterpolation(false, cols...)`。
- 表格的 `FillWithMean`、`FillWithMedian`、`FillByInterpolation`、`FillWithMode` 遇到你明確指定、卻補不了的欄位（數值補法遇到文字欄或混合欄，或整欄沒有任何值）時會記錄錯誤，寫出欄位與它的資料型別，其他指定的欄位照常補完。以前會默默跳過，讓人以為補好了。沒有指定欄位時，補不了的欄位仍然會跳過。
- **BREAKING**：`NewSimpleImputer` 改成接受可省略的 `SimpleImputerOptions{Strategy, FillValue}`，取代 `(strategy, constant ...any)`。不給設定就用平均。`NewSimpleImputer(insyra.ImputeMedian)` 要改成 `NewSimpleImputer(insyra.SimpleImputerOptions{Strategy: insyra.ImputeMedian})`，`NewSimpleImputer(insyra.ImputeConstant, 7)` 要改成 `…{Strategy: insyra.ImputeConstant, FillValue: 7}`。
- **BREAKING（行為，簽名不變）**：`SimpleImputer.Fit` 用平均或中位數時，只要選到的欄位不是數字欄就會失敗，錯誤會寫出欄位和它的資料型別，而且一欄都不學。以前會讓那一欄維持沒補，只在 `Params()[name].PassThrough` 記一筆，看起來像是 fit 成功了。`ScalerParams.PassThrough` 現在永遠是 false，已標為 **Deprecated**，下一版移除。
- `ShowRange`、`ShowTypesRange` 與 `Show` 的範圍參數，遇到超過兩個值、第一個值不是 `int`、或結尾既不是 `int` 也不是 `nil` 時，會印出錯誤訊息；以前會忽略這些參數並顯示全部。DataList 與 DataTable 新增 `ShowHead(n)`、`ShowTail(n)`（以及 `ShowHeadTo`／`ShowTailTo`），是 `ShowRange(n)`、`ShowRange(-n)` 比較直白的寫法。
- **BREAKING**：CSV 與 JSON 的讀寫函式各只剩一個名字，設定都放進一個可省略的設定包。`ReadCSVFile(path, opts...)` 與 `ReadCSVString(s, opts...)` 取代 `ReadCSV_File`／`ReadCSV_FileWithOptions` 和 `ReadCSV_String`／`ReadCSV_StringWithOptions`；`ReadJSONFile` 取代 `ReadJSON_File`；`ToJSONBytes`／`ToJSONString` 取代 `ToJSON_Bytes`／`ToJSON_String`。舊名字這一版仍可使用，意思不變，已標為 **Deprecated**，下一版移除。
- **BREAKING**：`DataTable.ToCSV(path, opts ...CSVWriteOptions)` 取代 `ToCSV(path, rowNames, colNames, bom bool)`，舊寫法會編譯失敗；`ToCSVWithOptions` 標為 **Deprecated**。`dt.ToCSV(path, false, true, false)` 改成 `dt.ToCSV(path)`。`WriteCSV` 與 `ReadCSV` 的設定包改成可省略；`StreamCSV` 的批次大小移到設定包前面，`StreamCSV(r, opts, 1000)` 改成 `StreamCSV(r, 1000, opts)`。
- **BREAKING**：所有設定包裡的標題列與列名欄位統一命名為 `NoHeaderRow` 與 `HasRowNames`，全部留空就代表最常見的檔案：第一列是欄名、沒有列名欄。`CSVReadOptions.FirstRowToColNames` 與 `CSVWriteOptions.SetColNamesToFirstRow` 改成意思相反的 `NoHeaderRow`；`FirstColToRowNames` 與 `SetRowNamesToFirstCol` 改成 `HasRowNames`；`ExcelWriteOptions` 同步改名，`ToSQLOptions.RowNames` 改成 `HasRowNames`。有設定舊欄位的程式會編譯失敗。**有傳設定包但沒寫標題列那一項的程式，現在讀寫時都會有標題列**：例如 `CSVReadOptions{Encoding: "big5"}` 以前會把第一列當成資料。`ReadExcelSheet` 與 `ReadExcel` 只改了參數名稱。
- **BREAKING（小）**：`IDataTable` 與 `IDataList` 現在列出 `*DataTable`、`*DataList` 的全部方法，以前分別少了 27 個和 17 個。例外是 `ClearErr`、`SetErr`、`Pivot`、`Unpivot`，刻意不列，因為內嵌核心型別的擴充型別會用自己的回傳型別改寫它們；拿介面型別的變數呼叫 `ClearErr()` 或 `SetErr()` 的程式要改用實際型別。內嵌 `*DataTable` 的型別現在能用在所有收表格的地方：`Merge` 以前只收純粹的 `*DataTable`。
- 對 nil 的 `DataList` 或 `DataTable` 呼叫 `Err()`（例如 `GetCol` 找不到欄位時回傳的值），現在會回報 `nil DataList`／`nil DataTable`，不會讓程式當掉。只有 `Err()` 有這個保護：nil 的值呼叫其他方法仍會直接失敗，避免 nil 被默默往下傳。
- **BREAKING**：`Counter()` 改以數值當整數的 key，一律用 `int`，跟 `Count` 一致。CSV 或 JSON 讀進來的整數是 `int64`，所以以前一欄有兩個 5，`counter[5]` 卻查到 0；一欄混了 `int64(5)` 和 Go 直接寫的 `5`，也會分成兩個 key，而 `Count(5)` 回報的是兩者合計。現在這兩種情況 `counter[5]` 都是 2。用 `int64(5)` 查 counter 的程式會查不到，要改寫成 `counter[5]` 或 `counter[insyra.ToMapKey(v)]`，把 key 斷言成 `int64` 的程式要改成 `int`。`ToMapKey` 會把任何寬度的整數轉成同一個 key。`int` 裝不下的整數保留 `uint64` 或 `int64`，小數和文字仍是各自的 key，跟 `Count` 一樣。
- `StandardScaler`、`MinMaxScaler`、`RobustScaler` 與 `MaxAbsScaler` 實作了 `json.Marshaler` 與 `json.Unmarshaler`，已擬合的 scaler 可以存起來之後再用。用 `json.Unmarshal` 讀回同一型別後，轉換結果與原本完全相同：每個擬合欄都以欄名記住，沒有欄名的才記位置，NaN 參數讀回仍是 NaN。以前對 scaler 呼叫 `json.Marshal` 只會得到 `{}`。
- CCL 的 `TOSTR` 遇到動詞是標點或非 ASCII 字母、又沒有值可填的格式，會像字母動詞一樣回傳錯誤：v0.3.3 的 `TOSTR(A, '%v %_')` 會把 `1 %!_(MISSING)` 寫進儲存格。
- 內容相同的巢狀陣列與巢狀 slice，不論大小，對 `Count`、`Counter`、分組與合併都是同一個值。v0.3.3 把大的巢狀 slice 寫成摘要、陣列則從不這樣做，所以內容超過約 1 KiB 後兩者就對不上了。
- **BREAKING**：`ProcessData` 改成回傳 `([]any, error)`，不再回傳 `([]any, int)`。原本的 int 只是切片長度，讀不了的值則回傳 `nil, 0` 並寫一行 log，呼叫端分不出這和空切片的差別。現在讀不了的型別、`nil` 和 nil 指標都會回傳錯誤，nil 的 `*DataList` 也不會再讓它當掉。寫法改成 `values, err := insyra.ProcessData(x)`，長度用 `len(values)` 取得。
- `SqrtRat`、`PowRat`、`SortTimes` 與 `F64orRat` 標為 **Deprecated**，下一版移除。insyra 本身沒有用到它們，每個函式的說明都寫了可以改用的 `math/big` 或 `slices` 寫法。移除前的這一版先修好兩個問題：`SqrtRat` 收到負數或 `nil` 會回傳 `nil`，不再 panic。`PowRat` 的指數是負數時會回傳倒數，以前 `PowRat(big.NewRat(2, 3), -2)` 會回傳 `1`。`PowRat(nil, n)` 以及 0 的負次方回傳 `nil`。
- 新增 `DataTable.FilterRowsWhere(keep func(row *DataList) bool)`。`keep` 每次拿到一整列，回傳 true 的列會保留，所以條件可以比較同一列的兩個欄位，例如 `dt.FilterRowsWhere(func(row *insyra.DataList) bool { return insyra.ToFloat64(row.Get(0)) < insyra.ToFloat64(row.Get(1)) })`。`Filter` 與 `FilterRows` 是一格一格呼叫函式，只要有一格通過就保留整列，文件現在把這點寫清楚了。`Filter` 收到的欄位參數是 Excel 式的欄位字母，文件裡拿它和欄位名稱比較的範例永遠不會成立，已經改正。`FilterByCustomElement` 的結果和 `Filter` 完全相同，標為 **Deprecated**，請改用 `Filter`，下一版移除。`Filter` 與 `FilterRows` 的函式若在過程中替同一張表新增欄位，不會再因索引超出範圍而當掉。
- 新增 `DataTable.SliceRows(from, to)` 與 `SliceCols(from, to)`，取出一段連續的列或欄，規則和 Go 的 `s[from:to]` 一樣：包含 `from`，不含 `to`。`SliceRows` 用從 0 起算的位置。`SliceCols` 的兩端都是欄位選擇器，所以在欄名是 a～d 的表上，`SliceCols("B", "D")`、`SliceCols(1, 3)` 和 `SliceCols(insyra.Name("b"), insyra.Name("d"))` 是同一個呼叫。傳 `nil` 代表從第一欄開始或一路到最後一欄。範圍用不了時會記錄錯誤並回傳空表，不會 panic。以前用篩選名稱做這件事的十個方法，也就是 `FilterColsByColIndex…` 與 `FilterRowsByRowIndex…` 加上 `GreaterThan`、`GreaterThanOrEqualTo`、`EqualTo`、`LessThan`、`LessThanOrEqualTo`，標為 **Deprecated**，下一版移除，每個方法的說明都寫了對應的切片寫法。它們的結果不變，只有 `FilterColsByColIndexLessThan` 與 `…LessThanOrEqualTo` 收到超過最後一欄的字母時，會保留所有欄位，以前會 panic。`Headers` 與 `SetHeaders` 標為 **Deprecated**，請改用 `ColNames` 與 `SetColNames`。
- **BREAKING**：`PivotConfig.AggFunc` 改收 `Aggregate` 與 `Resample` 使用的 `AggregateOp`，型別是指標：寫成 `AggFunc: new(insyra.OpSum)`，不再寫 `AggFunc: "sum"`。`nil` 仍代表不彙總，所以 `(Index, Columns)` 重複的列會回傳錯誤。之所以用指標，是因為 `OpSum` 是 `AggregateOp` 的零值，一般的 `AggregateOp` 欄位無法表達「沒有設定」。字串別名（`"avg"`、`"std"`、`"average"` 等）隨字串一起取消，打錯的 op 現在無法編譯。
- `GroupBy`、`Pivot`、`Merge` 的鍵、`OpNUnique` 以及 `Describe` 的相異值計數，現在把 `uintptr` 和其他寬度的整數視為同一類，和 `Count`、`Counter` 一致，以前它會自成一組。規則也寫進了 `Docs/DataTable.md` 的 `GroupBy` 段落：整數不論寬度都依數值分組，所以 CSV 讀進來的 `int64(1)` 和 Go 直接寫的 `1` 同組，而 `1`、`1.0` 和 `"1"` 仍是三組。
- DataTable 的 `ShowRange`、`Show` 與 `ShowTypesRange` 改成只依要印出的列計算欄寬，不再把整張表的每一格都格式化一遍。在 1,000,000 × 3 的表上，`ShowRange(5)` 原本要 279–437 ms，現在約 40 µs。`ShowTypesRange(5)` 從約 175 ms 降到 18 ms，`Show()` 從約 500 ms 降到 300 ms，剩下的時間花在涵蓋所有列的統計摘要行。印出所有列的畫面和以前完全相同，只印部分列的畫面可能變窄，因為範圍外的寬儲存格不會再撐寬欄位。`Show` 的參數型別改為公開的 `Showable`，自己寫的函式也能接收任何 `Show` 接受的東西。
- `ToSQL` 使用 `IfExists: SQLActionIfTableExistsReplace` 時，MySQL 上寫入失敗不會再弄丟舊表。MySQL 的 `DROP TABLE` 一執行就會提交，即使放在交易裡也一樣，所以之後的 `INSERT` 一旦失敗（例如超過 65,535 個參數上限），舊表和新資料都不見了。現在 MySQL 上的 Replace 會先把資料寫進暫存表（`insyra_new_<亂數>`），再用一道 `RENAME TABLE` 換上，最後刪掉舊表。寫入失敗或被取消時，會刪掉暫存表，舊表維持原狀。暫存表有自己的名稱，所以 MySQL 上的 Replace 現在需要整個資料庫的 CREATE、INSERT、ALTER、DROP 權限，只有那張表的權限不夠。被其他表以外鍵參照的表也會被拒絕。這兩種情況都會在改動任何東西之前就失敗。SQLite 與 PostgreSQL 的 DDL 可以放在交易裡，仍在同一個交易中刪除再重建。`ToSQL` 文件也不再宣稱每種資料庫的每道語句失敗時都會復原：在 MySQL 上，因為不存在而新建的表，以及 append 模式新增的欄位，在寫入失敗後會留下來。
- **BREAKING**：`DataList.WeightedMean` 與 `WeightedMovingAverage` 的權重改收 `[]float64`，跟 `RollingOptions.Weights` 用同一個型別。以前收 `any`，執行時才判斷傳進來的是 slice、陣列還是 `DataList`，其他型別一律當成空的權重，最後以「長度不符」報錯，看不出真正的問題是型別。傳 `[]float64` 的程式不用改；傳 `DataList`、`[]int` 或 `[]any` 的程式會編譯失敗，要先把權重轉成 `[]float64`。計算結果不變。
- **BREAKING**：`DataList.ParseNumbers` 改用 CSV 讀檔判斷欄位型別的規則。list 裡的數字全是整數、也沒有空字串時，全部轉成 `int64`；否則全部轉成 `float64`，空字串變成 `NaN`。以前一律轉成 `float64`，所以 `"9007199254740993"` 會變成 `9007199254740992`，`NewDataList("1", 2).ParseNumbers()` 得到 `[1.0 2.0]`，現在則是 `int64`。list 裡原本就是數字的值也一起判斷、轉成同一種型別。`nil` 保持原樣，不再被當成失敗；不是數字的文字與 `bool` 之類的值保持原樣，並合併成一筆錯誤，說明有幾個值沒轉、第一個在第幾列，以前是每個值各記一次。
- 視窗類轉換無法執行時，回傳的 list 會帶著原因。`Rolling` 的 `Window` 小於 1、`MinObs` 大於 `Window` 或 `Weights` 長度不對，`EWM` 沒有恰好指定一個衰減參數，`Apply(nil)`，以及 `Corr`、`Cov`、`Beta` 傳入 `nil` 的 list，以前回傳的空 list 的 `Err()` 是 `nil`，現在會帶著錯誤。`DataTable.RollingCol` 與 `EWMCol` 是在欄位的副本上計算，所以選項不合法時，錯誤只記在副本上，呼叫端手上沒有任何東西記到；現在表格也會記錄。`ShiftCol`、`DiffCol`、`PctChangeCol`、四個 `Cum*Col` 與三個視窗 builder 遇到不存在的欄位時，回傳帶著表格錯誤的空 list。分組轉換遇到不合法的參數（`RollingCol` 的 `Window: 0`、`DiffCol` 的 `periods` 為 0、`ShiftCol` 給兩個填補值）時，以前會回傳整欄 `nil` 且什麼都沒記錄；現在 `.As` 會在表格記錄錯誤，並回傳帶著錯誤的空 list。失敗的結果仍是空的，不會變成整欄 `nil`，因為合法的視窗觀察值不足時也會回傳整欄 `nil`。
- **BREAKING（只影響顯示）**：`Show`，以及沒有給範圍的 `ShowRange` 與 `ShowTypesRange`，表格或 list 在 60 列以內會全部印出，以前是超過 25 列就截斷。超過 60 列時仍印前 20 列與最後 5 列，中間以 `...` 省略。pandas 也是 60 列以內全部印出，只是超過後改印頭尾各 5 列。會解析 26 到 60 列輸出內容的程式，現在會看到所有列。文件以前說 `ShowRange()` 會印出所有列，但超過 25 列時從來不是這樣，現在寫明了實際規則。
- `DataList.Difference`、`MovingAverage`、`MovingStdev` 與 `WeightedMovingAverage` 標為 **Deprecated**，下一版移除，在那之前行為不變。對應的方法是 `Diff(1)`、`Rolling(RollingOptions{Window: w}).Mean()`、`Rolling(...).Std()` 與 `Rolling(RollingOptions{Window: w, Weights: ws}).Mean()`，但不能直接替換：它們的結果和輸入一樣長，前面補 `nil`（五個值做 `MovingAverage(3)` 得到三格，`Rolling` 得到五格），遇到空值時給 `nil`，舊方法則是直接失敗或回傳 `NaN`。資料全是數字時，新方法結果的 `Data()[w-1:]` 就是舊方法的結果。完整差異列在 `Docs/DataList.md` 的「Methods that look alike but differ」。`ExponentialSmoothing` 不標 Deprecated，因為 `EWM` 不接受 `alpha = 0`。
- CCL 把每一種 Go 整數型別與 `float32` 的格子都當成數字。`int8`、`int16` 與無號整數欄（`parquet.Read` 讀到檔案裡這類欄位時給的就是這些型別）過去不被 CCL 當成數字：`SUM` 回 0、`MAX` 回 `nil`、`A == 1` 回 false，全都不報錯，而 `A * 2`、`IF(A, 'y', 'n')` 與 `ROUND(A, 2)` 會失敗。`A.B` 也接受存成 `int32`、`int64`、`float32` 或其他整數型別的列號，過去會被拒絕；`ISNA` 與 `IFNA` 也把 `float32` 的 NaN 當成缺值。
- `engine/ccl` 可以註冊序列函式了：用 `RegisterSequenceFunction(name, fn)` 搭配新的 `SeqFunc` 型別，加入一個吃整欄、回傳同樣長度一欄的函式，就像 `LAG`、`CUMSUM` 那樣。以前從模組外只能註冊純量函式與彙總函式。註冊函式的說明也寫清楚函式登錄表的行為：整個程式共用同一份，可以從任意多個 goroutine 同時使用，名稱不分大小寫，同名再註冊一次就會取代原本的函式。用內建函式的名稱註冊彙總或序列函式時，函式和「這是使用者的函式」的標記現在會一起更新，同一時間進行的求值不會看到你的函式卻仍當成內建函式來算。
- `engine/ccl.ResetEvalDepth` 與 `ResetFuncCallDepth` 標為 **Deprecated**。自從 CCL 把遞迴深度放在呼叫堆疊上，它們就什麼也不做，下一版會移除，請直接刪掉這些呼叫。

### CLI
- **BREAKING**：環境名稱只能包含字母、數字、`.`、`_`、`-`，必須以字母或數字開頭，且不得含 `..`。v0.3.3 只拒絕會解析到環境目錄之外的名稱，其他名稱都接受，包括含空格、非 ASCII 字元或 `/` 的名稱。以這類名稱建立的環境，CLI 已無法再開啟、改名或刪除，請手動到環境目錄（預設為 `~/.insyra/envs/`）把資料夾改名。
- **BREAKING**：指令做不到被交代的事時，改為回傳錯誤並以非零狀態結束，不再印出成功訊息。`sort`、`dropcol`、`droprow`、`swap`、`setcolnames`、`sample` 會先檢查目標並指出缺少什麼；`ccl` 與 `addcolccl` 會回報無法編譯的運算式。原本被靜默忽略的腳本步驟現在會失敗。
- **BREAKING**：拼錯的選項值改為拒絕，不再退回預設值。`sort … dsc`、`ttest … eqaul`、`ztest … bogus`、`clean … outliers abc` 過去分別會以升冪、合併變異數、雙尾、2.0 個標準差執行——那是在呼叫端沒有做過的假設下算出來的統計結果。
- **BREAKING**：`plot`、`fetch`、`merge` 對不認識的引數改為回報而非丟棄；`plot` 的 Usage 也不再宣告它從來不接受的選項。
- `config` 拒絕未知的 key（並列出可用的），並驗證 `log-level`、`no-color`、`accel-mode`，無效設定不會寫進設定檔。
- `sample` 對小於等於 0、或不放回時超過來源長度的數量改為回錯，不再存下空結果；`setcolnames` 要求名稱數量與欄數相同，不再把其餘欄名清空或新增空欄。
- `encode … ordinal … order` 現在能對上 CSV 載入的表，以及 one-shot 模式下每次還原的變數裡的整數。原本打的 `1` 是 `int`，存著的是 `int64`，所以 `encode … order 1,2,3` 會把每一格都編成 nil。
- **BREAKING**：`accel` 與九個 DataList 統計指令（`sum`、`mean`、`median`、`mode`、`stdev`、`var`、`min`、`max`、`range`）遇到用不到的引數改為回錯，不再默默忽略。原本 `mean x as m` 會印出平均數卻什麼都沒存，現在會回傳指出那個引數的錯誤。`accel` 不再接受 `--precision`，這個旗標原本用來選 `accel run` 的精度，`accel run` 在 v0.3.1 就已移除，之後沒有任何地方讀它。
- 修正 `insyra env import` 在沒有 `--force` 時，只要目標環境有檔案存在但讀不到，就會把非空的環境蓋掉。判斷目標是否為空的檢查把讀不到 `config.json` 當成「空的」，讀不到 `state.json` 與 `history.txt` 也一樣被忽略。檔案不存在仍然視為空；其他讀取失敗現在會停止匯入，並指出哪個環境無法確認。
- `save` 可以存 Excel：`save <var> report.xlsx [sheet <名稱>] [if-exists fail|replace]`。它只寫一張工作表（預設 `Sheet1`），活頁簿其他工作表都會保留。存到已經存在的工作表會被拒絕，不會自動改名成 `Sheet2`；訊息會提示加上 `if-exists replace` 覆蓋，沒指定工作表時也會提示用 `sheet <名稱>` 另存一張。`.xls` 會被拒絕並提示改用 `.xlsx`，`sheet` 與 `if-exists` 用在其他檔案類型也會被拒絕。以前 `save … report.xlsx` 只會回報 `unsupported output file type`。
- 表格的 `fillna` 現在會照 `extrapolate` 外插，以前這個選項會被丟掉。補值做不到要求的事時，指令會失敗且不存檔：`cols` 指定的欄位補不了、或對文字清單用 `mean`，以前都會被當成補好了存起來。
- `convert` 從 xlsx 轉 csv 時預設會防範公式注入，加上 `allowformulas true` 就原樣寫出，跟 `save` 一致。`convert` 看不懂的參數現在會報錯，不再默默忽略。
- **BREAKING**：每個指令都會拒絕它用不到的參數。超過五十個指令，包括 `iqr`、`cov`、`corr`、`summary`、`get`、`find`、`transpose`、`ttest` 和 `version`，以前只讀自己需要的參數，多的直接丟掉，所以 `iqr x junk` 照樣印出答案，看起來像打對了，`find t 1 as found` 也什麼都沒存。現在指令不會執行，錯誤訊息會指出是哪個參數並附上用法：`iqr: unexpected argument "junk"; usage: iqr <var>`。只有會存結果的指令接受 `as <var>`。`.isr` 腳本裡這一行會失敗，`run` 接著執行下一行。
- **BREAKING**：所有用來選欄或選列的參數都照同一套規則。以前 `col`、`sort`、`swap`、`dropcol` 認得 `price` 卻不認得 `B`，`get`、`set` 認得 `B`，打 `price` 卻印出 `<nil>`，`fillna` 和 `groupby` 兩種都認而且先找欄名。現在沒加前綴的參數會用所有可能的方式解讀：數字是從 0 起算的編號，負數從最後倒數；英文字母是 Excel 式欄索引（只用於欄）；也會當成名稱比對。解讀結果一致就用那一欄，所以 `sort t price`、`sort t B`、`sort t 1` 都能用，以年份命名的欄照樣能用 `2021` 找到。兩種解讀指向不同欄時，指令會停下來告訴你該怎麼寫，不會自己猜：欄位是 `x, a` 的表格執行 `sort t a`，會回報 `a` 可能是 A 欄（`x`）也可能是名為 `a` 的欄，請寫 `index:a` 或 `name:a`。`number:`、`index:`、`name:` 會強制採用其中一種；列接受 `number:` 和 `name:`，`get` 和 `set` 現在也接受列名。適用於 `col`、`row`、`get`、`set`、`sort`、`swap`、`dropcol`、`droprow`，以及 `fillna`、`groupby`、`describe`、`encode`、`parsedates`、`pivot`、`unpivot`、`resample`、`scale`、`merge … on` 的欄清單。
- 一個指令失敗後，下一個指令不再跟著失敗。以前表格會留著程式庫記下的錯誤，所以 `col t B` 失敗後，剛剛才成功的 `sort t price` 會回報「no column is named B」。
- 一次性命令不再改動它還原的變數。`state.json` 現在連同 Go 型別儲存每個變數：DataTable 保留欄位順序、欄名、列名與每一格的型別。`insyra load c.csv as t` 之後另外執行 `insyra cols t`，欄位會照檔案的順序列出，不再變成字母序，欄字母在每個命令裡也都指向同一欄。`parsedates` 轉成日期的欄，到了 `resample` 還是日期，CCL 日期相減得到的欄仍是 `time.Duration`，`3.0` 也仍是 `float64`，不會變成 `int64`。`scale fit` 擬合的 scaler 與 `hclust` 的樹也會保存，`scale transform` 與 `cutree` 可以分開執行。以前 scaler 會消失，`cutree` 也不接受讀回來的樹。
- 環境無法保存的變數（例如 `regression` 的結果）會在儲存時印出一行 `warning:`，在 REPL 或腳本中每個變數只提示一次，不再無聲無息地被丟掉或變成 map。產生它的命令照常成功，其他變數照常保存。直接使用 `cli/env` 的 Go 程式可以用新增的 `Manager.SaveVariables` 取得同一份清單，`SaveState` 仍然只在檔案沒寫成時回傳錯誤。
- 先前版本寫入的 `state.json` 仍可讀取，下次儲存時改寫成新格式。含 NaN 值的環境現在可以 `env export`，`env import` 也會完整保留超過 2^53 的整數。
- `pivot … agg <op>` 改用 `groupby` 與 `resample` 的方式讀取 op，和文件原本的描述一致。只有 `pivot` 接受、文件也從未提過的 `agg average` 現在會被拒絕，`agg custom` 也一樣，它原本就會失敗，因為 CLI 無法傳入函式。
- `rolling` 遇到程式庫拒絕的視窗設定時會回報錯誤，不存任何變數：以前 `rolling x 0 mean` 與 `rolling x 2 mean minobs 3` 會印出 `saved as $result` 並存一個空的 list。`ewm` 改為回報程式庫的錯誤，不再從結果長度推斷是否失敗。
- `parsenums` 跟著 `ParseNumbers` 改變：數字全是整數的 list 會轉成 `int64`，不再是 `float64`。
- **BREAKING**：`ttest two <var1> <var2>` 沒有加 `equal` 或 `unequal` 時，改為執行函式庫預設的 Welch t 檢定，不再假設兩組變異數相等。要得到以前的結果，請加上 `equal`；`help ttest` 會列出預設值。
- `fillna … ffill|bfill … missing nan|nil` 不再把另一種缺值算進 `limit`。以前這個命令會先把兩種缺值一起補、再把另一種放回去，所以另一種缺值的格子會佔用 `limit` 的名額：`fillna x ffill limit 1 missing nan` 對 `[1, nil, NaN]` 會讓那個 `NaN` 補不到。現在會跳過這些格子，結果是 `[1, nil, 1]`。沒有 `limit` 時，以及 `mean`、`median`、`mode`、`interpolate` 的結果都不變。
- `plot` 無法建立圖表時會說明原因，例如沒有任何欄的表格會回報 `plot: CreateLineChart: no data to draw`，不再只說 `failed to create chart`。
- **BREAKING**：`chisq gof` 的每個比例都要寫出所屬類別，例如 `chisq gof colors red=0.5 green=0.3 blue=0.2`。以前只寫數字，會照類別名稱排序後的順序對位，和 `ChiSquareGoodnessOfFit` 一樣容易對錯。現在只寫數字會回傳錯誤，錯誤訊息會附上新的寫法。
- `anova twoway`、`anova repeated` 與新的 `friedman` 指令可以直接對每列一個觀察值的表格執行，也就是讀進來的 CSV 通常的樣子：`anova twoway scores score drug dose`、`anova repeated trial value visit patient`、`friedman trial value visit patient`。第一個參數是 DataTable 變數時就走這個形式；參數是 DataList 變數時，`anova twoway` 與 `anova repeated` 的行為和以前完全一樣，`friedman s1 s2 s3` 則是每位受試者一個 list。欄位照一般的 token 規則指定。`friedman` 會印出 `Q`、自由度與 p 值，CLI 以前沒有 Friedman 檢定。
- CLI 可以執行資料不接近常態時用來取代 t 檢定與單因子 ANOVA 的秩檢定：`wilcoxon single <var> <mu>`、`wilcoxon paired <var1> <var2>`、`mannwhitney <var1> <var2>`（都可以再加 `two-sided`、`greater` 或 `less`），以及 `kruskal <group1> <group2> [groupN]`，分別印出 `W=… p=…`、`U=… p=…` 與 `H=… df=… p=…`。以前只能從 Go 呼叫。

### `ml` 與 `nn`
- **BREAKING（行為改變，簽章不變）**：`Classes()` 不再回傳 nil。`ml` 與 `nn` 共十個分類器型別，在模型尚未 fit、或 pipeline 包的不是分類器時，改為回傳長度 0 的 `*insyra.DataList`，並把原因記在它的 `Err()` 上。過去 nil 的 `*insyra.DataList` 呼叫任何方法都會 panic，連 `Err()` 也不例外，也就是說「問它出了什麼事」這個最安全的第一步，本身就是崩潰的原因。**簽章沒變，所以什麼都不會編譯失敗：寫成 `if classes == nil` 的程式照樣能編，但那個分支從此永遠不會執行。** 請改成 `if classes.Err() != nil`。
- **BREAKING（行為，簽名不變）**：`nn.NewTape` 最多只能給一個種子，給兩個時 `Param` 與 `Backward` 會回報錯誤，不再只用第一個。`Conv2D`、`MaxPool2D`、`AvgPool2D` 及對應的 `New…` 寫法給超過一個設定包時，以前會全部丟掉改用預設值建立；現在由 `Build` 回報錯誤。
- **BREAKING**：`Tape.BatchNormalizationTraining` 與 `BatchNormTraining` 改成接受 `opts ...nn.BatchNormOptions{Momentum, Epsilon}`，不再用 `options ...float32` 依位置解讀。現在可以只設定 epsilon，不必連 momentum 一起寫，也看得出哪個值是哪個。欄位沒填就用 torch 的預設值。`…, 0.2, 1e-5)` 要改成 `…, nn.BatchNormOptions{Momentum: 0.2, Epsilon: 1e-5})`。
- 新增 `Tape.Custom(name, inputs, output, vjp)`，把在 tape 外算出來的運算放上 tape：它的輸入會拿到反向規則回傳的梯度。以前在 tape 外算出的張量會讓它的輸入默默拿到零梯度。反向規則可以不是前向的導數，例如替硬門檻宣告一個平滑的替代梯度。`Custom` 會拒絕格式不對的宣告，包括先前記錄的運算已經讀過的輸出。反向規則回傳錯誤，或梯度的數量、型別、形狀不對時，`Backward` 會失敗並指出是哪個運算。（[issue #375](https://github.com/HazelnutParadise/insyra/issues/375)）
- 新增 `Tape.BackwardFrom(output, upstream)`，可以從 tape 上任何運算產生的張量開始反向傳播，並帶入呼叫者給的、形狀相同的上游梯度。（[issue #375](https://github.com/HazelnutParadise/insyra/issues/375)）
- 失敗的 `Backward` 不再讓 `Tape.Grad` 回傳算到一半的梯度：`Tape.Grad` 與 `Parameter.Grad` 都保留上一次成功的結果。
- 新增 `NewEdgeTopology`、`EdgeSum` 與 `Tape.EdgeSum`，處理以邊列表表示的圖：每個節點加總自己收到的加權邊，成本只跟邊數和數值量成正比，不需要 N×N 的稠密矩陣，tape 也會算出邊權重和節點數值的梯度。數值可以是 `[N]` 或帶批次的 `[B, N]`。每個輸出都是所有乘積的精確總和只捨入一次到最近的 float32，所以邊的順序和核心數量都改變不了結果，在每個平台上都一樣。大型圖會用滿所有核心。（[issue #379](https://github.com/HazelnutParadise/insyra/issues/379)）
- `Tanh` 對每個輸入、在每個平台上都回傳正確捨入的值，也就是真正的 `tanh(x)` 只捨入一次到最近的 float32。以前是把 Go 的 `math.Tanh` 捨入成 float32，而它的 float64 結果並非每個平台都一樣（在 arm64 會合併乘加，在 amd64 執行時依 CPU 選擇是否用 FMA，在 s390x 則是組合語言），所以只在那個結果剛好夠準的地方才正確。全部 2^32 個輸入都在 darwin/arm64 上比對過，2^-13 到 9.5 之間的輸入也在 linux/amd64 與 windows/amd64 上比對過。這三個平台上舊的結果原本就正確，所以沒有任何結果改變。
- `Tape.Tanh` 的梯度每一步都捨入成 float32。以前 Go 編譯器在 arm64 上會把 `1 - y*y` 合併成一次乘加，在 amd64 上不會，同一個梯度在兩邊可能差最後一位。在 arm64 上，10 萬個隨機梯度有 24,892 個改變。現在每個平台上位元都相同。
- **BREAKING**：`ml` 的每個 `Fit*` 函式都回傳自己的型別。包裝 `stats` 的十四個函式原本回傳 `Model`、`ProbaModel` 或 `Transformer`，要讀 fit 出來的 `Result` 得先做型別斷言；現在 `FitLinearRegression` 回傳 `*ml.LinearModel`、`FitLogisticRegression` 回傳 `*ml.LogisticModel`、`FitPCA` 回傳 `*ml.PCATransformer`，其餘依此類推，和樹模型與整體模型的函式一致。把結果指派給介面變數的程式照常編譯；`Estimator` 裡的 `Fit: ml.FitLinearRegression` 則不能編譯，因為函式值的回傳型別必須完全相同，請改寫成 `Fit: func(x *insyra.DataTable, y *insyra.DataList) (ml.Model, error) { return ml.FitLinearRegression(x, y) }`。另外兩種寫法也不能編譯：對回傳值做型別斷言（例如 `model.(*ml.LinearModel)`），現在不需要了，直接拿掉；以及用同一個 `:=` 變數先後接兩個不同 `Fit*` 函式的結果，請改宣告成 `var model ml.Model`。（[issue #263](https://github.com/HazelnutParadise/insyra/issues/263)）
- `ml` 的 `Fit*` 函式回傳的每種型別，模型是 nil 時每個方法都能安全呼叫，回傳錯誤或空值：`Predict`、`PredictProba`、`Transform`、`ExportONNX` 回傳錯誤，`Features`、`FeatureImportances`、`LeafValues` 回傳 nil，`Clusters` 回傳 0，`Classes` 回傳帶著錯誤的空 list。在宣告回傳 `ml.Model` 的 closure 裡 fit 失敗時，拿到的是裝著 nil 指標、本身卻不是 nil 的介面。以前對這種模型呼叫 `Features`，除了 `PCATransformer` 每種型別都會 panic，有 `ExportONNX` 的型別呼叫它也會，十個迴歸與 logistic 模型的 `Predict` 同樣會；`ml.ExportONNX(w, (*ml.LinearModel)(nil))` 同樣會 panic。
- **BREAKING**：`ml.ExportONNX(w, fitted)` 的參數從 `any` 改成 `ml.Model`，傳入不是模型的值（本來就一律被拒絕）現在無法編譯。它能匯出的每個模型，包括 fit 好的 pipeline，都是 `Model`；傳入 nil 模型會回傳錯誤。同一個函式的第二個名字 `ml.WriteONNX` 已 Deprecated，移除前仍接受 `any`。`DecisionTreeOptions` 的兩個別名 `DecisionTreeClassifierOptions` 與 `DecisionTreeRegressorOptions` 也已 Deprecated。這些 Deprecated 名稱會在下一版移除。（[issue #264](https://github.com/HazelnutParadise/insyra/issues/264)）
- `nn` 每樣東西只留一個名字。以下名稱已 Deprecated，意義不變，下一版移除：層建構子的十二個 `New…` 雙胞胎（`NewDense`、`NewReLU`、`NewDropout`、`NewFunc`、`NewMultiHeadAttention`、`NewConv2D`、`NewMaxPool2D`、`NewAvgPool2D`、`NewGlobalAvgPool`、`NewBatchNorm2D`、`NewLayerNorm`、`NewEmbedding`，改用不加 `New` 的名字）；loss 選擇器 `SoftmaxCrossEntropy`、`MSELoss`、`BCEWithLogitsLoss`（改用 `CrossEntropy`、`MSE`、`BCEWithLogits`；同名的 `Tape` 方法不受影響）；`Classifier` 與 `Regressor`（改用 `BoundClassifier`、`BoundRegressor`）；`MaxPoolOptions` 與 `AveragePoolOptions`（改用 `PoolOptions`）；`DataType` 與常數 `Float32`、`Float16`、`Float64`（改用 `DType` 與 `DTypeFloat32`、`DTypeFloat16`、`DTypeFloat64`）；`NewFloat32Tensor` 與 `NewTensorWithDType`（改用 `NewTensor`）；以及 `Tensor.Data`，它遇到不是 float32 的張量會回傳 nil（改用 `Tensor.Float32Data`，它會把這種情況回報成錯誤）。`NewSigmoid`、`NewTanh`、`NewGelu`、`NewFlatten` 維持原名，因為不加 `New` 的名字是核心運算函式。（[issue #265](https://github.com/HazelnutParadise/insyra/issues/265)）
- **BREAKING**：`nn.LayerNorm(dim int)` 改收最後一維的大小，新增的 `nn.LayerNormShape(dims []int)` 正規化結尾的多個維度，對應 torch 的 `normalized_shape` 清單。`LayerNorm` 原本收 `interface{}`，傳入既不是 `int` 也不是 `[]int` 的值（例如 `int64(4)`）會建出一個到 `Build` 才失敗的層，錯誤訊息是 `layernorm dimensions must be positive, got [0]`。`LayerNorm(16)` 照常編譯；`LayerNorm([]int{2, 3})` 請改成 `LayerNormShape([]int{2, 3})`。（[issue #265](https://github.com/HazelnutParadise/insyra/issues/265)）
- 新增 `Sequential.FitContext(ctx, x, y, cfg)`，訓練可以取消。每個 batch 開始前都會檢查 context；context 結束時回傳 `ctx.Err()` 與已經跑完的 epoch，模型保留已經做過的優化步驟。`Fit` 就是用 `context.Background()` 呼叫的 `FitContext`。（[issue #266](https://github.com/HazelnutParadise/insyra/issues/266)）
- 新增 `nn.CustomLoss{Name, Loss, Validate}`，`Sequential.Fit` 可以用呼叫者在 tape 上算的 loss 訓練，不再只限 `CrossEntropy`、`MSE`、`BCEWithLogits`。`Loss` 用 tape 上的運算算出這個 batch 的損失，回傳 float32 純量；`Validate` 選填，用來檢查每個 batch 的目標值；`Name` 會標在錯誤訊息裡。`Fit` 會在跑任何 batch 之前拒絕沒有 `Loss` 的 `CustomLoss`；`Loss` 回傳 nil、不是 float32 純量，或是在 tape 之外算出的值（否則參數拿到的梯度全是零）時，也會回傳錯誤。（[issue #266](https://github.com/HazelnutParadise/insyra/issues/266)）

### `datafetch`
- 檔案版 geocode 快取（`NewFileGeocodeCache`）改為先寫暫存檔再 rename，寫入中斷不再留下損壞、下次執行被靜默丟棄的快取檔。
- `GetReviews` 收到超過一個設定包時，會記錄問題並在發出任何請求前回傳 nil，不再改用預設設定抓取。
- 建構子回傳的客戶端改為匯出型別：`TWStock` 回傳 `*TWStockClient`，`YFinance` 回傳 `*YFinanceClient`，它的 `Ticker` 回傳 `*YFTicker`，`TWGeocoding` 回傳 `*TWGeocodingClient`，`GoogleMapsStores` 回傳 `*GoogleMapsStoresClient`。現在可以把客戶端存進自己的 struct 欄位或當成參數傳遞，既有的呼叫不必修改。不是由建構子建立的客戶端，例如建構子出錯時回傳的 `nil` 或零值，呼叫任何方法都會回傳錯誤（Google Maps 客戶端則回傳 `nil` 並記錄警告）。過去 `nil` 的 TWSE／TPEx 或地理編碼客戶端會直接 panic。
- **BREAKING**：`YFHistoryParams` 改為 `datafetch` 自己宣告的結構，不再是 go-yfinance `models.HistoryParams` 的別名，欄位、JSON tag 與意義都不變，其中 `RepairOptions` 改為 `*YFRepairOptions`。`News` 的分頁參數改為 `YFNewsTab`（`YFNewsTabNews`、`YFNewsTabAll` 或 `YFNewsTabPressReleases`），不再是 go-yfinance 的 `models.NewsTab`，其他值會在發出請求前回傳錯誤，而 go-yfinance 過去遇到不認得的分頁會照樣抓新聞。空字串的分頁仍然代表新聞。`YFHistoryParams{Period: "1mo"}` 這類寫法不必修改，傳入 go-yfinance 型別、或只為了指定分頁而 import go-yfinance 的程式碼，要改用 `datafetch` 的型別。`datafetch` 的匯出宣告不再出現 go-yfinance 的型別，升級這個相依套件不會再改變 insyra 的 API。
- 依「每個東西只有一個名字」的規則整理名稱：`YFPeriodYearly` 改為 Deprecated，請改用抓取相同財報的 `YFPeriodAnnual`，移除前它產生的表格仍標示 `yearly`。評論排序常數改為 `GoogleMapsStoreReviewSortByRelevance`、`GoogleMapsStoreReviewSortByNewest`、`GoogleMapsStoreReviewSortByHighestRating`、`GoogleMapsStoreReviewSortByLowestRating`，`SortByRelevance` 等四個舊名改為數值相同的 Deprecated 常數。`GoogleMapsStoreReviewsFetchingOptions.MaxWaitingInterval` 以 `time.Duration` 設定翻頁之間最長的等待，規則與原本的毫秒欄位相同，`MaxWaitingInterval_Milliseconds` 改為 Deprecated，兩個都設定時 `GetReviews` 會在發出請求前回傳 `nil` 並記錄警告。`TWGeocoding` 的 `ReverseTable` 改為以函式庫的欄位選擇器指定欄位，`ReverseTable(dt, insyra.Name("lat"), insyra.Name("lng"))` 就是以名稱選欄，`ReverseTableByColName` 改為 Deprecated。原本傳兩個字串的呼叫仍把字串當成 Excel 式欄位索引。這些 Deprecated 名稱會在下一個版本移除。
- 每個抓取方法都有接收 `context.Context` 的 `...Context` 版本，做法與 `ReadSQLContext` 相同，原本的方法改為以 `context.Background()` 呼叫它：`TWStockClient` 的六個方法（例如 `DailyPricesContext`）、`ReverseContext`、`ReverseColsContext`、`ReverseTableContext`、`SearchContext`、`GetReviewsContext`，以及 `YFTicker` 所有會發出請求的 26 個方法（例如 `HistoryContext`）。context 會傳到限流等待、重試前的退避、每個 HTTP 請求與評論翻頁之間的等待。被取消的呼叫回傳 `ctx.Err()` 且不重試。`ReverseContext` 的請求若被 context 中斷，回報的是 context 的錯誤而不是 `ErrGeocodeTimeout`，`ErrGeocodeTimeout` 仍只代表服務沒在 `Timeout` 內回應。`ReverseCols` 批次中途被取消時，會回傳已解析的列，其餘標為 `pending`，與配額用完時相同。go-yfinance 無法中途停止已經開始的呼叫，所以 Yahoo Finance 的方法在 context 結束時會立刻返回，那次呼叫則留在背景跑完，期間仍可能送出剩下的請求。

### `stats`
- **BREAKING**：`FactorAnalysis` 在 `Rotation.Method: FactorRotationOblimin` 且 `Rotation.Restarts` 大於 1 時，現在真的會做參數所描述的搜尋。Oblimin 過去每一輪都自己建一個單位矩陣當起點，忽略傳進來的起點，所以 `Restarts: 20` 是把同一份計算跑 20 次再回傳第一個結果，實測 20 個起點花 202 ms 回傳和 9.9 ms 一模一樣的答案。現在它和其他九個方法一樣從每個起點各旋轉一次，而且 `Delta: 0` 時與同樣 `Restarts` 的 Quartimin 結果逐位元相同，因為兩者本來就是同一個準則。在測試套件的 12 個生成資料集上，載荷最多移動 1.2e-5；在過因子的模型上（用三因子資料抽四個因子），額外的起點會找到單位矩陣起點到不了的另一個盆地，準則值低 47 倍。那個解的因子相關也高達 0.905：`Restarts` 要的是最低的準則值，不保證那個解就是你要的，這對任何多起點的斜交旋轉都成立。預設的 `Restarts: 1` 不變。SPSS 的 Direct Oblimin 是單起點，`Restarts: 1` 仍然是。
- **BREAKING**：`DefaultFactorAnalysisOptions()` 的 `Rotation.Restarts` 改為 20，沒有設定（為 0）時也視為 20。這跟隨 psych 2.6.5，它在遇到一個單起點停在局部最小值的真實案例後，把 `fa()` 的 `n.rotations` 預設從 1 改成 20，`fungible::faMain` 也基於同樣理由預設 10。起點之間的挑選規則不變，並正式記為決定：取收斂的起點中準則值最小者，與 `GPArotation` 的引擎及 `fungible` 相同。psych 改用 hyperplane count 排序，但在所有實測資料上兩種規則選到同一個解。在測試套件的 20 個資料集上，預設載荷最多移動 1.2e-5，從未換到不同的解。多起點搜尋用來當資訊起點的 Varimax 改以旋轉本身的容忍度執行（`eps = 1e-5`、`maxit = 1000`），不再用 `1e-8`／`5000`，過去它在其中五個資料集上跑到迭代上限，每次花 80 ms，現在整個 20 起點搜尋在這些資料集上花 1.5 到 9 ms。要 SPSS 與 `GPArotation` 預設的單一單位矩陣起點，設 `Restarts: 1`。
- **BREAKING**：Promax 搭配 MINRES、ML、PAF 抽取時改為跟隨 psych 2.6.5。psych 的 `fa()` 經由 `kaiser()` 呼叫 `psych::Promax`，它的前置旋轉在 2.6.5 從 `stats::varimax` 換成 `GPArotation::Varimax`（原始碼註記「replaced with GPArotation Varimax 5/9/26」）；`principal()` 仍用 `stats::promax`，前置旋轉是帶 Kaiser 正規化的 `stats::varimax`。我們過去所有 Promax 都用 `stats::varimax` 那一套，所以因子抽取搭 Promax 得到的是 2.6.5 之前的答案。varimax 最佳解明確的資料上兩種前置旋轉只差收斂容忍度，但 parity 那張十列表的 varimax 準則接近平坦，兩者停在不同角度，Promax 再把差距放大到四次方，載荷 [0,0] 是 0.700 對 psych 的 0.625。以 psych 2.6.5 在 16 個資料集實測，MINRES、ML、PAF 現在最差差 6.4e-4，過去最差 0.29。PCA 抽取的結果不變，與 psych 差 4e-13。
- **BREAKING**：`FactorAnalysis` 的所有梯度投影旋轉，現在都照 GPArotation 2026.8.2 的預設演算法逐步計算，每個起點的迭代上限也從 1000 改成 2000。受影響的有 Varimax、Quartimax、GeominT、BentlerT、Quartimin、Oblimin、GeominQ、BentlerQ、Simplimax，以及因素抽取後 Promax 裡的 Varimax 步驟。這些旋轉原本照 GPArotation 的舊版步驟移植，但 GPArotation 已改用 `"bb"` 演算法作為預設，psych 2.6.5 也沿用這個預設。舊版步驟在準則值很平的資料上前進得很慢：兩張十列測試表抽兩個因子時，400 個隨機起點中，quartimin 只有 7 個和 31 個在 1000 次迭代內收斂，geomin 只有 0 個和 3 個，GPArotation 則 400 個全部在 200 次內收斂。原本停在上限的旋轉現在會收斂，所以載荷可能改變，`RotationConverged` 也可能變成 `true`。多起點搜尋完成收斂的起點變多，選出的解也可能不同。主成分分析後 Promax 使用的 `stats::varimax` 步驟不變。原本 Promax 搭配 MINRES 與 psych 2.6.5 最多差 5e-4 的那張十列表，現在差距在 1e-10 以內。
- **BREAKING**：`FactorAnalysis` 使用 `FactorRotationSimplimax` 時，現在最小化的是 psych 2.6.5 所用的 simplimax 準則。psych 呼叫 `GPArotation::simplimax` 時不指定 `k`，這個準則會把平方載荷由小到大排序，恰好取最小的 `k` 個相加，`k` 等於變數數乘以（因子數減一），平方載荷相等時依欄優先的位置決定取哪幾個。這裡原本的 `k` 是變數數，而且與第 `k` 小相等的平方載荷也會一起算入。因此在三個以上因子，或有相等平方載荷時，最小化的是另一個準則，回傳的旋轉也不同：以 6 個變數、3 個因子的載荷為例，起點的準則值是 0.125，GPArotation 是 0.245；旋轉後把準則壓到 3e-9，GPArotation 則停在 0.0218。兩個因子且沒有相等平方載荷時，結果不變。
- `PairedTTest`、`SingleSampleWilcoxon`、`PairedWilcoxon`、`MannWhitneyU`、`OneWayANOVA`、`TwoWayANOVA`、`RepeatedMeasuresANOVA`、`KruskalWallis` 與 `FriedmanTest` 改為拒絕 `NaN` 或 `±Inf` 的格子，這正是 `stats` 文件一直對每個數值入口的描述。它們原本只檢查格子能不能轉成數字，所以這些值會進入計算，而且錯誤是 nil：`PairedTTest` 和三種 ANOVA 回傳 `NaN` 的統計量與 p 值，排序類檢定則回傳看起來正常的結果，因為 `NaN` 一樣會被排出名次。`KruskalWallis` 對 `[1, 2, NaN, 4]` 與 `[1, 2, 3, 4]` 回報 H = 0.54、p = 0.46。現在錯誤訊息和其他檢定一致，list 與位置都從 1 起算，例如 `group 2 contains a non-finite value at row 3: NaN`、`cell (A=2, B=1) contains a non-numeric value at row 2: <nil>` 與 `subject 2 contains a non-finite value at condition 2: NaN`，成對與雙樣本檢定則用 `data1` 或 `data2` 指出是哪個 list。以前的訊息是沒有位置的 `invalid numeric value in data1`，或從 0 起算的 `invalid data at group 0 index 2`。`LeveneTest` 與 `BartlettTest` 的組號也改從 1 起算（以前的 `group 1` 指的是第二組），空組、空格與條件數不符的受試者錯誤也一樣。`nil` 的 list（不論是否帶型別）會得到空 list 會得到的錯誤。`OneWayANOVA`、`KruskalWallis` 與 `FriedmanTest` 以前遇到它會讓整個程式結束，因為問題發生在 `recover` 接不到的 goroutine 裡，`TwoWayANOVA`、`RepeatedMeasuresANOVA` 與 `SingleSampleWilcoxon` 則會 panic。全為有限數值的輸入結果不變。CLI 的 `ttest paired`、`anova` 與 `ftest levene|bartlett` 指令會印出新的訊息。
- 本身不是 `*insyra.DataList` 的 list（例如 `isr.DL` 建立的 list）現在會照原本存的樣子讀取。`stats` 以前會用 `NewDataList` 重建這種 list，而它會把一格 slice 拆成好幾個數字，所以 `SingleSampleTTest` 等會轉換輸入的函式把這一格算成多個觀察值，四格的 list 進到 `PairedTTest` 的長度檢查時也變成了五格。現在這一格會被拒絕，和同一格放在 `*insyra.DataList` 裡的結果一樣：`data contains a non-numeric value at row 4: [10 11]`。
- `CutTreeByK` 與 `CutTreeByHeight` 遇到合併、高度與標籤數量對不上，或合併對象既不是葉節點也不是先前合併的樹時，改為回傳指出問題的錯誤。高度比合併少的樹以前會 panic；手動建立或從檔案還原的樹可能是任何形狀，而 CLI 現在會在一次性指令之間還原 `hclust` 的樹。
- `Skewness` 與 `Kurtosis` 讀不了輸入時（包括 nil 的 `*DataList`），會回傳以 `sample:` 開頭的錯誤。以前 nil 的 list 會讓它們當掉，一般數字這類其他型別的值則被回報成 `empty data`。
- **BREAKING**：假設檢定的對立假設與信賴水準統一用同一種方式傳入，放在最後一個可省略的設定值裡。t 檢定原本的 `confidenceLevel ...float64` 改為 `opts ...TTestOptions`；z 檢定原本必填的 `alternative` 與 `confidenceLevel` 改為 `opts ...ZTestOptions`；`SingleSampleWilcoxon`、`PairedWilcoxon` 改收 `opts ...WilcoxonOptions`，`MannWhitneyU` 改收 `opts ...MannWhitneyUOptions`，取代原本必填的 `alt` 和選填的 `confidenceLevel`。每個設定型別都有 `Alternative` 與 `ConfidenceLevel` 兩個欄位，不給設定或欄位留 0，就是雙尾檢定加 95% 信賴區間，所以 `SingleSampleZTest(x, 100, 15)`、`MannWhitneyU(a, b)` 現在照字面就能用。`SingleSampleTTest(x, 50, 0.99)` 要改寫成 `SingleSampleTTest(x, 50, stats.TTestOptions{ConfidenceLevel: 0.99})`，`SingleSampleZTest(x, 100, 15, stats.Greater, 0.95)` 則改成 `SingleSampleZTest(x, 100, 15, stats.ZTestOptions{Alternative: stats.Greater})`。0 以外、不在 (0, 1) 之間的信賴水準一樣會回傳錯誤，`NaN` 也是：它以前會被放行，t 檢定與 Wilcoxon 檢定照樣回報 95% 的區間，z 檢定則回報 `[NaN, NaN]`，錯誤都是 nil。無法辨識的對立假設、給了兩個設定值，也都會回傳錯誤。舊寫法算得出的結果全部不變。
- t 檢定可以做單尾檢定。`TTestOptions{Alternative: stats.Greater}` 或 `stats.Less` 會給出和 R `t.test(..., alternative = "greater")` 相同的 p 值與單尾信賴區間，開放的那一端是 `+Inf` 或 `-Inf`，和 z 檢定原本的回報方式一樣。三種 t 檢定在兩個方向、兩種信賴水準下，都和 R 的 `t.test` 及 SciPy 對照過。
- **BREAKING**：`OneWayANOVA`、`TwoWayANOVA`、`RepeatedMeasuresANOVA`、`KruskalWallis` 與 `FriedmanTest` 改以一個 `[]insyra.IDataList` 接收各組、各格或各受試者，和 `LeveneTest`、`BartlettTest` 原本的寫法一致。`OneWayANOVA(a, b, c)` 要改成 `OneWayANOVA([]insyra.IDataList{a, b, c})`，`OneWayANOVA(groups...)` 改成 `OneWayANOVA(groups)`。Go 只允許最後一個參數是可變長度，各組分開傳入時，這些檢定永遠沒辦法在最後加上設定值；改成 slice 之後，日後要加上 Levene 檢定的中心值、Welch 單因子變異數分析這類設定，就不必再讓所有呼叫改寫一次。
- `FactorAnalysis` 的設定改為可省略的 `opts ...FactorAnalysisOptions`，`FactorAnalysis(dt)` 會直接使用 `DefaultFactorAnalysisOptions` 的預設值。傳入一個設定值的既有呼叫不用改；把 `stats.FactorAnalysis` 存進舊函式型別變數的程式則要改型別。
- 每個假設檢定的結果都公開了彼此共用的部分。存放 `Statistic`、`PValue`、`DF`、`CI` 與 `EffectSizes` 的結構匯出為 `TestResult`，新增的介面 `HypothesisTestResult` 只有一個方法 `Base() *TestResult`，`TTestResult`、`ZTestResult`、`FTestResult`、`ChiSquareTestResult`、`CorrelationResult`、`WilcoxonTestResult`、`MannWhitneyUResult`、`KruskalWallisResult` 與 `FriedmanTestResult` 都符合它。現在同一個函式或同一個 slice 可以同時接收 t 檢定、Mann-Whitney U 檢定和卡方檢定的結果，用 `r.Base().PValue` 讀取；這個結構以前沒有匯出，所以根本沒有型別可以寫進參數。`r.PValue` 這類欄位存取不變。
- **BREAKING**：只有部分檢定會填的結果欄位一律改成指標，不適用時為 `nil`；每個檢定都會填的欄位則是一般值。原本同一件事有的用指標、有的用 `NaN`、有的用 0 表示。`TTestResult.Mean` 改成 `float64`，和 `ZTestResult.Mean` 一樣；`PairedTTest` 以往把 `Mean`、`Mean2`、`N2` 留成 `nil`，現在分別填入 `data1` 與 `data2` 的平均數及成對數。`WilcoxonTestResult.Z` 與 `MannWhitneyUResult.Z` 改成 `*float64`，走精確分布時為 `nil`，不再是 `NaN`。`FTestResult.DF2` 改成 `*float64`，`BartlettTest` 的 `DF2` 為 `nil`；它以前回報 0，而 0 是可能被拿去代入 F 分布的數字。t 檢定的 `*r.Mean` 要改寫成 `r.Mean`，讀 `Z`、`DF2` 的地方改成 `*r.Z`、`*r.DF2`。把 `r.Z` 或 `r.DF2` 直接交給 `fmt` 仍然能編譯，但印出來的會是指標。各檢定回報的數值全部不變。
- **BREAKING**：`TwoSampleTTest` 的變異數假設改放在 `TTestOptions` 的 `EqualVariance`，不設定就執行 Welch t 檢定，和 R `t.test` 的預設相同。位置參數 `equalVariance bool` 已移除：`TwoSampleTTest(a, b, false)` 改成 `TwoSampleTTest(a, b)`，`TwoSampleTTest(a, b, true)` 改成 `TwoSampleTTest(a, b, stats.TTestOptions{EqualVariance: true})`。bool 放不進設定值的位置，所以舊寫法會直接編譯失敗，不會默默改變意思；改寫後的呼叫結果和以前相同。兩組變異數真的相等時，Welch 檢定和 Student 檢定的結果幾乎一樣；不相等時 Welch 仍然可靠，所以不設定時用它。
- 新增 `DiagOf`、`DiagMatrix`、`DiagMatrixSize` 與 `IdentityMatrix`，建立與讀取對角線時，傳入與回傳都只有一種型別：`DiagOf(m mat.Matrix) ([]float64, error)` 回傳矩陣的對角線，`DiagMatrix(v)` 把 `v` 放在方陣的對角線上，`DiagMatrixSize(v, nrow, ncol)` 可以指定任意形狀，`IdentityMatrix(n)` 則是單位矩陣。原本一個函式包辦這四件事、呼叫端每次都得自己做型別斷言的 `Diag(x any, dims ...int) (any, error)` 改標為 **Deprecated**，下一版移除，它的 doc comment 與 `Docs/stats.md` 列出每種寫法對應的新函式。`v` 比對角線長時，`DiagMatrixSize` 會回傳錯誤，`Diag` 則是直接截掉。`Diag(0)`、`Diag([]float64{})` 與 `Diag(-1)` 以前會 panic，現在改為回傳錯誤。
- **BREAKING**：`ChiSquareTestResult` 改用 `Observed` 與 `Expected` 兩張表回報次數，取代 `ContingencyTable`。`ContingencyTable` 的每一格是 `[2]float64{observed, expected}` 陣列，程式庫裡沒有其他功能讀得了：對其中一欄呼叫 `Sum()` 會每格記一次警告並回傳 `NaN`，`Show` 則印出 `[5 4.5]`。新的兩張表列與欄都和 `ContingencyTable` 相同，每格是一般的 `float64` 次數，只有適合度檢定那一欄的名稱從 `Observed_Expected` 改成 `Observed` 或 `Expected`。原本讀 `pair[0]` 的程式改讀 `res.Observed` 的同一格，`pair[1]` 改讀 `res.Expected`。統計量、p 值與自由度都不變，兩個檢定現在也直接和 R 的 `chisq.test` 比對。
- **BREAKING**：`ChiSquareGoodnessOfFit` 的期望機率改以 `map[string]float64` 傳入，鍵是類別名稱。原本的 slice 是照類別名稱排序後的順序對位，所以照紅、綠、藍的順序寫 `[]float64{0.5, 0.3, 0.2}`，實際檢定的是藍色 0.5、紅色 0.2，而且不會報錯。現在寫成 `map[string]float64{"red": 0.5, "green": 0.3, "blue": 0.2}`。鍵不是輸入中的類別時會回傳錯誤並指出是哪一個，類別沒有對應的鍵也會回傳錯誤。傳 `nil` 和以前一樣，每個類別的機率相等。
- 二因子 ANOVA、重複量數 ANOVA 與 Friedman 檢定可以直接讀每列一個觀察值的表格。`TwoWayANOVAFromTable(dt, valueCol, factorACol, factorBCol)`、`RepeatedMeasuresANOVAFromTable(dt, valueCol, conditionCol, subjectCol)` 與 `FriedmanTestFromTable(dt, valueCol, conditionCol, subjectCol)` 接受實驗資料 CSV 通常的樣子，也就是 R 的 `aov` 與 `friedman.test` 讀的形狀，不必再自己把資料切成 row-major 的格子，或每位受試者一個 list。欄位照一般的欄位選擇器指定。因子欄的相異值就是它的水準，比較方式和 `GroupBy` 比較分組鍵相同。水準是 `nil` 或 `NaN`、數值不是有限數字、某個水準組合沒有觀察值、受試者缺了某個條件或同一條件量了兩次，都會回傳錯誤並指出是哪一列或哪些水準。結果和以同樣資料呼叫 `TwoWayANOVA`、`RepeatedMeasuresANOVA`、`FriedmanTest` 相同，也以打亂列順序的資料和 R 的 `aov`、`friedman.test` 比對過。`Docs/stats.md` 也補上原本沒有的 `RepeatedMeasuresANOVA` 章節。
- `stats` 的每個結果型別都能印出自己。每個型別都有 `String()`，`fmt.Println(res)`、寫進 log，或用 `fmt.Fprintln(w, res)` 寫到任何 `io.Writer`，都會得到完整的結果；`Show()` 則把 `String()` 原樣印到標準輸出。以前只有 `ChiSquareTestResult` 與 `FactorAnalysisResult` 有 `Show`，其他型別印出來是 Go 原始的 struct 格式，指標只看得到位址。現在所有結果都照同一種版面：第一行是分析名稱，接著每個欄位一行、用 Go 的欄位名稱，值為 `nil` 的欄位不印，數字照 insyra 輸出文字的規則寫，清單與表格超過 60 筆時和表格檢視一樣只列頭尾，表格畫成格線。輸出不含色碼。**原有的兩個 `Show` 也改用這個版面**，`FactorAnalysisResult.Show` 給了列範圍時，仍照該範圍顯示每張表。
- **BREAKING**：卡方檢定的類別名稱改照 insyra 把任何值寫成文字的規則，也就是 `ToStringSlice` 用的規則。以前用 `fmt` 的 `%v`，所以浮點數類別 1,500,000 會標成 `1.5e+06`、0.00001 會標成 `1e-05`，現在是 `1500000` 與 `0.00001`。這些標籤是 `Observed` 與 `Expected` 的列名與欄名，也是 `ChiSquareGoodnessOfFit` 的 `p` 要寫的鍵，所以寫成 `"1.5e+06"` 的鍵現在會被當成不存在的類別而回傳錯誤，請改寫成 `"1500000"`。字串、整數、`nil`，以及介於 0.0001 到一百萬之間的浮點數，標籤都不變。`ChiSquareIndependenceTest` 也改在同一個 `AtomicDoAll` 內讀取兩個 list，和雙樣本檢定一樣，呼叫端同時改變兩個 list 的長度時，不會再讀到兩個時點混在一起的資料。

### `csvxl`
- **BREAKING**：`ExcelToCsv` 與 `EachExcelToCsv` 會用 v0.3.3 檢查工作表名稱的同一套規則檢查 `csvNames` 指定的檔名：含路徑分隔符號、或不會直接落在輸出目錄內的檔名都會被拒絕，所以 `csvNames: []string{"sub/out.csv"}` 不再寫進子目錄。每張 CSV 先寫入暫存檔再 rename 到目標位置，寫入失敗時不會留下截斷的檔案。
- `ExcelToCsv` 遇到 `onlyContainSheets` 裡工作簿沒有的名稱會回報錯誤，並列出檔案實際有哪些工作表。名稱拼錯過去會被靜默略過，轉出來的檔案少了幾張，看起來卻像成功。
- **BREAKING**：`CsvToExcel`、`AppendCsvToExcel` 與 `EachCsvToOneExcel` 會指出哪些 CSV 失敗，也不再留下損壞的工作表。過去錯誤只寫「2 files failed to convert」，每個失敗的檔案都在工作簿裡留下一張空工作表，`AppendCsvToExcel` 甚至先清空同名的既有工作表，才發現 CSV 讀不到，接著照樣存檔。現在每個 CSV 會先完整讀完，才建立或取代工作表。失敗的檔案不產生工作表，既有工作表保留原內容，其他檔案照常轉換並存檔。錯誤逐行列出每個失敗的檔案與原因，`errors.Is(err, os.ErrNotExist)` 也能用。Excel 不接受的工作表名稱現在只讓那個檔案失敗，過去會讓整個呼叫在存檔前就中止。全部失敗時，`CsvToExcel` 不寫出工作簿，`AppendCsvToExcel` 不改動檔案。
- **BREAKING**：`ExcelToCsv` 的 `csvNames` 若已帶副檔名（不分大小寫）就照用，所以 `report.txt` 寫成 `report.txt`、`REPORT.CSV` 寫成 `REPORT.CSV`，過去會變成 `report.txt.csv` 與 `REPORT.CSV.csv`；沒有副檔名的名稱仍然補上 `.csv`。CLI 的 `convert` 跟著改，`insyra convert book.xlsx out.txt` 現在寫出 `out.txt`。讀取端 `CsvToExcel` 與 `AppendCsvToExcel` 先照原路徑開，原路徑不存在才補 `.csv`：名為 `export.txt`、`DATA.CSV` 或完全沒有副檔名的 CSV 過去都讀不到，因為任何不是以小寫 `.csv` 結尾的路徑都會被補上 `.csv`，現在都讀得到。過去讀得到的路徑全部照樣讀得到，唯一差別是 `x` 與 `x.csv` 同時存在時讀的是 `x`；兩個都不存在時，錯誤訊息會列出兩條試過的路徑。
- 編碼參數給空字串，或任何大小寫的 `"auto"`，現在都代表自動偵測，跟核心的 CSV 讀取函式一致。以前空字串代表直接當成 UTF-8，`"AUTO"` 則會被當成不支援的編碼而報錯。
- **BREAKING**：`ExcelToCsv` 與 `EachExcelToCsv` 跟 `ToCSV` 一樣，預設會防範公式注入：在活頁簿裡安全的文字，轉成 CSV 再用試算表打開時會被當成公式。要原樣寫出就用 `ExcelToCsvOptions{AllowFormulas: true}`。挑選工作表的清單也移進同一個設定包：`ExcelToCsv(file, dir, names, "2024", "2025")` 改成 `ExcelToCsv(file, dir, names, csvxl.ExcelToCsvOptions{Sheets: []string{"2024", "2025"}})`。
- **BREAKING（輸出）**：`AppendCsvToExcel` 改用全新的工作表取代既有的同名工作表，做法與 `DataTable.ToExcel` 相同，不再像 v0.3.3 那樣就地清空儲存格。就地清空只移除值與公式，列與儲存格上的其他東西都還在：新資料落在舊的隱藏列上，在 Excel 裡看不到，`ExcelToCsv` 讀回時也會跳過；舊的註解與超連結也留在新值上。舊工作表內容越多也越慢：取代一張有 2 萬個公式的工作表要 157 毫秒，現在是 6 毫秒。工作表仍保留原本的位置，原本隱藏的也維持隱藏，作用中的工作表不變，活頁簿其他地方的定義名稱與公式也都保留，但舊工作表的欄寬、檢視與合併範圍不再保留，只屬於這張工作表的定義名稱也會一起消失。名稱只差大小寫的工作表，現在會改用呼叫時給的名稱：附加到 `TARGET` 會把名為 `Target` 的工作表改名，v0.3.3 則維持原名。
- 函式名稱改成 Go 對縮寫的寫法，與 insyra 其他套件一致：`CSVToExcel`、`AppendCSVToExcel`、`ExcelToCSV` 與 `ExcelToCSVOptions`、`ReadCSVToString`，處理整個目錄的兩個函式則是 `CSVDirToExcel` 與 `ExcelDirToCSV`。舊名稱 `CsvToExcel`、`AppendCsvToExcel`、`ExcelToCsv`、`ExcelToCsvOptions`、`EachCsvToOneExcel`、`EachExcelToCsv` 與 `ReadCsvToString` 的行為與新名稱完全相同，標為 **Deprecated**，下一版移除。CLI 的 `convert` 不受影響。
- `CSVToExcel` 與 `AppendCSVToExcel` 會先檢查編碼名稱才讀檔，解碼表裡沒有的名稱會回傳一個指出該名稱的錯誤。過去要讀到 CSV 時才會發現：沒有任何檔案時（例如 `CSVDirToExcel` 遇到空目錄），不存在的編碼照樣被接受並寫出工作簿。有好幾個檔案時，同一個錯誤會每個檔案重複一次。

### `parquet`
- `Write` 先寫入暫存檔再 rename 到目標位置，中途失敗時不會留下截斷的 Parquet 檔。
- `Write` 把 `[]byte` 格子的欄位寫成 Arrow `Binary` 欄，用 `Read` 讀進來的二進位欄再寫回去，位元組與型別都能保住。v0.3.3 會把這種欄寫成字串欄，每格存的是 Go 對該 slice 的文字表示，例如 `[65 45 48 49]`。
- **BREAKING**：`ApplyCCL` 的賦值目標與 `DataTable` 的規則一致。裸目標只當欄位字母（`A`、`B`、... `AA`），過去在字母落到範圍外時會退回同名欄位，於是 `score = A * 2` 在有 `score` 欄的檔案上這邊成功、那邊失敗。請寫 `['score'] = A * 2`，而且檔案真的有同名欄位時，錯誤訊息會告訴你。
- **BREAKING**：`Stream` 改回傳 `iter.Seq2[*insyra.DataTable, error]`，不再回傳兩個 channel，用法變成 `for dt, err := range parquet.Stream(ctx, path, opt, 1000)`。迴圈中途離開會一併停止讀取。過去用 channel 時，呼叫端 break 出迴圈又沒 cancel context，負責讀檔的 goroutine 會卡在下一次送資料直到程式結束，而文件自己的範例還讓兩個 channel 互相競爭。每一批資料都帶 nil 錯誤；失敗只出現一次，以 nil 表格加錯誤的形式結束迴圈；cancel `ctx` 時以 context 的錯誤結束。
- `ReadFrom(ctx, r io.ReaderAt, size, opt)`、`StreamFrom(ctx, r, size, opt, batchSize)` 與 `WriteTo(dt, w io.Writer)` 可以讀寫不在硬碟上的 Parquet，例如 S3 物件或記憶體中的位元組；`Read`、`Stream`、`Write` 現在都呼叫它們。讀取需要 `io.ReaderAt` 與大小，因為 Parquet 把索引放在檔案結尾。`WriteTo` 不會關閉呼叫端給的 writer。
- `Write` 與 `WriteTo` 可以多給一個 `WriteOptions`：`Compression`（預設 `CompressionNone`，另有 `CompressionSnappy`、`CompressionGzip`、`CompressionBrotli`、`CompressionZstd`）與 `RowGroupSize`，也就是每個 row group 最多幾列（預設 1,048,576）。不給設定時，寫出的檔案與過去逐位元組相同。未知的壓縮格式、負的 row group 大小或給了兩個設定包，都會在寫入任何東西之前回傳錯誤。`WriteContext` 與 `WriteToContext` 接受 `context.Context`，取消時會在欄與欄、row group 與 row group 之間停下。`WriteContext` 取消後，原路徑上的檔案維持原樣。傳入 nil context 會回傳錯誤，不會 panic。
- `Write` 改用名稱各自不同的暫存檔，不再用固定的 `<path>.tmp`。用固定名稱時，同時對同一路徑寫兩次會共用同一個暫存檔：一邊可能在改名時失敗，另一邊卻回報成功，留下混著兩邊位元組的檔案；呼叫端自己放在 `<path>.tmp` 的檔案也會被覆寫後刪掉。
- `Read`、`ReadFrom`、`ReadColumn`、`Inspect`、`Stream`、`StreamFrom`、`FilterWithCCL` 與 `ApplyCCL` 遇到 Arrow 讀取器無法解讀的檔案時，會回傳錯誤說明它不是可讀的 Parquet 檔，不再讓讀取器的 panic 直接傳到你的程式。資料頁與 footer 來自不同次寫入的檔案，在試過的 655 個裡有 169 個會讓 `ReadFrom` 因 nil pointer 而 panic。`ApplyCCL` 遇到這種檔案時不會改動原檔。
- `ApplyCCL` 改寫檔案時，會沿用每一欄原本的壓縮格式，row group 也維持原檔最大的那一組的大小；腳本新增的欄位則採用第一欄的壓縮格式。過去一律寫成未壓縮、每 1,000 列一組：一個 20 萬列、用 Zstd 壓縮的檔案加一欄後，從 1.7 MB 變成 9.3 MB，分成 200 組。也可以像 `Write` 一樣傳入 `WriteOptions` 自行指定這兩項。另外改用名稱各自不同的暫存檔，不再用 `<path>.tmp`，所以使用者放在那個名稱的檔案不會被動到。
- `FilterWithCCL` 與 `ApplyCCL` 因錯誤提早返回時，會停止讀檔。過去負責讀檔的 goroutine 會卡在送下一批資料，檔案也一直開著，直到 context 結束為止；用 `context.Background()` 的話，就是一直到程式結束。
- **BREAKING**：`FilterWithCCL` 與 `ApplyCCL` 的結果，與先用 `Read` 把檔案讀成表格再算同一個運算式相同。它們每次讀 1,000 列，過去每批各自計算，所以彙總只看得到自己那一批，`#` 每 1,000 列從 0 重新算起，`A.0` 指的是每一批的第一列：某欄是 1 到 2,500 時，`A > AVG(A)` 保留的是 501 以後的列，而不是 1,251 以後，`"# == 0"` 則保留第 1、1,001、2,001 列。現在 `SUM`、`AVG`、`COUNT`、`MIN`、`MAX`、`VAR`、`VARP`、`STDEV` 與 `STDEVP` 會在逐列計算前再讀一次檔案，以整個檔案算出來，記憶體裡只留累計值；`#` 是該列在檔案中的位置；固定列或列範圍讀的是檔案中的那些列。`ApplyCCL` 的每一句看到的，是前面幾句套用後的檔案，與 `ExecuteCCL` 相同：過去讀取前面 `NEW` 建立的欄會出現 `column name 'c' not found` 錯誤，讀取前面賦值改過的欄則拿到檔案原本的值。某一列算不出來時，錯誤訊息標的是它在檔案中的列號。用不到這些的運算式跟以前一樣只讀一次；其他情況要多讀幾次，寫在 `Docs/parquet.md`。
- `FilterWithCCL` 與 `ApplyCCL` 遇到 `MEDIAN`、用 `engine/ccl` 的 `RegisterAggregateFunction` 註冊的彙總函式、由目前這一列算出來的列參照（例如 `A.(# - 1)`）、藏在彙總參數運算式裡的欄位範圍（例如 `COUNT(IF(A > 0, A:B, 0))`），以及不是整個運算式本身的序列函式（例如 `LAG(CUMSUM(A), 1)`、`SUM(CUMSUM(A))`）時，會只把運算式讀到的那幾欄整欄讀進記憶體，照 `AddColUsingCCL` 與 `ExecuteCCL` 的方式計算，答案與載入成表格時相同，失敗時的原因也相同。過去它們會不聲不響地逐批算出結果。`A > MEDIAN(A)` 只會讀進 `A` 這一欄；這類運算式若用到 `@`，就會讀進整個檔案。
- 序列函式 `LAG`、`LEAD`、`DIFF`、`PCT_CHANGE`、`CUMSUM`、`CUMPROD`、`CUMMAX`、`CUMMIN` 與 `ROLLING_*` 是整個篩選運算式，或是 `NEW`、賦值的整個右邊時，`FilterWithCCL` 與 `ApplyCCL` 會以整個檔案計算：`NEW('c') = CUMSUM(A)` 寫出的是整個檔案的累計和。過去每一批各自算出序列，再以文字寫進那一批的每一格。序列函式本身不會多讀檔案，記憶體只留跨批需要的部分：位移或視窗往回需要的值、累計函式的累計值，以及 `LEAD(x, n)` 往後的 `n` 列。
- **BREAKING**：`ApplyCCL` 寫出被賦值的欄時，新值若都能用原本的型別無損表示就保留原型別，否則就和腳本新建的欄一樣，用 `Write` 依整欄的值會給的型別；每個被敘述寫入的欄都標成可為 null。過去小數賦值到整數欄會被截斷，`['B'] = B / 2` 把 3.5 寫成 3，現在會把該欄放寬成 `float64`。`NEW` 建立的欄只看前 1,000 列決定型別，所以前面全是缺值的欄（例如 `ROLLING_MEAN(A, 1500)`）會被寫成文字，之後出現別種型別的值則以 schema 不符的錯誤中止。新欄的缺值會寫成 0，`[1, nil, 3]` 的欄跑 `NEW('c') = A` 會得到 `[1, 0, 3]`。檔案裡的 `time.Time` 或 `[]byte` 欄，或無法轉換的值，會讓 `ApplyCCL` panic。檔案裡只要有其他型別的欄（例如 `int32`、`float32`、日期、decimal、list 或 struct），就算腳本沒碰那一欄，整個呼叫也會失敗，更早之前則會 panic；現在沒被寫到的欄會直接沿用檔案原本的資料寫回，型別與每個值都不變，被賦值的 `int32`、`float32` 或日期欄，新值放得進去時也會保留原型別。
- `ApplyCCL` 的 `NEW('r') = @` 讓每一列寫入自己那一列的值。過去每批 1,000 列裡的每一格，寫入的都是該批最後一列。
- 檔案中某個 row group 損壞時，`Read`、`ReadFrom`、`ReadColumn`、`Stream`、`StreamFrom`、`FilterWithCCL` 與 `ApplyCCL` 會回傳錯誤，指出是哪個檔案、哪些 row group 沒能完整讀出。本套件過去使用的 Arrow 讀取器會把解不開的頁標頭當成那個 row group 的結尾，接著讀下一組，所以過去它們會不報錯地回傳其他組的列，`ApplyCCL` 還會用這些列覆寫原檔：以每組 1,000 列存成 3,000 列的檔案，第三組損壞時最後 1,000 列就此遺失，第二組損壞時中間 1,000 列被悄悄略過。現在 `ApplyCCL` 遇到這種檔案不會改動它。
- `parquet` 改用 `github.com/apache/arrow-go/v18` v18.8.0 讀寫，取代 `github.com/apache/arrow/go/v17` v17.0.0，公開的名稱都沒有改變。壓縮頁損壞時會回傳錯誤：使用 Snappy 時，Arrow v17 會在它自己開的 goroutine 裡以 `snappy: corrupt input` panic，任何 `recover` 都攔不到，所以 `Read`、`Stream`、`FilterWithCCL` 與 `ApplyCCL` 會讓整個程式結束。現在解不開的頁由讀取器自己回報，它的錯誤會保留在本套件回傳的錯誤裡。

### `mkt`
- **BREAKING**：`CAI` 改成一般函式，不再是存著 `CustomerActivityIndex` 的變數。呼叫方式不變，只有對它賦值會編譯失敗。
- **BREAKING**：`RFM` 讀金額的方式與這一版其他地方讀數字相同：`"150"` 這樣的數字字串是文字，不是數字，所以那一列會被略過並記一則指出它的警告。v0.3.3 會把這種字串當數字解析，因此從試算表讀進來、金額存成文字的表，現在會得到零列；請先轉換該欄，例如用 `ReadCSV` 的型別推斷或 `DataList.ParseNumbers`。

### `finance`
- **BREAKING（行為，簽名不變）**：所有接受 `opts ...Options` 的函式收到超過一個 `Options` 時會回傳錯誤。以前是默默採用最後一個，跟函式庫其他地方相反。

### `lpgen`
- 新增 `LPModel.WriteLP(io.Writer) error`，把 `GenerateLPFile` 存檔的同一份 CPLEX LP 文字寫進任何 writer，遇到不認識的目標型別時回傳錯誤。`GenerateLPFile` 改為透過它寫檔，輸出內容不變。
- **BREAKING**：`GenerateLPFile` 改為回傳 `error`，不再只記錄警告：檔案建立或寫入失敗、目標型別不是 minimize 或 maximize 時都會回傳錯誤。它先寫進暫存檔，整個模型寫完才取代目標檔，所以存檔失敗時不會留下檔案，也不會再毀掉既有的檔案。v0.3.3 會先清空既有檔案才檢查目標型別，最後檔案裡只剩開頭兩行說明文字。存檔方式與 `ToCSV` 相同：寫出權限為 0644 的新檔取代舊檔，所以既有檔案的權限不會保留，路徑若是符號連結，會被換成一般檔案，不會寫進連結指向的檔案。當作一般敘述呼叫的寫法仍可編譯，請記得檢查新的回傳值。
- `ParseLingo(text)` 與 `ParseLingoFile(path)` 讀取 LINGO 模型，回傳 `(*LPModel, error)`：檔案打不開或讀不了、某一行長達 64 KiB 以上，都會回傳錯誤，檔案不存在時可用 `errors.Is` 比對 `fs.ErrNotExist`。與舊函式不同，它們不會回傳缺了一部分的模型：認不得的敘述、讀不出變數的宣告，以及最後一句漏了 `;`，都會回傳錯誤並指出行號與該句，舊函式則是不吭一聲地丟掉。它們也會讀 `@GIN(x)`（一般整數）、`@FREE(x)`（可為負的變數）與 `@BND(l, x, u)`（範圍 `l <= x <= u`），舊函式會把這三種都丟掉，於是模型寫成 LP 後，可為負的變數變成了非負。以 `!` 開頭的 LINGO 註解會略過。`ParseLingoModel_str` 與 `ParseLingoModel_txt` 標為 **Deprecated**，下一版移除。移除前，失敗時照舊回傳 `nil` 並記錄警告。

### `lp`
- **BREAKING**：`lp` 改用以 Go 撰寫的求解器 [go-milp](https://github.com/daniel-sullivan/go-milp) 求解，不需要另外安裝任何程式。函式庫不再下載、編譯或安裝 GLPK，也不再修改程序的 `PATH`。`SolveFromFile(path, seconds)` 與 `SolveModel(model, seconds)` 由 `SolveFile(path, opts)` 與 `Solve(model, opts)` 取代，回傳 `(*lp.Solution, error)`。解答帶有 `Status`（`StatusOptimal`、`StatusFeasible`、`StatusInfeasible`、`StatusUnbounded` 或 `StatusStopped`）、`Objective` 與 `Values`。`Values` 是以完整精度讀出的 `map[string]float64`，舊版結果表放的則是 GLPK 報告的逐行文字。模型無解或無界時回傳對應的狀態，error 為 nil。回傳 error 表示沒有解答，可用 `errors.Is` 比對 `ErrInvalidModel`、`ErrEngineUnavailable` 或 `ErrSolverFailed`。`Options.TimeLimit` 的型別是 `time.Duration`。附加資訊表已移除，原本的執行時間、節點數與輸出改由 `Solution.Elapsed`、`Nodes` 與 `Log` 提供，`Solution.ToDataTable()` 則回傳 `Variable`／`Value` 兩欄的表。要繼續用 GLPK 求解，請先安裝（`Docs/lp.md` 列有各系統的安裝指令），再傳入 `lp.Options{Engine: lp.EngineGLPK}`，`lp` 會從 `GLPK_PATH` 或 `PATH` 找 `glpsol`。遷移範例：`result, info := lp.SolveModel(model, 10)` 改成 `sol, err := lp.Solve(model, lp.Options{TimeLimit: 10 * time.Second})`。

### `plot`
- **BREAKING**：`SavePNG` 預設不再退回線上渲染服務。不傳第三個參數（或傳 `false`）時，本機 Chrome／Chromium 渲染失敗會回傳錯誤；傳 `true` 才允許退回線上服務，該服務會把圖表連同資料上傳到 `server3.hazelnut-paradise.com`。過去的預設會在沒有詢問的情況下把使用者資料送出主機。
- **BREAKING**：`CreateRadarChart` 既沒有 `Indicators` 也沒有 `MaxValues` 時回傳錯誤；v0.3.3 則是記錄警告並回傳沒有 indicators 的圖表。
- `SaveHTML` 收到超過一個動畫旗標時會回傳錯誤，跟 `SavePNG` 對自己的選填旗標一樣，不再只讀第一個。
- **BREAKING**：熱圖的點型別改成公開的 `HeatMapPoint[X, Y]`，型別限制改成公開的 `HeatMapAxis`（以前拼成 `heapMapAxisValue`，而且沒有公開）。現在可以在迴圈裡把點收集成 `[]plot.HeatMapPoint[int, int]`。建立點的函式改名為 `NewHeatMapPoint` 與 `NewHeatMapMissingPoint`，`HeatMapPoint(x, y, v)` 要改成 `NewHeatMapPoint(x, y, v)`。
- **BREAKING**：每個 `Create...` 函式都改為回傳圖表與 `error`：`chart, err := plot.CreateBarChart(config, data)`。無法建立圖表時（通常是沒有給資料），會回傳 `nil` 圖表與開頭是函式名稱的錯誤，例如 `plot: CreateBarChart: no data to draw`；v0.3.3 則是記錄警告後回傳 `nil`。這種失敗不會再另外寫進紀錄。`CreateGaugeChart` 不會失敗，錯誤一律是 `nil`，保留這個回傳值是為了讓每個建構函式的呼叫方式都一樣。
- `CreateBoxPlot` 丟掉沒有資料清單的系列時，會在警告裡指出是哪一個系列；全部都被丟掉時，錯誤會說沒有任何系列有資料。v0.3.3 明明收到了系列，警告卻寫「no series provided」。

### `isr`
- `DT` 與 `DL` 都可用 `Err()`、`PopErr()`、`ClearErr()`、`SetErr()`；`ClearErr`／`SetErr` 回傳 isr 型別，積木語法不會斷在 `*insyra.DataTable`。
- **BREAKING（行為改變，簽章不變）**：`DT.From`、`Col`、`Row`、`UseDL`、`UseDT` 失敗時回傳帶著錯誤、可繼續串接的物件，錯誤記在 `Err()` 上，區塊語法不會因失敗而中斷：`t := isr.DT.From(isr.CSV{FilePath: p}); if err := t.PopErr(); err != nil { ... }`。v0.3.3 回傳的是包著 `nil` 表格或 list 的 `DT`／`DL`，所以 `t.DataTable == nil` 這類檢查不會再成立。
- **BREAKING**：`CSV_inOpts`、`CSV_outOpts`、`Excel_inOpts` 的標題列與列名欄位改成 `NoHeaderRow` 與 `HasRowNames`，跟核心的設定包一致。`FirstRow2ColNames: true` 現在是預設值，可以直接拿掉；`FirstRow2ColNames: false` 要改成 `NoHeaderRow: true`。型別名稱維持簡短的 `Opts` 寫法。
- isr 的表格與清單現在符合 `insyra.IDataTable` 與 `insyra.IDataList`，可以直接傳給 `stats`、`plot`、`mkt` 與 `Merge`：以前 `stats.PCA(isrTable)` 會編譯失敗，必須寫成 `isrTable.DataTable`。
- **BREAKING**：`isr.Pivot.Agg` 改為 `*insyra.AggregateOp`，和 `PivotConfig.AggFunc` 一致：`Agg: "sum"` 要改寫成 `Agg: new(insyra.OpSum)`。

### `gplot`
- **BREAKING**：`SaveChart` 檔案寫不出來時改為回傳 `error`，不再結束程式。既有呼叫要改成 `if err := gplot.SaveChart(...); err != nil { ... }` 或明確寫 `_ =`。
- **BREAKING**：每個 `Create...` 函式都改為回傳圖表與 `error`。無法建立圖表時，會回傳 `nil` 圖表與開頭是函式名稱的錯誤，例如 `gplot: CreateBarChart: the data list is empty`，不再只記錄警告並回傳 `nil`。
- **BREAKING**：建構函式改收 insyra 自己的型別，不再收 `any`，傳錯型別會在編譯時就失敗，不會等到執行時才回傳 `nil`。`CreateBarChart` 與 `CreateHistogram` 收 `insyra.IDataList`，`CreateLineChart` 與 `CreateStepChart` 收 `...insyra.IDataList`，`CreateHeatmapChart` 收 `insyra.IDataTable`，`CreateScatterPlot` 收 `...gplot.ScatterSeries`。`[]float64` 請改傳 `insyra.NewDataList(values)`；`map[string][]float64` 的每一筆改傳 `insyra.NewDataList(v).SetName(name)`；`[][]float64` 改傳 `insyra.ReadSlice2D(grid)` 產生的表格，比第一列短的列會補上 `nil`，畫成 0，比第一列長的列則會丟掉超出的值。以前用 map 傳入時，各系列的顏色與虛線樣式依 map 的走訪順序決定，每次執行都可能不同。
- **BREAKING**：散佈圖的一個系列改成兩個清單 `ScatterSeries{Name, X, Y}`；v0.3.3 把一個清單當成 x、y 交錯排列的值，多出來的最後一個值直接丟掉。`X` 與 `Y` 長度不同時會回傳錯誤，並指出是哪個系列。
- **BREAKING（行為改變）**：`CreateLineChart`、`CreateStepChart` 與 `CreateScatterPlot` 要嘛畫出所有系列，要嘛回傳錯誤。折線圖或階梯圖的系列長度和 `XAxis` 不同、系列是空的，或含有 `NaN` 或無限大，都會讓呼叫回傳 `nil` 圖表與錯誤，錯誤裡逐一寫出每個畫不出來的系列和原因。v0.3.3 會略過這種系列並記一則警告，把其他系列照樣畫出來；全部都被略過時，則回傳一張什麼都沒畫的圖。混在真實清單中的 `nil` 清單仍然只記警告並略過。
- **BREAKING（行為改變）**：`CreateHistogram` 拒絕 `NaN` 與無限大，`CreateHeatmapChart` 拒絕無限大以及只有 `NaN` 的表格，`CreateFunctionPlot` 拒絕不是有限值的 `XMin`、`XMax`、`YMin`、`YMax`，都會回傳指出是哪個值的錯誤。v0.3.3 遇到這些值時，依圖表與平台不同，有的會 panic，有的會卡住，有的會畫出錯誤的圖。熱圖裡夾在數字之間的 `NaN` 仍然畫成空白格。
- `CreateHeatmapChart` 能讀所有數值型別，`int8`、`int16` 或無號整數的欄位會畫出原本的值，不再全部畫成 0。它讀取儲存格的方式也和其他 `gplot` 圖表一致：不是數字的儲存格一律畫成 0，數字字串也一樣。
- **BREAKING（行為改變）**：`gplot` 做不到的設定改為回傳錯誤，不再只記警告。`CreateStepChart` 遇到不是 `"pre"`、`"mid"`、`"post"` 也不是空字串的 `StepStyle` 會回傳錯誤；v0.3.3 會把拼錯的 `"pr"` 當成 `"post"` 畫出來。`CreateBarChart` 的 `ErrorBars` 數量和長條數不同，或含有 `NaN` 或無限大時，也會回傳錯誤；v0.3.3 會畫出沒有誤差線的長條圖。

### `py`
- `PipInstall` 與 `PipUninstall` 拒絕以 `-` 開頭的依賴名稱，並在名稱前加上 `--`。呼叫端的字串過去是以單一 argv 交給 `uv pip install`，所以 `--requirement=/path` 會讓 uv 去讀那個檔案並安裝裡面列的東西。
- IPC 伺服器不再在監聽器持續失敗時空轉，每條連線設十分鐘期限。過去 `Accept` 持續失敗會在行程的餘生每次迭代印一行警告。
- 把 Python 結果傳回 Go 的 IPC 伺服器，改成在 `Run…` 呼叫需要時才開啟，同時執行的最後一個呼叫結束時關閉；在 Unix 上關閉時會一併刪掉暫存目錄裡的 socket 檔，v0.3.x 每個執行過 Python 的程式都會留下一個。伺服器開不起來時（例如暫存目錄的路徑太長，放不下 Unix socket），呼叫會在準備環境、啟動 Python 之前回傳包住原因的錯誤，下一次呼叫會再試一次。v0.3.x 只記一則警告，照樣啟動 Python，最後回傳 `exit status 1`。伺服器接受連線失敗時現在會關閉，正在執行的呼叫會回傳錯誤；過去伺服器仍然開著，之後連進來的 Python 會永遠等不到回覆。
- **BREAKING（行為改變）**：Python 環境改為釘版並經過驗證。不再用 uv 的安裝腳本安裝 uv，也不再使用 `PATH` 上的 `uv`：第一次執行時會從 GitHub release 下載 uv 0.12.20，以 release 公布的 SHA-256 核對壓縮檔，存放在 `.insyra_env/`。Python 固定為 CPython 3.12.14，一律使用 uv 下載並驗證過的版本；過去是當下最新的 `3.12.*`，或機器上已有的 Python。12 個套件釘在確切版本，連同所有相依套件，從記錄了每個檔案雜湊值的 lock 檔安裝；過去每個套件都裝當下的最新版。因為 Python 版本變了，環境改放在新目錄 `.insyra_env/py26a_<os>_<arch>`，第一次執行時從頭建立：在舊目錄 `.insyra_env/py25c_<os>_<arch>` 用 `PipInstall` 加裝的套件需要重新安裝，舊目錄不再使用，可以刪除。之後的版本若只改套件版本，第一次執行時會在原目錄把環境調整到新版本，原本較新的套件可能因此降版，用 `PipInstall` 加裝的套件會保留。版本記錄在 `py/environment/`，升級方式寫在 `Docs/py.md`。
- 環境準備中途失敗時，下一次呼叫會重新準備。過去在第一步之前就先標記為準備完成，下一次呼叫會直接使用建到一半的環境。傳給 `RunCodeContext` 等 `…Context` 函式的 context 現在也涵蓋第一次使用時的環境準備，準備環境時也不再往標準輸出印進度條。
- `ReinstallPyEnv` 執行期間會持有環境準備的鎖，文件也寫明它一直以來的行為：先刪掉整個 `.insyra_env/py26a_<os>_<arch>` 目錄（包含用 `PipInstall` 加裝的套件），再重新建立環境。
- 新增 `Run[T](ctx, code, args...)`：執行 Python 程式碼，並把傳給 `insyra.Return` 的值依 `RunCode` 的綁定規則解碼成 `T` 回傳。`dt, err := py.Run[*insyra.DataTable](ctx, code)` 取代了先宣告變數、再傳位址的寫法。`$v1`、`$v2`… 依 `RunCodef` 的方式由 `args` 代入。
- 新增 `PipInstallContext` 與 `PipUninstallContext`，傳入的 context 涵蓋環境準備與 `uv pip` 指令；`PipInstall` 與 `PipUninstall` 以 `context.Background()` 呼叫它們。
- `RunCodeWithTimeout` 標為 **Deprecated**：它就是用 `context.WithTimeout` 建 context 再呼叫 `RunCodeContext`，請改呼叫後者。它保留原本的意義，下一個版本移除。
- 傳 `nil` context 給 `Run` 或任何 `…Context` 函式會回傳錯誤。過去會讓 `exec.CommandContext` 在執行器啟動的 goroutine 裡 panic，整個程式因此結束。已經結束的 context 會在啟動任何東西之前回傳 `ctx.Err()`。
- Python 行程失敗時（回傳值之前就崩潰，或因 context 結束被終止），呼叫會回傳這個失敗；context 結束時回傳 `ctx.Err()`。過去約有一半機率回傳 nil 錯誤且沒有結果，因為執行器同時看到失敗與行程結束，隨機挑了其中一個。Python 在行程失敗前已經送回的結果仍會回傳。
- 新增 `Setup(ctx)`：讓你自己決定何時準備 Python 環境（例如程式啟動時），不必等到第一次呼叫 `RunCode` 或 `PipInstall` 時才順帶進行。這是選用的，不呼叫的話，第一次呼叫仍會照常準備環境。context 涵蓋所有下載；環境已經準備好時，`Setup` 會立即回傳，不會執行 uv。
- 結果裡的表格與清單現在會解碼成表格與清單。`insyra.Return` 會把 dict、list、tuple 裡的 DataFrame 與 Series 轉成表格或清單的格式；過去 `json.dumps` 會失敗，錯誤是 `Object of type DataFrame is not JSON serializable`。型別是 `*insyra.DataTable`、`*insyra.DataList`、`insyra.IDataTable` 或 `insyra.IDataList` 的 struct 欄位、map 值、slice 或 array 元素，會解碼成表格或清單；過去走 JSON 解碼只會得到空表。這類型別的其他部分，解碼方式和 `encoding/json` 完全相同。struct 裡只是名稱叫 `DataTable` 或 `DataList` 的一般欄位，現在逐欄解碼；過去整個結果都被塞進那個欄位，其他欄位維持零值。內嵌的 `*insyra.DataTable` 或 `*insyra.DataList` 比照內嵌 struct 的規則：有 tag 就是該名稱的欄位；沒有 tag 就拿到整個值，例如 `isr` 的型別，現在放在 slice 或 map 裡也一樣。旁邊還有其他欄位時，只有 DataFrame、Series 或 list 會交給它，dict 則交給那些欄位。`isr` 型別收到另一種值（例如清單收到 DataFrame）會回傳錯誤；過去會直接忽略，也不報錯。`isr` 指標收到 `None` 時會設為 nil，和 JSON 的 null 一樣。
- 空的 DataFrame 回傳時保留欄名；過去欄名會被改成 `a_1`、`b_1` 之類的名稱。
- 結果裡的 NaN 或無限大會解碼成 `math.NaN()`、`math.Inf(1)` 或 `math.Inf(-1)`。Python 的 `json.dumps` 會把它們寫成 `NaN`、`Infinity`、`-Infinity`，go-json 不接受，過去整個呼叫會回傳 `nil`，也不報錯：DataFrame 裡只要有一個缺值，拿回來的就是 nil。`None` 跟以前一樣解碼成 `nil`。因為其他原因讀不了的結果（例如 float64 放不下的 `10**400`），現在也會回傳錯誤並說明原因，過去同樣會變成 `nil`。Go 端拒收結果或沒有回應時，`insyra.Return` 會在 Python 裡 raise。
- 有一欄叫 `name` 的 pandas DataFrame，回傳後不會有表名。pandas 會把欄位當成屬性回傳，過去表名會變成那一欄印出來的內容。用 `df.name = "scores"` 設定的名稱一樣會成為表名。
- **BREAKING（行為改變）**：結果裡超過 2^53 的整數會保留每一位數。過去每個數字都會解成 float64，而 float64 存不下超過 2^53 的每一個整數，所以 `2**53 + 1` 回來會變成 `9007199254740992`，連 `int64` 欄位也一樣。現在這種整數在表格或清單的格子、`any` 裡會回成 `int64`（超出 `int64` 範圍時是 `uint64`），不再是那個 float64，放進整數欄位也完全精確。±2^53 以內的整數跟以前一樣是 float64。pandas 與 polars 的 DataFrame 改成逐欄轉換後送出，id 欄旁邊有小數欄時，整數不會在 Python 那邊就被轉成小數。整數欄位放不下的數字（例如 `int64` 收到 `2**63`）會回傳錯誤，過去 go-json 遇到 19、20 位數的數字會默默繞成錯的值。
- `RunCodef`、`RunFilef`、它們的 `Context` 版本和 `Run` 的參數，不會再變成 Python 程式碼。過去佔位字元是一個參數接一個參數、對整份程式碼輪流替換，後面的參數替換時，也會改到前面參數插進去的文字：傳入 `$v2` 和 `+__import__('os').system('id')+` 兩段文字，就會執行那個指令。`$v1` 也會吃掉 `$v10` 的開頭。JSON 寫不出來的值（例如 `[]any{文字, math.NaN()}`）會照 Go 的格式原樣寫進程式碼，裡面的文字就成了程式碼。現在佔位字元只在原本的模板裡找一次，依完整編號替換，也只轉換模板用到的參數。寫不成 Python 值的參數，會在 Python 啟動前回傳錯誤，並註明是哪個佔位字元。含有 NaN 或無限大的 `[]float64` 也一樣，過去會寫成 Python 不認得的 `NaN` 或 `+Inf`。

### `pd`
- **BREAKING（行為改變）**：`FromDataList` 遇到空的 list 會回傳空的 `Series`，也就是長度 0、`any` 型別的 gpandas series，和 pandas 的 `pd.Series([])` 一樣。過去會回傳 `empty DataList` 錯誤。`nil` list 仍然回傳錯誤。

### `parallel`
- **BREAKING**：`Run` 改為回傳新的 `*RunningGroup`，等待時會回報失敗的函式。`ParallelGroup` 現在是一份可以重複使用的函式清單，只有 `Run` 方法，每次 `Run` 都是一次獨立的執行，結果各自存放。過去同一個 group 跑兩次會把所有函式重跑一遍、寫進同樣的結果欄位，兩個 `Run` 同時呼叫還會發生資料競爭。`RunningGroup` 只有 `AwaitResult` 與 `AwaitNoResult`，所以等待一個從沒啟動的 group 會直接編譯失敗，過去則是立刻回傳空結果。`AwaitResult` 改為回傳 `([][]any, error)`，`AwaitNoResult` 改為回傳 `error`。函式 panic 或傳入的值無法呼叫時，該格為 `nil`，並以帶有位置、panic 值與呼叫堆疊的 `*parallel.WorkerError` 回報。過去該格會放一個 `error`，和函式自己回傳的 error 分不出來；函式自己回傳的 error 現在就單純留在結果格裡。`GroupUp` 會複製傳入的參數。遷移方式：`results := g.Run().AwaitResult()` 改成 `results, err := g.Run().AwaitResult()`，宣告為 `*parallel.ParallelGroup` 並用來接 `Run()` 結果的變數，改成 `*parallel.RunningGroup`。

### `accel`
- `NewSession` 最多只能給一個 `Config`，給兩個時由 `Discover` 回報錯誤，不再只用第一個。

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
