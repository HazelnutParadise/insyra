package accel

import "testing"

// NewSession used to keep the first Config and drop the rest. More than one
// is an error, reported by Discover, the first step that can fail.
func TestNewSessionRefusesMoreThanOneConfig(t *testing.T) {
	session := NewSession(DefaultConfig(), DefaultConfig())
	if err := session.Discover(); err == nil {
		t.Fatal("a session given two configs discovered devices")
	}
}
