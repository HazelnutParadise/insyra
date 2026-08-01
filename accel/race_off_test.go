//go:build !race

package accel

// raceDetectorEnabled reports whether the binary was built with -race, which
// also turns on checkptr.
const raceDetectorEnabled = false
