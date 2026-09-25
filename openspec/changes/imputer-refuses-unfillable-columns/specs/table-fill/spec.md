## ADDED Requirements

### Requirement: A fitted imputer refuses a selected column it cannot fill

`SimpleImputer.Fit` with the mean or median strategy SHALL fail when any selected column is not a number column, naming the column and its data type, and SHALL learn no column in that call.

#### Scenario: A text column among the selected ones

- **WHEN** `NewSimpleImputer().Fit(train, Name("income"), Name("color"))` runs and `color` holds text
- **THEN** it returns an error naming `color` and its string values, and the imputer is not fitted
