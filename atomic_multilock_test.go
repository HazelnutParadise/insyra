package insyra

import (
	"runtime"
	"sync"
	"testing"
	"time"
)

// Concurrent cross-instance operations must be race-free (run with -race) and
// must not deadlock. Covers the AtomicDoAll multi-lock path and the root
// cross-instance ops rewritten to use it.
func TestAtomicDoAll_ConcurrentNoRace(t *testing.T) {
	a := NewDataList(1, 2, 3)
	b := NewDataList(4, 5, 6)
	dt := NewDataTable(NewDataList(7, 8, 9).SetName("x"))

	done := make(chan struct{})
	var wg sync.WaitGroup

	// Mutators: concurrently append to the shared instances.
	for g := 0; g < 4; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-done:
					return
				default:
					a.Append(1)
					b.Append(2)
				}
			}
		}()
	}

	// Cross-instance operations that lock two instances together.
	for g := 0; g < 4; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 300; i++ {
				_ = a.Concat(b)
				_ = a.IsEqualTo(b)
				AtomicDoAll(func() { _ = len(a.data) + len(b.data) }, a, b)
				AtomicDoAll(func() { _ = len(dt.columns) + len(a.data) }, dt, a) // mixed group
			}
		}()
	}

	time.Sleep(60 * time.Millisecond)
	close(done)
	wg.Wait()
}

// A duplicate instance in the same batch must be de-duplicated, not self-deadlock.
func TestAtomicDoAll_DuplicateInstanceNoDeadlock(t *testing.T) {
	a := NewDataList(1, 2, 3)
	withTimeout(t, "duplicate-instance", func() {
		AtomicDoAll(func() { _ = len(a.data) }, a, a)
		AtomicDoAll(func() { _ = len(a.data) }, a, a, a)
	})
}

// A goroutine already holding one instance must be able to lock a batch that
// includes it (the held one is skipped, not re-locked → no self-deadlock).
func TestAtomicDoAll_NestedAlreadyHeldNoDeadlock(t *testing.T) {
	a := NewDataList(1)
	b := NewDataList(2)
	withTimeout(t, "nested-already-held", func() {
		a.AtomicDo(func(a *DataList) {
			AtomicDoAll(func() { _ = len(a.data) + len(b.data) }, a, b) // a already held
		})
	})
}

// Two goroutines locking the SAME pair in opposite argument order must not
// deadlock — the canonical address ordering makes AB-BA impossible.
func TestAtomicDoAll_OppositeOrderNoDeadlock(t *testing.T) {
	a := NewDataList(1)
	b := NewDataList(2)
	dt1 := NewDataTable(NewDataList(3))
	dt2 := NewDataTable(NewDataList(4))
	withTimeout(t, "opposite-order", func() {
		var wg sync.WaitGroup
		for g := 0; g < 2; g++ {
			wg.Add(1)
			go func(g int) {
				defer wg.Done()
				for i := 0; i < 5000; i++ {
					if g == 0 {
						AtomicDoAll(func() { _ = len(a.data) + len(b.data) }, a, b)
						AtomicDoAll(func() { _ = len(dt1.columns) + len(dt2.columns) }, dt1, dt2)
					} else {
						AtomicDoAll(func() { _ = len(a.data) + len(b.data) }, b, a)
						AtomicDoAll(func() { _ = len(dt1.columns) + len(dt2.columns) }, dt2, dt1)
					}
				}
			}(g)
		}
		wg.Wait()
	})
}

// Merge / AppendCols / UpdateRow (rewritten to AtomicDoAll) must be race-free
// while the passed instances are concurrently mutated.
func TestAtomicDoAll_TableOpsNoRace(t *testing.T) {
	done := make(chan struct{})
	var wg sync.WaitGroup
	shared := NewDataList(1, 2, 3).SetName("s")

	for g := 0; g < 3; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-done:
					return
				default:
					shared.Append(9)
				}
			}
		}()
	}
	for g := 0; g < 3; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 200; i++ {
				dt := NewDataTable(NewDataList(4, 5, 6).SetName("a"))
				dt.AppendCols(shared)
				dt.UpdateRow(0, shared)
			}
		}()
	}
	time.Sleep(60 * time.Millisecond)
	close(done)
	wg.Wait()
}

