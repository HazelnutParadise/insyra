## ADDED Requirements

### Requirement: A fitter's failure on a column says why

The scalers, the simple imputer and the one-hot, label and ordinal encoders SHALL report a column they cannot resolve with the same explanation every other selector gives, including the `Name(...)` form when the table has a column of that name.

#### Scenario: A scaler given a name as a bare string

- **WHEN** `NewStandardScaler().FitTransform(dt, "Age")` runs on a table with a column named `Age`
- **THEN** it fails, and the error says to write `Name("Age")`

#### Scenario: An encoder given a name as a bare string

- **WHEN** `LabelEncode` is given `Column: "segment"` on a table with a column named `segment`
- **THEN** it fails, and the error says to write `Name("segment")`
