## ADDED Requirements

### Requirement: A caller's kernel runs on resident buffers
`accel.NewKernel` SHALL accept a WGSL compute shader whose bindings are `Reads` read-only storage buffers, then `Writes` read-write storage buffers, then one 64-byte uniform of sixteen `u32`. `accel.Run` SHALL bind the given `DeviceBuffer`s and uniform values in that order and dispatch the given workgroup counts on the resident device.

#### Scenario: A kernel writes a resident buffer
- **WHEN** a kernel that doubles each element of its read buffer into its write buffer runs on two resident buffers
- **THEN** `ToHost` of the write buffer returns the doubled values and the read buffer is unchanged

### Requirement: Dispatches in one Run are ordered in one submission
`accel.Run` SHALL encode all of its dispatches into one submission in the order given, so that each dispatch sees every write of the dispatches before it. Insyra's own resident kernels SHALL go through the same submission path.

#### Scenario: A chained Run
- **WHEN** one `Run` holds a dispatch that writes buffer B from A, then a dispatch that writes C from B
- **THEN** C holds the result of applying both kernels to A

### Requirement: A Run that cannot be bound submits nothing
`accel.Run` SHALL return an error and submit no dispatch when any dispatch has the wrong number of buffers for its kernel, a released buffer, buffers on different devices, or buffers on the host. `accel.NewKernel` SHALL refuse an empty name, an empty source, or more storage buffers than the device allows per shader stage.

#### Scenario: A released buffer
- **WHEN** the second of two dispatches binds a buffer that was released
- **THEN** `Run` returns an error naming that dispatch and the first dispatch has not run

#### Scenario: A host buffer
- **WHEN** a dispatch binds a buffer of the zero Device
- **THEN** `Run` returns an error saying the buffer is on the host

### Requirement: A buffer can hold bit patterns
`accel.NewDeviceBufferBits` SHALL place a `[]uint32` on a device without reinterpreting it, and `DeviceBuffer.BitsToHost` SHALL return the stored bits. Such a buffer SHALL count against the device memory budget like any other.

#### Scenario: Bits round-trip
- **WHEN** the values 0, 1, 0xFFFFFFFF and the bits of a float32 NaN are placed with `NewDeviceBufferBits` and read with `BitsToHost`
- **THEN** every value comes back with the same bits

### Requirement: A kernel can use the exact-arithmetic library
A `KernelSpec` with `Library` set SHALL have Insyra's exact-sum and IEEE WGSL library prepended to its source, so that a caller's kernel can form correctly rounded sums with the register Insyra's own kernels use.

#### Scenario: A caller's exact dot product
- **WHEN** a caller's `Library` kernel computes a dot product with `exact_add_product` and `exact_round`
- **THEN** its result has the same bits as `RoundedMatMul` on the same inputs
