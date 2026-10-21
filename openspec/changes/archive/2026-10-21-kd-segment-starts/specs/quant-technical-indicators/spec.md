## ADDED Requirements

### Requirement: KD restarts at each segment start
`ta.KD` SHALL treat each row listed in KDOptions.SegmentStarts as the first row of a new data segment and compute every segment on its own, for both smoothing methods, so K, D and the high-low windows start again and a segment shows NaN through its own warm-up; an empty list SHALL leave the result unchanged.

#### Scenario: A segment equals KD of that segment alone
- **WHEN** KD is given a segment start at row index 20
- **THEN** rows from 20 equal KD of rows 20 onward taken alone, and rows before 20 equal KD of rows 0 to 19 alone

#### Scenario: A segment shorter than the warm-up
- **WHEN** a segment has fewer rows than the warm-up needs
- **THEN** its K and D are NaN on every row

### Requirement: KD refuses a segment start outside the series
`ta.KD` SHALL return an error naming the row index for a segment start that is negative or not below the number of rows.

#### Scenario: A start past the end
- **WHEN** a segment start equals the number of rows
- **THEN** an error names that row index
