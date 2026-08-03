# Tasks

## 1. Layers

- [x] 1.1 MultiHeadAttention(embed, heads): fused in-projection, out-projection, self-attention from existing tape ops; torch in_proj/out_proj weight mapping in LoadWeights/SaveWeights
- [x] 1.2 Residual(layers ...Layer) combinator; an encoder block composes from MHA + LayerNorm + Dense FFN + Residual alone

## 2. Proof

- [x] 2.1 Gated: torch.nn.MultiheadAttention (batch_first) weight round-trip and forward parity; layer-built encoder one AdamW step matching torch on loss, gradients, and parameters
- [x] 2.2 Ungated: finite differences through the whole MHA layer; structural tests (head divisibility errors named, Residual composes, ONNX export refuses MHA by name)
- [x] 2.3 Existing suites pass unchanged

## 3. Sync

- [x] 3.1 Docs/nn.md attention layer section; changelogs both languages; skills
