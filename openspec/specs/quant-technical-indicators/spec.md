# quant-technical-indicators Specification

## Purpose
The common technical indicators (moving averages, MACD, Bollinger bands, ATR and KD) computed with TA-Lib's definitions and checked against it.

## Requirements

### Requirement: Moving averages follow TA-Lib
`ta.SMA` SHALL return the mean of the last n values and `ta.EMA` the exponential average with weight 2/(n+1) started from the simple average of the first n values, both NaN until the window is full, and both SHALL equal TA-Lib's results.

#### Scenario: A simple average
- **WHEN** the prices are 1, 2, 3, 4, 5 and n is 3
- **THEN** the result is NaN, NaN, 2, 3, 4

### Requirement: MACD returns three lines
`ta.MACD` SHALL return the fast EMA minus the slow EMA, the EMA of that line over the signal period, and their difference, started as TA-Lib starts them, and SHALL refuse a slow period that is not longer than the fast one.

#### Scenario: Default settings
- **WHEN** fast 12, slow 26 and signal 9
- **THEN** the lines equal TA-Lib's wherever TA-Lib has a value

### Requirement: Bollinger bands use the population deviation
`ta.Bollinger` SHALL return the simple average of the last n prices and bands k population standard deviations above and below it, NaN until the window is full.

#### Scenario: Zero width
- **WHEN** k is 0
- **THEN** the three bands are equal

### Requirement: True range and ATR follow Wilder
`ta.TrueRange` SHALL return the largest of high minus low and the distances from the previous close to the high and the low, NaN on row 0, and `ta.ATR` SHALL start with the mean of the first n true ranges on row n and smooth later rows with weight 1/n.

#### Scenario: A short series
- **WHEN** the series has fewer rows than n
- **THEN** every row is NaN and no error is returned

### Requirement: KD has two smoothing methods
`ta.KD` SHALL compute the raw stochastic value over the last n rows, counting a flat window as 50, and SHALL smooth it either recursively from a start of 50 (KDRecursive) or with moving averages as TA-Lib's STOCH does (KDMovingAverage).

#### Scenario: A close at the top of the range
- **WHEN** RSV is 100 and K was 50
- **THEN** K becomes 66.67 with the default smoothing of 3

### Requirement: Unusable input is refused
Each indicator SHALL return an error naming the row index for a missing or non-finite value, and an error for a period below its minimum, series of different lengths, a high below its low, and an unknown KD method.

#### Scenario: A NaN price
- **WHEN** a price is NaN
- **THEN** an error names its row index

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
