## ADDED Requirements

### Requirement: A tape runs a caller's operation where its inputs live
`Tape.Apply(op, inputs...)` SHALL run `op.Device` when every input is resident on one device and `op.CPU` when every input is on the host, SHALL record the returned VJP for the reverse pass, and SHALL own the output so that `Clear` and `Close` release it. Inputs on different locations SHALL be an error naming both.

#### Scenario: The device half on resident inputs
- **WHEN** `Apply` runs a `DeviceOp` on resident inputs and `Backward` follows
- **THEN** the output and the gradients the VJP returns are resident and nothing was read back

#### Scenario: The CPU half without a device
- **WHEN** the same program runs where `accel.DefaultDevice` is the host
- **THEN** `Apply` runs the CPU half and the step completes

### Requirement: A caller's operation always has a CPU half
`Tape.Apply` SHALL refuse a `DeviceOp` whose `CPU` function is nil, and SHALL refuse a VJP result whose gradients do not live where the inputs live.

#### Scenario: No CPU half
- **WHEN** `Apply` receives a `DeviceOp` with only a `Device` function
- **THEN** it returns an error saying the operation needs a CPU half, and nothing runs

### Requirement: A caller's operation declares its precision
A `DeviceOp` SHALL declare `PrecisionFloat32Rounded` or `PrecisionFloat32`. A tape in `PrecisionFloat32Rounded` SHALL refuse a `PrecisionFloat32` operation unless `Tape.AllowDeviceDependent` was called, and once one has run, `Tape.DeviceDependent` SHALL report true and the accel report SHALL name the operation.

#### Scenario: A device-dependent operation is labelled
- **WHEN** a tape that allows device-dependent operations runs a `PrecisionFloat32` `DeviceOp`
- **THEN** `DeviceDependent` reports true and the report names the operation

#### Scenario: Refused without consent
- **WHEN** a Rounded tape that was not told to allow them receives a `PrecisionFloat32` `DeviceOp`
- **THEN** `Apply` returns an error naming `AllowDeviceDependent`

### Requirement: The two halves of an operation can be checked
`nn.CheckDeviceOp` SHALL run both halves of a `DeviceOp` and their VJPs on the same inputs and upstream gradient, and SHALL return the first element whose bits differ, or none.

#### Scenario: A faithful pair
- **WHEN** `CheckDeviceOp` runs a `DeviceOp` built from a `Library` dot-product kernel and its CPU twin
- **THEN** it reports no difference

### Requirement: A tensor lends and adopts device buffers
`Tensor.DeviceBuffer` SHALL return the resident buffer of a float32 tensor without copying it and SHALL be an error for a host or int64 tensor. `nn.NewDeviceTensor(buffer, shape)` SHALL wrap a buffer whose length matches the shape as a tensor that owns it.

#### Scenario: A kernel writes into a tensor
- **WHEN** a kernel writes a buffer and `NewDeviceTensor` wraps it
- **THEN** `ToHost` of the tensor returns what the kernel wrote
