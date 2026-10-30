# Proposal: engine-ring-releases-references

## Why

[#588](https://github.com/HazelnutParadise/insyra/issues/588), from StoreCoach's use of `engine/ring`: `PopFront` and `PopBack` left the removed element in the backing array, so a ring of pointers, strings or slices kept their referents reachable until the slot was overwritten, and `Clear` allocated a new array each time although it promised to keep the capacity. Not a wrong result and not a permanent leak, but memory the garbage collector could not reclaim and an allocation a reuse loop did not need.

## What Changes

- `PopFront` and `PopBack` set the vacated slot to the zero value.
- `Clear` zeroes the existing backing array in place (a zero ring still gets a buffer of one slot).
- Public methods, FIFO and LIFO order, capacity and the zero-value ring behave as before; the ring stays non-concurrent.
- Measured: pushing 64 elements then `Clear` went from 188 ns and one 512-byte allocation to 138 ns and none; 64 pushes and pops take about 7% longer for the zeroing.

## Capabilities

### New Capabilities

- `engine-ring`: the circular buffer's handling of removed elements.

## Impact

`internal/core/ring.go` (the type `engine/ring` aliases), `internal/core/ring_test.go`, both CHANGELOGs.
