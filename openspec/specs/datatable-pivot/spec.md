# datatable-pivot Specification

## Purpose
Says how `Pivot` takes the aggregation it applies to duplicate (Index, Columns) rows: the same `AggregateOp` that `Aggregate` and `Resample` take, with nil meaning that duplicates are an error, and how the CLI reads it.
## Requirements
### Requirement: Pivot takes its aggregation as an AggregateOp

`PivotConfig.AggFunc` SHALL be a `*AggregateOp`. A nil `AggFunc` SHALL make duplicate `(Index, Columns)` rows an error. A non-nil `AggFunc` SHALL aggregate duplicates with that op, exactly as `Aggregate` does. A value outside the `AggregateOp` constants SHALL be refused as an unknown `AggFunc`, and `OpCustom` without `Custom` SHALL be refused. `isr.Pivot.Agg` SHALL have the same type and meaning.

#### Scenario: Sum of duplicates

- **WHEN** `Pivot` is called with `AggFunc: new(OpSum)` on a table where region APAC has product A twice, with sales 10 and 5
- **THEN** the APAC row's A cell is 15

#### Scenario: No aggregation set

- **WHEN** `Pivot` is called on the same table with a nil `AggFunc`
- **THEN** it returns an error about duplicate combinations

### Requirement: The CLI pivot reads its op like groupby and resample

The `pivot` command SHALL read `agg <op>` with the parser `groupby` and `resample` use, and SHALL refuse a name that parser does not know with an `unknown aggregate op` error.

#### Scenario: An alias the three commands share

- **WHEN** a user runs `pivot sales index region columns product values sales agg avg`
- **THEN** duplicate cells hold their mean

#### Scenario: A name only pivot used to take

- **WHEN** a user runs `pivot … agg average`
- **THEN** the command fails with `unknown aggregate op`

