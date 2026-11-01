## ADDED Requirements

### Requirement: WEEKDAY counts Sunday as one

`WEEKDAY(d)` SHALL return 1 for Sunday through 7 for Saturday, Excel's default.

#### Scenario: Sunday and Saturday
- **WHEN** `WEEKDAY('2024-03-10')` and `WEEKDAY('2024-03-16')` are evaluated
- **THEN** they give 1 and 7

### Requirement: TRIM collapses runs of spaces

`TRIM(s)` SHALL strip white space from both ends and turn each run of the space character inside the text into one space. Other white space inside the text SHALL stay.

#### Scenario: Spaces inside
- **WHEN** `TRIM('  a   b  ')` is evaluated
- **THEN** it gives `'a b'`

### Requirement: COUNT counts numbers and dates, COUNTA counts values

`COUNT` SHALL count numbers and dates, and SHALL NOT count nil, text (including text that reads as a number), a bool or `NaN`. `COUNTA` SHALL count every value that is not nil. The streaming forms SHALL give the same answers.

#### Scenario: A mixed column
- **WHEN** a column holds `[1, 2.5, "4", "text", true, nil]`
- **THEN** `COUNT` gives 2 and `COUNTA` gives 5
