## Context

Parameter owns host velocity, Adam moments and a uint64 step, or resident Adam moment buffers. Optimizers read Tape.grads. Clear already releases graph history and all gradients. Sequential.NamedParameters provides names; SaveWeights transposes Dense weights into torch layout, unlike optimizer state, which uses Parameter.Value's internal layout.

## Goals / Non-Goals

**Goals:** resume each supported optimizer path exactly across processes; reject invalid imports atomically; keep Clear's signature and default; explicitly freeze built-in updates.

**Non-Goals:** weight loading, RNG/schedule/custom-optimizer persistence, device momentum kernels, concurrency with training, new reset methods or changing released arithmetic.

## Decisions

- Add Tape.SaveOptimizerState(w, named) and Tape.LoadOptimizerState(r, named). Both require a one-to-one nonempty-name mapping covering every tape parameter. Sequential users pass NamedParameters; custom loops choose their own stable names. Nil/closed tapes and invalid mappings return errors.
- The existing SafeTensors reader incorrectly skipped multiplication by zero when counting elements. Match makeLayout by multiplying every dimension, guarding division only for nonzero dimensions. This restores the existing file contract for empty tensors and makes scalar shape descriptors readable; scalar `[]` still contains one element.
- Use the existing SafeTensors codec without changing its public API. Scalars `version` (1) and `precision` (0 exact, 1 rounded, 2 caller-kernel float32) are I64. Each canonical unpadded base64url UTF-8 parameter name prefixes `/shape` (I64 vector), `/dtype` (I64 scalar, 0 means float32), `/step` (I64[2], low then high uint32 limbs), and optional `/adam_m`, `/adam_v`, `/velocity` F32 tensors in internal layout. Even uninitialized parameters have descriptors. Binary F32 payloads preserve signed zero and subnormals. This avoids text floats and a second checkpoint codec.
- Adam moments occur together with a positive step; absence requires step zero. Reject non-finite arrays, negative second moments, malformed limbs, maximal uint64 steps (no representable next step), shape/dtype mismatch and unexpected entries. Inspect wire dtypes before loading to disallow half-precision widening. Precision must match GetPrecision().
- Load parses, validates and allocates/uploads all replacement state before any mutation. On failure release only staged buffers. On success replace all slots, including clearing unused old slots, then release old device moments. Weights, gradients, graph and freeze settings are untouched. Save reads resident moments without moving parameters.
- Add ClearOptions{KeepGradients bool}, SetClearOptions and GetClearOptions. Clear's zero/default behavior stays unchanged. With retention, keep only parameter-gradient map entries and deduplicated ownership from both owned and gradOwned. Release all graph/intermediate resources. A subsequent successful Backward replaces retained gradients. Close forces full cleanup regardless of options.
- Add Parameter.SetFrozen/GetFrozen. SGD, momentum, Adam and AdamW skip frozen parameters before inspecting their gradient or residency. Freeze preserves state and does not block caller-managed Parameter.Update.

## Risks / Trade-offs

- Separate weights/state files can be mismatched → document a caller-managed checkpoint boundary, identical model topology, precision, hyperparameters and next batch.
- Resident allocation can fail during restore → stage every buffer, test budget failures with all old state intact.
- Retained gradients can alias an operation output → preserve ownership from both lists and deduplicate; test device byte counts across cleanup and Close.
- Positional Sequential names can change when layers are inserted → reject incompatibilities detectable from names, shapes and dtype; the caller ensures the model structure and meaning of each parameter still match, even when descriptors are identical.

## Migration Plan

Addition on dev and 0.4, with no rename, removal or changed Clear signature. Rollback removes only the new API and archived capability; existing files remain readable by the weight codec.

## Verification

Public Save/Load is the persistence seam. A test subprocess rebuilds the model, loads independent weight and state streams, continues Adam/AdamW/momentum, and returns byte-compared checkpoints. Both exact and rounded host modes run. Gated real-device Adam/AdamW tests cover restart, host/device transfer, atomic allocation failure and retained-gradient ownership. Invalid files test every parameter, weight, gradient and state slot unchanged. Repeated actual forward/backward/update/Clear cycles compare against a fresh graph reference with identical optimizer history.
