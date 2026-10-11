package datafetch

import (
	"testing"
	"time"

	"github.com/wnjoon/go-yfinance/pkg/models"
)

// HistoryContext may abandon a request that is still running, so the
// parameters it hands to go-yfinance must not share memory with the caller's:
// a caller changing its start date afterwards would race the request.
func TestYFHistoryParamsConversionCopiesTheDates(t *testing.T) {
	start := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 3, 4, 0, 0, 0, 0, time.UTC)
	params := YFHistoryParams{Start: &start, End: &end, RepairOptions: &models.RepairOptions{FixZeroes: true}}

	model := copyHistoryParams(params)
	start = start.AddDate(1, 0, 0)
	end = end.AddDate(1, 0, 0)
	params.RepairOptions.FixZeroes = false

	if !model.Start.Equal(time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)) || !model.End.Equal(time.Date(2026, 3, 4, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("the converted dates followed the caller's change: start %v, end %v", model.Start, model.End)
	}
	if !model.RepairOptions.FixZeroes {
		t.Error("the converted repair options followed the caller's change")
	}
}
