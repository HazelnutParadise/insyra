## High-precision reference values for stats/anova.go and stats/ftest.go.
##
## Sources of truth (all R 4.5.1):
##   one-way ANOVA      -> aov(values ~ factor)
##   two-way ANOVA      -> aov(values ~ A * B)            (Type I sums of squares
##                          ; matches insyra for *balanced* designs only)
##   repeated-measures  -> aov(values ~ cond + Error(subj/cond))
##   var.test           -> stats::var.test
##   Levene's test      -> car::leveneTest(values ~ factor, center = median)
##   Bartlett's test    -> stats::bartlett.test
##
## Eta-squared convention: insyra returns *partial* eta² for between-subjects
## ANOVAs:  ssEffect / (ssEffect + ssWithin).   For one-way ANOVA partial-eta²
## equals classical eta² (= ssEffect/ssTotal). For two-way ANOVA we compute
## partial-eta² explicitly (R's aov() does not give it directly).
##
## Repeated-measures: insyra computes Eta = ssBetween / ssTotal (classical, NOT
## partial). We mirror that here.

options(digits = 17)

fmt <- function(x) {
  if (is.na(x)) return("NaN")
  if (is.infinite(x)) return(if (x > 0) "Inf" else "-Inf")
  formatC(x, digits = 17, format = "g")
}
emit <- function(key, val) cat(key, "=", fmt(val), "\n", sep = "")

# ---- one-way ANOVA ----
emit_oneway <- function(prefix, ...) {
  groups <- list(...)
  k <- length(groups)
  values <- unlist(groups)
  factor_id <- factor(rep(seq_along(groups), sapply(groups, length)))
  fit <- aov(values ~ factor_id)
  s   <- summary(fit)[[1]]
  ssb <- s[["Sum Sq"]][1]
  ssw <- s[["Sum Sq"]][2]
  dfb <- s[["Df"]][1]
  dfw <- s[["Df"]][2]
  fval <- s[["F value"]][1]
  pval <- s[["Pr(>F)"]][1]
  eta_partial <- ssb / (ssb + ssw)  # insyra's etaSquared(ssb, ssw)
  emit(paste0(prefix, ".SSB"), ssb)
  emit(paste0(prefix, ".SSW"), ssw)
  emit(paste0(prefix, ".DFB"), dfb)
  emit(paste0(prefix, ".DFW"), dfw)
  emit(paste0(prefix, ".F"),   fval)
  emit(paste0(prefix, ".P"),   pval)
  emit(paste0(prefix, ".eta"), eta_partial)
  emit(paste0(prefix, ".totalSS"), ssb + ssw)
}

