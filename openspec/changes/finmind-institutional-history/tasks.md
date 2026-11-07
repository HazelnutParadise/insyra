## 1. Tests

- [x] 1.1 Two TX days and one TXO day read from FinMind equal the tables built from the exchange's own CSV for the same days, except net values within one thousand
- [x] 1.2 A code that is both a futures and an options code takes the name of the kind asked for
- [x] 1.3 Contract names are requested once per client and again for an unknown code, GTF takes the exchange's name, and a failed names request fails the call without being kept
- [x] 1.5 Calls come before puts whatever order FinMind sends, and waiting for another call's names stops with the context
- [x] 1.4 An empty contract, `to` before `from`, an unknown code, a cancelled context and a zero client are errors

## 2. Implementation

- [x] 2.1 `FinMindClient.rows`, shared by `Dataset` and the institutional methods
- [x] 2.2 `InstitutionalFutures`, `InstitutionalOptions` and their Context forms

## 3. Documentation

- [x] 3.1 TAIFEX history godoc, `ErrTAIFEXRefused` and `Docs/datafetch.md` describe the rolling three-year window and point to FinMind
- [x] 3.2 `Docs/datafetch.md` FinMind section and both changelogs
