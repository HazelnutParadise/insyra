# Delivery Status

## Current Phase
`insyra/ml` v1 shipped and archived, then audited against its own claims and repaired. Acceleration is dormant by choice — no call site was measured to beat the CPU, so the runtime waits for one rather than being wired into something that would lose.

## Stage Objective
Nothing is in flight. The stage that just closed had one objective and met it: audit `insyra/ml` against what it claimed and fix what did not hold. Six findings were reported; five were real and are fixed, one was a misdiagnosis and is recorded as one.

## Active Workstreams
None. Every proposed change is implemented, verified and archived — `openspec/changes/` holds nothing but `archive/`.

## Milestones
| id | target | owner | status | verification_signal |
| --- | --- | --- | --- | --- |
| M6 | One operation executes on a real device | planning | done | a `float32` column reduced on a discovered device returned the CPU's value, with measured transfer/dispatch/readback replacing fabricated constants |
| M7 | Foundations for default-on acceleration | planning | done | `Session` safe for concurrent use, the execution seam carries a whole dataset in one submission, `accel.Default()` provides a process-shared session |
| M8 | Find a call site a device wins | planning | done | answered, negatively. 96 shapes measured against an all-core host: the device wins 42, by at most 3.63x, and no `stats` call site clears the threshold. Three operations were removed rather than shipped |
| M9 | `stats` can be wrapped | planning | done | every regression predicts and matches R's `predict()`; KMeans assigns new rows; logistic and Poisson publish their link; PCA returns its projection parameters |
| M10 | `insyra/ml` v1 | planning | done | protocol, pipelines, model selection, decision trees and ONNX export archived; 21 review findings raised, the blocking ones fixed, the rest adjudicated |
| M11 | `ml` holds up under audit | planning | done | ONNX round-trip passes against a real `onnxruntime` for all five model shapes; `stats` refuses input it cannot read instead of scoring zeros; metrics declare a direction; pipelines name the columns their estimator saw |

Milestone order is the blocking sequence. OpenSpec has no dependency relationship between changes, so nothing else carries it.

## Current Blockers
None. One coverage gap is carried in `AGENTS.md` follow-ups rather than here, because it waits on hardware nobody has rather than on a decision: the device path is verified on Apple and Metal only.

## Next Verifiable Output
Undecided, and deliberately. The acceleration line has no landing site that measurement supports, and `insyra/ml` v1 is complete and now audited. The two things worth measuring before anything is proposed are brute-force KNN, whose candidate count is the training-set size and therefore always clears the floor, and the DBSCAN neighbourhood scan. Both need measuring against a parallel host first.

The one thing that would pay for itself immediately is neither: run the suite on a machine with every reference toolchain installed, in CI rather than by hand. Two verifications in this project have been believed on the strength of a skip.

## Next Ticket
None. Proposing one before that measurement would repeat the mistake this phase already made once — building a kernel, then discovering its intended caller could not clear its own profitability floor.

## Decision Log
Deltas that still change what someone would do. The standing technical decisions they produced — the precision contract, the device rules, the measured thresholds — live in [ENG.md](ENG.md); the full history is in git.

- decision: A verification that skips is not a verification. Install every reference toolchain before calling a suite green.
  rationale: The ONNX round-trip needed `onnxruntime`, which was on no machine it ever ran on, so it skipped everywhere and reported nothing. Executed for the first time, it failed immediately on two defects that made every exported model unloadable: a stray string field on every non-string attribute, and tree nodes written deepest-leaf-first when the runtime reads the first entry as the root. Both had been archived as verified.
  timestamp: 2026-08-01
  impacted_ticket_ids: archived

- decision: A conversion that cannot fail is a conversion that fabricates. Numeric input in `stats` goes through one validating helper.
  rationale: Regression and correlation read every value through `ToF64Slice`, which yields 0 for anything unparseable and returns a full-length slice regardless. One blank among six observations moved a Pearson coefficient from 0.9992 to 0.9879, silently. Three treatments of unreadable values now coexist deliberately — refusal, listwise deletion, a learned tree direction — and each is documented; substituting a zero is not one of them.
  timestamp: 2026-08-01
  impacted_ticket_ids: archived

- decision: Do not ship a control that provably cannot change the answer.
  rationale: A positive-class option for `ROCAUCMetric` was proposed, built, and then withdrawn on measurement: the metric receives the whole probability table, so naming the other class swaps both the label and the score column and the two cancel exactly — all three choices returned 0.50838574423480087. It was the only one of six reported findings that had not been measured before being reported.
  timestamp: 2026-08-01
  impacted_ticket_ids: archived

- decision: Judge every acceleration claim against a host using all its cores.
  rationale: `accel` contains no parallelism at all, while the rest of the library parallelises its hot loops as a matter of course. Every speedup recorded before this compared a GPU against one core on an eight-core machine, so none described the alternative a user actually has. Re-measured honestly, the best case fell from 7.1x to 3.63x.
  timestamp: 2026-07-29
  impacted_ticket_ids: all prior accel measurements

