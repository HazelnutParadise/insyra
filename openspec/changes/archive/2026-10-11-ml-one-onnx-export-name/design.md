# Design: ml-one-onnx-export-name

## Which name stays

`ExportONNX` stays. It is the name of the method every exportable model has (`model.ExportONNX(w)`), of the `Exporter` interface's method, of `nn.Sequential.ExportONNX`, and the only spelling `Docs/ml.md` and the tests use. `WriteONNX` appears nowhere outside its own definition.

## The deprecated wrapper keeps `any`

A deprecation period exists so that old code keeps compiling. `WriteONNX(w, fitted any)` therefore keeps its parameter type and forwards to `ExportONNX`, so the two write the same bytes and refuse the same values.

`ExportONNX` keeps `any` on this line for the same reason. Typing it as `Model` would make every caller holding an `any` value stop compiling, which is a breaking change. The typed form is on 0.4.

## Options aliases

The two tree option aliases stay as type aliases, so code that names either one still compiles and fits the same tree as `DecisionTreeOptions`. Their doc comments name `DecisionTreeOptions` as the replacement.

## Verification

- `onnx_one_name_test.go`: `WriteONNX`, `DecisionTreeClassifierOptions` and `DecisionTreeRegressorOptions` carry a `Deprecated:` paragraph naming the replacement and the removal sentence.
- `TestWriteONNXKeepsItsMeaning`: `WriteONNX` writes the same bytes as `ExportONNX` for a fitted model and refuses a value that is not a model, and nil, without writing.
- `TestDeprecatedTreeOptionsAreTheSameType`: the two aliases fit the same tree as `DecisionTreeOptions`.
