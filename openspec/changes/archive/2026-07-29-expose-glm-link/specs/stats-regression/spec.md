## ADDED Requirements

### Requirement: A fitted generalised linear model publishes its link
The system SHALL let a caller read which link function a fitted model was estimated under.

#### Scenario: A logistic or Poisson model is inspected
- **WHEN** a caller holds a fitted logistic or Poisson regression result
- **THEN** the link function it was fitted with is readable from the result
- **AND** it is named and typed the same way the general GLM result names it

#### Scenario: The link is used to reproduce a prediction outside the package
- **WHEN** code outside `stats` applies the published link to the linear predictor
- **THEN** it obtains the same response the result's own prediction returns
