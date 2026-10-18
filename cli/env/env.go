// Package env stores the CLI's named environments: each one's variables,
// command history and configuration, under <UserHomeDir>/.insyra/envs/<name>/
// by default. A Manager owns one storage root; Default is the one the insyra
// command uses. A program embedding the command language gets its Manager,
// and the types its methods use, from engine/dsl.
package env

import "github.com/HazelnutParadise/insyra/internal/dsl/env"

// Manager owns where environments live and reads and writes them. Every
// operation on an environment is one of its methods.
//
// Deprecated: use Manager from engine/dsl instead, the same type.
// Removed in the release after the one that deprecated it.
type Manager = env.Manager

// EnvironmentInfo describes one environment, as Manager.List and Manager.Info
// report it.
//
// Deprecated: use EnvironmentInfo from engine/dsl instead, the same type.
// Removed in the release after the one that deprecated it.
type EnvironmentInfo = env.EnvironmentInfo

// ExportPayload is the file Manager.Export writes and Manager.Import reads.
type ExportPayload = env.ExportPayload

// GlobalConfig is the configuration shared by every environment of a Manager.
//
// Deprecated: use GlobalConfig from engine/dsl instead, the same type.
// Removed in the release after the one that deprecated it.
type GlobalConfig = env.GlobalConfig

// State is an environment's saved variables, as state.json holds them.
//
// Deprecated: use State from engine/dsl instead, the same type.
// Removed in the release after the one that deprecated it.
type State = env.State

// SerializedVariable is one variable as state.json holds it.
//
// Deprecated: use SerializedVariable from engine/dsl instead, the same type.
// Removed in the release after the one that deprecated it.
type SerializedVariable = env.SerializedVariable

// UnsavedVariable names a variable Manager.SaveVariables could not store, with
// its Go type and the reason.
//
// Deprecated: use UnsavedVariable from engine/dsl instead, the same type.
// Removed in the release after the one that deprecated it.
type UnsavedVariable = env.UnsavedVariable

// NewManager returns a Manager rooted at basePath, keeping each environment in
// basePath/envsDirName/<name>/. "" for basePath means <UserHomeDir>/.insyra,
// and "" for envsDirName means "envs".
//
// Deprecated: use NewManager from engine/dsl instead, the same function.
// Removed in the release after the one that deprecated it.
func NewManager(basePath, envsDirName string) *Manager {
	return env.NewManager(basePath, envsDirName)
}

// Default returns the shared process-wide Manager the insyra command uses,
// rooted at <UserHomeDir>/.insyra unless Default().SetBasePath moves it. A
// program embedding the command language wants DefaultManager from engine/dsl,
// which returns a Manager of its own at the same place.
func Default() *Manager { return env.Default() }

// ConfigKeys lists the keys `insyra config <key> <value>` accepts.
func ConfigKeys() []string { return env.ConfigKeys() }

// The functions below only call the same method on Default(). Each is
// Deprecated in favour of calling the method.
