## Dumps rnorm-generated data for correlation diverse cases.
set.seed(901); xL <- round(rnorm(50, 0, 1), 5)
set.seed(902); yL <- round(rnorm(50, 0, 1), 5) + 0.3 * xL
cat("n50_x:", paste(xL, collapse=","), "\n", sep="")
cat("n50_y:", paste(yL, collapse=","), "\n", sep="")

set.seed(903); xN <- round(rnorm(20, 0, 1), 5)
set.seed(904); yN <- round(-0.5 * xN + rnorm(20, 0, 1), 5)
cat("n20neg_x:", paste(xN, collapse=","), "\n", sep="")
cat("n20neg_y:", paste(yN, collapse=","), "\n", sep="")

set.seed(999); cx <- round(rnorm(30, 100, 10), 3)
set.seed(998); cy <- round(rnorm(30, 100, 10), 3)
cat("cov_n30_x:", paste(cx, collapse=","), "\n", sep="")
cat("cov_n30_y:", paste(cy, collapse=","), "\n", sep="")

set.seed(1001)
n <- 30
v1 <- round(rnorm(n), 4)
v2 <- round(0.7 * v1 + rnorm(n, 0, 0.5), 4)
v3 <- round(0.3 * v1 + 0.5 * v2 + rnorm(n, 0, 0.5), 4)
cat("bs_3v_v1:", paste(v1, collapse=","), "\n", sep="")
cat("bs_3v_v2:", paste(v2, collapse=","), "\n", sep="")
cat("bs_3v_v3:", paste(v3, collapse=","), "\n", sep="")

set.seed(1002)
n <- 50
a <- round(rnorm(n), 4)
b <- round(0.8 * a + rnorm(n, 0, 0.3), 4)
c_ <- round(0.6 * a + rnorm(n, 0, 0.4), 4)
d <- round(0.4 * b + 0.3 * c_ + rnorm(n, 0, 0.3), 4)
cat("bs_4v_a:", paste(a, collapse=","), "\n", sep="")
cat("bs_4v_b:", paste(b, collapse=","), "\n", sep="")
cat("bs_4v_c:", paste(c_, collapse=","), "\n", sep="")
cat("bs_4v_d:", paste(d, collapse=","), "\n", sep="")
