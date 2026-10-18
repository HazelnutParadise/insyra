## REMOVED Requirements

### Requirement: One statement frequency has one name
**Reason**: It named deprecated APIs that were removed in 0.4, and its scenarios cannot be edited in place; it is restated under a new name in this change.
**Migration**: See the requirement "A statement frequency has one constant".

## ADDED Requirements

### Requirement: A statement frequency has one constant

`YFPeriodAnnual` SHALL be the name for annual statements, with the value `"annual"`. The former synonym `YFPeriodYearly` SHALL NOT exist, so a statement frequency has exactly one constant.

#### Scenario: The annual constant
- **WHEN** 讀取 `YFPeriodAnnual` 的值
- **THEN** 值為 `"annual"`，套件中沒有 `YFPeriodYearly`
