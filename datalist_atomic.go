package insyra

import (
	"bytes"
	"runtime"
	"strconv"
	"sync"
)

// ! 這裡的機制尚未啟用，若要保證邏輯原子性，需以AtomicDo取代所有互斥鎖

var actorContext sync.Map // map[uint64]bool
func inActorLoop() bool {
	gid := getGID()
	_, ok := actorContext.Load(gid)
	return ok
}

func (s *DataList) AtomicDo(f func(*DataList)) {
	// 檢查是否已關閉
	if s.closed.Load() {
		f(s)
		return
	}
	s.initOnce.Do(func() {
		s.cmdCh = make(chan func())
		go s.actorLoop()
		// finalizer 已經在 NewDataList 中設置，不需要重複設置
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
	select {
	case s.cmdCh <- func() {
		gid := getGID()
		actorContext.Store(gid, true) // 標記為 actor goroutine
		defer actorContext.Delete(gid)

		// 在執行前再次檢查是否已關閉
		if !s.closed.Load() {
			f(s)
		}
		close(done)
	}:
		<-done
	}
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
