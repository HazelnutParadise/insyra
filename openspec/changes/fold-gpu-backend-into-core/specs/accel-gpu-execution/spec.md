## MODIFIED Requirements

### Requirement: GPU dependency stays outside the core module
The system SHALL make GPU execution available to every consumer of the core module without a second install step or a registration import.

#### Scenario: A consumer installs the library
- **WHEN** a consumer adds the core `insyra` module to a project
- **THEN** GPU execution is available through the accel runtime without installing a further module
- **AND** without importing a package for its registration side effect

#### Scenario: A consumer never uses acceleration
- **WHEN** a consumer imports only non-accel packages
- **THEN** the GPU implementation is not compiled into their program
- **AND** the GPU dependency's minimum Go version does not apply to their build

#### Scenario: A third party supplies its own backend
- **WHEN** a third party registers an execution backend for a backend kind
- **THEN** the runtime routes eligible workloads for that kind to the registered backend

#### Scenario: A consumer installs through allpkgs
- **WHEN** a consumer installs Insyra through the `allpkgs` convenience package
- **THEN** the GPU backend is registered without any further import
- **AND** no device is probed until an accel session opens
