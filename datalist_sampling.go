package insyra

import "time"

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
