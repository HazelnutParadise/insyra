## ADDED Requirements

### Requirement: Regime durations include the still-going stretch

`ta.RegimeDurations` SHALL return, for every regime of the given kind, its length with observed true, and for each segment whose last regime is followed by a stretch of that kind, the rows from the last turning point to the segment's last row with observed false, leaving out a stretch of 0 rows. It SHALL refuse an unknown kind, fewer than one row, a regime outside the rows, and a segment start outside them.

#### Scenario: Two segments

- **WHEN** a bear market is still running at the end of the first of two segments
- **THEN** its length to the first segment's last row is returned with observed false
