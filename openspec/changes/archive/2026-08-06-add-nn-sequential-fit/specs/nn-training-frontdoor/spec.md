# nn-training-frontdoor Delta: add-nn-sequential-fit

## ADDED Requirements

### Requirement: Fit trains a Sequential deterministically and visibly
`Sequential.Fit` SHALL train with epochs, batch size, seeded deterministic shuffling, an explicitly selected existing optimizer and loss, and optional validation tensors; SHALL refuse a config with no loss or no optimizer rather than defaulting one; SHALL emit one info-level progress line per epoch through the root logger unless silenced, with a callback receiving the same numbers when provided; and the same inputs, config, and seed SHALL produce the same parameter trajectory.

#### Scenario: Training is visible by default

- **WHEN** Fit runs for N epochs at default settings
- **THEN** exactly N info-level lines report epoch, mean training loss, elapsed time, and throughput
- **AND** validation loss appears in each line when validation tensors were given

#### Scenario: Quiet and callback compose

- **WHEN** a Progress callback is set and Quiet is true
- **THEN** the callback receives every epoch's numbers and no default line is written

#### Scenario: Determinism holds

- **WHEN** Fit runs twice with identical inputs, config, and seed
- **THEN** the resulting parameters are identical both times

#### Scenario: An unstated objective is refused

- **WHEN** FitConfig lacks a loss or an optimizer
- **THEN** Fit returns an error naming the missing selection instead of guessing

### Requirement: Fit is sugar that changes nothing
A Fit call configured to match the documented hand-written tape loop — same seed, same batch walk, same optimizer and loss — SHALL reproduce that loop's loss sequence to the last digit, extending the ENG.md sugar-changes-nothing gate to Fit.

#### Scenario: Digit-for-digit parity with the hand loop

- **WHEN** the documented MNIST-class hand loop and an equivalently configured Fit run under the same seed
- **THEN** their per-epoch loss sequences are identical to the last digit

#### Scenario: Convergence carries over

- **WHEN** the ungated micro-convergence setup runs through Fit
- **THEN** it reaches the same result as the tape-loop version

#### Scenario: Dropout stays out of validation

- **WHEN** a Sequential containing training-only layers reports validation loss
- **THEN** the validation pass structurally excludes those layers, exactly as Predict does
