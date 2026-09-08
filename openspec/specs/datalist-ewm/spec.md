# datalist-ewm Specification

## Purpose
`DataList` 的指數加權統計：以 `Alpha`／`Span`／`HalfLife` 三選一指定衰減，`Adjust`、`Bias`、`MinObs` 語意與 pandas `Series.ewm` 相同，由 pandas 產生的 fixture 釘住結果；`DataTable.EWMCol` 為欄位版。

## Requirements
### Requirement: Exponentially weighted mean, variance, and standard deviation

`DataList` SHALL 提供 `EWM(opts EWMOptions) *EWMDataList`，`EWMDataList` SHALL 提供 `Mean()`、`Var()`、`Std()` 三個 reducer，各回傳與來源等長的 `*DataList`。`EWMOptions{Alpha, Span, HalfLife float64; Adjust bool; Bias bool; MinObs int}` SHALL 恰好指定 `Alpha`（`0 < Alpha <= 1`）、`Span`（`>= 1`）、`HalfLife`（`> 0`）三者之一，換算與 pandas 相同：`alpha = 2/(span+1)`、`alpha = 1 − exp(ln 0.5 / halflife)`。`Adjust`、`Bias` 的語意 SHALL 與 pandas `Series.ewm(adjust=…).var(bias=…)` 相同；`MinObs <= 0` 視為 1。非數值或 nil 的格子 SHALL 被略過且不重置權重累積，有效觀察數未達 `MinObs` 的位置輸出 nil。`DataTable` SHALL 提供 `EWMCol(col string, opts EWMOptions) *EWMDataList`，欄位參照規則與 `RollingCol` 相同。

#### Scenario: Matches pandas on the fixture corpus

- **WHEN** 對 `testdata/window_fixtures.json` 中 `ewm_mean`、`ewm_var`、`ewm_std` 各案例（涵蓋 alpha／span／halflife、adjust 真假、bias 真假、含 nil 的輸入）呼叫對應 reducer
- **THEN** 每個輸出值與 pandas 產生的 `expected` 在 1e-9 內相等，nil 位置一致

#### Scenario: Exactly one decay parameter

- **WHEN** `EWMOptions` 未指定任何衰減參數，或同時指定兩個以上
- **THEN** `EWM` 發出 warning，所有 reducer 回傳空的 `DataList`，不 panic

#### Scenario: Adjust false is the recursive form

- **WHEN** `Adjust: false`、`Alpha: 0.5`、輸入 `[1, 2, 3]`
- **THEN** `Mean()` 為 `[1, 1.5, 2.25]`

#### Scenario: MinObs suppresses early output

- **WHEN** `MinObs: 3`，輸入 5 個數值
- **THEN** 前兩個輸出為 nil，之後為數值

### Requirement: EWM appears on the interfaces

`IDataList` SHALL 宣告 `EWM(opts EWMOptions) *EWMDataList`；`IDataTable` SHALL 宣告 `EWMCol(col string, opts EWMOptions) *EWMDataList`。

#### Scenario: Interface satisfaction compiles

- **WHEN** 套件編譯
- **THEN** `*DataList` 與 `*DataTable` 仍分別滿足 `IDataList` 與 `IDataTable`

