package insyra

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
	now := clock().UnixNano()
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

// Shuffled returns a copy of the DataList with its elements in random order,
// named after the list with "_Shuffled" appended, and leaves the list as it
// is. On the 0.4 line Shuffle reorders the list itself, so new code that wants
// a copy should call Shuffled.
func (dl *DataList) Shuffled(options ...SamplingOptions) *DataList {
	return dl.shuffledCopy("Shuffled", options)
}

// Shuffle returns a randomly reordered copy of the DataList and leaves the
// list as it is, exactly as Shuffled does. On the 0.4 line it shuffles the
// list in place and returns it, so call Shuffled where a copy is meant.
func (dl *DataList) Shuffle(options ...SamplingOptions) *DataList {
	return dl.shuffledCopy("Shuffle", options)
}

func (dl *DataList) shuffledCopy(funcName string, options []SamplingOptions) *DataList {
	var data []any
	var name string
	dl.AtomicDo(func(dl *DataList) {
		data = make([]any, len(dl.data))
		copy(data, dl.data)
		name = dl.name
	})
	if len(data) == 0 {
		dl.warn(funcName, "DataList is empty")
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
	now := clock().UnixNano()
	out.creationTimestamp = now
	out.lastModifiedTimestamp.Store(now)
	return out
}
