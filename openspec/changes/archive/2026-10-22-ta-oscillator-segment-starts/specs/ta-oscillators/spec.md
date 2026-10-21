## ADDED Requirements

### Requirement: StochRSI and CR restart at each segment start
`ta.StochRSI` SHALL treat each row in KDOptions.SegmentStarts, and `ta.CR` each row in CROptions.SegmentStarts, as the first row of a new data segment and compute every segment on its own, StochRSI's inner RSI included, so nothing is carried across a start and a segment shows NaN through its own warm-up; an empty list SHALL leave the result unchanged.

#### Scenario: A segment equals the indicator of that segment alone
- **WHEN** StochRSI or CR is given a segment start at row index 20
- **THEN** rows from 20 equal the result for rows 20 onward taken alone, and rows before 20 equal the result for rows 0 to 19 alone

### Requirement: StochRSI and CR refuse a segment start outside the series
`ta.StochRSI` and `ta.CR` SHALL return an error naming the row index for a segment start that is negative or not below the number of rows.

#### Scenario: A start past the end
- **WHEN** a segment start equals the number of rows
- **THEN** an error names that row index

### Requirement: StochRSI recovers from an undefined RSI
`ta.StochRSI` SHALL leave a row NaN when its RSI window is undefined and, with KDRecursive, begin the smoothing from 50 on the first row that has a stochastic value instead of staying NaN afterwards.

#### Scenario: A flat start
- **WHEN** the prices are flat for the first rows and then move
- **THEN** K and D are finite from the first row whose raw stochastic value is finite

### Requirement: A window without range adds nothing to the ultimate oscillator
`ta.UltimateOscillator` SHALL treat a window whose true range sums to zero as contributing 0 while the weights still total 7, as TA-Lib does, instead of returning NaN for the row.

#### Scenario: A flat short window
- **WHEN** the price has not moved for the whole short window but has in the longer ones
- **THEN** the value is 100 times (2 times the medium ratio plus the long ratio) over 7, and not NaN
