# Tasks

## 1. Tensor layer
- [ ] 1.1 N-D batched MatMul with batch-shape broadcasting; 2-D fast path preserved; incompatible shapes refused naming both
- [ ] 1.2 Broadcasting utilities shared with element-wise ops rather than re-derived per kernel

## 2. Kernels
- [ ] 2.1 LayerNormalization (axis, epsilon), Gelu, Erf
- [ ] 2.2 Div, Sqrt, Pow, ReduceMean, Softmax with axis, Squeeze, Unsqueeze, Expand, Shape
- [ ] 2.3 Slice, Split, Where, Equal, Greater, general-permutation Transpose

## 3. Verification
- [ ] 3.1 One-op parity rows per kernel with broadcast/axis cases in the generated inputs
- [ ] 3.2 Encoder-block whole-model parity: multi-head self-attention + FFN + LayerNorm, fixed weights via the Python onnx builder, matched against onnxruntime
- [ ] 3.3 Manual real-model smoke path behind an env flag, documented

## 4. Documentation
- [ ] 4.1 Docs/dl.md operator table updated; changelogs both languages; skills note the encoder capability
