package stats

import (
	"errors"
	"fmt"

	"github.com/HazelnutParadise/insyra"
	internalknn "github.com/HazelnutParadise/insyra/stats/internal/knn"
)

type KNNWeighting string

const (
	KNNUniformWeighting  KNNWeighting = "uniform"
	KNNDistanceWeighting KNNWeighting = "distance"
)

type KNNOptions struct {
	Weighting KNNWeighting
}

type KNNClassificationResult struct {
	Predictions   insyra.IDataList
	Classes       insyra.IDataList
	Probabilities insyra.IDataTable
}

type KNNRegressionResult struct {
	Predictions []float64
}

func KNNClassify(trainData insyra.IDataTable, trainLabels insyra.IDataList, testData insyra.IDataTable, k int, opts ...KNNOptions) (*KNNClassificationResult, error) {
	train, _, err := numericMatrixFromTable(trainData)
	if err != nil {
		return nil, err
	}
	test, _, err := numericMatrixFromTable(testData)
	if err != nil {
		return nil, err
	}
	if len(opts) > 1 {
		return nil, errors.New("opts accepts at most one value")
	}
	var options KNNOptions
	if len(opts) == 1 {
		options = opts[0]
	}
	labelInfo, err := categoricalLabelsFromDataList(trainLabels, len(train))
	if err != nil {
		return nil, err
	}
	got, err := internalknn.Classify(train, test, labelInfo.encoded, k, internalknn.Options{
		Weighting: internalknn.Weighting(options.Weighting),
	})
	if err != nil {
		return nil, err
	}

	predictions := insyra.NewDataList()
	for _, encoded := range got.Predictions {
		predictions.Append(labelInfo.values[encoded])
	}

	classes := insyra.NewDataList()
	columnNames := make([]string, len(got.Classes))
	for i, encoded := range got.Classes {
		classes.Append(labelInfo.values[encoded])
		columnNames[i] = labelInfo.names[encoded]
	}

	return &KNNClassificationResult{
		Predictions:   predictions,
		Classes:       classes,
		Probabilities: matrixToNamedDataTable(got.Probabilities, columnNames),
	}, nil
}

func KNNRegress(trainData insyra.IDataTable, trainTargets insyra.IDataList, testData insyra.IDataTable, k int, opts ...KNNOptions) (*KNNRegressionResult, error) {
	train, _, err := numericMatrixFromTable(trainData)
	if err != nil {
		return nil, err
	}
	test, _, err := numericMatrixFromTable(testData)
	if err != nil {
		return nil, err
	}
	if len(opts) > 1 {
		return nil, errors.New("opts accepts at most one value")
	}
	var options KNNOptions
	if len(opts) == 1 {
		options = opts[0]
	}
	targets, err := numericVectorFromDataList(trainTargets, len(train))
	if err != nil {
		return nil, err
	}
	got, err := internalknn.Regress(train, test, targets, k, internalknn.Options{
		Weighting: internalknn.Weighting(options.Weighting),
	})
	if err != nil {
		return nil, err
	}
	return &KNNRegressionResult{
		Predictions: append([]float64(nil), got.Predictions...),
	}, nil
}

type categoricalLabelSet struct {
	encoded []string
	values  map[string]any
	names   map[string]string
}

func categoricalLabelsFromDataList(labels insyra.IDataList, expected int) (*categoricalLabelSet, error) {
	out := &categoricalLabelSet{
		encoded: make([]string, 0, expected),
		values:  make(map[string]any, expected),
		names:   make(map[string]string, expected),
	}
	var size int
	labels.AtomicDo(func(dl *insyra.DataList) {
		size = dl.Len()
		for i := 0; i < dl.Len(); i++ {
			value := dl.Get(i)
			if value == nil {
				out.encoded = nil
				return
			}
			key := labelKey(value)
			out.encoded = append(out.encoded, key)
			if _, ok := out.values[key]; !ok {
				out.values[key] = value
				out.names[key] = fmt.Sprint(value)
			}
		}
	})
	if out.encoded == nil {
		return nil, errors.New("labels must not contain nil values")
	}
	if size != expected {
		return nil, errors.New("labels length must match training row count")
	}
	return out, nil
}

func numericVectorFromDataList(values insyra.IDataList, expected int) ([]float64, error) {
	out := make([]float64, 0, expected)
	var size int
	values.AtomicDo(func(dl *insyra.DataList) {
		size = dl.Len()
		for i := 0; i < dl.Len(); i++ {
			v, ok := insyra.ToFloat64Safe(dl.Get(i))
			if !ok {
				out = nil
				return
			}
			out = append(out, v)
		}
	})
	if out == nil {
		return nil, errors.New("targets must contain numeric values")
	}
	if size != expected {
		return nil, errors.New("targets length must match training row count")
	}
	return out, nil
}

func labelKey(v any) string {
	return fmt.Sprintf("%T:%#v", v, v)
}

func matrixToNamedDataTable(rows [][]float64, columnNames []string) *insyra.DataTable {
	if len(rows) == 0 {
		return insyra.NewDataTable()
	}
	dt := insyra.NewDataTable()
	for c := range len(rows[0]) {
		name := fmt.Sprintf("V%d", c+1)
		if c < len(columnNames) && columnNames[c] != "" {
			name = columnNames[c]
		}
		col := insyra.NewDataList().SetName(name)
		for r := range len(rows) {
			col.Append(rows[r][c])
		}
		dt.AppendCols(col)
	}
	rowNames := make([]string, len(rows))
	for i := range rowNames {
		rowNames[i] = fmt.Sprintf("%d", i+1)
	}
	dt.SetRowNames(rowNames)
	return dt
}
