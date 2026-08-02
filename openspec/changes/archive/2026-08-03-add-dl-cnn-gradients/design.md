## Context

`dl` already records inference kernels as plain-function VJPs for the MLP and
attention families. The CNN inference kernels in `dl/kernels.go` already define
the required NCHW indexing, padding, grouping, pooling denominators, max ties,
and inference-mode BatchNormalization semantics. This change extends the tape
without changing those forward kernels or the ONNX graph runner.

## Goals / Non-Goals

**Goals:**

- Add direct-loop VJPs for Conv, MaxPool, AveragePool, GlobalAveragePool, and
  inference-mode BatchNormalization.
- Keep accumulation in `float64` until gradients are narrowed to the tape's
  float32 tensor representation.
- Prove every VJP with ungated central finite differences and prove one CNN
  Adam step against a deterministic PyTorch/SafeTensors fixture.

**Non-Goals:**

- Training-mode BatchNormalization, dropout, device training, multi-step
  optimisation, and a Pad VJP when the CNN fixture does not use Pad.
- Rewriting the tape, changing inference behavior, or introducing a separate
  graph transformation system.

## Decisions

- **Mirror forward index arithmetic in direct loops.** Conv backward scatters
  each upstream output into input and weight gradients and sums bias gradients.
  This keeps explicit and automatic padding, strides, dilations, and groups in
  one semantic path. An im2col rewrite would duplicate the boundary rules and
  make parity failures harder to localize.
- **Recompute pooling windows during backward.** MaxPool scans kernel rows and
  columns in the same order as forward, so strict `>` preserves the first
  maximum. AveragePool recomputes the valid-element count for each output and
  applies `count_include_pad` to the denominator exactly as forward does.
  Recording argmax indices would reduce recomputation but would add tape state
  without improving the required proof.
- **Treat BatchNormalization running statistics as constants.** The tape
  records only input, scale, and bias as differentiable inputs. The VJP uses
  the loaded mean and variance and refuses no new training mode because the
  existing forward API is inference-only; any future training-statistics API
  must name and implement that separate operation explicitly.
- **Use the established reference-gate shape.** The Python fixture captures
  stdout and stderr separately, emits JSON only on stdout, saves deterministic
  F32 parameters plus running statistics in SafeTensors, and the Go test
  replays the same tensor graph before comparing loss, gradients, and updated
  parameters within the existing f32 tolerance.

## Risks / Trade-offs

- [Risk] Large grouped or dilated convolutions make the direct backward loops
  expensive. → [Mitigation] The path is correctness-first and follows the
  existing CPU kernel; performance and device training remain out of scope.
- [Risk] MaxPool is nondifferentiable at ties. → [Mitigation] The VJP matches
  forward's first-maximum rule and has a dedicated tie regression test.
- [Risk] PyTorch and Go can differ in floating-point reduction order. →
  [Mitigation] Accumulate local Conv and BatchNormalization reductions in
  float64, narrow once, and verify against both finite differences and the
  fixed one-step reference.

## Migration Plan

No migration is required. Existing inference callers continue using the plain
forward kernels. Training callers opt into the new Tape wrappers. Rollback is
limited to removing the new wrappers, tests, fixture, and synchronized docs.

## Open Questions

None for this change. A future training-mode BatchNormalization design must be
proposed separately rather than inferred from the inference-mode VJP.
