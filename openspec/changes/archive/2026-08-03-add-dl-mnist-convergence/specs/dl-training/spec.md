## ADDED Requirements

### Requirement: Training converges on a real dataset

The tape SHALL train an MLP classifier on the real MNIST dataset from
seeded initialization to at least 95% test-set accuracy within a bounded
number of epochs, with the final training loss well below the initial
loss. The run SHALL be deterministic under its fixed seed and SHALL be
gated on an operator-provided dataset directory, skipping cleanly and
attempting no network access when it is absent. A dataset-free
micro-convergence test SHALL verify the loop mechanics everywhere.

#### Scenario: MNIST reaches the accuracy target

- **WHEN** the gated test runs with the four IDX files present
- **THEN** the trained MLP SHALL score at least 95% on the 10k test
  images and the loss curve SHALL end well below where it started

#### Scenario: The micro-convergence test runs everywhere

- **WHEN** the ungated test trains the tiny two-class problem
- **THEN** it SHALL reach perfect accuracy in bounded steps with no
  dataset or toolchain involved

#### Scenario: Absent data skips cleanly

- **WHEN** `INSYRA_DL_MNIST_DIR` is unset or files are missing
- **THEN** the test SHALL skip naming the variable and touch no network
