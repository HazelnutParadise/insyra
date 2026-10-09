package commands

import (
	"bytes"
	"sort"
	"strings"
	"testing"

	insyra "github.com/HazelnutParadise/insyra"
)

// A command that stores variables beside its main result replaced any
// variable under those names without saying so (#325). Every variable a
// command adds is now named in its output.
func TestEveryStoredVariableIsNamedInTheOutput(t *testing.T) {
	clustering := func(t *testing.T) *ExecContext {
		return &ExecContext{
			Vars: map[string]any{
				"t":      clusteringDataTable(),
				"labels": insyra.NewDataList(1, 1, 1, 2, 2, 2),
			},
			Output: &bytes.Buffer{},
		}
	}
	knn := func(t *testing.T) *ExecContext {
		return &ExecContext{
			Vars: map[string]any{
				"train":  knnTrainTable(),
				"test":   knnTestTable(),
				"labels": insyra.NewDataList("red", "red", "red", "blue", "blue", "blue"),
			},
			Output: &bytes.Buffer{},
		}
	}
	cases := []struct {
		name string
		ctx  func(t *testing.T) *ExecContext
		args []string
	}{
		{"kmeans", clustering, []string{"kmeans", "t", "2", "seed", "7", "as", "km"}},
		{"dbscan", clustering, []string{"dbscan", "t", "1.5", "2", "as", "db"}},
		{"silhouette", clustering, []string{"silhouette", "t", "labels", "as", "sil"}},
		{"pca", clustering, []string{"pca", "t", "2", "as", "pc"}},
		{"corrmatrix", clustering, []string{"corrmatrix", "t", "as", "cm"}},
		{"knn_classify", knn, []string{"knn_classify", "train", "labels", "test", "3", "as", "cls"}},
		{"knn_neighbors", knn, []string{"knn_neighbors", "train", "test", "2", "as", "nbr"}},
		{"quant factor", quantCtx, []string{"quant", "factor", "a", "f", "as", "fm"}},
		{"quant portfolio", quantPortfolioCtx, []string{"quant", "portfolio", "dt", "minvar", "as", "w"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := tc.ctx(t)
			before := map[string]bool{}
			for name := range ctx.Vars {
				before[name] = true
			}
			if err := Dispatch(ctx, tc.args[0], tc.args[1:]); err != nil {
				t.Fatalf("%v: %v", tc.args, err)
			}
			var added []string
			for name := range ctx.Vars {
				if !before[name] {
					added = append(added, name)
				}
			}
			sort.Strings(added)
			if len(added) < 2 {
				t.Fatalf("expected more than one stored variable, got %v", added)
			}
			out := ctx.Output.(*bytes.Buffer).String()
			for _, name := range added {
				if !containsWord(out, name) {
					t.Errorf("output does not name %s:\n%s", name, out)
				}
			}
		})
	}
}

// containsWord reports whether name appears in out as a whole word, so that
// km does not count as naming km_size.
func containsWord(out, name string) bool {
	for i := 0; ; {
		j := strings.Index(out[i:], name)
		if j < 0 {
			return false
		}
		start, end := i+j, i+j+len(name)
		if (start == 0 || !isNameByte(out[start-1])) && (end == len(out) || !isNameByte(out[end])) {
			return true
		}
		i = start + 1
	}
}

func isNameByte(b byte) bool {
	return b == '_' || b == '$' || b >= '0' && b <= '9' || b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z'
}
