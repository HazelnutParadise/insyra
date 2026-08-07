## ADDED Requirements

### Requirement: Convolution matches the reference across its attribute space
The system SHALL execute 2-D convolution with padding, strides, dilations and groups, matching the reference runtime across attribute combinations rather than only defaults.

#### Scenario: Attribute combinations are proved individually

- **WHEN** the one-op parity harness runs for convolution and pooling
- **THEN** generated cases cover asymmetric padding, non-unit strides, dilation, grouped and depthwise convolution, and pooling padding modes, each matching the reference within single-precision tolerance

#### Scenario: An image classifier runs

- **WHEN** a fixed-weight convolutional classifier is run by both runtimes on the same inputs
- **THEN** outputs agree within single-precision tolerance
