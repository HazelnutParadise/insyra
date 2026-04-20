package stats

import (
	"errors"
	"fmt"
	"math"

	"github.com/HazelnutParadise/insyra"
	internalcluster "github.com/HazelnutParadise/insyra/stats/internal/clustering"
)

type KMeansOptions struct {
	NStart  int
	IterMax int
	Seed    *int64
}

type KMeansResult struct {
	Cluster     []int
	Centers     insyra.IDataTable
	TotSS       float64
	WithinSS    []float64
	TotWithinSS float64
	BetweenSS   float64
	Size        []int
	Iter        int
	IFault      int
}

type AgglomerativeMethod string

const (
	AggloComplete AgglomerativeMethod = "complete"
	AggloSingle   AgglomerativeMethod = "single"
	AggloAverage  AgglomerativeMethod = "average"
	AggloWardD    AgglomerativeMethod = "ward.D"
	AggloWardD2   AgglomerativeMethod = "ward.D2"
	AggloMcQuitty AgglomerativeMethod = "mcquitty"
	AggloMedian   AgglomerativeMethod = "median"
	AggloCentroid AgglomerativeMethod = "centroid"
)

type HierarchicalResult struct {
	Merge      [][2]int
	Height     []float64
	Order      []int
	Labels     []string
	Method     AgglomerativeMethod
	DistMethod string
}

type DBSCANOptions struct {
	BorderPoints *bool
}

type DBSCANResult struct {
	Cluster []int
	IsSeed  []bool
}

type SilhouettePoint struct {
	Cluster  int
	Neighbor int
	SilWidth float64
}

type SilhouetteResult struct {
	Points            []SilhouettePoint
	AverageSilhouette float64
}

func KMeans(dataTable insyra.IDataTable, centers int, opts ...KMeansOptions) (*KMeansResult, error) {
	data, _, err := numericMatrixFromTable(dataTable)
	if err != nil {
		return nil, err
	}
	if len(opts) > 1 {
		return nil, errors.New("opts accepts at most one value")
	}
	var options KMeansOptions
	if len(opts) == 1 {
		options = opts[0]
	}
	got, err := internalcluster.KMeans(data, centers, internalcluster.KMeansOptions(options))
	if err != nil {
		return nil, err
	}
	return &KMeansResult{
		Cluster:     append([]int(nil), got.Cluster...),
		Centers:     matrixToDataTable(got.Centers, "V"),
		TotSS:       got.TotSS,
		WithinSS:    append([]float64(nil), got.WithinSS...),
		TotWithinSS: got.TotWithinSS,
		BetweenSS:   got.BetweenSS,
		Size:        append([]int(nil), got.Size...),
		Iter:        got.Iter,
		IFault:      got.IFault,
	}, nil
}

func HierarchicalAgglomerative(dataTable insyra.IDataTable, method AgglomerativeMethod) (*HierarchicalResult, error) {
	data, labels, err := numericMatrixFromTable(dataTable)
	if err != nil {
		return nil, err
	}
	got, err := internalcluster.Hierarchical(data, labels, string(method))
	if err != nil {
		return nil, err
	}
	return &HierarchicalResult{
		Merge:      append([][2]int(nil), got.Merge...),
		Height:     append([]float64(nil), got.Height...),
		Order:      append([]int(nil), got.Order...),
		Labels:     append([]string(nil), got.Labels...),
		Method:     method,
		DistMethod: got.DistMethod,
	}, nil
}

func CutTreeByK(tree *HierarchicalResult, k int) ([]int, error) {
	if tree == nil {
		return nil, errors.New("tree must not be nil")
	}
	return internalcluster.CutTreeByK(&internalcluster.HierarchicalResult{
		Merge:      tree.Merge,
		Height:     tree.Height,
		Order:      tree.Order,
		Labels:     tree.Labels,
		DistMethod: tree.DistMethod,
	}, k)
}

