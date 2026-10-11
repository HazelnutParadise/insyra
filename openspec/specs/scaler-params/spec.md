# scaler-params Specification

## Purpose
What the fitted parameters a scaler or imputer reports per column mean, and which of them are deprecated on this line.

## Requirements

### Requirement: ScalerParams.PassThrough is deprecated and keeps its meaning

`ScalerParams.PassThrough` SHALL be marked deprecated, to be removed in 0.4, and SHALL keep reporting that a fitted imputer leaves a column unchanged because its numeric strategy met non-numeric values.

#### Scenario: A mean imputer over a text column

- **WHEN** a `SimpleImputer` with the mean strategy is fitted on a table whose selected column holds text
- **THEN** that column's `ScalerParams.PassThrough` is true and the column is left unchanged when the imputer transforms, as on v0.3.4
