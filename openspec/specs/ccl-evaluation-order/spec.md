# ccl-evaluation-order Specification

## Purpose
CCL evaluates only what it needs to, and rejects an expression whose shape does not say what it appears to say. A guard written to prevent an error must run before the thing it guards, an operator must bind where a reader expects it to, and a typo in an argument list must not compile into a different question.

## Requirements

### Requirement: Logical operators short-circuit

`&&` SHALL evaluate its right operand only when the left operand is true; `||` SHALL evaluate its right operand only when the left operand is false. A chain of the same operator SHALL behave identically whether the parser folded it or nested it.

#### Scenario: A guard prevents the error it guards against
- **WHEN** 欄 B 含有 0，求值 `B != 0 && A / B > 1`
- **THEN** 回傳 false 而不是 `division by zero`

#### Scenario: Folded and nested chains agree
- **WHEN** 求值 `false && (1/0 > 0) && true`
- **THEN** 回傳 false，且與同一運算式的巢狀二元形式結果相同

### Requirement: CASE evaluates only the selected branch

`CASE` SHALL evaluate each condition in order and SHALL evaluate a value expression only when its condition is the one selected. Conditions after the selected one SHALL NOT be evaluated.

#### Scenario: A division guarded by CASE
- **WHEN** 欄 B 含有 0，求值 `CASE(B != 0, A / B, nil)`
- **THEN** B 為 0 的列得到 nil，其餘列得到商，且不回報錯誤

### Requirement: Argument lists require commas

A function call's arguments SHALL be separated by exactly one comma. A missing comma, a trailing comma and a doubled comma SHALL each be a compile error naming the position.

#### Scenario: Missing comma
- **WHEN** 編譯 `SUM(A B)`
- **THEN** 回傳編譯錯誤，SHALL NOT 當成 `SUM(A, B)`

#### Scenario: Trailing comma
- **WHEN** 編譯 `IF(A > 15, 1, 0,)`
- **THEN** 回傳編譯錯誤

### Requirement: `&` binds looser than addition

The concatenation operator `&` SHALL have lower precedence than `+` and `-` and higher precedence than the comparison operators, as in Excel. The operator precedence table, including the binding of unary minus relative to `^` and the left-associativity of `^`, SHALL be documented.

#### Scenario: Concatenating the result of a sum
- **WHEN** 求值 `'a' & 1 + 2`
- **THEN** 得到 `"a3"`
