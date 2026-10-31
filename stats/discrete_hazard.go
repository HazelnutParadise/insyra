package stats

import (
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/HazelnutParadise/insyra"
)

// HazardBaseline is how DiscreteTimeHazard lets the hazard change with the
// period.
type HazardBaseline int

const (
	// HazardBaselineLog uses log t. Its coefficient does not depend on the unit
	// of the period, and with the Cloglog link it is the shape of a discrete
	// Weibull minus 1, so above 0 means a rising hazard. It is the default.
	HazardBaselineLog HazardBaseline = iota
	// HazardBaselineLinear uses t itself.
	HazardBaselineLinear
	// HazardBaselineDummies gives every period its own level, period 1 being
	// the reference, so the hazard may have any shape.
	HazardBaselineDummies
)

// String returns the constant's name, or HazardBaseline(n) for a value that
// is not one of the constants.
func (b HazardBaseline) String() string {
	switch b {
	case HazardBaselineLog:
		return "HazardBaselineLog"
	case HazardBaselineLinear:
		return "HazardBaselineLinear"
	case HazardBaselineDummies:
		return "HazardBaselineDummies"
	}
	return fmt.Sprintf("HazardBaseline(%d)", int(b))
}

// DiscreteTimeHazardOptions are the optional settings of DiscreteTimeHazard.
type DiscreteTimeHazardOptions struct {
	// Covariates hold one value per spell, constant over its life, named by
	// their list names (X1, X2, ... for an unnamed list).
	Covariates []insyra.IDataList
	// Link is Logit (the default) or Cloglog.
	Link GLMLink
	// Baseline is how the hazard changes with the period; the zero value is
	// HazardBaselineLog.
	Baseline HazardBaseline
	// ConfidenceLevel is the level of the terms' intervals; 0 means 0.95.
	ConfidenceLevel float64
}

// HazardTerm is one term of a DiscreteTimeHazard fit.
type HazardTerm struct {
	Name                                string
	Coefficient, StdErr, ZValue, PValue float64
	CI                                  [2]float64
}

// DiscreteTimeHazardResult is the result of DiscreteTimeHazard.
type DiscreteTimeHazardResult struct {
	// Duration is the term in the period, LogPeriod or Period, whose sign says
	// whether a spell is more (above 0) or less likely to end the longer it has
	// lasted. It is nil for HazardBaselineDummies.
	Duration *HazardTerm
	// Terms are every term in order: Intercept, the baseline terms (LogPeriod,
	// Period, or Period2 ... PeriodK), then the covariates.
	Terms    []HazardTerm
	Link     GLMLink
	Baseline HazardBaseline
	// GLM is the whole binomial fit. Its rows are person-periods, not spells.
	GLM *GLMResult
	// N is the number of spells and Events how many of them ended.
	N, Events int
}

// Term returns the term named name and whether there is one.
func (r *DiscreteTimeHazardResult) Term(name string) (HazardTerm, bool) {
	if r == nil {
		return HazardTerm{}, false
	}
	for _, t := range r.Terms {
		if t.Name == name {
			return t, true
		}
	}
	return HazardTerm{}, false
}

// PersonPeriod expands spells into one row per period each spell was at
// risk, Allison's person-period table: a spell of d periods gives rows for
// periods 1 to d, and Event is 1 only on the last row of a spell that ended.
// The columns are Spell (the spell's 0-based row), Period, Event, then each
// covariate under its list name, repeated on every row of its spell. Fitting
// a binomial GLM of Event on functions of Period and the covariates gives a
// discrete-time hazard model; DiscreteTimeHazard does that, and this table is
// for a model it does not offer, such as an interaction or a time-varying
// covariate.
func PersonPeriod(durations, observed insyra.IDataList, covariates ...insyra.IDataList) (*insyra.DataTable, error) {
	d, flags, err := spells(durations, observed)
	if err != nil {
		return nil, err
	}
	cov, names, err := spellCovariates(covariates, len(d))
	if err != nil {
		return nil, err
	}
	rows := 0
	for _, v := range d {
		rows += v
	}
	spell := make([]any, 0, rows)
	period := make([]any, 0, rows)
	event := make([]any, 0, rows)
	covCols := make([][]any, len(cov))
	for i, v := range d {
		for p := 1; p <= v; p++ {
			spell = append(spell, i)
			period = append(period, p)
			e := 0
			if p == v && flags[i] {
				e = 1
			}
			event = append(event, e)
			for j := range cov {
				covCols[j] = append(covCols[j], cov[j][i])
			}
		}
	}
	cols := []*insyra.DataList{
		insyra.NewDataList(spell...).SetName("Spell"),
		insyra.NewDataList(period...).SetName("Period"),
		insyra.NewDataList(event...).SetName("Event"),
	}
	for j := range covCols {
		cols = append(cols, insyra.NewDataList(covCols[j]...).SetName(names[j]))
	}
	return insyra.NewDataTable(cols...), nil
}

