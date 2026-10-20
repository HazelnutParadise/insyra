# quant-forward-returns Specification

## Purpose
Measuring the excess return after each event or by signal bucket on a stock panel, entered after the signal is seen and tested by month, so a signal study does not mistake many same-day observations for independent evidence.

## Requirements

### Requirement: A signal is traded after it is seen
`quant.EventStudy` and `quant.SignalBuckets` SHALL enter each observation at the close Delay rows after its signal row, with Delay at least 1 and defaulting to 1, hold it for each horizon, and report its return less the benchmark over the same window.

#### Scenario: Delay 2
- **WHEN** an event on row 0 is entered with Delay 2 and the stock rises 10% over the next row while the market rises 5%
- **THEN** the excess return is 5%

### Requirement: Observations are averaged by month before the t-test
Each horizon's summary SHALL report the observation count, the number left out for a missing price, an unfinished window or an untradable entry, the mean excess return, the win rate, and the Newey-West t-test that the mean of the monthly averages is zero, which is NaN with fewer than two months.

#### Scenario: Two months
- **WHEN** three events produce excess returns of 5%, 5% and 0%, the first two in January
- **THEN** the monthly means are 5% and 0% and the observation count is 3

### Requirement: The Newey-West lag has one setting and the correction its own switch
`ForwardOptions.NeweyWestLag` SHALL choose `stats.NeweyWestLag(months)` when 0, be used as given when positive, and be an error when negative. `ForwardOptions.NoNeweyWest` SHALL switch the correction off by running the test with lag 0, whose variance divides by the number of months, and SHALL be an error together with a non-zero `NeweyWestLag`.

#### Scenario: A negative lag is refused
- **WHEN** `NeweyWestLag` is -1
- **THEN** an error is returned

#### Scenario: The correction is off
- **WHEN** `NoNeweyWest` is true
- **THEN** `Lag` is 0 and the t statistic equals `stats.NeweyWestMeanTest` of the monthly means with lag 0

#### Scenario: Switch and lag together
- **WHEN** `NoNeweyWest` is true and `NeweyWestLag` is 2
- **THEN** an error is returned

### Requirement: Events are de-duplicated and filtered
`quant.EventStudy` SHALL drop an event closer than MinGap rows to the last counted event of the same stock, and both functions SHALL leave out an observation that is not tradable at its entry row and count it as missing.

#### Scenario: MinGap
- **WHEN** a stock has events on rows 0, 1 and 2 and MinGap is 2
- **THEN** the events on rows 0 and 2 are counted

### Requirement: Signal buckets follow their edges
`quant.SignalBuckets` SHALL put an observation in the bucket above the last edge not exceeding its signal, so n edges make n+1 buckets, SHALL take edges from pooled quantiles when none are given, and SHALL measure a single series against its own mean return when asked to.

#### Scenario: Edge at zero
- **WHEN** the only edge is 0
- **THEN** signals below 0 are in bucket 0 and signals of 0 or more in bucket 1

### Requirement: Bias indicators do not look ahead
`quant.Bias` SHALL return price over its n-day moving average less 1, and `quant.BiasZ` SHALL divide it by the standard deviation of the previous window biases, so that changing a later price never changes an earlier value.

#### Scenario: A changed last price
- **WHEN** only the last price is changed
- **THEN** every BiasZ value before the last row is unchanged

### Requirement: Unusable input is refused
The functions SHALL return an error for no horizon, a horizon below 1, a Delay below 1, a negative MinGap, a nil table or list, a table of the wrong shape, a date that is not a time or a YYYY-MM-DD string, a missing or mis-sized benchmark, unsorted edges and a quantile outside (0, 1).

#### Scenario: Delay 0 is not a way to trade at the signal's close
- **WHEN** Delay is negative
- **THEN** an error is returned
