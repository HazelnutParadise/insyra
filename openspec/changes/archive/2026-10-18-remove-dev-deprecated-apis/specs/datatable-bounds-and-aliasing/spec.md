## REMOVED Requirements

### Requirement: The largest parsed column index does not overflow
**Reason**: `FilterColsByColIndexLessThanOrEqualTo` was removed in 0.4 with the other deprecated index filters.
**Migration**: Use `SliceCols`, which has its own requirements.
