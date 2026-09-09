## MODIFIED Requirements

### Requirement: Internal types never reach a cell

No internal representation SHALL become a cell value. A column range used as a whole result SHALL resolve to the current row restricted to those columns, the same way a bare column reference resolves to the current row's value. A row range SHALL remain an error, because it names rows but not what they are rows of, and the message SHALL say how to attach it to a column. `LAG` and `LEAD` SHALL accept a row-shaped argument (`@` or a column range); every other sequence function SHALL refuse one, because it does arithmetic on each element.

#### Scenario: A bare column range
- **WHEN** 執行 `AddColUsingCCL("r", "A:B")`
- **THEN** 每一格是該列的 A、B 值（`[]any`），與 `(A:B).#` 相同，SHALL NOT 出現 `ColumnRange`

#### Scenario: A sequence function over the whole row
- **WHEN** 求值 `LAG(@, 1)`
- **THEN** 每一列得到前一列的內容，與 `LAG(@.#, 1)` 相同，第一列為 nil

#### Scenario: A sequence function that needs numbers
- **WHEN** 求值 `CUMSUM(@)`
- **THEN** 回傳錯誤說明整列不是數字

#### Scenario: A bare row range
- **WHEN** 求值 `1:2`
- **THEN** 回傳錯誤，訊息指出要接在欄位上（`A.(1:2)`）

#### Scenario: The consumers of a range are unchanged
- **WHEN** 求值 `SUM(A:C)`、`SUM((A:C).(2:5))` 與 `SUM(A.(0:1))`
- **THEN** 結果與批次 8 之前相同，範圍內的每個儲存格只計入一次
