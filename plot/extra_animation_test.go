package plot

import (
	"path/filepath"
	"testing"
)

// SaveHTML used to read only the first animation flag. More than one is an
// error, as SavePNG already treats its own optional flag.
func TestSaveHTMLRefusesMoreThanOneAnimationFlag(t *testing.T) {
	path := filepath.Join(t.TempDir(), "chart.html")
	if err := SaveHTML(nil, path, true, false); err == nil {
		t.Fatal("SaveHTML accepted two animation flags")
	}
}
