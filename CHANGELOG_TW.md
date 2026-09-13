# 變更紀錄

影響 Insyra 使用者的變更，依套件分類，分法與 release note 相同。`## Unreleased` 收錄下一個版本會包含的內容。

v0.3.0 及更早的版本不重複收錄於此，請見 [GitHub Releases](https://github.com/HazelnutParadise/insyra/releases)。

English: [CHANGELOG.md](CHANGELOG.md)

## Unreleased

### Core

- 修正整數排序失去精度。過去所有整數都經 float64 比較，任何兩個大於 2^53 的 `int64` 都會被視為相等，`Sort`、`SortBy`、`Pivot`、`Describe` 的 min/max 因此排錯。現在整數以精確方式比較，含混合有號／無號與超出 `int64` 的值。
- 修正 `HermiteInterpolation` 用錯基底，完全不滿足它名稱所宣稱的導數條件。現在會通過每個值、符合每個給定導數，並精確重現低次多項式。
- `TryParseTime` 接受常見的無時區版面：`2006-01-02 15:04:05`、`2006-01-02T15:04:05`、`2006-01-02 15:04` 及以 `/` 分隔的等價寫法，一律視為 UTC。CCL 的日期函式與 `datafetch` 過去會把這些字串當成純文字。
- 修正 `ShowTypes` 超過 26 欄時印成 `A, AA, AB, B, …`，現在與 `Show` 同順序。`ShowRange` 的文件改為與實作一致：end 為排除，負數 end 由尾端往回數且仍為排除（同 Python slice），要顯示到最後請傳 `nil`。
- `DataList`／`DataTable` 的 `Close()` 不再丟棄已在等鎖的操作。Close 停止的是加鎖，不是已排隊的工作。
- 修正 `DataList.ReplaceLast` 在 list 以 `NaN` 結尾時，改掉最後一個 `NaN` 而不是最後一個等於 `oldValue` 的格子（`[5, NaN].ReplaceLast(5, 0)` 得到 `[5, 0]`）。
- 修正 `ReadJSON_File` 把整數字面值讀成 `float64`、而 `ReadJSON` 讀成 `int64` 的不一致；兩者現在走同一條解碼路徑，從檔案讀大整數不失真，內容為單一物件的檔案載入為一列。
- 修正 API 審查找到的多個 `DataTable` 行為：`GetElementByNumberIndex`、`SetRowToColNames`、`SetColToRowNames` 遇到越界索引不再 panic（改設 `Err()`），`GetElementByNumberIndex` 也接受負的欄索引；`Filter*` 沒有符合時回傳可安全呼叫方法的空表；`FilterRows`／`FilterCols` 對參差不齊的表不再 panic；`DropRowsByIndex` 以原始列數換算負索引並忽略重複（過去 `(-1, 0)` 會留下第 0 列、`(1, 1)` 會刪掉兩列）；`Transpose` 保留所有列名（超過原欄數的列名過去會遺失）；`AppendRowsByColIndex` 補欄到指定索引而不是丟掉值；`DropColsContainNumber`／`DropRowsContainNumber` 認得所有數值型別（CSV 推斷出的 `int64` 欄過去不會被刪）；`Mean` 以數值格數作分母。
- `Config.SetLogLevel`、`SetUseColoredOutput`、`SetDontPanic`、`SetDefaultErrHandlingFunc` 改為原子操作，另一個 goroutine 正在寫 log 時呼叫它們不再是 data race。`SetDefaultErrHandlingFunc` 設定的 hook 改由單一 goroutine 依序處理、佇列有上限，不再每個 warning 開一個 goroutine；已有 1,024 個呼叫在排隊時，後續的 hook 呼叫會被丟棄，錯誤本身仍會記錄。import 套件不再印出「Welcome to Insyra」橫幅，改由 CLI REPL 啟動時印出。
- `DetectEncoding` 不再因多位元組字元恰好被 8 KB 取樣邊界切開而誤判 UTF-8 檔案。
- `DataList.IsEqualTo` 與 `IsTheSameAs` 遇到 Go 無法用 `==` 比較的格子（例如含 slice 的 struct）不再 panic，這類格子依型別與內容比較。`ClearNaNs`、`ClearNils`、`ClearNilsAndNaNs`、`ClearNumbers`、`DropAll`、`ClearStrings` 改為單趟過濾，結果不變，不再是平方級原地刪除或每次呼叫開 goroutine；`Update` 在 `Err()` 記的是自己而不是 `ReplaceAtIndex`。`DataTable.FindColsIfContains`／`FindColsIfContainsAll` 不再在每個不含該值的欄上留下警告；`Count` 與 `Clone` 移除 goroutine 分派。`AppendRowsByColName`（連帶 `ReadJSON`／`ReadJSON_File`）新增欄位時依欄名排序，同一份 JSON 不再每次讀出不同的欄序。
- CCL：`NULL`、`TRUE`、`FALSE` 不分大小寫都是關鍵字而非欄位參照；`@` 當值使用時每列拿到自己的 slice（過去每格都顯示最後一列）；日期相減以秒參與比較與除法，`(A - B) > 0` 不再靜默為 false；序列函數放在另一個序列或聚合函數裡（`LAG(LAG(A,1),1)`、`SUM(LAG(A,1))`）保留整欄；離譜的 `LAG`／`LEAD`／`ROLLING_*` 位移與 `REPEAT` 次數回錯誤而非 panic；函數註冊表可在另一個 goroutine 求值時安全新增；`engine/ccl.NewMapContext` 依欄名排序，`A`／`B` 指向固定的欄。
- `ToCSV` 會回傳最後一次 flush 的錯誤；小表寫到已斷的 pipe 過去會回報成功。
- 在某個實例的 `AtomicDo` 內呼叫 `AtomicDoAll` 不再與另一個做鏡像操作的 goroutine 死鎖：改成不再加鎖、內聯執行回呼，與巢狀 `AtomicDo` 同一規則。
- `DataList` 與 `DataTable` 新增 `PopErr()`，回傳目前的 `Err()` 並清除它；另新增 `SetErr(packageName, funcName, msg, args...)`，以 insyra 自身方法相同的方式在實例上記錄錯誤（寫一筆警告並取代 `Err()`）。`IDataList` 與 `IDataTable` 都包含這兩個方法。
- 全域錯誤緩衝區上限為 `ErrorBufferCapacity`（1536 筆），滿了丟最舊的，不再無限成長。文件定位改為診斷用日誌而非錯誤處理 API；其中九個存取函式（`PopError`、`PopErrorByPackageName`、`PopErrorByFuncName`、`PopErrorAndCallback`、`PeekError`、`GetErrorsByLevel`、`GetErrorsByPackage`、`PopErrorInfo`、`HasErrorAboveLevel`）標為 deprecated，改用 `GetAllErrors`、`PopAllErrors`、`HasError`、`GetErrorCount`、`ClearErrors`。
- 讀取 CSV（`ReadCSV_File` 與 `csvxl` 的 CSV 讀取函式）能解碼偵測器可能回報的每一種字元集：UTF-16／32、所有 ISO-8859 分部、Windows-1250 到 1258、KOI8-R／U、Shift-JIS、ISO-2022-JP、EUC-JP、EUC-KR、IBM866、Macintosh，以及常見別名；分隔符號與大小寫都不影響（`ISO-8859-1`、`iso8859_1`、`latin1` 視為相同）。過去只有 UTF-8、UTF-16、Big5、GB18030 有解碼器：偵測器判斷正確的 `iso-8859-1` 會被原樣塞進表，儲存格不是有效的 UTF-8；沒有 BOM 的 `utf-16be` 則被當成 little-endian 讀取。其他名稱的讀法維持不變。
- `DetectEncoding` 能辨識 UTF-32 的 BOM；過去會被判成 UTF-16，因為後者的 BOM 是前者的前綴。
- 新增 `DataTable.ToCSVWithOptions` 與 `CSVWriteOptions`，其中 `SanitizeFormulas` 會在開頭為 `=`、`+`、`-`、`@` 的儲存格前加上單引號，避免試算表把它當公式執行。預設關閉，因為它會改變寫出的值；`ToCSV` 的輸出不變。
- 修正 SQLite 上 `ToSQL` 無法附加到名稱含空白的資料表：查詢既有欄位的語句沒有像其他語句一樣為識別字加引號。
- CCL 的錯誤會說清楚是哪個階段、出在哪裡。編譯失敗指出運算式、問題所在的位元組偏移量與該處的文字（`cannot compile "(A B)" at offset 3 (near "B"): expected ')'`）；執行期失敗指出列號（`cannot evaluate "A / B" at row 1: division by zero`），不依賴列的運算式則不報列號，而不是報一個誤導的。`AddColUsingCCL`、`EditColByIndexUsingCCL`、`EditColByNameUsingCCL` 過去兩種失敗共用 `Failed to apply CCL on DataTable after 213.792µs` 這個前綴，既回答不了「是我公式寫錯還是資料有問題」，還把碼錶讀數放在原因該在的位置。`position N` 原本在一個地方是位元組偏移、在另一個地方是 token 序號，現在一律是位元組偏移，並且會對齊到字元邊界，非 ASCII 的運算式才指得到真的位置。兩則訊息會印出 Go 的內部結構（`unexpected token: {5 )}`、`invalid range operands: &{: 0x1d48…}`），現在改印原始文字。`ExecuteCCL` 裡的失敗會指出是哪一條語句，五行的腳本不會再只回 `Failed to execute CCL statement: division by zero` 卻沒說是哪一行。錯誤本身是匯出型別 `engine/ccl` 的 `CompileError` 與 `EvalError`，`ErrorInfo` 也加上 `Cause` 欄位與 `Unwrap`，所以可以用 `errors.As(dt.Err(), &compileErr)` 判斷，不必比對訊息字串。它們仍和其他 `DataTable` 錯誤一樣以警告記錄。
- CCL 不再重複計算答案不會變的東西。`AddColUsingCCL`、`EditColByIndexUsingCCL`、`EditColByNameUsingCCL` 裡沒有用到 `#` 的聚合函數，改為在逐列走訪之前算一次，而不是每一列都對整欄的新副本重算一遍：20,000 列的 `A / SUM(A)` 從 2.1 秒變成 1.0 毫秒，z-score `(A - AVG(A)) / STDEV(A)` 從 6.2 秒變成 1.8 毫秒。有用到 `#` 的聚合會依賴當前列，仍然逐列計算。同一條路徑上另外三項：字串只有在首字元可能是日期開頭時才交給日期解析器、`REGEX_MATCH` 每個樣式只編譯一次而不是每列編譯、`ROLLING_*` 整欄只轉換一次而不是每個視窗把每個元素各轉一次。**結果完全沒有改變**；`ROLLING_*` 刻意維持與視窗成正比的計算量，因為改用累加器會產生重算視窗所沒有的誤差飄移。
- `AtomicDoAll` 遇到 nil 的實例（nil 的 `*DataList`、nil 的 `*DataTable` 或 nil 值）會直接略過，不再 panic；同時傳入的其他實例照樣上鎖。
- 修正 `GroupBy` 算出來的是 `Aggregate` 執行當下的父表資料，而不是分組當下的資料。它原本只保留父表欄位的指標，所以兩次呼叫之間對父表的修改會跑進結果（原本是 `3` 的一組加總變成 `102`），而且 `Aggregate` 讀那些資料時沒有上鎖，其他 goroutine 可能同時在寫。現在 `GroupBy` 會複製它分組用的欄位資料。
- 修正依值搜尋、計數、取代與刪除時，找不到以另一種 Go 整數型別儲存的整數。CSV 讀進來的整數是 `int64`，Go 程式裡寫的 `2` 是 `int`，原本用 `==` 比對，`int64` 和 `int` 永遠不相等，所以對 CSV 讀入的表，`Count(2)` 回傳 0，`FindAll(2)`、`FindRowsIfContains(2)` 什麼都找不到，`Replace(2, 0)` 什麼都沒改。現在 `Count`、`FindFirst`、`FindLast`、`FindAll`、各個 `Replace` 方法、`DropAll`、`FindRowsIfContains(All)`、`FindColsIfContains(All)`、`DropRowsContain`、`DropColsContain` 都依數值比對整數，`int8` 到 `int64`、`uint8` 到 `uint64` 一律如此。小數仍然不等於整數，要找 `2.0` 請用 `2.0` 搜尋；小數、字串、布林值、`nil` 與 `NaN` 的比對結果都和原本完全相同。`IsEqualTo` 與 `IsTheSameAs` 仍然連型別一起比較。遇到 Go 無法用 `==` 比較的儲存格（例如含 slice 的 struct），這些查找也不再 panic，而是依型別與內容比對。比對方式改成依要找的值的型別，每次呼叫只挑一次，所以搜尋小數或字串也變快了：一百萬格的 `Count`，小數從 2.3 毫秒降到 1.7 毫秒，字串從 2.8 毫秒降到 2.0 毫秒。
- 修正 CCL 在 amd64 與 arm64 上答案不同的問題：當計數、小數位數或日期位移大到超出 Go 轉成整數或時長的範圍時，這種轉換在 Go 裡沒有定義。在 Linux 與 Windows 上 `MID('abc', 2, 10^300)` 回傳 `""`，Mac 上卻是 `"bc"`；`ROUND(x, NaN)` 在一邊回傳 `NaN`、在另一邊四捨五入成整數；`DATEADD(d, 10^300, 'day')` 在兩邊各算出不同的錯誤日期。現在字元數或位置超過字串結尾，一律代表「到結尾為止」。`NaN` 的計數或小數位數、`DATEADD` 超過 2,147,483,647 天／月／年或以時分秒計超過約 292 年、日期加減超過 106,751 天、`DAY`／`HOUR`／`MINUTE`／`SECOND` 超過約 292 年的秒數，以及為 `NaN` 或超出 int32 範圍的列範圍邊界，都改為回報錯誤，不再產生繞回的值。一般數值的結果不變，捨去小數的轉換也照舊：日期加上天數仍以整小時移動。
- 修正 `Show`（以及其他所有顯示路徑）在 amd64 與 arm64 上印出不同的數字。格式化以 `v == float64(int(v))` 判斷浮點數是否為整數，而該轉換在 Go 裡對超出 `int` 範圍的值沒有定義結果：`2^63` 在 Mac 上印成 `9223372036854775807`，看起來像精確整數但比實際值少 1，在 Linux 與 Windows 上印成 `9.2234e+18`。超出 `int64` 範圍的值現在一律使用指數形式。範圍內的整數不變。
- 修正毫秒的 Unix 時間戳超過 2262-04-11 後讀成錯誤日期。轉換時先把毫秒乘成奈秒，而該乘法在它自己接受的區間內就會溢位 `int64`，`99999999999999` 因此回傳 2216 年而不是 5138 年。
- 修正六個安靜出錯的值。16 位（微秒）時間戳被當成秒，`2023-11-14` 變成 `53872825-06-17`；毫秒的區間也沒接到微秒，中間留了一段會掉到秒的範圍。`CalcColIndex` 在 `int` 範圍的最上緣產生出自己的 `ParseColIndex` 讀不回來的欄位名稱。`Show` 把 `9999.99999` 印成 `10000`——四捨五入到小數第四位後再去掉尾零，看起來就是個整數。`IsNumeric` 與數值讀取路徑對「以數值 kind 為底的具名型別」（`type Celsius float64`、`time.Duration`）看法不同：`IsNumeric` 說這種儲存格是數字，`ToFloat64Safe` 卻拒絕、`ToFloat64` 給 `0`，所以 `Mean` 與 `Sum` 會跳過它，`ToF64Slice` 會把它變成 `0`。兩邊現在都讀出它的值（`time.Duration` 以奈秒讀取）。`DataList.NearestNeighborInterpolation` 對 NaN 的 x 回傳第一個值——與 NaN 的比較永遠為 false，搜尋根本沒動過——而 `LagrangeInterpolation` 與 `NewtonInterpolation` 回傳 NaN 且 `Err()` 沒有任何紀錄；三者現在都回傳 NaN 並把失敗記在 `Err()`，與原本就會拒絕的 `LinearInterpolation`／`QuadraticInterpolation` 一致。
- 修正兩張沒有欄位名稱的表無法垂直合併。`NewDataTable(NewDataList(...))` 建出來的每一欄都沒有名稱，而重複名稱的檢查把空字串當成自己的重複，所以最單純的建構式產生的形狀無法與同類合併。沒有名稱的欄位現在依「在無名欄中的位置」對齊，有名稱的仍然依名稱對齊。
- CCL：超出 `float64` 範圍的數字字面值改為回報錯誤，不再靜默變成 `+Inf`——`strconv.ParseFloat` 的錯誤本來被丟掉了。指數形式現在是合法的字面值：`1e5`、`1.5e-3`、`2E+3` 都能編譯，過去會被拆成數字加識別字然後以 `unexpected token` 失敗，儘管 `VALUE('1e3')` 一直可用、CCL 自己的字串輸出也用指數形式。`e` 後面要有數字才會併入數字，所以名為 `E` 或 `E1` 的欄位不受影響。`TOSTR(1.5, '%d')` 與 `TOSTR(1, '%')` 改為回報格式不符，不再把 Go 自己的抱怨——`%!d(float64=1.5)`、`%!(NOVERB)`——寫進儲存格。
- `Show` 與其他顯示路徑對實作 `fmt.Stringer` 的 struct 值，改為印出它自己的文字。格式化函式最後會走到一個分支，對 struct 直接印 `<pkg.Type>`，從來沒問過它能不能自己印，所以 Parquet 的十進位欄位每一列都顯示成 `<decimal.Decimal>`。沒有 `String()` 的 struct 仍然顯示型別名稱，儲存的值不變。
- 不是合法 UTF-8 的字串格子改以十六進位顯示，與 `[]byte` 原本的顯示方式相同，不再是加了引號的亂碼。除了看不懂，加引號的位元組還會讓整列歪掉一欄：`runewidth` 把 NUL 算 0 欄寬、把非法位元組算成一個 `U+FFFD`，而終端機實際畫幾欄由它自己決定，任何寬度計算都不可能算對。現在這種字串顯示成 `00ff41`，超過 20 個位元組會截斷並附上總長度；十六進位是 ASCII，欄位就對得齊。合法的 UTF-8 完全不受影響，中日韓文字與 emoji 也一樣，儲存的值不變。
- `Counter` 遇到 Go 無法當 map key 的格子值不再 panic，各種查找也與它一致。`Counter` 以格子值當 `map[any]int` 的 key，不可雜湊的值會讓程式直接死掉；而 `Count`、`FindAll`、`Replace`、`DropAll`、`IsEqualTo`、`IsTheSameAs` 與 `DataTable` 的查找、刪除方法把它當成跟什麼都不相等，所以 `Count` 對明明存在於兩列的值回報 0。這不需要刻意就會遇到：`ReadSQL` 刻意把二進位欄位保留成 `[]byte`（BLOB、BYTEA、BINARY、VARBINARY、LONGBLOB）。現在這種值以型別與內容識別：`[]byte{0, 255}` 與另一個 `[]byte{0, 255}` 相符，但與 `[]int{0, 255}` 不相符，裡面的整數則和 `Count` 一樣依數值比較。`Counter` 用新的 `insyra.UncomparableKey` 當它的 key，由新的 `insyra.ToMapKey(v)` 產生，供需要自行索引整張 map、或自己用格子值建 map 的呼叫端使用；可比較的值仍以自身當 key，`counter[1]` 寫法不變，印出來的替代 key 會顯示型別與縮短後的內容，例如 `[]uint8(00ff41)`。要查單一值的次數請用 `Count`，它以數值比對整數，而 map 是以 Go 型別當 key。`IsEqualTo` 裡單獨一格的 `NaN` 仍然不等於另一個 `NaN`；在 slice、map 或 struct 裡面，`NaN` 是內容的一部分，所以兩格 `[]any{NaN}` 相等，也彼此相符。`GroupBy`、`Pivot`、`Merge` 與 `Describe` 的 key 維持不變。
- `insyra.Cell(v)` 讓一個值在建構子會攤平它的情況下仍然佔一格。`NewDataList` 刻意攤平切片，好讓建構清單讀起來像建構 pandas Series，而在此之前沒有辦法只讓其中一個引數例外，只能離開建構子改用 `Append`，但那沒辦法跟其他值寫在同一次呼叫裡。`NewDataList(insyra.Cell([]int{1, 2}), 3, "a")` 是三格，第一格以原本的型別持有那個切片。每個接受呼叫端傳入值的入口都接受這個標記並拆掉它，包括 `Append`、`Update`、`InsertAt`、`DataList` 與 `DataTable` 的 `Replace` 和 `Replace…With` 系列、`Shift` 的填補值、`UpdateElement` 與兩個列附加方法，其中也包含本來就不攤平的那些，所以 `Append(Cell(x))` 與 `Append(x)` 意思相同，`Count(Cell(x))` 與 `Count(x)` 也一致。沒有標記的切片仍然攤平。
- `insyra.UncomparableKey` 改以值自己的寫法顯示。它原本印的是編碼內容，對結構而言就是其欄位，所以實作 `fmt.Stringer` 的不可比較值在印出的 `Counter` 結果裡會顯示內部欄位：Parquet `Decimal128` 欄讀成的 `decimal.Decimal` 會印出其 `big.Int` 的正負號與位元組。現在這種值顯示它自己的文字，例如 `decimal.Decimal(-340.0221114815)`，套用同一個截斷上限；沒有 `String()` 的值顯示不變。識別仍然由編碼決定而不是文字，因為 `String` 可能失真，兩個不同的值若文字相同絕不能被併成一組。
- 修正 `SortBy` 處理不存在欄位的方式。過去找不到的索引、名稱或數字是由內部的 `GetCol`、`GetColByName`、`GetColByNumber` 回報，而不是 `SortBy`，多層排序其中一層無效時其他層照樣套用。現在會在移動任何一列之前檢查每一層，找不到欄位時在 `SortBy` 記錄錯誤並保持表格不變。同一個設定同時給了 `ColumnIndex`、`ColumnName`、`ColumnNumber` 其中多個時，仍照文件的優先順序（索引、名稱、數字）排序，並新增一則警告指出被忽略的欄位，`Err()` 維持 nil。沒有指定任何欄位的設定仍依第一欄排序，`Docs/DataTable.md` 現在有寫明：`ColumnNumber` 不為零才算有指定，所以它的零值不會蓋過名稱或索引。

### CLI

- 會解析到環境目錄之外的環境名稱現在會被拒絕：空白或絕對路徑的名稱，以及經由 `..` 跳出目錄的名稱（`../x`、`a/../../x`）。過去名稱直接接在環境目錄後面，`../x` 會在目錄外建立或刪除資料夾。其他名稱照常可用，包含含空格或非 ASCII 字元的名稱。
- 命令登錄表加上鎖，多個 goroutine（嵌入端）同時註冊命令不再是 data race。
- 修正 `col`、`row`、`movavg`、`expsmooth`、`diff` 找不到或算不出結果時把 nil 存進變數，下一次存檔整個 session panic 的問題；現在回錯誤且不存。
- 含 `NaN` 或 ±Inf 的變數（例如有空白格的 CSV）能完整存檔與還原；過去這種表會變成空字串，這種 list 或一般值則讓整次存檔失敗。其他變數的寫法與過去完全相同，既有的 state 檔也照常能讀。
- `--env`、`--no-color`、`--log-level` 放在 `newdl`、`addcol`、`addrow`、`show` 前面時會生效，不再被當成資料寫進 default 環境。
- `run` 遇到腳本裡的 `env open` 不再開啟互動 REPL；腳本自己呼叫自己超過 16 層會停止。
- `db connect` 寫進 `history.txt`、REPL 歷史與 `env export` 時密碼會被遮罩（URL、`user:pass@`、`password=` 三種形式）；history 檔以 0600 建立。
- `accel` 的 Usage 不再宣稱有不存在的 `run` 子命令。
- `help` 現在如實列出 `pca`、`regression`、`count` 的參數：前兩者可以用 `as <var>` 存結果，`count` 的 value 是必填，不再標成選填。`save … sql` 的用法錯誤訊息也跟 Usage 一致，列出 `rownames [true|false]`。
- `count`、`find`、`replace` 現在能對上 CSV 載入的表，以及 one-shot 模式下每次還原的變數裡的整數。原本打的 `2` 是 `int`，存著的是 `int64`，永遠比對不到：`count x 2` 印出 0，`find x 2` 印出 `[]`，`replace x 2 0` 印出 `replaced` 卻什麼都沒改。
- 資料庫連線不再印出 gorm 的查詢日誌。它的預設 logger 在查詢失敗或過慢時會把綁定參數內插進訊息，所以 `WHERE token = ?` 會把 token 印在終端機以及任何收集它的地方。CLI 本來就自己回報錯誤，不會少掉什麼。
- 錯誤訊息會指出是哪個指令、哪個引數，不再直接把標準函式庫的文字丟回來。`ttest single x abc` 過去回答 `strconv.ParseFloat: parsing "abc": invalid syntax`，現在是 `ttest: invalid mu "abc", expected a number`。`ttest`、`ztest`、`anova`、`chisq`、`movavg`、`expsmooth`、`shift`、`diffn`、`pctchange`、`rolling`、`expanding`、`ewm`、`quartile`、`percentile` 與 `fetch` 共 21 處統一成這個寫法。
- `kmeans` 與 `knn` 的選項鍵不分大小寫——`NSTART 3` 可以用了——未知的選項會列出有哪些。`knn` 的 `weighting` 與 `algorithm` 會先不分大小寫地對照允許的值，不再把字串直接往下傳，所以拼錯會在這裡被擋下，而不是變成函式庫收到一個不認識的模式。
- `clone`、`replace`、`clean`、`fillna`、`count` 能分辨「變數不存在」與「變數存在但型別不對」。五個過去都對明明就在的變數說「variable not found」，害人去找一個根本不存在的拼字錯誤。
- `help` 的表格依最長的指令名稱對齊，`knn_neighbors` 不再把描述擠歪；`read` 與 `env` 補上 Forms 與 Examples——`env` 有九個子指令，過去一個都沒列。`help` 與 Tab 補全改為在鎖保護下讀取指令登錄表，透過新匯出的 `commands.LookupCommand` 與 `commands.SnapshotRegistry`。
- `read sales.csv as x` 會告訴你該怎麼做，不再回答 `unknown option "as"`。`read` 只做預覽，內部自己補了一個別名，使用者再給一個就變成第二個 `as`，抱怨的地方完全不對。現在的訊息是「read only previews a file. Use `load sales.csv as <var>` to keep it」。

### `ml` 與 `nn`

- `ml/mltest.RunConformance` 遇到 `Classes()` 回傳 nil 的實作時會判定失敗，不再直接 panic。`ml.Classifier` 是公開介面，本 repo 以外的程式也能實作；測試套件過去對結果直接呼叫 `Len()`，沒有任何檢查。

### `datafetch`

- 檔案版 geocode 快取（`NewFileGeocodeCache`）改為先寫暫存檔再 rename，寫入中斷不再留下損壞、下次執行被靜默丟棄的快取檔。
- Google Maps 爬蟲的 `Search` 恢復可用。Google 不再把店家 ID 放在爬蟲讀取的頁面裡，所以過去任何查詢都回傳空結果，也沒有警告。現在改讀 Maps 網頁本身請求的搜尋結果清單，一次請求就取得最多 20 家店和店名，不必再為每家店多開一個頁面，查不到時也會警告。`GoogleMapsStores()` 不再於執行期從 GitHub repo 下載端點與請求標頭，建立時不需要網路，也不會回傳 nil，每個請求都有 30 秒逾時。`GetReviews` 的進度改記在 debug log，不再印到標準輸出，`MaxWaitingInterval_Milliseconds` 剛好是 1000 時不再 panic，`SortBy` 或 `MaxWaitingInterval_Milliseconds` 為零時直接使用預設值，不再警告。`GetReviews` 也恢復可用。Google 會拒絕它原本送出的評論請求，Google 地圖對未登入的訪客也只顯示五則評論，所以現在改讀 Google 搜尋結果中評論視窗的評論頁：每頁 10 則，四種排序都能用，不需要登入（[#249](https://github.com/HazelnutParadise/insyra/issues/249)）。`ReviewDate` 改為 UTC 的發布日期（`YYYY-MM-DD`），`Content` 的 `<br>` 改為換行，`ReviewerState` 與 `ReviewerLevel` 則一律為空，因為新的評論頁不再提供。
- `GoogleMapsStoreReview` 新增四個欄位，由 `GetReviews` 填入，`ToDataTable` 也會產生同名的欄：`ReviewID`、`Language`（評論的語言代碼，例如 `zh-Hant`，只給星等沒寫內容的評論則為空）、`ReviewerReviewCount`（評論者寫過幾則評論）與 `ReviewerPhotoCount`（評論者上傳過幾張相片）。

### `stats`

- `KMeans` 的初始中心改為相異列，與 R 一致。資料含重複列時，單次啟動過去會抽到同一列兩次而回報 "empty cluster"（實測 50 個 seed 中有 44 個失敗）。現在抽到重複才從相異列重抽；本來就相異的抽樣完全不動，既有 seed 的結果逐位不變。
- 接受 `insyra.IDataList` 的函式對 `nil` 或非 `*insyra.DataList` 的實作不再 panic；值會被轉換，`nil` 以一般錯誤回報。
- `FactorAnalysisResult.RotationConverged` 改為回報旋轉實際上有沒有收斂。它過去恆為 `true`：多起點挑出來的候選從來沒帶收斂旗標，`fa.Rotate` 只好用預設值，所以一個在 1e-12 容忍度下只跑一次迭代就停的旋轉也回報收斂。現在旗標描述的是實際回傳的那個解，多起點之間怎麼挑選則不變。
- 旋轉沒有收斂時，每次搜尋只記錄一則警告，指出方法、起點數與迭代上限，而且只在回傳的解沒有收斂時才記錄，也就是 `RotationConverged` 為 `false` 的情況。過去 `GPForth` 與 `GPFoblq` 對每個跑到上限的起點都警告一次，多起點搜尋自己建的 Varimax 起點也會警告，所以 `Restarts` 大於 1 的呼叫可能記下好幾則警告、往全域錯誤緩衝區塞同樣多筆，而回傳的解其實已經收斂。那些個別起點現在只在 debug 層級回報。

### `csvxl`

- 修正 `AppendCsvToExcel` 遇到同名工作表時舊儲存格殘留的問題：`excelize.NewSheet` 對既有名稱只回傳原工作表，所以只有新 CSV 覆蓋到的儲存格被改寫，其餘保留。現在會先刪除再重建，工作簿只有那一張工作表時也能完成。
- 修正 `AppendCsvToExcel`、`ExcelToCsv`、`EachExcelToCsv` 開啟的工作簿從未關閉。
- 錯誤改用 `%w` 包裝底層原因（`errors.Is(err, os.ErrNotExist)` 可用），輸出目錄改以 0755 建立而不是 0777。
- `ExcelToCsv` 與 `EachExcelToCsv` 拒絕無法當單一檔名的工作表名稱（`../x`、`a/b`），惡意 workbook 過去可藉此截斷輸出目錄外的檔案。每張工作表先讀完才建立 CSV，最後一次 flush 的寫入錯誤也會回傳。

### `parquet`

- 修正 `ReadColumnOptions.MaxValues` 完全沒有作用。`ReadColumn` 現在先從檔案 metadata 加總所選 row group 的列數，超過上限時在讀取任何資料前就拒絕，這才是該欄位文件寫的行為。
- `Write` 關閉 Parquet writer 失敗時改為回傳錯誤，而不是只寫 log。檔尾在關閉時才寫入，過去關閉失敗會留下無法讀取的檔案卻回傳 `nil`。套件其他地方關閉時的錯誤改經 Insyra 的 logger 而非標準 `log` 套件，`Config.SetLogLevel` 對它們生效。
- 修正 `FilterWithCCL` 只回傳前 1000 列符合的資料。檔案以每批 1000 列讀取，而第一批之後的每一批都被接到結果欄位的複本上而不是欄位本身，所以 2500 列的檔案用每列都成立的條件過濾，回傳的是 1000 列，而且完全沒有錯誤。符合的列現在跨整段串流收集。
- 最後一批之後才發生的讀取錯誤不再被換成部分結果。讀取端的紀錄通道與錯誤通道是一起關閉的，`FilterWithCCL`、`ApplyCCL` 與 `Stream` 的 `select` 可能挑中任何一個，因此讀到一半失敗的檔案可能回傳截斷的表格而 error 為 nil，`ApplyCCL` 更會把截斷的結果覆蓋回原檔。三者現在都先讀錯誤通道再結束。成功的串流呼叫也不再每次都印出 `failed to close file … file already closed`。
- 讀取其他工具寫出的 Parquet 檔時，reader 不認識的欄位型別不再變成一串重複的字。`getVal` 的 fallback 回傳 `arr.String()`，也就是整個 array 的字串形式，忽略列索引，所以那一欄每一列都讀成類似 `[19000 19001 19002]` 的東西，而 `Read` 回傳 nil error、表格的 `Err()` 也是 nil。`parquet.Write` 只寫得出七種 Arrow 型別，所以這個問題在本函式庫自己寫的檔案上看不到，只在讀別人的檔案時發生。現在 `Date32`／`Date64` 讀成 UTC 午夜的 `time.Time`，`Int8`／`Int16` 與四種無號整數讀成同名的 Go 型別，`LargeString` 讀成 `string`，`Decimal128`／`Decimal256` 以檔案自己的未縮放整數與 scale 精確讀成 [go-decimal](https://github.com/TimLai666/go-decimal) 的 `decimal.Decimal`。其餘的 `List`、`Struct`、`Map`、`Time32`、`Time64`、`Duration`、`Interval` 讀成 `nil`，檔案其他欄位照常讀取。`Read`、`Stream` 與 `ReadColumn` 會以警告記錄原因，並寫進回傳表格或 list 的 `Err()`，內容是 `column "tags": unsupported Arrow column type list<item: int64>; its cells were read as nil`；有好幾欄不支援時，每一欄都會記錄警告，`Err()` 留下的是最後一欄。`FilterWithCCL` 對這種欄位同樣拿到 `nil`，但不記錄原因。可以用 `ReadOptions.Columns` 跳過該欄。Dictionary 編碼的欄位從來不受影響，reader 會把它還原成底層型別。
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
- GLPK 下載、解壓或編譯失敗不再結束程式：失敗以警告記錄，之後呼叫 `SolveModel`／`SolveFromFile` 時透過附加資訊表回報找不到求解器。`SolveModel` 兩處建立暫存檔失敗改為記錄警告並回傳 `nil, nil`，不再結束程式。
- `SolveFromFile` 收到超過一個 `timeoutSeconds` 時，改在尋找或安裝 GLPK 之前就拒絕，已經寫錯的呼叫不會再觸發安裝。它仍然記錄警告並回傳 `nil, nil`。
- 解壓 GLPK 時建立的目錄權限改為 0o755，不再是 0777。

### `plot`

- `CreateRadarChart` 未提供 indicators、`CreateHeatMap` 日曆模式的 X 型別錯誤或未設 `CalendarOpts` 時，改為記錄警告並回傳 `nil`，不再結束程式或 panic。
- `nil` 的 `IDataList` 不再讓程式當掉。`CreateBarChart` 與 `CreateLineChart` 會記錄警告、略過 nil 的清單並畫出其餘部分，全部都是 nil 時才回傳 `nil`。`CreateBoxPlot` 以同樣方式略過序列裡 nil 的清單，清單全是 nil 的序列會被拿掉；原本就沒有任何清單的序列照舊保留。`CreateWordCloud` 回傳 `nil`。每個圖表都透過 `AtomicDo` 讀資料，而那會解參考接收者，所以夾在正常清單裡的一個 nil 過去會 panic。
- `SavePNG` 在輸出路徑沒有副檔名時回傳錯誤，不再在快照套件裡 panic，該套件是以副檔名決定圖片格式的。
- `SavePNG` 的線上備援（本機 Chrome／Chromium 渲染失敗時預設仍會使用）現在 60 秒放棄，回應最多讀 64 MiB。過去用的是沒有 timeout 的 `http.Client{}`——伺服器接了連線然後不講話就會永遠等下去——以及對遠端回應無上限的 `io.ReadAll`。

### `isr`

- `DT.From`、`Col`、`Row`、`Push`、`UseDL`、`UseDT` 遇到不支援的型別或讀檔失敗時不再結束程式，改為回傳記錄了錯誤、可繼續使用的物件，串接結束後可用 `Err()` 或 `PopErr()` 檢查。
- 修正 `DT.From(map[int]any{...})` 永遠產生空表格。鍵被直接轉成字串，`0` 變成 `"0"`，而 `AppendRowsByColIndex` 要的是 Excel 式的欄位索引，因此每個鍵都被拒絕。現在改用 `Row` 路徑相同的轉換，`0` 就是 A 欄。負數的鍵沒有對應欄位，會被回報。

### `gplot`

- `CreateHistogram` 用零值設定不再 panic：`Bins` 為 0 或負數時採用預設值 10。`CreateLineChart` 與 `CreateStepChart` 遇到無法繪製的序列（例如含 `NaN`）時改為記錄警告並略過該序列，不再 panic。
- 另外四個繪圖呼叫遇到一般的錯誤輸入也不再 panic。`CreateBarChart` 沒有 `XAxis` 時（零值設定就是這樣）改為比照 `plot.CreateBarChart` 把長條編號成 1、2、3……，不再在 gonum 的 `NominalX` 裡當掉。`CreateFunctionPlot` 收到 `nil` 函式時記錄警告並回傳 `nil`。`CreateHeatmapChart` 遇到比第 0 列短的列時記錄警告、指出第一個這樣的列並回傳 `nil`（比較長的列照常繪製，多出來的值會被忽略），`Colors` 為負數時比照 0 採用預設值 20。

### `py`

- IPC 監聽失敗改為記錄警告並讓伺服器保持關閉，不再結束程式。
- `ipc.WriteMessage` 在寫入任何位元組之前，拒絕超過讀取端上限（256 MiB）的訊息。過去會寫出一個對端會拒絕的長度，超過 4 GiB 時前綴還會被截斷，讓對端之後每一則訊息都解框錯位。
- 建立 Python 環境的目錄權限改為 0o755，不再是 0777。

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
