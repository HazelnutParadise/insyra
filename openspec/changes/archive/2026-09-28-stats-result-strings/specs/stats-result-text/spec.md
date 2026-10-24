## ADDED Requirements

### Requirement: Every stats result prints itself the same way

Every exported struct type in `stats` whose name ends in `Result` SHALL declare its own pointer-receiver `String() string` and `Show()` (`Show(startEndRange ...any)` for `FactorAnalysisResult` and `FactorModel`). `Show()` SHALL print `String()` and a newline to standard output. A test SHALL fail when such a type lacks either method.

#### Scenario: A new result type
- **WHEN** an exported `…Result` struct is added without `String`
- **THEN** the test suite fails

#### Scenario: Printing to a writer
- **WHEN** `fmt.Fprintln(w, res)` is called with a result
- **THEN** `w` receives the text `String()` returns

### Requirement: One layout for every result

`String()` SHALL write a title line, then one `  Name: value` line per exported field in declaration order with the fields of an embedded struct in its place. A `nil` pointer or interface field SHALL be left out, and the text SHALL contain no colour codes.

#### Scenario: A field that does not apply
- **WHEN** a result has a `nil` `CI`
- **THEN** its text has no `CI` line

### Requirement: Numbers and long values are written by one rule

A float SHALL be written as a plain decimal from 1e-6 up to 1e21 and in exponent form outside it. A list or table SHALL print whole up to 60 entries, otherwise its first 20 and last 5.

#### Scenario: A large sum
- **WHEN** a field holds 1500000
- **THEN** its text is `1500000`, not `1.5e+06`

#### Scenario: A long list
- **WHEN** a field holds 100 values
- **THEN** only the first 20 and last 5 print, with the total count
