## REMOVED Requirements

### Requirement: The replaced names keep their meaning for one release
**Reason**: `ParseColIndex`, `CalcColIndex` and `Config.GetDoesUseColoredOutput` were removed from the root package in 0.4, the release after the one that deprecated them.
**Migration**: Use `ColIndexToNumber`, `ColNumberToIndex` and `Config.GetUseColoredOutput`.
