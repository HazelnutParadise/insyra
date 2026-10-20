## ADDED Requirements

### Requirement: A tensor knows where it lives and moves only when asked
`Tensor.To(device)` SHALL return a tensor holding the same values on that device, `Tensor.ToHost()` SHALL return a host tensor with the same values, and `Tensor.Device()` SHALL report where a tensor lives, the zero value meaning the host. Neither move SHALL change the source tensor. Values SHALL round-trip bit for bit, including NaN payloads, infinities and signed zeros.

#### Scenario: A round trip
- **WHEN** a float32 tensor is moved with `To` and back with `ToHost`
- **THEN** every element has the same bits as before

#### Scenario: A host tensor reports the host
- **WHEN** `Device()` is called on a tensor that was never moved
- **THEN** it reports the host

### Requirement: A host view of a resident tensor is refused
`Float32Data`, `Parameter.Update` and `Parameter.Grad` on a resident tensor SHALL return an error that names `ToHost`. Nothing SHALL read a resident tensor back without the caller asking.

#### Scenario: A silent readback is impossible
- **WHEN** `Float32Data` is called on a resident tensor
- **THEN** it returns an error saying the tensor is on a device and to call `ToHost`

### Requirement: A tape operation runs where its operands live
`Tape.MatMul`, `Tape.Add` and the gradient accumulation and `SGD` that follow them SHALL run on the device when every operand is resident on it and SHALL leave their results there. `MatMul` SHALL read a transposed operand in place, so a reverse pass materialises no `Transpose`. An operation whose operands live on different devices SHALL be an error.

#### Scenario: A forward and reverse pass stay resident
- **WHEN** a tape with resident parameters runs `MatMul`, `Add` and `Backward`
- **THEN** the outputs and gradients are resident and nothing was read back

#### Scenario: Mixed devices
- **WHEN** a resident tensor and a host tensor reach one operation
- **THEN** it returns an error naming both locations

### Requirement: An operation without a device kernel falls back in the open
An operation on resident operands that has no device kernel SHALL download its inputs, run on the host, upload its result and record a fallback reason in the accel report. In strict GPU mode it SHALL return an error instead.

#### Scenario: No kernel yet
- **WHEN** an operation without a device kernel receives resident tensors
- **THEN** it returns the host result as a resident tensor and the report names the fallback

#### Scenario: Strict mode
- **WHEN** the same operation runs in strict GPU mode
- **THEN** it returns an error and nothing runs on the host

### Requirement: A missing device changes speed, never the answer
With no usable device, `To` SHALL return a host tensor, every operation SHALL run on the host, and the results SHALL equal those of a run with a device, bit for bit in `PrecisionFloat32Rounded`. The report SHALL say why no device ran.

#### Scenario: No accelerator
- **WHEN** the same training step runs on a machine without a usable device
- **THEN** it returns the same loss and gradients and the report names `no-accelerator`

### Requirement: A tape releases what it holds
`Tape.Clear()` SHALL drop the recorded operations and every intermediate tensor of the step and SHALL keep parameters and optimizer state. `Tape.Close()` SHALL release everything the tape holds on a device. Resident tensors SHALL be charged to the accel memory budget, and a tensor that cannot be placed within it SHALL fail with an error.

#### Scenario: Clear keeps training state
- **WHEN** `Clear` is called between two steps
- **THEN** the parameters and optimizer state are those the first step left, and the first step's intermediates are released

#### Scenario: Close frees device memory
- **WHEN** `Close` is called
- **THEN** the memory the tape held on the device is released and later use is an error
