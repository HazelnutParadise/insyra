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

