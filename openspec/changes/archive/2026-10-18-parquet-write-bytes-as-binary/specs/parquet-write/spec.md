## ADDED Requirements

### Requirement: A column of byte slices is written as Binary
When every non-nil cell of a column is a `[]byte`, `Write` SHALL store the column as Parquet `Binary` and a later read SHALL return the same bytes.

#### Scenario: Bytes round trip
- **WHEN** a column holding `A-01`, `ff 00` and a nil is written and read back
- **THEN** the cells are the same bytes and the nil is nil

### Requirement: A byte slice among other kinds is written as text
When a column holds `[]byte` cells together with other kinds of value, `Write` SHALL store a `[]byte` that is valid UTF-8 as that text.

#### Scenario: Bytes beside text
- **WHEN** a column holding `x` and the bytes of `A-01` is written and read back
- **THEN** the second cell is `A-01`
