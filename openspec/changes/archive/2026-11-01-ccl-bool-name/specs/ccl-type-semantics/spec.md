## ADDED Requirements

### Requirement: BOOL converts a value to a boolean

`BOOL(x)` SHALL read `x` as a boolean, give an empty result for `nil` and for a value that cannot be read as one, and require one argument. `TOBOOL` SHALL NOT be a second working name in a release that has removed it, and SHALL point to `BOOL` where it remains.

#### Scenario: Text that reads as a boolean

- **WHEN** an expression is `BOOL('yes')`
- **THEN** the result is true

#### Scenario: Text that does not

- **WHEN** an expression is `BOOL('maybe')`
- **THEN** the result is empty
