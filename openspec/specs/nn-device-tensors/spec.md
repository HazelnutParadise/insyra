# nn-device-tensors Specification

## Purpose
Tensors that live on a device through a whole training step: explicit movement to and from the host, operations that run where their operands live, and a tape that releases what it holds. In PrecisionFloat32Rounded the same step returns the same bits on the host, on any GPU and with no GPU.
## Requirements
### Requirement: A tensor knows where it lives and moves only when asked
`Tensor.To(device)` SHALL return a tensor holding the same values on that device, `Tensor.ToHost()` SHALL return a host tensor with the same values, and `Tensor.Device()` SHALL report where a tensor lives, the zero value meaning the host. Float32 and int64 tensors SHALL be accepted; int64 values SHALL be held as int32 and a value that does not fit, or is negative, SHALL be an error naming it. Neither move SHALL change the source tensor. Values SHALL round-trip bit for bit.

#### Scenario: A round trip
- **WHEN** a float32 or int64 tensor is moved with `To` and back with `ToHost`
- **THEN** every element equals the original

#### Scenario: A host tensor reports the host
- **WHEN** `Device()` is called on a tensor that was never moved
- **THEN** it reports the host

#### Scenario: A label that does not fit
- **WHEN** an int64 tensor holding 2^40 is moved to a device
- **THEN** `To` returns an error naming that value

#### Scenario: A label past the class count
- **WHEN** the largest label on the device is not below the logits' class count
- **THEN** `SoftmaxCrossEntropy` returns an error before any kernel runs

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

### Requirement: An operation without a device kernel refuses a device tensor
An operation on a device tensor that has no device kernel yet SHALL return an error that names `ToHost`. It SHALL NOT read the tensor back by itself. Automatic download, host execution and upload with a recorded fallback reason arrive with the kernels that need them.

#### Scenario: No kernel yet
- **WHEN** `Relu` or `Tape.Relu` receives a device tensor
- **THEN** it returns an error naming `ToHost` and nothing runs on the host

#### Scenario: A device tensor on a default tape
- **WHEN** a device tensor reaches a tape not set to `PrecisionFloat32Rounded`
- **THEN** the operation returns an error naming `PrecisionFloat32Rounded`

### Requirement: A missing device changes speed, never the answer
`accel.DefaultDevice` SHALL return the zero Device when no device is usable and record why in the report; `To` with the zero Device SHALL return a host copy. A step on host tensors in `PrecisionFloat32Rounded` SHALL equal the same step on a device, bit for bit.

#### Scenario: No accelerator
- **WHEN** the same training step runs on a machine without a usable device
- **THEN** it returns the same loss and gradients, and the report names the reason

### Requirement: A tape releases what it holds
`Tape.Clear()` SHALL drop the recorded operations and every intermediate tensor of the step and SHALL keep parameters and optimizer state. `Tape.Close()` SHALL release everything the tape holds on a device. Resident tensors SHALL be charged to the accel memory budget, and a tensor that cannot be placed within it SHALL fail with an error.

#### Scenario: Clear keeps training state
- **WHEN** `Clear` is called between two steps
- **THEN** the parameters and optimizer state are those the first step left, and the first step's intermediates are released

#### Scenario: Close frees device memory
- **WHEN** `Close` is called
- **THEN** the memory the tape held on the device is released and later use is an error

### Requirement: Optimizer state lives where the parameter lives
A parameter on a device SHALL keep its Adam moments on the same device, allocated on first use and released by `Tape.Close`. A step SHALL not read the state back.

#### Scenario: State stays resident
- **WHEN** several Adam steps run on a device parameter
- **THEN** no moment is read back to the host between steps

### Requirement: Custom records resident tensors on a PrecisionFloat32 tape
`Tape.Custom` SHALL accept an output and inputs that are all resident on one device when the tape is in `accel.PrecisionFloat32`, SHALL refuse them in any other mode with an error naming `PrecisionFloat32`, and SHALL refuse a mix of host and device tensors or of two devices. `SetPrecision` SHALL accept `accel.PrecisionFloat32`, and built-in operations on such a tape SHALL compute as in `PrecisionFloat32Rounded`.

#### Scenario: A caller kernel between built-in operations
- **WHEN** a resident `MatMul`, a `Custom` operation whose output a caller's kernel wrote, and `BackwardFrom` run on a `PrecisionFloat32` tape
- **THEN** the weight gradient has the same bits as the host oracle and no tensor was read back during the step

#### Scenario: Wrong mode
- **WHEN** `Custom` receives resident tensors on a `PrecisionFloat32Rounded` or default tape
- **THEN** it returns an error naming `PrecisionFloat32` and records nothing

### Requirement: The tape owns a resident Custom output and its gradients
A resident output given to `Tape.Custom` SHALL be released by `Clear` and `Close`. Every gradient the reverse rule returns SHALL be owned the same way, and SHALL be refused when it is not a float32 tensor on the output's device.

#### Scenario: Clear releases the output
- **WHEN** a resident output was recorded with `Custom` and the tape is cleared
- **THEN** the device's resident byte count returns to what it was before the output existed

#### Scenario: A gradient on the host
- **WHEN** the reverse rule of a resident operation returns a host tensor
- **THEN** the backward pass fails with an error saying the gradient is not on the device

### Requirement: A tensor lends and adopts device buffers
`Tensor.DeviceBuffer` SHALL return the resident buffer of a float32 tensor without copying it and SHALL be an error for a host or int64 tensor. `nn.NewDeviceTensor(shape, buffer)` SHALL wrap a non-released device buffer whose length matches the shape as a tensor that owns it, and SHALL leave the buffer with the caller when it returns an error.

#### Scenario: A refused adoption
- **WHEN** `NewDeviceTensor` receives a buffer whose length does not match the shape
- **THEN** it returns an error and the buffer is not released

