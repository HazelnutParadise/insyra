# core-preprocessing Specification

## Purpose
TBD - created by archiving change add-fitted-imputer. Update Purpose after archive.
## Requirements
### Requirement: Missing values can be filled with a remembered value
The system SHALL let a caller derive replacement values from one table and apply those same values to another.

#### Scenario: An imputer is fitted and then applied to new data
- **WHEN** a caller fits an imputer on one table and applies it to another
- **THEN** the missing values in the second table are filled with the values derived from the first
- **AND** nothing in the second table influences what it is filled with

#### Scenario: The same table is fitted and transformed
- **WHEN** a caller fits and transforms in one call
- **THEN** the result equals filling that table with statistics computed from itself

#### Scenario: The fitted values are inspected
- **WHEN** a caller asks a fitted imputer what it will substitute
- **THEN** the value it holds for each column is readable

#### Scenario: A column has no observed values to fit from
- **WHEN** every value in a column is missing at fitting time
- **THEN** the request is refused with an error naming the column

#### Scenario: A column absent at fitting time appears at transform time
- **WHEN** a table carries a column the imputer was not fitted on
- **THEN** that column passes through unchanged

### Requirement: The strategies already available in place are available fitted
The system SHALL offer, in fitted form, the replacement strategies it offers as in-place operations.

#### Scenario: A strategy is chosen
- **WHEN** a caller chooses to substitute the mean, the median, the mode, or a supplied constant
- **THEN** the fitted imputer derives and stores that quantity per column

#### Scenario: A numeric strategy meets a non-numeric column
- **WHEN** a mean or median strategy is fitted on a column with non-numeric observed values
- **THEN** the column is left alone rather than being given a numeric substitute
- **AND** this matches how the existing in-place methods behave

### Requirement: An imputer is a preprocessing step like any other
The system SHALL make a fitted imputer usable wherever a fitted scaler or encoder is usable.

#### Scenario: An imputer is used as a pipeline step
- **WHEN** a fitted imputer is used where a preprocessing step is expected
- **THEN** it is accepted with no wrapping

#### Scenario: A caller asks whether the imputation can be undone
- **WHEN** a caller tests whether a fitted imputer can reverse its transformation
- **THEN** the answer is that it cannot, and it is available before the call rather than as a failure during it

