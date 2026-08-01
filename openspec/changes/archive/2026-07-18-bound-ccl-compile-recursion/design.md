# Design — bound CCL compile-time recursion

## Threat model recap

Two independent crash axes, requiring two different guards:

| Input shape | Parse recursion | AST depth | First crash site (pre-fix) |
|---|---|---|---|
| `((((...1` / nested calls / `- - - -5` | O(n) | O(n) | parser (~1.3M chars) |
| `1+1+1+...` (left-associative chain) | **O(1)** — iterative precedence loop | O(n) | `Bind` (~4M terms) |

A parse-depth guard alone misses the second row; an AST-depth check alone misses the first (the stack dies *during* parsing, before any AST exists to measure).

## Decision 1 — guard location: parser chokepoint, not per-walker counters

All AST node types (`cclBinaryOpNode` etc.) are unexported, so the parser is the **only** producer of ASTs. Bounding what the parser can emit therefore protects every downstream recursive consumer (`Bind`, `IsRowDependent`, `containsRowIndex`, `evaluateWithContext`) without touching them.

Rejected alternatives:
- **Depth parameter threaded through `Bind`/`IsRowDependent`/`containsRowIndex`** — triplicates the guard, churns exported-within-package signatures, and still leaves the parser itself unprotected.
- **Input length / token count cap** — blunt: to guarantee AST depth ≤ 10000 the token cap would also have to be ~10000, which rejects legitimate *flat* formulas (e.g. a `CONCAT` with thousands of arguments has huge token count but depth ~2).
- **Rewriting `Bind` iteratively** — most invasive, no benefit once parser output is bounded.

## Decision 2 — parse-recursion guard in `parsePrimary` only

Every recursion cycle passes through `parsePrimary`: `parseExpression` begins by calling it (ccl_parser.go:128), and the only self-recursion that bypasses `parseExpression` (unary `-`/`+`) is inside `parsePrimary` itself. One `depth` field on the `parser` struct, `p.depth++` / `defer p.depth--` at `parsePrimary` entry, error above `maxParseDepth`. Parsing runs once per formula (not per row), so the `defer` cost is irrelevant.

`maxParseDepth = 10000`, matching `maxEvalDepth`. At that depth the parser uses on the order of single-digit MB of stack — three orders of magnitude below the 1GB fatal limit, and far beyond any real formula (real-world nesting is < 100).

## Decision 3 — AST depth capped at `maxEvalDepth`, checked iteratively

`CompileExpression` and `compileStatement` run `astDepth(node)` after a successful parse and reject depth > `maxEvalDepth`. Rationale for reusing the evaluator's constant: evaluation recursion tracks AST depth, so **any AST deeper than `maxEvalDepth` was already un-evaluable** — it would fail later with "maximum recursion depth exceeded" (or crash `Bind` first). Rejecting it at compile time changes the outcome of zero previously-working expressions; it only converts crashes and guaranteed eval failures into prompt compile errors.

`astDepth` must be **iterative** (explicit stack of `(node, depth)` pairs) — a recursive checker would itself overflow on the very inputs it exists to reject. It enumerates children per node type: `cclBinaryOpNode{left,right}`, `cclChainedComparisonNode{values}`, `funcCallNode{args}`, `cclAssignmentNode{expr}`, `cclNewColNode{expr}`; all other node types are leaves.

`compileStatement` covers `CompileMultiline` (per-statement), so `ExecuteCCL` is protected by the same two checks.

## Failure modes considered

- **False rejection of legal formulas**: only if depth ∈ (10000, ∞) were evaluable — impossible, see Decision 3. Deep-but-legal inputs (hundreds of levels) are covered by tests.
- **Transient memory before rejection**: a 100MB garbage formula still allocates tokens + AST before `astDepth` rejects it. Linear, non-fatal, same class as loading a huge file; documented as out of scope (boundary length caps belong to the caller).
- **Future node types**: a new container node type omitted from `astDepth`'s switch would be treated as a leaf, under-counting depth. Mitigated by a code comment beside the node-type declarations pointing at `astDepth`.
- **Concurrent compiles**: the depth counter lives on the per-call `parser` struct (no shared state); `astDepth` is pure. No race surface — unlike `maxEvalDepth`, which needed per-goroutine tracking because evaluation is re-entrant across shared state.
