# 0004 — The root package reaches acceleration through a seam

**Status:** accepted
**Date:** 2026-07-29

## Context
`accel` imports the root `insyra` package, so the root package cannot import `accel`.

## Decision
The root package reaches acceleration only through a registration seam. It switches to a direct import only when a root-package operation is measured to win on a device.

## Options rejected
- Move the device layer into a package with no `insyra` dependency now, so the root can import it. Affordable (a hello-world gains 1.9 s of cold build and about 200 KB), but nothing in the root package clears the threshold, so there is nothing to gain yet.

## Consequences
Nothing in `stats` or the root package calls `accel`. `nn` imports `accel` directly, which record 0009 decided separately because `nn → accel → insyra` has no cycle.
