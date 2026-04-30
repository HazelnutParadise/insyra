## Reference values for stats/ztest.go tests, computed in R 4.5.1 using base
## qnorm/pnorm (R's gold-standard normal CDF/quantile).
##
## CI margin convention (matches insyra after the one-sided fix):
##   two-sided alternative -> qnorm(1 - (1-cl)/2) * SE     (e.g. qnorm(.975) for cl=.95)
##   one-sided alternative -> qnorm(cl) * SE               (e.g. qnorm(.95)  for cl=.95)
## Same as R's BSDA::z.test().
##
## Effect size formulas (preserve insyra's, not Cohen's classical):
##   single  : |mean - mu| / sigma
##   two-sample : |meanDiff| / sqrt((n1*s1^2 + n2*s2^2) / (n1+n2))

options(digits = 17)
fmt <- function(x) {
  if (is.na(x)) return("NaN")
  if (is.infinite(x)) return(if (x > 0) "Inf" else "-Inf")
  formatC(x, digits = 17, format = "g")
}
emit <- function(key, val) cat(key, "=", fmt(val), "\n", sep = "")

# alt: "two.sided" | "greater" | "less"
ztest_single <- function(prefix, x, mu, sigma, alt, cl) {
  n <- length(x)
  m <- mean(x)
  se <- sigma / sqrt(n)
  z <- (m - mu) / se
  p <- switch(alt,
    two.sided = 2 * pnorm(-abs(z)),
    greater   = 1 - pnorm(z),
    less      = pnorm(z))
  margin_two <- qnorm(1 - (1 - cl) / 2) * se
  margin_one <- qnorm(cl) * se
  ci <- switch(alt,
    two.sided = c(m - margin_two, m + margin_two),
    greater   = c(m - margin_one, Inf),
    less      = c(-Inf, m + margin_one))
  d <- abs(m - mu) / sigma
  emit(paste0(prefix, ".z"),    z)
  emit(paste0(prefix, ".p"),    p)
  emit(paste0(prefix, ".ci_lo"), ci[1])
  emit(paste0(prefix, ".ci_hi"), ci[2])
  emit(paste0(prefix, ".mean"), m)
  emit(paste0(prefix, ".n"),    n)
  emit(paste0(prefix, ".d"),    d)
}

ztest_two <- function(prefix, x, y, sigma1, sigma2, alt, cl) {
  n1 <- length(x); n2 <- length(y)
  m1 <- mean(x); m2 <- mean(y); diff <- m1 - m2
  se <- sqrt(sigma1^2/n1 + sigma2^2/n2)
  z <- diff / se
  p <- switch(alt,
    two.sided = 2 * pnorm(-abs(z)),
    greater   = 1 - pnorm(z),
    less      = pnorm(z))
  margin_two <- qnorm(1 - (1 - cl) / 2) * se
  margin_one <- qnorm(cl) * se
  ci <- switch(alt,
    two.sided = c(diff - margin_two, diff + margin_two),
    greater   = c(diff - margin_one, Inf),
    less      = c(-Inf, diff + margin_one))
  pooled_sigma <- sqrt((sigma1^2 + sigma2^2) / 2)   # Cohen's d_av (matches R effectsize)
  d <- abs(diff) / pooled_sigma
  emit(paste0(prefix, ".z"),    z)
  emit(paste0(prefix, ".p"),    p)
  emit(paste0(prefix, ".ci_lo"), ci[1])
  emit(paste0(prefix, ".ci_hi"), ci[2])
  emit(paste0(prefix, ".mean1"), m1)
  emit(paste0(prefix, ".mean2"), m2)
  emit(paste0(prefix, ".n1"),   n1)
  emit(paste0(prefix, ".n2"),   n2)
  emit(paste0(prefix, ".d"),    d)
}

