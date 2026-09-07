package insyra

import "testing"

// IN-7: the global buffer is a bounded diagnostic log, not a growing leak.
func TestGlobalErrorBufferIsBounded(t *testing.T) {
	restoreConfig(t)
	Config.SetLogLevel(LogLevelFatal)
	Config.SetPanicOnError(false)
	ClearErrors()
	t.Cleanup(ClearErrors)

	for i := 0; i < 5000; i++ {
		LogWarning("test", "TestGlobalErrorBufferIsBounded", "filler %d", i)
	}
	if got := GetErrorCount(); got > ErrorBufferCapacity {
		t.Fatalf("buffer holds %d records, capacity is %d", got, ErrorBufferCapacity)
	}
	// The newest record must survive; the oldest is the one dropped.
	all := GetAllErrors()
	if len(all) == 0 {
		t.Fatal("buffer is empty after 5000 pushes")
	}
	if last := all[len(all)-1]; last.Message != "filler 4999" {
		t.Fatalf("newest record is %q, want the last one pushed", last.Message)
	}
}

// The instance error and the global buffer are independent stores.
func TestGlobalPopDoesNotClearInstanceError(t *testing.T) {
	restoreConfig(t)
	Config.SetLogLevel(LogLevelFatal)
	Config.SetPanicOnError(false)
	ClearErrors()
	t.Cleanup(ClearErrors)

	dl := NewDataList(1.0, "x")
	dl.Normalize()
	if dl.Err() == nil {
		t.Fatal("setup: expected an instance error")
	}
	PopAllErrors()
	if GetErrorCount() != 0 {
		t.Fatal("PopAllErrors left records behind")
	}
	if dl.Err() == nil {
		t.Fatal("popping the global buffer cleared the instance error")
	}
	dl.ClearErr()
	LogWarning("test", "TestGlobalPopDoesNotClearInstanceError", "after")
	if GetErrorCount() == 0 {
		t.Fatal("ClearErr() emptied the global buffer")
	}
}
