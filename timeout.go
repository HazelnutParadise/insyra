package insyra

import (
	"time"
)

// TimeAfter 函數用於創建一個超時 channel
func TimeAfter(seconds int) <-chan time.Time {
	return time.After(time.Duration(seconds) * time.Second)
}
