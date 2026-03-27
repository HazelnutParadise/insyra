# Change: Define acceleration runtime capability

## Why
Acceleration work needs a frozen public API and package boundary before backend discovery, memory modeling, scheduling, or CLI exposure can converge. Without a runtime capability spec, later changes would each make their own assumptions about types and session ownership.

## What Changes
- Add a new `accel-runtime` capability
- Freeze the opt-in package boundary under `insyra/accel`
- Define the public runtime surface: `Config`, `Session`, `Device`, `Dataset`, `Buffer`, and `Report`
- Record the design constraints for typed columnar execution and CPU compatibility

## Impact
- Affected specs: `accel-runtime`
- Affected code: future `insyra/accel` packages and any CPU entry points that delegate to accel sessions
