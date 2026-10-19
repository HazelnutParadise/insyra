## ADDED Requirements

### Requirement: Patterns are known only from the row that confirms them
`quant.DetectPatterns` SHALL report each pattern from the row that confirms its last turning point, SHALL measure its breakout as the first later close beyond its line before its expiry, and a series cut short SHALL give the patterns of the full series formed before the cut with any later breakout forgotten.

#### Scenario: A double top
- **WHEN** peaks at 100 and 99 either side of a trough at 85 and a close of 84 on row 36
- **THEN** a PatternDoubleTop breaks down on row 36

### Requirement: A pattern the price cancels never breaks
`quant.DetectPatterns` SHALL report no breakout for a pattern whose price closes beyond the head or the double's peaks, or passes its expiry, before breaking the textbook way.

#### Scenario: A new high
- **WHEN** the price closes at 101 above the 100 peaks before closing at 84
- **THEN** BreakoutAt is -1 and Direction is BreakNone

### Requirement: Tolerances are options
`quant.DetectPatterns` SHALL take how flat a line counts as flat, how alike peaks must be, the longest wait for a breakout, the least separation of a double and the trend look-back from `PatternOptions`, and SHALL NOT join pivots from different segments.

#### Scenario: A wider flat tolerance
- **WHEN** lows rise by 25% of the height and FlatTolerance is 0.3
- **THEN** the pattern is a PatternBox and not an ascending triangle

### Requirement: Fan lines are numbered and spent after the third break
`quant.FanBreaks` SHALL return each close through the line joining a fan's anchor to the next flatter turning point, numbered 1 to 3, SHALL discard a fan when the price passes its anchor, and SHALL start a new fan after the third break.

#### Scenario: Three breaks
- **WHEN** three rising troughs and closes under each fan line in turn
- **THEN** breaks 1, 2 and 3 are returned, all BreakDown

### Requirement: Unusable input is refused
`quant.DetectPatterns` and `quant.FanBreaks` SHALL return an error for pivots that are out of order or do not fit the prices, for a negative option and for a price that is not positive.

#### Scenario: A pivot confirmed before it happened
- **WHEN** ConfirmedAt is not after Index
- **THEN** an error names the pivot's list position

### Requirement: Breaks that differ from the pivots are refused
`quant.DetectPatterns` and `quant.FanBreaks` SHALL return an error when a pivot's Segment does not match the segment its row lies in under the Breaks they are given.

#### Scenario: Pivots made without breaks
- **WHEN** the pivots all have Segment 0 and Breaks is `[15]` with a pivot after row 15
- **THEN** an error names the pivot's list position
