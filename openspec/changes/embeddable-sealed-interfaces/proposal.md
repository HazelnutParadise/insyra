## Why

K-7 of #208 said `IDataList`/`IDataTable` carry hidden methods, so nothing outside the module can implement them, and asked whether to open them or drop them; #228's remaining half was `Merge` taking an `IDataTable` and then insisting on a bare `*DataTable`. Measured on 2026-09-26: a caller's type that embeds `*insyra.DataTable` already satisfies the interface and works with `stats`, but `Merge` rejected it, and isr's tables failed the interface because isr overrides `ClearErr` and `SetErr` to keep chaining. The interfaces also missed 27 and 17 of the concrete types' methods. The owner ruled to keep them sealed, make embedding the supported way to extend, list every method, and record why any method is left out.

## What Changes

- `IDataTable` and `IDataList` list every exported method of `*DataTable` and `*DataList`, except `ClearErr`, `SetErr`, `Pivot` and `Unpivot`, which isr overrides with its own signatures. **BREAKING (small)**: calling `ClearErr()`/`SetErr()` on an interface-typed variable needs the concrete type.
- A hidden `coreTable()` on `IDataTable` lets `Merge` reach the table inside any embedding type.
- `TestInterfacesListEveryMethod` enforces the list and names each exception with its reason; tests pin a caller-style embedding type and an isr table going through `Merge` and `stats`.
- QU-2: `CAPM` and `Beta` share one input check.

## Capabilities

### New Capabilities
- `embeddable-interfaces`: the interfaces are sealed, complete, and satisfied by any type embedding the core ones.

### Modified Capabilities
None.

## Impact

- `interfaces.go`, `datatable_merge.go`, `quant/capm.go`; tests `interface_contract_test.go`, `interface_embedding_test.go`, `isr/interface_satisfaction_test.go`.
- `Docs/DataTable.md`, `skills/insyra/SKILL.md`, both changelogs, `AGENTS.md`.