# ============================================================
# Existing single-sample test cases (preserve previous datasets)
# ============================================================
ztest_single("zs1",
  c(56.96714153011233, 50.61735698828815, 58.47688538100692, 67.23029856408026,
    49.658466252766644, 49.6586304305082, 67.79212815507391, 59.67434729152909,
    47.30525614065048, 57.42560043585965),
  50, 10, "two.sided", 0.95)
ztest_single("zs2",
  c(47.365823071875376, 47.34270246429743, 54.41962271566034, 32.86719755342202,
    34.75082167486967, 46.37712470759027, 41.87168879665576, 55.142473325952736,
    42.91975924478789, 37.87696298664709, 66.65648768921554, 49.74223699513465,
    52.67528204687924, 37.75251813786544, 46.556172754748175),
  50, 10, "greater", 0.95)
ztest_single("zs3",
  c(53.10922589709866, 40.490064225776976, 55.75698018345672, 45.99361310081195,
    49.083062502067236, 45.98293387770603, 70.52278184508938, 51.86502775262066,
    41.422890710440996, 60.225449121031886, 39.791563500289776, 54.08863595004755,
    32.403298761202244, 38.71813951101569, 53.96861235869123, 59.38466579995411,
    53.7136828118997, 50.843517176117594, 48.98896304410711, 37.214780096325725),
  50, 10, "less", 0.95)
ztest_single("zs4",
  c(44.801557916052914, 47.393612290402125, 62.57122226218915, 55.43618289568462,
    34.36959844637266, 55.24083969394795, 48.149177195836835, 45.23077999694041,
    58.11676288840868, 62.30999522495951, 61.31280119116199, 43.607824767773614,
    48.907876241487855, 55.31263431403564, 61.75545127122359, 47.2082576215471,
    50.14341023336183, 40.93665025993972, 40.037933759193294, 60.125258223941984,
    65.56240028570824, 51.27989878419666, 62.03532897892024, 55.61636025047634,
    45.54880245394876),
  50, 10, "two.sided", 0.95)

# ============================================================
# Diverse single-sample cases
# ============================================================
ztest_single("zsA_minN", c(50, 52), 50, 5, "two.sided", 0.95)
set.seed(42); ztest_single("zsA_largeN_two", round(rnorm(200, 100, 10), 4), 100, 10, "two.sided", 0.95)
set.seed(43); ztest_single("zsA_largeN_grt", round(rnorm(200, 100, 10), 4), 100, 10, "greater", 0.95)
set.seed(44); ztest_single("zsA_largeN_less", round(rnorm(200, 100, 10), 4), 100, 10, "less", 0.95)
ztest_single("zsA_huge",
  c(1.0e9, 1.05e9, 0.97e9, 1.02e9, 0.99e9, 1.01e9), 1.0e9, 5.0e7, "two.sided", 0.95)
ztest_single("zsA_tiny",
  c(1e-9, 2e-9, 1.5e-9, 0.5e-9, 1.2e-9), 0, 1e-9, "two.sided", 0.95)
ztest_single("zsA_cl99",
  c(20.1, 19.8, 20.5, 20.0, 19.9, 20.3, 20.2), 20, 0.5, "two.sided", 0.99)
ztest_single("zsA_cl90",
  c(20.1, 19.8, 20.5, 20.0, 19.9, 20.3, 20.2), 20, 0.5, "two.sided", 0.90)
ztest_single("zsA_zero_z",
  c(48, 50, 52), 50, 5, "two.sided", 0.95)
ztest_single("zsA_neg_data",
  c(-5, -2, -6, -3, -4, -7, -1, -3, -5, -4), 0, 5, "less", 0.95)
ztest_single("zsA_grt_cl99",
  c(105, 110, 102, 108, 107, 112, 100, 109, 106, 111), 100, 5, "greater", 0.99)

# ============================================================
# Existing two-sample test cases
# ============================================================
ztest_two("zt1",
  c(65.97, 47.63, 57.80, 51.74, 63.65, 60.38, 50.41, 58.52, 58.88, 57.38),
  c(40.76, 43.83, 42.98, 55.62, 61.43, 53.83, 40.01, 51.52, 48.68, 45.75, 38.06, 50.13),
  10, 12, "two.sided", 0.95)
