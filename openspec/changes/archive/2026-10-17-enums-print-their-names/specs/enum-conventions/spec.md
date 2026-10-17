## ADDED Requirements

### Requirement: An enum's underlying type follows one rule

An exported enum type SHALL take its underlying type from the first rule that applies:
1. If the number means something outside insyra, such as Excel's day-count `basis`, it SHALL be `int`, numbered as the outside source numbers it.
2. If the constant's text leaves Go unchanged, it SHALL be `string`, and the constant's value SHALL be that text. Text leaves Go unchanged when it is a name defined by an outside standard or tool, or when it is written to a configuration file, JSON, a log or a result report.
3. Otherwise it SHALL be `int`, numbered with `iota`, with its zero value as the documented default.

#### Scenario: External numbering
- **WHEN** `finance.Basis30_360EU` is converted to `int`
- **THEN** the result is 4, Excel's basis number for that convention

### Requirement: Every numeric enum prints its name

Every exported enum type whose underlying type is an integer SHALL have a `String()` method. A known value SHALL print a name. An unknown value SHALL print `TypeName(n)`.

#### Scenario: Known value
- **WHEN** `quant.VaRParametric` is formatted with `%v`
- **THEN** the text is `VaRParametric`

#### Scenario: Unknown value
- **WHEN** `ml.ClassAverage(9)` is formatted with `%v`
- **THEN** the text is `ClassAverage(9)`, not `macro`

### Requirement: Errors name an enum value

An error that reports an enum value SHALL format it with `%v`.

#### Scenario: Unknown method
- **WHEN** `quant.ValueAtRisk` is called with `VaRMethod(3)`
- **THEN** the error text contains `VaRMethod(3)`
