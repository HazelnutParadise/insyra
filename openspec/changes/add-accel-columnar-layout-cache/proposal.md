# Change: Define acceleration columnar layout and cache

## Why
Acceleration cannot be layered directly on top of `[]any` containers. The phase needs a shared spec for typed columnar projection, null handling, string transport, and device/shared-memory caching before implementation work starts.

## What Changes
- Add a new `accel-memory-cache` capability
- Define typed columnar projection from `DataTable` and `DataList`
- Define validity bitmaps, numeric buffers, and encoded string transport
- Define device/shared-memory cache budgets, keys, residency, and eviction behavior

## Impact
- Affected specs: `accel-memory-cache`
- Affected code: future dataset conversion, cache management, and backend memory allocators
