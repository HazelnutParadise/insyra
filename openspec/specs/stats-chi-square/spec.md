# stats-chi-square Specification

## Purpose
How the chi-square tests in `stats` read their input lists, starting with reading both lists of the independence test at one moment.

## Requirements

### Requirement: The independence test reads both lists at one moment

`ChiSquareIndependenceTest` SHALL read `rowData` and `colData` within one `insyra.AtomicDoAll`, so that a writer which resizes both lists together cannot land between the two reads.

#### Scenario: A writer resizes both lists together
- **WHEN** 另一個 goroutine 在 `insyra.AtomicDoAll` 內把兩個 list 一起在 10 與 20 個值之間切換，同時反覆呼叫 `ChiSquareIndependenceTest(rows, cols)`
- **THEN** 沒有任何一次回傳 `both DataLists must have the same length`
