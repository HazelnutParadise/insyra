# Arrow IPC reader verification

## Test-first evidence

The public API skeleton returned `not implemented`. Before streaming was implemented, `go test ./arrowipc` compiled and failed the successful-read, column-selection, cancellation and scalar-parity tests with that error. `TestInspectDictionaryAndNestedSchemaWithoutValues` failed with `not implemented` before the inspection implementation, then passed with a ReaderAt that refuses record and dictionary regions.

`TestReaderAtMayReturnEOFWithFullRead` reproduced a valid ReaderAt reporting `(len(p), io.EOF)` at the trailer. Inspection initially returned EOF; normalizing a complete EOF read fixes that contract. Short reads with no error are rejected as `io.ErrUnexpectedEOF`.

Fixture corrections before implementation: Arrow DataType compares through `arrow.TypeEqual`, the float64 fixture uses `ptr(float64(0))`, native records of 4 and 2 rows split at 3 produce 3 outputs (3, 1, 2), and successful stream termination may close the error channel without a value. Arrow's Parquet writer rejects Duration, so the Parquet comparison stores schema-unit counts as int64, matching Parquet's cell rule.

## Review corrections

A 298-byte schema-only IPC with a corrupted 65,536-field count allocated 5,770,336 bytes before Arrow returned an error. The regression now checks vector extents before decoding and keeps allocation below 512 KiB for this input. Forward-only table references, a traversal budget and 64 field levels bound malformed schema expansion. Unknown extension metadata is explicitly refused for selected columns instead of accepting the storage type.

Source-error regressions distinguish pure EOF from joined errors. A complete record-body read can carry a real error that io.ReadFull would discard; the guarded reader reports zero usable bytes on any rejected message so the original error propagates.

Decoded-byte regression fixtures verify that MaxBatchBytes rejects oversized compressed-buffer claims before decompression and counts retained dictionaries together with record buffers. The default is 256 MiB. The guard validates the existing metadata/body reads and adds no Range requests. All three public probes were repeated after these changes with the same byte/request totals.

## Manual MaleCNS sources

The [manual probe source](manual_probe.go.txt) is included as text so it cannot become a CI network test. Run it from the repository root with the current package checked out:

```sh
cp openspec/changes/archive/2026-11-03-add-arrow-ipc-reader/manual_probe.go.txt /tmp/insyra-arrow-probe.go
GOTOOLCHAIN=go1.26.10 go run /tmp/insyra-arrow-probe.go
```

The probe requires HTTP 206 and exact Content-Range for every ReadAt request, records byte/request counts, samples three rows, then cancels and breaks from the iterator (which drains internal production).

