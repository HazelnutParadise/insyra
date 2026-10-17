# in-place-or-copy Specification

## Purpose
Which DataList and DataTable methods change the value they are called on and which return a copy: Shuffled returns the shuffled copy.

## Requirements

### Requirement: Shuffled returns a shuffled copy

`DataList.Shuffled` and `DataTable.Shuffled` SHALL return a new value holding the receiver's elements or rows in random order, named after the receiver with `_Shuffled` appended, and SHALL leave the receiver unchanged.

#### Scenario: The receiver keeps its order
- **WHEN** `Shuffled(seed)` is called on a list named `ids`
- **THEN** it returns another list named `ids_Shuffled` and the original list keeps its order

#### Scenario: A table
- **WHEN** `Shuffled(seed)` is called on a table with row names
- **THEN** the copy holds the same rows in another order with each row's cells and name together, and the original table is unchanged

### Requirement: Shuffle and Shuffled agree for one seed

For the same `SamplingOptions` seed, `Shuffled` SHALL give the order `Shuffle` gives on this line.

#### Scenario: One seed, one order
- **WHEN** `Shuffle` and `Shuffled` are called with the same seed on equal lists or tables
- **THEN** the results are equal

### Requirement: The CLI's shuffle stores a copy

The CLI's `shuffle <var> … as <name>` SHALL store a shuffled copy under `<name>` and SHALL leave `<var>` in its order, for a DataList and a DataTable alike.

#### Scenario: Shuffling a variable
- **WHEN** `shuffle x seed 42 as out` runs
- **THEN** `out` is another value and `x` keeps its order
