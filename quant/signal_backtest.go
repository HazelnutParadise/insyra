package quant

import (
	"errors"
	"fmt"
	"math"

	"github.com/HazelnutParadise/insyra"
)

// BacktestOptions are the optional settings of BacktestSignal.
type BacktestOptions struct {
	// CashReturns is the per-period return on the part of the portfolio not in
	// the asset. Nil means 0.
	CashReturns insyra.IDataList
	// BuyCost and SellCost are the costs of moving the position, as fractions of
	// the portfolio traded: moving from 0 to 1 costs BuyCost, from 1 to 0 costs
	// SellCost, and half a position costs half as much.
	BuyCost, SellCost float64
	// InitialPosition is the position held before the first row, from 0 to 1.
	// The first row then pays to move from it to the first effective position.
	InitialPosition float64
}

// BacktestResult is the result of BacktestSignal. All lists have one value per
// row of the input.
type BacktestResult struct {
	// NetReturns is the portfolio return after costs.
	NetReturns *insyra.DataList
	// Positions is the share of the portfolio in the asset during each row.
	Positions *insyra.DataList
	// Turnover is how much of the portfolio was traded at the start of each row,
	// the absolute change in position.
	Turnover *insyra.DataList
	// Trades is the number of rows on which the position changed.
	Trades int
}

// BacktestSignal turns a position signal into a net return series. signal[t] is
// the position, from 0 to 1, decided from the close of row t; it takes effect
// one row later, so the position during row t is signal[t-1] and the row-t asset
// return is never earned by a signal that saw it. Row 0 is held at
// InitialPosition.
//
// The return of row t is position * assetReturn + (1 - position) * cashReturn,
// less BuyCost times the position added and SellCost times the position removed
// at the start of that row. The portfolio is taken to be rebalanced to the
// target position every row, so drift between rows is ignored.
func BacktestSignal(signal, assetReturns insyra.IDataList, opts ...BacktestOptions) (*BacktestResult, error) {
	if len(opts) > 1 {
		return nil, errors.New("BacktestSignal accepts at most one BacktestOptions")
	}
	var o BacktestOptions
	if len(opts) == 1 {
		o = opts[0]
	}
	for name, c := range map[string]float64{"BuyCost": o.BuyCost, "SellCost": o.SellCost} {
		if math.IsNaN(c) || c < 0 || c >= 1 {
			return nil, fmt.Errorf("%s must be a fraction from 0 up to, not including, 1", name)
		}
	}
	if math.IsNaN(o.InitialPosition) || o.InitialPosition < 0 || o.InitialPosition > 1 {
		return nil, errors.New("InitialPosition must be from 0 to 1")
	}
	sig, err := numericSeries(signal, "signal")
	if err != nil {
		return nil, err
	}
	for i, s := range sig {
		if s < 0 || s > 1 {
			return nil, fmt.Errorf("signal holds %v at row index %d; a position must be from 0 to 1", s, i)
		}
	}
	asset, err := numericSeries(assetReturns, "assetReturns")
	if err != nil {
		return nil, err
	}
	if len(sig) != len(asset) {
		return nil, fmt.Errorf("signal has %d rows and assetReturns has %d", len(sig), len(asset))
	}
	cash := make([]float64, len(asset))
	if o.CashReturns != nil {
		if cash, err = numericSeries(o.CashReturns, "CashReturns"); err != nil {
			return nil, err
		}
		if len(cash) != len(asset) {
			return nil, fmt.Errorf("CashReturns has %d rows and assetReturns has %d", len(cash), len(asset))
		}
	}

	net, pos, turn := make([]any, len(sig)), make([]any, len(sig)), make([]any, len(sig))
	res := &BacktestResult{}
	previous := o.InitialPosition
	for t := range sig {
		held := o.InitialPosition
		if t > 0 {
			held = sig[t-1]
		}
		change := held - previous
		cost := 0.0
		if change > 0 {
			cost = change * o.BuyCost
		} else {
			cost = -change * o.SellCost
		}
		if change != 0 {
			res.Trades++
		}
		net[t] = held*asset[t] + (1-held)*cash[t] - cost
		pos[t] = held
		turn[t] = math.Abs(change)
		previous = held
	}
	res.NetReturns = insyra.NewDataList(net...)
	res.Positions = insyra.NewDataList(pos...)
	res.Turnover = insyra.NewDataList(turn...)
	return res, nil
}
