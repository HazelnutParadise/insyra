## Context

State currently stores a string. Info parses it and ignores invalid timestamp text. LoadState and RestoreVariables reject a JSON type error; callers can fall back to empty state, so introducing an error for formerly accepted timestamp text risks losing variables.

## Goals / Non-Goals

Goals: a time.Time API on 0.4 and unchanged string wire compatibility with 0.3.6, including exact variable numbers. Non-goals: other #261 findings, general corrupt-file recovery, variable codecs or concurrent writers.

## Decisions

State owns MarshalJSON and UnmarshalJSON so direct calls and nested export/import use the same compatibility rule. Decode a string timestamp and variables with Decoder.UseNumber. Empty, missing, null and unparsable timestamp text give zero time; other JSON types retain the existing decode error. Assign decoded state only after structural decoding succeeds. Marshal zero as an empty string and nonzero times through time.Time.MarshalJSON (RFC3339 with fractional seconds and range validation), preserving fractional seconds in valid older input. SaveVariables keeps its existing RFC3339 writer. Info directly copies LastAccess; Import uses IsZero and UTC time.

Using time.Time's decoder alone rejects empty strings. Decoding through json.Unmarshal without UseNumber rounds integers beyond 2^53. Official references: https://pkg.go.dev/time#Time.UnmarshalJSON and https://pkg.go.dev/encoding/json#Decoder.UseNumber.

## Risks / Trade-offs

Invalid timestamp text is normalized to unknown time rather than preserved verbatim; variables survive as before. Malformed documents and non-string timestamps keep existing errors; broader recovery is out of scope. The public type change breaks Go string assignments only on 0.4, with a dev notice ahead of release.

## Migration Plan

No file migration. Existing files load in place and exported timestamps remain strings readable on dev. Callers replace string comparisons with IsZero and use Format(time.RFC3339) when they need text.
