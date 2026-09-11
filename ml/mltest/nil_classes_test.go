package mltest

import (
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/ml"
)

// nilClassesModel is what a third-party implementation looks like when it
// returns nil from Classes(). The library's own classifiers no longer do this,
// but ml.Classifier is a public interface with only exported methods, so anyone
// can implement it — and this suite is where they find out.
type nilClassesModel struct{}

func (nilClassesModel) Features() []string { return []string{"a"} }
func (nilClassesModel) Predict(dt *insyra.DataTable) (*insyra.DataList, error) {
	return insyra.NewDataList("yes"), nil
}
func (nilClassesModel) Classes() *insyra.DataList { return nil }

func TestConformanceRejectsNilClasses(t *testing.T) {
	var _ ml.Classifier = nilClassesModel{} // the shape a third party implements

	err := checkClassesUsable(nilClassesModel{}.Classes())
	if err == nil {
		t.Fatal("a nil Classes() was accepted; the suite would panic on Len() instead of reporting")
	}
	if !strings.Contains(err.Error(), "nil") || !strings.Contains(err.Error(), "Err()") {
		t.Errorf("the message should name the problem and the fix: %v", err)
	}
}

func TestConformanceAcceptsEmptyClasses(t *testing.T) {
	prev := insyra.Config.GetLogLevel()
	insyra.Config.SetLogLevel(insyra.LogLevelFatal)
	t.Cleanup(func() { insyra.Config.SetLogLevel(prev) })

	// An empty list carrying a reason is what the library's own classifiers
	// return when they have nothing; the suite must let it through to the
	// checks that follow rather than treating it as a protocol violation.
	empty := insyra.NewDataList()
	empty.SetErr("ml", "Classes", "not fitted")
	if err := checkClassesUsable(empty); err != nil {
		t.Errorf("an empty list is usable: %v", err)
	}
}
