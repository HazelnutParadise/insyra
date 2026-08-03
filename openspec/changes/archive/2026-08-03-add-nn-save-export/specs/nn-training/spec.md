## ADDED Requirements

### Requirement: What nn trains, nn persists, and everyone can read

`nn` SHALL write SafeTensors files that its own loader, the Python
`safetensors` library, and torch read back identically, and a trained
`Sequential` SHALL export an ONNX graph carrying its trained weights
and running statistics that `nn`'s own runtime matches exactly and
`onnxruntime` matches within f32 tolerance. Layers without an ONNX
mapping SHALL refuse export naming the layer.

#### Scenario: SafeTensors round-trips three ways

- **WHEN** trained weights are saved with SaveSafeTensors
- **THEN** LoadSafeTensors SHALL read identical tensors, and torch
  SHALL load the file as a state dict and reproduce Predict through
  the reference gate

#### Scenario: The ONNX export runs everywhere

- **WHEN** a trained Sequential MLP and CNN export to ONNX
- **THEN** nn.LoadONNX SHALL run them matching Predict exactly, and
  onnxruntime SHALL match within f32 tolerance, with BatchNorm
  carrying the trained running statistics

#### Scenario: Unmappable layers refuse by name

- **WHEN** a model containing Func or Embedding exports
- **THEN** ExportONNX SHALL return an error naming the layer and its
  position