// AppendCols / UpdateCol / UpdateColByNumber must COPY the passed DataList's data
// into a table-owned column, never alias it. If the table shared a backing array (or
// the whole *DataList) with the still-live source, mutating the source afterwards
// would leak into the table and race dt operations across two independent actor
// locks. This deterministic test pins the copy-on-insert contract.
func TestTableInsert_ClonesInput(t *testing.T) {
	check := func(name string, tableCol *DataList) {
		if tableCol == nil {
			t.Fatalf("%s: table column is nil", name)
		}
		if got := tableCol.Len(); got != 3 {
			t.Fatalf("%s aliased the source: table col len=%d, want 3", name, got)
		}
	}

	// AppendCols
	src := NewDataList(1, 2, 3).SetName("s")
	dtA := NewDataTable()
	dtA.AppendCols(src)
	src.Append(999) // must NOT reach dtA's column
	check("AppendCols", dtA.GetColByNumber(0))

	// UpdateCol (by alphabetic index)
	dtB := NewDataTable(NewDataList(0, 0, 0).SetName("x"))
	replB := NewDataList(7, 8, 9)
	dtB.UpdateCol("A", replB)
	replB.Append(999)
	check("UpdateCol", dtB.GetColByNumber(0))

	// UpdateColByNumber
	dtC := NewDataTable(NewDataList(0, 0, 0).SetName("x"))
	replC := NewDataList(7, 8, 9)
	dtC.UpdateColByNumber(0, replC)
	replC.Append(999)
	check("UpdateColByNumber", dtC.GetColByNumber(0))
}

func withTimeout(t *testing.T, name string, f func()) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		f()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatalf("%s: deadlocked (timed out)", name)
	}
}

// ---------------------------------------------------------------------------
// Heavier stress coverage. These hammer the ACTUAL rewritten public operations
// concurrently with mutation of the shared instances. Deadlock-freedom is a
// design property (total pointer-address lock order ⇒ no cycle); these validate
// race-freedom under -race and confirm no deadlock under heavy contention.
// ---------------------------------------------------------------------------

// stressDeadline runs body in a goroutine and fails if it does not finish in d.
func stressDeadline(t *testing.T, name string, d time.Duration, body func()) {
	t.Helper()
	done := make(chan struct{})
	go func() { body(); close(done) }()
	select {
	case <-done:
	case <-time.After(d):
		t.Fatalf("%s: deadlocked / timed out after %v", name, d)
	}
}

// All rewritten root cross-instance ops, hammered while mutators append.
func TestStress_RootCrossInstanceOps(t *testing.T) {
	// Bounded work: every DataList mutation fires a background `go
	// updateTimestamp()`, so an unbounded tight mutation loop would flood the
	// scheduler (and the -race detector's goroutine cap) — that would be a
	// pre-existing artifact, not a locking issue. We keep the counts bounded and
	// modest so the test reliably COMPLETES under -race (completion proves no
	// deadlock in the rewritten ops), while still interleaving mutation with
	// every rewritten cross-instance operation for race detection.
	iters := 25
	appendsPerMutator := 300
	if testing.Short() {
		iters = 6
		appendsPerMutator = 60
	}
	stressDeadline(t, "root-cross-instance", 90*time.Second, func() {
		a := NewDataList(1, 2, 3, 4, 5).SetName("a")
		b := NewDataList(6, 7, 8, 9, 10).SetName("b")
		c := NewDataList(11, 12).SetName("c")
		shared := NewDataList(1, 2, 3).SetName("s")

		var wg sync.WaitGroup

		// Bounded mutators: append a fixed number of times, yielding between
		// appends so cross-instance workers interleave with the mutations.
		for g := 0; g < 3; g++ {
			wg.Add(1)
			go func(g int) {
				defer wg.Done()
				for i := 0; i < appendsPerMutator; i++ {
					switch g {
					case 0:
						a.Append(i)
					case 1:
						b.Append(i)
					default:
						shared.Append(i)
					}
					runtime.Gosched()
				}
			}(g)
		}

		// Workers exercising every rewritten cross-instance op.
		for g := 0; g < 4; g++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for i := 0; i < iters; i++ {
					_ = a.Concat(b)
					_ = a.AppendDataList(b)
					_ = a.IsEqualTo(b)
					_ = a.IsTheSameAs(b)
					// N-way batches of varying arity and order.
					AtomicDoAll(func() { _ = len(a.data) + len(b.data) + len(c.data) }, a, b, c)
					AtomicDoAll(func() { _ = len(c.data) + len(a.data) }, c, a)
					// Table ops that lock table + passed list(s).
					dt := NewDataTable(NewDataList(1, 2, 3).SetName("x"))
					dt.AppendCols(shared)
					dt.AppendCols(a, b, c)
					dt.UpdateCol("A", shared)
					dt.UpdateRow(0, shared)
					// Horizontal + vertical merge (only occasionally — Merge is the
					// heaviest op and spawns many background goroutines).
					if i%5 == 0 {
						lt := NewDataTable(NewDataList("k1", "k2").SetName("key"), NewDataList(1, 2).SetName("v"))
						rt := NewDataTable(NewDataList("k1", "k2").SetName("key"), NewDataList(3, 4).SetName("w"))
						_, _ = lt.Merge(rt, MergeDirectionHorizontal, MergeModeInner, "key")
						_, _ = lt.Merge(rt, MergeDirectionVertical, MergeModeOuter)
					}
				}
			}()
		}

		wg.Wait()
	})
}

