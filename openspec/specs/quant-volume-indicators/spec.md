# quant-volume-indicators Specification

## Purpose
Volume-based indicators for price research: on-balance volume, a scale-free signed-volume ratio and divergence flags, so a volume study does not need hand-written arithmetic.

## Requirements

### Requirement: On-balance volume accumulates signed volume
`quant.OBV` and `quant.OBVFromReturns` SHALL return a series whose first value is 0 and whose value at each later row is the previous value plus the volume, signed by the direction of the day's change in close or in the given return, with a flat day adding 0.

#### Scenario: A rising, flat, falling and rising close
- **WHEN** the close is 10, 11, 11, 10, 12 and the volume is 100, 200, 300, 400, 500
- **THEN** OBV is 0, 200, 200, -200, 300

### Requirement: The signed-volume ratio is scale-free
`quant.SignedVolumeRatio` SHALL return, at each row with a full window of n days, the sum of signed volume over the window divided by the sum of volume, and NaN where the window is incomplete or holds no volume.

#### Scenario: A two-day window
- **WHEN** returns are 1, -1, 1, 0, volume is 1, 2, 3, 4 and n is 2
- **THEN** the values are NaN, -1/3, 0.2, 3/7

### Requirement: Divergence flags an unconfirmed extreme once per episode
`quant.Divergence` SHALL return -1 where price is above its highest of the previous n days while the indicator is below its highest, +1 for the mirror image on lows, and 0 elsewhere, and SHALL NOT flag the same kind again within cooldown days of a flag.

#### Scenario: Cooldown
- **WHEN** price is 1, 2, 3, 4, 5, the indicator is 1, 2, 3, 2, 1, n is 2 and cooldown is 1
- **THEN** the flags are 0, 0, 0, -1, 0

### Requirement: Unusable series are refused
The functions SHALL return an error for a nil series, a NaN, an infinity, a non-number, a negative volume, series of different lengths, and a window or cooldown out of range.

#### Scenario: A negative volume
- **WHEN** a volume value is -2
- **THEN** an error names its row index
