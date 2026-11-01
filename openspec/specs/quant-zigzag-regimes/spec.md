# quant-zigzag-regimes Specification

## Purpose
Finding the turning points of a price series without looking ahead and turning them into bull and bear regimes, the front half of a study of trends.
## Requirements
### Requirement: Turning points are found without looking ahead
`ta.ZigZag` SHALL return the alternating highs and lows of a positive price series that reverse by at least the threshold, each with the row of the extreme and the first row on which the price was the threshold away from it, and a series cut short SHALL give exactly the swings of the full series whose confirming row lies before the cut.

#### Scenario: The worked example
- **WHEN** the prices are 100, 111, 120, 105, 95, 100, 115, 130, 116 and the threshold is 0.10
- **THEN** the swings are low at 0 confirmed at 1, high at 2 confirmed at 3, low at 4 confirmed at 6 and high at 7 confirmed at 8

### Requirement: Swings become bull and bear regimes
`ta.Regimes` SHALL return a bull regime from each low to the next high and a bear regime from each high to the next low with start, end, length and change, SHALL remove a phase shorter than MinPhase with the turning points around it and a cycle shorter than MinCycle, and SHALL return an error for swings that do not alternate.

#### Scenario: A short bear phase
- **WHEN** a bear stretch of 2 rows lies between two bull stretches and MinPhase is 3
- **THEN** the two bull stretches are returned as one

### Requirement: Unusable input is refused
`ta.ZigZag` SHALL return an error for a threshold outside (0, 1), a nil list, a non-finite value and a price that is not positive.

#### Scenario: A zero price
- **WHEN** a price is 0
- **THEN** an error names its row index

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
