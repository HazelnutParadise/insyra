package dsl

import "github.com/HazelnutParadise/insyra/internal/dsl/env"

// Manager owns where a session's environments live, and reads and writes them:
// each environment's variables, command history and configuration. Every
// operation on an environment is one of its methods. SetBasePath and
// SetEnvsDirName move it; do not move a Manager while a session uses it.
type Manager = env.Manager

// EnvironmentInfo describes one environment, as Manager.List and Manager.Info
// report it.
type EnvironmentInfo = env.EnvironmentInfo

// GlobalConfig is the configuration shared by every environment of a Manager.
type GlobalConfig = env.GlobalConfig

// State is an environment's saved variables, as its state.json holds them.
type State = env.State

// SerializedVariable is one variable as state.json holds it.
type SerializedVariable = env.SerializedVariable

// UnsavedVariable names a variable Manager.SaveVariables could not store, with
// its Go type and the reason.
type UnsavedVariable = env.UnsavedVariable

// NewManager returns a Manager rooted at basePath, keeping each environment in
// basePath/envsDirName/<name>/. "" for basePath means <UserHomeDir>/.insyra,
// and "" for envsDirName means "envs":
//
//	dsl.NewManager("/ws/.idensyra", "insights")  // /ws/.idensyra/insights/<name>/
func NewManager(basePath, envsDirName string) *Manager {
	return env.NewManager(basePath, envsDirName)
}

// DefaultManager returns a Manager rooted where the insyra command keeps its
// environments, <UserHomeDir>/.insyra/envs/<name>/. Each call returns a Manager
// of its own, so moving one with SetBasePath moves no other.
func DefaultManager() *Manager {
	return env.NewManager("", "")
}
