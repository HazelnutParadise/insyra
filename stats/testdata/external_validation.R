## External-validation pass for insyra outputs that R's base functions don't
## return. Compares insyra's documented formulas against the canonical R
## third-party packages (effectsize, effsize, DescTools).
##
## This script PRINTS comparison reports — it does not generate test
## reference files. Used to decide whether insyra's formula choice agrees
## with the R community standard.

options(digits = 12)
suppressPackageStartupMessages({
  library(effectsize)
  library(effsize)
  library(DescTools)
})

cat("================================================================\n")
cat("Cohen's d — t-tests\n")
cat("================================================================\n")

# Single-sample t (data from insyra ttest_test.go basic case)
x_single <- c(52.12, 58.36, 57.49, 51.31, 61.25, 42.88, 46.89, 59.45, 56.20,
              44.39, 48.15, 50.99, 56.84, 47.90, 49.78)
mu <- 50
insyra_single <- (mean(x_single) - mu) / sd(x_single)
es_single <- effectsize::cohens_d(x_single, mu = mu)
cat(sprintf("  one-sample n=15  insyra=%.10g  effectsize=%.10g  diff=%.3g\n",
            insyra_single, es_single$Cohens_d,
            abs(insyra_single - es_single$Cohens_d)))

# Two-sample equal var
x_two <- c(55.1, 49.3, 58.2, 61.9, 47.3, 51.0, 53.8, 59.7, 52.4, 56.1)
y_two <- c(46.9, 41.2, 45.7, 49.8, 44.0, 47.6, 46.5, 43.9, 48.3, 45.4)
v1 <- var(x_two); v2 <- var(y_two); n1 <- length(x_two); n2 <- length(y_two)
pv <- ((n1 - 1) * v1 + (n2 - 1) * v2) / (n1 + n2 - 2)
insyra_two_eq <- (mean(x_two) - mean(y_two)) / sqrt(pv)
es_two_eq <- effectsize::cohens_d(x_two, y_two, pooled_sd = TRUE)
ef_two_eq <- effsize::cohen.d(x_two, y_two, pooled = TRUE)
cat(sprintf("  two-sample EV    insyra=%.10g  effectsize=%.10g  effsize=%.10g\n",
            insyra_two_eq, es_two_eq$Cohens_d, ef_two_eq$estimate))

# Two-sample Welch (insyra: d_av)
insyra_two_uv <- (mean(x_two) - mean(y_two)) / sqrt((v1 + v2) / 2)
es_two_uv <- effectsize::cohens_d(x_two, y_two, pooled_sd = FALSE)
cat(sprintf("  two-sample UV    insyra(d_av)=%.10g  effectsize(d_pooled_unequal)=%.10g\n",
            insyra_two_uv, es_two_uv$Cohens_d))

# Paired
xp <- c(55.0, 48.5, 51.6, 53.2, 50.4, 52.1, 54.0, 56.3)
yp <- c(52.2, 45.7, 49.8, 50.9, 47.6, 50.3, 52.1, 53.9)
d <- xp - yp
insyra_paired <- mean(d) / sd(d)
es_paired <- effectsize::cohens_d(xp, yp, paired = TRUE)
ef_paired <- effsize::cohen.d(xp, yp, paired = TRUE)
cat(sprintf("  paired           insyra=%.10g  effectsize=%.10g  effsize=%.10g\n",
            insyra_paired, es_paired$Cohens_d, ef_paired$estimate))

cat("\n================================================================\n")
cat("Cohen's d — z-tests (no canonical R z-test effect size; compare to\n")
cat("textbook formulas the closest R packages use)\n")
cat("================================================================\n")

# Single-sample z: insyra |mean - mu| / sigma — this is just Cohen's d
# treating sigma as known. Standard formula. effectsize::cohens_d on
# (x, mu) with sigma known doesn't have a direct API; we compare against
# the trivial formula (which is what every textbook gives).
xz <- c(56.96714153, 50.61735698, 58.47688538, 67.23029856, 49.65846625,
        49.65863043, 67.79212815, 59.67434729, 47.30525614, 57.42560043)
sigma <- 10
mu_z <- 50
insyra_zsingle <- abs(mean(xz) - mu_z) / sigma
cat(sprintf("  one-sample z     insyra=%.10g  textbook |x̄−μ|/σ=%.10g\n",
            insyra_zsingle, abs(mean(xz) - mu_z) / sigma))

