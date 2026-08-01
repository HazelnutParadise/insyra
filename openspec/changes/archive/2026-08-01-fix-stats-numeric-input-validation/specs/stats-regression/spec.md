## ADDED Requirements

### Requirement: Numeric input that cannot be read is refused, never replaced
The system SHALL refuse any observation it cannot convert to a finite number, and SHALL NOT substitute a value of its own choosing.

This applies to every numeric input a statistical routine reads: predictors, targets, offsets, and the paired series of a correlation.

#### Scenario: A predictor column contains a value that is not a number

- **WHEN** a regression is fitted on data where a predictor holds a missing value, a blank, or text
- **THEN** the request is refused with an error
- **AND** the error names the column and the row where the value was found
- **AND** no coefficient is returned

#### Scenario: A target column contains a value that is not a number

- **WHEN** a regression is fitted on data where the target holds a missing value, a blank, or text
- **THEN** the request is refused with an error naming the row
- **AND** the value is not treated as zero

#### Scenario: A correlation is computed over data containing a blank

- **WHEN** a correlation coefficient is requested for two series and either contains a value that is not a finite number
- **THEN** the request is refused with an error
- **AND** no coefficient is returned

#### Scenario: An infinite or undefined value reaches a numeric input

- **WHEN** an input holds an infinity or an undefined numeric value
- **THEN** it is refused on the same terms as a value that is not a number

### Requirement: Each family states how it treats values it cannot read
The system SHALL document, for every statistical family it exposes, which of its three treatments applies to unreadable values: refusal, removal of the whole observation, or a learned direction.

#### Scenario: A caller asks what happens to their missing values

- **WHEN** a caller consults the documentation for a statistical family
- **THEN** that family's treatment of unreadable values is stated there
- **AND** where a family removes observations rather than refusing them, the documentation says so
