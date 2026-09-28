# Nonparametric Tests When Normality Fails

This tutorial shows how to decide between parametric and rank-based tests,
then run the right Insyra `stats` nonparametric test for the data shape.

## What you will build

You will build a script that:

- screens a sample for the normal-assumption violation,
- routes to the correct rank-based test,
- runs Wilcoxon / Mann-Whitney U / Kruskal-Wallis / Friedman,
- reads the effect size and confidence interval,
- produces a ship/hold-style decision from rank-based evidence.

## Prerequisites

- Go 1.25+.
- Insyra + stats:

```bash
go get github.com/HazelnutParadise/insyra
go get github.com/HazelnutParadise/insyra/stats
```

## Scenario

A product team collected post-launch **satisfaction scores on a 1–5
Likert scale** for two onboarding variants, plus a small follow-up panel
measured under three help-center layouts. Likert data is ordinal and the
samples are small, so the t-test / ANOVA normality assumption does not
hold. You must still produce a defensible decision.

## Decision tree: when to go nonparametric

```text
Is the response interval/ratio AND ~normal per group?
   │ yes                                  │ no
   ▼                                      ▼
Equal variances? (Levene/Bartlett)   Ordinal / small n / heavy tails?
   │ yes        │ no                      │ yes
   ▼            ▼                          ▼
t-test/ANOVA   Welch t-test         Rank-based test:
                                     1 sample / paired → Wilcoxon
                                     2 indep groups    → Mann-Whitney U
                                     k indep groups    → Kruskal-Wallis
                                     k repeated meas.  → Friedman
```

## Step 1: Prepare ordinal sample data

**Goal**
Create Likert-scale samples that violate normality.

**Code**

```go
// Paired: same users rated before vs after the new onboarding copy.
before := insyra.NewDataList(3, 4, 2, 5, 3, 4, 2, 3, 4, 2)
after := insyra.NewDataList(4, 5, 4, 5, 4, 5, 3, 4, 5, 3)

// Two independent variants (different users).
variantA := insyra.NewDataList(4, 5, 3, 4, 5, 4, 3, 5)
variantB := insyra.NewDataList(2, 3, 3, 2, 4, 3, 2, 3, 2)
```

**Expected outcome**
You have ordinal samples unsuitable for a mean-based test.

## Step 2: Paired Wilcoxon for before/after

**Goal**
Test whether satisfaction increased after the change (one-sided).

**Code**

```go
w, err := stats.PairedWilcoxon(before, after, stats.Less)
if err != nil {
	log.Fatal(err)
}
fmt.Printf("Wilcoxon W+=%.1f p=%.4f method=%s r_rb=%.3f\n",
	w.Statistic, w.PValue, w.Method, w.EffectSizes[0].Value)
```

**Expected outcome**
A p-value plus the rank-biserial effect size. `stats.Less` tests
"median(before − after) < 0", i.e. scores went up.

## Step 3: Mann-Whitney U for the two variants

**Goal**
Compare two independent variants without assuming normality.

**Code**

```go
u, err := stats.MannWhitneyU(variantA, variantB, stats.TwoSided)
if err != nil {
	log.Fatal(err)
}
fmt.Printf("MWU U1=%.1f U2=%.1f p=%.4f CI=[%.2f, %.2f] A12=%.3f\n",
	u.U1, u.U2, u.PValue, u.CI[0], u.CI[1], u.EffectSizes[1].Value)
```

**Expected outcome**
`U1`/`U2`, p-value, Hodges-Lehmann shift CI, and the CLES A12 (the
probability a random A rating exceeds a random B rating).

## Step 4: Kruskal-Wallis for k independent groups

**Goal**
Compare three help-center layouts measured on different users.

**Code**

```go
layout1 := insyra.NewDataList(3, 4, 2, 3, 4, 3)
layout2 := insyra.NewDataList(4, 5, 4, 5, 4, 5)
layout3 := insyra.NewDataList(2, 3, 2, 3, 2, 3)

kw, err := stats.KruskalWallis(layout1, layout2, layout3)
if err != nil {
	log.Fatal(err)
}
fmt.Printf("KW H=%.4f df=%.0f p=%.4f eps2=%.3f\n",
	kw.Statistic, *kw.DF, kw.PValue, kw.EffectSizes[0].Value)
```

**Expected outcome**
The tie-corrected H, its χ² df, p-value, and epsilon-squared effect size.

## Step 5: Friedman for repeated measures

