# Tasks

- [x] `FinMind`, `FinMindConfig`, `FinMindClient.Dataset`, `ErrFinMindLimit`, `ErrFinMindLevel` (`datafetch/finmind.go`).
- [x] `doGet`: send fixed headers; return a `finalError` without retrying.
- [x] Tests: typed columns in order, the token header, plan and limit errors not retried, an empty dataset, a ragged dataset, bad calls.
- [x] Live check on the free tier.
- [x] `Docs/datafetch.md` section, entries in both CHANGELOGs.
- [x] `openspec validate --specs --strict` passes.
