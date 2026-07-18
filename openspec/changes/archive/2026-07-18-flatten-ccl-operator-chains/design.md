# Design — flatten CCL operator chains

## Semantic ground truth (verified by reading the evaluator)

- Binary eval order: left, then right, then `applyOperator` — **no short-circuit** for `&&`/`||` (ccl_evaluator.go:377-393). The fold must therefore evaluate every operand left-to-right and must NOT introduce short-circuiting.
- `.` and `:` never reach `applyOperator`; they dispatch on the **nodes** (`evaluateRowAccess`, `evaluateRange`), and `IsRowDependent` special-cases `:` between static column refs. They are excluded from folding entirely.
- The parser's precedence loop (ccl_parser.go:178-190) builds the left spine one invocation at a time; higher-precedence right operands are absorbed by recursion. Everything the loop itself stacks is by construction a left fold — so folding exactly that loop's accumulation is semantics-preserving for **any** operator mix on the spine (`a*b+c-d` folds to `fold(a, [(*,b),(+,c),(-,d)])`, evaluated identically).

## Node & parser

`cclFoldChainNode{init cclNode; ops []string; operands []cclNode}` with the invariant `len(ops) == len(operands) >= 2`.

Loop accumulates `(op, right)` pairs; `flushFold` materializes:
- 0 ops → `left` unchanged; 1 op → `cclBinaryOpNode` (identical shape to today); ≥2 → fold node.
- `.`/`:` flushes first, then wraps in a binary node as today.

Single-op guarantee matters twice: no perf/structure change for simple formulas, and `2^3^2`-style left association is pinned by golden tests either way.

## Evaluation

```
acc = eval(init); for i: rv = eval(operands[i]); acc = applyOperator(ops[i], acc, rv)
```
Identical operation sequence to the nested-binary recursion (a, b, apply, c, apply, …), identical first-error-wins order, one recursion-depth bookkeeping entry per chain instead of per node (the per-node `goid` + `sync.Map` cost is the dominant eval overhead for chains — long chains get *faster*).

## Switch-site completeness

`grep cclBinaryOpNode` enumerates every consumer: evaluator (eval, `IsRowDependent`, `containsRowIndex`), compiler (`Bind`, `exceedsDepth`). Each gains a fold case (init + operands; no `.`/`:` logic needed since those never fold). `checkExpressionMode` is token-level — untouched. `GetExpressionNode`/`IsAssignmentNode`/`IsNewColNode` only inspect statement wrappers — untouched. No code outside `internal/ccl` can see node types (unexported; `CCLNode` is opaque).

## Behavior-preservation harness

1. **Differential**: test-only `unfoldChains(n)` rewrites fold nodes back into nested binaries; a corpus of expressions (mixed ops/precedences/types, nil, errors at first/middle/last position, comparisons feeding logicals, `.`/`:` mixes) must produce identical values **and identical error strings** through both shapes.
2. **Golden**: exact values for order-sensitive cases (`2^3^2`=64, `10-2-3`=5, `100/5/2`=10, string-concat coercion, non-short-circuit errors like `true || (1/0)` erroring).
3. **Structural invariants**: single op → binary node; ≥2 ops → fold; `.`/`:` always binary; flush interactions (`A:B + 1 + 2`).
4. Full existing CCL suite unchanged.

## Performance gate (user condition: no regression)

Benchmarks (compile + eval × simple/medium/chain-100) run on the **pre-change HEAD** first, then post-change, compared. Expected: simple/medium within noise (compile adds two small slice allocs per operator run; eval path for binary nodes byte-identical), chain eval faster. If simple/medium eval regresses beyond noise, the change does not land as-is.

## Depth-limit interactions

- Long chains no longer hit `expression too complex` — the over-deep-chain tests flip from expect-error to expect-correct-value.
- Nesting limits unchanged (`maxParseDepth` guard untouched; `exceedsDepth` still rejects deep genuine nesting; statement-mode limit test switches to a nesting input).
- Evaluator `maxEvalDepth` still bounds remaining recursion; fold operands are shallow by construction.
