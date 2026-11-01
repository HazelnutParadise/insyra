package stats

import (
	"math"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra"
)

// hazardD, hazardObs and hazardX are 60 spells simulated in R (set.seed(21));
// testdata/discrete_hazard.R reproduces them and the glm fits below.
var (
	hazardD   = []any{3, 1, 3, 7, 2, 2, 5, 1, 1, 7, 1, 1, 7, 3, 9, 4, 6, 4, 3, 1, 3, 3, 1, 2, 1, 2, 2, 4, 1, 1, 6, 4, 4, 1, 2, 3, 1, 1, 5, 1, 11, 1, 8, 6, 4, 10, 1, 1, 1, 5, 8, 6, 5, 8, 2, 1, 7, 10, 12, 4}
	hazardObs = []any{1, 1, 1, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 1, 0, 1, 1, 1, 1, 1, 0, 1, 0, 1, 0, 0, 1, 0, 1, 1, 1, 1, 0, 1, 1, 1, 1, 1, 1, 1, 0, 1}
	hazardX   = []any{0.79, 0.52, 1.75, -1.27, 2.20, 0.43, -1.57, -0.93, 0.06, 0.00, -2.28, 0.76, -0.55, 0.17, 0.56, 1.51, 0.66, 1.12, -0.78, -0.43, 0.39, 0.04, -1.03, -1.26, -0.23, 0.75, 0.33, -1.12, -0.71, -0.73, -1.83, -0.41, 0.03, 0.91, 1.63, 0.06, 1.85, 0.08, 1.42, 1.46, 0.06, -1.52, -0.05, -0.21, 2.10, 0.20, 0.52, 1.68, 0.39, -1.28, -0.58, 1.77, -0.21, -0.35, 0.59, 1.01, -0.02, -0.90, 0.91, 1.16}
)

func closeRel(got, want, tol float64) bool {
	return math.Abs(got-want) <= tol*(1+math.Abs(want))
}

func TestDiscreteTimeHazardMatchesRGLM(t *testing.T) {
	x := insyra.NewDataList(hazardX...).SetName("x")
	cases := []struct {
		name     string
		opts     DiscreteTimeHazardOptions
		coef, se []float64
		dev      float64
		terms    []string
	}{
		{"logit log", DiscreteTimeHazardOptions{Covariates: []insyra.IDataList{x}},
			[]float64{-0.93341637562706392, -0.39110078604456106, 0.16813719798991172},
			[]float64{0.25703274689409489, 0.22144023864532497, 0.16779046813775075},
			236.33137608369213, []string{"Intercept", "LogPeriod", "x"}},
		{"cloglog log", DiscreteTimeHazardOptions{Covariates: []insyra.IDataList{x}, Link: Cloglog},
			[]float64{-1.10360639721179021, -0.34592181156413643, 0.14622999988016300},
			[]float64{0.22084590738324270, 0.19545401705562521, 0.14650566735905196},
			236.32796910534489, []string{"Intercept", "LogPeriod", "x"}},
		{"logit linear", DiscreteTimeHazardOptions{Baseline: HazardBaselineLinear},
			[]float64{-0.91867576126294892, -0.10799945822542198},
			[]float64{0.275269535274534627, 0.070514476164220449},
			238.32521040292232, []string{"Intercept", "Period"}},
	}
	for _, c := range cases {
		res, err := DiscreteTimeHazard(insyra.NewDataList(hazardD...), insyra.NewDataList(hazardObs...), c.opts)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if len(res.Terms) != len(c.coef) {
			t.Fatalf("%s: %d terms", c.name, len(res.Terms))
		}
		for k, term := range res.Terms {
			if term.Name != c.terms[k] || !closeRel(term.Coefficient, c.coef[k], 1e-6) || !closeRel(term.StdErr, c.se[k], 1e-6) {
				t.Errorf("%s %s: coef %.12g se %.12g, R %.12g %.12g", c.name, term.Name, term.Coefficient, term.StdErr, c.coef[k], c.se[k])
			}
		}
		if !closeRel(res.GLM.Deviance, c.dev, 1e-8) {
			t.Errorf("%s: deviance %.12g, R %.12g", c.name, res.GLM.Deviance, c.dev)
		}
		if res.Duration == nil || res.Duration.Name != c.terms[1] || res.N != 60 || res.Events != 50 {
			t.Errorf("%s: Duration %+v N %d Events %d", c.name, res.Duration, res.N, res.Events)
		}
		if term, ok := res.Term(c.terms[1]); !ok || term != *res.Duration {
			t.Errorf("%s: Term lookup", c.name)
		}
	}
}