- decision: Decide whether a device may be used from the shape of the result, not from how hot the operation is.
  rationale: A selection can be made exact by having the device propose and the host decide; a new value cannot. This puts CCL on both sides of the line — its predicates qualify, its value expressions do not, despite being the 11x and 146x measurements. That is the cost of choosing bit parity, chosen knowingly.
  timestamp: 2026-07-28
  impacted_ticket_ids: future kernel work

- decision: Remove every accelerated operation except exact-nearest, and require a measurement before adding any back.
  rationale: Three of four were known not to pay. Keeping them meant a kernel to maintain against a single-maintainer backend, a method to document, and a changelog entry claiming a speedup measured against one core.
  timestamp: 2026-07-29
  impacted_ticket_ids: archived

- decision: `insyra/ml` imitates scikit-learn, with fitting returning a separate value.
  rationale: Not a preference — the library had already imitated it in four places without saying so. `datatable_scale.go` ships `Scaler` with sklearn's four verbs and the same four scaler names; `stats/knn.go` reproduces `KNeighborsClassifier`'s options verbatim. Consistency with itself outranks any other precedent. What does not translate is the machinery: sklearn discovers parameters by reflecting over `__init__`, which Go cannot do, so a pipeline step is a fit function rather than a configured object.
  timestamp: 2026-07-29
  impacted_ticket_ids: archived

- decision: `ml` wraps `stats` and reimplements nothing, except the one family `stats` lacks.
  rationale: The numerics are already validated against R. Only decision trees are written from scratch — the family with no `float64` legacy, the one the ONNX corpus is full of, and the one whose split finding is selection-shaped.
  timestamp: 2026-07-29
  impacted_ticket_ids: archived

- decision: Keep the root package reaching acceleration through a seam, and switch to a direct import only when a root operation is measured to win.
  rationale: `accel` imports the root package, so the root cannot import `accel`. Moving the device layer would fix that and costs a hello-world 1.9 s of cold build and 200 KB — affordable, but nothing in the root package clears the threshold, so there is no reason yet.
  timestamp: 2026-07-29
  impacted_ticket_ids: future root-package work

- decision: Withdraw the string-kernel deferral rather than leave it open.
  rationale: It deferred work to a Phase 2 that no longer exists, and the result-shape rule would exclude string kernels regardless of phasing. Deferring something to a future that has been ruled out is not a plan.
  timestamp: 2026-08-01
  impacted_ticket_ids: archived

- decision: A fix is not verified until the test covering it has been shown to fail without it, and a claim about code is not established until the code has been read.
  rationale: The first half caught three unearned claims, including one of my own — a probe written to prove a conformance fix passed with the fix disabled, because the fake model was failing an unrelated assertion first. The second half was learned the hard way: `SimpleImputer` was described in three commit messages and in `ENG.md` as having claimed a reversibility it lacked, on the strength of a handoff summary. It never did. It deliberately omits `InverseTransform` and its comment explains why, which is the same reasoning applied correctly. The false premise also reached a review brief as the exemplar of a failure class.
  timestamp: 2026-08-01
  impacted_ticket_ids: none

## Source Links
- [ENG.md](ENG.md) — architecture, test seams, the precision contract, standing assumptions. Read before changing any of them.
- [AGENTS.md](AGENTS.md) — the operating contract, including the acceleration rules and the open follow-ups.
- [openspec/changes/archive/](openspec/changes/archive/) — 41 archived changes, each holding its own proposal, spec deltas and tasks.
- [openspec/specs/](openspec/specs/) — the current capability specs, which reflect the code as it stands.
- [Docs/accel.md](Docs/accel.md), [Docs/ml.md](Docs/ml.md) — the user-facing surfaces.
- Open issues: [#190](https://github.com/HazelnutParadise/insyra/issues/190) KNN algorithm selection, [#191](https://github.com/HazelnutParadise/insyra/issues/191) CCL recursion-depth overhead.

## Handoff Notes
- **Two pure-CPU wins are measured and unclaimed**, and both are worth more than anything acceleration offered. `stats`' KNN auto-selection picks a ball tree up to 3.3x slower than parallel brute force on unstructured data (#190). CCL spends 4.8x–6.2x more time on recursion-depth bookkeeping than on evaluating (#191).
- **Cross-language tests skip silently** without `Rscript` and `python` plus their scientific stacks — now including `onnxruntime`. A skipped verification is indistinguishable from a passing one. Two changes were nearly archived on tests that had never run, and one was archived on a round-trip that had never run anywhere, hiding two defects that made every exported model invalid.
- **The refusal of unreadable numeric input is a breaking change** for anyone whose data contains blanks. What used to return a number now returns an error; the number it used to return was wrong. `ToF64Slice` still backs 54 call sites in `plot`, `gplot`, `quant` and the CLI — display paths where a zero is visible rather than laundered into a coefficient. A new numeric analysis must not join them.
- **Nothing calls `accel`.** It is reachable through `allpkgs` or a direct import only, so its dormancy costs users nothing today.
- **The race detector cannot reach the device on macOS**, upstream and pre-existing (gogpu/wgpu#280). Device tests skip under the `race` build tag.
