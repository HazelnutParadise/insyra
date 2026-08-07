# ccl-evaluation Specification

## Purpose
Recursion-depth guarding in the CCL evaluator: strict limits enforced through stack-local state, with no process-global goroutine-keyed bookkeeping on the per-node hot path. Created by archiving change thread-ccl-eval-depth.
## Requirements
### Requirement: Depth guards are stack-local and exactly as strict
The CCL evaluator SHALL enforce the maximum evaluation depth and the maximum function-call depth through state threaded on the call stack, SHALL NOT consult process-global goroutine-keyed state on the per-node evaluation path, and SHALL keep both limits and both error messages identical to the previous implementation.

#### Scenario: A pathologically deep expression still fails cleanly

- **WHEN** an AST nested deeper than the maximum evaluation depth is evaluated
- **THEN** evaluation returns the existing maximum-recursion-depth error instead of overflowing the stack

#### Scenario: The function-call guard still trips

- **WHEN** function-call nesting exceeds the maximum function-call depth
- **THEN** the call returns the existing maximum-function-call-depth error

#### Scenario: Concurrent evaluations are independent

- **WHEN** multiple goroutines evaluate expressions concurrently under the race detector
- **THEN** no goroutine's depth accounting observes another's, and no race is reported

#### Scenario: Results are unchanged

- **WHEN** the full existing CCL test suite runs against the threaded implementation
- **THEN** every result and every error is identical to before

### Requirement: Bookkeeping no longer dominates evaluation
Depth accounting SHALL cost no more than integer parameter passing per node, and the recovered speedup SHALL be measured against a same-session guard-removed upper bound rather than quoted from prior measurements.

#### Scenario: The #191 expression recovers most of the upper bound

- **WHEN** the issue #191 benchmark expression runs over 10k and 100k rows with guards threaded, guards global, and guards removed, in the same session
- **THEN** the threaded version's wall time is recorded and lands close to the guard-removed bound, and the recovery fraction is recorded in the change

#### Scenario: The hot path carries no goroutine-ID lookups

- **WHEN** `internal/ccl` is inspected after the change
- **THEN** it contains no `goid` import and no goroutine-keyed depth maps