These are the three inputs in [issue #377](https://github.com/HazelnutParadise/insyra/issues/377), linked by the [official download page](https://male-cns.janelia.org/download/). They are used only in a manual HTTP Range probe, never CI. Observed on 2026-11-02 (Asia/Taipei).

| File | Total bytes | Native batches | Inspect bytes / requests |
| --- | ---: | ---: | ---: |
| [connectome-weights-male-cns-v1.0-minconf-0.5.feather](https://storage.googleapis.com/flyem-male-cns/v1.0/connectome-data/flat-connectome/connectome-weights-male-cns-v1.0-minconf-0.5.feather) | 1,051,241,946 | 2,318 | 56,624 / 3 |
| [body-annotations-male-cns-v1.0-minconf-0.5.feather](https://storage.googleapis.com/flyem-male-cns/v1.0/connectome-data/flat-connectome/body-annotations-male-cns-v1.0-minconf-0.5.feather) | 14,483,314 | 4 | 6,872 / 3 |
| [body-neurotransmitters-male-cns-v1.0.feather](https://storage.googleapis.com/flyem-male-cns/v1.0/connectome-data/flat-connectome/body-neurotransmitters-male-cns-v1.0.feather) | 43,282,834 | 29 | 3,136 / 3 |

All inspect as IPC metadata V5. Each inspection fetches a 6-byte header, 10-byte trailer and the exact footer. No dictionary or record bodies are fetched.

The weights schema has nullable int64 `body_pre`, `body_post`, `weight`. Annotations retain the ordered int8-to-UTF8 `statusLabel` dictionary and int64-list `somaLocation` and `tosomaLocation`. Neurotransmitters has int64 `body` (not `bodyId`), int32 counts, float64 confidence and UTF8 labels. Each `StreamFrom` probe requested a three-row output and then cancelled and drained the stream. Weights returned exact Go int64 triples (first row: body_pre 10352, body_post 10351, weight 2591). Annotations returned int64 bodyId and decoded statusLabel strings (10001 → "Roughly traced", 10003 → "Prelim Roughly traced"). Neurotransmitters returned int64 body, int32 total_nt_predictions, float64 confidence and UTF8 predicted_nt (first row: 10001, 1015, 0.5004955542656235, "acetylcholine"). All ended with the intentionally requested context cancellation.

Inspection plus opening and reading one native batch fetched 690,690 bytes / 10 ranges for weights, 5,574,538 / 12 for annotations, and 3,114,298 / 10 for neurotransmitters. These include both inspection and stream-open footer reads. Selected-column conversion does not avoid reading the other columns in that native batch. The annotation lists can be inspected; selecting them is explicitly rejected by the offline generated-list tests. No full-file scan was performed.

## External API evidence

Checked the installed v18.8.0 `arrow/ipc/file_reader.go`, public `go doc` and official documentation:

- [IPC v18.8.0 API and RecordBatchAt ownership](https://pkg.go.dev/github.com/apache/arrow-go/v18@v18.8.0/arrow/ipc)
- [Arrow File.fbs footer fields](https://github.com/apache/arrow/blob/main/format/File.fbs)
- [IPC file layout](https://arrow.apache.org/docs/format/Columnar.html#ipc-file-format)
- [Feather V2 format](https://arrow.apache.org/docs/python/feather.html)

## Verification state

Local dev verification passed: go build ./..., go test ./arrowipc ./parquet, go test -race ./arrowipc ./parquet, go vet ./..., golangci-lint run (0 issues), and GOTOOLCHAIN=go1.26.10 govulncheck ./... (0 affected vulnerabilities). Strict change validation passed. Both-line PR checks are pending; no merge is authorized. The original Go 1.26.9 baseline reported existing [GO-2026-6809](https://pkg.go.dev/vuln/GO-2026-6809), fixed in Go 1.26.10. The patched toolchain is available; the module directive remains unchanged.

## 0.4 port

This port preserves the dev schema, conversion, byte-bound and malformed-input code. Stream/StreamFrom expose this line's Parquet-shaped lazy iter.Seq2 interface. The 0.4 column selector additionally follows AGENTS.md: []any with index strings, insyra.Name and int positions. Dedicated iterator tests were written first and failed compilation against the copied two-channel signatures. Shared semantic tests use a test-only sequence-to-channel adapter; direct iterator tests cover laziness, reiteration, one terminal error, early break and caller-panic cleanup.

The go.mod directive on this line remains 1.26.8. The manual iterator probe succeeded on all three inputs with the same schema-only and total byte/request counts recorded above. It cancelled and broke after the three-row sample; iterator cleanup finished before returning, with no terminal error consumed after the caller stopped. Local 0.4 build/test/vet/lint/vulnerability results are recorded separately below; dev results are not substituted for them.

0.4 local results: go build ./..., go test ./arrowipc ./parquet, go test -race ./arrowipc ./parquet, go vet ./..., golangci-lint run (0 issues), GOTOOLCHAIN=go1.26.10 govulncheck ./... (0 affected vulnerabilities), and strict change validation all passed. Cell conversion, schema and decoded-buffer validators are byte-identical to the dev companion; selection follows each line's contract. Independent port review found no confirmed code defect; its release-note placement finding was corrected on both lines. The owner's explicit same-PR archive instruction is recorded in proposal.md.

## Final PR-review bounds and selectors

Offline output-budget tests first failed to compile without MaxOutputBytes, then failed behaviorally against its no-op skeleton: repeated dictionary strings/binary and null-only cell slots all emitted a table with no error. The implementation now preflights the whole output before cell allocation or payload copying, charging 16 bytes per selected cell and copied payload bytes per occurrence. The independent 256 MiB default keeps MaxBatchBytes's native-buffer semantics unchanged. Boundary, negative-limit and repeated-value cases pass.

The 0.4 selector test first failed against []string, then passed with []any: B selects the second field, Name("B") selects the first field named B, int and negative positions work, and invalid/duplicate aliases fail. Shared name-based tests were only adapted to explicit Name selectors; their value, ownership and error assertions were preserved. The dev companion retains its Parquet name-only option.

No total-stream row cap is added. A large logical Null record is handled through bounded output batches, backpressure and context cancellation. A caller retaining every batch chooses to materialize the stream; the package does not claim a bound on that retained total or on total runtime.

After these review corrections, 0.4 build, touched-package tests, touched-package race tests, vet, lint (0 issues), strict change/main-spec validation and Go 1.26.10 govulncheck (0 affected vulnerabilities) passed again. Latest-head CI remains authoritative on PR #676.