ztest_two("zt2",
  c(48.93, 66.06, 53.40, 55.11, 56.53, 62.99, 54.89, 51.29, 58.49, 53.33,
    58.78, 48.63, 62.46, 52.57, 55.71),
  c(40.63, 48.26, 41.43, 42.70, 39.87, 51.94, 44.94, 54.12, 42.10, 49.86,
    58.14, 51.30, 38.90, 45.13, 41.72),
  10, 12, "greater", 0.95)
ztest_two("zt3",
  c(55.35, 61.14, 61.00, 53.47, 56.38, 56.06, 50.17, 51.50, 61.04, 63.08,
    56.49, 48.32, 48.94, 64.49, 45.36, 51.70, 50.22, 54.16, 51.85, 54.18),
  c(46.89, 53.49, 47.95, 50.52, 48.51, 53.21, 56.18, 43.75, 40.55, 48.88,
    42.79, 42.89, 45.74, 51.65, 44.95, 45.33, 45.52, 55.80, 53.03, 48.86,
    45.91, 41.34, 48.97, 43.75, 48.46),
  10, 12, "less", 0.95)
ztest_two("zt4",
  c(53.11, 52.58, 56.17, 54.85, 59.40, 57.34, 55.61, 49.82, 58.10, 48.55,
    47.38, 57.07, 52.47, 54.85, 57.63, 57.76, 50.61, 47.55, 48.85, 59.08,
    49.53, 51.87, 54.56, 50.96, 58.01, 57.93, 50.88, 58.08, 53.13, 55.87),
  c(48.84, 47.56, 50.28, 53.09, 52.16, 50.62, 55.20, 50.91, 54.53, 51.13,
    48.50, 49.15, 51.12, 47.39, 51.50, 46.57, 51.94, 49.06, 54.28, 46.91,
    52.21, 52.00, 52.40, 53.57, 49.97, 51.17, 54.51, 45.27, 50.86, 52.99),
  10, 12, "two.sided", 0.95)

# ============================================================
# Diverse two-sample cases
# ============================================================
ztest_two("ztA_minBoth", c(50, 52), c(48, 49), 5, 5, "two.sided", 0.95)
ztest_two("ztA_imbalance",
  c(50.0, 51.0, 49.5, 50.5),
  c(45.1, 46.3, 44.8, 45.9, 46.0, 45.5, 46.2, 45.7, 45.8, 46.4,
    45.6, 45.3, 46.1, 45.4, 45.9, 46.0, 45.7, 45.8, 45.6, 45.5),
  3, 3, "greater", 0.95)
ztest_two("ztA_diff_sigma",
  c(10.0, 10.1, 9.9, 10.05, 9.95, 10.02, 9.98),
  c(0, 5, -5, 10, -10, 15, -15, 20),
  0.5, 12, "two.sided", 0.95)
ztest_two("ztA_zero_z",
  c(10, 11, 9, 12, 8), c(11, 9, 10, 8, 12), 5, 5, "two.sided", 0.95)
ztest_two("ztA_cl99",
  c(101, 102, 100, 103, 99), c(95, 94, 96, 93, 97), 2, 2, "two.sided", 0.99)
ztest_two("ztA_cl90_less",
  c(8.1, 8.3, 8.0, 8.4), c(7.5, 7.8, 7.2, 7.9, 7.6, 7.7), 0.5, 0.5, "less", 0.90)
set.seed(7); d1L <- round(rnorm(200, 100, 5), 3)
set.seed(8); d2L <- round(rnorm(250, 102, 5), 3)
ztest_two("ztA_largeN", d1L, d2L, 5, 5, "two.sided", 0.95)
ztest_two("ztA_huge",
  c(1e6, 1.001e6, 0.999e6, 1.0005e6),
  c(1.01e6, 1.011e6, 1.009e6, 1.0105e6),
  5e3, 5e3, "less", 0.95)
