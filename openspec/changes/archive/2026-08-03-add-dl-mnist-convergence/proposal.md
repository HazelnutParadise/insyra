# Change: An MNIST classifier trains to convergence in dl

## Why

Every training claim so far is first-step-exact: gradients and one
optimizer step match PyTorch. That proves the mechanism, not the
practice — nothing has ever been trained from initialization to a
working model inside dl. The convergence run is the claim-alignment
milestone (M21): real dataset, real loop, a target accuracy, and a
sane loss curve.

## What Changes

- An IDX reader for the MNIST files (magic, dims, raw bytes — the
  format is trivial and self-contained; test-side code, not public
  API).
- Seeded deterministic initialization helpers for the test (He-style
  scaling from a fixed-seed PRNG), so the run is reproducible without
  adding a public RNG surface prematurely.
- The gated convergence test behind `INSYRA_DL_MNIST_DIR` (the operator
  keeps the four IDX files in `~/.cache/insyra-dl-models/mnist`): a
  784→128→10 MLP with Relu and fused softmax–cross-entropy trains with
  Adam over shuffled minibatches for a bounded number of epochs, then
  SHALL reach ≥95% accuracy on the 10k test set, with the final
  training loss well below the initial loss. Skips cleanly without the
  directory; no network access ever.
- An ungated micro-convergence test: a tiny two-class problem trained
  to 100% in under a second, so the loop mechanics are verified on
  every machine.
- Docs (training section gains the convergence example), changelogs
  both languages, skills — same change.

## Non-Goals

- No CNN convergence run (the measured Conv throughput makes it a
  minutes-long test; the MLP proves the loop and the CNN gradients are
  already PyTorch-exact).
- No dropout/weight-decay/schedules (M22), no public data-loading API,
  no public RNG/init API — those graduate to API only when a design is
  decided.

## Impact

- Affected specs: `dl-training`
- Affected code: dl test files (IDX reader, convergence tests), docs,
  changelogs, skills.
