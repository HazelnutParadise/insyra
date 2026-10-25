# accel-custom-kernels Specification

## Purpose
Lets a program run its own WGSL compute kernels on the resident device buffers Insyra holds, in one ordered queue shared with Insyra's own kernels, so a step that mixes caller and built-in GPU work never copies data back to the host.
## Requirements
### Requirement: A caller's kernel runs on resident buffers
`accel.NewKernel` SHALL accept a WGSL compute shader whose bindings are `ReadOnly` read-only storage buffers, then `ReadWrite` read-write storage buffers, then one 64-byte uniform of sixteen `u32`. `Kernel.Bind` SHALL tie the kernel to the given `DeviceBuffer`s in that order, and `accel.Submit` SHALL launch a binding with the given uniform values and workgroup counts on the resident device.

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

### Requirement: A Bind or Submit that cannot run does nothing
`Kernel.Bind` SHALL return an error when given the wrong number of buffers, a nil or released buffer, a buffer on the host or on another device, one buffer both read and written or written twice, or more storage buffers than the device allows. `accel.Submit` SHALL return an error naming the dispatch and submit no work when a dispatch has a released binding or a binding whose buffer was released, or a workgroup count of zero or above the device's limit.

#### Scenario: A released buffer
- **WHEN** the second of two dispatches uses a binding whose buffer was released
- **THEN** `Submit` returns an error naming dispatch 1 and the first dispatch has not run

#### Scenario: A buffer read and written
- **WHEN** `Bind` is given one buffer as both a read and a write buffer
- **THEN** it returns an error and no binding

### Requirement: A kernel config is checked at creation
`accel.NewKernel` SHALL refuse an empty name, an empty source and a negative buffer count. A shader that does not compile SHALL fail `Bind` with `ErrShaderCompile`.

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

### Requirement: Launches reuse their bind group and read their own parameters
A `Binding` SHALL be built once and reused by every `Submit`, and the parameters of each dispatch in a `Submit` SHALL be read from their own block of a shared ring, so two launches of one binding with different parameters in one `Submit` each see their own. A block SHALL be rewritten only after the queue has been drained.

#### Scenario: One binding, four parameter sets
- **WHEN** one binding is launched four times in one `Submit` with different element counts
- **THEN** each element is incremented by exactly the launches whose count covers it

#### Scenario: More launches than the ring holds
- **WHEN** successive Submits together launch more dispatches than the ring has blocks
- **THEN** every launch still ran with its own parameters and the final buffer is correct

