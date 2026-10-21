## ADDED Requirements

### Requirement: Exp and log are correctly rounded
`nn` SHALL provide float32 `exp` and `log` whose result is the exact value rounded once to nearest-even, on every platform, computed with IEEE `+ - * /` only. Special inputs SHALL follow IEEE 754: `exp(-inf)` is `+0`, `exp(+inf)` is `+inf`, `log(0)` is `-inf`, `log` of a negative is NaN.

#### Scenario: Exhaustive agreement
- **WHEN** each function is compared with its oracle over every float32 input
- **THEN** every result has the oracle's bits

#### Scenario: The device agrees
- **WHEN** the device versions run on every input on hardware
- **THEN** every result has the host's bits
