package accel

import (
	"strings"
	"testing"
)

func TestCacheKeyIncludesOperationLineage(t *testing.T) {
	dataset := &Dataset{
		Name:        "numbers",
		Fingerprint: "fp-1",
		Lineage:     "project:datalist",
	}
	buffer := Buffer{Name: "numbers", Type: DataTypeInt64, Len: 4}

	key := cacheKey(dataset, buffer, 0)
	if !strings.Contains(key, "project:datalist") {
		t.Fatalf("expected cache key to include operation lineage, got %q", key)
	}
}
