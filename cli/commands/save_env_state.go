package commands

import (
	"fmt"

	"github.com/HazelnutParadise/insyra/cli/env"
)

// SaveEnvState saves ctx.Vars to the environment ctx.EnvName. A variable the
// environment cannot store is left out and reported on ctx.Output, once per
// variable and type for the life of ctx, and does not make the save fail. The
// error is for a state that could not be written.
func SaveEnvState(ctx *ExecContext) error {
	if ctx == nil || ctx.EnvName == "" {
		return nil
	}
	mgr := ctx.Env
	if mgr == nil {
		mgr = env.Default()
	}
	unsaved, err := mgr.SaveVariables(ctx.EnvName, ctx.Vars)
	if err != nil {
		// The state was not written, so nothing was reported about any variable
		// and the previous report stands.
		return err
	}
	var warned map[string]string
	if len(unsaved) > 0 {
		warned = make(map[string]string, len(unsaved))
		for _, v := range unsaved {
			warned[v.Name] = v.Type
			if ctx.unsavedWarned[v.Name] == v.Type || ctx.Output == nil {
				continue
			}
			fmt.Fprintf(ctx.Output, "warning: %s (%s) was not saved to environment %s: %s; it is gone when this process ends or another environment is opened\n", v.Name, v.Type, ctx.EnvName, v.Reason)
		}
	}
	// Rebuilt from this save alone: a variable that is gone, or storable again,
	// drops out, and is reported afresh if it becomes unstorable later.
	ctx.unsavedWarned = warned
	return nil
}
