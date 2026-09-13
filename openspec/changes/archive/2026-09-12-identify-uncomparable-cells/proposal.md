# Proposal: identify-uncomparable-cells

## Why

A cell Go cannot compare crashes the library. `DataList.Counter` and `DataTable.Counter` build a `map[any]int` keyed by the cell value, and a value that cannot be hashed panics there. Reproduced end to end against sqlite on 2026-09-12:

```
BLOB column cell type = []uint8
col.Count(thatValue)  = 0      (two rows hold exactly it)
col.Counter()         → panic: hash of unhashable type []uint8
```

This is not a corner someone has to go looking for. `convertSQLValue` deliberately preserves binary columns as `[]byte` — its comment says so and `isBinaryColumn` matches BLOB, BYTEA, BINARY, VARBINARY, LONGBLOB, MEDIUMBLOB and TINYBLOB — so **reading a BLOB column and calling `Counter` kills the process**, against the rule that the library never panics by default.

The `Count` result in the same reproduction is the other half. `equalCell` catches the comparison panic and reports "not equal", which is a deliberate convention, so a value that is plainly in the list is never found.

One rule fixes both, and it is the rule the rest of the library already follows for grouping: **a cell Go cannot compare is identified by its type and its content.**

Three things have to be true for that rule to hold, and none of them is optional:

- **The encoding has to be recursive.** `encodeGroupKey`'s default arm is `o:%T:%v`, which does not descend into a slice, array or map. Measured: `[]any{1}` and `[]any{"1"}` produce the same key, so the integer and the string are counted as one. `GroupBy`, `Pivot` and `Merge` merge them today; a stand-in key built on the same encoding would inherit it on day one. The encoding is the identity, so this is a prerequisite rather than an extra.
- **The comparability test has to be the right one.** `reflect.TypeOf(v).Comparable()` is wrong. Measured: `[2]any{[]int{1}, 2}` and `struct{X any}{[]int{1}}` both report `true` and both still panic on `==`. `reflect.ValueOf(v).Comparable()` gets all four cases right.
- **`Count` and `Counter` have to agree.** Fixing only `Counter` would leave the same value counted twice by one method and not found at all by the other.

## What Changes

- A new unexported `encodeCell(v any) string`: the canonical identity of a cell value. It keeps `encodeGroupKey`'s existing scalar rules (`n:`, `s:`, `b:`, `i:`, `f:`) so grouping of ordinary values is unchanged, and descends into slices, arrays, maps and structs, encoding each element by the same rules. Maps are written in encoded-key order so the result does not depend on map iteration. Pointers, channels and funcs are encoded by address, matching what `==` means for them. Recursion stops at depth 64 and falls back to the address: a self-referential slice would otherwise overflow the stack, which in Go is a fatal error that `recover` cannot catch.
- `encodeGroupKey` and `uniqueKey` call it from their default arms. Their scalar output is byte-for-byte what it was.
- `UncomparableKey`, a two-field struct of strings, stands in for a cell value that cannot be a map key. `Type` is exported; the encoded content is not, because the format is shared with the encoder and will change. `String()` renders the type with a truncated, hex-for-bytes rendering of the content, so printing a whole counter stays readable.
- `ToMapKey(v any) any` returns `v` when `reflect.ValueOf(v).Comparable()` and an `UncomparableKey` otherwise. A caller looks up an uncomparable value with `counter[insyra.ToMapKey(v)]`.
- Both `Counter` methods key through `ToMapKey`. A comparable value is still keyed by itself, so `counter[1]` and `counter["a"]` are unchanged.
- `equalCell` and `valueMatcher` compare encodings when the value is not comparable, so `Count`, `FindAll`, `Replace`, `DropAll` and `IsEqualTo` find a `[]byte` cell that is there.

Not in scope: `labelKey` in `datatable_encode.go` has the same non-recursive default arm, but its integer rule is by value rather than by identity, so it is a different question. Recorded as a follow-up.

Also not settled here: whether `Counter` should merge `int(1)` and `int64(1)`. Comparable values keep being keyed by themselves, so that open question is left exactly as it was.

## Capabilities

### New Capabilities

- `cell-identity`: how a cell value that Go cannot compare is identified, counted, matched and grouped.

### Modified Capabilities

(none)

## Impact

- `Counter` stops panicking and starts counting values it used to crash on. This is the live crash on the SQL BLOB path.
- `Count`, `FindAll`, `Replace`, `DropAll` and `IsEqualTo` start matching uncomparable cells that hold equal content. They used to report "not found" and "not equal" for a value sitting in the list.
- `GroupBy`, `Pivot` and `Merge` stop merging distinct nested values that happened to print alike. Ordinary scalar keys are unaffected, byte for byte.
- `UncomparableKey` and `ToMapKey` are new exported names.

## Backport to dev (0.3.x)

Dev received: `encodeCell`, `UncomparableKey` and `ToMapKey` in `cell_identity.go`, with the later fixes to the composite key's last character and the display limit; both `Counter` methods keying through `ToMapKey`; `comparableEqual`, which `equalCell` and so every lookup go through, comparing an uncomparable value by its encoding, so `Count`, `FindAll`, `Replace`, `DropAll`, the `DataTable` find and drop methods, `IsEqualTo` and `IsTheSameAs` all identify such a cell by type and content; the tests, including the `DataTable` side; the `Counter` docs; both changelogs; two AGENTS.md follow-ups measured on this line.

Adapted:
- The three renames (`KeyOf`, `CounterKey`, `MapKey`) are collapsed into the final name, `ToMapKey`, here and in the tasks, so no intermediate name appears on this line. 0.4's `ToToMapKey` typo in its docs and changelog was not carried.
- On this line `IsEqualTo` and `IsTheSameAs` use `comparableEqual`, not `equalCell`, so the encoding fallback went into `comparableEqual`. A scalar `NaN` cell stays unequal in `IsEqualTo`; inside a slice, map or struct a `NaN` is encoded as content, so two `[]any{NaN}` cells are equal and match. A test pins both.
- `TestIsEqualToUncomparableCellsDoNotPanic` asserted that such cells are unequal, the behaviour this change replaces; it was rewritten to the new rule.
- The changelog entries that said such a cell "simply does not match" or is "unequal" now say it is matched by type and content, so `## Unreleased` states the net behaviour.
- `valueMatcher` encodes a searched value Go cannot compare once per call and rejects a cell of another dynamic type before encoding it (backport review). Going through `equalCell` for every cell re-encoded the searched value each time: `FindAll`, `Count` and `DropRowsContain` with a 64 KB `[]byte` over 20,000 `int64` cells took 5.3 s together.
- The AGENTS.md follow-ups: the `labelKey` entry is rewritten to cover `encodeGroupKey` and `uniqueKey` too, which still merge nested values that print alike on this line (measured); the self-referential `NewDataList` entry was re-measured here.

Left on 0.4:
- `encodeGroupKey` and `uniqueKey` calling `encodeCell`: breaking, it changes the keys `GroupBy`, `Pivot`, `Merge` and `Describe`'s unique count use for existing data, and 0.4's encoder also regresses `time.Time` keys in uncached zones. The requirement scenarios about grouping and the grouping changelog entry go with it.
- The `Counter` `int(1)`/`int64(1)` AGENTS.md note: this line has no such follow-up to annotate.
- `delivery-status.md` edits.
