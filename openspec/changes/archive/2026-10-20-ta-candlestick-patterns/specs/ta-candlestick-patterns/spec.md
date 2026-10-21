## ADDED Requirements

### Requirement: Candlestick patterns are flagged on the row that completes them
`ta.CandlePatterns` SHALL return one list per pattern (doji, hammer, hanging man, inverted hammer, shooting star, engulfing, harami, piercing line, dark cloud cover, morning star, evening star, three white soldiers, three black crows), as long as the bars, holding +1, -1 or 0, with each flag on the row whose close completes the pattern.

#### Scenario: A hammer
- **WHEN** a bar with a small body and a long lower shadow follows a fall
- **THEN** Hammer is 1 on that row and HangingMan is 0

### Requirement: The cut-offs are options
`ta.CandlePatterns` SHALL take the doji body, the small and long body shares, the shadow ratio and the trend look-back from CandleOptions, with defaults of 0.1, 0.3, 0.6, 2 and 5 rows, and SHALL return an error for a negative option, a share above 1 and an open or close outside the bar's range.

#### Scenario: A looser doji
- **WHEN** a body is 25% of the range and DojiBody is 0.3
- **THEN** Doji is 1 on that row, and 0 with the default

### Requirement: Unusable input is refused
`ta.CandlePatterns` SHALL return an error naming the row index for a missing, non-numeric or non-finite price and for a high below its low, and an error for series of different lengths.

#### Scenario: An open outside the range
- **WHEN** an open is above the high
- **THEN** an error names its row index
