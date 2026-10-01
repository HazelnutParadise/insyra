# Proposal: py-nested-table-results

## Why

Two `AGENTS.md` follow-ups the adversarial review of `py-typed-run` recorded on 2026-09-30, both in how `py` decodes the value Python passes to `insyra.Return`:

- A struct with a field named `DataTable` or `DataList` of insyra's pointer type is taken for an `isr` wrapper. The whole result is decoded as a table into that field and JSON decoding never runs: ``Run[struct{DataTable *insyra.DataTable `json:"table"`; Score float64 `json:"score"`}]`` on `{"table": …, "score": 7}` returned `Score` 0 with a nil error. Restricting that path to embedded fields, as the follow-up suggested, is not enough: measured on 2026-09-30, JSON decoding gives such a field an empty 0×0 table with a nil error, because `*insyra.DataTable` has no exported fields and no `UnmarshalJSON`. A table or list anywhere below the top level of the result type, in a struct field, a map value or a slice element, is lost the same way.
- Python cannot send a table inside another value either: measured with the pinned environment, `insyra.Return({"table": pd.DataFrame({"a": [1, 2]}), "score": 7})` fails with `Object of type DataFrame is not JSON serializable`, because only a DataFrame or Series at the top level is turned into its table payload.
- An empty DataFrame comes back with its columns renamed: `Run[*insyra.DataTable]` on `{"data": [], "columns": ["a", "b"]}` gave a 0×2 table named `[a_1 b_1]`, which a filter matching no rows produces. The empty path names the columns and then `SetColNames` names them again.

## What Changes

- `insyra.Return` turns a DataFrame or Series anywhere inside a dict, list or tuple into its table or list payload, so `insyra.Return({"table": df, "score": 7})` and `insyra.Return([df1, df2])` can be sent.
- When the type a result is decoded into holds `*insyra.DataTable`, `*insyra.DataList`, `insyra.IDataTable` or `insyra.IDataList` below its top level, in a struct field, a map value, a slice or array element or behind a pointer, those parts are decoded with the table and list decoders and everything else exactly as `encoding/json` decodes it. The struct field rules are a port of `typeFields` from `encoding/json`, not a matcher of this package's own: the adversarial review of the first version measured a matcher of its own recursing until the stack ran out on a struct embedding itself, dropping `,string`, choosing among case-insensitive keys at random and filling hidden fields. A type that implements `json.Unmarshaler` or `encoding.TextUnmarshaler`, or holds no table or list, is decoded through JSON exactly as before.
- An embedded table or list follows `encoding/json`'s rule for an embedded struct: with a tag it is a field under that name, and without one it is decoded from the whole value, which is how the `isr` types hold theirs. A value of the other kind for it is an error. A struct with an ordinary field named `DataTable` or `DataList` is decoded field by field.
- An empty DataFrame keeps its column names.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `py-run-api`: how a result holding tables or lists below its top level, and an empty DataFrame, are decoded.

## Impact

- Code: `py/pyresult_decode.go`, and `insyra.Return` in `py/builtin.go`, which hands `json.dumps` a `default` that turns a DataFrame or Series it meets into its payload. `json.dumps` calls it only for an object it cannot write, so plain results cost nothing more, where walking every list element in Python would have.
- Tests: `py/nested_result_test.go`, with the test binary standing in for Python; the gated end-to-end test returns a dict holding a DataFrame through the real environment.
- Docs: `Docs/py.md`, `CHANGELOG.md`, `CHANGELOG_TW.md`, `delivery-status.md`, and `AGENTS.md`, where the two follow-ups are deleted and the `SetColNames` defect behind the empty-frame one is recorded.
