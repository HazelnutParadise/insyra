## REMOVED Requirements

### Requirement: No-op functions say they do nothing
**Reason**: `ResetEvalDepth` and `ResetFuncCallDepth` did nothing and were removed from `engine/ccl` in 0.4.
**Migration**: Delete the calls; there is nothing to replace them with.
