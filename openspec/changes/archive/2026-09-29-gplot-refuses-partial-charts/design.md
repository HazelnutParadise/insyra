# Design: gplot-refuses-partial-charts

## Decisions

### Every series that cannot be drawn fails the call, not only a length mismatch

The owner approved failing the chart on a length mismatch. An empty series and a series holding a `NaN` or an infinity reach the same code path and have the same effect on the caller: fewer series on the chart than were asked for, with a nil error. Refusing only the length mismatch would leave that effect in place for the other two. Treating all three alike also matches the rest of `gplot`, where a bar chart or histogram holding a `NaN` is already an error.

### The error lists every failing series

The constructor still tries every series before it returns, so one call reports all of them: `gplot: CreateLineChart: series "b" has 2 values but XAxis has 3; series "c" cannot be drawn: …`. A caller fixing the data sees the whole list at once instead of one failure per attempt.

### A nil list is not a series

A nil list among real ones stays a warning. It is missing input rather than a series that failed to draw, and `plot` treats it the same way; #413 decides what both packages do with missing input.