func CutTreeByHeight(tree *HierarchicalResult, h float64) ([]int, error) {
	if tree == nil {
		return nil, errors.New("tree must not be nil")
	}
	return internalcluster.CutTreeByHeight(&internalcluster.HierarchicalResult{
		Merge:      tree.Merge,
		Height:     tree.Height,
		Order:      tree.Order,
		Labels:     tree.Labels,
		DistMethod: tree.DistMethod,
	}, h)
}

func DBSCAN(dataTable insyra.IDataTable, eps float64, minPts int, opts ...DBSCANOptions) (*DBSCANResult, error) {
	data, _, err := numericMatrixFromTable(dataTable)
	if err != nil {
		return nil, err
	}
	if len(opts) > 1 {
		return nil, errors.New("opts accepts at most one value")
	}
	var options DBSCANOptions
	if len(opts) == 1 {
		options = opts[0]
	}
	got, err := internalcluster.DBSCAN(data, eps, minPts, internalcluster.DBSCANOptions(options))
	if err != nil {
		return nil, err
	}
	return &DBSCANResult{
		Cluster: append([]int(nil), got.Cluster...),
		IsSeed:  append([]bool(nil), got.IsSeed...),
	}, nil
}

func Silhouette(dataTable insyra.IDataTable, labels insyra.IDataList) (*SilhouetteResult, error) {
	data, _, err := numericMatrixFromTable(dataTable)
	if err != nil {
		return nil, err
	}
	labs, err := intLabelsFromDataList(labels, len(data))
	if err != nil {
		return nil, err
	}
	got, err := internalcluster.Silhouette(data, labs)
	if err != nil {
		return nil, err
	}
	points := make([]SilhouettePoint, len(got.Points))
	for i, pt := range got.Points {
		points[i] = SilhouettePoint(pt)
	}
	return &SilhouetteResult{
		Points:            points,
		AverageSilhouette: got.Average,
	}, nil
}

func defaultInt(v, def int) int {
	if v <= 0 {
		return def
	}
	return v
}

func defaultSeed(seed *int64) int64 {
	if seed == nil {
		return 1
	}
	return *seed
}

func numericMatrixFromTable(dataTable insyra.IDataTable) ([][]float64, []string, error) {
	var rows, cols int
	var matrix [][]float64
	var labels []string
	dataTable.AtomicDo(func(dt *insyra.DataTable) {
		rows, cols = dt.Size()
		matrix = make([][]float64, rows)
		labels = make([]string, rows)
		for i := range rows {
			matrix[i] = make([]float64, cols)
			row := dt.GetRow(i)
			for j := range cols {
				v, ok := insyra.ToFloat64Safe(row.Get(j))
				if !ok || math.IsNaN(v) || math.IsInf(v, 0) {
					matrix = nil
					return
				}
				matrix[i][j] = v
			}
			if name, ok := dt.GetRowNameByIndex(i); ok && name != "" {
				labels[i] = name
			} else {
				labels[i] = fmt.Sprintf("%d", i+1)
			}
		}
	})
	if matrix == nil {
		return nil, nil, errors.New("input contains non-numeric values")
	}
	if rows < 1 || cols < 1 {
		return nil, nil, errors.New("input must have at least 1 row and 1 column")
	}
	return matrix, labels, nil
}

func intLabelsFromDataList(labels insyra.IDataList, expected int) ([]int, error) {
	out := make([]int, 0, expected)
	var size int
	labels.AtomicDo(func(dl *insyra.DataList) {
		size = dl.Len()
		for i := 0; i < dl.Len(); i++ {
			v, ok := insyra.ToFloat64Safe(dl.Get(i))
			if !ok || math.Trunc(v) != v {
				out = nil
				return
			}
			out = append(out, int(v))
		}
	})
	if out == nil {
		return nil, errors.New("labels must contain integer values")
	}
	if size != expected {
		return nil, errors.New("labels length must match row count")
	}
	return out, nil
}

func matrixToDataTable(rows [][]float64, colPrefix string) *insyra.DataTable {
	if len(rows) == 0 {
		return insyra.NewDataTable()
	}
	dt := insyra.NewDataTable()
	for c := range len(rows[0]) {
		col := insyra.NewDataList().SetName(fmt.Sprintf("%s%d", colPrefix, c+1))
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
