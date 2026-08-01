# Change: Name the positive class instead of leaving it to column order — WITHDRAWN

## Status
Withdrawn on 2026-08-01, implemented and then reverted the same day. Kept as the record of a defect that was reported without being measured.

## What it proposed
Let a caller name which class `ROCAUCMetric` treats as positive, on the reasoning that the current default — the second of the model's classes — is decided by sort order over the training labels, so for `"churned"` and `"retained"` it lands on `"retained"` and reports the complement of the intended score.

## Why it is withdrawn
The premise is false. Naming the other class cannot change the result.

Area under the ROC curve is invariant under swapping which class is positive, provided the score column swaps with it — and in this API it must, because the metric receives the whole probability table rather than a single column. The two probability columns of a binary problem are complementary, so ranking by the second class's probability descending is ranking by the first class's ascending; swapping the labels and swapping the ranking cancel exactly.

Measured, on a deliberately weak logistic fit chosen so the area would not sit at 1:

| Positive class | Area under the curve |
| --- | --- |
| default | 0.50838574423480087 |
| `"retained"` (the second class) | 0.50838574423480087 |
| `"churned"` (the first class) | 0.50838574423480087 |

The option was built, and every test written to show it changing the answer failed because the answer does not change. Shipping it would have added a control that provably does nothing — the same "satisfied in form but not in substance" defect this package has already found three times, introduced deliberately this time.

The complement scenario the proposal described is real, but it belongs to an API that takes a single score column with a separate label convention, which is scikit-learn's shape and not this one.

## What survives
The documentation, which was the part that was actually missing. `ROCAUCMetric` now states which class it treats as positive, where that order comes from, and that the choice does not affect the result — so the next reader does not reopen this. That went in as a comment-only edit, outside this change.

## What it cost
One misdiagnosis, reported as established in a list of six findings. It was the only one of the six that had not been measured before being reported.
