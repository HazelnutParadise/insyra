## REMOVED Requirements

### Requirement: Diag is deprecated and no longer panics
**Reason**: `Diag` was removed from `stats` in 0.4, the release after the one that deprecated it.
**Migration**: Use `DiagOf`, `DiagMatrix`, `IdentityMatrix` and `DiagMatrixSize`.
