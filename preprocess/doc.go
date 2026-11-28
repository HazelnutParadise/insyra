// Package preprocess turns a DataTable into the numbers a model reads: it
// scales numeric columns, encodes categorical columns and fills missing cells
// with values fitted on training data.
//
// Each step is a fitted object. A fit learns the parameters from one table,
// and Transform applies those same parameters to any other table, so a test set
// is scaled with statistics from the training set and never with its own:
//
//	train, scaler, err := preprocess.StandardScale(trainTable, insyra.Name("age"))
//	if err != nil {
//		return err
//	}
//	test, err := scaler.Transform(testTable) // uses the training mean and stdev
//
// The package is separate from the core because the core is the table and its
// operations, and a program that only reads, filters and summarises data does
// not need the fitted transformers. Those reach for this package, which in turn
// imports insyra; insyra never imports it. The functions take an
// insyra.IDataTable, so a table from the isr package works without conversion.
//
// The fill methods on the core (DataTable.FillForward, FillWithMean and the
// rest) are not here: they fill a table in place from the table's own values
// and keep no fitted state.
package preprocess
