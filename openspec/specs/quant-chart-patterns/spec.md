# quant-chart-patterns Specification

## Purpose
Chart patterns and fan-principle breaks found from causal turning points, each reported only from the row on which it could be known, for backtesting price-pattern theory.

## Requirements

### Requirement: Patterns are known only from the row that confirms them
`ta.DetectPatterns` SHALL report each pattern from the row that confirms its last turning point, SHALL measure its breakout as the first close beyond its line before its expiry, the confirming row's own close included, and a series cut short SHALL give the patterns of the full series formed before the cut with any later breakout forgotten.

#### Scenario: A double top
- **WHEN** peaks at 100 and 99 either side of a trough at 85 and a close of 84 on row 36
- **THEN** a PatternDoubleTop breaks down on row 36

### Requirement: A pattern the price cancels never breaks
`ta.DetectPatterns` SHALL report no breakout for a pattern whose price closes beyond the head or the double's peaks, or passes its expiry, before breaking the textbook way.

#### Scenario: A new high
- **WHEN** the price closes at 101 above the 100 peaks before closing at 84
- **THEN** BreakoutAt is -1 and Direction is BreakNone

### Requirement: Tolerances are options
`ta.DetectPatterns` SHALL take how flat a line counts as flat, how alike peaks must be, the longest wait for a breakout, the least separation of a double and the trend look-back from `PatternOptions`, and SHALL NOT join swings from different segments.

#### Scenario: A wider flat tolerance
- **WHEN** lows rise by 25% of the height and FlatTolerance is 0.3
- **THEN** the pattern is a PatternRectangle and not an ascending triangle

### Requirement: Fan lines are numbered and spent after the third break
`ta.FanBreaks` SHALL return each close through the line joining a fan's anchor to the next flatter turning point, numbered 1 to 3, SHALL discard a fan when the price passes its anchor, and SHALL start a new fan after the third break.

#### Scenario: Three breaks
- **WHEN** three rising troughs and closes under each fan line in turn
- **THEN** breaks 1, 2 and 3 are returned, all BreakDown

### Requirement: Unusable input is refused
`ta.DetectPatterns` and `ta.FanBreaks` SHALL return an error for swings that are out of order or do not fit the prices, for a negative option and for a price that is not positive.

#### Scenario: A swing confirmed before it happened
- **WHEN** ConfirmedAt is not after Index
- **THEN** an error names the swing's list position

### Requirement: SegmentStarts that differ from the swings are refused
`ta.DetectPatterns` and `ta.FanBreaks` SHALL return an error when a swing's Segment does not match the segment its row lies in under the SegmentStarts they are given.

#### Scenario: Swings made without breaks
- **WHEN** the swings all have Segment 0 and SegmentStarts is `[15]` with a swing after row 15
- **THEN** an error names the swing's list position
