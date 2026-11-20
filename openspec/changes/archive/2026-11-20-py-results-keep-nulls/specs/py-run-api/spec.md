## ADDED Requirements

### Requirement: A polars DataFrame keeps its nulls and nested values

`insyra.Return` SHALL send each column of a polars DataFrame as a polars Series is sent: a null SHALL come back as `nil`, an integer column with nulls SHALL keep its integers, a list cell SHALL come back as a list and a struct cell as a map. A NaN that polars holds as a value SHALL come back as NaN.

#### Scenario: An id column with a null
- **WHEN** Python returns `pl.DataFrame({"id": [7, None], "x": [1.5, float("nan")]})` as a `*insyra.DataTable`
- **THEN** the `id` cells are the float64 7 and `nil`, and the `x` cells are 1.5 and NaN

#### Scenario: List and struct columns
- **WHEN** Python returns a polars DataFrame with a list column `[[1, 2], None]` and a struct column `[{"x": 1}, None]`
- **THEN** the call succeeds, the list cell is a slice of 1 and 2, the struct cell is a map with `x` 1, and both missing cells are `nil`

### Requirement: A pandas missing value of a nullable dtype is nil

`insyra.Return` SHALL send a pandas column or Series of an extension dtype as its own values, with `pd.NA` sent as `None`, so it comes back as `nil` and an `Int64` column keeps its integers. A column of a NumPy dtype SHALL be sent as before, its missing values coming back as NaN.

#### Scenario: Nullable integer, boolean and string columns
- **WHEN** Python returns a pandas DataFrame with an `Int64` column `[7, pd.NA]`, a `boolean` column `[True, pd.NA]` and a `string` column `["a", pd.NA]`
- **THEN** the call succeeds, the cells are the float64 7, `true` and `"a"`, and the second row is all `nil`

#### Scenario: A nullable Series
- **WHEN** Python returns `pd.Series([7, pd.NA], dtype="Int64")` as a `*insyra.DataList`
- **THEN** the list holds the float64 7 and `nil`
