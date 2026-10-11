## ADDED Requirements

### Requirement: A command that stores several variables names each one

A command that stores variables besides its `as <var>` result SHALL name every variable it stores in its success line.

#### Scenario: kmeans names all nine variables
- **WHEN** the user runs `kmeans t 2 as km`
- **THEN** the output lists `km` and the eight `km_*` variables, and all nine are stored

#### Scenario: quant factor names its alpha table
- **WHEN** the user runs `quant factor a f as fm`
- **THEN** the output lists `fm` and `fm_alpha`

#### Scenario: quant portfolio names its stats table
- **WHEN** the user runs `quant portfolio rets maxsharpe rf 0.0001 as w`
- **THEN** the output lists `w` and `w_stats`

#### Scenario: A stored variable replaces one of the same name
- **WHEN** a variable `km_size` already exists and the user runs `kmeans t 2 as km`
- **THEN** the output names `km_size`, and `km_size` holds the new result