// Many goroutines locking overlapping N-way batches in RANDOM-ish order must not
// deadlock (canonical address order guarantees it). Uses a fixed instance pool.
func TestStress_NWayOverlappingBatches(t *testing.T) {
	rounds := 4000
	if testing.Short() {
		rounds = 400
	}
	stressDeadline(t, "nway-overlap", 60*time.Second, func() {
		// Mixed pool: DataLists and DataTables (mixed groups).
		lists := []*DataList{NewDataList(1), NewDataList(2), NewDataList(3), NewDataList(4), NewDataList(5)}
		tables := []*DataTable{NewDataTable(NewDataList(1)), NewDataTable(NewDataList(2)), NewDataTable(NewDataList(3))}
		pool := make([]any, 0, len(lists)+len(tables))
		for _, l := range lists {
			pool = append(pool, l)
		}
		for _, tb := range tables {
			pool = append(pool, tb)
		}

		var wg sync.WaitGroup
		for g := 0; g < 8; g++ {
			wg.Add(1)
			go func(g int) {
				defer wg.Done()
				for i := 0; i < rounds; i++ {
					// Deterministic-but-goroutine-varying subset + order, so different
					// goroutines request overlapping instances in different arg orders.
					step := (g*7 + i*3) % len(pool)
					size := 2 + (i % 4) // 2..5 instances
					batch := make([]any, 0, size)
					for k := 0; k < size; k++ {
						batch = append(batch, pool[(step+k*(g+1))%len(pool)])
					}
					AtomicDoAll(func() {}, batch...)
				}
			}(g)
		}
		wg.Wait()
	})
}

// Nested AtomicDoAll: an outer batch holds some instances, an inner batch
// overlaps them — the already-held ones are skipped (no self-deadlock), the new
// ones are locked in canonical order (no AB-BA with concurrent workers).
func TestStress_NestedAtomicDoAll(t *testing.T) {
	rounds := 3000
	if testing.Short() {
		rounds = 300
	}
	stressDeadline(t, "nested-batches", 60*time.Second, func() {
		a, b, c, d := NewDataList(1), NewDataList(2), NewDataList(3), NewDataList(4)
		var wg sync.WaitGroup
		for g := 0; g < 6; g++ {
			wg.Add(1)
			go func(g int) {
				defer wg.Done()
				for i := 0; i < rounds; i++ {
					AtomicDoAll(func() {
						_ = len(a.data) + len(b.data)
						// inner batch overlaps a,b and adds c,d
						AtomicDoAll(func() {
							_ = len(a.data) + len(b.data) + len(c.data) + len(d.data)
						}, a, c, b, d)
					}, a, b)
				}
			}(g)
		}
		wg.Wait()
	})
}
