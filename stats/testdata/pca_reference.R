## Reference values for stats/pca.go computed in R 4.5.1.
##
## Source of truth: prcomp(data, center = TRUE, scale = TRUE).
##   prcomp standardises each column by the sample SD (n-1 divisor) and
##   performs SVD on the standardised matrix.
##   eigenvalues = sdev^2  (variance along each PC, equals eigenvalues of
##                          the correlation matrix of the original data).
##   explained   = 100 * eigenvalues / sum(eigenvalues)  (insyra reports %).
##   loadings    = rotation matrix; insyra's Components table has one
##                 column per PC, rows = variables, in the same shape.
##
## Sign canonicalisation: insyra forces each PC's loading vector to have
## a non-negative first element. R doesn't, so we apply the same
## convention here when emitting reference values.

options(digits = 17)

fmt <- function(x) {
  if (is.na(x)) return("NaN")
  if (is.infinite(x)) return(if (x > 0) "Inf" else "-Inf")
  formatC(x, digits = 17, format = "g")
}
emit <- function(key, val) cat(key, "=", fmt(val), "\n", sep = "")

emit_pca <- function(prefix, rows, nComponents) {
  m <- do.call(rbind, rows)
  fit <- prcomp(m, center = TRUE, scale = TRUE)
  eigs <- fit$sdev^2
  total_var <- sum(eigs)
  explained <- 100 * eigs / total_var
  rot <- fit$rotation                       # ncol × ncol matrix
  # Apply insyra's sign convention: flip each PC so loadings[1, j] >= 0
  for (j in seq_len(ncol(rot))) {
    if (rot[1, j] < 0) rot[, j] <- -rot[, j]
  }

  for (i in seq_len(nComponents)) {
    emit(paste0(prefix, ".eig[", i - 1, "]"), eigs[i])
    emit(paste0(prefix, ".exp[", i - 1, "]"), explained[i])
  }
  for (j in seq_len(nComponents)) {
    for (i in seq_len(nrow(rot))) {
      emit(paste0(prefix, ".pc", j, "[", i - 1, "]"), rot[i, j])
    }
  }
}

# ============================================================
# Test cases shared with crosslang_public_methods_test.go (different tolerance)
# ============================================================
emit_pca("pca_a",
  list(c(2.5, 2.4, 1.2), c(0.5, 0.7, 0.3), c(2.2, 2.9, 1.1),
       c(1.9, 2.2, 0.9), c(3.1, 3.0, 1.5), c(2.3, 2.7, 1.3)),
  2)
emit_pca("pca_b",
  list(c(10, 20, 30), c(11, 19, 29), c(12, 21, 31),
       c(13, 22, 33), c( 9, 18, 28), c(14, 23, 34)),
  2)
emit_pca("pca_c",
  list(c(100, 60, 30), c(98, 58, 29), c(102, 62, 31),
       c(101, 61, 32), c( 99, 59, 30), c( 97, 57, 28)),
  3)

# ============================================================
# Diverse cases
# ============================================================
# 4 variables, n=10, weak correlation between two pairs
set.seed(4001)
n <- 10
v1 <- round(rnorm(n), 4)
v2 <- round(0.5 * v1 + rnorm(n, 0, 0.5), 4)
v3 <- round(rnorm(n), 4)
v4 <- round(0.7 * v3 + rnorm(n, 0, 0.3), 4)
emit_pca("pca_4var", split(rbind(v1, v2, v3, v4), col(rbind(v1, v2, v3, v4))), 4)

# 5 variables, n=20, mixed correlations
set.seed(4002)
n <- 20
m5 <- matrix(round(rnorm(5 * n), 4), nrow = n)
emit_pca("pca_5var",
  lapply(seq_len(nrow(m5)), function(i) m5[i, ]), 5)

# Small case: 3 variables × 4 observations (minimum-ish)
emit_pca("pca_n4",
  list(c(1, 2, 3), c(2, 3, 4), c(3, 4, 5), c(4, 5, 7)),
  3)

# Huge magnitude — values around 1e6 (after standardisation should be unaffected)
emit_pca("pca_huge",
  list(c(1e6, 2e6, 3e6), c(1.001e6, 2.002e6, 3.001e6),
       c(0.999e6, 1.998e6, 2.999e6), c(1.0005e6, 2.001e6, 3.0005e6),
       c(1.0015e6, 2.003e6, 3.002e6)),
  3)