**Goal**
Same panelists rated all three layouts — use repeated-measures rank test.

**Code**

```go
// One IDataList per subject; each has k=3 condition scores.
fr, err := stats.FriedmanTest(
	insyra.NewDataList(3, 4, 2),
	insyra.NewDataList(4, 5, 3),
	insyra.NewDataList(2, 4, 2),
	insyra.NewDataList(3, 5, 3),
	insyra.NewDataList(4, 5, 2),
)
if err != nil {
	log.Fatal(err)
}
fmt.Printf("Friedman Q=%.4f df=%.0f p=%.4f W=%.3f\n",
	fr.Statistic, *fr.DF, fr.PValue, fr.EffectSizes[0].Value)
```

**Expected outcome**
The tie-corrected Q, df, p-value, and Kendall's W concordance.

## Step 6: Turn rank-based evidence into a decision

**Goal**
Produce a deterministic recommendation from the Wilcoxon result.

**Code**

```go
decision := "HOLD"
if w.PValue < 0.05 && w.EffectSizes[0].Value < 0 {
	decision = "SHIP_NEW_ONBOARDING"
}
fmt.Println("decision:", decision)
```

**Expected outcome**
A clear recommendation backed by a distribution-free test.

## Complete runnable Go program

```go
package main

import (
	"fmt"
	"log"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats"
)

func main() {
	before := insyra.NewDataList(3, 4, 2, 5, 3, 4, 2, 3, 4, 2)
	after := insyra.NewDataList(4, 5, 4, 5, 4, 5, 3, 4, 5, 3)
	variantA := insyra.NewDataList(4, 5, 3, 4, 5, 4, 3, 5)
	variantB := insyra.NewDataList(2, 3, 3, 2, 4, 3, 2, 3, 2)

	w, err := stats.PairedWilcoxon(before, after, stats.Less)
	if err != nil {
		log.Fatal(err)
	}
	u, err := stats.MannWhitneyU(variantA, variantB, stats.TwoSided)
	if err != nil {
		log.Fatal(err)
	}
	kw, err := stats.KruskalWallis(
		insyra.NewDataList(3, 4, 2, 3, 4, 3),
		insyra.NewDataList(4, 5, 4, 5, 4, 5),
		insyra.NewDataList(2, 3, 2, 3, 2, 3),
	)
	if err != nil {
		log.Fatal(err)
	}
	fr, err := stats.FriedmanTest(
		insyra.NewDataList(3, 4, 2),
		insyra.NewDataList(4, 5, 3),
		insyra.NewDataList(2, 4, 2),
		insyra.NewDataList(3, 5, 3),
		insyra.NewDataList(4, 5, 2),
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Wilcoxon  W+=%.1f p=%.4f method=%s r_rb=%.3f\n",
		w.Statistic, w.PValue, w.Method, w.EffectSizes[0].Value)
	fmt.Printf("MWU       U1=%.1f U2=%.1f p=%.4f A12=%.3f\n",
		u.U1, u.U2, u.PValue, u.EffectSizes[1].Value)
	fmt.Printf("Kruskal   H=%.4f df=%.0f p=%.4f eps2=%.3f\n",
		kw.Statistic, *kw.DF, kw.PValue, kw.EffectSizes[0].Value)
	fmt.Printf("Friedman  Q=%.4f df=%.0f p=%.4f W=%.3f\n",
		fr.Statistic, *fr.DF, fr.PValue, fr.EffectSizes[0].Value)

	decision := "HOLD"
	if w.PValue < 0.05 && w.EffectSizes[0].Value < 0 {
		decision = "SHIP_NEW_ONBOARDING"
	}
	fmt.Println("decision:", decision)
}
```

## Notes on results

- `Method` is `"exact"` for small untied samples and `"asymptotic"`
  (normal approx. + continuity correction) otherwise — auto-selected to
  match R `wilcox.test`.
- Effect sizes: `rank_biserial` (Wilcoxon/MWU), `cles_a12` (MWU only),
  `epsilon_squared` (Kruskal-Wallis), `kendalls_w` (Friedman).
- `CI` is the Hodges-Lehmann interval (pseudo-median for Wilcoxon,
  location shift for Mann-Whitney). Kruskal-Wallis / Friedman do not
  return a CI (`CI` is `nil`).

## Where to go next

- [stats](../stats.md) — full nonparametric API and algorithm notes
- [A/B Test Decision with Statistics](./ab-test-decision-with-statistics.md)
- [DataList](../DataList.md)
