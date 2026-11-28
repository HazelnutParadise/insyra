# preprocess-package Specification

## Purpose
Gives 0.3.x the `preprocess` package with the API it has on 0.4: scalers, encoders and `SimpleImputer` that fit on one table and transform others, while the deprecated core scalers and encoders keep working and a scaler they saved still loads in the new package.

## Requirements

### Requirement: Fitted preprocessing lives in the preprocess package

The scalers, the categorical encoders and the fitted imputer SHALL be provided by the package `preprocess`, which SHALL read a column the 0.4 way: a bare string as an Excel-style index, `insyra.Name` as a name and an `int` as a position. On the 0.3.x line the root package's scaling and encoding methods and types SHALL keep their behaviour and SHALL be marked Deprecated, naming their `preprocess` replacement.

#### Scenario: Scaling a table
- **WHEN** `preprocess.StandardScale(dt, insyra.Name("price"))` runs on a table whose `price` column is 1, 2, 3
- **THEN** it returns the scaled table, a fitted `*preprocess.StandardScaler` and a nil error, and `price` is -1, 0, 1

#### Scenario: A bare string is an index
- **WHEN** `preprocess.StandardScale(dt, "price")` runs on a table with one column, named `price`
- **THEN** it returns an error that says to write `Name("price")`

#### Scenario: The old names are deprecated
- **WHEN** the doc comment of `insyra.NewStandardScaler` is read
- **THEN** it has a `Deprecated:` paragraph naming `preprocess.NewStandardScaler`

### Requirement: A scaler saved by the core still loads

A scaler's JSON whose column `"ref"` is a plain string, as the core scalers write it, SHALL decode into the matching `preprocess` scaler, SHALL resolve that reference as a column name first and as an Excel-style index second, and SHALL be written back as the same string. An empty reference SHALL be refused.

#### Scenario: Decoding core JSON
- **WHEN** JSON an `insyra.StandardScaler` fitted on columns `a` and `B` wrote is decoded into a `preprocess.StandardScaler`
- **THEN** it decodes without error and transforms every table exactly as the core scaler does

### Requirement: ml takes either kind

`ml` pipelines, `ColumnTransformer` and `ExportONNX` SHALL accept the `preprocess` scalers, encoders and imputer as well as the deprecated core ones, and SHALL export the same ONNX bytes for the same fitted parameters.

#### Scenario: The same export
- **WHEN** a pipeline of `preprocess.StandardScaler` and `preprocess.OneHotEncoder` and one of the core types are fitted on the same data and exported
- **THEN** the two ONNX outputs are byte for byte equal
