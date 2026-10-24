package datafetch

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/datafetch/internal/biff"
	"github.com/xuri/excelize/v2"
)

var (
	jpxLabelPattern = regexp.MustCompile(`(\d{4})年(\d{1,2})月`)
	jpxRangePattern = regexp.MustCompile(`(\d{1,2})/(\d{1,2})\D+(\d{1,2})/(\d{1,2})`)
)

// jpxLegacyLeaves maps the Japanese label of a three-row block (sales,
// purchases, total) to the investor group it holds.
var jpxLegacyLeaves = map[string]string{
	"証券会社":    "Securities Cos.",
	"投資信託":    "Investment Trusts",
	"事業法人":    "Business Cos.",
	"その他法人等":  "Other Cos.",
	"生保・損保":   "Life & Non-Life",
	"都銀・地銀等":  "City & Regional BK",
	"信託銀行":    "Trust BK",
	"その他金融機関": "Other Financials",
}

func jpxClean(s string) string {
	return strings.Map(func(r rune) rune {
		if r == ' ' || r == '　' || r == '\n' || r == '\r' || r == ' ' {
			return -1
		}
		return r
	}, s)
}

func jpxCellInt(sheet *biff.Sheet, row, col int) (int64, bool) {
	c, ok := sheet.Cell(row, col)
	if !ok {
		return 0, false
	}
	if c.IsNumber {
		return int64(c.Number), true
	}
	return jpxNumber(c.Text)
}

func jpxCellText(sheet *biff.Sheet, row, col int) string {
	c, ok := sheet.Cell(row, col)
	if !ok || c.IsNumber {
		return ""
	}
	return c.Text
}

// parseJPXLegacy reads the report week of every market sheet of an .xls file.
func parseJPXLegacy(data []byte, unit JPXUnit) ([]jpxRow, error) {
	wb, err := biff.Read(data)
	if err != nil {
		return nil, err
	}
	var rows []jpxRow
	for _, sheet := range wb.Sheets {
		part, err := parseJPXLegacySheet(sheet)
		if err != nil {
			return nil, fmt.Errorf("sheet %q: %w", sheet.Name, err)
		}
		rows = append(rows, part...)
	}
	return rows, nil
}

// jpxWeekDates turns the "(MM/DD～MM/DD)" text of a week and the year and month
// of the file's label into dates. A week at the start of January can begin in
// December, and one at the end of December can end in January.
func jpxWeekDates(rangeText string, labelYear, labelMonth int) (start, end time.Time, ok bool) {
	m := jpxRangePattern.FindStringSubmatch(rangeText)
	if m == nil {
		return start, end, false
	}
	n := func(i int) int { v, _ := strconv.Atoi(m[i]); return v }
	startMonth, startDay, endMonth, endDay := n(1), n(2), n(3), n(4)
	startYear := labelYear
	switch {
	case startMonth-labelMonth > 6:
		startYear--
	case labelMonth-startMonth > 6:
		startYear++
	}
	endYear := startYear
	if endMonth < startMonth {
		endYear++
	}
	start = time.Date(startYear, time.Month(startMonth), startDay, 0, 0, 0, 0, time.UTC)
	end = time.Date(endYear, time.Month(endMonth), endDay, 0, 0, 0, 0, time.UTC)
	return start, end, true
}

func parseJPXLegacySheet(sheet *biff.Sheet) ([]jpxRow, error) {
	label := jpxLabelPattern.FindStringSubmatch(jpxCellText(sheet, 3, 0))
	if label == nil {
		return nil, nil // a sheet with no figures, such as a market not yet listed
	}
	year, _ := strconv.Atoi(label[1])
	month, _ := strconv.Atoi(label[2])
	start, end, ok := jpxWeekDates(jpxCellText(sheet, 10, 7), year, month)
	if !ok {
		return nil, fmt.Errorf("the week of the report is not where it was (%q)", jpxCellText(sheet, 10, 7))
	}
	market := strings.TrimSpace(sheet.Name)

	type pair struct {
		sales, purchases int64
		ok               bool
	}
	found := map[string]pair{}
	aggregates := map[string]pair{} // 個人, 海外投資家 and 自己計 for the checks
	for r := 12; r < sheet.Rows(); r++ {
		jp := jpxClean(jpxCellText(sheet, r, 0))
		if jp == "" {
			continue
		}
		switch {
		case jpxClean(jpxCellText(sheet, r, 1)) == "売り":
			sales, ok1 := jpxCellInt(sheet, r, 8)
			purchases, ok2 := jpxCellInt(sheet, r+1, 8)
			p := pair{sales, purchases, ok1 && ok2}
			if investor, leaf := jpxLegacyLeaves[jp]; leaf {
				found[investor] = p
			} else {
				aggregates[jp] = p
			}
		case strings.Contains(jp, "個人現金"):
			found["Individuals Cash"] = legacyPair(sheet, r, 2, 4)
		case strings.Contains(jp, "個人信用"):
			found["Individuals Margin"] = legacyPair(sheet, r, 2, 4)
		case strings.Contains(jp, "自己現金"):
			found["Proprietary Cash"] = legacyPair(sheet, r, 2, 4)
		case strings.Contains(jp, "自己信用"):
			found["Proprietary Margin"] = legacyPair(sheet, r, 2, 4)
		case strings.HasPrefix(jp, "海外投資家売買における"):
			for k := 1; k <= 3 && r+k < sheet.Rows(); k++ {
				switch next := jpxClean(jpxCellText(sheet, r+k, 0)); {
				case strings.HasPrefix(next, "法人"):
					found["Foreigners Institutions"] = legacyPair(sheet, r+k, 3, 5)
				case strings.HasPrefix(next, "個人"):
					found["Foreigners Individuals"] = legacyPair(sheet, r+k, 3, 5)
				}
			}
		}
	}

	var rows []jpxRow
	for _, investor := range jpxInvestors {
		if p, ok := found[investor]; ok && p.ok {
			rows = append(rows, newJPXRow(start, end, market, investor, p.sales, p.purchases))
		}
	}
	// The split rows must add up to the blocks they split; a mismatch means the
	// layout moved, or the JPX corrected one figure and not the other.
	for _, check := range []struct {
		aggregate string
		parts     [2]string
	}{
		{"個人", [2]string{"Individuals Cash", "Individuals Margin"}},
		{"海外投資家", [2]string{"Foreigners Institutions", "Foreigners Individuals"}},
		{"自己計", [2]string{"Proprietary Cash", "Proprietary Margin"}},
	} {
		agg, a, b := aggregates[check.aggregate], found[check.parts[0]], found[check.parts[1]]
		if agg.ok && a.ok && b.ok && (a.sales+b.sales != agg.sales || a.purchases+b.purchases != agg.purchases) {
			insyra.LogWarning("datafetch", "JPX", "%s %s: %s and %s do not add up to the block they split", market, start.Format("2006-01-02"), check.parts[0], check.parts[1])
		}
	}
	return rows, nil
}

