## Generates high-precision reference values for stats/ttest.go tests.
## Output: stdout in `key = value` lines, value in 17 significant digits.
## Run: Rscript ttest_reference.R > ttest_reference.txt

options(digits = 17)
fmt <- function(x) {
  if (is.na(x)) return("NaN")
  if (is.infinite(x)) return(if (x > 0) "Inf" else "-Inf")
  formatC(x, digits = 17, format = "g")
}
emit <- function(key, val) cat(key, "=", fmt(val), "\n", sep = "")

# Cohen's d formulas as implemented in insyra (preserve sign):
#   single:        (mean - mu) / sd
#   two-sample EV: (m1 - m2) / sqrt(pooledVar)            pooledVar = ((n1-1)v1 + (n2-1)v2) / (n1+n2-2)
#   two-sample UV: (m1 - m2) / sqrt((v1 + v2) / 2)
#   paired:        |meanDiff| / sd(diff)                  -- absolute value, no sign
cohen_single <- function(x, mu)        (mean(x) - mu) / sd(x)
cohen_two_eq <- function(x, y) {
  v1 <- var(x); v2 <- var(y); n1 <- length(x); n2 <- length(y)
  pv <- ((n1 - 1) * v1 + (n2 - 1) * v2) / (n1 + n2 - 2)
  (mean(x) - mean(y)) / sqrt(pv)
}
cohen_two_uneq <- function(x, y) {
  v1 <- var(x); v2 <- var(y)
  (mean(x) - mean(y)) / sqrt((v1 + v2) / 2)
}
cohen_paired <- function(x, y) {
  d <- x - y
  mean(d) / sd(d)        # sign-preserving (Cohen's d_z)
}

emit_single <- function(prefix, x, mu, cl = 0.95) {
  res <- t.test(x, mu = mu, conf.level = cl)
  emit(paste0(prefix, ".t"),    res$statistic)
  emit(paste0(prefix, ".p"),    res$p.value)
  emit(paste0(prefix, ".df"),   res$parameter)
  emit(paste0(prefix, ".ci_lo"), res$conf.int[1])
  emit(paste0(prefix, ".ci_hi"), res$conf.int[2])
  emit(paste0(prefix, ".mean"), mean(x))
  emit(paste0(prefix, ".n"),    length(x))
  emit(paste0(prefix, ".d"),    cohen_single(x, mu))
}

emit_two <- function(prefix, x, y, equalVar, cl = 0.95) {
  res <- t.test(x, y, var.equal = equalVar, conf.level = cl)
  emit(paste0(prefix, ".t"),    res$statistic)
  emit(paste0(prefix, ".p"),    res$p.value)
  emit(paste0(prefix, ".df"),   res$parameter)
  emit(paste0(prefix, ".ci_lo"), res$conf.int[1])
  emit(paste0(prefix, ".ci_hi"), res$conf.int[2])
  emit(paste0(prefix, ".mean1"), mean(x))
  emit(paste0(prefix, ".mean2"), mean(y))
  emit(paste0(prefix, ".n1"),    length(x))
  emit(paste0(prefix, ".n2"),    length(y))
  d <- if (equalVar) cohen_two_eq(x, y) else cohen_two_uneq(x, y)
  emit(paste0(prefix, ".d"),     d)
}

emit_paired <- function(prefix, x, y, cl = 0.95) {
  res <- t.test(x, y, paired = TRUE, conf.level = cl)
  emit(paste0(prefix, ".t"),    res$statistic)
  emit(paste0(prefix, ".p"),    res$p.value)
  emit(paste0(prefix, ".df"),   res$parameter)
  emit(paste0(prefix, ".ci_lo"), res$conf.int[1])
  emit(paste0(prefix, ".ci_hi"), res$conf.int[2])
  emit(paste0(prefix, ".meandiff"), mean(x - y))
  emit(paste0(prefix, ".n"),     length(x))
  emit(paste0(prefix, ".d"),     cohen_paired(x, y))
}

# ---- Single-sample cases ----
emit_single("single1", c(52.12, 58.36, 57.49, 51.31, 61.25, 42.88, 46.89, 59.45, 56.20, 44.39, 48.15, 50.99, 56.84, 47.90, 49.78), 50)
emit_single("single2", c(49.55, 42.01, 50.22, 52.88, 42.75, 47.36, 54.17, 41.29, 45.63, 47.48, 55.81, 44.10, 43.75, 46.52, 50.94, 53.60, 43.07, 48.77, 46.82, 50.90), 50)
emit_single("single3", c(12, 14, 15, 11, 13), 10)
emit_single("single4", c(9.8, 10.1, 9.9, 10.2, 9.7, 10.0, 10.3, 9.6, 10.4, 10.0, 9.9, 10.1, 9.8, 10.2, 10.0, 9.7, 10.3, 9.6, 10.4, 9.9), 10)
emit_single("single7", c(-5, -2, -6, -3, -4), 0)
emit_single("single8", c(10.01, 9.99, 10.02, 9.98, 10.00), 10)

