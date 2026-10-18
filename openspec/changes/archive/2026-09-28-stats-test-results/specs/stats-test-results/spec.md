## ADDED Requirements

### Requirement: Every hypothesis test result is a HypothesisTestResult

`TTestResult`, `ZTestResult`, `FTestResult`, `ChiSquareTestResult`, `CorrelationResult`, `WilcoxonTestResult`, `MannWhitneyUResult`, `KruskalWallisResult` and `FriedmanTestResult` SHALL embed the exported `TestResult` and satisfy `HypothesisTestResult`. `Base()` SHALL return the embedded struct itself, not a copy.

#### Scenario: Mixed results in one slice
- **WHEN** a t-test result and a Mann-Whitney U result are put in a `[]HypothesisTestResult`
- **THEN** `Base().PValue` of each equals the `PValue` read on the result

#### Scenario: Base is not a copy
- **WHEN** a field is changed through `Base()`
- **THEN** the same field on the result reads the new value
