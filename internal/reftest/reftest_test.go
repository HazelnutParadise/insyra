package reftest_test

import (
	"errors"
	"testing"

	"github.com/HazelnutParadise/insyra/internal/reftest"
)

// run calls fn with a throwaway *testing.T. Both t.Skip and t.Fatal unwind the
// calling goroutine through runtime.Goexit, so it has to have one of its own.
func run(fn func(t *testing.T)) *testing.T {
	fake := &testing.T{}
	done := make(chan struct{})
	go func() {
		defer close(done)
		fn(fake)
	}()
	<-done
	return fake
}

func TestMissingSkipsByDefaultAndFailsUnderStrict(t *testing.T) {
	cause := errors.New("exec: \"Rscript\": executable file not found in $PATH")

	t.Setenv(reftest.StrictEnv, "")
	lenient := run(func(t *testing.T) {
		reftest.Missing(t, "Rscript", "the R parity check", cause)
	})
	if !lenient.Skipped() {
		t.Fatal("a missing toolchain did not skip by default")
	}
	if lenient.Failed() {
		t.Fatal("a missing toolchain failed the run by default; the suite must stay usable without the toolchains")
	}

	t.Setenv(reftest.StrictEnv, "1")
	strict := run(func(t *testing.T) {
		reftest.Missing(t, "Rscript", "the R parity check", cause)
	})
	if !strict.Failed() {
		t.Fatalf("a missing toolchain skipped under %s=1", reftest.StrictEnv)
	}
}

func TestStrictReadsTheEnvironment(t *testing.T) {
	for _, tc := range []struct {
		value string
		want  bool
	}{
		{"", false},
		{"0", false},
		{"true", false}, // only "1" counts, so a typo cannot silently enable it
		{"1", true},
		{" 1 ", true},
	} {
		t.Setenv(reftest.StrictEnv, tc.value)
		if got := reftest.Strict(); got != tc.want {
			t.Errorf("%s=%q gives Strict()=%v, want %v", reftest.StrictEnv, tc.value, got, tc.want)
		}
	}
}

// The skip line is the only record that a verification did not happen, so it
// has to name both the tool and what went unchecked.
func TestBothMessagesNameTheToolAndTheVerification(t *testing.T) {
	for _, strict := range []string{"", "1"} {
		t.Setenv(reftest.StrictEnv, strict)
		fake := run(func(t *testing.T) {
			reftest.Missing(t, "onnxruntime", "the ONNX round trip", nil)
		})
		if !fake.Skipped() && !fake.Failed() {
			t.Fatalf("%s=%q neither skipped nor failed", reftest.StrictEnv, strict)
		}
	}
}

func TestMissingOutputFallsBackWhenThereIsNoOutput(t *testing.T) {
	t.Setenv(reftest.StrictEnv, "")
	fake := run(func(t *testing.T) {
		reftest.MissingOutput(t, "python", "the scikit-learn comparison", errors.New("exit status 1"), []byte("   \n "))
	})
	if !fake.Skipped() {
		t.Fatal("empty probe output did not fall back to the plain form")
	}
}
