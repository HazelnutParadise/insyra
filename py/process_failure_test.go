package py

import (
	"strings"
	"testing"
)

// A Python that fails reports its error and then closes processDone, so the
// runner sees both at once and select picks either. When it picked
// processDone, a Python that exited with an error, such as one given code it
// cannot compile, came back as a run that returned nothing, with a nil error,
// about half the time.
func TestAPythonThatExitsWithAnErrorIsTheCallsError(t *testing.T) {
	useFakePython(t, "exit:3")
	for i := 0; i < 20; i++ {
		err := RunCode(nil, "insyra.Return(1)")
		if err == nil {
			t.Fatalf("run %d: a Python that exited with status 3 gave no error", i)
		}
		if !strings.Contains(err.Error(), "exit status 3") {
			t.Fatalf("run %d: got %q, want the process's exit status", i, err)
		}
	}
}
