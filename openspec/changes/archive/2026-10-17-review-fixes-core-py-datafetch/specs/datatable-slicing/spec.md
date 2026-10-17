## ADDED Requirements

### Requirement: SliceCols pads a short column with nil

When a column that `SliceCols` selects is shorter than the table's longest column, the result SHALL pad that column with `nil` to the table's longest column length, so the result keeps every row of the table, and the source table SHALL NOT change.

#### Scenario: A short column in the range

- **WHEN** `SliceCols(1, 2)` is called on a table whose column `A` holds three values and whose column `B` holds one value, `"x"`
- **THEN** the result holds one column of three values, `"x"`, `nil` and `nil`, and the source table is unchanged
