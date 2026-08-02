# `dlbridge` Package

`accel/dlbridge` is the opt-in connection between `dl` inference and the
device MatMul implementation. Import it for its side effect:

```go
import _ "github.com/HazelnutParadise/insyra/accel/dlbridge"
```

Without this import, `dl` has no accelerator dependency and every MatMul uses
its existing CPU implementation. With it, only 2-D float32 MatMuls at or above
the measured 16Mi multiply-accumulate floor try the device. Batched products
and smaller products stay on the CPU because the measured prototype did not
make those shapes profitable.

The device kernel keeps each output's accumulation serial along `k`. On
hardware where parity has been verified, the device result is bit-identical to
the CPU result. A missing device, device error, or unsupported result falls
back to the CPU without changing the answer. The shared `accel.Default()`
report records whether the bridge accelerated the call and the fallback reason
when it did not.

The floor is provisional until the operator runs the hardware ladder:

```bash
INSYRA_ACCEL_GPU_TESTS=1 \
GOCACHE=/private/tmp/insyra-gocache GOTMPDIR=/private/tmp \
go test -run TestDLMatMulDeviceFloorLadder -v ./accel/internal/wgpu
```

Run every `dl` test with the bridge blank-imported by enabling the test-only
build tag:

```bash
INSYRA_ACCEL_GPU_TESTS=1 \
GOCACHE=/private/tmp/insyra-gocache GOTMPDIR=/private/tmp \
go test -tags dlbridge ./dl/...
```

The bridge is deliberately not included in `allpkgs`; it must be blank-imported
when a program explicitly opts into device MatMul for `dl`.
