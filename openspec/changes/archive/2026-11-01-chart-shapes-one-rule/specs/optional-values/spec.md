## MODIFIED Requirements

### Requirement: Heat map points have an exported type

The heat map point type SHALL be exported as `HeatmapPoint[X, Y]` with the exported axis constraint `HeatmapAxis`, built by `NewHeatmapPoint` and `NewHeatmapMissingPoint`, so points can be collected in a slice outside the package.

#### Scenario: Points built in a loop

- **WHEN** a caller appends `NewHeatmapPoint(x, y, v)` results to a `[]HeatmapPoint[int, int]` and passes it to `CreateHeatmapChart`
- **THEN** it compiles and produces a chart