# ---- two-way ANOVA (BALANCED designs; matches insyra's Type I unweighted-cells) ----
emit_twoway <- function(prefix, A_levels, B_levels, cells_list) {
  values   <- c()
  factorA  <- c()
  factorB  <- c()
  for (i in seq_len(A_levels)) {
    for (j in seq_len(B_levels)) {
      cell <- cells_list[[(i - 1) * B_levels + j]]
      values  <- c(values,  cell)
      factorA <- c(factorA, rep(i, length(cell)))
      factorB <- c(factorB, rep(j, length(cell)))
    }
  }
  fA <- factor(factorA); fB <- factor(factorB)
  fit <- aov(values ~ fA * fB)
  s   <- summary(fit)[[1]]
  ssa <- s[["Sum Sq"]][1]; ssb <- s[["Sum Sq"]][2]
  ssab <- s[["Sum Sq"]][3]; ssw <- s[["Sum Sq"]][4]
  dfa <- s[["Df"]][1]; dfb <- s[["Df"]][2]; dfab <- s[["Df"]][3]; dfw <- s[["Df"]][4]
  fa  <- s[["F value"]][1]; fb <- s[["F value"]][2]; fab <- s[["F value"]][3]
  pa  <- s[["Pr(>F)"]][1];  pb <- s[["Pr(>F)"]][2];  pab <- s[["Pr(>F)"]][3]
  # partial eta-squared = SSEffect / (SSEffect + SSWithin), matches insyra
  ea  <- ssa  / (ssa  + ssw)
  eb  <- ssb  / (ssb  + ssw)
  eab <- ssab / (ssab + ssw)
  totalSS <- ssa + ssb + ssab + ssw
  for (k in seq_along(list(
        SSA = ssa, SSB = ssb, SSAB = ssab, SSW = ssw,
        DFA = dfa, DFB = dfb, DFAB = dfab, DFW = dfw,
        FA  = fa,  FB  = fb,  FAB  = fab,
        PA  = pa,  PB  = pb,  PAB  = pab,
        EA  = ea,  EB  = eb,  EAB  = eab,
        totalSS = totalSS))) {
  }
  emit(paste0(prefix, ".SSA"),  ssa); emit(paste0(prefix, ".SSB"),  ssb)
  emit(paste0(prefix, ".SSAB"), ssab); emit(paste0(prefix, ".SSW"),  ssw)
  emit(paste0(prefix, ".DFA"),  dfa); emit(paste0(prefix, ".DFB"),  dfb)
  emit(paste0(prefix, ".DFAB"), dfab); emit(paste0(prefix, ".DFW"),  dfw)
  emit(paste0(prefix, ".FA"),   fa);  emit(paste0(prefix, ".FB"),   fb)
  emit(paste0(prefix, ".FAB"),  fab)
  emit(paste0(prefix, ".PA"),   pa);  emit(paste0(prefix, ".PB"),   pb)
  emit(paste0(prefix, ".PAB"),  pab)
  emit(paste0(prefix, ".EA"),   ea);  emit(paste0(prefix, ".EB"),   eb)
  emit(paste0(prefix, ".EAB"),  eab)
  emit(paste0(prefix, ".totalSS"), totalSS)
}

# ---- repeated-measures ANOVA ----
# Layout: subjects[[j]] is a numeric vector of length C (one value per condition)
emit_rm <- function(prefix, subjects) {
  C <- length(subjects[[1]])
  S <- length(subjects)
  values <- unlist(lapply(subjects, function(s) s))   # subj-major (s1c1, s1c2, ...)
  cond   <- factor(rep(seq_len(C), times = S))
  subj   <- factor(rep(seq_len(S), each  = C))
  fit <- aov(values ~ cond + Error(subj/cond))
  s   <- summary(fit)
  ss_subj <- s[["Error: subj"]][[1]][["Sum Sq"]][1]
  df_subj <- s[["Error: subj"]][[1]][["Df"]][1]
  inner  <- s[["Error: subj:cond"]][[1]]
  ss_cond  <- inner[["Sum Sq"]][1]
  ss_within <- inner[["Sum Sq"]][2]
  df_cond  <- inner[["Df"]][1]
  df_within <- inner[["Df"]][2]
  fval <- inner[["F value"]][1]
  pval <- inner[["Pr(>F)"]][1]
  ssTotal <- ss_subj + ss_cond + ss_within
  eta <- ss_cond / ssTotal   # insyra's RepeatedMeasuresANOVA uses ssBetween/ssTotal
  emit(paste0(prefix, ".SSB"),  ss_cond)
  emit(paste0(prefix, ".SSS"),  ss_subj)
  emit(paste0(prefix, ".SSW"),  ss_within)
  emit(paste0(prefix, ".DFB"),  df_cond)
  emit(paste0(prefix, ".DFS"),  df_subj)
  emit(paste0(prefix, ".DFW"),  df_within)
  emit(paste0(prefix, ".F"),    fval)
  emit(paste0(prefix, ".P"),    pval)
  emit(paste0(prefix, ".eta"),  eta)
  emit(paste0(prefix, ".totalSS"), ssTotal)
}

