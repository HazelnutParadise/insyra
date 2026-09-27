# cli-env-scalar-roundtrip Specification

## Purpose
CLI 環境的頂層純量變數在儲存與載入後保留 `float64`／`int64` 型別，不留成 `json.Number`。

## Requirements

### Requirement: Scalar variables keep their numeric type across save and load

頂層純量變數 SHALL 在 `SaveState` 與 `RestoreVariables` 之後保留原本的 Go 型別與值：`float64` 仍為 `float64`（包括 `3.0` 這類整數值），`int` 仍為 `int`，`int64` 仍為 `int64`（超過 2^53 也不失真），`bool`、`string`、`time.Time` 亦同。`env.Manager.LoadState` SHALL 繼續回傳已轉型的頂層純量：新格式的純量轉回儲存時的 Go 型別；先前版本寫入的純量依原規則，`json.Number` 可轉 `int64` 者為 `int64`，否則為 `float64`。DataList 與 DataTable 變數在 `LoadState` 中維持儲存形式。

#### Scenario: Float round trip

- **WHEN** `SaveState` 存入 `{"s": 1.25, "w": 3.0}` 後 `RestoreVariables`
- **THEN** `s` 為 `float64(1.25)`，`w` 為 `float64(3)`

#### Scenario: Integer round trip

- **WHEN** 存入 `{"n": int64(7), "i": 7, "big": int64(9007199254740993)}`
- **THEN** 讀回 `n` 為 `int64(7)`，`i` 為 `int(7)`，`big` 為 `int64(9007199254740993)`

#### Scenario: LoadState returns typed scalars

- **WHEN** `SaveState` 存入 `{"s": 1.25, "i": 7}` 後呼叫 `LoadState`
- **THEN** `s` 的 `Data` 為 `float64(1.25)`，`i` 的 `Data` 為 `int(7)`

#### Scenario: Legacy scalars

- **WHEN** 以 `RestoreVariables` 或 `LoadState` 讀取先前版本寫入、`data` 為 JSON 數字 `7` 與 `1.25` 的純量
- **THEN** 讀回為 `int64(7)` 與 `float64(1.25)`

#### Scenario: Command reads a reloaded scalar

- **WHEN** 新 session 中對讀回的 `float64` 變數執行會做型別斷言的命令（測試中以 `quant` 的儲存值為例）
- **THEN** 不因型別不符而失敗