func TestDiscreteTimeHazardDummiesMatchKaplanMeier(t *testing.T) {
	// With one level per period and no covariates, the fitted hazard of each
	// period is that period's events over the spells at risk, as Kaplan-Meier
	// counts them.
	d := insyra.NewDataList(1, 1, 2, 2, 2, 3, 3, 3, 3, 1, 2, 3, 3)
	obs := insyra.NewDataList(1, 0, 1, 1, 0, 1, 0, 1, 0, 1, 1, 1, 0)
	res, err := DiscreteTimeHazard(d, obs, DiscreteTimeHazardOptions{Baseline: HazardBaselineDummies})
	if err != nil {
		t.Fatal(err)
	}
	if res.Duration != nil || len(res.Terms) != 3 || res.Terms[2].Name != "Period3" {
		t.Fatalf("terms %+v", res.Terms)
	}
	km, err := KaplanMeier(d, obs)
	if err != nil {
		t.Fatal(err)
	}
	b0 := res.Terms[0].Coefficient
	for k, period := range km.Times {
		eta := b0
		if period > 1 {
			eta += res.Terms[int(period)-1].Coefficient
		}
		want := float64(km.Events[k]) / float64(km.AtRisk[k])
		if got := 1 / (1 + math.Exp(-eta)); !closeRel(got, want, 1e-8) {
			t.Errorf("period %v: hazard %.10g, Kaplan-Meier %.10g", period, got, want)
		}
	}
}

func TestPersonPeriodTable(t *testing.T) {
	dt, err := PersonPeriod(insyra.NewDataList(2, 1, 3), insyra.NewDataList(true, false, 1), insyra.NewDataList(5, 6, 7).SetName("size"))
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(dt.ColNames(), ","); got != "Spell,Period,Event,size" {
		t.Fatalf("columns %s", got)
	}
	want := [][]any{{0, 1, 0, 5.0}, {0, 2, 1, 5.0}, {1, 1, 0, 6.0}, {2, 1, 0, 7.0}, {2, 2, 0, 7.0}, {2, 3, 1, 7.0}}
	if dt.NumRows() != len(want) {
		t.Fatalf("%d rows", dt.NumRows())
	}
	for r, row := range want {
		got := dt.GetRow(r).Data()
		for c := range row {
			if got[c] != row[c] {
				t.Errorf("row %d: %v, want %v", r, got, row)
				break
			}
		}
	}
}

func TestDiscreteTimeHazardRefusesBadInput(t *testing.T) {
	d := insyra.NewDataList(1, 2, 3)
	o := insyra.NewDataList(1, 0, 1)
	cases := map[string]func() error{
		"zero duration": func() error { _, e := DiscreteTimeHazard(insyra.NewDataList(0, 2, 3), o); return e },
		"fraction":      func() error { _, e := DiscreteTimeHazard(insyra.NewDataList(1.5, 2, 3), o); return e },
		"all censored":  func() error { _, e := DiscreteTimeHazard(d, insyra.NewDataList(0, 0, 0)); return e },
		"probit":        func() error { _, e := DiscreteTimeHazard(d, o, DiscreteTimeHazardOptions{Link: Probit}); return e },
		"bad baseline":  func() error { _, e := DiscreteTimeHazard(d, o, DiscreteTimeHazardOptions{Baseline: 9}); return e },
		"two options": func() error {
			_, e := DiscreteTimeHazard(d, o, DiscreteTimeHazardOptions{}, DiscreteTimeHazardOptions{})
			return e
		},
		"short covariate": func() error {
			_, e := DiscreteTimeHazard(d, o, DiscreteTimeHazardOptions{Covariates: []insyra.IDataList{insyra.NewDataList(1, 2)}})
			return e
		},
		"reserved name": func() error {
			_, e := DiscreteTimeHazard(d, o, DiscreteTimeHazardOptions{Covariates: []insyra.IDataList{insyra.NewDataList(1, 2, 3).SetName("Period")}})
			return e
		},
	}
	for name, f := range cases {
		if f() == nil {
			t.Errorf("%s was accepted", name)
		}
	}
	// Dummies with a period in which nothing ended names the period.
	_, err := DiscreteTimeHazard(insyra.NewDataList(1, 2, 2, 3), insyra.NewDataList(1, 0, 0, 1), DiscreteTimeHazardOptions{Baseline: HazardBaselineDummies})
	if err == nil || !strings.Contains(err.Error(), "period 2") {
		t.Errorf("dummies with an empty period: %v", err)
	}
}

func TestDiscreteTimeHazardShows(t *testing.T) {
	res, err := DiscreteTimeHazard(insyra.NewDataList(hazardD...), insyra.NewDataList(hazardObs...))
	if err != nil {
		t.Fatal(err)
	}
	s := res.String()
	for _, want := range []string{"Discrete-time hazard", "Link: logit", "HazardBaselineLog", "Events: 50", "LogPeriod", "PValue"} {
		if !strings.Contains(s, want) {
			t.Errorf("String lacks %q:\n%s", want, s)
		}
	}
	if HazardBaseline(7).String() != "HazardBaseline(7)" {
		t.Error("unknown baseline name")
	}
}
