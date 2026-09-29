# Design: nn-fit-context

## Where the context is checked

Between batches: at the top of each batch, and once more after an epoch's last batch, before its validation. A batch is the unit `Fit` cannot interrupt without leaving the tape half-recorded: its forward pass, backward pass and optimizer step either all run or none do. Checking inside a kernel would need the context threaded through the tape, which the tape's plain-function design keeps out. A batch of the sizes `Fit` is used with takes milliseconds to a few seconds, which bounds how late a cancellation is seen.

## What comes back

A cancelled run returns `ctx.Err()` itself, so `errors.Is(err, context.Canceled)` and `errors.Is(err, context.DeadlineExceeded)` tell a cancellation from a failure, as in `datafetch`. It also returns the `FitResult` of the epochs that finished, the way `ReverseTableContext` returns the rows it resolved: the caller chose to stop, and the losses so far are what they need to decide what to do next. An epoch counts as finished once its batches and its validation have run and its `Progress` call has returned, so every epoch in the result has the fields a normal run gives it. A failure other than cancellation keeps returning a nil result, as it does today.

The parameters are updated in place by every optimizer step, so after a cancellation they hold the state after the last step taken. Cancelling from `Progress` stops before the next epoch's first batch, which leaves the model exactly as a run of that many epochs would have, bit for bit: the shuffle and dropout streams are the same up to that point.

## A context that is already done

`FitContext` validates its arguments first, so a bad configuration is reported as such, then checks the context before it seeds the tape's random stream or touches a parameter. A done context therefore changes nothing. A nil context is refused, as the `datafetch` context methods refuse one.

## Verification

- Cancelling from `Progress` after epoch 1 of 3 returns `context.Canceled`, a result with one epoch, and parameters bit-identical to a `Fit` with `Epochs: 1` and the same seed.
- A `Func` layer that cancels during the forward pass of batch 2 sees no batch 3: the run returns `context.Canceled` with no finished epoch, after exactly two forward passes.
- A context cancelled before the call leaves every parameter unchanged, calls `Progress` never, and returns an empty result; an expired deadline returns `context.DeadlineExceeded`; a nil context is an error.
- The existing `Fit` tests, the MNIST convergence proof and its parity with the hand-written loop keep passing unchanged, since `Fit` is `FitContext` with a context that is never done.
