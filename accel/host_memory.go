package accel

import (
	"os"
	"runtime"
	"strconv"
	"strings"
)

var hostMemoryBytesFunc = defaultHostMemoryBytes

func hostMemoryBytes() uint64 {
	if hostMemoryBytesFunc == nil {
		return 0
	}
	return hostMemoryBytesFunc()
}

func defaultHostMemoryBytes() uint64 {
	switch runtime.GOOS {
	case "linux":
		return readLinuxHostMemoryBytes()
	default:
		return 0
	}
}

func readLinuxHostMemoryBytes() uint64 {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(string(data), "\n") {
		if !strings.HasPrefix(line, "MemTotal:") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			return 0
		}
		value, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			return 0
		}
		return value * 1024
	}
	return 0
}
