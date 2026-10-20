# ta-money-flow-and-volume-profile Specification

## Purpose
Volume-based indicators (money flow index, accumulation/distribution, Chaikin, ease of movement, volume indexes, volume profile) with TA-Lib's or the standard definitions.

## Requirements

### Requirement: Money flow indicators follow their definitions
`ta.MFI`, `ta.AccumulationDistribution` and `ta.ChaikinOscillator` SHALL equal TA-Lib's MFI, AD and ADOSC, and `ta.CMF` SHALL return the sum of the money flow volume over the sum of the volume over n rows.

#### Scenario: A close at the high
- **WHEN** the close equals the high and high differs from low
- **THEN** the money flow volume of the row is the whole volume

### Requirement: Volume indexes and ease of movement are available
`ta.NVI` and `ta.PVI` SHALL start at 1000 and move by the price's percentage change on rows whose volume is lower, or higher, than the previous row's, and `ta.EaseOfMovement` SHALL return the n-row average of the midpoint move divided by the box ratio, with the volume scale an option.

#### Scenario: A quiet day
- **WHEN** volume falls from 100 to 50 and the price rises 10%
- **THEN** NVI rises 10% and PVI is unchanged

### Requirement: The volume profile is computed per window
`ta.VolumeProfile` SHALL spread each bar's volume over the equal price bins it covers in proportion to the overlap, return the middle of the bin with the most volume as POC, the lowest bin on a tie, and the value area grown toward the side with more volume until it holds the given share of the volume, and SHALL return NaN for a window with no range or no volume.

#### Scenario: Volume at one level
- **WHEN** all the volume traded in the lower of two bins
- **THEN** POC is the middle of that bin

### Requirement: Unusable input is refused
Each function here SHALL return an error naming the row index for a missing, non-numeric, non-finite or negative volume value and for a high below its low, and an error for a period below 1, slow not above fast, a negative bin count or scale, a value area outside 0 to 1 and series of different lengths.

#### Scenario: Negative volume
- **WHEN** a volume is -1
- **THEN** an error names its row index
