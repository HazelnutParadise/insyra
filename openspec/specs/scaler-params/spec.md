# scaler-params Specification

## Purpose
What the fitted parameters a scaler or imputer reports per column mean, and which of them are deprecated on this line.

## Requirements

### Requirement: ScalerParams.PassThrough is deprecated and keeps its meaning

`ScalerParams.PassThrough` SHALL be marked deprecated, to be removed in 0.4, and SHALL keep reporting that a fitted imputer leaves a column unchanged because its numeric strategy met non-numeric values.

#### Scenario: A mean imputer over a text column

- **WHEN** a `SimpleImputer` with the mean strategy is fitted on a table whose selected column holds text
- **THEN** that column's `ScalerParams.PassThrough` is true and the column is left unchanged when the imputer transforms, as on v0.3.4

### Requirement: A scaler read from JSON refuses an empty column reference
The system SHALL refuse scaler JSON in which a fitted column has an empty reference, return an error naming the empty reference, and leave the receiver unchanged.

#### Scenario: Empty reference
- **WHEN** scaler JSON holds a column whose ref is an empty string
- **THEN** reading it returns an error and nothing is fitted
