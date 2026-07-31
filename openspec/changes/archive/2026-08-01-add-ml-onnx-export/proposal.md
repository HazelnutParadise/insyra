# Change: Export fitted models as ONNX

## Why
"Connect with the machine-learning ecosystem" has one concrete meaning, established by checking what exists: there is no living Go classical-machine-learning library to connect to. Gorgonia last released in December 2023, GoLearn has never cut a tagged release, onnx-go is abandoned, and gota — the DataFrame layer everything else needed — is archived outright.

What does exist is ONNX, and its `ai.onnx.ml` domain describes exactly the models this package fits: `LinearRegressor`, `LinearClassifier`, `TreeEnsembleRegressor`, `TreeEnsembleClassifier`, `Scaler`, `LabelEncoder`, `OneHotEncoder`, `Imputer`.

Export is cheap and import is not. An `.onnx` file is protobuf, so writing one needs no C dependency and no runtime — it is a serialiser over parameters that are already fitted, not new numerics. And it is asymmetric in value: a model exported from here is immediately readable by Python's onnxruntime, by Netron, by Triton, by C# and by the browser. Import is a second project and is deliberately not in this change.

It also buys the test that makes every numeric claim in the package checkable rather than asserted: fit here, export, score in onnxruntime, get the same answer.

## What Changes
- Export the linear and logistic models as their `ai.onnx.ml` equivalents
- Export the tree models as tree ensembles
- Export the scalers and encoders the root package fits, so a pipeline exports as one graph rather than as a model with the preprocessing lost
- Export a fitted pipeline as a single graph
- Verify by round trip: score the exported model in onnxruntime and compare against this package, on the same data
- Refuse to export a model with no `ai.onnx.ml` equivalent, naming it, rather than emitting something that loads and returns wrong numbers

## Impact
- Affected specs: `ml-onnx`
- Affected code: `ml/`
- Depends on `add-ml-estimator-protocol`, `add-ml-pipeline` and `add-ml-decision-tree`
- Import is out of scope
- Imputation is out of scope, and not by preference. `ai.onnx.ml.Imputer` needs a fitted replacement value, and nothing in the repository produces one: `FillWithMean` and its siblings compute a statistic of whatever table they are called on and mutate it in place. A fitted imputer is its own change, and it is needed for pipelines before it is needed for export
