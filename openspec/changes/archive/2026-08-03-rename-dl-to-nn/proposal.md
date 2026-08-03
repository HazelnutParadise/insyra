# Change: The deep-learning package is named nn, before anyone depends on dl

## Why

`dl` collides with the library's own oldest name: `isr/dl.go` exports
`DL` as the DataList sugar entry, every documentation example binds
`dl := insyra.NewDataList(...)`, and a decade of this project's usage
reads "DL" as DataList. A user following the docs shadows the package
with their first variable. The package has never been released, so the
rename is free now and breaking later. `nn` says what the package
actually contains — neural-network machinery (tensors, kernels, tape,
optimizers) — matching torch.nn precedent and pairing with the decided
future `llm` track. Non-neural "deep" architectures would belong to
`ml`/`stats` or their own package regardless.

## What Changes

- Package directory, name, and import path: `dl` → `nn`. Every in-repo
  import, the accel wiring, and test files follow.
- Environment variables (all unreleased): `INSYRA_NN_REAL_MODEL` →
  `INSYRA_NN_REAL_MODEL`, `INSYRA_NN_REAL_MODELS_DIR` →
  `INSYRA_NN_REAL_MODELS_DIR`, `INSYRA_NN_MNIST_DIR` →
  `INSYRA_NN_MNIST_DIR`.
- Docs: `Docs/dl.md` → `Docs/nn.md`, index and cross-references, both
  README package tables, skills. Changelog Unreleased sections retitle
  `### dl` → `### nn` and prose follows — nothing shipped under the old
  name, so no breaking-change note exists to write.
- OpenSpec current specs rename: `dl-inference` → `nn-inference`,
  `dl-training` → `nn-training`. Archives stay as history.
- `ENG.md` architecture diagram and `delivery-status.md` current-state
  prose follow; decision-log history keeps its original wording.

## Non-Goals

- No behavioral change of any kind; every test must pass unmodified in
  meaning.

## Impact

- Affected specs: `nn-inference` (rename marker)
- Affected code: package tree rename plus references.
