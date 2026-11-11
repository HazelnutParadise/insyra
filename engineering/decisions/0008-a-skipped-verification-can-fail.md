# 0008 — A verification that skips can be made to fail

**Status:** accepted
**Date:** 2026-08-01

## Context
The ONNX round trip needed `onnxruntime`, which was on no machine it ever ran on, so it skipped everywhere. Executed for the first time, it failed at once on two defects that made every exported model unloadable. Measured in a clean environment, the `Clustering Parity` workflow's own installs left `sklearn` missing, so the job dedicated to the parity suite was green while running none of it.

## Decision
Every reference gate routes through `internal/reftest`. `INSYRA_REQUIRE_REFERENCE_TOOLCHAINS=1` turns every skip for a missing toolchain into a failure that names what was missing and what went unverified, and the `Reference Verification` workflow installs every toolchain and runs with it set. Without the variable the default is unchanged, so `go test ./...` passes on a machine with none of the toolchains.

## Options rejected
- Install the toolchains in CI and rely on that. Toolchains go missing again, and a skip reports nothing.

## Consequences
- The one deliberate exception is a `psych::factor.scores` upstream bug, where R is present and broken rather than absent.
- Gates that need data rather than a package follow the same rule: the `Neural Network Data Gates` workflow fails when one of its tests skipped.
