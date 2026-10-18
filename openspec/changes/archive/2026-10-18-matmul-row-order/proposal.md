# Proposal: matmul-row-order

## Why

The 2-D `MatMul` kernel keeps one running sum per output element and walks `b` down a column. Each multiply-add waits for the previous addition, and every read of `b` jumps `n` floats ahead. Measured on 2026-10-18 on a 4-vCPU Intel Xeon (Linux amd64, Go 1.26.9), best of the runs below, `BenchmarkMatMul2D`:

| shape m×k×n | before | after |
| --- | --- | --- |
| 1×784×28 | 23.0 µs | 19.5 µs |
| 2×24×11 | 929 ns | 601 ns |
| 64×784×28 | 694 µs | 331 µs |
| 256×256×256 | 5.32 ms | 1.81 ms |

Reported in [#463](https://github.com/HazelnutParadise/insyra/issues/463) from CoImNet, where this kernel is the largest remaining cost of single-sample inference and training.

## What Changes

- `matMul2DWithWorkers` walks each output row once per inner index, adding the products of one row of `b` into that row's running sums. Each output element still starts at zero and adds its products in inner-index order, so every result that is not NaN is bit-identical to what the kernel returned before. The parallel split by rows is unchanged.
- Which NaN comes out when two NaNs meet in one addition may differ. Go does not specify it, and the compiler orders the operands of the addition either way: on amd64 the old kernel kept the running sum's NaN and the new one keeps the product's.
- The batched path, the device path and every VJP are unchanged.

## Capabilities

### Modified Capabilities

- `nn-inference`: the 2-D MatMul states that it keeps each output's dot-product order.

## Impact

`nn/kernels.go`, the new `nn/matmul_order_test.go`, both CHANGELOGs.
