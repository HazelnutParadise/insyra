## REMOVED Requirements

### Requirement: The index filters and Headers are deprecated spellings
**Reason**: The ten `FilterColsByColIndex…` and `FilterRowsByRowIndex…` methods were removed in 0.4, the release after the one that deprecated them.
**Migration**: Use `SliceCols` and `SliceRows`.

### Requirement: Headers and SetHeaders are ColNames and SetColNames
**Reason**: `Headers` and `SetHeaders` were removed in 0.4, the release after the one that deprecated them.
**Migration**: Use `ColNames` and `SetColNames`.
