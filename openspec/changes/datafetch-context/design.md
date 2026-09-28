# Design: datafetch-context

## Variants, not new signatures

The finding offered two ways: a `...Context` variant beside each method, or a context parameter added to every signature. The repository already chose the first for `ReadSQL`, `ToSQL` and `py.RunCode`, and it breaks no caller, so it is used here too. The plain method is a one-line call to its variant with `context.Background()`, so there is one implementation of each fetch.

## What a cancelled call returns

A cancelled call returns `ctx.Err()` itself rather than a wrapped transport error, the same contract `py.RunCodeContext` documents, so `errors.Is(err, context.Canceled)` and a plain comparison both work. After any failed request each method checks `ctx.Err()` first: a request ended by the caller is neither retried nor classified. Without that check `Reverse` would turn a caller's deadline into `ErrGeocodeTimeout`, which is retryable, and spend a retry on a request that can no longer be made.

Each method also checks the context before its first request. `IntervalLimiter.Wait` returns at once without looking at the context when throttling is off, and a custom `http.RoundTripper` is not obliged to honour a request's context, so relying on either would let a cancelled call send a request.

The limiter itself is not changed: it already takes a context, and its spacing contract belongs to another change.

## The geocoding batch

`ReverseCols` already has a partial-result shape for an exhausted quota: the resolved rows, the rest `pending`, and the error. A cancelled batch uses the same shape, which is what a caller resuming a large batch needs; the rows it already paid quota for are not thrown away.

## Google Maps

`Search` and `GetReviews` report every failure as a `nil` result and a warning. Their `Context` forms keep that, and the wait between review pages becomes a timer that also ends when the context does. Changing them to return an `error` is a separate question for the error-shape rule and is not part of this change.

## Yahoo Finance

go-yfinance sends its requests through CycleTLS and offers no context or cancel function, and one call can send several requests: a cookie, a crumb, then the data. The only way to return promptly is to run the call on another goroutine and stop waiting for it. The goroutine owns nothing the caller keeps. `HistoryContext` converts its parameters, copying the dates, before the call starts, so a caller changing them afterwards cannot race it. The per-call go-yfinance ticker is created with the shared client, whose closing it does not own. The result goes into a channel with room for one value, so the goroutine never blocks and ends when the call does. A panic inside it is recovered and turned into an error, because a panic on a goroutine the caller did not start cannot be recovered by the caller. With `context.Background()` there is nothing to abandon, so the call runs on the caller's goroutine exactly as before.

Abandoning has costs, and the documentation states them. The abandoned call runs to its end and may send its remaining requests, each bounded only by go-yfinance's own timeout, which a stalled TLS handshake was measured to outlast. Until it ends it holds the client's read lock, so a later call on the same client may wait for it. The goroutine also keeps the `YFinanceClient` reachable with `runtime.KeepAlive`: without it, a caller dropping the client could let its finalizer run `Close`, which blocks on that lock, and the runtime's one finalizer goroutine would stall with it. The alternative, waiting for the call to end, would not return promptly, which is what a caller passing a context asks for.

The 26 methods differ only in which go-yfinance method they call, the table name, and whether date columns are converted. Adding a variant to each by copying their bodies would double about 600 lines of repetition, so they are rewritten around one generic helper that creates the ticker, runs the call under the context and builds the named table. `History` and `Quote` keep their own loop, because they are the two that wait for the limiter and retry; the other methods make one attempt, as they do today.
