package stats

// RunExportRotation is an optional hook used by local diagnostic tools to
// programmatically obtain a rotation result (the value returned by
// FaRotations). The main binary may set this at startup; it is nil by
// default. This file intentionally contains only the hook declaration so
// tests and local tools can be added without changing existing behavior.

// Debug controls verbose internal diagnostic prints. Set to true for developer debug.
var Debug = false
var RunExportRotation func() interface{}
