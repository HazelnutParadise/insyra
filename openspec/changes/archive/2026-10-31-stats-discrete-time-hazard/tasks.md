# Tasks

- [x] Reference fits from R's `glm` (`stats/testdata/discrete_hazard.R`): logit and cloglog with `log t` and a covariate, logit with `t`.
- [x] Tests first: the fits against R, dummies against Kaplan-Meier's events over at-risk, the person-period table, refused input, `String`; `RegimeDurations` with and without segments and from `ZigZag`.
- [x] `stats/glm_links.go` cloglog and its admission in `resolveGLMLink`; `stats/discrete_hazard.go`; the input readers in `mathutil.go`; `String`/`Show`; `quant/ta/zigzag.go` `RegimeDurations`.
- [x] `Docs/stats.md` and `Docs/quant-ta.md`.
- [x] Entries under `## Unreleased` in both CHANGELOGs.
- [x] `openspec validate stats-discrete-time-hazard --strict` passes.
