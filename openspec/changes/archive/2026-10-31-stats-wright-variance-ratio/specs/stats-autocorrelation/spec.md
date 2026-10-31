## ADDED Requirements

### Requirement: Autocorrelation results print as one table

`Autocorrelation` SHALL return `AutocorrelationResults`, a named slice of its per-lag results whose `String` is "Autocorrelation" followed by a header of the field names and one row per lag, and whose `Show` prints it; it SHALL be assignable to a plain `[]AutocorrelationResult`.

#### Scenario: Three lags

- **WHEN** Autocorrelation is called with maxLag 3 and the result is shown
- **THEN** it prints "Autocorrelation", a header and three rows
