## ADDED Requirements

### Requirement: A batch reports an ended context even when its input is empty or invalid

`ReverseColsContext` SHALL return the context's error when the context has already ended, even when the lat and lng lists are empty or every coordinate in them is invalid, and `ReverseTableContext` SHALL do the same once its two columns resolve. Neither MUST send a request in that case.

#### Scenario: Cancelled before an empty batch

- **WHEN** `ReverseColsContext` is called with a cancelled context and two empty lists
- **THEN** it returns an error for which `errors.Is(err, context.Canceled)` holds, and sends no request

#### Scenario: Cancelled before a batch of invalid coordinates

- **WHEN** `ReverseColsContext` is called with a cancelled context, a latitude list holding `"bad"` and a longitude list holding `nil`
- **THEN** it returns `context.Canceled`, and sends no request
