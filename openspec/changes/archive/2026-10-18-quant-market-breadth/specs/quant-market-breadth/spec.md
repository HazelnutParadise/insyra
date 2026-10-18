## ADDED Requirements

### Requirement: Breadth indicators follow their definitions
`quant.AdvanceDeclineRatio`, `OverboughtOversold` and `NetAdvanceRatio` SHALL return, over the last window valid days, the advancing total over the declining total, the advancing total minus the declining total, and that difference over the sum of the two. `quant.AdvanceDeclineLine` SHALL return the start value plus the running sum of advancing minus declining.

#### Scenario: A two-day window
- **WHEN** adv is 10, 20, 30, 0, 5, dec is 5, 20, 10, 0, 15 and the window is 2
- **THEN** OBOS is NaN, 5, 20, 20, -10

### Requirement: A zero denominator and a missing day are NaN
The indicators SHALL return NaN, not 0 or an infinity, where a denominator is 0, on a day whose count is nil or NaN, and before the window holds enough valid days, and the window SHALL count valid days only.

#### Scenario: A day without data
- **WHEN** adv is 10, nil, 20, 30, dec is 5, 7, NaN, 10 and the window for OBOS is 2
- **THEN** the values are NaN, NaN, NaN, 25

### Requirement: Counts that are not counts are refused
The indicators SHALL return an error for a nil list, a non-number, an infinity, a negative count, lists of different lengths, and a window below 1.

#### Scenario: A negative count
- **WHEN** a count is -1
- **THEN** an error names its row index
