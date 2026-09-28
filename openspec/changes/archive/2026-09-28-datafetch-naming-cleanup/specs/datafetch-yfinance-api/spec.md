## ADDED Requirements

### Requirement: One statement frequency has one name

`YFPeriodAnnual` SHALL be the name for annual statements. `YFPeriodYearly` SHALL remain for one release as a Deprecated constant with its value `"yearly"`, whose doc comment names `YFPeriodAnnual`, so a table fetched with it keeps its `yearly` label until the constant is removed.

#### Scenario: The deprecated synonym
- **WHEN** 讀取 `YFPeriodYearly` 的 doc comment 與值
- **THEN** 有指名 `YFPeriodAnnual` 的 `Deprecated:` 段落，值仍為 `"yearly"`
