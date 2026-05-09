## Reference values for stats/regression.go computed in R 4.5.1.
##
## Sources of truth (all base R):
##   linear / multiple : lm(y ~ x1 + x2 + ...)
##   polynomial        : lm(y ~ x + I(x^2) + ... + I(x^d))   (raw, not poly())
##   exponential       : lm(log(y) ~ x)  then a = exp(intercept_log)
##                       — insyra reports inference on (a,b) via direct
##                         transformation:
##                           seA = a * seLnA            (delta method, 1st order)
##                           tA  = a / seA = 1 / seLnA  (insyra's formula)
##                           pA  = 2*(1-pt(|tA|, df))
##                         se_b/p_b/CI_b come from the linear fit on log(y).
##   logarithmic       : lm(y ~ log(x))   — straight lm on log-transformed x.
##
## CI: 95% Wald via t-quantile (qt(0.975, df) * SE).

options(digits = 17)

fmt <- function(x) {
  if (is.na(x)) return("NaN")
  if (is.infinite(x)) return(if (x > 0) "Inf" else "-Inf")
  formatC(x, digits = 17, format = "g")
}
emit <- function(key, val) cat(key, "=", fmt(val), "\n", sep = "")

emit_vec <- function(key, v) {
  for (i in seq_along(v)) emit(paste0(key, "[", i - 1, "]"), v[i])
}

emit_lm <- function(prefix, fit, df, n_coeffs) {
  s <- summary(fit)
  coefs <- s$coefficients
  emit_vec(paste0(prefix, ".coef"),  coefs[, 1])
  emit_vec(paste0(prefix, ".se"),    coefs[, 2])
  emit_vec(paste0(prefix, ".t"),     coefs[, 3])
  emit_vec(paste0(prefix, ".p"),     coefs[, 4])
  emit(paste0(prefix, ".rsq"),       s$r.squared)
  emit(paste0(prefix, ".adjr"),      s$adj.r.squared)
  emit(paste0(prefix, ".df"),        df)
  emit_vec(paste0(prefix, ".resid"), residuals(fit))
  ci <- confint(fit, level = 0.95)
  emit_vec(paste0(prefix, ".cilo"),  ci[, 1])
  emit_vec(paste0(prefix, ".cihi"),  ci[, 2])
}

# Simple linear: y ~ x
emit_simple <- function(prefix, x, y) {
  fit <- lm(y ~ x)
  emit_lm(prefix, fit, df = length(x) - 2, n_coeffs = 2)
}

# Multiple linear: y ~ x1 + x2 + ...
emit_multi <- function(prefix, y, xs) {
  d <- as.data.frame(xs); names(d) <- paste0("x", seq_along(xs))
  d$y <- y
  fml <- as.formula(paste("y ~", paste(names(d)[-length(d)], collapse = " + ")))
  fit <- lm(fml, data = d)
  emit_lm(prefix, fit, df = nrow(d) - length(xs) - 1, n_coeffs = length(xs) + 1)
}

# Polynomial regression: y ~ x + I(x^2) + ...
emit_poly <- function(prefix, x, y, degree) {
  terms <- paste("I(x^", seq_len(degree), ")", sep = "", collapse = " + ")
  fml   <- as.formula(paste("y ~", terms))
  fit   <- lm(fml)
  emit_lm(prefix, fit, df = length(x) - degree - 1, n_coeffs = degree + 1)
}

# Exponential: y = a*e^(b*x)  — insyra reports via direct param transformation
emit_exp <- function(prefix, x, y) {
  fit_log <- lm(log(y) ~ x)
  s       <- summary(fit_log)
  lna     <- coef(fit_log)[1]
  b       <- coef(fit_log)[2]
  se_lna  <- s$coefficients[1, 2]
  se_b    <- s$coefficients[2, 2]
  a       <- exp(lna)
  se_a    <- a * se_lna             # delta-method (matches insyra)
  t_a     <- a / se_a               # = 1/se_lna  — insyra formula
  t_b     <- b / se_b
  df      <- length(x) - 2
  p_a     <- 2 * (1 - pt(abs(t_a), df))
  p_b     <- 2 * (1 - pt(abs(t_b), df))
  qcrit   <- qt(0.975, df)
  ci_a    <- c(a - qcrit * se_a, a + qcrit * se_a)
  ci_b    <- c(b - qcrit * se_b, b + qcrit * se_b)
  # residuals: y - a*exp(b*x)  (against original y, not log-y)
  yhat <- a * exp(b * x)
  resid <- y - yhat
  ssr <- sum((y - mean(y))^2)
  sse <- sum(resid^2)
  r2  <- 1 - sse / ssr
  adj <- 1 - (1 - r2) * (length(x) - 1) / df
  emit(paste0(prefix, ".a"),    a)
  emit(paste0(prefix, ".b"),    b)
  emit(paste0(prefix, ".seA"),  se_a)
  emit(paste0(prefix, ".seB"),  se_b)
  emit(paste0(prefix, ".tA"),   t_a)
  emit(paste0(prefix, ".tB"),   t_b)
  emit(paste0(prefix, ".pA"),   p_a)
  emit(paste0(prefix, ".pB"),   p_b)
  emit(paste0(prefix, ".ciAlo"), ci_a[1])
  emit(paste0(prefix, ".ciAhi"), ci_a[2])
  emit(paste0(prefix, ".ciBlo"), ci_b[1])
  emit(paste0(prefix, ".ciBhi"), ci_b[2])
  emit(paste0(prefix, ".rsq"),   r2)
  emit(paste0(prefix, ".adjr"),  adj)
  emit(paste0(prefix, ".df"),    df)
  emit_vec(paste0(prefix, ".resid"), resid)
}

