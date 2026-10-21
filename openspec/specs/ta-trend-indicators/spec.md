# ta-trend-indicators Specification

## Purpose
Trend indicators (Aroon, TRIX, KAMA, Ichimoku) computed with their standard definitions, matching TA-Lib where it has them and using only information known on each row.

## Requirements

### Requirement: Trend indicators follow their definitions
`ta.Aroon`, `ta.TRIX` and `ta.KAMA` SHALL equal TA-Lib's AROON and AROONOSC, TRIX and KAMA on the same series, `ta.TRIX` SHALL also return the EMA of TRIX as Signal, and `ta.KAMA` SHALL take the fast and slow periods as arguments.

#### Scenario: Aroon against TA-Lib
- **WHEN** n is 14 on a series of bars
- **THEN** Up, Down and Oscillator equal TA-Lib's values on every row where TA-Lib has one

### Requirement: Ichimoku gives the cloud in force on each row
`ta.Ichimoku` SHALL return Tenkan and Kijun as the midpoints of the highest high and lowest low of their rows including the current one, and SenkouA and SenkouB on row t as the cloud computed from row t minus Displacement, so that no value of those four lines uses a later row.

#### Scenario: A later change does not move the cloud
- **WHEN** the bars from row 60 on are changed
- **THEN** the cloud on every row before row 86 is the same as before

### Requirement: Unusable input is refused
Each trend function SHALL return an error naming the row index for a missing, non-numeric or non-finite value and for a high below its low, and an error for a period below 1, a slow period not above the fast one and a negative Displacement.

#### Scenario: Slow not above fast
- **WHEN** fast is 5 and slow is 5
- **THEN** an error is returned

### Requirement: Ichimoku returns the lagging span
`ta.Ichimoku` SHALL return Chikou, a list named "Chikou" as long as the input, whose value on row t is the close on row t plus Displacement, and NaN for the last Displacement rows of each segment. The close argument SHALL be validated.

#### Scenario: The close drawn back
- **WHEN** Displacement is 26 on 80 rows
- **THEN** Chikou on row 10 is the close on row 36 and Chikou on rows 54 to 79 is NaN

### Requirement: Ichimoku computes data segments on their own
`IchimokuOptions.SegmentStarts` SHALL list rows where a new data segment starts, as in the RSI and KD options, and every line SHALL be computed on each segment alone, so the cloud and Chikou never take a value from another segment. An empty list SHALL leave all lines as without the option.

#### Scenario: A halt in the middle
- **WHEN** SegmentStarts is 50 and Displacement is 10
- **THEN** the cloud on rows 50 to 59 is NaN and Chikou on rows 40 to 49 is NaN
