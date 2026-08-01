# Tasks

## 1. The gate
- [x] 1.1 Add a shared helper that decides between skipping and failing, reading the strict-mode variable once
- [x] 1.2 Have it name the missing tool and the verification that did not run, in both modes
- [x] 1.3 Test that the same missing tool skips by default and fails under strict mode

## 2. Route every gate through it
- [x] 2.1 The R and Python cross-language gates in `stats`
- [x] 2.2 The R-only gate used by factor analysis and clustering
- [x] 2.3 The ONNX round trip in `ml`
- [x] 2.4 The scikit-learn decision-tree comparison, which is behind an opt-in flag rather than a tool check — under strict mode it runs, and it passes
- [x] 2.6 Leave `INSYRA_STRICT_FACTOR_R_PARITY` opt-in. Measured: promoting it failed six tests, because that flag guards a comparison known not to pass — ~595 sub-tests differ on three adversarial datasets, traced to gonum's LAPACK port differing from R's by 1 ULP and amplified by ill-conditioning, and documented as mathematically equivalent solutions. Strict mode promotes an opt-in whose reason was a usually-absent tool; that is a different reason. Its toolchain probes still report through the gate.
- [x] 2.5 Confirm by search that no reference-implementation check skips outside the gate — one does, deliberately: the `psych::factor.scores` upstream bug, where R is present and broken rather than missing. A comment at that site says why.

## 3. Continuous integration
- [x] 3.1 Install `scikit-learn` in `clustering-parity.yml`, whose gate requires it and which therefore runs nothing today
- [x] 3.2 Add a workflow that installs R, Python with the scientific stack and scikit-learn, and onnxruntime, and runs the verification set with strict mode on
- [x] 3.4 Exclude, explicitly and with the reason in the file, the three groups the new workflow should not run: the two 25-minute generated corpora already covered by the dedicated parity workflows, and the known-failing factor-analysis comparison. Verified all three steps report 0 skips; the cross-language step reports 8 without the exclusion.
- [x] 3.3 Keep the ordinary test workflow as it is, so the suite still passes on a machine with no toolchains

## 4. Documentation
- [x] 4.1 State the variable and what it does in `ENG.md`'s test-matrix section
- [x] 4.2 Resolve the `AGENTS.md` follow-up that asked for this
- [x] 4.3 No changelog entry: nothing user-visible changes
