## ADDED Requirements

### Requirement: A date has a type that can be compared exactly
`quant.QuantileSort` SHALL group dates by exact value, comparing a string as given with its whitespace and a time.Time by instant, and SHALL return an error for a date that is not a string, time.Time, bool, integer or float, or is NaN, so no two different dates share a group.

#### Scenario: Dates that differ only by whitespace
- **WHEN** a date column holds "2026-01-02" and " 2026-01-02 "
- **THEN** they are two dates, each sorted on its own

#### Scenario: Instants outside the UnixNano range
- **WHEN** a date column holds 0001-01-01T00:00:00Z and 0585-07-21T23:34:33.709551616Z
- **THEN** they are two dates, and the same instant in different time zones is one date

#### Scenario: A slice as a date
- **WHEN** a date cell is a slice
- **THEN** an error is returned
