package insyra

import (
	"bytes"
	"runtime"
	"strconv"
	"sync"
)

// ----------------------- DataList Atomic ----------------------

var actorContext sync.Map // map[uint64]bool
func inActorLoop() bool {
	gid := getGID()
	_, ok := actorContext.Load(gid)
	return ok
}

func (s *DataList) AtomicDo(f func(*DataList)) {
	// LogDebug("DataList", "AtomicDo", "threadSafe: %v", Config.threadSafe)
	if !Config.threadSafe {
		// 如果全域配置關閉了線程安全，直接執行
		f(s)
		return
	}

	// 檢查是否已關閉
	if s.closed.Load() {
		f(s)
		return
	}
	s.initOnce.Do(func() {
		s.cmdCh = make(chan func())
		go s.actorLoop()
		// 設置finalizer來清理資源
		runtime.SetFinalizer(s, (*DataList).cleanup)
	})

	if inActorLoop() {
		// ✅ 直接執行（已在 actor goroutine）
		f(s)
		return
	}

	// 再次檢查是否在初始化後被關閉
	if s.closed.Load() {
		f(s)
		return
	}

	done := make(chan struct{})
	defer func() {
		if r := recover(); r != nil {
			// 如果發送失敗（通道關閉），直接執行 f
			f(s)
		}
	}()

	// 阻塞發送，確保序列化
	s.cmdCh <- func() {
		gid := getGID()
		actorContext.Store(gid, true) // 標記為 actor goroutine
		defer actorContext.Delete(gid)

		// 在執行前再次檢查是否已關閉
		if !s.closed.Load() {
			f(s)
		}
		close(done)
	}
	<-done
}

func getGID() uint64 {
	b := make([]byte, 64)
	b = b[:runtime.Stack(b, false)]
	b = bytes.TrimPrefix(b, []byte("goroutine "))
	i := bytes.IndexByte(b, ' ')
	id, _ := strconv.ParseUint(string(b[:i]), 10, 64)
	return id
}

func (s *DataList) actorLoop() {
	for fn := range s.cmdCh {
		fn()
	}
}

// Close 關閉DataList，清理資源
func (s *DataList) Close() {
	if s.closed.Load() {
		return
	}
	s.closed.Store(true)

	// 關閉channel，這會導致actorLoop goroutine退出
	if s.cmdCh != nil {
		close(s.cmdCh)
		s.cmdCh = nil
	}
}

// cleanup 是finalizer函數，用於垃圾回收時清理資源
func (s *DataList) cleanup() {
	s.Close()
}

// ----------------------- DataTable Atomic ----------------------

var datatableActorContext sync.Map // map[uint64]bool
func inDataTableActorLoop() bool {
	gid := getGID()
	_, ok := datatableActorContext.Load(gid)
	return ok
}

func (dt *DataTable) AtomicDo(f func(*DataTable)) {
	// LogDebug("DataTable", "AtomicDo", "threadSafe: %v", Config.threadSafe)
	if !Config.threadSafe {
		// 如果全域配置關閉了線程安全，直接執行
		f(dt)
		return
	}

	// 檢查是否已關閉
	if dt.closed.Load() {
		f(dt)
		return
	}
	dt.initOnce.Do(func() {
		dt.cmdCh = make(chan func())
		go dt.actorLoop()
		// 設置finalizer來清理資源
		runtime.SetFinalizer(dt, (*DataTable).cleanup)
	})

	if inDataTableActorLoop() {
		// ✅ 直接執行（已在 actor goroutine）
		f(dt)
		return
	}

	// 再次檢查是否在初始化後被關閉
	if dt.closed.Load() {
		f(dt)
		return
	}

	done := make(chan struct{})
	defer func() {
		if r := recover(); r != nil {
			// 如果發送失敗（通道關閉），直接執行 f
			f(dt)
		}
	}()

	// 阻塞發送，確保序列化
	dt.cmdCh <- func() {
		gid := getGID()
		datatableActorContext.Store(gid, true) // 標記為 actor goroutine
		defer datatableActorContext.Delete(gid)

		// 在執行前再次檢查是否已關閉
		if !dt.closed.Load() {
			f(dt)
		}
		close(done)
	}
	<-done
}

// Close 關閉DataTable，清理資源
func (dt *DataTable) Close() {
	if dt.closed.Load() {
		return
	}
	dt.closed.Store(true)

	// 關閉channel，這會導致actorLoop goroutine退出
	if dt.cmdCh != nil {
		close(dt.cmdCh)
		dt.cmdCh = nil
	}
}

// cleanup 是finalizer函數，用於垃圾回收時清理資源
func (dt *DataTable) cleanup() {
	dt.Close()
}

func (dt *DataTable) actorLoop() {
	for fn := range dt.cmdCh {
		fn()
	}
}
