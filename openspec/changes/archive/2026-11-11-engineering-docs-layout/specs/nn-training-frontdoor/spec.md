## MODIFIED Requirements

### Requirement: Fit is sugar that changes nothing
A Fit call configured to match the documented hand-written tape loop — same seed, same batch walk, same optimizer and loss — SHALL reproduce that loop's loss sequence to the last digit, extending the sugar-changes-nothing gate in `engineering/architecture.md` to Fit.

#### Scenario: Digit-for-digit parity with the hand loop

- **WHEN** the documented MNIST-class hand loop and an equivalently configured Fit run under the same seed
- **THEN** their per-epoch loss sequences are identical to the last digit

#### Scenario: Convergence carries over

- **WHEN** the ungated micro-convergence setup runs through Fit
- **THEN** it reaches the same result as the tape-loop version

#### Scenario: Dropout stays out of validation

- **WHEN** a Sequential containing training-only layers reports validation loss
- **THEN** the validation pass structurally excludes those layers, exactly as Predict does
