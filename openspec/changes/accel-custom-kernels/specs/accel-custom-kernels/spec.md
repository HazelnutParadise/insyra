## ADDED Requirements

### Requirement: A caller's kernel runs on resident buffers
`accel.NewKernel` SHALL accept a WGSL compute shader whose bindings are `ReadOnly` read-only storage buffers, then `ReadWrite` read-write storage buffers, then one 64-byte uniform of sixteen `u32`. `accel.Submit` SHALL bind the given `DeviceBuffer`s and uniform values in that order and dispatch the given workgroup counts on the resident device.

#### Scenario: A kernel writes a resident buffer
- **WHEN** a kernel that doubles each element of its read buffer into its write buffer runs on two resident buffers
- **THEN** `ToHost` of the write buffer returns the doubled values and the read buffer is unchanged

### Requirement: Dispatches in one Submit are ordered in one submission
`accel.Submit` SHALL encode all of its dispatches into one submission in the order given, so that each dispatch sees every write of the dispatches before it. Insyra's own resident kernels SHALL go through the same submission path and queue.

#### Scenario: A chained Submit
- **WHEN** one `Submit` holds a dispatch that writes buffer B from A, then a dispatch that writes C from B
- **THEN** C holds the result of applying both kernels to A

#### Scenario: Built-in and caller kernels interleave
- **WHEN** a built-in resident matrix product feeds a caller's `Submit`, whose output feeds another built-in product, with no synchronisation between them
- **THEN** the final result is correct

### Requirement: A Submit that cannot be bound submits nothing
`accel.Submit` SHALL return an error naming the dispatch and submit no work when a dispatch has no kernel, the wrong number of buffers, a nil or released buffer, a buffer on the host or on another device, one buffer both read and written or written twice, or a workgroup count of zero or above the device's limit.

#### Scenario: A released buffer
- **WHEN** the second of two dispatches binds a buffer that was released
- **THEN** `Submit` returns an error naming dispatch 1 and the first dispatch has not run

#### Scenario: A buffer read and written
- **WHEN** a dispatch binds one buffer as both a read and a write buffer
- **THEN** `Submit` returns an error naming that dispatch and the buffer is unchanged

### Requirement: A kernel config is checked at creation
`accel.NewKernel` SHALL refuse an empty name, an empty source and a negative buffer count. A shader that does not compile SHALL fail the first `Submit` that uses it with `ErrShaderCompile`.

#### Scenario: Bad configs
- **WHEN** `NewKernel` receives no name, or a negative count
- **THEN** it returns an error and no kernel

### Requirement: A buffer can hold bit patterns and be rewritten in place
`accel.NewBitsBuffer` SHALL place a `[]uint32` on a device without reinterpreting it and `BitsToHost` SHALL return the stored bits, including NaN payloads. `Write`, `WriteBits` and `Zero` SHALL overwrite a whole buffer after waiting for every earlier submission, and SHALL refuse a length that differs from the buffer's.

#### Scenario: Bits round-trip
- **WHEN** 0, 1, 0xFFFFFFFF and NaN bit patterns are placed with `NewBitsBuffer` and read with `BitsToHost`
- **THEN** every value comes back with the same bits

#### Scenario: A write after a kernel
- **WHEN** a kernel reads a buffer, then `Write` replaces the buffer, then the kernel runs again
- **THEN** the first run saw the old contents and the second the new

### Requirement: The device's limits are reported
`accel.ResidentLimits` SHALL report the resident device's maximum storage buffers per kernel, buffer size in bytes, workgroups per dimension and invocations per workgroup.

#### Scenario: Limits are usable
- **WHEN** a resident device exists
- **THEN** `ResidentLimits` returns positive values for all four
