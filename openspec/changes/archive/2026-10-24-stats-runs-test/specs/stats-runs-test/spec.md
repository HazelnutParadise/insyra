## ADDED Requirements

### Requirement: RunsTest counts the runs above and below a threshold

`RunsTest(series, opts...)` SHALL mark each value above or below `Threshold` (default 0), drop values equal to it and report their number in `Dropped`, and count a run as an unbroken stretch of one mark.

#### Scenario: Agreement with R

- **WHEN** `RunsTest` is called on a series for which R's `rle` counts the runs after dropping the values equal to 0
- **THEN** `Runs`, `Above`, `Below` and `Dropped` equal R's counts

### Requirement: RunsTest tests the number of runs against a random order

`RunsTest` SHALL report `Expected` as 2·n1·n2/n + 1, `Variance` as 2·n1·n2·(2·n1·n2 − n)/(n²·(n − 1)), `Z` as (Runs − Expected)/√Variance and `PValue` as its two-tailed normal p-value. With `Correction` set, the difference SHALL move 0.5 toward zero before it is divided.

#### Scenario: Statistics and correction

- **WHEN** `RunsTest` is called with and without `Correction` on the same series
- **THEN** both match R's formula to within 1e-12, and the corrected `Z` is closer to zero

### Requirement: RunsTest refuses input it cannot test

`RunsTest` SHALL return an error for a nil or empty list, a value that is not a finite number, a non-finite `Threshold`, more than one options value, and a series with no value above or none below the threshold.

#### Scenario: One side only

- **WHEN** every value lies above the threshold
- **THEN** the call returns an error naming the counts above and below
