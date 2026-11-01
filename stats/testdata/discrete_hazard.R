# Reference values for discrete_hazard_test.go: R's glm on Allison's
# person-period table. Run with Rscript; the printed numbers are copied into
# the test.
set.seed(21)
n <- 60
x <- round(rnorm(n), 2)
true <- pmin(1 + rgeom(n, plogis(-1.6 + 0.6 * x)), 12)
cens <- sample(4:14, n, replace = TRUE)
d <- pmin(true, cens)
obs <- as.integer(true <= cens & true < 12)
cat("d <- c(", paste(d, collapse = ", "), ")\n")
cat("obs <- c(", paste(obs, collapse = ", "), ")\n")
cat("x <- c(", paste(format(x, nsmall = 2), collapse = ", "), ")\n")
pp <- do.call(rbind, lapply(seq_len(n), function(i) {
  data.frame(t = seq_len(d[i]), y = c(rep(0, d[i] - 1), obs[i]), x = x[i])
}))
show <- function(label, f) {
  s <- summary(f)$coefficients
  cat(label, "\n")
  cat("  coef:", format(s[, 1], digits = 17), "\n")
  cat("  se:  ", format(s[, 2], digits = 17), "\n")
  cat("  dev: ", format(deviance(f), digits = 17), "\n")
}
show("logit log", glm(y ~ log(t) + x, binomial("logit"), pp))
show("cloglog log", glm(y ~ log(t) + x, binomial("cloglog"), pp))
show("logit linear", glm(y ~ t, binomial("logit"), pp))
