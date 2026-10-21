## MODIFIED Requirements

### Requirement: The cut-offs are options
`ta.CandlePatterns` SHALL take the doji body, the small and long body shares, the long and short shadow ratios and the trend look-back from CandleOptions, with defaults of 0.1, 0.3, 0.6, 2, 2 and 5 rows, and SHALL return an error for a negative option, a share above 1 and an open or close outside the bar's range.

#### Scenario: A looser doji
- **WHEN** a body is 25% of the range and DojiBody is 0.3
- **THEN** Doji is 1 on that row, and 0 with the default

## ADDED Requirements

### Requirement: The two shadow rules have their own ratios
`ta.CandlePatterns` SHALL require a hammer's or star's long shadow to be at least the body times LongShadowRatio, and the opposite shadow to be at most the long shadow divided by ShortShadowRatio, and a soldier's or crow's shadow to be at most the body divided by ShortShadowRatio.

#### Scenario: A stricter short shadow
- **WHEN** a hammer's upper shadow is 0.4, its lower shadow 2.0, and ShortShadowRatio is 10
- **THEN** Hammer is 0 on that row, and 1 with the default

### Requirement: Candle patterns restart at each segment start
`ta.CandlePatterns` SHALL treat each row in CandleOptions.SegmentStarts as the first row of a new data segment and compute every segment on its own, so no pattern uses a bar of an earlier segment and a row lacking the bars a pattern needs is 0; an empty list SHALL leave the result unchanged and a start that is negative or not below the number of rows SHALL be an error.

#### Scenario: A hammer after a segment start
- **WHEN** a fall is followed by a hammer bar and a segment starts on the hammer's row
- **THEN** Hammer is 0 on that row, and 1 without the start
