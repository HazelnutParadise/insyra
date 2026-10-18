## ADDED Requirements

### Requirement: Oblimin rotates from the start it is given
For each rotation start, the system SHALL rotate oblimin from that start, and with `Restarts` of 1 SHALL rotate from the identity as before.

#### Scenario: Same criterion as quartimin
- **WHEN** oblimin with `Delta` 0 and quartimin rotate the same loadings with the same `Restarts`
- **THEN** their criterion values agree to 1e-12

#### Scenario: A random start escapes the identity basin
- **WHEN** oblimin rotates the over-factored fixture with `Restarts` of 5
- **THEN** the criterion value is below a tenth of the single-start value