func legacyPair(sheet *biff.Sheet, row, salesCol, purchasesCol int) (p struct {
	sales, purchases int64
	ok               bool
}) {
	s, ok1 := jpxCellInt(sheet, row, salesCol)
	b, ok2 := jpxCellInt(sheet, row, purchasesCol)
	p.sales, p.purchases, p.ok = s, b, ok1 && ok2
	return p
}

// jpxWeeklyLeaves are the English fragments the leaf header of each of the
// fourteen groups of the weekly workbook must contain, in order.
var jpxWeeklyLeaves = []string{
	"Cash", "Margin", "Cash", "Margin", "Institutions", "Individuals",
	"Securities Cos.", "Investment Trusts", "Business Cos.", "Other Cos.",
	"Life & Non-Life", "City & Regional BK", "Trust BK", "Other Financials",
}

// parseJPXWeekly reads the one-sheet workbook the JPX publishes for each week
// since September 2026: per market a shares row and a value row, and for each of
// fourteen groups four columns (sales, purchases, balance, total) from column D.
func parseJPXWeekly(data []byte, start, end time.Time, unit JPXUnit) ([]jpxRow, error) {
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, fmt.Errorf("the workbook has no sheet")
	}
	table, err := f.GetRows(sheets[0])
	if err != nil {
		return nil, err
	}
	if len(table) < 8 {
		return nil, fmt.Errorf("the sheet has %d rows, fewer than the layout needs", len(table))
	}
	cellAt := func(row, col int) string {
		if row >= len(table) || col >= len(table[row]) {
			return ""
		}
		return table[row][col]
	}
	for k, want := range jpxWeeklyLeaves {
		if got := cellAt(5, 3+4*k); !strings.Contains(got, want) {
			return nil, fmt.Errorf("the header of group %d is %q, expected one containing %q: the layout has changed", k, got, want)
		}
	}

	wanted := "Shares"
	if unit == JPXUnitValue {
		wanted = "Value"
	}
	var rows []jpxRow
	market := ""
	for r := 7; r < len(table); r++ {
		if m := strings.TrimSpace(cellAt(r, 1)); m != "" {
			market = jpxWeeklyMarket(m)
		}
		if !strings.Contains(cellAt(r, 2), wanted) || market == "" {
			continue
		}
		for k, investor := range jpxInvestors {
			col := 3 + 4*k
			sales, ok1 := jpxNumber(cellAt(r, col))
			purchases, ok2 := jpxNumber(cellAt(r, col+1))
			if !ok1 || !ok2 {
				continue
			}
			row := newJPXRow(start, end, market, investor, sales, purchases)
			if balance, ok := jpxNumber(cellAt(r, col+2)); ok && balance != row.balance {
				insyra.LogWarning("datafetch", "JPX", "%s %s %s: the balance %d is not purchases less sales", market, start.Format("2006-01-02"), investor, balance)
			}
			rows = append(rows, row)
		}
	}
	return rows, nil
}

// jpxWeeklyMarket turns "東証プライム\nTSE Prime Market" into "TSE Prime".
func jpxWeeklyMarket(cell string) string {
	lines := strings.Split(strings.ReplaceAll(cell, "\r", ""), "\n")
	name := strings.TrimSpace(lines[len(lines)-1])
	name = strings.TrimSuffix(name, " Markets")
	name = strings.TrimSuffix(name, " Market")
	return name
}
