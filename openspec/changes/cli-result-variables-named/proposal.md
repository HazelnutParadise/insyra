# Proposal: cli-result-variables-named

## Why

Several commands store variables beside the one `as <var>` names, and replace any variable already under those names (#325, api-review CLI-17). Measured on 2026-10-03 with the CLI built from `0.4` at 61e5b0e9: `kmeans … as km` stores `km` and eight `km_*` variables and prints only `stored km (labels)`; `pca`, `dbscan`, `silhouette`, `knn_classify` and `knn_neighbors` likewise print only the main name, and `quant factor` and `quant portfolio` print no name at all although they store `<var>_alpha` and `<var>_stats`. A user who already had a `km_size` loses it without a word.

The finding's second half is already fixed on this line. `cli-env-typed-state` (5671e0e1) stores an `hclust` tree so that `cutree` reads it back after a one-shot restore, and leaves out a variable the environment cannot store, such as a `regression` result, with a `warning:` line naming it; `Docs/cli-dsl.md` already says such a variable is gone after a one-shot command. Measured on 2026-10-03: `insyra hclust dt average as h` then `insyra cutree h k 2` works, and `insyra regression linear y x as r` prints the warning and `insyra show r` reports the variable missing. What the docs do not say is that `show` refuses both kinds, and that nothing but `vars` reads a regression result.

## What Changes

- The success line of `kmeans`, `dbscan`, `silhouette`, `pca`, `knn_classify` and `knn_neighbors` names every variable the command stored, for example `stored km (labels) and km_centers, km_size, …, km_ifault`. `quant factor` and `quant portfolio` end their output with `stored fm and fm_alpha` and `stored w and w_stats`. `corrmatrix` already names `<var>_p`.
- `Docs/cli-dsl.md`'s "Extra Result Variables" table adds `quant factor` and `quant portfolio`, and says that each extra variable replaces one already under that name and that the success line names them. A paragraph says how long an `hclust` tree and a `regression` result live and what reads them: the tree is saved with the environment and read by `cutree`; the regression result lives only for the session and is read by nothing but `vars`; `show` displays neither.
- Both CHANGELOGs, `api-review.md` and `delivery-status.md` are updated. `skills/use-insyra-cli/` already teaches that a regression result is not saved and lists no commands, so it needs no change.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `dsl-commands`: a command that stores more than one variable names each one in its output.

## Impact

- Code: `cli/commands/helpers.go` (`alsoStored`), `cli/commands/clustering.go`, `cli/commands/pca.go`, `cli/commands/knn.go`, `cli/commands/quant.go`, success lines only.
- Tests: `cli/commands/result_variables_test.go`.
- Docs: `Docs/cli-dsl.md`, `CHANGELOG.md`, `CHANGELOG_TW.md`, `api-review.md`, `delivery-status.md`.
- No stored value changes. No library change, no new dependency.
