// fa/objects_index.go
package fa

// ObjectsIndex contains mappings of functions to files.
// Mirrors objects_index.R
var ObjectsIndex = map[string]string{
	"psych::fac":           "psych_fac_full.R",
	"psych::fa":            "psych_fa_wrapper.R",
	"psych::smc":           "psych_smc.R",
	"psych::Pinv":          "psych_Pinv.R",
	"psych::factor.stats":  "psych_factor_stats.R",
	"psych::faRotations":   "psych_faRotations.R",
	"psych::Promax":        "psych_Promax.R",
	"psych::target.rot":    "psych_target_rot.R",
	"psych::glb.algebraic": "psych_glb_algebraic.R",
	"GPArotation::GPForth": "GPArotation_GPForth.R",
	"GPArotation::GPFoblq": "GPArotation_GPFoblq.R",
	// ... and so on
}
