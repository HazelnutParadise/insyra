## ADDED Requirements

### Requirement: Heat map points have an exported type

The heat map point type SHALL be exported as `HeatMapPoint[X, Y]` with the exported axis constraint `HeatMapAxis`, built by `NewHeatMapPoint` and `NewHeatMapMissingPoint`, so points can be collected in a slice outside the package.

#### Scenario: Points built in a loop

- **WHEN** a caller appends `NewHeatMapPoint(x, y, v)` results to a `[]HeatMapPoint[int, int]` and passes it to `CreateHeatMap`
- **THEN** it compiles and produces a chart
