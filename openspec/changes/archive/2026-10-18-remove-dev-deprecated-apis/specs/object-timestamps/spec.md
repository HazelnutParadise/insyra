## REMOVED Requirements

### Requirement: The second-resolution getters remain for one release
**Reason**: `GetCreationTimestamp()` and `GetLastModifiedTimestamp()` were removed from `DataList` and `DataTable` in 0.4, the release after the one that deprecated them.
**Migration**: Use `CreationTime()` and `LastModifiedTime()`, and `.Unix()` when seconds are wanted.
