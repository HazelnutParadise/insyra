# Change: Separate full string kernels into Phase 2

## Why
The project wants broad string support, but full GPU string-kernel parity would slow or derail Phase 1 runtime convergence. The phase needs an explicit planning boundary so string eligibility and string-kernel completeness are not conflated.

## What Changes
- Add a new `accel-string-kernels` capability for Phase 2
- Define full GPU string-kernel work as a separate proposal track
- Preserve Phase 1 support for encoded-string transport and key-based eligibility without blocking runtime convergence

## Impact
- Affected specs: `accel-string-kernels`
- Affected code: future string kernel implementations and eligibility expansion
