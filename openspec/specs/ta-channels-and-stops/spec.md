# ta-channels-and-stops Specification

## Purpose
Channel and trailing-stop indicators (Keltner, Donchian, SuperTrend, Parabolic SAR, chandelier exit) computed with their standard definitions and checked against TA-Lib or independent code.

## Requirements

### Requirement: Channels follow their definitions
`ta.Keltner` SHALL return the n-row EMA of the close plus and minus k times the atrN-row ATR, `ta.KeltnerClassic` the n-row simple average of the typical price plus and minus the n-row average of high minus low, and `ta.Donchian` the highest high and lowest low of the n rows before each row with their midpoint, each with NaN until the lines exist.

#### Scenario: A breakout of the Donchian channel
- **WHEN** a close is above the highest high of the previous n rows
- **THEN** it is above `Upper` on that row, which does not include the row's own high

### Requirement: Stops follow their definitions
`ta.SuperTrend`, `ta.ParabolicSAR` and `ta.Chandelier` SHALL return the SuperTrend line and direction as the TradingView script defines them, Wilder's parabolic stop and reverse as TA-Lib computes it, and the highest high minus k ATRs and lowest low plus k ATRs of the last n rows.

#### Scenario: Parabolic SAR against TA-Lib
- **WHEN** the start and step are both 0.02 and the cap is 0.2
- **THEN** the values equal TA-Lib's SAR on the same bars

### Requirement: Unusable input is refused
Each channel and stop function SHALL return an error naming the row index for a missing, non-numeric or non-finite value and for a high below its low, and an error for a period below 1, a negative k and series of different lengths.

#### Scenario: A period of 0
- **WHEN** n is 0
- **THEN** an error is returned
