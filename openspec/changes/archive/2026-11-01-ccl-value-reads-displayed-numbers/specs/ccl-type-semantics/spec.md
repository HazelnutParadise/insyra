## ADDED Requirements

### Requirement: VALUE reads numbers as a spreadsheet displays them

`VALUE` and `TONUM` SHALL read text with a trailing percent sign as that number divided by 100, and SHALL read commas that group digits in threes as thousands separators, with or without decimals. Text with misplaced commas, or a percent sign on anything but a plain decimal, SHALL give an empty result.

#### Scenario: Percent and thousands

- **WHEN** the text is `10%`, `12.3%` or `1,234.5`
- **THEN** `VALUE` gives 0.1, 0.123 and 1234.5

#### Scenario: Misplaced commas

- **WHEN** the text is `1,23` or `1234,567`
- **THEN** `VALUE` gives an empty result
