package datafetch

import (
	"context"
	"errors"
	"net/url"
	"time"

	"github.com/HazelnutParadise/insyra"
)

// FugleCorporateActionsOptions are the optional settings of the corporate
// action methods.
type FugleCorporateActionsOptions struct {
	// To is the last date; zero leaves it to Fugle. A future date includes
	// announced events that have not happened yet.
	To time.Time
	// Descending lists the newest first; the default is oldest first.
	Descending bool
}

var (
	fugleDividendColumns = []string{
		"date", "exchange", "symbol", "name", "dividendType", "dividend", "cashDividend", "stockDividendShares",
		"previousClose", "referencePrice", "openingReferencePrice", "exdividendReferencePrice", "limitUpPrice", "limitDownPrice",
	}
	fugleCapitalChangeColumns = []string{
		"symbol", "name", "exchange", "actionType", "haltDate", "resumeDate",
		"raw_splitRatio", "raw_splitType", "raw_parValueBefore", "raw_parValueAfter",
		"raw_reason", "raw_refundPerShare", "raw_sharesPerThousand",
		"raw_cashIncreaseRatioAfterReduction", "raw_subscriptionPrice", "raw_exrightReferencePrice",
		"raw_previousClose", "raw_referencePrice", "raw_openingReferencePrice", "raw_limitUpPrice", "raw_limitDownPrice",
	}
	fugleListingApplicantColumns = []string{
		"symbol", "name", "exchange", "applicationDate", "chairman", "capitalAtApplication",
		"reviewCommitteeDate", "boardApprovalDate", "contractFilingDate", "contractApprovalDate",
		"listedDate", "underwriter", "underwritingPrice", "remarks",
	}
	// fugleEventDates are the event fields read as dates when they hold one.
	fugleEventDates = []string{
		"haltDate", "resumeDate", "applicationDate", "reviewCommitteeDate", "boardApprovalDate",
		"contractFilingDate", "contractApprovalDate", "listedDate",
	}
)

// corporateActions fetches one corporate actions endpoint from from through
// opts' To.
func (s FugleStock) corporateActions(ctx context.Context, path string, from time.Time, columns []string, opts []FugleCorporateActionsOptions) (*insyra.DataTable, error) {
	c := s.c
	p, err := oneFugleOptions(opts)
	if err != nil {
		return nil, err
	}
	if err := contextErr(ctx); err != nil {
		return nil, err
	}
	if err := c.usable(); err != nil {
		return nil, err
	}
	if from.IsZero() {
		return nil, errors.New("datafetch: Fugle corporate actions need a from date")
	}
	values := url.Values{"start_date": {from.Format("2006-01-02")}, "sort": {"asc"}}
	if !p.To.IsZero() {
		if p.To.Before(from) {
			return nil, errors.New("datafetch: Fugle To is before from")
		}
		values.Set("end_date", p.To.Format("2006-01-02"))
	}
	if p.Descending {
		values.Set("sort", "desc")
	}
	resp, err := c.fugleGet(ctx, "/stock/corporate-actions/"+path, values)
	if errors.Is(err, errTWStockNoData) {
		return fugleTable(nil, columns, columns), nil
	}
	if err != nil {
		return nil, err
	}
	rows, err := fugleRows(resp)
	if err != nil {
		return nil, err
	}
	flat := make([]map[string]any, len(rows))
	for i, row := range rows {
		m := map[string]any{}
		flattenFugle("", row, m)
		for _, k := range fugleEventDates {
			if str, ok := m[k].(string); ok {
				if d, err := time.Parse("2006-01-02", str); err == nil {
					m[k] = d
				}
			}
		}
		flat[i] = m
	}
	return fugleTable(flat, columns, columns), nil
}

// CorporateActionsDividends returns the ex-dividend and ex-rights events of
// listed and OTC stocks from from onwards, one row per event (Developer or
// Advanced plan). It is CorporateActionsDividendsContext with
// context.Background().
func (s FugleStock) CorporateActionsDividends(from time.Time, opts ...FugleCorporateActionsOptions) (*insyra.DataTable, error) {
	return s.CorporateActionsDividendsContext(context.Background(), from, opts...)
}

// CorporateActionsDividendsContext is CorporateActionsDividends with a context.
// dividend is the cash and stock dividend together, cashDividend the cash part
// and stockDividendShares the shares granted per thousand. An announced event
// that has not happened yet has nil prices.
func (s FugleStock) CorporateActionsDividendsContext(ctx context.Context, from time.Time, opts ...FugleCorporateActionsOptions) (*insyra.DataTable, error) {
	return s.corporateActions(ctx, "dividends", from, fugleDividendColumns, opts)
}

// CorporateActionsCapitalChanges returns the par value changes, capital
// reductions and ETF splits from from onwards, one row per event (Developer or
// Advanced plan). It is CorporateActionsCapitalChangesContext with
// context.Background().
func (s FugleStock) CorporateActionsCapitalChanges(from time.Time, opts ...FugleCorporateActionsOptions) (*insyra.DataTable, error) {
	return s.CorporateActionsCapitalChangesContext(context.Background(), from, opts...)
}

// CorporateActionsCapitalChangesContext is CorporateActionsCapitalChanges with a
// context. actionType is etf_split_or_merge, par_value_change or
// capital_reduction; the event's details are the raw_ columns, nil where the
// type has no such field.
func (s FugleStock) CorporateActionsCapitalChangesContext(ctx context.Context, from time.Time, opts ...FugleCorporateActionsOptions) (*insyra.DataTable, error) {
	return s.corporateActions(ctx, "capital-changes", from, fugleCapitalChangeColumns, opts)
}

// CorporateActionsListingApplicants returns the companies that applied to list
// on TWSE or TPEx from from onwards, one row per company (Developer or Advanced
// plan). It is CorporateActionsListingApplicantsContext with
// context.Background().
func (s FugleStock) CorporateActionsListingApplicants(from time.Time, opts ...FugleCorporateActionsOptions) (*insyra.DataTable, error) {
	return s.CorporateActionsListingApplicantsContext(context.Background(), from, opts...)
}

// CorporateActionsListingApplicantsContext is CorporateActionsListingApplicants
// with a context. A step not reached yet is nil. capitalAtApplication is in
// thousands of dollars for TWSE and in dollars for TPEx, as Fugle gives it.
func (s FugleStock) CorporateActionsListingApplicantsContext(ctx context.Context, from time.Time, opts ...FugleCorporateActionsOptions) (*insyra.DataTable, error) {
	return s.corporateActions(ctx, "listing-applicants", from, fugleListingApplicantColumns, opts)
}
