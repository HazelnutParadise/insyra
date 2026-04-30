## Reference values for stats/moments.go, stats/skewness.go, stats/kurtosis.go.
## Computed in R 4.5.1 using base R formulas (no external packages required).
##
## Definitions matching insyra's implementation:
##   raw moment of order k     : mean(x^k)
##   central moment of order k : mean((x - mean(x))^k)
##
## Skewness:
##   Type 1 (G1, default): g1 = m3 / m2^(3/2)
##   Type 2 (Adjusted)   : g1 * sqrt(n(n-1)) / (n-2)
##   Type 3 (Bias-adj)   : g1 * ((n-1)/n)^(3/2)
##
## Kurtosis (excess form, i.e. -3 already applied):
##   Type 1 (g2)        : m4 / m2^2 - 3
##   Type 2 (Adjusted)  : ((g2 (n+1) + 6)(n-1)) / ((n-2)(n-3))
##   Type 3 (Bias-adj)  : (g2 + 3) * ((n-1)/n)^2 - 3

options(digits = 17)

fmt <- function(x) {
  if (is.na(x)) return("NaN")
  if (is.infinite(x)) return(if (x > 0) "Inf" else "-Inf")
  formatC(x, digits = 17, format = "g")
}
emit <- function(key, val) cat(key, "=", fmt(val), "\n", sep = "")

raw_moment    <- function(x, k) mean(x^k)
central_moment <- function(x, k) mean((x - mean(x))^k)

emit_moment_set <- function(prefix, x, max_k = 5) {
  for (k in seq_len(max_k)) {
    emit(paste0(prefix, ".raw[", k, "]"),     raw_moment(x, k))
    emit(paste0(prefix, ".central[", k, "]"), central_moment(x, k))
  }
}

emit_skew <- function(prefix, x) {
  n  <- length(x)
  m2 <- central_moment(x, 2)
  m3 <- central_moment(x, 3)
  if (m2 == 0) {
    emit(paste0(prefix, ".g1"),      NaN)
    emit(paste0(prefix, ".adj"),     NaN)
    emit(paste0(prefix, ".biasadj"), NaN)
    return(invisible())
  }
  g1  <- m3 / m2^1.5
  adj <- if (n >= 3) g1 * sqrt(n * (n - 1)) / (n - 2) else NaN
  bia <- g1 * ((n - 1) / n)^1.5
  emit(paste0(prefix, ".g1"),      g1)
  emit(paste0(prefix, ".adj"),     adj)
  emit(paste0(prefix, ".biasadj"), bia)
}

emit_kurt <- function(prefix, x) {
  n  <- length(x)
  m2 <- central_moment(x, 2)
  m4 <- central_moment(x, 4)
  if (m2 == 0) {
    emit(paste0(prefix, ".g2"),      NaN)
    emit(paste0(prefix, ".adj"),     NaN)
    emit(paste0(prefix, ".biasadj"), NaN)
    return(invisible())
  }
  g2  <- m4 / m2^2 - 3
  adj <- if (n >= 4) ((g2 * (n + 1) + 6) * (n - 1)) / ((n - 2) * (n - 3)) else NaN
  bia <- (g2 + 3) * ((n - 1) / n)^2 - 3
  emit(paste0(prefix, ".g2"),      g2)
  emit(paste0(prefix, ".adj"),     adj)
  emit(paste0(prefix, ".biasadj"), bia)
}

# ============================================================
# Moments — existing test dataset
# ============================================================
emit_moment_set("mom_basic", c(2, 4, 7, 1, 8, 3, 9, 2))

# Diverse moment cases
emit_moment_set("mom_smallN",  c(1, 2, 3))                      # n=3
emit_moment_set("mom_largeN",  seq(1, 50, by = 0.5))            # n=99 deterministic
emit_moment_set("mom_neg",     c(-3, -1, 0, 1, 3, 5, -2, 4))   # mixed signs
emit_moment_set("mom_huge",    c(1e6, 2e6, 3e6, 4e6, 5e6))      # large magnitude

# ============================================================
# Skewness — existing test dataset
# ============================================================
emit_skew("sk_basic", c(2, 4, 7, 1, 8, 3, 9, 2))

# Diverse skewness cases
emit_skew("sk_minN",       c(10, 12, 8))                        # n=3 (boundary for Adjusted)
emit_skew("sk_n4",         c(1, 2, 3, 100))                     # heavy positive skew (outlier)
emit_skew("sk_symmetric",  c(1, 2, 3, 4, 5, 6, 7, 8, 9))        # ≈0 skew
emit_skew("sk_negSkew",    c(-100, 1, 2, 3, 4, 5))              # negative skew
emit_skew("sk_largeN",     seq(1, 100))                         # n=100 uniform
emit_skew("sk_huge",       c(1e6, 2e6, 1.5e6, 3e6, 1.2e6))      # large magnitude

# ============================================================
# Kurtosis — existing test dataset
# ============================================================
emit_kurt("ku_basic", c(2, 4, 7, 1, 8, 3, 9, 2))

# Diverse kurtosis cases
emit_kurt("ku_n5",         c(1, 2, 3, 4, 5))                    # platykurtic uniform
emit_kurt("ku_n8",         c(10, 12, 23, 23, 16, 23, 21, 16))   # bimodal-ish
emit_kurt("ku_extreme",    c(1, 100, 1000, 10000, 100000))      # heavy-tail growth
emit_kurt("ku_tight",      c(2.5, 3.5, 2.8, 3.3, 3.0))          # nearly uniform
emit_kurt("ku_largeN",     c(rep(0, 90), rep(10, 10)))          # bimodal binary-ish, n=100
emit_kurt("ku_n4",         c(1, 2, 3, 4))                       # n=4 (boundary for Adjusted)
emit_kurt("ku_huge",       c(1e6, 1.001e6, 0.999e6, 1.0005e6, 1.0015e6))  # tight near 1e6
