## ADDED Requirements

### Requirement: Fitted preprocessing lives in the preprocess package

The scalers, the categorical encoders and the fitted imputer SHALL be provided by the package `preprocess`, and the root package `insyra` SHALL NOT declare them or the `DataTable` methods that called them. The in-place fills and `DataList.Normalize` and `Standardize` SHALL stay in the root package.

#### Scenario: Scaling a table
- **WHEN** `preprocess.StandardScale(dt, insyra.Name("price"))` runs on a table whose `price` column is 1, 2, 3
- **THEN** it returns the scaled table, a fitted `*preprocess.StandardScaler` and a nil error, and `price` is -1, 0, 1

#### Scenario: Encoding a column
- **WHEN** `preprocess.OneHotEncode(dt, preprocess.OneHotOptions{Columns: []any{insyra.Name("city")}})` runs
- **THEN** it returns the encoded table and a fitted `*preprocess.OneHotEncoder`

### Requirement: A scaler saved before the move still loads

A scaler's JSON SHALL carry its kind and fitted state and no package name, so JSON written by the root package's scalers SHALL decode into the matching `preprocess` scaler, and a CLI environment saved before the move SHALL restore its scaler variables.

#### Scenario: Decoding old JSON
- **WHEN** JSON a `StandardScaler` wrote before the move is decoded into a `preprocess.StandardScaler`
- **THEN** it decodes without error and transforms a table the way the original did
