## ADDED Requirements

### Requirement: Whole bars are shuffled and the path is rebuilt
`quant.ShuffleBars` SHALL keep the first bar of each segment, put the other bars in a random order, rebuild each bar from the previous rebuilt close with its own gap, high, low and close ratios and its own volume when ShuffleOptions.Volume is given, and so end each segment on the real last close.

#### Scenario: A rebuilt bar
- **WHEN** a bar with a 2% gap and a high 1% over its open
- **THEN** its rebuilt open is 2% over the rebuilt previous close and its high 1% over that open

### Requirement: The seed decides the path
`quant.ShuffleBars` SHALL give the same table for the same bars and seed and a different one for a different seed, the zero seed being an ordinary seed.

#### Scenario: The same seed twice
- **WHEN** two calls with seed 5
- **THEN** the tables are equal

### Requirement: Unusable input is refused
`quant.ShuffleBars` SHALL return an error for a price that is not positive, series of different lengths, a high below its low, negative volume and a break outside the series, and a volume of a different length, and SHALL leave out the Volume column when ShuffleOptions.Volume is not given.

#### Scenario: A zero open
- **WHEN** an open is 0
- **THEN** an error names the row index
