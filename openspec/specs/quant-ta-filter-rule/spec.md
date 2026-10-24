# quant-ta-filter-rule Specification

## Purpose
Alexander filter rule as a position series, so a study of whether price trends can be traded (Fama and Blume 1966) gets its trading rule without hand-writing it.
## Requirements
### Requirement: FilterRule returns a position at every row

`ta.FilterRule(prices, threshold, opts...)` SHALL return one position per row: 1 from the row that confirms a low until the row that confirms the next high, -1 (0 with LongOnly) from that row until the next confirmed low, and 0 before the first confirmed turning point, using the turning points of ZigZag with the same threshold.

#### Scenario: A 5% filter

- **WHEN** the prices are 100, 94, 100, 110, 103, 100 and the threshold is 0.05
- **THEN** the positions are 0, -1, 1, 1, -1, -1, and 0, 0, 1, 1, 0, 0 with LongOnly

### Requirement: FilterRule uses no later price and respects segments

The position on a row SHALL not depend on any later row, so cutting the series short leaves the positions of the rows that remain unchanged. A row listed in SegmentStarts SHALL start a new segment whose first position is 0, and nothing SHALL be carried over the break.

#### Scenario: A break at row 3

- **WHEN** the prices above are given with SegmentStarts [3]
- **THEN** the positions are 0, -1, 1, 0, -1, -1

### Requirement: FilterRule refuses unusable input

`ta.FilterRule` SHALL return an error for a threshold outside (0, 1) or not a number, a nil list or a price that is not positive and finite, more than one options value, and a segment start outside the rows.

#### Scenario: A threshold of 1

- **WHEN** the threshold is 1
- **THEN** an error asks for a fraction between 0 and 1

