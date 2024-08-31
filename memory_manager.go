package insyra

import (
	"sync"
	"time"
)

// ======================== Memory Compaction Mechanism ========================
// 这是一个全局的内存管理器，用于管理所有被注册的 DataList 对象。
type MemoryManager struct {
	lists map[*DataList]struct{}
	mu    sync.Mutex
}

var globalMemoryManager *MemoryManager
var once sync.Once

// GetMemoryManager retrieves the global memory manager singleton instance.
func GetMemoryManager() *MemoryManager {
	once.Do(func() {
		globalMemoryManager = &MemoryManager{
			lists: make(map[*DataList]struct{}),
		}
		// 启动垃圾回收器
		go globalMemoryManager.startGarbageCollector()
	})
	return globalMemoryManager
}

// Register adds a DataList to the memory manager for tracking and potential compaction.
func (m *MemoryManager) Register(dl *DataList) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lists[dl] = struct{}{}
	dl.incrementRef() // 增加引用计数
	LogDebug("MemoryManager.Register(): DataList registered with refCount=%d\n", dl.refCount)
}

// Unregister removes a DataList from the memory manager.
func (m *MemoryManager) Unregister(dl *DataList) {
	m.mu.Lock()
	defer m.mu.Unlock()
	dl.decrementRef() // 减少引用计数
	if dl.refCount <= 0 {
		delete(m.lists, dl)
		LogDebug("MemoryManager.Unregister(): DataList unregistered\n")
	}
}

// ReorganizeMemory compacts the memory for a given DataList.
func (m *MemoryManager) ReorganizeMemory(dl *DataList) {
	dl.mu.Lock()
	defer dl.mu.Unlock()

	// Create a new slice with the exact needed capacity
	newData := make([]interface{}, len(dl.data))
	copy(newData, dl.data)
	dl.data = newData
	LogDebug("MemoryManager.ReorganizeMemory(): DataList reorganized with len=%d, cap=%d\n", len(dl.data), cap(dl.data))
}

// ======================== Garbage Collection Mechanism ========================

// startGarbageCollector runs the garbage collector periodically to clean up unused DataLists.
func (m *MemoryManager) startGarbageCollector() {
	LogDebug("MemoryManager.startGarbageCollector(): Starting garbage collector\n")
	for {
		time.Sleep(10 * time.Second) // 每10秒检查一次
		m.mu.Lock()
		for dl := range m.lists {
			if dl.refCount <= 0 {
				delete(m.lists, dl)
			} else if dl.isFragmented() {
				m.ReorganizeMemory(dl)
				dl.needReorganize = false
			}
		}
		m.mu.Unlock()
	}
}

// ======================== DataList Reference Management ========================

func (dl *DataList) incrementRef() {
	dl.mu.Lock()
	defer dl.mu.Unlock()
	dl.refCount++
}

func (dl *DataList) decrementRef() {
	dl.mu.Lock()
	defer dl.mu.Unlock()
	dl.refCount--
}

func (dl *DataList) isFragmented() bool {
	// Simple heuristic: if the length and capacity differ significantly, mark as fragmented
	LogDebug("DataList.isFragmented(): len=%d, cap=%d\n", len(dl.data), cap(dl.data))
	return len(dl.data) < cap(dl.data)/2
}
