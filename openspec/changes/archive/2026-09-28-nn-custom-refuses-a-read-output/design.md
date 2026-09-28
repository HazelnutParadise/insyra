# Design

## Context

`0.4` 183c3508 fixed four things in `nn`. Only the `Custom` refusal belongs to this change. On this line `NewTape` takes any number of seeds and uses the first, and `Tape` has no recorded error, so `0.4`'s `BackwardFrom` check for a tape given two seeds has nothing to check. `tanhHighPrecision`'s unreachable panic is left as it is, because this line's `error-philosophy` spec does not cover `nn`. The nil `EdgeTopology` accessors are ported separately as a plain fix.

## Decisions

- **Refuse, do not reorder.** `Custom` checks every recorded operation's inputs for `output`, the same scan that already refuses an output a recorded operation produced, and returns `tape custom <name>: output was already read by <op>; record the custom operation before the operations that use its output`. The code and message are `0.4`'s. Reordering the tape would change what the caller recorded; refusing tells the caller which call to move.
