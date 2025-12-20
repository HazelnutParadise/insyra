package csvxl

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadCsvToString_UTF8(t *testing.T) {
	temp := t.TempDir()
	f := filepath.Join(temp, "test.csv")
	content := "name,age,city\nJohn,25,New York\nJane,30,San Francisco"
	err := os.WriteFile(f, []byte(content), 0644)
	require.NoError(t, err)

	s, err := ReadCsvToString(f, UTF8)
	require.NoError(t, err)
	assert.Contains(t, s, "name,age,city")
	assert.Contains(t, s, "John,25,New York")
}

func TestReadCsvToString_Auto(t *testing.T) {
	temp := t.TempDir()
	f := filepath.Join(temp, "test.csv")
	content := "name,age,city\nJohn,25,New York"
	err := os.WriteFile(f, []byte(content), 0644)
	require.NoError(t, err)

	s, err := ReadCsvToString(f, "")
	require.NoError(t, err)
	assert.Contains(t, s, "John,25,New York")
}

func TestReadCsvToString_TrimBOM(t *testing.T) {
	temp := t.TempDir()
	f := filepath.Join(temp, "test.csv")
	content := "\uFEFFname,age\nJohn,25"
	err := os.WriteFile(f, []byte(content), 0644)
	require.NoError(t, err)

	s, err := ReadCsvToString(f, UTF8)
	require.NoError(t, err)
	assert.NotContains(t, s, "\uFEFF")
	assert.Contains(t, s, "name,age")
}
