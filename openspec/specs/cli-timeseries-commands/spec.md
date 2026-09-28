# cli-timeseries-commands Specification

## Purpose
CLI／REPL／DSL 的時間序列命令：`ewm`（alpha／span／halflife 三選一，adjust、bias、minobs 選項）、`rolling` 的配對 reducer `cov <other>` 與 `beta <other>`、`resample`（以 `<col>:<op>[:<name>]` 重用 `groupby` 運算子把時間鍵表彙總成週／月／季／年）。

## Requirements

### Requirement: ewm command

CLI SHALL 提供 `ewm <var> alpha|span|halflife <value> mean|var|std [adjust yes|no] [bias yes|no] [minobs <n>] [as <var>]`。`<var>` SHALL 為 `DataList`；衰減關鍵字三選一並帶數值；reducer 三選一；`adjust`、`bias` 預設 no，`minobs` 預設 0（交由函式庫處理）。結果 SHALL 為等長 `DataList`，存到 `as <var>`，未指定時存到 `$result`，與 `rolling` 相同。不合法的衰減值、未知 reducer、未知選項、非 `DataList` 變數 SHALL 回傳含 `ewm:` 前綴的錯誤。

#### Scenario: Recursive mean

- **WHEN** 變數 `x` 為 `[1, 2, 3]`，執行 `ewm x alpha 0.5 mean as m`
- **THEN** `m` 為 `[1, 1.5, 2.25]`

#### Scenario: Options are parsed

- **WHEN** 執行 `ewm x span 3 std adjust yes bias yes minobs 2 as s`
- **THEN** 結果等於 `x.EWM(EWMOptions{Span: 3, Adjust: true, Bias: true, MinObs: 2}).Std()`

#### Scenario: Invalid decay is refused

- **WHEN** 執行 `ewm x alpha 0 mean`
- **THEN** 回傳錯誤，不存變數

### Requirement: rolling cov and beta reducers

`rolling` SHALL 接受 `cov <other>` 與 `beta <other>` 兩個 reducer，`<other>` 為另一個 `DataList` 變數名，其餘選項（`minobs`、`center`、`as`）語意不變。`help rolling` 的 Forms SHALL 列出兩個新形式。

#### Scenario: Rolling beta of a scaled benchmark

- **WHEN** `a = 2·b + 1`，執行 `rolling a 3 beta b as rb`
- **THEN** `rb` 前兩格為 nil，其餘為 2

#### Scenario: Missing other argument

- **WHEN** 執行 `rolling a 3 cov`
- **THEN** 回傳提示需要第二個變數的錯誤

### Requirement: resample command

CLI SHALL 提供 `resample <dt> <timecol> weekly|monthly|quarterly|yearly <col>:<op>[:<name>] ... [as <var>]`。`op` SHALL 對應 `groupby` 已有的運算子名稱（`sum mean median min max count first last std var`）。結果 SHALL 為 `DataTable`，存到 `as <var>` 或 `$result`。未知頻率、未知 `op`、格式不是 `col:op[:name]`、時間欄含非 `time.Time` 值 SHALL 回傳含 `resample:` 前綴的錯誤，並包含函式庫的錯誤訊息。

#### Scenario: Monthly OHLCV from a script

- **WHEN** `dt` 有 `Date`（`time.Time`）與 OHLCV 欄，執行 `resample dt Date monthly Open:first High:max Low:min Close:last:MonthClose Volume:sum as m`
- **THEN** `m` 每月一列，欄名為 `Date, Open, High, Low, MonthClose, Volume`

#### Scenario: Unknown op

- **WHEN** 執行 `resample dt Date monthly Close:average`
- **THEN** 回傳列出可用運算子的錯誤

#### Scenario: Non-time column

- **WHEN** `Date` 欄為字串
- **THEN** 回傳含函式庫列號訊息的錯誤

### Requirement: A failed window reducer is an error

`rolling`、`ewm` 與 `expanding` 的 reducer 結果帶有錯誤時，命令 SHALL 回傳含命令名稱前綴與該錯誤的錯誤，SHALL NOT 存任何變數，也 SHALL NOT 印出 `saved as`。`ewm` SHALL 依結果的錯誤判斷失敗，SHALL NOT 以結果長度推斷。

#### Scenario: A window of zero
- **WHEN** 變數 `x` 為 `[1, 2, 3]`，執行 `rolling x 0 mean as m`
- **THEN** 回傳以 `rolling:` 開頭的錯誤，`m` 不存在

#### Scenario: MinObs above the window
- **WHEN** 執行 `rolling x 2 mean minobs 3 as m`
- **THEN** 回傳錯誤，`m` 不存在

### Requirement: fillna counts only the chosen kind of missing toward limit

With `missing nan` or `missing nil`, `fillna`'s `ffill` and `bfill` SHALL skip the cells of the other kind: they SHALL NOT be filled, SHALL NOT provide a value to copy, and SHALL NOT count toward `limit`. Without a `limit`, and for `mean`, `median`, `mode` and `interpolate`, the result SHALL be what it was.

#### Scenario: A nil before the NaN does not use up the limit
- **WHEN** 變數 `x` 為 `[1, nil, NaN]`，執行 `fillna x ffill limit 1 missing nan as f`
- **THEN** `f` 為 `[1, nil, 1]`

#### Scenario: The same backwards for nil
- **WHEN** 變數 `x` 為 `[nil, NaN, 5]`，執行 `fillna x bfill limit 1 missing nil as f`
- **THEN** `f` 的第 1 格為 5、第 2 格仍為 NaN、第 3 格為 5

#### Scenario: Only target cells count toward the limit
- **WHEN** 變數 `x` 為 `[1, NaN, nil, NaN]`，執行 `fillna x ffill limit 1 missing nan as f`
- **THEN** `f` 為 `[1, 1, nil, NaN]`

#### Scenario: A table column follows the same rule
- **WHEN** 表格欄 `v` 為 `[1, nil, NaN]`，執行 `fillna t ffill cols name:v limit 1 missing nan as f`
- **THEN** `f` 的 `v` 欄為 `[1, nil, 1]`
