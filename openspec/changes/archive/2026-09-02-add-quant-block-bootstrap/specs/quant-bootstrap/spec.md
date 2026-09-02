# quant-bootstrap Delta Spec

## ADDED Requirements

### Requirement: Block bootstrap resampling of a return series

`quant` SHALL 提供 `BootstrapConfig{Horizon int; BlockSize int; Paths int; Seed uint64; Stationary bool}` 與 `BlockBootstrap(returns insyra.IDataList, cfg BootstrapConfig) (*BootstrapResult, error)`。`BootstrapResult` SHALL 包含 `Returns [][]float64`（`Paths` 列，每列 `Horizon` 個重抽樣的每期報酬）與 `Equity [][]float64`（`Paths` 列，每列 `Horizon+1` 個值，`Equity[p][0] == 1.0`，`Equity[p][t] == Equity[p][t-1] * (1 + Returns[p][t-1])`）。`Stationary` 為 false 時 SHALL 使用 moving block bootstrap：每個區塊長度固定為 `BlockSize`，起點在 `[0, n-BlockSize]` 內均勻抽取；為 true 時 SHALL 使用 stationary bootstrap：區塊長度服從平均為 `BlockSize` 的幾何分布，起點在 `[0, n)` 內均勻抽取，索引超出尾端時 SHALL 環狀接回序列開頭。

#### Scenario: Output shapes

- **WHEN** 以 `Horizon: 252, BlockSize: 20, Paths: 100` 呼叫 `BlockBootstrap`
- **THEN** `Returns` 為 100 × 252，`Equity` 為 100 × 253，每列 `Equity[p][0]` 為 1.0

#### Scenario: Blocks are contiguous slices of the input

- **WHEN** 以 `BlockSize == len(returns)`、`Horizon == len(returns)`、`Stationary: false` 呼叫
- **THEN** 每條 `Returns[p]` 都等於原始序列，`Equity[p]` 等於原始序列的複利曲線

#### Scenario: Equity compounds from 1.0

- **WHEN** 輸入為常數報酬 `r` 的序列
- **THEN** 每條路徑 `Equity[p][t]` 等於 `(1+r)^t`（浮點容差內）

#### Scenario: Stationary block lengths have the requested mean

- **WHEN** 以 `Stationary: true`、`BlockSize: 20` 產生足夠多的區塊
- **THEN** 觀察到的平均區塊長度接近 20（在抽樣誤差內）

#### Scenario: Stationary indexing wraps around

- **WHEN** 以 `Stationary: true` 抽到的區塊跨越序列尾端
- **THEN** 區塊在尾端之後接續 `returns[0]`，不截斷也不回錯

### Requirement: Bootstrap output is reproducible from the seed

給定相同的 `returns` 與 `cfg`（含 `Seed`），`BlockBootstrap` SHALL 回傳逐位元相同的 `Returns` 與 `Equity`。`Seed` SHALL 永遠生效，零值即為種子 0，不存在「未設定即隨機」的模式。亂數 SHALL 只依賴以 `Seed` 初始化的 PCG 序列，整數與浮點的縮減 SHALL 在 `quant` 內完成。

#### Scenario: Same seed, same output

- **WHEN** 以相同輸入與 `Seed: 42` 呼叫兩次
- **THEN** 兩次的 `Returns` 與 `Equity` 逐值相等

#### Scenario: Different seeds diverge

- **WHEN** 以 `Seed: 1` 與 `Seed: 2` 各呼叫一次
- **THEN** 兩次的 `Returns` 不相等

#### Scenario: Golden output is pinned

- **WHEN** 以固定的小型輸入與固定 seed 呼叫
- **THEN** 前幾個重抽樣值等於測試中記錄的常數

### Requirement: Bootstrap configuration is validated, not defaulted

`BlockBootstrap` SHALL 對下列情況回傳指出欄位的錯誤而非套用預設值：`Horizon <= 0`、`Paths <= 0`、`BlockSize < 1`、`BlockSize > len(returns)`、`returns` 為空或為 nil。`returns` 含任何無法轉為數值、NaN 或 Inf 的元素時 SHALL 回傳指出列號的錯誤，且 SHALL NOT 以 0 或其他值替代。

#### Scenario: Non-positive horizon is refused

- **WHEN** 以 `Horizon: 0` 呼叫
- **THEN** 回傳錯誤，錯誤訊息提及 `Horizon`

#### Scenario: Block longer than the series is refused

- **WHEN** 序列有 50 筆而 `BlockSize: 51`
- **THEN** 回傳錯誤，錯誤訊息提及 `BlockSize`

#### Scenario: Unreadable value is refused with its row

- **WHEN** `returns` 第 3 個元素為字串 `"n/a"`
- **THEN** 回傳錯誤，錯誤訊息包含列號 3，且不產生任何路徑

### Requirement: Percentile bands over a path matrix

`quant` SHALL 提供 `PercentileBands(paths [][]float64, percentiles []float64) ([][]float64, error)`。對每個時點 `t`，系統 SHALL 取所有路徑在 `t` 的值，以 R type-7 分位數（與 `DataList.Percentile` 相同的定義，`percentiles` 使用 0 到 100 的刻度）計算每個要求的百分位。回傳的 `bands[i]` SHALL 對應 `percentiles[i]`，順序與呼叫者給定的相同，長度等於 `len(paths[0])`。`paths` 為空、各列長度不一致、`percentiles` 為空、或任一百分位不在 `[0, 100]` 內時 SHALL 回傳錯誤。

#### Scenario: Bands agree with DataList.Percentile

- **WHEN** 對隨機路徑矩陣呼叫 `PercentileBands(paths, []float64{5, 25, 50, 75, 95})`
- **THEN** 每個 `bands[i][t]` 等於以第 `t` 欄建立的 `DataList` 呼叫 `Percentile(percentiles[i])` 的結果

#### Scenario: Caller order is preserved

- **WHEN** 以 `percentiles: []float64{95, 5}` 呼叫
- **THEN** `bands[0]` 為 95 百分位序列，`bands[1]` 為 5 百分位序列

#### Scenario: Bands are monotone in the percentile

- **WHEN** 以遞增的百分位呼叫
- **THEN** 對每個 `t`，`bands[i][t] <= bands[i+1][t]`

#### Scenario: Ragged paths are refused

- **WHEN** `paths` 中某列長度與其他列不同
- **THEN** 回傳錯誤，不計算任何帶

#### Scenario: Out-of-range percentile is refused

- **WHEN** `percentiles` 含 101
- **THEN** 回傳錯誤

### Requirement: Docs, skills and changelogs describe the bootstrap API

`Docs/quant.md` SHALL 新增涵蓋 `BootstrapConfig`、`BlockBootstrap`、`BootstrapResult`、`PercentileBands` 的章節與扇形圖用法範例，並說明 seed 永遠生效、`BlockSize` 在兩種方法下的意義、以及區塊長度與樣本數的統計建議。`Docs/README.md`、`README.md`、`README_TW.md` 的 quant 列 SHALL 提及 bootstrap 路徑模擬。`skills/insyra/SKILL.md` SHALL 新增 quant 段落示範這兩個函式。`CHANGELOG.md` 與 `CHANGELOG_TW.md` 的 `## Unreleased` SHALL 各新增 `` ### `quant` `` 條目並連結 issue #199。

#### Scenario: Documentation set is complete

- **WHEN** 變更完成
- **THEN** 上述每個檔案都包含對應的新內容，英文與繁體中文版本同步
