# Design: ml-one-onnx-export-name

## Which name stays

`ExportONNX` stays. It is the name of the method every exportable model has (`model.ExportONNX(w)`), of the `Exporter` interface's method, of `nn.Sequential.ExportONNX`, and the only spelling `Docs/ml.md` and the tests use. `WriteONNX` appears nowhere outside its own definition.

## Why `Model`, not `Exporter`

`Exporter` would reject an unsupported model at compile time, but it would also make the package function a restatement of the method it calls, and it would reject a fitted pipeline held as a `Model`, which is how `Estimator.Fit` hands one back: a caller would have to assert `fitted.(ml.Exporter)` before exporting a pipeline that is exportable. `Model` keeps the function's reason to exist, exporting any fitted model a caller holds, and still turns every non-model argument into a compile error. An unsupported model is refused at run time with its name, as today.

## The deprecated wrapper keeps `any`

A deprecation period exists so that old code keeps compiling. `WriteONNX(w, fitted any)` therefore keeps its parameter type: a `Model` goes to `ExportONNX`, and anything else gets the `ml: ONNX export does not support %T` error the exporter already returns for it.

## Verification

- `reflect.TypeOf(ml.ExportONNX).In(1)` is `ml.Model` (red on the old `any`).
- The doc comments of `WriteONNX`, `DecisionTreeClassifierOptions` and `DecisionTreeRegressorOptions` carry a `Deprecated:` paragraph naming the replacement and the removal sentence (red: no such paragraph).
- `WriteONNX` writes the same bytes as `ExportONNX` for a fitted model, refuses a non-model and a nil value without writing, and `ExportONNX(w, nil)` is an error.
