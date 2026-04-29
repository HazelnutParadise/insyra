## Reference values for stats/correlation.go computed in R 4.5.1.
##
## Sources of truth:
##   Pearson r        -> cor(x, y, method="pearson")
##   Pearson p, df    -> cor.test(x, y, method="pearson") -> Student t (n-2 df)
##   Pearson Fisher CI-> direct formula matching insyra's pearsonFisherCI
##   Spearman rho     -> cor(x, y, method="spearman")  (computed as Pearson on ranks)
##   Spearman p, df   -> insyra reuses the Pearson t-formula on rho with df=n-2.
##                       R's cor.test default uses an asymptotic AS-89 algorithm
##                       which gives a slightly different p-value; we mirror
##                       insyra's t-based p instead.
##   Kendall tau      -> cor(x, y, method="kendall")  (Kendall's tau-b)
##   Kendall p        -> insyra-specific:
##                         n <= 7  : exact two-sided permutation p (using tau-b)
##                         n  > 7  : 2*(1 - pnorm(|z|)) with
##                                   z = S / sqrt(var(S))
##                                   S = concordant - discordant
##                                   var(S) = (n(n-1)(2n+5) - T1 - T2) / 18
##                                   T1 = Σ tx(tx-1)(2tx+5)  (tie groups in x)
##                                   T2 = Σ ty(ty-1)(2ty+5)  (tie groups in y)
##                       Reduces to Kendall's classical no-ties formula when
##                       T1=T2=0; for tied data this is the standard tie-
##                       corrected asymptotic (matches scipy.stats.kendalltau).
##
## Covariance   -> stats::cov(x, y) (sample covariance, n-1 divisor) — matches insyra.
##
## Bartlett sphericity -> manual: chi^2 = -((n-1) - (2p+5)/6) * log(det(R))
##                       df = p*(p-1)/2.   (psych::cortest.bartlett gives the same.)

options(digits = 17)

fmt <- function(x) {
  if (is.na(x)) return("NaN")
  if (is.infinite(x)) return(if (x > 0) "Inf" else "-Inf")
  formatC(x, digits = 17, format = "g")
}
emit <- function(key, val) cat(key, "=", fmt(val), "\n", sep = "")

fisher_ci <- function(r, n, cl = 0.95) {
  # Mirror insyra/mathutil.go:pearsonFisherCI edge cases
  if (n <= 3)   return(c(NaN, NaN))
  if (r >= 1)   return(c(1, 1))
  if (r <= -1)  return(c(-1, -1))
  z   <- 0.5 * log((1 + r) / (1 - r))
  se  <- 1 / sqrt(n - 3)
  zc  <- qnorm(1 - (1 - cl) / 2)
  lo  <- (exp(2 * (z - zc * se)) - 1) / (exp(2 * (z - zc * se)) + 1)
  hi  <- (exp(2 * (z + zc * se)) - 1) / (exp(2 * (z + zc * se)) + 1)
  c(lo, hi)
}

# Generate all permutations of a vector (base R, no extra packages).
all_perms <- function(v) {
  n <- length(v)
  if (n == 1) return(matrix(v, 1, 1))
  out <- NULL
  for (i in seq_len(n)) {
    sub <- all_perms(v[-i])
    rows <- cbind(rep(v[i], nrow(sub)), sub)
    out <- if (is.null(out)) rows else rbind(out, rows)
  }
  out
}

# Insyra's Kendall p-value (mirrors kendallCorrelationWithStats logic)
kendall_p_insyra <- function(x, y) {
  n <- length(x)
  tau <- cor(x, y, method = "kendall")
  if (n <= 7) {
    sorted_y <- sort(y)
    perms <- all_perms(sorted_y)
    obs <- abs(tau)
    extreme <- 0
    for (k in seq_len(nrow(perms))) {
      alt_tau <- cor(x, perms[k, ], method = "kendall")
      if (abs(alt_tau) >= obs) extreme <- extreme + 1
    }
    return(extreme / nrow(perms))
  } else {
    # tie-corrected asymptotic: z = S / sqrt(var(S))
    sval <- 0
    for (i in seq_len(n - 1)) {
      for (j in seq(i + 1, n)) {
        dx <- x[i] - x[j]; dy <- y[i] - y[j]
        if (dx == 0 || dy == 0) next
        if ((dx > 0) == (dy > 0)) sval <- sval + 1 else sval <- sval - 1
      }
    }
    tx <- table(x); tx <- tx[tx > 1]
    ty <- table(y); ty <- ty[ty > 1]
    T1 <- sum(tx * (tx - 1) * (2 * tx + 5))
    T2 <- sum(ty * (ty - 1) * (2 * ty + 5))
    varS <- (n * (n - 1) * (2 * n + 5) - T1 - T2) / 18
    if (varS <= 0) return(NaN)
    z <- sval / sqrt(varS)
    return(2 * (1 - pnorm(abs(z))))
  }
}

