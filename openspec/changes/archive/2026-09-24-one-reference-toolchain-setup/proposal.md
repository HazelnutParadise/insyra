## Why

`clustering-parity.yml`, `knn-parity.yml` and `reference-verification.yml` each carried their own copy of the same steps: Python 3.12, R release, the same four Python packages and the same three R packages (#280, RP-7). Adding a package or moving a version meant finding all three, and the history of these workflows shows what happens when one is missed: until 2026-08-01 the clustering workflow lacked scikit-learn, its gate skipped, and it reported green having run nothing.

## What Changes

- A composite action, `.github/actions/setup-reference-toolchains`, sets up Python 3.12 and R release and installs the shared Python and R packages. An input carries any Python packages a workflow needs beyond the shared set.
- The three workflows use it. `reference-verification.yml` passes its extra packages (`onnx onnxruntime safetensors cvxpy`) and keeps its PyTorch step, which only it needs.
- No test, gate or package list changes, so what each workflow installs is exactly what it installed before.

## Capabilities

### New Capabilities
None.

### Modified Capabilities
- `ci-workflows`: gains the requirement that the reference toolchains are set up in one place.

## Impact

- `.github/actions/setup-reference-toolchains/action.yml` (new); the three workflows.
- `api-review.md` RP-7, `delivery-status.md`, issue #280.
