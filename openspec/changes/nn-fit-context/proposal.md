# Proposal: nn-fit-context

## Why

NN-2 of the API review ([#266](https://github.com/HazelnutParadise/insyra/issues/266)): `Sequential.Fit(x, y, cfg)` takes no `context.Context`. An epoch on a real dataset runs for minutes, `FitConfig.Progress` reports each one, and nothing can stop the run short of ending the process. A server cannot bound a training request, and a caller watching the validation loss cannot stop when it stops improving.

The repository's pattern for work that should be cancellable is a second method named with `Context` that takes the context first, the plain name calling it with `context.Background()`: `ReadSQLContext`, `ToSQLContext`, the `datafetch` fetchers. A `FitConfig.Context` field would be the only context in the library carried inside a settings struct, and the Go documentation for `context` asks that it be passed explicitly as the first parameter rather than stored in a struct.

The other half of NN-2, letting callers implement `LossSpec` and `OptimizerSpec`, decides the package's public extension surface and is left to the owner on #266.

## What Changes

- New `(*Sequential).FitContext(ctx context.Context, x, y *Tensor, cfg FitConfig) (*FitResult, error)`. `Fit` calls it with `context.Background()` and is otherwise unchanged.
- The context is checked before every batch and before each epoch's validation. When it is done, `FitContext` returns `ctx.Err()` together with a `FitResult` that lists only the epochs that finished, validation included, and its `Elapsed`. The model keeps every optimizer step taken before the check, including the steps of an epoch that did not finish, so a caller can stop from inside `Progress` and keep the model as it stood after that epoch.
- A context that is already done returns before the first batch, with an empty result and the model and its tape untouched. A nil context is an error, reported before anything changes.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `nn-training-frontdoor`: adds "Training can be cancelled between batches".

## Impact

- `nn/fit.go` and the `nn` tests.
- `Docs/nn.md`, `skills/insyra/SKILL.md` (which names the kinds of work a context can cancel), both changelogs, `api-review.md` (NN-2, partly), `delivery-status.md`.
