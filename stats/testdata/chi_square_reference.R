## Reference values for stats/chi_square.go computed in R 4.5.1.
##
## Sources of truth (base R):
##   GoF          : chisq.test(observed, p = p_expected)
##                  insyra: ChiSquareGoodnessOfFit takes raw category strings
##                  and tabulates them; the equivalent R input is observed
##                  counts ordered by sort(unique(category)).
##   Independence : chisq.test(matrix(observed, nrow, ncol))
##                  insyra computes expected = rowSum*colSum/total without
##                  Yates' continuity correction; we mirror that (R's default
##                  for any > 2x2 table; for 2x2 set correct=FALSE).

options(digits = 17)

fmt <- function(x) {
  if (is.na(x)) return("NaN")
  if (is.infinite(x)) return(if (x > 0) "Inf" else "-Inf")
  formatC(x, digits = 17, format = "g")
}
emit <- function(key, val) cat(key, "=", fmt(val), "\n", sep = "")

emit_gof <- function(prefix, categories, p = NULL, rescale = FALSE) {
  keys <- sort(unique(categories))
  observed <- as.double(sapply(keys, function(k) sum(categories == k)))
  if (is.null(p)) p <- rep(1 / length(observed), length(observed))
  if (rescale) p <- p / sum(p)
  expected <- sum(observed) * p
  chi <- sum((observed - expected)^2 / expected)
  df  <- length(observed) - 1
  pval <- 1 - pchisq(chi, df)
  emit(paste0(prefix, ".chi"), chi)
  emit(paste0(prefix, ".p"),   pval)
  emit(paste0(prefix, ".df"),  df)
  for (i in seq_along(observed)) {
    emit(paste0(prefix, ".obs[", i - 1, "]"), observed[i])
    emit(paste0(prefix, ".exp[", i - 1, "]"), expected[i])
  }
}

emit_indep <- function(prefix, rowCats, colCats) {
  rkeys <- sort(unique(rowCats))
  ckeys <- sort(unique(colCats))
  obs <- matrix(0, nrow = length(rkeys), ncol = length(ckeys),
                dimnames = list(rkeys, ckeys))
  for (i in seq_along(rowCats)) {
    obs[rowCats[i], colCats[i]] <- obs[rowCats[i], colCats[i]] + 1
  }
  rs <- rowSums(obs); cs <- colSums(obs); total <- sum(obs)
  exp_mat <- (rs %*% t(cs)) / total
  chi <- sum((obs - exp_mat)^2 / exp_mat)
  df  <- (length(rkeys) - 1) * (length(ckeys) - 1)
  pval <- 1 - pchisq(chi, df)
  emit(paste0(prefix, ".chi"), chi)
  emit(paste0(prefix, ".p"),   pval)
  emit(paste0(prefix, ".df"),  df)
  # column-major flatten matching insyra's contingency table layout
  for (j in seq_along(ckeys)) {
    for (i in seq_along(rkeys)) {
      idx <- (i - 1) * length(ckeys) + (j - 1)
      emit(paste0(prefix, ".obs[", idx, "]"), obs[i, j])
      emit(paste0(prefix, ".exp[", idx, "]"), exp_mat[i, j])
    }
  }
}

# ============================================================
# Independence — existing test case
# ============================================================
emit_indep("ind_existing",
  c("A", "A", "B", "B", "B", "C"),
  c("X", "Y", "X", "Y", "Y", "Y"))

# ============================================================
# Goodness-of-fit cases
# ============================================================
# Uniform expected (no p given)
emit_gof("gof_uniform",
  c("A", "A", "A", "B", "B", "C", "D", "D", "D", "D"))
# Custom probabilities, sum=1
emit_gof("gof_custom",
  c("red", "red", "blue", "green", "blue", "red", "green", "red", "blue", "red"),
  c(0.5, 0.3, 0.2))   # alphabetical: blue, green, red
# rescaleP=TRUE
emit_gof("gof_rescale",
  c("A", "B", "B", "C", "C", "C", "D", "D", "D", "D"),
  c(1, 2, 3, 4),       # not normalized; will be rescaled to sum=1
  rescale = TRUE)
# Large n, two categories, slight imbalance
set.seed(3001)
big <- sample(c("yes", "no"), 200, replace = TRUE, prob = c(0.6, 0.4))
emit_gof("gof_largeN", big)
# Many categories
emit_gof("gof_many",
  rep(c("A", "B", "C", "D", "E", "F", "G", "H"),
      times = c(12, 8, 15, 10, 7, 13, 9, 11)))

# ============================================================
# Independence — diverse cases
# ============================================================
# 2x2 — strong association
emit_indep("ind_2x2_strong",
  rep(c("M", "F"), each = 50),
  c(rep("Y", 40), rep("N", 10), rep("Y", 12), rep("N", 38)))
# 3x3
emit_indep("ind_3x3",
  rep(c("low", "mid", "high"), each = 30),
  c(rep("a", 15), rep("b", 10), rep("c", 5),
    rep("a", 10), rep("b", 12), rep("c", 8),
    rep("a",  5), rep("b",  8), rep("c", 17)))
# Larger 4x3
set.seed(3002)
n <- 200
rs <- sample(c("r1", "r2", "r3", "r4"), n, replace = TRUE)
cs <- sample(c("c1", "c2", "c3"), n, replace = TRUE)
emit_indep("ind_4x3_largeN", rs, cs)
