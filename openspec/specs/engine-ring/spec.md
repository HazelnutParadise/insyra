# engine-ring Specification

## Purpose
How the circular buffer of engine/ring treats removed elements and Clear, so a ring of references does not keep removed data alive or allocate when reused.
## Requirements
### Requirement: A removed element is not kept alive by the ring

`PopFront` and `PopBack` SHALL set the slot they vacate to the zero value of the element type, so the ring holds no reference to a removed element, and SHALL return the same value and leave the remaining elements in the same order as before.

#### Scenario: A ring of pointers

- **WHEN** a pointer is popped from either end
- **THEN** no slot of the backing array holds it and the other elements are unchanged

### Requirement: Clear reuses the backing array

`Clear` SHALL set every slot to the zero value, reset the length to zero and keep the capacity and the backing array, allocating only when the ring has no buffer yet, and the ring SHALL work normally afterwards.

#### Scenario: Reuse in a loop

- **WHEN** a ring that has its buffer is filled and cleared repeatedly
- **THEN** no allocation happens and no stale reference is left in any slot

