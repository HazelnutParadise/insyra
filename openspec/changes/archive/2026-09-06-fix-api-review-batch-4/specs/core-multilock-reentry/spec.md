## ADDED Requirements

### Requirement: AtomicDoAll inside AtomicDo locks the instances not yet held

當呼叫 goroutine 已持有部分實例的鎖時，`AtomicDoAll`／`AtomicDoN` SHALL 略過這些實例，依固定順序鎖上其餘實例後才執行回呼，且 SHALL NOT 改走不加鎖的 trust-zone 內聯路徑。兩個 goroutine 以鏡像順序巢狀呼叫時仍可能互相等待，與 v0.3.2 相同，需要避免時應從最外層呼叫。

#### Scenario: A nested AppendCols does not race the source list
- **WHEN** 一個 goroutine 反覆執行 `dt.AtomicDo(func(t){ t.AppendCols(col) })`，另一個 goroutine 同時對 `col` 呼叫 `Append`
- **THEN** 以 `-race` 執行時不回報 data race

#### Scenario: The actor not held is locked inside the callback
- **WHEN** 在 actor `a` 的 `AtomicDo` 內呼叫 `AtomicDoN([a, b], f)`
- **THEN** `f` 執行期間 `b` 的鎖已被持有，trust-zone hook 不觸發
