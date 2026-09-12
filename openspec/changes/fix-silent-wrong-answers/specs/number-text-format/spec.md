## ADDED Requirements

### Requirement: A number's text does not depend on the architecture

把數字轉成文字的程式碼 SHALL NOT 使用 `int(f)` 或 `int64(f)` 把可能超出範圍的浮點數轉成整數，因為 Go 對超出範圍的轉換沒有定義結果（amd64 得到 MinInt64，arm64 飽和到 MaxInt64）。轉換前 SHALL 先確認值落在 `int64` 範圍內。同一個值在 amd64 與 arm64 上 SHALL 產生相同的文字。

#### Scenario: A float at the edge of int64
- **WHEN** 對 `2^63` 呼叫 `FormatValue`
- **THEN** 兩種架構都得到指數形式，而不是在其中一種得到比實際值少 1 的整數
