// Package ta holds the technical-analysis tools of Insyra: moving averages,
// MACD, Bollinger bands, ATR, KD, RSI, volume and market-breadth indicators,
// the bias ratio, turning points (ZigZag) and market regimes, chart patterns,
// fan lines, price gaps, and a shuffled-price random-walk control.
//
// Technical analysis is bullshit. These tools are here so you can test that
// claim yourself, not because they make money: in the study behind this
// package (October 2026, Taiwan and US stocks and 47 national indexes, costs
// deducted, scored only on 2015 to 2026) following an indicator stock by stock
// beat buy-and-hold for 0% to 0.1% of stocks, and no whole-market timing rule
// beat it significantly. See Docs/quant-ta.md for the findings per tool.
//
// Every function takes its series as insyra.IDataList and returns an error for
// invalid input, naming the row index. The tools use only information known at
// the close of each row, so a series cut short gives the results of the full
// series for the rows before the cut. A row without enough history is NaN, and
// SegmentStarts, where a function takes it, lists the rows on which a new data
// segment starts (a trading halt, a relisting, a gap in the data) so nothing is
// carried across.
package ta
