## ADDED Requirements

### Requirement: Single-sample tests accept a negative mu in one-shot mode

`wilcoxon`, `ttest` and `ztest` SHALL disable flag parsing, so that in one-shot CLI mode a negative value for `mu`, such as `-1` in `ttest single x -1`, is read as the value and not as a flag.

#### Scenario: Negative mu for a t-test
- **WHEN** `ttest single x -1` is run in one-shot mode against a variable `x`
- **THEN** the command runs with `mu` equal to -1 and returns no error

#### Scenario: Negative mu for a Z-test with a sigma
- **WHEN** `ztest single x -1 2` is run in one-shot mode
- **THEN** the command runs with `mu` equal to -1 and `sigma` equal to 2 and returns no error
