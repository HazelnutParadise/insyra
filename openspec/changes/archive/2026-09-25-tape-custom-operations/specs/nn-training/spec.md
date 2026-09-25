## ADDED Requirements

### Requirement: An operation computed outside the tape can join it

`Tape.Custom(name, inputs, output, vjp)` SHALL record an operation whose forward result was computed by the caller. During the reverse pass the tape SHALL call `vjp` with the upstream gradient of `output` and SHALL accumulate the gradients it returns into the inputs exactly as it does for a built-in operation. A nil entry SHALL mean that input receives no gradient from this operation. One call SHALL record one operation, whatever the operation computes internally.

#### Scenario: A custom product matches the built-in one
- **WHEN** `y = w·x` is computed outside the tape and recorded with `Custom` and the product rule, then fed to `MSELoss` against zero with `w = 3`, `x = 2`
- **THEN** the gradient of `w` is `24`, bit-identical to the same graph built with `Tape.Mul`

#### Scenario: A custom operation between built-in ones
- **WHEN** a custom operation sits between built-in operations, such as `Tanh(Custom(...))` followed by a loss
- **THEN** every parameter gradient is bit-identical to the same graph built from built-in operations only

#### Scenario: A reverse rule that is not the forward derivative
- **WHEN** the forward pass is a hard step and the declared `vjp` returns a smooth surrogate gradient
- **THEN** the tape propagates the surrogate gradient, not the zero derivative of the step

### Requirement: A malformed custom operation is an error, never a silent gradient

`Tape.Custom` SHALL refuse an empty name, a nil `vjp`, a nil or non-float32 `output` or input, and an `output` that is also one of its inputs, and SHALL record nothing when it refuses. During the reverse pass, a custom `vjp` that returns an error, the wrong number of gradients, a non-float32 gradient, or a gradient shaped unlike its input SHALL fail the pass with an error naming the operation.

#### Scenario: A gradient of the wrong shape
- **WHEN** a custom `vjp` returns a gradient shaped `[2]` for an input shaped `[3]`
- **THEN** `Backward` returns an error naming the operation and the two shapes

#### Scenario: A refused declaration
- **WHEN** `Custom` is called with a nil `vjp`
- **THEN** it returns an error and a later `Backward` behaves as if the call had not been made

### Requirement: A failed reverse pass publishes no gradient

A `Backward` or `BackwardFrom` that returns an error SHALL leave every gradient readable through `Tape.Grad` and `Parameter.Grad` as the last successful pass left it.

#### Scenario: A failure after a success
- **WHEN** one `Backward` succeeds and the next fails in a custom `vjp`
- **THEN** `Tape.Grad` and `Parameter.Grad` return the gradients of the successful pass

### Requirement: The reverse pass can start from an explicit upstream gradient

`Tape.BackwardFrom(output, upstream)` SHALL run the reverse pass from `output` seeded with `upstream`. `output` SHALL have been produced by an operation recorded on this tape, and `upstream` SHALL be float32 and shaped like `output`; otherwise it SHALL return an error. For a loss produced by an operation on this tape, `Backward(loss)` SHALL give the same result as `BackwardFrom(loss, 1)`; `Backward` keeps returning zero gradients for a loss the tape did not produce.

#### Scenario: A non-scalar output
- **WHEN** `y = MatMul(x, W)` is recorded and `BackwardFrom(y, g)` is called with `g` shaped like `y`
- **THEN** the gradient of `W` is bit-identical to `MatMul(xᵀ, g)`

#### Scenario: A scalar loss
- **WHEN** the same graph is run once with `Backward(loss)` and once with `BackwardFrom(loss, 1)`
- **THEN** every parameter gradient is bit-identical between the two

#### Scenario: An output the tape did not produce
- **WHEN** `BackwardFrom` is given a tensor no recorded operation produced
- **THEN** it returns an error instead of zero gradients
