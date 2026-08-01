## Context

`KMeansResult` already stores fitted centers and the training assignment, but
it did not expose the operation needed to assign new observations. A caller had
to duplicate the squared-distance loop and could easily diverge on tie rules or
center indexing.

Assignment is selection-shaped: each row chooses one center and a distance. It
can later be served by the exact nearest-query acceleration path, but this
change must first provide a complete CPU implementation with the same answer
regardless of device availability.

## Goals / Non-Goals

**Goals:**

- Add `KMeansResult.Assign` for held-out observations.
- Return the 1-based center index and squared Euclidean distance per row.
- Break equal-distance ties by the lowest center index.
- Reject a feature-count mismatch with both counts in the error.
- Reuse the existing clustering distance primitive and keep a device seam.

**Non-Goals:**

- Wiring `stats` to a GPU in this change.
- Changing KMeans fitting, initialization, convergence, or stored centers.
- Adding a second assignment representation or mutating the result.

## Decisions

### Put assignment on the fitted result

`Assign` takes an `IDataTable`, converts observations and fitted centers through
the existing numeric table conversion, and calls the internal clustering
assignment primitive. The result keeps the fitted centers as the single source
of truth, so new observations cannot accidentally be assigned against a new
fit.

Reimplementing the loop in the public package was rejected because it would
duplicate distance and tie semantics already used by clustering internals.

### Return 1-based center indices

The public method converts the internal zero-based assignment to the package's
existing 1-based cluster convention. Distances remain squared Euclidean values
in `float64` and are returned in observation order.

Using zero-based values was rejected because it would make `Assign` disagree
with `KMeansResult.Cluster` and the existing R-facing convention.

### Validate shape before assigning

Nil results, missing centers, non-numeric input, and mismatched column counts
are rejected before the assignment primitive runs. The mismatch error names the
observed and fitted widths so a caller can correct the table without inspecting
the result internals.

The assignment loop remains a clean operation boundary so a later exact device
path can propose nearest centers while the CPU retains the final decision and
signature.

## Risks / Trade-offs

- **[Risk] A tie is resolved differently from fitting]** → Use the shared
  internal assignment primitive and test exact midpoint observations.
- **[Risk] Center numbering changes during a future refactor]** → Keep the
  public conversion and cluster tests together, including the R comparison.
- **[Risk] Numeric conversion rejects a heterogeneous table]** → Return the
  existing numeric conversion error rather than coercing values silently.
- **[Risk] A future device path changes the CPU answer]** → Require the exact
  host result to remain authoritative and compare device-enabled and CPU-only
  assignments.

## Migration Plan

This is additive. Existing KMeans callers continue using the fitted result;
callers needing new observations can call `Assign`. No persisted result format
changes. Removing the method is a normal source revert and does not affect
fitting.

## Open Questions

Whether assignment crosses the acceleration profitability threshold for a given
shape belongs to the accel measurement work. Device dispatch must be added in a
separate change after that measurement.
