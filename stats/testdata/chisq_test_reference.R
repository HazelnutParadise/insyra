## Reference values for stats/chi_square.go, produced by R's own
## stats::chisq.test — every number below is that function's output, not a
## value recomputed here from the chi-square formula.
##
## Run with: R version 4.6.1 (2026-06-24)   [R.version.string]
##
## Regenerate:
##   cd stats/testdata && Rscript chisq_test_reference.R > chisq_test_reference.txt
##
## Conventions
##   Category labels are ordered with sort(..., method = "radix"), i.e. C-locale
##   byte order, which is what Go's sort.Strings does — so the label[i] /
##   obs[i] / exp[i] triples line up with the library's own tabulation.
##   For the goodness-of-fit cases the named p vector is read as p[keys], so the
##   order the probabilities were written in never changes the result; the
##   pkey/pval keys keep the written order so the call can be replayed.
##   Independence tables use correct = FALSE (no Yates' continuity correction);
##   goodness-of-fit cases use chisq.test's own defaults, plus rescale.p =
##   TRUE for the case that asks for it.

options(digits = 17)
fmt <- function(x) { if (is.na(x)) return("NaN"); if (is.infinite(x)) return(if (x > 0) "Inf" else "-Inf"); formatC(x, digits = 17, format = "g") }
emit <- function(key, val) cat(key, "=", fmt(val), "\n", sep = "")
emits <- function(key, s) cat(key, "=", s, "\n", sep = "")

## Goodness-of-fit. x is a character vector of raw observations; p is NULL
## (uniform) or a NAMED numeric vector whose names are the category labels.
emit_gof <- function(prefix, x, p = NULL, rescale = FALSE) {
  keys <- sort(unique(x), method = "radix")
  counts <- as.vector(table(factor(x, levels = keys)))
  if (is.null(p)) {
    r <- suppressWarnings(chisq.test(counts))
  } else {
    r <- suppressWarnings(chisq.test(counts, p = unname(p[keys]), rescale.p = rescale))
  }
  emits(paste0(prefix, ".data"), paste(x, collapse = ","))
  if (!is.null(p)) {
    for (i in seq_along(p)) {
      emits(paste0(prefix, ".pkey[", i - 1, "]"), names(p)[i])
      emit(paste0(prefix, ".pval[", i - 1, "]"), p[[i]])
    }
  }
  emits(paste0(prefix, ".rescale"), if (rescale) "true" else "false")
  emit(paste0(prefix, ".stat"), r$statistic)
  emit(paste0(prefix, ".p"), r$p.value)
  emit(paste0(prefix, ".df"), r$parameter)
  for (i in seq_along(keys)) {
    emits(paste0(prefix, ".label[", i - 1, "]"), keys[i])
    emit(paste0(prefix, ".obs[", i - 1, "]"), r$observed[i])
    emit(paste0(prefix, ".exp[", i - 1, "]"), r$expected[i])
  }
}

## Independence of a two-way table. rows/cols are the raw per-observation
## category vectors; the table is built by tabulating them.
emit_ind <- function(prefix, rows, cols) {
  rk <- sort(unique(rows), method = "radix")
  ck <- sort(unique(cols), method = "radix")
  tab <- table(factor(rows, levels = rk), factor(cols, levels = ck))
  r <- suppressWarnings(chisq.test(tab, correct = FALSE))
  emits(paste0(prefix, ".rows"), paste(rows, collapse = ","))
  emits(paste0(prefix, ".cols"), paste(cols, collapse = ","))
  emit(paste0(prefix, ".stat"), r$statistic)
  emit(paste0(prefix, ".p"), r$p.value)
  emit(paste0(prefix, ".df"), r$parameter)
  for (i in seq_along(rk)) emits(paste0(prefix, ".rowlabel[", i - 1, "]"), rk[i])
  for (j in seq_along(ck)) emits(paste0(prefix, ".collabel[", j - 1, "]"), ck[j])
  for (i in seq_along(rk)) {
    for (j in seq_along(ck)) {
      emit(paste0(prefix, ".obs[", i - 1, "][", j - 1, "]"), r$observed[i, j])
      emit(paste0(prefix, ".exp[", i - 1, "][", j - 1, "]"), r$expected[i, j])
    }
  }
}

# ============================================================
# Goodness-of-fit
# ============================================================
# Uniform expectations, no p given.
emit_gof("gof_uniform",
  c("A", "A", "A", "B", "B", "C", "D", "D", "D", "D"))

# Named probabilities whose written order differs from the sorted label order.
emit_gof("gof_named",
  c("red", "red", "blue", "green", "blue", "red", "green", "red", "blue", "red"),
  p = c(red = 0.5, green = 0.3, blue = 0.2))

# Raw weights summing to 10, rescaled by chisq.test itself.
emit_gof("gof_rescale",
  c("A", "B", "B", "C", "C", "C", "D", "D", "D", "D"),
  p = c(D = 4, C = 3, B = 2, A = 1), rescale = TRUE)

# Eight categories, uneven counts.
emit_gof("gof_many",
  rep(c("A", "B", "C", "D", "E", "F", "G", "H"),
      times = c(12, 8, 15, 10, 7, 13, 9, 11)))

# ============================================================
# Independence
# ============================================================
# 3x2 with a sparse cell.
emit_ind("ind_3x2",
  c("A", "A", "B", "B", "B", "C"),
  c("X", "Y", "X", "Y", "Y", "Y"))

# 2x2 strong association; correct = FALSE keeps it comparable to the library.
emit_ind("ind_2x2",
  rep(c("M", "F"), each = 50),
  c(rep("Y", 40), rep("N", 10), rep("Y", 12), rep("N", 38)))

# 3x3 balanced margins.
emit_ind("ind_3x3",
  rep(c("low", "mid", "high"), each = 30),
  c(rep("a", 15), rep("b", 10), rep("c",  5),
    rep("a", 10), rep("b", 12), rep("c",  8),
    rep("a",  5), rep("b",  8), rep("c", 17)))