// DiscreteTimeHazard fits a discrete-time hazard model: the chance that a
// spell ends in period t given that it lasted to t, as a function of t and
// of covariates, by Allison's person-period method (a binomial GLM on the
// table PersonPeriod builds). Spells that had not ended when observation
// stopped are used as they are, which is the exact likelihood, not an
// approximation.
//
// durations are whole numbers of periods, at least 1, and observed is true
// (or 1) for a spell that ended and false (or 0) for one still going, as for
// KaplanMeier. A spell that was already running when the data begins must be
// left out by the caller.
//
// The default, log t with the logit link, answers whether a spell is more
// likely to end the longer it lasts: a positive Duration coefficient says
// yes. With the Cloglog link the log t coefficient is the shape of a discrete
// Weibull minus 1. Compare R's glm(Event ~ log(Period) + ..., family =
// binomial(link)) on the same person-period table.
func DiscreteTimeHazard(durations, observed insyra.IDataList, opts ...DiscreteTimeHazardOptions) (*DiscreteTimeHazardResult, error) {
	if len(opts) > 1 {
		return nil, fmt.Errorf("at most one DiscreteTimeHazardOptions may be given, got %d", len(opts))
	}
	var o DiscreteTimeHazardOptions
	if len(opts) == 1 {
		o = opts[0]
	}
	link := o.Link
	switch link {
	case "":
		link = Logit
	case Logit, Cloglog:
	default:
		return nil, fmt.Errorf("link %q is not available; use Logit or Cloglog", link)
	}
	switch o.Baseline {
	case HazardBaselineLog, HazardBaselineLinear, HazardBaselineDummies:
	default:
		return nil, fmt.Errorf("unknown baseline %v", o.Baseline)
	}
	d, flags, err := spells(durations, observed)
	if err != nil {
		return nil, err
	}
	cov, covNames, err := spellCovariates(o.Covariates, len(d))
	if err != nil {
		return nil, err
	}
	events, maxD := 0, 0
	for i, v := range d {
		if flags[i] {
			events++
		}
		maxD = max(maxD, v)
	}
	if events == 0 {
		return nil, errors.New("no spell ended, so the hazard cannot be estimated")
	}
	if o.Baseline == HazardBaselineDummies {
		if maxD < 2 {
			return nil, errors.New("HazardBaselineDummies needs a spell of at least two periods")
		}
		var empty, full []string
		for p := 1; p <= maxD; p++ {
			atRisk, ended := 0, 0
			for i, v := range d {
				if v >= p {
					atRisk++
					if v == p && flags[i] {
						ended++
					}
				}
			}
			switch ended {
			case 0:
				empty = append(empty, fmt.Sprint(p))
			case atRisk:
				full = append(full, fmt.Sprint(p))
			}
		}
		if len(empty) > 0 || len(full) > 0 {
			var parts []string
			if len(empty) > 0 {
				parts = append(parts, "no spell ended in period "+strings.Join(empty, ", "))
			}
			if len(full) > 0 {
				parts = append(parts, "every spell at risk ended in period "+strings.Join(full, ", "))
			}
			return nil, fmt.Errorf("HazardBaselineDummies cannot estimate a period where %s; use HazardBaselineLog or HazardBaselineLinear, or group the periods", strings.Join(parts, " and "))
		}
	}

	var y []any
	var baseNames []string
	var base [][]any
	switch o.Baseline {
	case HazardBaselineLog:
		baseNames = []string{"LogPeriod"}
	case HazardBaselineLinear:
		baseNames = []string{"Period"}
	case HazardBaselineDummies:
		for p := 2; p <= maxD; p++ {
			baseNames = append(baseNames, fmt.Sprintf("Period%d", p))
		}
	}
	base = make([][]any, len(baseNames))
	covRows := make([][]any, len(cov))
	for i, v := range d {
		for p := 1; p <= v; p++ {
			e := 0.0
			if p == v && flags[i] {
				e = 1
			}
			y = append(y, e)
			switch o.Baseline {
			case HazardBaselineLog:
				base[0] = append(base[0], math.Log(float64(p)))
			case HazardBaselineLinear:
				base[0] = append(base[0], float64(p))
			case HazardBaselineDummies:
				for k := range base {
					x := 0.0
					if p == k+2 {
						x = 1
					}
					base[k] = append(base[k], x)
				}
			}
			for j := range cov {
				covRows[j] = append(covRows[j], cov[j][i])
			}
		}
	}
	xs := make([]insyra.IDataList, 0, len(base)+len(covRows))
	for _, b := range base {
		xs = append(xs, insyra.NewDataList(b...))
	}
	for _, c := range covRows {
		xs = append(xs, insyra.NewDataList(c...))
	}
	fit, err := GLM(GLMOptions{Family: Binomial, Link: link, ConfidenceLevel: o.ConfidenceLevel}, insyra.NewDataList(y...), xs...)
	if err != nil {
		return nil, err
	}
	names := append(append([]string{"Intercept"}, baseNames...), covNames...)
	terms := make([]HazardTerm, len(names))
	for k, name := range names {
		terms[k] = HazardTerm{
			Name:        name,
			Coefficient: fit.Coefficients[k],
			StdErr:      fit.StandardErrors[k],
			ZValue:      fit.ZValues[k],
			PValue:      fit.PValues[k],
			CI:          fit.ConfidenceIntervals[k],
		}
	}
	res := &DiscreteTimeHazardResult{
		Terms: terms, Link: link, Baseline: o.Baseline, GLM: fit, N: len(d), Events: events,
	}
	if o.Baseline != HazardBaselineDummies {
		dur := terms[1]
		res.Duration = &dur
	}
	return res, nil
}
