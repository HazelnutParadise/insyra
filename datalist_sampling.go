package insyra

// Sample returns a new DataList containing n randomly selected elements.
func (dl *DataList) Sample(n int, withReplacement bool, options ...SamplingOptions) *DataList {
	if msg := extraOptional("SamplingOptions", len(options)); msg != "" {
		dl.fail("Sample", "%s", msg)
		return dl.failedResult()
	}
	var data []any
	var name string
	dl.AtomicDo(func(dl *DataList) {
		data = make([]any, len(dl.data))
		copy(data, dl.data)
		name = dl.name
	})
	if n <= 0 {
		dl.fail("Sample", "n must be > 0")
		return dl.failedResult()
	}
	if len(data) == 0 {
		dl.warn("Sample", "DataList is empty")
		return NewDataList()
	}
	if !withReplacement && n > len(data) {
		dl.fail("Sample", "n cannot exceed DataList length when sampling without replacement")
		return dl.failedResult()
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
	if msg := extraOptional("SamplingOptions", len(options)); msg != "" {
		dl.fail("SampleFrac", "%s", msg)
		return dl.failedResult()
	}
	length := dl.Len()
	if frac <= 0 || frac > 1 {
		dl.fail("SampleFrac", "frac must be in (0, 1]")
		return dl.failedResult()
	}
	if length == 0 {
		dl.warn("SampleFrac", "DataList is empty")
		return NewDataList()
	}
	return dl.Sample(fracCount(length, frac), withReplacement, options...)
}

// Shuffle puts the elements of the DataList in random order, in place, and
// returns the list, the way Sort reorders it. Use Shuffled for a shuffled
// copy that leaves the list as it is. For one seed, Shuffle and Shuffled give
// the same order.
func (dl *DataList) Shuffle(options ...SamplingOptions) *DataList {
	if msg := extraOptional("SamplingOptions", len(options)); msg != "" {
		dl.fail("Shuffle", "%s", msg)
		return dl
	}
	empty := false
	dl.AtomicDo(func(dl *DataList) {
		if len(dl.data) == 0 {
			empty = true
			return
		}
		indices := newSamplingRandom(options).perm(len(dl.data))
		shuffled := make([]any, len(indices))
		for i, idx := range indices {
			shuffled[i] = dl.data[idx]
		}
		dl.data = shuffled
		dl.updateTimestamp()
	})
	if empty {
		dl.warn("Shuffle", "DataList is empty")
	}
	return dl
}

// Shuffled returns a copy of the DataList with its elements in random order,
// named after the list with "_Shuffled" appended, and leaves the list as it
// is. Use Shuffle to reorder the list itself.
func (dl *DataList) Shuffled(options ...SamplingOptions) *DataList {
	if msg := extraOptional("SamplingOptions", len(options)); msg != "" {
		dl.fail("Shuffled", "%s", msg)
		return dl.failedResult()
	}
	var data []any
	var name string
	dl.AtomicDo(func(dl *DataList) {
		data = make([]any, len(dl.data))
		copy(data, dl.data)
		name = dl.name
	})
	if len(data) == 0 {
		dl.warn("Shuffled", "DataList is empty")
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
