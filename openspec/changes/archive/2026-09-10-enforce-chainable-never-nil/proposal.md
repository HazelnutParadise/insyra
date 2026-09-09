# Proposal: enforce-chainable-never-nil

## Why

"A chainable method never returns nil" is the rule that keeps `dl.MovingAverage(0).Sort()` from panicking, and it is written down in `AGENTS.md`, in `Docs/DataList.md`, in `Docs/DataTable.md` and in the `chainable-never-nil` spec. Nothing checks it.

That is not hypothetical. Batch 1 fixed ten `DataList` transforms that returned nil; the `isr` overrides were added later when a chain stopped compiling; `Diff` and `PctChange` were found only on a second pass. Each was caught by a person reading code. A new method that returns nil on a bad argument would compile, pass every test, and be found by whoever's chain panicked in production — which is exactly the failure the rule exists to prevent, and the nil is especially unforgiving here because a nil `*DataList` panics on every method it has, `Err()` included.

A scan of the repository today finds 182 methods bound by the rule and none breaking it. That number is worth pinning while it is still zero.

## What Changes

- A test walks the module's own source and fails on any method that returns its receiver's type with no `error` result — the fluent shape, where `Err()` is the only channel — and returns a literal `nil` in that position.
- The test refuses to pass vacuously: it asserts that it parsed a plausible number of files and found a plausible number of methods, so a broken walk or a broken matcher fails loudly instead of reporting success on an empty set.
- The `chainable-never-nil` spec records that the rule is enforced mechanically, and over the whole module rather than `DataList` and `isr` alone.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `chainable-never-nil`: the existing requirements are joined by one saying the rule is checked by a test across the module.

## Impact

- `chainable_contract_test.go` (new). No production code, nothing user-visible, no changelog entry.
- The check is static: it sees a literal `return nil`, not a nil-valued variable. That is the shape every instance of this defect has taken so far, and a type-checking pass would cost far more than it catches.
