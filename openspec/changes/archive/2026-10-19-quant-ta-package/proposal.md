# Proposal: quant-ta-package

## Why

`quant` held two kinds of tools: the ones that evaluate strategies (performance and risk, CAPM, options, portfolio, bootstrap, walk-forward, event studies), and technical analysis, which our October 2026 study found has no edge. Mixed in one package, every `quant` user reads as if the whole package were technical analysis. None of the technical-analysis tools was released yet, so moving them costs no compatibility.

## What Changes

- New package `quant/ta` with RSI, the volume and breadth indicators, `Bias`, `BiasZ`, `ZigZag` and `Regimes`, and the moving-average, MACD, Bollinger, ATR and KD indicators.
- `Divergence` is `ta.PriceDivergence`, since on its own `Divergence` reads as MACD's divergence.
- `Docs/quant-ta.md` holds the warning table and the pages of the moved tools; `Docs/quant.md` points to it.
- Specs, both CHANGELOGs, the README package tables and `allpkgs` follow.
- Chart patterns, gaps and the shuffled-price control ('quant-chart-patterns', 'quant-gap-classes', 'quant-bar-shuffle') are in `ta` too.

## Capabilities

### New Capabilities

- `quant-ta-package`: where the technical-analysis tools live.

## Impact

`quant/ta/`, `quant/` (tests only), `Docs/`, `openspec/specs/`, `README.md`, `README_TW.md`, `allpkgs/allpkgs.go`, both CHANGELOGs.
