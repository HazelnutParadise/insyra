# Proposal: ml-classes-never-nil

## Why

Ten methods in `ml` and `nn` have the signature `Classes() *insyra.DataList` and return `nil` when the receiver is nil, the model is not fitted, or a pipeline wraps something that is not a classifier. A nil `*insyra.DataList` panics on every method it has — including `Err()`, the accessor whose whole job is to say what went wrong. So the caller's first safe move is impossible: asking the value what happened is itself the crash.

These ten are the only exported methods in `ml` and `nn` that return an `*insyra.DataList` or `*insyra.DataTable` with no `error` beside it. Every other getter in those packages returns a slice, where a nil is harmless. The library's own rule — a method that returns one of these types without an error channel returns something usable and records the failure on it — holds in 182 methods with no exceptions. These ten were written before that rule was stated and never brought under it.

The library already decided not to let a nil receiver crash: all ten catch it and return nil rather than panicking. This change does not revisit that decision; it fixes the type that decision returns.

Closes the `Classes()` half of the nil-return audit.

## What Changes

- **The ten `Classes()` methods return an empty, usable `*insyra.DataList`** carrying the reason on its `Err()`, instead of `nil`. The signature and the `Classifier` interface are unchanged.
- **No nil guard is removed.** The change was expected to leave three of them unable to fire; the test suite showed that none of the three is reached only from `Classes()`. `ml/helpers.go`'s guard covers `PCATransformer.Transform`, which passes nil on purpose because components are not classes, and three callers that pass the raw `m.classes` field rather than the method. The two in `ml/model_selection.go` receive the result of an arbitrary `ProbaModel`'s `Classes()`, which a third party may still implement as nil.
- **The redundant `isNilPointer(classes)` half of those two guards is dropped**: the parameter is a concrete `*insyra.DataList`, not an interface, so a typed nil cannot hide inside it and `!= nil` is exact. `isNilPointer` itself and its 32 other uses, which do guard interface-typed values, are untouched.
- **`ml/mltest` gains a conformance check** that `Classes()` is not nil. That package exists to hold third-party implementations to the protocol, and it currently calls `classes.Len()` with no guard — so a third-party model returning nil crashes the suite instead of being told what is wrong.
- **Docs and the agent skill are corrected.** `Docs/ml.md` does not document `Classes()` at all; the skill's example calls it and uses the result without checking anything.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `chainable-never-nil`: the rule gains the `ml`/`nn` getters that return `*DataList` without an error channel.

## Impact

- `ml/decision_tree.go`, `ml/gradient_boosting.go`, `ml/models.go`, `ml/pipeline.go`, `ml/random_forest.go`, `nn/protocol.go`; `ml/helpers.go` (the shared `noClasses` helper), `ml/model_selection.go`, `ml/mltest/conformance.go`.
- `Docs/ml.md`, `skills/insyra/SKILL.md`, both changelogs.
- **BREAKING in behaviour despite the compatible signature.** Code that reads `if classes == nil` compiles unchanged and its branch never runs again, so an unfitted model stops being noticed. The changelog must name the replacement — `classes.Err() != nil` — rather than only saying nil is gone.
- `nn`'s single method is included for consistency, not as a fix: `BindClassifier` rejects empty classes at construction, so its nil branch is not reachable through the public API.
