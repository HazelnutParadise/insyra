## Why

Custom training loops cannot resume the built-in optimizers after a process restart because their state is private. Issue #376 also needs graph cleanup with a caller-selected gradient policy and frozen parameters that do not receive stale momentum or weight decay.

## What Changes

- Add named, versioned optimizer-state Save/Load methods on Tape, using binary SafeTensors values and transactional restoration on host or device.
- Keep Clear's existing signature and default behavior; add ClearOptions configured by SetClearOptions to retain parameter gradients when requested.
- Add parameter freezing for built-in optimizers, preserving state for unfreezing.
- Fix SafeTensors element counting for zero dimensions, required by scalar parameter shape descriptors and empty parameters.
- Add subprocess continuation tests, failure atomicity tests, device lifecycle tests, documentation and bilingual changelog entries on dev and 0.4.

## Capabilities

### New Capabilities
- `nn-optimizer-state`: bit-exact named optimizer persistence and explicit cleanup/freeze policy for custom training loops.

### Modified Capabilities

None. Existing Clear cleanup remains the default.

## Impact

`nn` only, including the existing SafeTensors reader, plus Docs/nn.md, changelogs and delivery-status.md. No new dependencies, renamed APIs, kernels, weight format changes or second reset method. Device momentum remains unsupported; the existing device Adam/AdamW states round-trip. RNG, schedules, hyperparameters, weights and gradients remain caller-managed.
