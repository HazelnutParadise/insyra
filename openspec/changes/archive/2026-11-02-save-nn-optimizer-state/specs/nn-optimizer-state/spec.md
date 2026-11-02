## ADDED Requirements

### Requirement: Named binary optimizer persistence
Tape SHALL save and load versioned optimizer state using a one-to-one stable-name mapping covering all its parameters. State SHALL preserve every supported optimizer slot and uint64 step bit for bit in binary form, with shape, dtype and tape precision checked. Weights, gradients, graph, freeze settings, RNG and optimizer hyperparameters SHALL remain separate.

#### Scenario: Restart a custom loop
- **WHEN** Adam, AdamW or momentum updates are split by save and load in a new process with matching weights, precision and inputs, using the same arithmetic/platform in default precision or the rounded built-in host/device path
- **THEN** all subsequent weights, moments, velocities and counters equal the uninterrupted run bit for bit

#### Scenario: Resident Adam state
- **WHEN** resident Adam or AdamW state is saved and restored into a compatible new tape
- **THEN** the next step equals the uninterrupted rounded reference, with all resident resources explicitly owned

#### Scenario: Scalars and empty parameters
- **WHEN** parameters have scalar or zero-element shapes
- **THEN** their descriptors and initialized optimizer slots round-trip, and SafeTensors element counts treat any zero dimension as zero elements

### Requirement: Transactional import validation
Load SHALL reject unsupported versions, invalid names, missing or extra slots, incompatible shapes/dtypes/precision, non-finite arrays, negative second moments and inconsistent or overflowing counters before changing any parameter. All replacement device buffers SHALL be staged before commit; an allocation failure SHALL release staged buffers and preserve old state.

#### Scenario: Invalid later parameter
- **WHEN** a file contains valid state for one parameter and invalid state for another
- **THEN** every parameter's values, gradients and optimizer state remain unchanged

#### Scenario: Uninitialized optimizer slots
- **WHEN** valid uninitialized state is loaded over previously used parameters
- **THEN** their old optimizer slots are cleared without changing weights or gradients

### Requirement: Bounded optimizer checkpoint input
Load SHALL bound header allocation from registered names and ranks, allowing 64 KiB for metadata and whitespace. It SHALL reject unknown keys and incompatible wire descriptors before reading payload, read only the validated parameter-sized payload plus at most one trailing byte, and decode without a second whole-file copy. Invalid or oversized input SHALL leave all parameters unchanged.

#### Scenario: Oversized or unknown header
- **WHEN** the stream declares a header exceeding the model-derived budget or contains unknown keys or incompatible wire shapes
- **THEN** Load rejects it before reading payload or changing parameter state

#### Scenario: Unbounded trailing input
- **WHEN** a valid optimizer checkpoint is followed by additional bytes
- **THEN** Load rejects it after reading at most one byte beyond the checkpoint, with all parameter state unchanged

### Requirement: Explicit Clear gradient policy
Clear SHALL preserve its existing signature and clear-all default. SetClearOptions SHALL select whether parameter gradients survive Clear. Clear SHALL release graph history and intermediate resources while preserving parameters and optimizer state. Retained gradients SHALL be replaced by the next successful Backward; Close SHALL release them regardless of policy.

#### Scenario: Repeated cleanup cycles
- **WHEN** forward, backward, update and Clear run repeatedly
- **THEN** graph history is released and the next optimizer update exactly matches a fresh-graph reference

#### Scenario: Retain gradients
- **WHEN** KeepGradients is true and Clear runs
- **THEN** parameter gradients remain queryable and usable for updates while non-parameter gradients and graph resources are released

#### Scenario: Retained gradient becomes a graph input
- **WHEN** a retained resident gradient becomes an input to a later graph and that graph is differentiated repeatedly
- **THEN** its buffer remains available until the graph and all parameter-gradient references are cleared, and Close releases every owned buffer

### Requirement: Frozen built-in updates
A frozen parameter SHALL be skipped by SGD, momentum, Adam and AdamW without applying weight decay or advancing optimizer state. Unfreezing SHALL resume with its preserved state. Freeze SHALL not prevent explicit caller-managed Parameter.Update.

#### Scenario: Freeze with stale momentum and gradients
- **WHEN** a previously updated parameter is frozen and built-in optimizers run
- **THEN** its values, velocity, moments and step remain bit-identical until unfrozen
