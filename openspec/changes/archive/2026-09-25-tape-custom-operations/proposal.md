# Proposal: tape-custom-operations

## Why

#375: a Go package outside `nn` cannot put its own operation on a `Tape`. `tapeOp` and `Tape.record` are private, so a value computed by an external kernel is detached from the graph and its inputs silently receive a zero gradient. The reporter's reproduction computes `y = w·x` outside the tape and gets `dw = 0` where the built-in `Mul` gives `24`. CoImNet works around it with two separate tapes and a hand-written backward pass for its recurrent core.

It is also the first step of #379: the sparse edge-sum operation that issue needs is a custom operation with a reverse rule of its own, so it needs this hook before it can join a tape.

PyTorch answers the same need with `torch.autograd.Function`, and TensorFlow with `tf.custom_gradient`: the caller supplies the forward result and the reverse rule, and the framework treats it like any built-in operation.

## What Changes

- `Tape.Custom(name, inputs, output, vjp)` records an operation computed outside the tape. `vjp` receives the upstream gradient, shaped like `output`, and returns one gradient per input, or nil for an input that receives none. `Backward` treats it exactly like a built-in operation.
- The declaration is checked when it is recorded: a non-empty name, a non-nil `vjp`, non-nil float32 `output` and inputs, and an `output` that is not one of its own inputs.
- What a custom `vjp` returns is checked during `Backward`: one entry per input, each float32 and shaped like its input. A `vjp` that returns an error, or a result that fails the check, fails `Backward` with an error naming the operation.
- `Tape.BackwardFrom(output, upstream)` starts the reverse pass from any tensor an operation on this tape produced, seeded with an explicit upstream gradient of `output`'s shape. `Backward(loss)` becomes `BackwardFrom(loss, 1)`.
- A `Backward` or `BackwardFrom` that fails leaves the gradients where the last successful pass left them, for `Tape.Grad` and `Parameter.Grad` alike. Today `Tape.Grad` can return a half-built gradient after a failure.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `nn-training`: the tape accepts external operations and an explicit upstream gradient.

## Impact

- `nn/autodiff.go` and new tests. Additive: no existing signature or successful result changes.
- `Docs/nn.md`, `skills/insyra/SKILL.md`, both CHANGELOGs under `` ### `nn` ``.
- Unblocks #379's sparse edge-sum operation.
