package wgpu

import "testing"

// A launch never covers fewer than one workgroup of outputs, and covers fewer
// outputs the more products each sums.
func TestEdgeSumLaunchSpan(t *testing.T) {
	for _, c := range []struct{ perOutput, want int }{
		{0, edgeSumLaunchProducts},
		{1, edgeSumLaunchProducts},
		{16, edgeSumLaunchProducts / 16},
		{edgeSumLaunchProducts, 64},
		{edgeSumLaunchProducts * 4, 64},
	} {
		if got := edgeSumLaunchSpan(c.perOutput); got != c.want {
			t.Errorf("edgeSumLaunchSpan(%d) = %d, want %d", c.perOutput, got, c.want)
		}
	}
}
