# Change: A verification that did not run must not read as one that passed

## Why
This project has now been wrong twice in the same way, and the second time cost real defects.

Cross-language checks against R and Python, and the ONNX round trip, all begin by looking for their reference implementation and calling `t.Skip` when it is absent. In `go test` output a skip is a line that scrolls past; the package still prints `ok`. Nothing distinguishes "this was checked and held" from "this was never checked". Two changes were nearly archived on cross-language tests that had never executed. The ONNX round trip was archived on a test that had never executed anywhere — run for the first time on 2026-08-01 it failed immediately, on two defects that made every exported model unloadable by any runtime.

The workflows meant to close this do not. `clustering-parity.yml` exists for one purpose — run the cross-language clustering parity suite — and installs `numpy scipy statsmodels` for it. The gate those tests pass through requires `scipy, numpy, statsmodels, sklearn`. `scikit-learn` is not a dependency of any of the three, so the gate fails and every test in the dedicated parity workflow skips. The workflow has been reporting green by running nothing. `knn-parity.yml`, two files away, installs `scikit-learn` and works.

Neither the ONNX round trip nor the scikit-learn tree comparison has a workflow at all.

Making the toolchains available is half the fix. The other half is that a machine without them must say so loudly, because the toolchains will go missing again — a runner image changes, a package is renamed, someone runs the suite locally and reports the result.

## What Changes
- Add a strict mode: with `INSYRA_REQUIRE_REFERENCE_TOOLCHAINS=1` set, a check that would skip for a missing reference implementation fails instead, naming what was missing and what went unverified
- Route every reference-toolchain gate through it — the R and Python cross-language gates, the ONNX round trip, and the scikit-learn tree comparison — so no gate can be added later that quietly opts out
- Under strict mode, run the checks that are opt-in *because the tool might be absent* — read narrowly. A flag guarding a comparison known not to pass is a different thing and stays opt-in; the strict R factor-analysis parity suite is that case, and promoting it was measured to turn six tests red over a difference already examined and accepted
- **Install `scikit-learn` in `clustering-parity.yml`**, which cannot currently run the suite it exists to run
- Add a CI job that installs all three toolchains and runs the full verification set with strict mode on, so a skip there fails the run
- Leave the default alone: without the variable, a missing toolchain still skips, and `go test ./...` on a bare machine still passes

## Impact
- Affected specs: `verification-integrity` (new)
- Affected code: `stats/crosslang_helpers_test.go`, `ml/onnx_export_test.go`, `ml/decision_tree_test.go`, `stats/factor_analysis_test.go`, `stats/clustering_test.go`, `.github/workflows/`
- Test and CI only; no library behaviour changes
