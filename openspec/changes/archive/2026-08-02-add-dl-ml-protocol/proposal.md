# Change: `dl` models join the `ml` protocol, and `ml`'s own exports read back

## Why
A loaded network that only speaks tensors is a separate world from the rest of the library. `ml` established the contract — `Features()` plus name-bound `Predict` over a `DataTable` — and everything downstream (`Score`, `CrossValidate`, pipelines, conformance) works against that contract, not against any particular model kind. Binding `dl` models into it makes a neural network a peer of a random forest: same scoring, same metrics, same misuse refusals.

The second half closes a loop the library has promised implicitly since `ml` gained export: what insyra writes, insyra can read. `ml` exports eleven model families as ONNX using the `ai.onnx.ml` operator domain — `LinearRegressor`, `LinearClassifier`, `TreeEnsembleRegressor`, `TreeEnsembleClassifier`, and the preprocessing operators pipelines carry. Implementing that domain in `dl` gives every exported model a pure-Go execution path, and gives verification a double reference: `dl`'s output must match both `onnxruntime` *and* the original `ml` model's own predictions.

Go's structural interfaces keep the dependency direction clean: `dl` satisfies `ml.Model` without importing `ml` — the adapter needs only `insyra` for `DataTable`/`DataList`, and the conformance check runs from `dl`'s tests via `ml/mltest`.

## What Changes
- `dl.BindRegressor(model, inputName, features)` and `dl.BindClassifier(model, inputName, features, classes)`: adapters that satisfy the `ml.Model` (and `Classifier`/`ProbaModel`) contracts structurally — name-bound columns, missing features refused by name, extra columns ignored, probability rows from the network's output
- The `ai.onnx.ml` operator domain in `dl`'s interpreter: `LinearRegressor`, `LinearClassifier`, `TreeEnsembleRegressor`, `TreeEnsembleClassifier`, `Scaler`, plus the standard-domain operators `ml`'s pipeline exports use (`Concat`, `Unsqueeze`, `Gather`, `OneHotEncoder`, `LabelEncoder`, `Cast` extensions as needed — enumerate from `ml`'s exporter, not from guesswork)
- Round-trip closure: every model family `ml` exports loads and runs in `dl`, outputs matching both `onnxruntime` and the original fitted model's own `Predict`/`PredictProba`
- `mltest.RunConformance` passes for both adapters

## Impact
- Affected specs: `dl-inference`
- Affected code: `dl/` (adapters + ai.onnx.ml kernels), docs, changelogs, skills
- Blocked by: `add-dl-onnx-mlp-inference` (tensor, decoder, interpreter, harness)
- Additive; `ml` itself is untouched
