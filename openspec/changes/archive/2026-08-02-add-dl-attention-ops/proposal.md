# Change: The attention operator family — a transformer encoder runs in pure Go

## Why
The MLP family proves the pipeline; the attention family is where the value concentrates. BERT-class encoders are what production Go services actually want to run without Python — embedding and reranking inference — and these are exactly the kernels the decided GGUF/LLM future will call directly, so every operator landed here is paid for twice.

The engineering weight is not in the operators, it is in the tensor layer: batched `MatMul` over leading dimensions and full numpy-style broadcasting are where implementations go subtly wrong. The one-op parity harness carries the proof burden — every kernel is checked against `onnxruntime` on generated graphs, including rank-mismatch and broadcast cases chosen to hit the subtle paths.

## What Changes
- N-D `MatMul` with batched leading dimensions and broadcasting between batch shapes; the 2-D case stays as fast path
- Attention-family kernels: `LayerNormalization`, `Gelu` and `Erf`, `Div`, `Sqrt`, `Pow`, `ReduceMean`, `Softmax` with an axis attribute, `Squeeze`/`Unsqueeze`, `Slice`, `Split`, `Where`, `Equal`/`Greater`, `Expand`, `Shape`, general-permutation `Transpose` (extending the T1 kernel)
- Parity rows in the one-op harness for every new kernel, with broadcast-shape cases explicitly among the generated inputs
- Whole-model proof: a small transformer encoder block (multi-head self-attention + feed-forward + LayerNorm, fixed weights, built with the Python `onnx` package — no torch dependency) runs and matches `onnxruntime` within f32 tolerance
- A manual smoke path for a real downloaded encoder model behind an env flag, documented but not required by CI, so a real MiniLM-class file can be checked locally without shipping a 90MB fixture

## Impact
- Affected specs: `dl-inference`
- Affected code: `dl/` kernels and tensor broadcasting; docs, changelogs, skills
- Blocked by: `add-dl-onnx-mlp-inference`
- Additive
