## MODIFIED Requirements

### Requirement: A device edge sum is earned by measurement and matches the CPU bit for bit

No production device kernel for `EdgeSum` SHALL exist until a recorded measurement shows the device faster than the all-core CPU at the measured sizes, with every upload and readback counted, and shows a kernel whose results are bit-identical to the CPU's. The measurement and its verdict SHALL be recorded in a record under `engineering/decisions/`.

#### Scenario: The device is measured before a kernel is proposed
- **WHEN** a device path for `EdgeSum` is proposed
- **THEN** a record under `engineering/decisions/` already records device and all-core CPU times per size and the bit-for-bit comparison of the kernel against the CPU

#### Scenario: The device cannot match the CPU order
- **WHEN** a kernel is not bit-identical to the CPU's result
- **THEN** no device path is wired
