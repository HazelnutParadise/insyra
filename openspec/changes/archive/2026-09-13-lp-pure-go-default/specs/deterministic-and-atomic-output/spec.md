## REMOVED Requirements

### Requirement: lp additional-info table has a fixed row order

**Reason**: `SolveFromFile` and `SolveModel` are removed, and with them the additional-info table. The information it carried is now in `Solution`'s fields (`Status`, `Elapsed`, `Nodes`, `Log`).

**Migration**: Read the fields of the `Solution` that `lp.Solve` or `lp.SolveFile` returns.
