## ADDED Requirements

### Requirement: A fitted scaler can be written out and read back
`StandardScaler`, `MinMaxScaler`, `RobustScaler` and `MaxAbsScaler` SHALL each encode to JSON with their complete fitted state, and SHALL decode from that JSON into a scaler of the same kind whose `Params`, `Transform`, `InverseTransform` and DataList counterparts produce exactly what the original produces. Decoding SHALL refuse JSON written by a different kind of scaler and SHALL leave the receiver unchanged when it fails. Parameters that are NaN or infinite SHALL survive the round trip.

#### Scenario: Round trip of a fitted scaler
- **WHEN** a min-max scaler fitted with output range [-1, 1] on two columns, one addressed by name and one by column letter, is encoded to JSON and decoded into a new `MinMaxScaler`
- **THEN** the new scaler's `Params()` equals the original's, and `Transform` and `InverseTransform` of another table give bit-identical results

#### Scenario: Wrong kind
- **WHEN** JSON from a `StandardScaler` is decoded into a `RobustScaler`
- **THEN** decoding returns an error and the receiver keeps its previous state

#### Scenario: Column fitted with no values
- **WHEN** a scaler fitted on a column holding only missing values is encoded and decoded
- **THEN** encoding succeeds and the NaN parameters come back as NaN
