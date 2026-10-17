package datafetch

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileGeocodeCacheWritesAtomically(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "cache.json")
	c := NewFileGeocodeCache(p)
	c.Set("k", &ReverseGeocodeResult{CountyName: "x"})
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 || entries[0].Name() != "cache.json" {
		names := []string{}
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Fatalf("expected only cache.json, got %v", names)
	}
	if got, ok := NewFileGeocodeCache(p).Get("k"); !ok || got == nil || got.CountyName != "x" {
		t.Fatalf("reload failed: %v %v", got, ok)
	}
}

func TestFileGeocodeCacheLeavesAFileAtDotTmpAlone(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "cache.json")
	mine := p + ".tmp"
	if err := os.WriteFile(mine, []byte("not the cache"), 0o600); err != nil {
		t.Fatal(err)
	}
	NewFileGeocodeCache(p).Set("k", &ReverseGeocodeResult{CountyName: "x"})
	got, err := os.ReadFile(mine)
	if err != nil || string(got) != "not the cache" {
		t.Fatalf("a file kept at %s was touched: %q, %v", mine, got, err)
	}
	if got, ok := NewFileGeocodeCache(p).Get("k"); !ok || got == nil || got.CountyName != "x" {
		t.Fatalf("reload failed: %v %v", got, ok)
	}
}
