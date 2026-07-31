## ADDED Requirements

### Requirement: One protocol over every model
The system SHALL expose every fitted model through one interface, whatever algorithm produced it.

#### Scenario: Any model scores new observations
- **WHEN** a caller holds a fitted model obtained through the package
- **THEN** it can score new observations through the same method regardless of which algorithm fitted it

#### Scenario: A model is asked which columns it was fit on
- **WHEN** a caller holds a fitted model
- **THEN** the columns it was fitted on are readable from it, in the order it expects them

#### Scenario: New observations carry different columns than the fit
- **WHEN** observations are scored whose columns do not match what the model was fitted on
- **THEN** the request is refused with an error naming the mismatch
- **AND** the columns are matched by name rather than by position

### Requirement: Existing preprocessing satisfies the protocol unchanged
The system SHALL accept the scalers and encoders the root package already provides as protocol members without adaptation.

#### Scenario: A scaler or encoder is used as a transformer
- **WHEN** a caller uses one of the root package's fitted scalers or encoders where a transformer is expected
- **THEN** it is accepted with no wrapping
- **AND** its behaviour is unchanged from calling it directly

### Requirement: Wrapped models return what the wrapped function returns
The system SHALL produce, for any model it wraps, the same numbers the underlying `stats` function produces.

#### Scenario: A wrapped model is compared against the function it wraps
- **WHEN** a model fitted through the protocol is compared against the same fit performed directly
- **THEN** the coefficients, predictions and reported statistics are identical, not merely close

#### Scenario: The underlying fit fails
- **WHEN** the wrapped function returns an error
- **THEN** the error is returned unchanged rather than being replaced or swallowed

### Requirement: Capabilities beyond prediction are discoverable
The system SHALL let a caller discover whether a fitted model supports a capability rather than assuming it.

#### Scenario: A model that reports class probabilities
- **WHEN** a caller needs class probabilities
- **THEN** it can determine whether a given model provides them before asking
- **AND** the column order of the probabilities matches the order in which the model reports its classes

#### Scenario: A model that does not support a capability
- **WHEN** a caller checks for a capability a model does not have
- **THEN** the check reports its absence rather than failing at call time

### Requirement: A third party can check its own model
The system SHALL provide a way to verify that a model outside the package obeys the protocol.

#### Scenario: An external model is checked
- **WHEN** a model implemented outside the package is put through the conformance check
- **THEN** every rule the protocol states is exercised against it
- **AND** a violation is reported naming the rule that was broken
