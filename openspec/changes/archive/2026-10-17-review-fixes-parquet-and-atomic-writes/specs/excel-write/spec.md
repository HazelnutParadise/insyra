## ADDED Requirements

### Requirement: A replacement over a sheet named as the placeholder keeps no old content

When the workbook's only sheet is named `__insyra_placeholder__` and the sheet is replaced, the placeholder sheet SHALL take a name no other sheet uses, so the replaced sheet's old content does not stay behind.

#### Scenario: The only sheet is named as the placeholder
- **WHEN** the only sheet is named `__insyra_placeholder__` and holds `old` in A1, and the sheet is replaced
- **THEN** the workbook holds one sheet, and A1 of that sheet holds no value
