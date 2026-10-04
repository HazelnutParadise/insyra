## ADDED Requirements

### Requirement: A computed column holds one kind of number

A column written by `AddColUsingCCL`, `EditColByIndexUsingCCL`, `EditColByNameUsingCCL` or a statement of `ExecuteCCL` SHALL hold one kind of number: when the numbers among its values include a `float64` or `float32`, every value of a Go integer type whose magnitude is at most 2^53 SHALL become the `float64` of the same value. An integer of larger magnitude SHALL keep its type and value. A column whose numbers are all integers SHALL be left as computed. Values that are not numbers SHALL be left as computed.

#### Scenario: A literal fallback in a float column
- **WHEN** 欄 `A` 為 `["19.5", "abc", "30"]`，`AddColUsingCCL("r", "COALESCE(TONUM(A), 0)")`
- **THEN** `r` 是 `float64` 的 `[19.5, 0, 30]`

#### Scenario: Integers stay integers
- **WHEN** 欄 `A` 為 `[int64(1), nil, int64(3)]`，`AddColUsingCCL("r", "A + 1")`
- **THEN** `r` 是 `int64` 的 `[2, 1, 4]`

#### Scenario: Only numbers are settled
- **WHEN** 欄 `A` 為 `int64` 的 `[0, 1, 2]`，`ExecuteCCL("NEW('r') = IF(A == 0, 'none', IF(A == 1, 2.5, 3))")`
- **THEN** `r` 是 `["none", 2.5, float64(3)]`

#### Scenario: An integer a float64 cannot hold
- **WHEN** 欄 `A` 為 `[int64(1<<53 + 1), int64(2), int64(3)]`，`AddColUsingCCL("r", "IF(# == 2, 0.5, A)")`
- **THEN** `r` 是 `[int64(1<<53 + 1), float64(2), 0.5]`

#### Scenario: A copied integer column keeps its types
- **WHEN** 欄 `A` 為 `int16` 的 `[1, 2]`，`AddColUsingCCL("b", "A")`
- **THEN** `b` 仍是 `int16` 的 `[1, 2]`
