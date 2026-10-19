package datafetch

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/HazelnutParadise/insyra"
)

// shareholdingLevelLabels names the holding levels of the TDCC shareholding
// distribution table. Levels 1 to 15 are ranges of shares held; 16 is the
// difference adjustment the TDCC publishes so the levels add up, and 17 the
// total.
var shareholdingLevelLabels = map[int64]string{
	1: "1-999", 2: "1,000-5,000", 3: "5,001-10,000", 4: "10,001-15,000", 5: "15,001-20,000",
	6: "20,001-30,000", 7: "30,001-40,000", 8: "40,001-50,000", 9: "50,001-100,000",
	10: "100,001-200,000", 11: "200,001-400,000", 12: "400,001-600,000", 13: "600,001-800,000",
	14: "800,001-1,000,000", 15: "1,000,001 and above", 16: "difference adjustment", 17: "total",
}

// ShareholdingDistribution returns the Taiwan Depository & Clearing Corporation
// (TDCC) shareholding distribution of every security for the latest week. It
// is ShareholdingDistributionContext with context.Background().
func (t *TWStockClient) ShareholdingDistribution() (*insyra.DataTable, error) {
	return t.ShareholdingDistributionContext(context.Background())
}

// ShareholdingDistributionContext is ShareholdingDistribution with a context.
// The limiter wait, the request and each retry backoff stop when ctx is done,
// and the call then returns ctx.Err() without sending another request.
//
// The table has one row per security and holding level with Date (the week's
// data date, a time.Time), Code (string), Level (int64, 1 to 17), LevelLabel
// (string), Holders and Shares (int64) and Percent (float64, the share of the
// securities held in custody). Levels 1 to 15 are ranges of shares held, from
// 1-999 up to 1,000,001 and above; level 16 is the difference adjustment the
// TDCC publishes and level 17 the total.
//
// The TDCC publishes only the latest week and no history, so keep each week's
// result if you need a series. About 69,000 rows come back in one request.
func (t *TWStockClient) ShareholdingDistributionContext(ctx context.Context) (*insyra.DataTable, error) {
	if err := contextErr(ctx); err != nil {
		return nil, err
	}
	if err := t.usable(); err != nil {
		return nil, err
	}
	var rows [][]string
	err := t.doGet(ctx, requestURL(t.tdccBaseURL, "/getOD.ashx", url.Values{"id": {"1-5"}}), func(resp *http.Response) error {
		body, err := readBodyResponse(resp, 64<<20)
		if err != nil {
			return err
		}
		rows, err = readShareholdingCSV(body)
		return err
	})
	if err != nil {
		return nil, err
	}
	return shareholdingTable(rows)
}

// readShareholdingCSV checks the header of the TDCC file and returns its rows.
func readShareholdingCSV(body []byte) ([][]string, error) {
	body = bytes.TrimPrefix(body, []byte("\xef\xbb\xbf"))
	reader := csv.NewReader(bytes.NewReader(body))
	reader.FieldsPerRecord = -1
	var rows [][]string
	for {
		record, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("datafetch: TDCC shareholding file: %w", err)
		}
		rows = append(rows, record)
	}
	want := []string{"資料日期", "證券代號", "持股分級", "人數", "股數"}
	if len(rows) == 0 || len(rows[0]) < 6 {
		return nil, errors.New("datafetch: TDCC shareholding file has no header")
	}
	for i, name := range want {
		if strings.TrimSpace(rows[0][i]) != name {
			return nil, fmt.Errorf("datafetch: required header %q is missing", name)
		}
	}
	return rows[1:], nil
}

func shareholdingTable(rows [][]string) (*insyra.DataTable, error) {
	n := len(rows)
	dates, codes, levels := make([]any, n), make([]any, n), make([]any, n)
	labels, holders, shares, percent := make([]any, n), make([]any, n), make([]any, n), make([]any, n)
	for i, row := range rows {
		if len(row) < 6 {
			return nil, fmt.Errorf("datafetch: TDCC shareholding row %d is missing cells", i)
		}
		date, err := time.Parse("20060102", strings.TrimSpace(row[0]))
		if err != nil {
			return nil, fmt.Errorf("datafetch: TDCC shareholding row %d has date %q: %w", i, row[0], err)
		}
		level, err := strconv.ParseInt(strings.TrimSpace(row[2]), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("datafetch: TDCC shareholding row %d has level %q", i, row[2])
		}
		dates[i], codes[i], levels[i] = date, strings.TrimSpace(row[1]), level
		labels[i] = shareholdingLevelLabels[level]
		holders[i], shares[i], percent[i] = parseInt(row[3]), parseInt(row[4]), parseFloat(row[5])
	}
	return insyra.NewDataTable(
		newNamedCol("Date", dates), newNamedCol("Code", codes), newNamedCol("Level", levels),
		newNamedCol("LevelLabel", labels), newNamedCol("Holders", holders), newNamedCol("Shares", shares),
		newNamedCol("Percent", percent),
	), nil
}