# Logarithmic: y = a + b*ln(x)
emit_log <- function(prefix, x, y) {
  fit <- lm(y ~ log(x))
  emit_lm(prefix, fit, df = length(x) - 2, n_coeffs = 2)
}

# ============================================================
# Existing test datasets (preserve so old tests can be migrated)
# ============================================================

# Multiple linear
emit_multi("ml_basic",
  c(3.5, 7.2, 9.8, 15.1, 17.9),
  list(c(1.2, 2.5, 3.1, 4.8, 5.3),
       c(0.8, 1.9, 2.7, 3.4, 4.1)))

# Linear: 3 cases
emit_simple("ln_case1",
  c(38.08, 95.07, 73.20, 59.87, 15.60, 15.60, 5.81, 86.62, 60.11, 70.81),
  c(212.12, 466.84, 359.15, 280.27, 84.92, 77.83, 30.49, 446.84, 286.32, 369.78))
emit_simple("ln_case2",
  c(2.05, 96.99, 83.24, 21.23, 18.18, 18.34, 30.42, 52.48, 43.19, 29.12),
  c(-1.88, 450.65, 413.85, 84.84, 96.76, 93.63, 132.95, 252.36, 216.66, 151.36))
emit_simple("ln_case3",
  c(61.19, 13.95, 29.21, 36.64, 45.61, 78.52, 19.97, 51.42, 59.24, 4.65),
  c(321.77, 66.99, 142.93, 190.34, 216.55, 402.59, 96.80, 271.27, 304.18, 26.59))

# Polynomial: quadratic, cubic
emit_poly("pl_quad",
  c(1, 2, 3, 4, 5, 6, 7, 8, 9, 10),
  c(2.1, 8.9, 20.1, 35.8, 56.2, 81.1, 110.9, 145.2, 184.1, 227.8), 2)
emit_poly("pl_cubic", c(1, 2, 3, 4, 5), c(5, 18, 47, 98, 177), 3)

# Exponential: y = e^x with noise
emit_exp("ex_basic",
  c(1, 2, 3, 4, 5, 6, 7, 8, 9, 10),
  c(2.72, 7.39, 20.09, 54.60, 148.41, 403.43, 1096.63, 2980.96, 8103.08, 22026.47))

# Logarithmic: y = ln(x)
emit_log("lo_basic",
  c(1, 2, 3, 4, 5, 6, 7, 8, 9, 10),
  c(0, 0.693, 1.099, 1.386, 1.609, 1.792, 1.946, 2.079, 2.197, 2.303))

# ============================================================
# Diverse cases
# ============================================================
# Simple: minimum n=3
emit_simple("ln_minN", c(1, 2, 3), c(2.5, 5.1, 7.4))
# Simple: large n=50
set.seed(2001); xL <- round(rnorm(50, 50, 10), 4)
yL <- 2.5 + 1.3 * xL + round(rnorm(50, 0, 5), 4)
emit_simple("ln_largeN", xL, yL)
# Simple: negative slope
emit_simple("ln_neg", c(1, 2, 3, 4, 5, 6, 7),
  c(20.1, 18.5, 16.2, 14.8, 13.1, 11.5, 9.8))
# Simple: huge magnitude
emit_simple("ln_huge",
  c(1.0e6, 2.0e6, 3.0e6, 4.0e6, 5.0e6),
  c(2.5e6, 5.1e6, 7.4e6, 10.0e6, 12.6e6))

# Multiple: 3 predictors
set.seed(2002)
n <- 30
mx1 <- round(rnorm(n, 10, 2), 4)
mx2 <- round(rnorm(n, 5, 1.5), 4)
mx3 <- round(rnorm(n, 0, 1), 4)
my <- 2 + 1.5 * mx1 + 0.7 * mx2 - 0.3 * mx3 + round(rnorm(n, 0, 0.5), 4)
emit_multi("ml_3pred", my, list(mx1, mx2, mx3))

# Polynomial degree 4
set.seed(2003)
xp <- 1:15
yp <- round(0.5 * xp^4 - 2 * xp^3 + 3 * xp^2 - xp + 1 + rnorm(15, 0, 5), 3)
emit_poly("pl_deg4", xp, yp, 4)

# Logarithmic: noisy
set.seed(2004); xN <- round(seq(1, 20, length.out = 25), 4)
yN <- round(3 + 2 * log(xN) + rnorm(25, 0, 0.5), 4)
emit_log("lo_noisy", xN, yN)

# Exponential: decay (negative b)
emit_exp("ex_decay",
  c(0, 1, 2, 3, 4, 5, 6, 7, 8, 9),
  c(100, 60.65, 36.79, 22.31, 13.53, 8.21, 4.98, 3.02, 1.83, 1.11))
