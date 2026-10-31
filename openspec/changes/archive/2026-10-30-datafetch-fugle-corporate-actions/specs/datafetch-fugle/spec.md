## ADDED Requirements

### Requirement: Corporate actions start from a required date

`CorporateActionsDividends`, `CorporateActionsCapitalChanges` and `CorporateActionsListingApplicants` SHALL take the first date as a required parameter and send it as `start_date`, send `FugleCorporateActionsOptions.To` as `end_date` when it is set, sort oldest first unless `Descending` is set, and refuse a zero first date, a `To` before it and more than one options value before sending a request.

#### Scenario: The default order

- **WHEN** CorporateActionsDividends is called with a date and no options
- **THEN** the request carries start_date and sort=asc and no end_date

### Requirement: Corporate action tables keep every documented column

The corporate action methods SHALL return every documented column even when no row has it, as nil cells, turn the nested `raw` object of a capital change into columns prefixed `raw_`, and read the event dates as times, leaving a date that is null as nil.

#### Scenario: A capital reduction

- **WHEN** a capital reduction carries raw.reason and no raw.splitRatio
- **THEN** the table has raw_reason holding the reason and raw_splitRatio holding nil
