## MODIFIED Requirements

### Requirement: A compiled pattern is reused

`REGEXTEST` SHALL compile a given pattern once and reuse it. The cache SHALL be bounded so that patterns built per row cannot grow it without limit, and an invalid pattern SHALL still be reported.

#### Scenario: The same pattern over many rows
- **WHEN** 對 100,000 列求值 `REGEXTEST(B, 'item-[0-9]+')`
- **THEN** 該樣式只編譯一次，結果與每列重新編譯相同