emit_pearson <- function(prefix, x, y) {
  r <- cor(x, y, method = "pearson")
  n <- length(x)
  df <- n - 2
  t  <- r * sqrt(n - 2) / sqrt(1 - r * r)
  p  <- 2 * (1 - pt(abs(t), df))
  ci <- fisher_ci(r, n, 0.95)
  emit(paste0(prefix, ".r"),    r)
  emit(paste0(prefix, ".p"),    p)
  emit(paste0(prefix, ".df"),   df)
  emit(paste0(prefix, ".cilo"), ci[1])
  emit(paste0(prefix, ".cihi"), ci[2])
}

emit_spearman <- function(prefix, x, y) {
  rho <- cor(x, y, method = "spearman")
  n   <- length(x)
  df  <- n - 2
  t   <- rho * sqrt(n - 2) / sqrt(1 - rho * rho)
  p   <- 2 * (1 - pt(abs(t), df))
  ci  <- fisher_ci(rho, n, 0.95)
  emit(paste0(prefix, ".rho"),  rho)
  emit(paste0(prefix, ".p"),    p)
  emit(paste0(prefix, ".df"),   df)
  emit(paste0(prefix, ".cilo"), ci[1])
  emit(paste0(prefix, ".cihi"), ci[2])
}

emit_kendall <- function(prefix, x, y) {
  tau <- cor(x, y, method = "kendall")
  p   <- kendall_p_insyra(x, y)
  emit(paste0(prefix, ".tau"), tau)
  emit(paste0(prefix, ".p"),   p)
}

emit_cov <- function(prefix, x, y) {
  emit(paste0(prefix, ".cov"), cov(x, y))
}

emit_bartlett_sph <- function(prefix, mat) {
  cols <- ncol(mat)
  rows <- nrow(mat)
  R    <- cor(mat)
  d    <- det(R)
  n    <- rows
  p    <- cols
  chisq <- -((n - 1) - (2 * p + 5) / 6) * log(d)
  df    <- p * (p - 1) / 2
  pval  <- 1 - pchisq(chisq, df)
  emit(paste0(prefix, ".chisq"), chisq)
  emit(paste0(prefix, ".p"),     pval)
  emit(paste0(prefix, ".df"),    df)
}

# ============================================================
# Existing test datasets — preserved
# ============================================================
emit_pearson("p_existing", c(10, 20, 30, 40, 50), c(15, 22, 29, 41, 48))
emit_spearman("s_existing", c(10, 20, 30, 40, 50), c(15, 22, 29, 41, 48))
emit_kendall("k_existing", c(10, 20, 30, 40, 50), c(15, 22, 29, 41, 48))

# Existing TestCorrelation_MoreCases data
emit_pearson("p_perfectpos", c(1, 2, 3, 4, 5), c(10, 20, 30, 40, 50))
emit_spearman("s_perfectpos", c(1, 2, 3, 4, 5), c(10, 20, 30, 40, 50))
emit_kendall("k_perfectpos", c(1, 2, 3, 4, 5), c(10, 20, 30, 40, 50))

emit_pearson("p_perfectneg", c(1, 2, 3, 4, 5), c(50, 40, 30, 20, 10))
emit_spearman("s_perfectneg", c(1, 2, 3, 4, 5), c(50, 40, 30, 20, 10))
emit_kendall("k_perfectneg", c(1, 2, 3, 4, 5), c(50, 40, 30, 20, 10))

