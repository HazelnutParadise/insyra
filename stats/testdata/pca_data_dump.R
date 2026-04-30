## Dumps rnorm-generated PCA data so Go tests use byte-identical inputs.

set.seed(4001)
n <- 10
v1 <- round(rnorm(n), 4)
v2 <- round(0.5 * v1 + rnorm(n, 0, 0.5), 4)
v3 <- round(rnorm(n), 4)
v4 <- round(0.7 * v3 + rnorm(n, 0, 0.3), 4)
cat("pca_4var_v1:", paste(v1, collapse=","), "\n", sep="")
cat("pca_4var_v2:", paste(v2, collapse=","), "\n", sep="")
cat("pca_4var_v3:", paste(v3, collapse=","), "\n", sep="")
cat("pca_4var_v4:", paste(v4, collapse=","), "\n", sep="")

set.seed(4002)
n <- 20
m5 <- matrix(round(rnorm(5 * n), 4), nrow = n)
for (j in 1:5) cat(sprintf("pca_5var_v%d:", j), paste(m5[, j], collapse=","), "\n", sep="")
