package commands

import (
	"bytes"
	"sort"
	"strings"
	"testing"

	insyra "github.com/HazelnutParadise/insyra"
)

func TestCommandsNameEveryVariableTheyStore(t *testing.T) {
	cases := []struct {
		name string
		ctx  func(*testing.T) *ExecContext
		cmd  string
		args []string
	}{
		{"kmeans", func(t *testing.T) *ExecContext {
			return &ExecContext{Vars: map[string]any{"t": clusteringDataTable()}, Output: &bytes.Buffer{}}
		}, "kmeans", []string{"t", "2", "seed", "7", "as", "km"}},
		{"dbscan", func(t *testing.T) *ExecContext {
			return &ExecContext{Vars: map[string]any{"t": clusteringDataTable()}, Output: &bytes.Buffer{}}
		}, "dbscan", []string{"t", "1.5", "2", "as", "db"}},
		{"silhouette", func(t *testing.T) *ExecContext {
			return &ExecContext{Vars: map[string]any{"t": clusteringDataTable(), "labels": insyra.NewDataList(1, 1, 1, 2, 2, 2)}, Output: &bytes.Buffer{}}
		}, "silhouette", []string{"t", "labels", "as", "sil"}},
		{"pca", func(t *testing.T) *ExecContext {
			return &ExecContext{Vars: map[string]any{"t": clusteringDataTable()}, Output: &bytes.Buffer{}}
		}, "pca", []string{"t", "2", "as", "pc"}},
		{"corrmatrix", func(t *testing.T) *ExecContext {
			return &ExecContext{Vars: map[string]any{"t": clusteringDataTable()}, Output: &bytes.Buffer{}}
		}, "corrmatrix", []string{"t", "as", "cm"}},
		{"knn classify", func(t *testing.T) *ExecContext {
			return &ExecContext{Vars: map[string]any{"train": knnTrainTable(), "test": knnTestTable(), "labels": insyra.NewDataList("red", "red", "red", "blue", "blue", "blue")}, Output: &bytes.Buffer{}}
		}, "knn_classify", []string{"train", "labels", "test", "3", "as", "cls"}},
		{"knn neighbors", func(t *testing.T) *ExecContext {
			return &ExecContext{Vars: map[string]any{"train": knnTrainTable(), "test": knnTestTable()}, Output: &bytes.Buffer{}}
		}, "knn_neighbors", []string{"train", "test", "2", "as", "nbr"}},
		{"quant factor", quantCtx, "quant", []string{"factor", "a", "f", "as", "fm"}},
		{"quant portfolio", quantPortfolioCtx, "quant", []string{"portfolio", "dt", "minvar", "as", "w"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := tc.ctx(t)
			before := make(map[string]bool, len(ctx.Vars))
			for name := range ctx.Vars {
				before[name] = true
			}
			if err := Dispatch(ctx, tc.cmd, tc.args); err != nil {
				t.Fatal(err)
			}
			var added []string
			for name := range ctx.Vars {
				if !before[name] {
					added = append(added, name)
				}
			}
			sort.Strings(added)
			if len(added) < 2 {
				t.Fatalf("expected multiple stored variables, got %v", added)
			}
			out := ctx.Output.(*bytes.Buffer).String()
			for _, name := range added {
				if !containsWholeName(out, name) {
					t.Errorf("output does not name %s:\n%s", name, out)
				}
			}
		})
	}
}

func containsWholeName(out, name string) bool {
	for offset := 0; offset < len(out); {
		i := strings.Index(out[offset:], name)
		if i < 0 {
			return false
		}
		start, end := offset+i, offset+i+len(name)
		if (start == 0 || !isNameByte(out[start-1])) && (end == len(out) || !isNameByte(out[end])) {
			return true
		}
		offset = start + 1
	}
	return false
}

func isNameByte(b byte) bool {
	return b == '_' || b == '$' || b >= '0' && b <= '9' || b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z'
}