emit_pearson("p_random", c(5, 1, 3, 2, 4), c(10, 7, 8, 6, 9))
emit_spearman("s_random", c(5, 1, 3, 2, 4), c(10, 7, 8, 6, 9))
emit_kendall("k_random", c(5, 1, 3, 2, 4), c(10, 7, 8, 6, 9))

# ============================================================
# Diverse cases
# ============================================================
# Moderate positive correlation, n=10
emit_pearson("p_n10", c(1, 2, 3, 4, 5, 6, 7, 8, 9, 10),
                       c(1, 2, 3, 5, 4, 6, 8, 7, 10, 9))
emit_spearman("s_n10", c(1, 2, 3, 4, 5, 6, 7, 8, 9, 10),
                       c(1, 2, 3, 5, 4, 6, 8, 7, 10, 9))
emit_kendall("k_n10", c(1, 2, 3, 4, 5, 6, 7, 8, 9, 10),
                       c(1, 2, 3, 5, 4, 6, 8, 7, 10, 9))

# Larger n=50, weak correlation
set.seed(901); xL <- round(rnorm(50, 0, 1), 5)
set.seed(902); yL <- round(rnorm(50, 0, 1), 5) + 0.3 * xL
emit_pearson("p_n50", xL, yL)
emit_spearman("s_n50", xL, yL)
emit_kendall("k_n50", xL, yL)

# Negative weak correlation, n=20
set.seed(903); xN <- round(rnorm(20, 0, 1), 5)
set.seed(904); yN <- round(-0.5 * xN + rnorm(20, 0, 1), 5)
emit_pearson("p_n20neg", xN, yN)
emit_spearman("s_n20neg", xN, yN)
emit_kendall("k_n20neg", xN, yN)

# Tied data (Spearman/Kendall handle ties)
emit_pearson("p_ties", c(1, 2, 2, 3, 3, 3, 4, 5, 5, 6),
                       c(1, 1, 2, 3, 3, 4, 5, 5, 6, 7))
emit_spearman("s_ties", c(1, 2, 2, 3, 3, 3, 4, 5, 5, 6),
                       c(1, 1, 2, 3, 3, 4, 5, 5, 6, 7))
emit_kendall("k_ties", c(1, 2, 2, 3, 3, 3, 4, 5, 5, 6),
                      c(1, 1, 2, 3, 3, 4, 5, 5, 6, 7))

# Huge magnitude
emit_pearson("p_huge", c(1.0e9, 1.0001e9, 0.9999e9, 1.00005e9, 1.00015e9),
                       c(2.0e9, 2.0002e9, 1.9998e9, 2.00010e9, 2.00030e9))
# n=8 Kendall (boundary: > 7 uses normal approx)
emit_kendall("k_n8", c(1, 2, 3, 4, 5, 6, 7, 8),
                       c(2, 1, 4, 3, 6, 5, 8, 7))

# ============================================================
# Covariance
# ============================================================
emit_cov("cov_basic", c(10, 20, 30, 40, 50), c(15, 22, 29, 41, 48))
emit_cov("cov_neg", c(1, 2, 3, 4, 5), c(50, 40, 30, 20, 10))
set.seed(999); cx <- round(rnorm(30, 100, 10), 3)
set.seed(998); cy <- round(rnorm(30, 100, 10), 3)
emit_cov("cov_n30", cx, cy)

# ============================================================
# Bartlett sphericity (correlation matrices)
# ============================================================
# 3-variable correlated data
set.seed(1001)
n <- 30
v1 <- round(rnorm(n), 4)
v2 <- round(0.7 * v1 + rnorm(n, 0, 0.5), 4)
v3 <- round(0.3 * v1 + 0.5 * v2 + rnorm(n, 0, 0.5), 4)
mat3 <- cbind(v1, v2, v3)
emit_bartlett_sph("bs_3v", mat3)

# 4-variable with strong correlations
set.seed(1002)
n <- 50
a <- round(rnorm(n), 4)
b <- round(0.8 * a + rnorm(n, 0, 0.3), 4)
c_ <- round(0.6 * a + rnorm(n, 0, 0.4), 4)
d <- round(0.4 * b + 0.3 * c_ + rnorm(n, 0, 0.3), 4)
mat4 <- cbind(a, b, c_, d)
emit_bartlett_sph("bs_4v", mat4)
