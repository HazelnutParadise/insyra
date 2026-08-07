# Change: MultiHeadAttention as a layer — encoders from layers alone

## Why

The tape trains encoder blocks, but only through hand-composed Func
closures. M29 makes the transformer a first-class citizen of the layer
surface: a MultiHeadAttention layer with torch-convention weights, and
the pieces to compose an encoder block from layers alone — which is
also the groundwork the decided llm track builds on.

## What Changes

- `MultiHeadAttention(embed, heads int)` layer: fused in-projection
  (torch's in_proj_weight/in_proj_bias convention) and out-projection,
  self-attention over [batch, seq, embed] inputs, composed from the
  existing tape ops (batched MatMul, axis-Softmax, Transpose,
  Reshape/Split) — no new VJPs, the attention gradients already exist.
- `Residual(layers ...Layer)` — a block whose output is x plus the
  sub-stack's output, making encoder wiring declarative; and
  `LayerNorm` already exists, so a full encoder block is
  Residual(MHA)+LayerNorm+Residual(FFN)+LayerNorm from layers alone.
- torch interop: LoadWeights/SaveWeights map
  in_proj_weight/in_proj_bias/out_proj.weight/out_proj.bias with the
  established transpose conventions; the fixture round-trips against
  torch.nn.MultiheadAttention (batch_first=True).
- The proof: an encoder block built from layers alone trains one
  AdamW step matching a torch equivalent (MultiheadAttention +
  LayerNorm + Linear FFN) on loss, every gradient, and every post-step
  parameter; plus a finite-difference check through the whole layer
  and structural tests (mask-free v1; padding masks are future work).
- Docs, changelogs both languages, skills — same change.

## Non-Goals

- No attention masks or causal masking in v1 (the llm track owns
  causal attention with KV cache), no cross-attention, no ONNX export
  mapping for the layer yet (refused by name like Func).

## Impact

- Affected specs: `nn-training`
- Affected code: nn layers, fixtures, docs, changelogs, skills.
