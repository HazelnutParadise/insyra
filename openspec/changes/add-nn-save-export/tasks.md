# Tasks

## 1. Writers

- [ ] 1.1 SaveSafeTensors: sorted names, deterministic bytes, F32/I64/BOOL, reverse-validated; LoadSafeTensors reads it back identically (ungated round trip)
- [ ] 1.2 Sequential.SaveWeights: torch-convention names, Linear transpose applied, BatchNorm running stats included
- [ ] 1.3 Sequential.ExportONNX via the ml protowire patterns: Dense/activations/Conv2D/BatchNorm2D(inference, trained stats)/pools/Flatten/LayerNorm; Dropout omitted; Func and Embedding refused naming layer and position

## 2. Proof

- [ ] 2.1 Gated: torch loads our safetensors and reproduces Predict; python safetensors reads identical values
- [ ] 2.2 Round trip: trained MLP and CNN export → nn.LoadONNX matches Predict exactly; onnxruntime matches within f32 tolerance (gated)
- [ ] 2.3 Existing suites pass unchanged

## 3. Sync

- [ ] 3.1 Docs/nn.md save/export section; changelogs both languages; skills
