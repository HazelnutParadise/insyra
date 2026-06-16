package insyra

import (
	"math"
	"math/rand/v2"
	"time"

	"github.com/HazelnutParadise/insyra/internal/core"
)

const samplingSeedXor uint64 = 0x9E3779B97F4A7C15

// SamplingOptions configures random sampling and train/test splitting.
type SamplingOptions struct {
	Seed          uint64
	UseSeed       bool
	PreserveOrder bool
}

type samplingRandom struct {
	rng *rand.Rand
}

func newSamplingRandom(options []SamplingOptions) samplingRandom {
	var opts SamplingOptions
	if len(options) > 0 {
		opts = options[0]
	}
	if opts.UseSeed {
		return samplingRandom{rng: rand.New(rand.NewPCG(opts.Seed, opts.Seed^samplingSeedXor))}
	}
	return samplingRandom{}
}

func samplingOptions(options []SamplingOptions) SamplingOptions {
	if len(options) > 0 {
		return options[0]
	}
	return SamplingOptions{}
}

func (sr samplingRandom) perm(n int) []int {
	if sr.rng != nil {
		return sr.rng.Perm(n)
	}
	return rand.Perm(n)
}

func (sr samplingRandom) intN(n int) int {
	if sr.rng != nil {
		return sr.rng.IntN(n)
	}
	return rand.IntN(n)
}

func sampleIndexSet(size, n int, withReplacement bool, rng samplingRandom) []int {
	indices := make([]int, n)
	if withReplacement {
		for i := range n {
			indices[i] = rng.intN(size)
		}
		return indices
	}
	perm := rng.perm(size)
	copy(indices, perm[:n])
	return indices
}

func fracCount(length int, frac float64) int {
	n := int(math.Floor(float64(length) * frac))
	if n == 0 && length > 0 && frac > 0 {
		return 1
	}
	return n
}

// Sample returns a new DataList containing n randomly selected elements.
func (dl *DataList) Sample(n int, withReplacement bool, options ...SamplingOptions) *DataList {
	var data []any
	var name string
	dl.AtomicDo(func(dl *DataList) {
		data = make([]any, len(dl.data))
		copy(data, dl.data)
		name = dl.name
	})
	if n <= 0 {
		dl.warn("Sample", "n must be > 0")
		return NewDataList()
	}
	if len(data) == 0 {
		dl.warn("Sample", "DataList is empty")
		return NewDataList()
	}
	if !withReplacement && n > len(data) {
		dl.warn("Sample", "n cannot exceed DataList length when sampling without replacement")
		return NewDataList()
	}

	rng := newSamplingRandom(options)
	indices := sampleIndexSet(len(data), n, withReplacement, rng)
	out := NewDataList()
	out.name = name + "_Sampled"
	out.data = make([]any, len(indices))
	for i, idx := range indices {
		out.data[i] = data[idx]
	}
	now := time.Now().Unix()
	out.creationTimestamp = now
	out.lastModifiedTimestamp.Store(now)
	return out
}

// SampleFrac returns a new DataList containing frac of the elements.
func (dl *DataList) SampleFrac(frac float64, withReplacement bool, options ...SamplingOptions) *DataList {
	length := dl.Len()
	if frac <= 0 || frac > 1 {
		dl.warn("SampleFrac", "frac must be in (0, 1]")
		return NewDataList()
	}
	if length == 0 {
		dl.warn("SampleFrac", "DataList is empty")
		return NewDataList()
	}
	return dl.Sample(fracCount(length, frac), withReplacement, options...)
}

// Shuffle returns a randomly reordered copy of the DataList.
func (dl *DataList) Shuffle(options ...SamplingOptions) *DataList {
	var data []any
	var name string
	dl.AtomicDo(func(dl *DataList) {
		data = make([]any, len(dl.data))
		copy(data, dl.data)
		name = dl.name
	})
	if len(data) == 0 {
		dl.warn("Shuffle", "DataList is empty")
		return NewDataList()
	}

	rng := newSamplingRandom(options)
	indices := rng.perm(len(data))
	out := NewDataList()
	out.name = name + "_Shuffled"
	out.data = make([]any, len(indices))
	for i, idx := range indices {
		out.data[i] = data[idx]
	}
	now := time.Now().Unix()
	out.creationTimestamp = now
	out.lastModifiedTimestamp.Store(now)
	return out
}

type dataTableSamplingSnapshot struct {
	name     string
	colData  [][]any
	colNames []string
	rowNames *core.BiIndex
	rows     int
}

func (dt *DataTable) snapshotForSampling() dataTableSamplingSnapshot {
	var snap dataTableSamplingSnapshot
	dt.AtomicDo(func(dt *DataTable) {
		snap.name = dt.name
		snap.rows = dt.getMaxColLength()
		snap.colData = make([][]any, len(dt.columns))
		snap.colNames = make([]string, len(dt.columns))
		for i, col := range dt.columns {
			snap.colNames[i] = col.name
			snap.colData[i] = make([]any, len(col.data))
			copy(snap.colData[i], col.data)
		}
		if dt.rowNames != nil {
			snap.rowNames = dt.rowNames.Clone()
		} else {
			snap.rowNames = core.NewBiIndex(0)
		}
	})
	return snap
}

