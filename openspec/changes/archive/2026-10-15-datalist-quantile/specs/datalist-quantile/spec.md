## ADDED Requirements

### Requirement: Quantile, Percentile and Quartile are one quantile on three scales

`DataList.Quantile(p)` SHALL return the R type-7 quantile of the list's numeric values for `p` in `[0, 1]`. `Percentile(p)` SHALL equal `Quantile(p/100)` for `p` in `[0, 100]`, and `Quartile(q)` SHALL equal `Quantile(q/4)` for `q` in 1, 2, 3. `IQR()` SHALL equal `Quartile(3) - Quartile(1)`, and `Median()` SHALL equal `Quantile(0.5)`.

#### Scenario: Matches pandas
- **WHEN** `Quantile` is called with 0, 0.25, 0.5, 0.75 and 1 on `[1, 2, 3, 4, 10]`
- **THEN** it returns 1, 2, 3, 4 and 10, which is what pandas' `Series.quantile` returns for the same values

#### Scenario: The three scales agree
- **WHEN** `Quantile(0.75)`, `Percentile(75)` and `Quartile(3)` are called on the same list
- **THEN** they return the same number

### Requirement: Missing cells are skipped and other non-numeric cells fail

`Quantile`, `Percentile`, `Quartile`, `IQR` and `Median` SHALL skip `nil` and `NaN` cells. `Quantile`, `Percentile`, `Quartile` and `IQR` SHALL record an error naming the call and the 1-based row for any other cell that is not a number, and return `NaN`. A list with no numeric values SHALL return `NaN` without recording an error.

#### Scenario: NaN is skipped
- **WHEN** `Quartile(1)`, `Quartile(3)` and `Median()` are called on `[1, 2, 3, NaN]`
- **THEN** they return 1.5, 2.5 and 2, and `Err()` is nil

#### Scenario: A text cell
- **WHEN** `Percentile(50)` is called on `[1, 2, "x", 3]`
- **THEN** it returns `NaN`, and `Err()` names `DataList.Percentile` and row 3

### Requirement: An argument out of range is refused without a panic

`Quantile` SHALL refuse a `p` that is `NaN` or outside `[0, 1]`, `Percentile` SHALL refuse one that is `NaN` or outside `[0, 100]`, and `Quartile` SHALL refuse a `q` other than 1, 2 or 3. Each SHALL record an error naming the call and return `NaN`, and none SHALL panic.

#### Scenario: Percentile of NaN
- **WHEN** `Percentile(math.NaN())` is called
- **THEN** it returns `NaN` with an error naming `Percentile`, where it used to panic

### Requirement: Displaying or plotting a list does not record a quantile error

`DataList.Summary` and the `plot` box plot SHALL compute their statistics from the list's numeric cells only. A list holding text cells SHALL therefore have no error recorded on it by either.

#### Scenario: Box plot of a mixed list
- **WHEN** a box plot is drawn from `[1, "abc", 3, nil, 9, "5"]`
- **THEN** it shows the five-number summary of 1, 3 and 9
