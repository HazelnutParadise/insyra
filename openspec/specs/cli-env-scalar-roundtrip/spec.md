# cli-env-scalar-roundtrip Specification

## Purpose
CLI 環境的頂層純量變數在儲存與載入後保留 `float64`／`int64` 型別，不留成 `json.Number`。

## Requirements
### Requirement: Scalar variables keep their numeric type across save and load

`env.Manager.LoadState` SHALL 對頂層純量值套用與 DataList 元素相同的數值轉型：`json.Number` 可轉 `int64` 者為 `int64`，否則為 `float64`，其餘維持原值。字串、布林與 DataList／DataTable 變數 SHALL 不受影響。

#### Scenario: Float round trip

- **WHEN** `SaveState` 存入 `{"s": 1.25}` 後 `LoadState`
- **THEN** `s` 的型別為 `float64`，值為 1.25

#### Scenario: Integer round trip

- **WHEN** 存入 `{"n": int64(7)}`
- **THEN** 讀回為 `int64(7)`

#### Scenario: Command reads a reloaded scalar

- **WHEN** 新 session 中對讀回的 `float64` 變數執行會做型別斷言的命令（測試中以 `quant` 的儲存值為例）
- **THEN** 不因型別不符而失敗

