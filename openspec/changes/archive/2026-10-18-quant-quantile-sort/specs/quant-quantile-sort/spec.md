## ADDED Requirements

### Requirement: Every date is sorted into groups on its own
`quant.QuantileSort` SHALL sort the stocks with a signal on each date into n groups by that day's signal alone, group 1 holding the lowest signals, and SHALL return for each date and group the stock count and the mean of each return column, and for each date the highest group's mean minus the lowest group's.

#### Scenario: Six stocks into tertiles
- **WHEN** six stocks have signals 1 to 6 and returns ten times the signal, and n is 3
- **THEN** the group means are 15, 35 and 55 and the spread is 40

### Requirement: Ties, weights and thin days follow the options
`quant.QuantileSort` SHALL keep equal signals in one group under the average-rank method, SHALL cut at linearly interpolated quantiles and skip a day whose edges coincide under the qcut method, SHALL weight group means by a weight column when one is named, SHALL subtract the day's mean return when asked to demean, and SHALL skip and report a day with fewer stocks than MinStocks.

#### Scenario: A signal that qcut cannot cut
- **WHEN** a day's quantile edges coincide and the qcut method is chosen
- **THEN** the day is listed in Skipped and left out of both tables

### Requirement: Input that cannot be sorted is refused
`quant.QuantileSort` SHALL return an error for n below 2, no return column, a nil table, a column that is not found, a text or infinite value in the signal, return or weight columns, a missing date, a non-positive weight and an unknown tie method.

#### Scenario: A column that does not exist
- **WHEN** the date column is named "day" and the table has no such column
- **THEN** an error names the column
