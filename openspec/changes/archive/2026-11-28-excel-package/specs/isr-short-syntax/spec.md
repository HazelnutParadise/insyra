## MODIFIED Requirements

### Requirement: a table writes to the formats From reads
`(*isr.DataTable).To(target any) error` SHALL write the table to an `isr.CSV`, `isr.JSON` or `isr.Excel` target, given by value or by pointer, applying the target's `OutputOpts`, and SHALL return an error instead of panicking for any target it cannot write.

#### Scenario: writing a CSV file
- **WHEN** `t.To(isr.CSV{FilePath: p})` succeeds
- **THEN** `isr.DT.From(isr.CSV{FilePath: p})` reads back the same columns and rows

#### Scenario: writing into memory
- **WHEN** `t.To(&c)` is called with a `*isr.CSV` or `*isr.JSON` whose `FilePath` is empty
- **THEN** the output is stored in `c.String` or `c.Bytes` and nil is returned

#### Scenario: JSON keys
- **WHEN** a table with named columns is written to JSON
- **THEN** values are keyed by column name, or by column index when `OutputOpts.UseColIndex` is true

#### Scenario: an existing Excel sheet
- **WHEN** the named sheet already exists and `OutputOpts.IfSheetExists` is the zero value
- **THEN** `To` returns an error and leaves the file unchanged; with `excel.SheetExistsReplace` it replaces the sheet

#### Scenario: an unwritable target
- **WHEN** the target is another type, nil, a value `CSV` or `JSON` with no `FilePath`, an `Excel` with no `FilePath`, or the table is nil
- **THEN** `To` returns a non-nil error
