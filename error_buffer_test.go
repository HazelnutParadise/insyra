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

func TestPeekError(t *testing.T) {
	ClearErrors()

	// Peek on empty buffer should return nil
	if PeekError(ErrPoppingModeFIFO) != nil {
		t.Error("PeekError should return nil when buffer is empty")
	}

	// Push an error
	LogWarning("TestPackage", "TestFunc", "Test message")
	time.Sleep(10 * time.Millisecond)

	// Peek should return the error without removing it
	err := PeekError(ErrPoppingModeFIFO)
	if err == nil {
		t.Error("PeekError should return non-nil after pushing an error")
		return // Prevents nil dereference
	}

	if err.PackageName != "TestPackage" {
		t.Errorf("Expected PackageName 'TestPackage', got '%s'", err.PackageName)
	}

	if err.FuncName != "TestFunc" {
		t.Errorf("Expected FuncName 'TestFunc', got '%s'", err.FuncName)
	}

	if err.Level != LogLevelWarning {
		t.Errorf("Expected Level LogLevelWarning, got %v", err.Level)
	}

	// Count should still be 1 (not removed)
	if GetErrorCount() != 1 {
		t.Errorf("GetErrorCount should still be 1 after peek, got %d", GetErrorCount())
	}

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

func TestGetErrorsByLevel(t *testing.T) {
	ClearErrors()

	LogWarning("Pkg1", "Func1", "Warning 1")
	LogWarning("Pkg2", "Func2", "Warning 2")
	time.Sleep(20 * time.Millisecond)

	warnings := GetErrorsByLevel(LogLevelWarning)
	if len(warnings) != 2 {
		t.Errorf("Expected 2 warnings, got %d", len(warnings))
	}

	// Should not remove errors
	if GetErrorCount() != 2 {
		t.Error("GetErrorsByLevel should not remove errors")
	}

	ClearErrors()
}

func TestGetErrorsByPackage(t *testing.T) {
	ClearErrors()

	LogWarning("Pkg1", "Func1", "Message 1")
	LogWarning("Pkg2", "Func2", "Message 2")
	LogWarning("Pkg1", "Func3", "Message 3")
	time.Sleep(20 * time.Millisecond)

	pkg1Errors := GetErrorsByPackage("Pkg1")
	if len(pkg1Errors) != 2 {
		t.Errorf("Expected 2 errors from Pkg1, got %d", len(pkg1Errors))
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

func TestPopErrorInfo(t *testing.T) {
	ClearErrors()

	// Pop from empty should return nil
	if PopErrorInfo(ErrPoppingModeFIFO) != nil {
		t.Error("PopErrorInfo should return nil when buffer is empty")
	}

	LogWarning("TestPkg", "TestFn", "Test message")
	time.Sleep(10 * time.Millisecond)

	err := PopErrorInfo(ErrPoppingModeFIFO)
	if err == nil {
		t.Error("PopErrorInfo should return non-nil")
	}

	if err.PackageName != "TestPkg" {
		t.Errorf("Expected PackageName 'TestPkg', got '%s'", err.PackageName)
	}

	// Should be empty now
	if GetErrorCount() != 0 {
		t.Error("PopErrorInfo should remove the error")
	}
}

func TestHasErrorAboveLevel(t *testing.T) {
	ClearErrors()

	// No errors
	if HasErrorAboveLevel(LogLevelWarning) {
		t.Error("HasErrorAboveLevel should return false when empty")
	}

	LogWarning("Pkg", "Fn", "Warning")
	time.Sleep(10 * time.Millisecond)

	// Should have warning level errors
	if !HasErrorAboveLevel(LogLevelWarning) {
		t.Error("HasErrorAboveLevel should return true for Warning when warning exists")
	}

	// Should not have fatal level errors
	if HasErrorAboveLevel(LogLevelFatal) {
		t.Error("HasErrorAboveLevel should return false for Fatal when only warning exists")
	}

	ClearErrors()
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
		{LogLevel(99), "UNKNOWN"},
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

	err := PeekError(ErrPoppingModeFIFO)
	if err == nil {
		t.Fatal("Expected error to be present")
	}

	if err.Timestamp.Before(before) || err.Timestamp.After(after) {
		t.Error("Error timestamp should be between before and after times")
	}

	ClearErrors()
}
