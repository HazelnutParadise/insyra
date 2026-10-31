# Tasks

- [x] A local WebSocket server speaking Fugle's protocol for the tests.
- [x] Tests first: subscribe and receive, candles snapshots split, books flattened, unsubscribe by the stored id, reconnect with renewed ids, auth and plan errors ending the stream, drop-oldest with `Dropped`, `Collect`, `Close` twice, refused input sending no frame.
- [x] `datafetch/fugle_stream.go`; a live check in `fugle_live_test.go`.
- [x] `Docs/datafetch.md`: the streaming section and the Python SDK mapping.
- [x] Entries under `## Unreleased` / `### `datafetch`` in both CHANGELOGs.
- [x] `openspec validate datafetch-fugle-stream --strict` passes.
