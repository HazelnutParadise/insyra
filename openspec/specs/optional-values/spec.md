# optional-values Specification

## Purpose
What a function does when its trailing optional parameter is given more than one value. The parameter stands for one optional value, so a second one is an error the caller hears about, never a value quietly dropped.
## Requirements
### Requirement: A trailing optional parameter takes at most one value

A function whose last parameter is a variadic standing for one optional value (`...T` or `...XxxOptions`) SHALL treat more than one value as an error, reported through the function's normal error path: a chainable method SHALL record it and leave its receiver unchanged, a function returning an error SHALL return it, and a constructor without an error result SHALL keep it and report it from the first call that can fail. No such function SHALL keep the first or the last value and drop the others.

#### Scenario: Two options structs to a sampling method

- **WHEN** `dt.Sample(2, false, opts, opts)` is called
- **THEN** `dt.Err()` reports that at most one `SamplingOptions` may be given, and no sample is drawn

#### Scenario: Two options to a finance function

- **WHEN** `finance.PMT(rate, 12, pv, fv, finance.PaymentEnd, finance.Options{Scale: 2}, finance.Options{Scale: 4})` is called
- **THEN** it returns an error instead of computing with the last `Options`

#### Scenario: Two seeds to a tape

- **WHEN** a tape is created with `nn.NewTape(1, 2)`
- **THEN** its `Param` and `Backward` return an error

### Requirement: Training batch normalization takes its settings by name

`Tape.BatchNormalizationTraining` and `BatchNormTraining` SHALL take momentum and epsilon as fields of one optional `BatchNormOptions`, a zero field keeping torch's default, and SHALL refuse more than one.

#### Scenario: Only epsilon set

- **WHEN** training batch normalization runs with `BatchNormOptions{Epsilon: 1e-3}`
- **THEN** momentum keeps its default of 0.1 and epsilon is 1e-3

### Requirement: Heat map points have an exported type

The heat map point type SHALL be exported as `HeatMapPoint[X, Y]` with the exported axis constraint `HeatMapAxis`, built by `NewHeatMapPoint` and `NewHeatMapMissingPoint`, so points can be collected in a slice outside the package.

#### Scenario: Points built in a loop

- **WHEN** a caller appends `NewHeatMapPoint(x, y, v)` results to a `[]HeatMapPoint[int, int]` and passes it to `CreateHeatMap`
- **THEN** it compiles and produces a chart

### Requirement: An encoding argument means the same in csvxl as in the core readers

`csvxl`'s CSV readers SHALL detect the encoding when given an empty string or `"auto"` in any case, as the core CSV readers do.

#### Scenario: Uppercase AUTO on a Big5 file

- **WHEN** `ReadCsvToString(path, "AUTO")` reads a Big5 file
- **THEN** the text is decoded from Big5 instead of the call failing

