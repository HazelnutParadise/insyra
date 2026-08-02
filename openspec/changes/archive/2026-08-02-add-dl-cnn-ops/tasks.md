# Tasks

## 1. Kernels
- [x] 1.1 Conv 2-D: pads (explicit and auto_pad), strides, dilations, groups (depthwise included); shape errors name the node inputs
- [x] 1.2 MaxPool, AveragePool with count_include_pad, GlobalAveragePool
- [x] 1.3 BatchNormalization (inference), Pad (constant mode)

## 2. Verification
- [x] 2.1 One-op parity rows enumerating the attribute combinations, not sampling defaults
- [x] 2.2 Whole-model MNIST-class CNN with fixed weights via the Python onnx builder, matched against onnxruntime

## 3. Documentation
- [x] 3.1 Docs/dl.md operator table; changelogs both languages; skills
