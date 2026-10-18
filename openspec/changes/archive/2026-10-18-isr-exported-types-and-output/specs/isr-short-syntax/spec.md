## ADDED Requirements

### Requirement: isr types are exported
The isr table and list types SHALL be exported as `isr.DataTable` and `isr.DataList`, each embedding the root type under its own name, while `isr.DT` and `isr.DL` stay the entry-point variables.

#### Scenario: a caller names the type
- **WHEN** a caller declares `func f(t *isr.DataTable)` and passes it `isr.DT.From(...)`
- **THEN** the program compiles, and `t.DataTable` is the wrapped `*insyra.DataTable`

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
- **THEN** `To` returns an error and leaves the file unchanged; with `insyra.SheetExistsReplace` it replaces the sheet

#### Scenario: an unwritable target
- **WHEN** the target is another type, nil, a value `CSV` or `JSON` with no `FilePath`, an `Excel` with no `FilePath`, or the table is nil
- **THEN** `To` returns a non-nil error

### Requirement: Row and Col key errors name the problem
When a `Row` or `Col` passed to `DT.From` or `Push` has keys that are neither all column indices nor all `Name` selectors, the error SHALL name the isr type and the first key that is none of the accepted kinds, or say that the kinds are mixed.

#### Scenario: a float key
- **WHEN** `isr.DT.From(isr.Row{"A": 1, 3.5: 2})` is called
- **THEN** `Err()` mentions `isr.Row` and `3.5`
