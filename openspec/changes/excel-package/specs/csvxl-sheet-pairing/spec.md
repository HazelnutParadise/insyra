## REMOVED Requirements

### Requirement: The sheets of a workbook can be listed

**Reason**: Listing a workbook's sheets belongs with reading them, which moved to the `excel` package; `excel-package` carries the requirement.

**Migration**: call `excel.SheetNames(path)` instead of `csvxl.SheetNames(path)`.
