# Design: long-format-entry-points

## Decisions

### Named parameters, not `factorCols ...any`

The issue suggested `(dt, valueCol, factorCols...)`. Each of the three tests has a fixed number of roles, and the roles differ: the two-way test has two factors, the repeated-measures tests have a condition and a subject that are not interchangeable. Named parameters make a missing or swapped column a compile error or at least a visible one, where a variadic list would need a run-time count check. The order `(valueCol, conditionCol, subjectCol)` follows R's `friedman.test(y, groups, blocks)` and pingouin's `rm_anova(data, dv, within, subject)`, which agree.

### The table-taking functions call the list-taking ones

The new functions read the table, group the values by level, and call `TwoWayANOVA`, `RepeatedMeasuresANOVA` or `FriedmanTest` with the groups in the order their levels first appear. That makes the results bit-for-bit those of the existing functions by construction, keeps a single implementation of each computation, and leaves `stats/anova.go` and `stats/nonparam_friedman.go` untouched. The cost is building one `DataList` per cell or per subject, which the list-taking functions already require of their callers.

The statistics do not depend on the order of the levels or of the rows, so the first-appearance order only fixes which floating-point summation order is used; it is stated so the equality with the list-taking functions is exact.

### Columns are resolved on a copy of the table

`stats` has no silent way to resolve a selector: `GetCol` records a failure on the table's sticky `Err()`, which would leave the caller's table carrying an error from a function that already returned one. The functions take one `Clone()` of the table, the same snapshot `ml` takes of its inputs, resolve the columns on it, and return what it recorded as the error. The copy also gives one consistent snapshot of every column the test reads, so a concurrent writer cannot mix two states of the table. The failed lookup still writes its message to insyra's log, because `GetCol` logs as well as records; a silent resolver in the root package would remove both the log line and the copy, and is recorded as an `AGENTS.md` follow-up.

### Levels compare like group keys

A level is keyed with `insyra.ToMapKey`, the key `Counter` and `Count` use, so `int64(1)` from a CSV and a Go literal `1` are one level, while `1.0` and `"1"` are levels of their own, as they are to `GroupBy`. `nil` and `NaN` are refused rather than treated as a level: `NaN` is not equal to itself, so as a map key every `NaN` row would become a level of its own, and R drops such rows by default, which would silently change the design. An empty string is a level like any other text, as it is to `GroupBy` and to the chi-square tests.

## Risks / Trade-offs

- For a design with many subjects, building one list per subject costs more than a direct computation would. The list-taking functions already cost that for callers who build the lists themselves, and a direct path can be added later behind the same signatures if a measurement shows it matters.
