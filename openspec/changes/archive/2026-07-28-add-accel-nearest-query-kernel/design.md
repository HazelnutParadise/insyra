## Context
`OpSquaredDistance` measured 3.2x at 64 queries with readback taking 16 of the 18 ms. The output is the problem, and the fix is to stop returning it.

## Goals / Non-Goals
- Goals: one output pair per row; the same parity gate; the shape `stats.KMeans` needs
- Non-Goals: k-nearest for KNN, which needs a per-row top-k rather than a single minimum; wiring `stats` itself

## Decisions

- Decision: one thread per row, looping every query inside it.
  - Rationale: it keeps the operation order identical to the CPU reference, which is what parity needs, and it makes the reduction free — the minimum is tracked in a register instead of being read back and reduced on the host.

- Decision: seed the running minimum with the first query's distance rather than an infinity sentinel, then loop from the second.
  - Rationale: a sentinel forces a decision about what happens when every distance is NaN, and invites the two implementations to disagree about it. Seeding from a real value means the answer is always one of the queries, and the CPU reference is written the same way.

- Decision: ties go to the lowest query index, enforced by comparing with a strict less-than.
  - Rationale: floating point makes exact ties rare but not impossible — duplicate query points produce them immediately. A strict comparison keeps the first minimum on both sides, so the index is deterministic rather than a race between equal candidates.

- Decision: return the distance alongside the index.
  - Rationale: KMeans needs the distance for its inertia, and recomputing it on the host would cost another pass over the data. It is one extra float per row against the rows-times-queries matrix this change exists to avoid.

## Measured

Apple M3, 100,000 rows, 16 dimensions.

| Queries | CPU | GPU | readback | ratio | matrix kernel for comparison |
| --- | --- | --- | --- | --- | --- |
| 16 | 23.5 ms | 5.47 ms | 3.24 ms | 4.3x | 9.56 ms |
| 64 | 80.5 ms | 5.92 ms | 3.89 ms | 13.6x | 18.3 ms |

The shape of the numbers is the result, more than the ratio. Going from 16 to 64 query points quadruples the work: the CPU takes 3.4x longer, while the device goes from 5.47 to 5.92 ms. The output stopped growing with the query count, so the cost stopped growing with it too. Against the matrix kernel this is 3.1x at 64 queries, from nothing but not sending the matrix back.

Parity holds: 200,000 rows against 32 query points, every index and every distance bit-identical to the reference.

One caveat about the reported figure. `Submit` is asynchronous, so the readback timer starts before the device has finished and captures the wait as well as the copy. For a compute-heavy kernel like this one, "readback" is mostly the device working. The number is still the right one for deciding whether to dispatch — it is what the caller waits for — but it should not be read as bus time.

## Risks / Trade-offs
- Trade-off: a caller that genuinely wants the whole matrix still pays for it.
  - Accepted: `OpSquaredDistance` stays. The two operations answer different questions and the planner picks by what was asked.
