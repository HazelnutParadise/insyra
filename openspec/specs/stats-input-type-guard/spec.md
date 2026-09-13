# stats-input-type-guard Specification

## Purpose
`stats` 對介面參數的處理契約：任何 `IDataList` 實作都被接受，nil 回錯誤而非 panic。

## Requirements
### Requirement: Interface parameters are converted, not asserted

接受 `insyra.IDataList` 的 `stats` 函式 SHALL 接受任何實作（非 `*insyra.DataList` 時以其 `Data()` 建立副本），nil 時 SHALL 回傳錯誤而 SHALL NOT panic。

#### Scenario: Nil input
- **WHEN** `Correlation(nil, NewDataList(1.0, 2.0), PearsonCorrelation)`
- **THEN** 回傳錯誤，不 panic

