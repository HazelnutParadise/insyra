## MODIFIED Requirements

### Requirement: A tensor knows where it lives and moves only when asked
`Tensor.To(device)` SHALL return a tensor holding the same values on that device, `Tensor.ToHost()` SHALL return a host tensor with the same values, and `Tensor.Device()` SHALL report where a tensor lives, the zero value meaning the host. Float32 and int64 tensors SHALL be accepted; int64 values SHALL be held as int32 and a value that does not fit, or is negative, SHALL be an error naming it. Neither move SHALL change the source tensor. Values SHALL round-trip bit for bit.

#### Scenario: A round trip
- **WHEN** a float32 or int64 tensor is moved with `To` and back with `ToHost`
- **THEN** every element equals the original

#### Scenario: A label that does not fit
- **WHEN** an int64 tensor holding 2^40 is moved to a device
- **THEN** `To` returns an error naming that value

#### Scenario: A label past the class count
- **WHEN** the largest label on the device is not below the logits' class count
- **THEN** `SoftmaxCrossEntropy` returns an error before any kernel runs
