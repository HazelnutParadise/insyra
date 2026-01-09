package datafetch

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/HazelnutParadise/insyra"
	"github.com/wnjoon/go-yfinance/pkg/models"
)

// YFOptionChainTables splits option chains into separate tables.
type YFOptionChainTables struct {
	Calls      *insyra.DataTable
	Puts       *insyra.DataTable
	Underlying *insyra.DataTable
	Expiration time.Time
}

// YFFinancialStatementTables provides multiple views of a statement.
type YFFinancialStatementTables struct {
	Values *insyra.DataTable
	Items  *insyra.DataTable
	Meta   *insyra.DataTable
}

func normalizeUnixSecondsColumns(dt *insyra.DataTable, names ...string) *insyra.DataTable {
	if dt == nil || len(names) == 0 {
		return dt
	}

	targets := make(map[string]struct{}, len(names))
	for _, name := range names {
		if name == "" {
			continue
		}
		targets[strings.ToLower(name)] = struct{}{}
	}
	if len(targets) == 0 {
		return dt
	}

	return dt.Map(func(rowIndex int, colIndex string, element any) any {
		name := strings.ToLower(dt.GetColNameByIndex(colIndex))
		if _, ok := targets[name]; !ok {
			return element
		}
		switch v := element.(type) {
		case int64:
			return time.Unix(v, 0)
		case int:
			return time.Unix(int64(v), 0)
		case int32:
			return time.Unix(int64(v), 0)
		case float64:
			return time.Unix(int64(v), 0)
		case float32:
			return time.Unix(int64(v), 0)
		default:
			return element
		}
	})
}

func optionChainLabel(date string, expiration time.Time) string {
	if date != "" {
		return date
	}
	if !expiration.IsZero() {
		return expiration.Format("2006-01-02")
	}
	return "Nearest"
}

func buildOptionChainTables(symbol, date string, chain *models.OptionChain) (*YFOptionChainTables, error) {
	if chain == nil {
		return nil, fmt.Errorf("yfinance: option chain is nil")
	}

	label := optionChainLabel(date, chain.Expiration)
	symbol = strings.ToUpper(symbol)

	callsDT, err := insyra.ReadJSON(chain.Calls)
	if err != nil {
		return nil, err
	}
	callsDT = normalizeUnixSecondsColumns(callsDT, "expiration", "lastTradeDate")
	callsDT.SetName(fmt.Sprintf("%s.OptionChain_Calls(%s)", symbol, label))

	putsDT, err := insyra.ReadJSON(chain.Puts)
	if err != nil {
		return nil, err
	}
	putsDT = normalizeUnixSecondsColumns(putsDT, "expiration", "lastTradeDate")
	putsDT.SetName(fmt.Sprintf("%s.OptionChain_Puts(%s)", symbol, label))

	var underlyingDT *insyra.DataTable
	if chain.Underlying != nil {
		underlyingDT, err = insyra.ReadJSON(chain.Underlying)
		if err != nil {
			return nil, err
		}
		underlyingDT = normalizeUnixSecondsColumns(underlyingDT, "regularMarketTime")
		underlyingDT.SetName(fmt.Sprintf("%s.OptionChain_Underlying(%s)", symbol, label))
	}

	return &YFOptionChainTables{
		Calls:      callsDT,
		Puts:       putsDT,
		Underlying: underlyingDT,
		Expiration: chain.Expiration,
	}, nil
}

func buildFinancialStatementTables(symbol, statement string, freq YFPeriod, stmt *models.FinancialStatement) (*YFFinancialStatementTables, error) {
	if stmt == nil {
		return nil, fmt.Errorf("yfinance: financial statement is nil")
	}

	symbol = strings.ToUpper(symbol)
	freqLabel := string(freq)

	fields := stmt.Fields()
	sort.Strings(fields)

	valuesDT := insyra.NewDataTable()
	if len(fields) > 0 || len(stmt.Dates) > 0 {
		dateCol := insyra.NewDataList()
		dateCol.SetName("AsOfDate")
		cols := []*insyra.DataList{dateCol}

		fieldCols := make(map[string]*insyra.DataList, len(fields))
		fieldDateValues := make(map[string]map[time.Time]float64, len(fields))
		for _, field := range fields {
			dl := insyra.NewDataList()
			dl.SetName(field)
			fieldCols[field] = dl
			cols = append(cols, dl)

			items := stmt.Data[field]
			if len(items) == 0 {
				continue
			}
			dateValues := make(map[time.Time]float64, len(items))
			for _, item := range items {
				dateValues[item.AsOfDate] = item.Value
			}
			fieldDateValues[field] = dateValues
		}

		for _, date := range stmt.Dates {
			dateCol.Append(date)
			for _, field := range fields {
				if dateValues, ok := fieldDateValues[field]; ok {
					if value, ok := dateValues[date]; ok {
						fieldCols[field].Append(value)
						continue
					}
				}
				fieldCols[field].Append(nil)
			}
		}

		valuesDT.AppendCols(cols...)
	}
	valuesDT.SetName(fmt.Sprintf("%s.%s_Values(%s)", symbol, statement, freqLabel))

	itemsDT := insyra.NewDataTable()
	fieldCol := insyra.NewDataList()
	fieldCol.SetName("Field")
	dateCol := insyra.NewDataList()
	dateCol.SetName("AsOfDate")
	currencyCol := insyra.NewDataList()
	currencyCol.SetName("CurrencyCode")
	periodCol := insyra.NewDataList()
	periodCol.SetName("PeriodType")
	valueCol := insyra.NewDataList()
	valueCol.SetName("Value")
	formattedCol := insyra.NewDataList()
	formattedCol.SetName("Formatted")

	itemsDT.AppendCols(fieldCol, dateCol, currencyCol, periodCol, valueCol, formattedCol)

	for _, field := range fields {
		for _, item := range stmt.Data[field] {
			fieldCol.Append(field)
			dateCol.Append(item.AsOfDate)
			currencyCol.Append(item.CurrencyCode)
			periodCol.Append(item.PeriodType)
			valueCol.Append(item.Value)
			formattedCol.Append(item.Formatted)
		}
	}
	itemsDT.SetName(fmt.Sprintf("%s.%s_Items(%s)", symbol, statement, freqLabel))

	metaDT := insyra.NewDataTable()
	metaDT.AppendRowsByColName(map[string]any{
		"Symbol":    symbol,
		"Statement": statement,
		"Frequency": freqLabel,
		"Currency":  stmt.Currency,
	})
	metaDT.SetName(fmt.Sprintf("%s.%s_Meta(%s)", symbol, statement, freqLabel))

	return &YFFinancialStatementTables{
		Values: valuesDT,
		Items:  itemsDT,
		Meta:   metaDT,
	}, nil
}
