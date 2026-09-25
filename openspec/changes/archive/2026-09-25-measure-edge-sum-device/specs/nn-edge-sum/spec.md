## ADDED Requirements

### Requirement: A device edge sum is earned by measurement and matches the CPU bit for bit

No production device kernel for `EdgeSum` SHALL exist until a recorded measurement shows the device faster than the all-core CPU at the measured sizes, with every upload and readback counted, and shows a kernel variant whose results are bit-identical to the CPU's contracted order. The measurement and its verdict SHALL be recorded in `delivery-status.md`.

#### Scenario: The device is measured before a kernel is proposed
- **WHEN** a device path for `EdgeSum` is proposed
- **THEN** `delivery-status.md` already records device and all-core CPU times per size and the bit-for-bit comparison of each kernel variant against the CPU order

#### Scenario: The device cannot match the CPU order
- **WHEN** no kernel variant is bit-identical to the unfused CPU order
- **THEN** no device path is wired, and the choice of order goes to the owner
