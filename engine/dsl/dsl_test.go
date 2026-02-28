package dsl

import (
	"testing"
)

func TestNewSessionAndExecute(t *testing.T) {
	session, err := NewSession("", nil)
	if err != nil {
		t.Fatalf("new session failed: %v", err)
	}
	if session == nil {
		t.Fatalf("session should not be nil")
	}

	if err := session.Execute("newdl 1 2 3 as x"); err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if err := session.Execute("mean x"); err != nil {
		t.Fatalf("execute failed: %v", err)
	}
}