# ---- F-test for variance equality (two-tailed) ----
# insyra reports F = (larger var) / (smaller var) and the DFs follow whichever
# variance is in the numerator/denominator. Matches R's var.test() up to the
# choice of orientation.
emit_fvar <- function(prefix, x, y) {
  v1 <- var(x); v2 <- var(y)
  n1 <- length(x); n2 <- length(y)
  if (v1 > v2) {
    fval <- v1 / v2
    df1  <- n1 - 1; df2 <- n2 - 1
  } else {
    fval <- v2 / v1
    df1  <- n2 - 1; df2 <- n1 - 1
  }
  # two-tailed: 2 * min(P(F<=fval), P(F>=fval))
  p <- 2 * min(pf(fval, df1, df2), 1 - pf(fval, df1, df2))
  emit(paste0(prefix, ".F"),   fval)
  emit(paste0(prefix, ".P"),   p)
  emit(paste0(prefix, ".DF1"), df1)
  emit(paste0(prefix, ".DF2"), df2)
}

# ---- Levene's test (median-centered, aka Brown-Forsythe) ----
emit_levene <- function(prefix, ...) {
  groups <- list(...)
  diffs  <- c()
  fac    <- c()
  for (i in seq_along(groups)) {
    g <- groups[[i]]
    diffs <- c(diffs, abs(g - median(g)))
    fac   <- c(fac,   rep(i, length(g)))
  }
  fac <- factor(fac)
  fit <- aov(diffs ~ fac)
  s   <- summary(fit)[[1]]
  fval <- s[["F value"]][1]
  pval <- s[["Pr(>F)"]][1]
  df1  <- s[["Df"]][1]; df2 <- s[["Df"]][2]
  emit(paste0(prefix, ".F"),   fval)
  emit(paste0(prefix, ".P"),   pval)
  emit(paste0(prefix, ".DF1"), df1)
  emit(paste0(prefix, ".DF2"), df2)
}

# ---- Bartlett's test ----
emit_bartlett <- function(prefix, ...) {
  groups <- list(...)
  values <- unlist(groups)
  fac    <- factor(rep(seq_along(groups), sapply(groups, length)))
  res    <- bartlett.test(values, fac)
  emit(paste0(prefix, ".Stat"), res$statistic)
  emit(paste0(prefix, ".P"),    res$p.value)
  emit(paste0(prefix, ".DF"),   res$parameter)
}

# ---- F-test for regression (numeric inputs only; no R reference needed beyond
#                            base pf) ----
emit_freg <- function(prefix, ssr, sse, df1, df2) {
  fval <- (ssr / df1) / (sse / df2)
  pval <- 1 - pf(fval, df1, df2)
  emit(paste0(prefix, ".F"), fval)
  emit(paste0(prefix, ".P"), pval)
}

emit_fnest <- function(prefix, rssReduced, rssFull, dfReduced, dfFull) {
  numDF  <- dfReduced - dfFull
  denomDF <- dfFull
  fval <- ((rssReduced - rssFull) / numDF) / (rssFull / denomDF)
  pval <- 1 - pf(fval, numDF, denomDF)
  emit(paste0(prefix, ".F"), fval)
  emit(paste0(prefix, ".P"), pval)
  emit(paste0(prefix, ".DF1"), numDF)
  emit(paste0(prefix, ".DF2"), denomDF)
}

# ============================================================
# Existing test cases (preserved)
# ============================================================
emit_oneway("ow_basic",
  c(10, 12, 9, 11),
  c(20, 19, 21, 22),
  c(30, 29, 28, 32))

emit_twoway("tw_2x2", 2, 2, list(
  c(5, 6, 5),  c(7, 8, 9),
  c(4, 3, 4),  c(10, 11, 9)))

emit_rm("rm_basic", list(
  c(10, 15, 14),
  c(12, 14, 13),
  c(11, 13, 13),
  c(13, 15, 14),
  c(12, 13, 15)))

emit_fvar("fv_basic", c(10, 12, 9, 11), c(20, 19, 21, 22))

emit_levene("levene_basic",
  c(10, 12, 9, 11),
  c(20, 19, 21, 22),
  c(30, 29, 28, 32))

emit_bartlett("bartlett_basic",
  c(10, 12, 9, 11),
  c(20, 19, 21, 22),
  c(30, 29, 28, 32))

