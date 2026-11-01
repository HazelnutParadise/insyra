## ADDED Requirements

### Requirement: Regime durations include the still-going stretch

`ta.RegimeDurations` SHALL return, for every regime of the given kind, its length with observed true, and for each segment whose last regime is followed by a stretch of that kind, the rows from the last turning point to the most extreme price after it in that segment (lowest for bear, highest for bull) with observed false, leaving out a stretch of 0 rows.

#### Scenario: Two segments

- **WHEN** a bear market is still running at the end of the first of two segments
- **THEN** its length to its lowest price in the first segment is returned with observed false

### Requirement: Regime durations refuse regimes that do not fit the prices

`ta.RegimeDurations` SHALL refuse an unknown kind, no prices, a regime outside the prices or whose length is not End minus Start, regimes out of order or not consecutive within a segment, and a segment start outside the prices.

#### Scenario: Only the bull regimes

- **WHEN** the bear regimes are left out of the list
- **THEN** it returns an error instead of a wrong still-going stretch