# Two-sample z: insyra weights pooled σ by sample size:
#    sqrt((n1·σ1² + n2·σ2²) / (n1 + n2))
# Most textbooks (Cohen, Hedges, scipy implementations) use:
#    sqrt((σ1² + σ2²) / 2)    (Cohen's d_av for two known populations)
# Or, for SAMPLE-pooled estimate of the population:
#    sqrt(((n1−1)σ1² + (n2−1)σ2²) / (n1 + n2 − 2))
xz2 <- c(65.97, 47.63, 57.80, 51.74, 63.65, 60.38, 50.41, 58.52, 58.88, 57.38)
yz2 <- c(40.76, 43.83, 42.98, 55.62, 61.43, 53.83, 40.01, 51.52, 48.68, 45.75, 38.06, 50.13)
s1 <- 10; s2 <- 12
n1 <- length(xz2); n2 <- length(yz2)
diff <- mean(xz2) - mean(yz2)
insyra_zT <- abs(diff) / sqrt((n1 * s1^2 + n2 * s2^2) / (n1 + n2))
textbook_dav <- abs(diff) / sqrt((s1^2 + s2^2) / 2)
textbook_pooled <- abs(diff) / sqrt(((n1 - 1) * s1^2 + (n2 - 1) * s2^2) / (n1 + n2 - 2))
cat(sprintf("  two-sample z     insyra(n-weighted)=%.10g\n", insyra_zT))
cat(sprintf("                   textbook d_av=%.10g\n", textbook_dav))
cat(sprintf("                   textbook (n-1)-pooled=%.10g\n", textbook_pooled))

cat("\n================================================================\n")
cat("Partial η² — ANOVA\n")
cat("================================================================\n")

g1 <- c(10, 12, 9, 11)
g2 <- c(20, 19, 21, 22)
g3 <- c(30, 29, 28, 32)
values <- c(g1, g2, g3)
grp <- factor(rep(1:3, each = 4))
fit_one <- aov(values ~ grp)
es_one <- effectsize::eta_squared(fit_one, partial = TRUE)
ssA <- summary(fit_one)[[1]][["Sum Sq"]][1]
ssW <- summary(fit_one)[[1]][["Sum Sq"]][2]
insyra_eta_one <- ssA / (ssA + ssW)
cat(sprintf("  one-way          insyra=%.10g  effectsize(partial)=%.10g  diff=%.3g\n",
            insyra_eta_one, es_one$Eta2_partial,
            abs(insyra_eta_one - es_one$Eta2_partial)))

# Two-way (insyra basic 2x2)
y2w <- c(5, 6, 5, 7, 8, 9, 4, 3, 4, 10, 11, 9)
fA <- factor(rep(1:2, each = 6))
fB <- factor(rep(rep(1:2, each = 3), 2))
fit_two <- aov(y2w ~ fA * fB)
es_two_aov <- effectsize::eta_squared(fit_two, partial = TRUE)
s2 <- summary(fit_two)[[1]]
ssA2 <- s2[["Sum Sq"]][1]; ssB2 <- s2[["Sum Sq"]][2]; ssAB2 <- s2[["Sum Sq"]][3]; ssW2 <- s2[["Sum Sq"]][4]
insyra_eta_A <- ssA2 / (ssA2 + ssW2)
insyra_eta_B <- ssB2 / (ssB2 + ssW2)
insyra_eta_AB <- ssAB2 / (ssAB2 + ssW2)
cat(sprintf("  two-way A        insyra=%.10g  effectsize(partial)=%.10g\n",
            insyra_eta_A, es_two_aov$Eta2_partial[es_two_aov$Parameter == "fA"]))
cat(sprintf("  two-way B        insyra=%.10g  effectsize(partial)=%.10g\n",
            insyra_eta_B, es_two_aov$Eta2_partial[es_two_aov$Parameter == "fB"]))
cat(sprintf("  two-way A:B      insyra=%.10g  effectsize(partial)=%.10g\n",
            insyra_eta_AB, es_two_aov$Eta2_partial[es_two_aov$Parameter == "fA:fB"]))

cat("\n================================================================\n")
cat("Spearman CI — Fisher z-transform\n")
cat("================================================================\n")

xs <- c(1, 2, 3, 4, 5, 6, 7, 8, 9, 10)
ys <- c(1, 2, 3, 5, 4, 6, 8, 7, 10, 9)
rho_s <- cor(xs, ys, method = "spearman")
n_s <- length(xs)
fisher_lo <- function(r, n, cl = 0.95) {
  z <- 0.5 * log((1 + r) / (1 - r))
  se <- 1 / sqrt(n - 3)
  zc <- qnorm(1 - (1 - cl) / 2)
  (exp(2 * (z - zc * se)) - 1) / (exp(2 * (z - zc * se)) + 1)
}
fisher_hi <- function(r, n, cl = 0.95) {
  z <- 0.5 * log((1 + r) / (1 - r))
  se <- 1 / sqrt(n - 3)
  zc <- qnorm(1 - (1 - cl) / 2)
  (exp(2 * (z + zc * se)) - 1) / (exp(2 * (z + zc * se)) + 1)
}
insyra_lo <- fisher_lo(rho_s, n_s)
insyra_hi <- fisher_hi(rho_s, n_s)
desc_ci <- DescTools::SpearmanRho(xs, ys, conf.level = 0.95)
cat(sprintf("  Spearman CI lo   insyra=%.10g  DescTools=%.10g  diff=%.3g\n",
            insyra_lo, desc_ci["lwr.ci"], abs(insyra_lo - desc_ci["lwr.ci"])))
cat(sprintf("  Spearman CI hi   insyra=%.10g  DescTools=%.10g  diff=%.3g\n",
            insyra_hi, desc_ci["upr.ci"], abs(insyra_hi - desc_ci["upr.ci"])))
