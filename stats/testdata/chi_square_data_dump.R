## Dumps random-categorical data so Go tests use byte-identical inputs.
set.seed(3001)
big <- sample(c("yes", "no"), 200, replace = TRUE, prob = c(0.6, 0.4))
cat("gof_largeN:", paste(big, collapse=","), "\n", sep="")

set.seed(3002)
n <- 200
rs <- sample(c("r1", "r2", "r3", "r4"), n, replace = TRUE)
cs <- sample(c("c1", "c2", "c3"), n, replace = TRUE)
cat("ind_4x3_largeN_rows:", paste(rs, collapse=","), "\n", sep="")
cat("ind_4x3_largeN_cols:", paste(cs, collapse=","), "\n", sep="")