emit_freg("freg_basic", 500, 200, 3, 16)
emit_fnest("fnest_basic", 300, 200, 18, 16)

# ============================================================
# Diverse cases
# ============================================================
# One-way: 2 groups, equal sizes
emit_oneway("ow_2grp", c(1, 2, 3, 4, 5), c(2, 4, 6, 8, 10))
# One-way: many groups (5)
emit_oneway("ow_5grp",
  c(1, 2, 3),
  c(2, 3, 4),
  c(3, 4, 5),
  c(4, 5, 6),
  c(5, 6, 7))
# One-way: unequal group sizes
emit_oneway("ow_unequal",
  c(10, 11, 12),
  c(20, 21, 22, 23, 24),
  c(30, 31, 32, 33))
# One-way: large groups, small effect
set.seed(101)
g_a <- round(rnorm(50, 100, 10), 3)
g_b <- round(rnorm(50, 102, 10), 3)
g_c <- round(rnorm(50, 101, 10), 3)
emit_oneway("ow_largeN", g_a, g_b, g_c)
# One-way: huge magnitude
emit_oneway("ow_huge",
  c(1.0e9, 1.0001e9, 0.9999e9),
  c(1.0001e9, 1.0002e9, 1.0e9),
  c(0.9998e9, 0.9999e9, 1.0e9))
# One-way: zero F (equal means)
emit_oneway("ow_zero_f",
  c(10, 11, 12),
  c(11, 12, 10),
  c(12, 10, 11))

# Two-way 2x3 balanced
emit_twoway("tw_2x3", 2, 3, list(
  c(2, 3),  c(4, 5),  c(6, 7),
  c(3, 2),  c(5, 6),  c(7, 8)))

# Two-way 3x3 balanced (n=2 per cell)
emit_twoway("tw_3x3", 3, 3, list(
  c(1, 2),  c(3, 4),  c(5, 6),
  c(2, 3),  c(4, 5),  c(6, 7),
  c(3, 4),  c(5, 6),  c(7, 8)))

# Two-way with strong interaction
emit_twoway("tw_interact", 2, 2, list(
  c(10, 11),  c(20, 21),
  c(20, 21),  c(10, 11)))

# Repeated-measures: small (3 subjects, 4 conditions) — slight noise so SSW > 0
emit_rm("rm_4cond", list(
  c(5,  7, 6, 8),
  c(6,  8, 7, 10),
  c(4,  6, 6, 7)))
# Repeated-measures: many subjects (10), 3 conditions
emit_rm("rm_10sub", list(
  c(10, 12, 14),
  c(11, 13, 15),
  c(12, 14, 16),
  c(10, 11, 13),
  c(11, 12, 14),
  c(9,  11, 13),
  c(10, 13, 15),
  c(11, 14, 16),
  c(12, 13, 15),
  c(10, 12, 13)))

# F-test variance: very different variances
emit_fvar("fv_unequal", c(1, 2, 3, 4, 5), c(10, 30, 50, 70, 90, 100, 5))
# F-test variance: large samples, near equal
set.seed(102)
emit_fvar("fv_largeN", round(rnorm(60, 50, 10), 3), round(rnorm(60, 50, 11), 3))

# Levene: 2 groups
emit_levene("levene_2grp", c(1, 2, 3, 4, 5), c(2, 5, 8, 11, 14))
# Levene: 4 groups, unequal sizes
emit_levene("levene_4grp",
  c(10, 11, 9), c(20, 21, 22, 23),
  c(30, 32), c(40, 39, 41, 42, 38))

# Bartlett: 4 groups
emit_bartlett("bartlett_4grp",
  c(1, 2, 3, 4),
  c(2, 4, 6, 8),
  c(5, 6, 7, 8),
  c(10, 11, 12, 13))

# F-test for regression: large F
emit_freg("freg_large", 1000, 50, 4, 95)
emit_freg("freg_small", 50, 100, 2, 20)
# F-test nested: borderline
emit_fnest("fnest_2", 250, 200, 20, 18)
emit_fnest("fnest_5", 800, 500, 30, 25)
