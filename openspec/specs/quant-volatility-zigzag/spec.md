# quant-volatility-zigzag Specification

## Purpose
Turning points whose reversal threshold follows the volatility of the series and that restart at data breaks, so a pattern study can use them without looking ahead.

## Requirements

### Requirement: A threshold for every row keeps the search causal
`quant.ZigZagBy` SHALL confirm a high or low when the price has moved the threshold of the row of the extreme away from it, SHALL give the same pivots as `ZigZag` for a constant threshold, and a series cut short SHALL give exactly the pivots of the full series whose confirming row lies before the cut.

#### Scenario: A loose threshold at a peak
- **WHEN** a peak at row 1 has a threshold of 0.30 and the price falls 12.5% from it
- **THEN** no pivot is reported at row 1

### Requirement: Nothing is carried over a data break
`quant.ZigZagBy` SHALL start a new search at every row in `ZigZagOptions.Breaks`, SHALL NOT report a pivot confirmed after its segment ended, and SHALL set `Pivot.Segment` to the number of breaks before the pivot.

#### Scenario: Two breaks
- **WHEN** the breaks are rows 200 and 350
- **THEN** a pivot at row 250 has Segment 1 and was confirmed before row 350

### Requirement: Volatility thresholds use only past returns
`quant.VolatilityThresholds` SHALL return for every row the larger of Floor and Multiple times the population standard deviation of the log returns of the last Window rows of its segment, SHALL return Multiple times WarmupVolatility (at least Floor) while fewer than MinRows returns exist, and SHALL restart at every break.

#### Scenario: A flat series
- **WHEN** all prices are 100 and Multiple is 3
- **THEN** the thresholds are the 0.02 floor once MinRows returns exist

### Requirement: Unusable input is refused
`quant.ZigZagBy` and `quant.VolatilityThresholds` SHALL return an error for a threshold outside (0, 1), thresholds of a different length than the prices, a break outside the series, a non-positive Multiple, and a price that is not positive.

#### Scenario: A threshold of 1
- **WHEN** a threshold is 1
- **THEN** an error names its row index

### Requirement: Regimes never cross a data break
`quant.Regimes` SHALL turn the pivots of each segment into regimes on its own, SHALL apply MinPhase and MinCycle inside a segment, and SHALL NOT report two pivots of different segments as an alternation error or join them into a regime.

#### Scenario: Two highs either side of a break
- **WHEN** one segment ends with a high and the next starts with a high
- **THEN** no error is returned and no regime spans the break
