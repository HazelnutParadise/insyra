// Package accel defines the opt-in acceleration runtime surface for Insyra.
//
// This package intentionally exposes only CPU-safe runtime contracts in its
// first step: configuration, sessions, device metadata, typed datasets, and
// execution reports. Backend discovery and real GPU execution are introduced by
// later changes.
package accel
