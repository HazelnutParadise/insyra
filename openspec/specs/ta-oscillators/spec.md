# ta-oscillators Specification

## Purpose
Oscillator and momentum indicators (CCI, momentum, rate of change, ultimate oscillator, stochastic RSI, AR/BR, CR, force index, Elder-ray) with TA-Lib's or the standard definitions.

## Requirements

### Requirement: Oscillators follow their definitions
`ta.CCI`, `ta.Momentum`, `ta.ROC`, `ta.UltimateOscillator` and `ta.StochRSI` SHALL equal TA-Lib's CCI, MOM, ROC, ULTOSC and STOCHRSI, `ta.ROC` SHALL return a percentage, and `ta.StochRSI` SHALL smooth with the same KDOptions as `ta.KD`.

#### Scenario: A percentage
- **WHEN** the price rises from 100 to 105 over one row
- **THEN** ROC is 5

### Requirement: Taiwanese sentiment indicators are available
`ta.ARBR` SHALL return AR as 100 times the sum of high minus open over the sum of open minus low and BR as 100 times the sum of the rise of the high above the previous close over the fall of the low below it, over n rows, and `ta.CR` SHALL compare each day with the previous row's midpoint, chosen by CROptions.Midpoint from (high + low) / 2, (2 close + high + low) / 4, (high + low + close) / 3 and (open + high + low + close) / 4.

#### Scenario: The open is needed for one midpoint
- **WHEN** the midpoint is CRMidOHLC and no open is given
- **THEN** an error names CROptions.Open

### Requirement: Elder's indicators are available
`ta.ForceIndex` SHALL return the n-row EMA of the price change times the volume, and `ta.ElderRay` the high and the low minus the n-row EMA of the close.

#### Scenario: Force with n 1
- **WHEN** n is 1
- **THEN** the result is the raw change times volume from row 1

### Requirement: Unusable input is refused
Each oscillator function SHALL return an error naming the row index for a missing, non-numeric or non-finite value, and an error for a period below 1, windows that are not in increasing order, a high below its low, an unknown midpoint or method, and series of different lengths.

#### Scenario: Windows out of order
- **WHEN** the short window is 14 and the medium is 7
- **THEN** an error is returned

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
