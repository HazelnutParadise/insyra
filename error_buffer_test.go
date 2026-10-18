package insyra

import (
	"testing"
	"time"
)

func TestErrorBufferBasicOperations(t *testing.T) {
	// Clear any existing errors
	ClearErrors()

	// Test HasError returns false when empty
	if HasError() {
		t.Error("HasError should return false when error buffer is empty")
	}

	// Test GetErrorCount returns 0 when empty
	if GetErrorCount() != 0 {
		t.Error("GetErrorCount should return 0 when error buffer is empty")
	}

	// Push some errors
	LogWarning("TestPackage", "TestFunc", "Test warning message")

	// Wait for goroutine to process the error
	time.Sleep(10 * time.Millisecond)

	// Test HasError returns true
	if !HasError() {
		t.Error("HasError should return true after pushing an error")
	}

	// Test GetErrorCount returns 1
	if GetErrorCount() != 1 {
		t.Errorf("GetErrorCount should return 1, got %d", GetErrorCount())
	}

	// Clean up
	ClearErrors()
}

func TestGetAllErrors(t *testing.T) {
	ClearErrors()

	// Push multiple errors
	LogWarning("Pkg1", "Func1", "Message 1")
	LogWarning("Pkg2", "Func2", "Message 2")
	LogWarning("Pkg3", "Func3", "Message 3")
	time.Sleep(20 * time.Millisecond)

	// Get all errors
	errors := GetAllErrors()
	if len(errors) != 3 {
		t.Errorf("Expected 3 errors, got %d", len(errors))
	}

	// Verify order (FIFO)
	if errors[0].PackageName != "Pkg1" {
		t.Errorf("Expected first error from Pkg1, got %s", errors[0].PackageName)
	}
	if errors[2].PackageName != "Pkg3" {
		t.Errorf("Expected last error from Pkg3, got %s", errors[2].PackageName)
	}

	// Should not remove errors
	if GetErrorCount() != 3 {
		t.Errorf("GetAllErrors should not remove errors, count is %d", GetErrorCount())
	}

	ClearErrors()
}

func TestPopAllErrors(t *testing.T) {
	ClearErrors()

	LogWarning("Pkg1", "Func1", "Message 1")
	LogWarning("Pkg2", "Func2", "Message 2")
	time.Sleep(20 * time.Millisecond)

	errors := PopAllErrors()
	if len(errors) != 2 {
		t.Errorf("Expected 2 errors, got %d", len(errors))
	}

	// Should be empty now
	if GetErrorCount() != 0 {
		t.Errorf("PopAllErrors should remove all errors, count is %d", GetErrorCount())
	}
}

func TestErrorInfoError(t *testing.T) {
	err := ErrorInfo{
		Level:       LogLevelWarning,
		PackageName: "TestPkg",
		FuncName:    "TestFn",
		Message:     "Test message",
	}

	errStr := err.Error()
	expected := "[WARNING] TestPkg.TestFn: Test message"
	if errStr != expected {
		t.Errorf("Expected '%s', got '%s'", expected, errStr)
	}
}

func TestLogLevelString(t *testing.T) {
	tests := []struct {
		level    LogLevel
		expected string
	}{
		{LogLevelDebug, "DEBUG"},
		{LogLevelInfo, "INFO"},
		{LogLevelWarning, "WARNING"},
		{LogLevelFatal, "FATAL"},
		{LogLevel(99), "LogLevel(99)"},
	}

	for _, tt := range tests {
		if tt.level.String() != tt.expected {
			t.Errorf("LogLevel(%d).String() = %s, expected %s", tt.level, tt.level.String(), tt.expected)
		}
	}
}

func TestDataTableErr(t *testing.T) {
	dt := NewDataTable()

	// No error initially
	if dt.Err() != nil {
		t.Error("New DataTable should have no error")
	}

	// Set an error
	dt.setError(LogLevelWarning, "DataTable", "TestFunc", "Test error")

	err := dt.Err()
	if err == nil {
		t.Error("Err() should return the set error")
		return
	}

	if err.Message != "Test error" {
		t.Errorf("Expected message 'Test error', got '%s'", err.Message)
	}

	// Clear error
	dt.ClearErr()
	if dt.Err() != nil {
		t.Error("ClearErr should clear the error")
	}
}

func TestDataListErr(t *testing.T) {
	dl := NewDataList()

	// No error initially
	if dl.Err() != nil {
		t.Error("New DataList should have no error")
	}

	// Set an error
	dl.setError(LogLevelWarning, "DataList", "TestFunc", "Test error")

	err := dl.Err()
	if err == nil {
		t.Error("Err() should return the set error")
		return
	}

	if err.Message != "Test error" {
		t.Errorf("Expected message 'Test error', got '%s'", err.Message)
	}

	// Clear error
	dl.ClearErr()
	if dl.Err() != nil {
		t.Error("ClearErr should clear the error")
	}
}

func TestErrorTimestamp(t *testing.T) {
	ClearErrors()

	before := time.Now()
	LogWarning("Pkg", "Fn", "Message")
	time.Sleep(10 * time.Millisecond)
	after := time.Now()

	errs := GetAllErrors()
	if len(errs) != 1 {
		t.Fatalf("Expected one error to be present, got %d", len(errs))
	}
	err := errs[0]

	if err.Timestamp.Before(before) || err.Timestamp.After(after) {
		t.Error("Error timestamp should be between before and after times")
	}

	ClearErrors()
}
