## ADDED Requirements

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
