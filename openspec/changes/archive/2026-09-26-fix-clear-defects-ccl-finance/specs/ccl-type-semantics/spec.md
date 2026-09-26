## ADDED Requirements

### Requirement: A literal that cannot be represented is an error

數字字面值 SHALL 在超出 float64 範圍時回報編譯錯誤，SHALL NOT 靜默變成 `+Inf`。指數形式（`1e5`、`1.5e-3`、`2E+3`）SHALL 是合法的數字字面值；`e` 後面沒有數字時 SHALL 仍視為識別字，讓名為 `E` 或 `E1` 的欄位不受影響。

#### Scenario: A literal too large for a float64
- **WHEN** 運算式是 400 個 9
- **THEN** 回報超出範圍的編譯錯誤，而不是每列都得到 `+Inf`

#### Scenario: Exponent notation
- **WHEN** 運算式是 `1e5 + 1`
- **THEN** 得到 100001

### Requirement: A format that does not fit is an error

`TOSTR`／`TEXT` 的兩引數形式 SHALL 在格式與值不相符時回報錯誤，SHALL NOT 把 `fmt` 的錯誤標記（`%!d(...)`、`%!(NOVERB)`）當成結果寫進儲存格。

#### Scenario: A verb that does not fit
- **WHEN** `TOSTR(1.5, '%d')`
- **THEN** 回報錯誤，指出格式與值的型別
