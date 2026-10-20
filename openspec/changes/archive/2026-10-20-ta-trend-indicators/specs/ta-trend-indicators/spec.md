## ADDED Requirements

### Requirement: Trend indicators follow their definitions
`ta.Aroon`, `ta.TRIX` and `ta.KAMA` SHALL equal TA-Lib's AROON and AROONOSC, TRIX and KAMA on the same series, `ta.TRIX` SHALL also return the EMA of TRIX as Signal, and `ta.KAMA` SHALL take the fast and slow periods as arguments.

#### Scenario: Aroon against TA-Lib
- **WHEN** n is 14 on a series of bars
- **THEN** Up, Down and Oscillator equal TA-Lib's values on every row where TA-Lib has one

### Requirement: Ichimoku gives the cloud in force on each row
`ta.Ichimoku` SHALL return Tenkan and Kijun as the midpoints of the highest high and lowest low of their rows including the current one, and SenkouA and SenkouB on row t as the cloud computed from row t minus Displacement, so that no value uses a later row, and SHALL NOT return the lagging line.

#### Scenario: A later change does not move the cloud
- **WHEN** the bars from row 60 on are changed
- **THEN** the cloud on every row before row 86 is the same as before

### Requirement: Unusable input is refused
Each trend function SHALL return an error naming the row index for a missing, non-numeric or non-finite value and for a high below its low, and an error for a period below 1, a slow period not above the fast one and a negative Displacement.

#### Scenario: Slow not above fast
- **WHEN** fast is 5 and slow is 5
- **THEN** an error is returned