func dataTableFromSampledRows(snap dataTableSamplingSnapshot, indices []int, suffix string) *DataTable {
	now := time.Now().Unix()
	out := &DataTable{
		columns:           make([]*DataList, len(snap.colData)),
		rowNames:          core.NewBiIndex(0),
		name:              snap.name + suffix,
		creationTimestamp: now,
	}
	out.lastModifiedTimestamp.Store(now)
	for colIdx, col := range snap.colData {
		dl := &DataList{
			data:              make([]any, len(indices)),
			name:              snap.colNames[colIdx],
			creationTimestamp: now,
		}
		dl.lastModifiedTimestamp.Store(now)
		for rowIdx, srcIdx := range indices {
			if srcIdx >= 0 && srcIdx < len(col) {
				dl.data[rowIdx] = col[srcIdx]
			}
		}
		out.columns[colIdx] = dl
	}
	if snap.rowNames != nil && snap.rowNames.Len() > 0 {
		for newIdx, oldIdx := range indices {
			name, ok := snap.rowNames.Get(oldIdx)
			if !ok || name == "" {
				continue
			}
			_, _ = out.rowNames.Set(newIdx, safeRowName(out, name))
		}
	}
	return out
}

// Sample returns a new DataTable containing n randomly selected rows.
func (dt *DataTable) Sample(n int, withReplacement bool, options ...SamplingOptions) *DataTable {
	snap := dt.snapshotForSampling()
	if n <= 0 {
		dt.warn("Sample", "n must be > 0")
		return NewDataTable()
	}
	if snap.rows == 0 {
		dt.warn("Sample", "DataTable is empty")
		return NewDataTable()
	}
	if !withReplacement && n > snap.rows {
		dt.warn("Sample", "n cannot exceed DataTable row count when sampling without replacement")
		return NewDataTable()
	}

	rng := newSamplingRandom(options)
	indices := sampleIndexSet(snap.rows, n, withReplacement, rng)
	return dataTableFromSampledRows(snap, indices, "_Sampled")
}

// SampleFrac returns a new DataTable containing frac of the rows.
func (dt *DataTable) SampleFrac(frac float64, withReplacement bool, options ...SamplingOptions) *DataTable {
	rows := dt.NumRows()
	if frac <= 0 || frac > 1 {
		dt.warn("SampleFrac", "frac must be in (0, 1]")
		return NewDataTable()
	}
	if rows == 0 {
		dt.warn("SampleFrac", "DataTable is empty")
		return NewDataTable()
	}
	return dt.Sample(fracCount(rows, frac), withReplacement, options...)
}

// Shuffle returns a new DataTable with rows randomly reordered.
func (dt *DataTable) Shuffle(options ...SamplingOptions) *DataTable {
	snap := dt.snapshotForSampling()
	if snap.rows == 0 {
		dt.warn("Shuffle", "DataTable is empty")
		return NewDataTable()
	}
	rng := newSamplingRandom(options)
	return dataTableFromSampledRows(snap, rng.perm(snap.rows), "_Shuffled")
}

// TrainTestSplit splits the DataTable into train and test tables.
func (dt *DataTable) TrainTestSplit(trainFrac float64, options ...SamplingOptions) (*DataTable, *DataTable) {
	snap := dt.snapshotForSampling()
	if trainFrac <= 0 || trainFrac > 1 {
		dt.warn("TrainTestSplit", "trainFrac must be in (0, 1]")
		return NewDataTable(), NewDataTable()
	}
	if snap.rows == 0 {
		dt.warn("TrainTestSplit", "DataTable is empty")
		return NewDataTable(), NewDataTable()
	}

	opts := samplingOptions(options)
	indices := make([]int, snap.rows)
	if opts.PreserveOrder {
		for i := range snap.rows {
			indices[i] = i
		}
	} else {
		rng := newSamplingRandom(options)
		indices = rng.perm(snap.rows)
	}
	trainN := fracCount(snap.rows, trainFrac)
	train := dataTableFromSampledRows(snap, indices[:trainN], "_Train")
	test := dataTableFromSampledRows(snap, indices[trainN:], "_Test")
	return train, test
}

// SimpleRandomSample returns a new DataTable containing a simple random sample of the specified size.
// If sampleSize is greater than the number of rows in the DataTable, it returns a copy of the original DataTable.
// If sampleSize is less than or equal to 0, it returns an empty DataTable.
func (dt *DataTable) SimpleRandomSample(sampleSize int) *DataTable {
	if sampleSize <= 0 {
		dt.warn("SimpleRandomSample", "Sample size is less than or equal to 0. Returning an empty DataTable.")
		return NewDataTable()
	}
	var colNames []string
	sampledRows := []*DataList{}
	ndt := NewDataTable()
	var earlyResult *DataTable
	dt.AtomicDo(func(table *DataTable) {
		numRows, _ := table.Size()
		if sampleSize >= numRows {
			dt.warn("SimpleRandomSample", "Sample size is greater than or equal to the number of rows. Returning a copy of the original DataTable.")
			earlyResult = table.Clone()
			return
		}
		indices := rand.Perm(numRows)[:sampleSize]
		colNames = table.ColNames()

		ndt.SetName(table.GetName() + "_Sampled")
		for _, idx := range indices {
			sampledRows = append(sampledRows, table.GetRow(idx))
		}
	})
	if earlyResult != nil {
		return earlyResult
	}
	ndt.SetColNames(colNames)
	ndt.AppendRowsFromDataList(sampledRows...)

	return ndt
}
