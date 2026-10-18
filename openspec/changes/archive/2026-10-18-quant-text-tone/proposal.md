# Proposal: quant-text-tone

## Why

[#460](https://github.com/HazelnutParadise/insyra/issues/460): testing whether market moves follow over-optimistic news needed dictionary tone scoring and daily aggregation aligned to trading days, and `quant` had no text tooling. The alignment is the step most likely to leak the future when done by hand.

## What Changes

- `quant.ToneCounts(texts, lexicon, opts)` counts positive and negative words per text, in word mode or longest-match-first substring mode (consuming each match, for Chinese), with a negation window that cancels a positive word.
- `quant.DailyTone(published, counts, tradingDays, opts)` assigns each article to a trading day (weekend and after-close news to the next one) and returns daily counts, a rolling net tone and a trailing z-score that never includes the day itself.
- The lexicon is supplied by the caller; none is shipped.
- Not done from the issue: the NYT Archive API client, which needs an API key that was not available to test with, and so stays open.

## Capabilities

### New Capabilities

- `quant-text-tone`: dictionary tone scoring and look-ahead-safe daily aggregation.

## Impact

`quant/tone.go`, tests, `Docs/quant.md`, both CHANGELOGs.
