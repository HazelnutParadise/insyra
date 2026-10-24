package dsl

import "github.com/HazelnutParadise/insyra/cli/env"

// Manager owns where a session's environments are stored.
//
// This aliases cli/env.Manager to keep this compatibility line's existing
// environment implementation and public API intact.
type Manager = env.Manager

type EnvironmentInfo = env.EnvironmentInfo
type GlobalConfig = env.GlobalConfig
type State = env.State
type SerializedVariable = env.SerializedVariable
type UnsavedVariable = env.UnsavedVariable

// NewManager creates an environment manager rooted at basePath. An empty
// envsDirName uses the default "envs" directory.
func NewManager(basePath, envsDirName string) *Manager {
	return env.NewManager(basePath, envsDirName)
}

// DefaultManager returns a new manager rooted where the CLI stores its
// environments. Each call returns an independent manager.
func DefaultManager() *Manager { return env.NewManager("", "") }
