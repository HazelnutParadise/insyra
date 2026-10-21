## ADDED Requirements

### Requirement: A return that is not finite is missing data
A simple return of a stock or of the market that is not a finite number, such as the ratio of two finite levels overflowing, SHALL be treated as a missing value, so an event whose window needs it is dropped as a whole and counted in `Missing`.

#### Scenario: A price jump that overflows
- **WHEN** a price goes from 1e-300 to 1e300 inside an event's window
- **THEN** the event is counted in `Missing` and no mean abnormal return is infinite

### Requirement: Each event date is one cell
The event-date column of `CARs` SHALL hold each date as given in `dates` in a single cell, including a date that is a slice, so the table has one row per used event.

#### Scenario: A slice as a date
- **WHEN** an event's date cell is the slice `[2026 1 2]`
- **THEN** the table has one row for that event and its date cell is that slice
