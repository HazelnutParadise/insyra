## ADDED Requirements

### Requirement: PCA and KNNRegress refuse non-finite values
The system SHALL return an error from `PCA` when a cell of the table is NaN or infinite, and from `KNNRegress` when a target is NaN or infinite, instead of returning a result built from it.

#### Scenario: PCA with a NaN cell
- **WHEN** `PCA` receives a table with a NaN in one cell
- **THEN** it returns an error and no result

#### Scenario: KNNRegress with an infinite target
- **WHEN** `KNNRegress` receives a target list holding +Inf
- **THEN** it returns an error and no prediction
