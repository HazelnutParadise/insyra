# quant-abnormal-returns Specification

## Purpose
Showing how a stock moved against its market-implied normal on each day around an event, before it as well as after, so a leak ahead of the news shows up in the cumulative abnormal return.

## Requirements

### Requirement: Abnormal returns are reported for each day around an event
`quant.AbnormalReturns` SHALL report, for each relative day of `EventWindow`, the mean abnormal return and the mean cumulative abnormal return (cumulated from the first day) across the events, where an abnormal return is the stock's simple return less its expected return.

#### Scenario: A planted effect
- **WHEN** every event adds 5% to the stock's return on day 0
- **THEN** the mean abnormal return on day 0 is about 5% and its p value is small

#### Scenario: A leak before the event
- **WHEN** every event adds 1% on each of the five days before day 0
- **THEN** the mean CAR rises over those days

### Requirement: The expected return comes from the market
With `AbnormalMarketAdjusted` the expected return SHALL be the market's simple return that day. With `AbnormalMarketModel` it SHALL be alpha + beta times that return, from ordinary least squares on the usable rows of `EstimationWindow`, which must end before `EventWindow` starts.

#### Scenario: A stock that tracks the market
- **WHEN** a stock returns exactly the market's return except for a 2% shock on day 0
- **THEN** the abnormal return is 0 on every day but day 0, where it is 2%

### Requirement: The estimation length has no default
`AbnormalReturnsOptions.MinEstimation` SHALL be required under `AbnormalMarketModel` with a value of at least 3, and SHALL be 0 under `AbnormalMarketAdjusted`; `EstimationWindow` SHALL be the zero value under `AbnormalMarketAdjusted`.

#### Scenario: MinEstimation left out
- **WHEN** the market model is chosen and `MinEstimation` is 0
- **THEN** an error is returned

### Requirement: An event is dropped as a whole
An event whose stock price or market level is missing on any row its event window needs, including the row before the first one, or whose window runs outside the table, SHALL be counted in `Missing` and left out of every day.

#### Scenario: One missing price
- **WHEN** one event has a missing price on day 4 of its window
- **THEN** it is left out of every day, and the means equal those with that event not flagged

### Requirement: Too few estimation rows drop the event
Under `AbnormalMarketModel`, an event with fewer than `MinEstimation` usable estimation rows, or whose usable rows show no market movement, SHALL be dropped as a whole and counted in `Missing`.

#### Scenario: Not enough rows
- **WHEN** an event has 34 usable estimation rows and `MinEstimation` is 35
- **THEN** it is counted in `Missing`

### Requirement: Other events' windows are kept out of the estimation
A row of a stock's estimation window that lies inside the event window of another event of the same stock SHALL NOT be used to fit the market model.

#### Scenario: Two events 30 rows apart
- **WHEN** the second event's estimation window holds the first event's window
- **THEN** those rows are not in the second event's fit

### Requirement: Events are tested across the sample
For each relative day, `ARTStat` and `ARPValue` SHALL be the t statistic mean/(sd/sqrt(N)) and its two-tailed p value with N-1 degrees of freedom over the events, and `CARTStat` and `CARPValue` the same for the CAR. They SHALL be NaN with fewer than 2 events or no spread.

#### Scenario: One event
- **WHEN** only one event is used
- **THEN** every t statistic and p value is NaN

### Requirement: Events of one stock closer than MinGap are dropped
With `MinGap` above 0, an event of a stock closer than `MinGap` rows to its last counted event SHALL be left out and not counted in `Missing`.

#### Scenario: Three events
- **WHEN** events lie on rows 50, 55 and 100 and `MinGap` is 10
- **THEN** two events are used

### Requirement: Bad input is an error
`AbnormalReturns` SHALL return an error for a nil table or list, a column count or row count that differs from the prices, a non-positive market level, a window that starts after it ends, an unsupported model, a negative `MinGap` and a `MinEstimation` larger than the estimation window.

#### Scenario: Unsupported model
- **WHEN** `Model` is `AbnormalModel(7)`
- **THEN** the error names `AbnormalModel(7)`

### Requirement: The results table has one row per event
`AbnormalReturnsResult.CARs` SHALL have one row per used event with the stock's column name, the event date as given in `dates`, and the CAR on each relative day.

#### Scenario: Two events used
- **WHEN** two events are used over a window of 16 days
- **THEN** the table has 2 rows and 18 columns

### Requirement: A return that is not finite is missing data
A simple return of a stock or of the market that is not a finite number, such as the ratio of two finite levels overflowing, SHALL be treated as a missing value, so an event whose window needs it is dropped as a whole and counted in `Missing`.

#### Scenario: A price jump that overflows
- **WHEN** a price goes from 1e-300 to 1e300 inside an event's window
- **THEN** the event is counted in `Missing` and no mean abnormal return is infinite

### Requirement: Each event date is one cell
The event-date column of `CARs` SHALL hold each date as given in `dates` in a single cell, including a date that is a slice, so the table has one row per used event.

#### Scenario: A slice as a date
- **WHEN** an event's date cell is the slice `[2026 1 2]`
- **THEN** the table has one row for that event and its date cell is that slice

### Requirement: Windows are bounded before their size is used
`AbnormalReturns` SHALL return an error, without panicking, when an end of `EventWindow` or `EstimationWindow` is more than 2^29 rows from the event, when `dates` has more than 2^29 rows, or when `EventWindow` spans more rows than `dates` has. No allocation or arithmetic sized by a window or offset SHALL happen before these checks; copying the input tables is proportional to their own size.

#### Scenario: An extreme window
- **WHEN** `EventWindow` is `[minInt, 0]`
- **THEN** an error is returned

### Requirement: A non-finite abnormal return or CAR is missing data
An event whose abnormal return, running CAR or market-model fit is not a finite number SHALL be dropped as a whole and counted in `Missing`, and a cross-sectional mean or spread whose sum overflows SHALL give NaN statistics.

#### Scenario: A CAR that overflows
- **WHEN** an event has abnormal returns of 1e308, -1 and 1e308
- **THEN** the event is counted in `Missing` and no mean CAR is infinite
