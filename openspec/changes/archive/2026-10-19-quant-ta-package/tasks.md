# Tasks

- [x] Move the code and tests to `quant/ta`; `quant/ta` does not import `quant`.
- [x] `Divergence` becomes `PriceDivergence`.
- [x] `Docs/quant-ta.md`, `Docs/quant.md`, `Docs/README.md`, README tables, `allpkgs`.
- [x] Specs and both CHANGELOGs use the new names.
- [x] Chart patterns, gaps and the shuffled-price control join `ta` with the patterns change; RSI takes `SegmentStarts`.
- [x] `openspec validate --specs --strict` passes.
