# Spec Delta

## MODIFIED Requirements

### Requirement: A malformed custom operation is an error, never a silent gradient

`Tape.Custom` SHALL refuse an empty name, a nil `vjp`, a nil or non-float32 `output` or input, an `output` that is also one of its inputs, an `output` that an operation already recorded on this tape produced, and an `output` that such an operation already read as an input, and SHALL record nothing when it refuses. During the reverse pass, a custom `vjp` that returns an error, the wrong number of gradients, a non-float32 gradient, or a gradient shaped unlike its input SHALL fail the pass with an error naming the operation.

#### Scenario: A gradient of the wrong shape
- **WHEN** a custom `vjp` returns a gradient shaped `[2]` for an input shaped `[3]`
- **THEN** `Backward` returns an error naming the operation and the two shapes

#### Scenario: A refused declaration
- **WHEN** `Custom` is called with a nil `vjp`
- **THEN** it returns an error and a later `Backward` behaves as if the call had not been made

#### Scenario: An output recorded twice
- **WHEN** `y` comes from `Tape.Mul(w, x)` and `Custom` is then called with `y` as its output
- **THEN** `Custom` returns an error, and the gradient of `w` is the one `Mul` alone gives, not twice it

#### Scenario: An output an earlier operation read
- **WHEN** `b` was computed outside the tape, `Tape.Mul(b, c)` read it, and `Custom` is then called with `b` as its output
- **THEN** `Custom` returns an error naming `Mul`, because the reverse pass would visit the custom operation before `Mul` and `b`'s gradient from `Mul` would never reach the custom operation's inputs
