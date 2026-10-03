package py

import (
	"slices"
	"strings"
	"testing"
)

// The pinned uv writes every tool it installs into a build environment to its
// debug log, so a real source build is the only thing that shows whether the
// build constraints cover them. readBuildRequirements reads that log; the test
// below pins the lines it reads and the ones it must leave alone.

const uvBuildLogSample = "DEBUG Resolving build requirements\n" +
	"DEBUG Installing in cython==3.3.0, numpy==2.5.3, setuptools==84.0.0 in /tmp/uvcache/builds-v0/.tmp2ZGeb9\n" +
	"DEBUG Installing build requirements: numpy==2.5.3, cython==3.3.0, setuptools==84.0.0\n" +
	"DEBUG Calling mesonpy.get_requires_for_build_wheel()\n" +
	"DEBUG Installing build requirement: ninja==1.13.2\n" +
	"      Built blis==1.3.3"

func TestBuildRequirementsAreReadFromTheUVLog(t *testing.T) {
	want := []string{"numpy==2.5.3", "cython==3.3.0", "setuptools==84.0.0", "ninja==1.13.2"}

	got := readBuildRequirements(uvBuildLogSample)
	if len(got) != len(want) {
		t.Fatalf("readBuildRequirements returned %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("requirement %d is %q, want %q; the full result was %v", i+1, got[i], want[i], got)
		}
	}

	if crlf := readBuildRequirements(strings.ReplaceAll(uvBuildLogSample, "\n", "\r\n")); !slices.Equal(crlf, want) {
		t.Errorf("a log written with CRLF gave %v, want %v", crlf, want)
	}

	if got := readBuildRequirements(""); len(got) != 0 {
		t.Errorf("readBuildRequirements(\"\") returned %v, want nothing", got)
	}
}
