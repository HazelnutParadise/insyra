# Proposal: pivot-takes-aggregate-op

## Why

The aggregation APIs take their operation three ways (#230, T-17 in `api-review.md`). `GroupBy(...).Aggregate` and `Resample` take the typed `AggregateOp`; `PivotConfig.AggFunc` took a string with aliases (`"avg"`, `"std"`, `"average"`, `"stddev"`, `"variance"` and more), parsed by a private `parseAggOpName` that duplicated the CLI's `parseAggregateOp` with a slightly different list. A misspelled string was only caught at run time; a misspelled constant does not compile.

## What Changes

- **BREAKING**: `PivotConfig.AggFunc` is `*AggregateOp`, written `new(insyra.OpSum)` (Go 1.26, which insyra already requires). `nil` keeps the old meaning of the empty string: duplicate `(Index, Columns)` rows are an error. It has to be a pointer: `OpSum` is `AggregateOp`'s zero value, so a plain `AggregateOp` field would turn every config that sets no aggregation into a silent sum, and making `OpSum` non-zero would change what `Aggregate` does for a config with no `Op`. A value outside the constants is refused as an unknown `AggFunc`, as an unknown name was.
- **BREAKING**: `isr.Pivot.Agg` follows: `*insyra.AggregateOp`.
- The private `parseAggOpName` is deleted. The CLI's `pivot … agg <op>` reads the name with `parseAggregateOp`, the parser `groupby` and `resample` use, so the three commands accept the same names, as `Docs/cli-dsl.md` already said they did. Two names `pivot` took and the others did not stop working there: `average`, never documented for `pivot`, and `custom`, which failed anyway because the CLI cannot pass a function.

## Capabilities

### New Capabilities

- `datatable-pivot`: how `Pivot` takes its aggregation.

### Modified Capabilities

(none)

## Impact

- `datatable_pivot.go`, `isr/pivot.go`, `cli/commands/pivot.go`; tests in `datatable_pivot_test.go`, `isr/pivot_test.go`, `cli/commands/pivot_test.go`.
- `Docs/DataTable.md` (the `AggFunc` and `Custom` fields and the example), both changelogs, `api-review.md`, `delivery-status.md`. `Docs/cli-dsl.md` already describes the behaviour the CLI now has.
- The agent skills name no Pivot field and are unchanged.
