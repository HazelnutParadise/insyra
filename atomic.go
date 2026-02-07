package insyra

import (
	"runtime"

	"github.com/HazelnutParadise/insyra/internal/core"
)

// ----------------------- DataList Atomic ----------------------

var dataListAtomicGroup = core.NewAtomicGroup()

func (s *DataList) AtomicDo(f func(*DataList)) {
	// LogDebug("DataList", "AtomicDo", "threadSafe: %v", Config.threadSafe)
	if !Config.threadSafe {
		// 憒??典??蔭??鈭?蝔??剁??湔?瑁?
		f(s)
		return
	}
	s.atomicActor.SetGroupOnce(dataListAtomicGroup)
	core.AtomicDoWithInit(&s.atomicActor, s, f, func() {
		// 閮剔蔭finalizer靘???皞?
		runtime.SetFinalizer(s, (*DataList).cleanup)
	})
}

// Close ??DataList嚗???皞?
func (s *DataList) Close() {
	if s.atomicActor.IsClosed() {
		return
	}
	s.atomicActor.Close()
}

// cleanup ?病inalizer?賣嚗?澆??曉??嗆?皜?鞈?
func (s *DataList) cleanup() {
	s.Close()
}

// ----------------------- DataTable Atomic ----------------------

var dataTableAtomicGroup = core.NewAtomicGroup()

func (dt *DataTable) AtomicDo(f func(*DataTable)) {
	// LogDebug("DataTable", "AtomicDo", "threadSafe: %v", Config.threadSafe)
	if !Config.threadSafe {
		// 憒??典??蔭??鈭?蝔??剁??湔?瑁?
		f(dt)
		return
	}
	dt.atomicActor.SetGroupOnce(dataTableAtomicGroup)
	core.AtomicDoWithInit(&dt.atomicActor, dt, f, func() {
		// 閮剔蔭finalizer靘???皞?
		runtime.SetFinalizer(dt, (*DataTable).cleanup)
	})
}

// Close ??DataTable嚗???皞?
func (dt *DataTable) Close() {
	if dt.atomicActor.IsClosed() {
		return
	}
	dt.atomicActor.Close()
}

// cleanup ?病inalizer?賣嚗?澆??曉??嗆?皜?鞈?
func (dt *DataTable) cleanup() {
	dt.Close()
}
