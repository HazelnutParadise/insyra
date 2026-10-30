# Tasks

- [x] Tests first: slots are zero after each pop, across wrap-around, growth and mixed ends; `Clear` keeps the array and its capacity, empties every slot and does not allocate on an allocated ring; a zero ring still clears; benchmarks for push/pop and push/clear.
- [x] Zero the vacated slot in `PopFront` and `PopBack`; zero in place in `Clear`.
- [x] Entries under `## Unreleased` / `### Core` in both CHANGELOGs.
- [x] `openspec validate engine-ring-releases-references --strict` passes.
