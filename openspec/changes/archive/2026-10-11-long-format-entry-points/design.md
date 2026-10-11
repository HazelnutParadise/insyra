# Design: long-format-entry-points

## Decisions

### String parameters for column selection

On this branch, column selectors are strings rather than typed `any` selectors (`Name(...)` or int positions are left on 0.4). Selectors resolve as an existing column's Excel letter first, then an exact column name, matching `GetCol`.

### Pre-allocation level combination checks

`TwoWayANOVAFromTable`, `RepeatedMeasuresANOVAFromTable` and `FriedmanTestFromTable` check for missing or repeated factor level combinations before allocating the cell grid, avoiding allocating levels-squared cells for tables with many distinct values.

### The table-taking functions call the list-taking ones

The new functions read the table, group the values by level, and call `TwoWayANOVA`, `RepeatedMeasuresANOVA` or `FriedmanTest` with the groups in the order their levels first appear. That makes the results bit-for-bit those of the existing functions by construction, keeps a single implementation of each computation, and leaves `stats/anova.go` and `stats/nonparam_friedman.go` untouched.

### Columns are resolved on a copy of the table

`stats` resolves selectors on a `Clone()` of the table, returning any error recorded on the clone, so the caller's table does not retain sticky errors in `Err()`.

### Levels compare like group keys

A level is keyed with `insyra.ToMapKey`, so `int64(1)` and Go literal `1` are one level, while `1.0` and `"1"` are levels of their own. `nil` and `NaN` are refused rather than treated as a level.
