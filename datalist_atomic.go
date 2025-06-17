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
	s.initOnce.Do(func() {
		s.cmdCh = make(chan func())
		go s.actorLoop()
	})

	if inActorLoop() {
		// ✅ 直接執行（已在 actor goroutine）
		f(s)
		return
	}

	done := make(chan struct{})
	s.cmdCh <- func() {
		gid := getGID()
		actorContext.Store(gid, true) // 標記為 actor goroutine
		defer actorContext.Delete(gid)

		f(s)
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
