## ADDED Requirements

### Requirement: CCL reports what kind of value a cell holds

`ISBLANK` SHALL be true for `nil` and empty text and false for `NaN`. `ISNUM` SHALL be true for a value of a number type other than `NaN`, and false for text. `ISTEXT` and `ISBOOL` SHALL follow the value's type. `ISDATE` SHALL be true for a date and for text that reads as one. `ISERROR` SHALL be true when its argument cannot be computed on that row, and an undefined function or a missing column SHALL still be an error.

#### Scenario: A blank and a NaN

- **WHEN** a cell holds `NaN`
- **THEN** `ISBLANK` is false and `ISNA` is true

#### Scenario: A number written as text

- **WHEN** a cell holds the text `12`
- **THEN** `ISNUM` is false and `ISTEXT` is true

#### Scenario: A failure on a row

- **WHEN** column B holds 0
- **THEN** `ISERROR(A / B)` is true
