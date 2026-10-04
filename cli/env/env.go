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

// SetBasePath calls Default().SetBasePath.
//
// Deprecated: use Default().SetBasePath instead. Removed in the release after
// the one that deprecated it.
func SetBasePath(path string) { Default().SetBasePath(path) }

// BasePath calls Default().BasePath.
//
// Deprecated: use Default().BasePath instead. Removed in the release after
// the one that deprecated it.
func BasePath() (string, error) { return Default().BasePath() }

// EnvsPath calls Default().EnvsPath.
//
// Deprecated: use Default().EnvsPath instead. Removed in the release after
// the one that deprecated it.
func EnvsPath() (string, error) { return Default().EnvsPath() }

// ResolveEnvPath calls Default().ResolveEnvPath.
//
// Deprecated: use Default().ResolveEnvPath instead. Removed in the release after
// the one that deprecated it.
func ResolveEnvPath(name string) (string, error) { return Default().ResolveEnvPath(name) }

// EnsureDefaultEnvironment calls Default().EnsureDefaultEnvironment.
//
// Deprecated: use Default().EnsureDefaultEnvironment instead. Removed in the release after
// the one that deprecated it.
func EnsureDefaultEnvironment() error { return Default().EnsureDefaultEnvironment() }

// EnsureBaseStructure calls Default().EnsureBaseStructure.
//
// Deprecated: use Default().EnsureBaseStructure instead. Removed in the release after
// the one that deprecated it.
func EnsureBaseStructure() error { return Default().EnsureBaseStructure() }

// Exists calls Default().Exists.
//
// Deprecated: use Default().Exists instead. Removed in the release after
// the one that deprecated it.
func Exists(name string) bool { return Default().Exists(name) }

// Create calls Default().Create.
//
// Deprecated: use Default().Create instead. Removed in the release after
// the one that deprecated it.
func Create(name string) error { return Default().Create(name) }

// Open calls Default().Open.
//
// Deprecated: use Default().Open instead. Removed in the release after
// the one that deprecated it.
func Open(name string) (string, error) { return Default().Open(name) }

// Delete calls Default().Delete.
//
// Deprecated: use Default().Delete instead. Removed in the release after
// the one that deprecated it.
func Delete(name string) error { return Default().Delete(name) }

// Clear calls Default().Clear.
//
// Deprecated: use Default().Clear instead. Removed in the release after
// the one that deprecated it.
func Clear(name string, keepHistory bool) error { return Default().Clear(name, keepHistory) }

// Rename calls Default().Rename.
//
// Deprecated: use Default().Rename instead. Removed in the release after
// the one that deprecated it.
func Rename(oldName, newName string) error { return Default().Rename(oldName, newName) }

// List calls Default().List.
//
// Deprecated: use Default().List instead. Removed in the release after
// the one that deprecated it.
func List() ([]EnvironmentInfo, error) { return Default().List() }

// Info calls Default().Info.
//
// Deprecated: use Default().Info instead. Removed in the release after
// the one that deprecated it.
func Info(name string) (EnvironmentInfo, error) { return Default().Info(name) }

// Export calls Default().Export.
//
// Deprecated: use Default().Export instead. Removed in the release after
// the one that deprecated it.
func Export(name, outputPath string) error { return Default().Export(name, outputPath) }

// Import calls Default().Import.
//
// Deprecated: use Default().Import instead. Removed in the release after
// the one that deprecated it.
func Import(inputPath, targetName string, force bool) (string, error) {
	return Default().Import(inputPath, targetName, force)
}

// GlobalConfigPath calls Default().GlobalConfigPath.
//
// Deprecated: use Default().GlobalConfigPath instead. Removed in the release after
// the one that deprecated it.
func GlobalConfigPath() (string, error) { return Default().GlobalConfigPath() }

// LoadGlobalConfig calls Default().LoadGlobalConfig.
//
// Deprecated: use Default().LoadGlobalConfig instead. Removed in the release after
// the one that deprecated it.
func LoadGlobalConfig() (GlobalConfig, error) { return Default().LoadGlobalConfig() }

// SaveGlobalConfig calls Default().SaveGlobalConfig.
//
// Deprecated: use Default().SaveGlobalConfig instead. Removed in the release after
// the one that deprecated it.
func SaveGlobalConfig(cfg GlobalConfig) error { return Default().SaveGlobalConfig(cfg) }

// UpdateGlobalConfig calls Default().UpdateGlobalConfig.
//
// Deprecated: use Default().UpdateGlobalConfig instead. Removed in the release after
// the one that deprecated it.
func UpdateGlobalConfig(key, value string) (GlobalConfig, error) {
	return Default().UpdateGlobalConfig(key, value)
}

// SaveState calls Default().SaveState.
//
// Deprecated: use Default().SaveState instead. Removed in the release after
// the one that deprecated it.
func SaveState(envName string, vars map[string]any) error { return Default().SaveState(envName, vars) }

// SaveVariables calls Default().SaveVariables.
//
// Deprecated: use Default().SaveVariables instead. Removed in the release after
// the one that deprecated it.
func SaveVariables(envName string, vars map[string]any) ([]UnsavedVariable, error) {
	return Default().SaveVariables(envName, vars)
}

// LoadState calls Default().LoadState.
//
// Deprecated: use Default().LoadState instead. Removed in the release after
// the one that deprecated it.
func LoadState(envName string) (*State, error) { return Default().LoadState(envName) }

// RestoreVariables calls Default().RestoreVariables.
//
// Deprecated: use Default().RestoreVariables instead. Removed in the release after
// the one that deprecated it.
func RestoreVariables(envName string) (map[string]any, error) {
	return Default().RestoreVariables(envName)
}

// AppendHistory calls Default().AppendHistory.
//
// Deprecated: use Default().AppendHistory instead. Removed in the release after
// the one that deprecated it.
func AppendHistory(envName, command string) error { return Default().AppendHistory(envName, command) }

// ReadHistory calls Default().ReadHistory.
//
// Deprecated: use Default().ReadHistory instead. Removed in the release after
// the one that deprecated it.
func ReadHistory(envName string) ([]string, error) { return Default().ReadHistory(envName) }
