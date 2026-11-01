## ADDED Requirements

### Requirement: CCL rounds in a chosen direction

`ROUNDUP(x, n)` SHALL round away from zero and `ROUNDDOWN(x, n)` toward zero, each to `n` decimal places with `n` required and possibly negative, reading `x` as it is written. `INT(x)` SHALL round down to an integer, as `FLOOR` does.

#### Scenario: Away from and toward zero

- **WHEN** the number is -3.14159 and `n` is 1
- **THEN** `ROUNDUP` gives -3.2 and `ROUNDDOWN` gives -3.1

#### Scenario: Written value

- **WHEN** an expression is `ROUNDUP(1.1, 1)`
- **THEN** the result is 1.1

#### Scenario: INT of a negative number

- **WHEN** an expression is `INT(-8.9)`
- **THEN** the result is -9
