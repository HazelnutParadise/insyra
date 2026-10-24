# Design: accel-custom-kernels

## Decisions

**1. Reuse the internal kernel layout as the public one.** `KernelConfig{Name, WGSL string; ReadOnly, ReadWrite int}`. Bindings are `0..ReadOnly-1` read-only storage, then `ReadWrite` read-write, then the uniform at `ReadOnly+ReadWrite`. That is the layout `kernel.build` already makes. `NewKernel` refuses an empty name, an empty source or a negative count; how many buffers a kernel may bind is the device's limit, checked at `Submit`, so Vulkan and DirectX 12 callers are not capped at Metal's 8. The source is compiled alone, with none of Insyra's WGSL before it.

**2. Bind once, then one submission for many launches.** `Kernel.Bind` compiles the kernel, checks the buffers and builds the bind group once; the uniform is bound through a dynamic offset into one shared ring of 4096 parameter blocks (256-byte stride or the device's alignment). `accel.Submit` takes the submit lock once, uploads the parameters of all its launches in one write at consecutive blocks, encodes one compute pass per launch (each with its own offset) into one encoder and submits once. A block is rewritten only after the queue was drained: when a Submit would run past the end of the ring, the queue is synchronised and the ring restarts, so no launch can read a block that a later write changes. A Submit holds at most 4096 dispatches. Insyra's own kernels keep their per-dispatch objects through the internal single-launch path. Measured on an M3 with trivial kernels: 168 dispatches in one `Submit` 3.7 ms, as 168 calls 23.8 ms; 28 dispatches 1.05 ms against 4.47 ms (an earlier version that created a uniform and bind group per launch took 19.4 ms and 8.4 ms). The zero-workgroup and device-limit checks run in `Submit`, since the backend would otherwise report them only at `Finish`, without a dispatch number.

**3. The caller's launch is a `Binding`, not a kernel plus buffers.** Doing the buffer checks at `Bind` leaves `Submit` to check only that nothing was released since. A bound buffer released later is an error at `Submit`, never a use of freed memory; releasing the `Binding` defers freeing the bind group until the queue is idle.

**4. Bit-pattern buffers.** `NewBitsBuffer(device, []uint32)` and `BitsToHost()` sit beside the float constructor and `ToHost`; the type is the same. Bits are read back as bytes, never through `float32`, so a NaN payload survives on every platform.

**5. In-place writes wait.** `Write`, `WriteBits` and `Zero` first synchronise the queue, then write. A write that raced a kernel still reading the buffer would hand that kernel a mix of old and new words; waiting is the cheap, safe choice and is documented as a cost.

**6. `Custom` accepts resident tensors; no second operation type.** An earlier prototype added `DeviceOp` and `Tape.Apply`. A review found it a second name for `Tape.Custom`, and its "required CPU half" a nil check that proves nothing. `Custom` keeps its signature. On resident tensors it needs a tape in `PrecisionFloat32`, checks all operands are on one device, adopts the output, and wraps the reverse rule so every gradient it returns is a resident tensor on that device that the tape owns (all are owned first, so a refusal leaves none behind).

**7. `PrecisionFloat32` on a tape.** `accel.PrecisionFloat32` is already the name for "accept whatever the device returns". `SetPrecision` accepts it; `rounded()` is true for it, so every built-in operation behaves as in `PrecisionFloat32Rounded` and results stay bit-identical for them. Only `Custom` on resident tensors requires it. `GetPrecision` reports it, which is the label: a tape that says `PrecisionFloat32` does not promise the same bits on another device.

**8. Lending and adopting.** `Tensor.DeviceBuffer()` returns the buffer of a resident float32 tensor, refused for host and int64 tensors; the tensor keeps owning it. `NewDeviceTensor(shape, buffer)` takes the buffer, checks it is non-nil, on a device, not released, and as long as the shape needs, and leaves it with the caller on an error. `DeviceBuffer.Released()` was added for that check.

## Risks / Trade-offs

- **A caller's kernel can write anywhere in a read-write buffer.** Metal clamps or skips an access past the end; the Vulkan backend (gogpu v0.34.5) does not enable bounds checks or `robustBufferAccess`, so there a stray write can overwrite another resident buffer or lose the device. Read from gogpu and naga source, not run on non-Apple hardware, and documented rather than papered over.
- **Windows stops a dispatch that runs about two seconds.** Documented.
- **Cross-device bits.** Caller kernels are plain `f32` WGSL. Their results may differ between devices, which is why the tape mode says so.
- **Tape is not safe for concurrent use**, as before.

## Verification

- Host tests: refusals of kernel configs, unbindable dispatches, `NewDeviceTensor`, `DeviceBuffer`, `SetPrecision`.
- Hardware (`INSYRA_ACCEL_GPU_TESTS=1`, M3): a single kernel, a chained `Submit`, nothing submitted when dispatch 1 binds a released buffer, aliasing and workgroup refusals, bit round trip with NaN payloads, built-in to `Submit` to built-in with no sync, 400 dispatches past the pending bound, `Write`/`Zero` ordering, and a tape step in `PrecisionFloat32` with a caller kernel between two built-in operations that matches the host oracle bit for bit.
- Not run: any non-Apple device.
