package insyra

import (
	"sync"
)

type NameManager struct {
	names map[string]struct{}
	mu    sync.Mutex
}

var globalNameManager *NameManager
var nameManagerOnce sync.Once

// GetNameManager 初始化全局名稱管理器
func getNameManager() *NameManager {
	nameManagerOnce.Do(func() {
		globalNameManager = &NameManager{
			names: make(map[string]struct{}),
		}
	})
	return globalNameManager
}

// registerName 註冊一個名稱，如果名稱已存在則返回錯誤 (私有方法)(目前使用註解關閉，因為 DataTable 已有 SafeName 機制)
func (nm *NameManager) registerName(name string) error {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	// if _, exists := nm.names[name]; exists {
	// 	return fmt.Errorf("名稱 '%s' 已經被使用", name)
	// }
	nm.names[name] = struct{}{}
	return nil
}

// unregisterName 取消註冊一個名稱 (私有方法)
func (nm *NameManager) unregisterName(name string) {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	delete(nm.names, name)
}
