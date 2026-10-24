## ADDED Requirements

### Requirement: Gaps are classed from the gap day alone
`quant.ClassifyGaps` SHALL find a gap up where the low is above the previous high and a gap down where the high is below the previous low, and SHALL class it from the range, volume and price move of the rows up to and including the gap day, so that cutting the series after the gap day does not change its class.

#### Scenario: A gap out of a quiet range
- **WHEN** a gap up closes above the 20-day high on three times the usual volume after a flat month
- **THEN** the gap is GapBreakaway with HighVolume true

### Requirement: The earlier move separates runaway from exhaustion
`quant.ClassifyGaps` SHALL class a gap to a new extreme as a runaway after an earlier move of TrendZ to ExhaustionZ standard deviations in its direction, and as an exhaustion gap after more than ExhaustionZ on volume of HighVolumeRatio times the recent mean or more.

#### Scenario: A long trend on light volume
- **WHEN** the earlier move is above ExhaustionZ and the volume is average
- **THEN** the gap is GapUnclassified

### Requirement: Fills look only forward and stop at a break
`quant.ClassifyGaps` SHALL report, for each window in GapOptions.FillWindows (5 and 20 rows by default), whether a gap was filled within it, that is when a later low (up) or high (down) reaches the previous high (low), and SHALL report GapFillUnknown when the window runs past the data or the end of the segment.

#### Scenario: A window past the data
- **WHEN** a gap on the 95th of 100 rows
- **THEN** the 20-row entry of Fills is GapFillUnknown

### Requirement: Unusable input is refused
`quant.ClassifyGaps` SHALL return an error for series of different lengths, a high below its low, negative volume, a price that is not positive, a negative option and an ExhaustionZ below TrendZ.

#### Scenario: A high below the low
- **WHEN** a high is 98 and the low is 99
- **THEN** an error names the row index
