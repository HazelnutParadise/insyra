# quant-rsi Specification

## Purpose
The relative strength index with Wilder or simple averaging and an explicit policy for missing prices, so an RSI study uses one tested definition across thousands of stocks.

## Requirements

### Requirement: RSI follows the chosen average
`quant.RSI` SHALL return 100 * avgGain / (avgGain + avgLoss) over n days with the first n values NaN, using Wilder's smoothing (weight 1/n, started from the mean of the first n changes) or the mean of the last n changes as the method says, and SHALL return 100 for gains only, 0 for losses only and NaN where both are 0.

#### Scenario: The StockCharts example
- **WHEN** Wilder's RSI(14) is taken on the 33 closes of the StockCharts worked example
- **THEN** from row index 14 the values round to 70.53, 66.32, 66.55, 69.41, 66.36, 57.97

### Requirement: Missing prices follow the chosen policy
`quant.RSI` SHALL return an error for a nil or NaN price by default, SHALL carry the last price over with the missing day itself NaN under the flat policy, and SHALL restart the series after a missing price under the break policy, and SHALL always refuse infinities, non-numbers, a nil list and n below 1.

#### Scenario: A gap under the break policy
- **WHEN** the prices are 1, 2, nil, 4, 3, 5 and n is 2
- **THEN** the values at row indices 2, 3 and 4 are NaN and the value at row index 5 is defined