# ---- Two-sample cases ----
emit_two("two1", c(55.1, 49.3, 58.2, 61.9, 47.3, 51.0, 53.8, 59.7, 52.4, 56.1), c(46.9, 41.2, 45.7, 49.8, 44.0, 47.6, 46.5, 43.9, 48.3, 45.4), TRUE)
emit_two("two2", c(55.1, 49.3, 58.2, 61.9, 47.3, 51.0, 53.8, 59.7, 52.4, 56.1), c(46.9, 41.2, 45.7, 49.8, 44.0, 47.6, 46.5, 43.9, 48.3, 45.4), FALSE)
emit_two("two3", c(50.2, 48.5, 52.1, 51.3, 50.5, 49.9, 48.7, 50.8), c(47.9, 48.1, 49.5, 50.2, 48.8, 49.7, 48.9, 49.1), TRUE)
emit_two("two4", c(60.0, 59.5, 58.8, 60.2, 59.9), c(55.1, 52.3, 58.4, 53.6, 54.2, 53.1), FALSE)
emit_two("two5", c(10.1, 10.3, 10.0, 9.9, 10.2), c(9.8, 9.7, 9.9, 10.0, 9.6, 9.5, 9.4), TRUE)
emit_two("two6", c(100, 105, 98, 110, 102, 103), c(80, 85, 78, 90, 82), FALSE)
emit_two("two7", c(0.01, 0.02, 0.015, 0.025), c(-0.01, -0.005, -0.012, -0.008), TRUE)
emit_two("two8", c(-5, -4, -6, -3, -5), c(1, 2, 0, 3, 1, 2), FALSE)
emit_two("two9", c(10, 12, 10, 11, 12, 10), c(8, 9, 8, 7, 9, 8), TRUE)

# ---- Paired ----
emit_paired("paired1",
  c(55.0, 48.5, 51.6, 53.2, 50.4, 52.1, 54.0, 56.3),
  c(52.2, 45.7, 49.8, 50.9, 47.6, 50.3, 52.1, 53.9))

# ============================================================
# Additional diverse cases (per user request: cover edge cases)
# ============================================================

# ---- Single-sample diverse ----
# minimal n=2
emit_single("singleA_minN", c(7, 9), 5)
# large n=100 (rnorm-style fixed values, deterministic)
set.seed(42)
emit_single("singleA_largeN", round(rnorm(100, mean = 5, sd = 2), 4), 5)
# huge magnitude
emit_single("singleA_huge", c(1.0e9, 1.05e9, 0.97e9, 1.02e9, 0.99e9, 1.01e9), 1.0e9)
# tiny magnitude
emit_single("singleA_tiny", c(1e-9, 2e-9, 1.5e-9, 0.5e-9, 1.2e-9), 0)
# 99% confidence
emit_single("singleA_cl99", c(20.1, 19.8, 20.5, 20.0, 19.9, 20.3, 20.2), 20, 0.99)
# 90% confidence
emit_single("singleA_cl90", c(20.1, 19.8, 20.5, 20.0, 19.9, 20.3, 20.2), 20, 0.90)
# heavy positive skew
emit_single("singleA_skew", c(1, 2, 2, 3, 3, 3, 4, 4, 5, 100), 4)
# all integers, ties
emit_single("singleA_ties", c(5, 5, 5, 6, 6, 6, 7, 7, 7, 8), 5)

# ---- Two-sample diverse ----
# minimal n1=n2=2 (df=2 / Welch can give wild df)
emit_two("twoA_minBoth", c(10, 12), c(8, 9), TRUE)
# very unequal sample sizes (Welch territory)
emit_two("twoA_imbalance", c(50.0, 51.0, 49.5, 50.5),
  c(45.1, 46.3, 44.8, 45.9, 46.0, 45.5, 46.2, 45.7, 45.8, 46.4,
    45.6, 45.3, 46.1, 45.4, 45.9, 46.0, 45.7, 45.8, 45.6, 45.5),
  FALSE)
# very different variances (one tight, one wide)
emit_two("twoA_diffVar", c(10.0, 10.1, 9.9, 10.05, 9.95, 10.02, 9.98),
  c(0, 5, -5, 10, -10, 15, -15, 20), FALSE)
# both means equal, t≈0
emit_two("twoA_zeroT", c(10, 11, 9, 12, 8), c(11, 9, 10, 8, 12), TRUE)
# 99% CL equal var
emit_two("twoA_cl99", c(101, 102, 100, 103, 99), c(95, 94, 96, 93, 97), TRUE, 0.99)
# 90% CL Welch
emit_two("twoA_cl90Welch", c(8.1, 8.3, 8.0, 8.4), c(7.5, 7.8, 7.2, 7.9, 7.6, 7.7), FALSE, 0.90)
# huge n: 200 vs 250
set.seed(7)
emit_two("twoA_largeN",
  round(rnorm(200, mean = 100, sd = 5), 3),
  round(rnorm(250, mean = 102, sd = 5), 3),
  TRUE)
# big magnitude diff
emit_two("twoA_huge", c(1e6, 1.001e6, 0.999e6, 1.0005e6),
  c(1.01e6, 1.011e6, 1.009e6, 1.0105e6), TRUE)

# ---- Paired diverse ----
# minimal n=2
emit_paired("pairedA_minN", c(10, 12), c(8, 9))
# all-zero diff (degenerate, but t.test rejects this; covered by Go-side edge test instead)
# negative meanDiff (after > before)
emit_paired("pairedA_neg", c(50, 48, 52, 49, 51), c(53, 50, 54, 51, 53))
# some pairs with zero diff
emit_paired("pairedA_someZero", c(10, 11, 12, 13, 14), c(10, 10, 12, 12, 14))
# large n=40
set.seed(11)
b <- round(rnorm(40, mean = 100, sd = 10), 3)
a <- b + round(rnorm(40, mean = 2, sd = 1), 3)
emit_paired("pairedA_largeN", a, b)
# 99% CL
emit_paired("pairedA_cl99",
  c(20.1, 19.8, 20.5, 20.0, 19.9, 20.3, 20.2),
  c(19.0, 18.5, 19.5, 19.1, 18.8, 19.4, 19.2), 0.99)
# huge magnitude (small but non-constant diffs)
emit_paired("pairedA_huge",
  c(1.000000e9, 1.000100e9, 0.999900e9, 1.000050e9, 1.000200e9),
  c(1.000080e9, 1.000150e9, 1.000010e9, 1.000180e9, 1.000220e9))
