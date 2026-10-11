## ADDED Requirements

### Requirement: A test comparison never lets NaN stand for a number

A shared numeric comparison helper in the tests SHALL treat `NaN` as equal only to `NaN`: a `NaN` where a number is expected, or a number where `NaN` is expected, SHALL fail the comparison.

#### Scenario: NaN where a number is expected
- **WHEN** `approxEqualAny([]any{NaN}, []any{1.0}, 1e-9)`
- **THEN** 回傳 false

#### Scenario: NaN where NaN is expected
- **WHEN** `approxEqualAny([]any{NaN}, []any{NaN}, 1e-9)`
- **THEN** 回傳 true
