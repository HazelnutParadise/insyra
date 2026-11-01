## ADDED Requirements

### Requirement: TEXT and FORMAT_DATE read Go and Excel formats

`TEXT` and `FORMAT_DATE` SHALL read a format written the Go way or the Excel way, deciding from the format alone: a Go `fmt` verb is Go; a digit from 1 to 9 is a Go date layout; `0` or `#` with no letters is an Excel number code; other text with letters is a date format.

#### Scenario: Both spellings
- **WHEN** `TEXT(1.5, '%.2f')`, `TEXT(1.5, '0.00')`, `FORMAT_DATE(d, '2006-01-02')`, `FORMAT_DATE(d, 'yyyy-mm-dd')`, `TEXT(d, '15:04')` and `TEXT(d, 'hh:mm')` are evaluated for 2024-03-09 14:05
- **THEN** they give `1.50`, `1.50`, `2024-03-09`, `2024-03-09`, `14:05` and `14:05`

### Requirement: An Excel number code rounds half away from zero

An Excel number code SHALL round half away from zero on the shortest decimal text of the number. Go verbs SHALL keep their own rounding.

#### Scenario: Ties
- **WHEN** `TEXT(2.5, '0')` and `TEXT(1.005, '0.00')` are evaluated
- **THEN** they give `3` and `1.01`

### Requirement: A format that cannot be read is an error

A date format holding an ASCII letter that is not a code, or no code at all, SHALL be an error naming it, and SHALL NOT come back as its own text. A value of the other kind than the format SHALL be an error. Under an Excel code or a date format, a nil value SHALL give nil.

#### Scenario: A typo
- **WHEN** `FORMAT_DATE(d, 'yyyy-qq')` is evaluated
- **THEN** the error names `qq`

#### Scenario: The wrong kind of value
- **WHEN** `TEXT(1.5, 'yyyy')` is evaluated
- **THEN** it is an error saying the value is not a date
